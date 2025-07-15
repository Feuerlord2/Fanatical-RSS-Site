package gofanatical

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/feeds"
	log "github.com/sirupsen/logrus"
)

// Neue API Response Struktur für /api/all/de
type FanaticalAllResponse struct {
	StarDeal         *FanaticalAPIBundle   `json:"stardeal"`
	PickAndMix       []PickAndMixBundle    `json:"pickandmix"`
	LiveDisplayTypes map[string]int        `json:"liveDisplayTypes"`
	Images           []interface{}         `json:"images"`
	// Weitere Felder falls nötig
}

// Bestehende Struktur für einzelne Bundle-Items
type FanaticalAPIBundle struct {
	ProductID       string            `json:"product_id"`
	SKU             string            `json:"sku"`
	Name            string            `json:"name"`
	Slug            string            `json:"slug"`
	Type            string            `json:"type"`
	DisplayType     string            `json:"display_type"`
	Cover           string            `json:"cover"`
	Tiered          bool              `json:"tiered"`
	DiscountPercent int               `json:"discount_percent"`
	OnSale          bool              `json:"on_sale"`
	BestEver        bool              `json:"best_ever"`
	FlashSale       bool              `json:"flash_sale"`
	StarDeal        bool              `json:"star_deal"`
	Price           map[string]float64 `json:"price"`
	FullPrice       map[string]float64 `json:"fullPrice"`
	DRM             []string          `json:"drm"`
	ValidFrom       int64             `json:"available_valid_from"`
	ValidUntil      int64             `json:"available_valid_until"`
	Position        int               `json:"position"`
	ReleaseDate     int64             `json:"release_date"`
	Giveaway        bool              `json:"giveaway"`
	GameTotal       int               `json:"game_total"`
	DLCTotal        int               `json:"dlc_total"`
	BundleCovers    []BundleGame      `json:"bundle_covers"`
	PNMSaving       int               `json:"pnm_saving"`
	OperatingSystems []string         `json:"operating_systems"`
	Features         []string         `json:"features"`
	Categories       []string         `json:"categories"`
	// Neue Felder aus dem JSON
	SDescs          map[string]string `json:"sdescs"`
	Screenshots     []string          `json:"screenshots"`
}

// Pick-and-Mix Bundle Struktur
type PickAndMixBundle struct {
	ID          string                 `json:"_id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	CoverImage  string                 `json:"cover_image"`
	SKU         string                 `json:"sku"`
	Type        string                 `json:"type"`
	ValidFrom   string                 `json:"valid_from"`
	ValidUntil  string                 `json:"valid_until"`
	Tiers       []PickAndMixTier       `json:"tiers"`
	Products    []PickAndMixProduct    `json:"products"`
}

type PickAndMixTier struct {
	Quantity int                   `json:"quantity"`
	Price    map[string]float64    `json:"price"`
}

type PickAndMixProduct struct {
	ID    string `json:"_id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Cover string `json:"cover"`
}

type BundleGame struct {
	Name             string            `json:"name"`
	Slug             string            `json:"slug"`
	Cover            string            `json:"cover"`
	Type             string            `json:"type"`
	DRM              []string          `json:"drm"`
	OperatingSystems []string          `json:"operating_systems"`
	Price            map[string]float64 `json:"price"`
}

// Bestehende Strukturen für Promotions (unverändert)
type PromotionsResponse struct {
	SaleRecords  []interface{}      `json:"saleRecords"`
	FreeProducts []FreeProduct      `json:"freeProducts"`
	Deliveries   []Delivery         `json:"deliveries"`
	Vouchers     []Voucher          `json:"vouchers"`
}

type FreeProduct struct {
	ID               string            `json:"_id"`
	PartnerBrand     string            `json:"partnerBrand"`
	Public           bool              `json:"public"`
	ValidFrom        string            `json:"valid_from"`
	ValidUntil       string            `json:"valid_until"`
	MinSpend         map[string]float64 `json:"min_spend"`
	MaxSpend         map[string]float64 `json:"max_spend"`
	Book             bool              `json:"book"`
	Bundle           bool              `json:"bundle"`
	Game             bool              `json:"game"`
	Products         []PromotionProduct `json:"products"`
}

type PromotionProduct struct {
	ID       string            `json:"_id"`
	Name     string            `json:"name"`
	Slug     string            `json:"slug"`
	Type     string            `json:"type"`
	Cover    string            `json:"cover"`
	Price    map[string]float64 `json:"price"`
	Mystery  bool              `json:"mystery"`
	Visible  ProductVisibility  `json:"visible"`
	IsVisible bool             `json:"is_visible"`
}

type ProductVisibility struct {
	ValidFrom string `json:"valid_from"`
	ValidUntil string `json:"valid_until"`
}

type Delivery struct {
	ID          string            `json:"_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Percent     int               `json:"percent"`
	MinSpend    map[string]float64 `json:"min_spend"`
	ValidFrom   string            `json:"valid_from"`
	ValidUntil  string            `json:"valid_until"`
	Public      bool              `json:"public"`
	StarDeal    bool              `json:"star_deal"`
}

type Voucher struct {
	ID           string            `json:"_id"`
	Code         string            `json:"code"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Percent      int               `json:"percent"`
	MinSpend     map[string]float64 `json:"min_spend"`
	MaxSpend     map[string]float64 `json:"max_spend"`
	ValidFrom    string            `json:"valid_from"`
	ValidUntil   string            `json:"valid_until"`
	Public       bool              `json:"public"`
	StarDeal     bool              `json:"star_deal"`
	Game         bool              `json:"game"`
	Bundle       bool              `json:"bundle"`
	Book         bool              `json:"book"`
	FullPriceOnly bool             `json:"full_price_only"`
}

func Run() {
	wg := sync.WaitGroup{}
	for _, category := range []string{"books", "games", "software"} {
		wg.Add(1)
		go updateCategory(&wg, category)
	}
	wg.Wait()
}

func createFeed(bundles []FanaticalBundle, category string) (feeds.Feed, error) {
	feed := feeds.Feed{
		Title:       fmt.Sprintf("Fanatical RSS %s Bundles", strings.Title(category)),
		Link:        &feeds.Link{Href: "https://feuerlord2.github.io/Fanatical-RSS-Site/"},
		Description: fmt.Sprintf("Latest Fanatical %s bundles with amazing deals and discounts!", category),
		Author:      &feeds.Author{Name: "Daniel Winter", Email: "DanielWinterEmsdetten+rss@gmail.com"},
		Created:     time.Now(),
	}

	feed.Items = make([]*feeds.Item, len(bundles))
	for idx, bundle := range bundles {
		// Verbesserter Content mit mehr Details
		content := createRichContent(bundle)
		
		// Verbesserter Titel mit Emoji und Discount Info
		title := createEnhancedTitle(bundle)

		feed.Items[idx] = &feeds.Item{
			Title:       title,
			Link:        &feeds.Link{Href: fmt.Sprintf("https://www.fanatical.com%s", bundle.URL)},
			Content:     content,
			Created:     bundle.StartDate,
			Description: bundle.Description,
			Id:          fmt.Sprintf("fanatical-%s-%d", bundle.Slug, bundle.StartDate.Unix()),
		}
	}

	// Sort items so that latest bundles are on the top
	sort.Slice(feed.Items, func(i, j int) bool { 
		return feed.Items[i].Created.After(feed.Items[j].Created) 
	})

	return feed, nil
}

func createEnhancedTitle(bundle FanaticalBundle) string {
	var titleParts []string
	
	// Emoji basierend auf Discount
	if bundle.Price.Discount >= 90 {
		titleParts = append(titleParts, "🔥")
	} else if bundle.Price.Discount >= 75 {
		titleParts = append(titleParts, "⚡")
	} else if bundle.Price.Discount >= 50 {
		titleParts = append(titleParts, "💥")
	}
	
	titleParts = append(titleParts, bundle.Title)
	
	// Discount und Preis Info
	if bundle.Price.Discount > 0 {
		titleParts = append(titleParts, fmt.Sprintf("(%d%% OFF)", bundle.Price.Discount))
	}
	
	if bundle.Price.Amount > 0 {
		titleParts = append(titleParts, fmt.Sprintf("- $%.2f", bundle.Price.Amount))
	} else {
		titleParts = append(titleParts, "- FREE")
	}
	
	return strings.Join(titleParts, " ")
}

func createRichContent(bundle FanaticalBundle) string {
	var content strings.Builder
	
	content.WriteString(fmt.Sprintf("<h3>%s</h3>\n", bundle.Title))
	content.WriteString(fmt.Sprintf("<p>%s</p>\n", bundle.Description))
	
	// Preis-Tabelle
	content.WriteString("<table border='1' style='border-collapse: collapse; margin: 10px 0;'>\n")
	content.WriteString("<tr style='background-color: #f0f0f0;'><th style='padding: 5px;'>Current Price</th><th style='padding: 5px;'>Original Price</th><th style='padding: 5px;'>Discount</th><th style='padding: 5px;'>You Save</th></tr>\n")
	
	currentPrice := "$" + fmt.Sprintf("%.2f", bundle.Price.Amount)
	if bundle.Price.Amount == 0 {
		currentPrice = "FREE"
	}
	
	originalPrice := "$" + fmt.Sprintf("%.2f", bundle.Price.Original)
	if bundle.Price.Original == 0 {
		originalPrice = "N/A"
	}
	
	savings := bundle.Price.Original - bundle.Price.Amount
	savingsText := "$" + fmt.Sprintf("%.2f", savings)
	if savings <= 0 {
		savingsText = "N/A"
	}
	
	content.WriteString(fmt.Sprintf("<tr><td style='padding: 5px; text-align: center;'><strong>%s</strong></td><td style='padding: 5px; text-align: center;'>%s</td><td style='padding: 5px; text-align: center;'>%d%%</td><td style='padding: 5px; text-align: center;'>%s</td></tr>\n",
		currentPrice, originalPrice, bundle.Price.Discount, savingsText))
	content.WriteString("</table>\n")
	
	// Verfügbarkeit
	content.WriteString("<h4>⏰ Availability</h4>\n")
	content.WriteString("<ul>\n")
	content.WriteString(fmt.Sprintf("<li><strong>Ends:</strong> %s</li>\n", bundle.EndDate.Format("January 2, 2006 15:04 MST")))
	
	// Verbleibende Zeit
	timeRemaining := time.Until(bundle.EndDate)
	if timeRemaining > 0 {
		days := int(timeRemaining.Hours() / 24)
		hours := int(timeRemaining.Hours()) % 24
		content.WriteString(fmt.Sprintf("<li><strong>Time Remaining:</strong> %d days, %d hours</li>\n", days, hours))
	}
	content.WriteString("</ul>\n")
	
	// Direct Link
	content.WriteString(fmt.Sprintf("<p><a href='https://www.fanatical.com%s' style='background-color: #ff6f00; color: white; padding: 10px 15px; text-decoration: none; border-radius: 5px;'>🛒 Get this deal on Fanatical</a></p>\n", bundle.URL))
	
	return content.String()
}

func updateCategory(wg *sync.WaitGroup, category string) {
	defer wg.Done()

	log.WithField("category", category).Info("Fetching data from Fanatical APIs")
	
	// Fetch from new /api/all/de endpoint
	bundles, err := fetchBundlesFromNewAPI()
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to fetch bundles from new API")
		bundles = []FanaticalBundle{}
	}

	// Also fetch promotions
	promotions, err := fetchPromotionsFromAPI()
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to fetch promotions from API")
	} else {
		// Convert promotions to bundles and add them
		promotionBundles := convertPromotionsToBundles(promotions, category)
		bundles = append(bundles, promotionBundles...)
	}
	
	// Filter bundles by category
	var filteredBundles []FanaticalBundle
	for _, bundle := range bundles {
		if shouldIncludeBundle(bundle, category) {
			filteredBundles = append(filteredBundles, bundle)
		}
	}
	
	log.WithFields(log.Fields{
		"category": category,
		"total":    len(bundles),
		"filtered": len(filteredBundles),
	}).Info("Bundle filtering completed")
	
	// If no bundles found after filtering, create a test bundle
	if len(filteredBundles) == 0 {
		log.WithField("category", category).Warn("No bundles found for category, creating test bundle")
		filteredBundles = createTestBundle(category)
	}
	
	feed, err := createFeed(filteredBundles, category)
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to create feed")
		return
	}

	if err := writeFeedToFile(feed, category); err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to write feed to file")
	} else {
		log.WithFields(log.Fields{
			"category": category,
			"bundles":  len(filteredBundles),
		}).Info("Successfully created RSS feed")
	}
}

// NEUE Funktion für /api/all/de
func fetchBundlesFromNewAPI() ([]FanaticalBundle, error) {
	url := "https://www.fanatical.com/api/all/de"
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers to appear like a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.fanatical.com/en/bundles")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch new API: %w", err)
	}
	defer resp.Body.Close()
	
	log.WithField("status", resp.StatusCode).Info("New API response received")
	
	if resp.StatusCode != 200 {
		// Lese Body für bessere Fehlerdiagnose
		body, _ := io.ReadAll(resp.Body)
		log.WithFields(log.Fields{
			"status": resp.StatusCode,
			"body":   string(body)[:min(500, len(body))],
		}).Error("New API returned non-200 status")
		return nil, fmt.Errorf("new API returned status %d", resp.StatusCode)
	}
	
	var apiResponse FanaticalAllResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode new API response: %w", err)
	}
	
	var allBundles []FanaticalAPIBundle
	
	// Add StarDeal if available
	if apiResponse.StarDeal != nil {
		allBundles = append(allBundles, *apiResponse.StarDeal)
	}
	
	// Convert PickAndMix bundles
	for _, pnm := range apiResponse.PickAndMix {
		bundle := convertPickAndMixToBundle(pnm)
		allBundles = append(allBundles, bundle)
	}
	
	log.WithField("bundles", len(allBundles)).Info("Successfully fetched bundles from new API")
	
	return convertAPIBundlesToInternal(allBundles), nil
}

// Helper function für Go 1.20
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Konvertiere PickAndMix zu Standard Bundle Format
func convertPickAndMixToBundle(pnm PickAndMixBundle) FanaticalAPIBundle {
	validFrom, _ := time.Parse(time.RFC3339, pnm.ValidFrom)
	validUntil, _ := time.Parse(time.RFC3339, pnm.ValidUntil)
	
	// Nimm den günstigsten Tier-Preis
	var minPrice float64 = 999999
	for _, tier := range pnm.Tiers {
		if usdPrice, exists := tier.Price["USD"]; exists && usdPrice < minPrice {
			minPrice = usdPrice
		}
	}
	
	// Erstelle Fake-FullPrice (doppelter Preis für Discount-Berechnung)
	fullPrice := minPrice * 2
	
	return FanaticalAPIBundle{
		ProductID:       pnm.ID,
		Name:            pnm.Name,
		Slug:            pnm.Slug,
		Type:            "pick-and-mix", // WICHTIG: Setze korrekten Typ
		DisplayType:     pnm.Type,
		Cover:           pnm.CoverImage,
		OnSale:          true,
		ValidFrom:       validFrom.Unix(),
		ValidUntil:      validUntil.Unix(),
		Price:           map[string]float64{"USD": minPrice},
		FullPrice:       map[string]float64{"USD": fullPrice},
		DiscountPercent: 50, // Standard Discount für PickAndMix
		GameTotal:       len(pnm.Products),
	}
}

// Bestehende Funktionen (unverändert)
func fetchPromotionsFromAPI() (*PromotionsResponse, error) {
	url := "https://www.fanatical.com/api/all-promotions/de"
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers to appear like a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.fanatical.com/en/bundles")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch promotions API: %w", err)
	}
	defer resp.Body.Close()
	
	log.WithField("status", resp.StatusCode).Info("Promotions API response received")
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("promotions API returned status %d", resp.StatusCode)
	}
	
	var promotions PromotionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&promotions); err != nil {
		return nil, fmt.Errorf("failed to decode promotions API response: %w", err)
	}
	
	log.WithFields(log.Fields{
		"free_products": len(promotions.FreeProducts),
		"deliveries":    len(promotions.Deliveries),
		"vouchers":      len(promotions.Vouchers),
	}).Info("Successfully fetched promotions from API")
	
	return &promotions, nil
}

func convertAPIBundlesToInternal(apiBundles []FanaticalAPIBundle) []FanaticalBundle {
	var bundles []FanaticalBundle
	var skippedCount int
	
	for _, apiBundle := range apiBundles {
		// Skip invalid bundles
		if apiBundle.Name == "" {
			skippedCount++
			continue
		}
		
		// Skip expired bundles
		if isExpired(apiBundle.ValidUntil) {
			skippedCount++
			continue
		}
		
		// Skip giveaways unless they have meaningful content
		if apiBundle.Giveaway && apiBundle.GameTotal < 5 {
			skippedCount++
			continue
		}
		
		// Get USD price, fallback to other currencies
		price := getPrice(apiBundle.Price, "USD")
		originalPrice := getPrice(apiBundle.FullPrice, "USD")
		
		// Calculate discount if not provided
		discount := apiBundle.DiscountPercent
		if discount == 0 && originalPrice > price && originalPrice > 0 {
			discount = int(((originalPrice - price) / originalPrice) * 100)
		}
		
		// Use PNM saving as discount if available and higher
		if apiBundle.PNMSaving > discount {
			discount = apiBundle.PNMSaving
		}
		
		// Create enhanced description
		description := createEnhancedBundleDescription(apiBundle)
		
		// Determine URL based on type
		var url string
		switch apiBundle.Type {
		case "pick-and-mix":
			url = fmt.Sprintf("/en/pick-and-mix/%s", apiBundle.Slug)
		case "book-bundle":
			url = fmt.Sprintf("/en/pick-and-mix/%s", apiBundle.Slug)
		case "elearning-bundle":
			url = fmt.Sprintf("/en/pick-and-mix/%s", apiBundle.Slug)
		case "software-bundle":
			url = fmt.Sprintf("/en/pick-and-mix/%s", apiBundle.Slug)
		case "game":
			url = fmt.Sprintf("/en/game/%s", apiBundle.Slug)
		case "bundle":
			url = fmt.Sprintf("/en/bundle/%s", apiBundle.Slug)
		default:
			// Fallback: versuche herauszufinden basierend auf dem Slug
			if strings.Contains(apiBundle.Slug, "build-your-own") || strings.Contains(apiBundle.Slug, "bundle") {
				url = fmt.Sprintf("/en/pick-and-mix/%s", apiBundle.Slug)
			} else {
				url = fmt.Sprintf("/en/bundle/%s", apiBundle.Slug)
			}
		}
		
		bundle := FanaticalBundle{
			ID:          apiBundle.ProductID,
			Title:       apiBundle.Name,
			Description: description,
			URL:         url,
			Slug:        apiBundle.Slug,
			Category:    determineBundleCategory(apiBundle),
			StartDate:   time.Unix(apiBundle.ValidFrom, 0),
			EndDate:     time.Unix(apiBundle.ValidUntil, 0),
			IsActive:    apiBundle.OnSale && !isExpired(apiBundle.ValidUntil),
			Price: Price{
				Currency: "USD",
				Amount:   price,
				Original: originalPrice,
				Discount: discount,
			},
		}
		
		// Only include active bundles
		if bundle.IsActive {
			bundles = append(bundles, bundle)
		} else {
			skippedCount++
		}
	}
	
	log.WithFields(log.Fields{
		"total":    len(apiBundles),
		"active":   len(bundles),
		"skipped":  skippedCount,
	}).Info("Bundle conversion completed")
	
	return bundles
}

// Verbesserte Bundle-Beschreibung
func createEnhancedBundleDescription(apiBundle FanaticalAPIBundle) string {
	parts := []string{}
	
	// Type-specific descriptions
	switch apiBundle.Type {
	case "pick-and-mix":
		parts = append(parts, "🎯 Build your own bundle")
	case "bundle":
		parts = append(parts, "📦 Bundle")
	case "game":
		if apiBundle.StarDeal {
			parts = append(parts, "⭐ Star Deal")
		}
	case "book-bundle":
		parts = append(parts, "📚 Book Bundle")
	case "elearning-bundle":
		parts = append(parts, "🎓 Learning Bundle")
	case "software-bundle":
		parts = append(parts, "💻 Software Bundle")
	}
	
	// Game/DLC count
	if apiBundle.GameTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d games", apiBundle.GameTotal))
	}
	if apiBundle.DLCTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d DLC", apiBundle.DLCTotal))
	}
	
	// Special indicators
	if apiBundle.BestEver {
		parts = append(parts, "🏆 Best price ever!")
	}
	if apiBundle.FlashSale {
		parts = append(parts, "⚡ Flash sale!")
	}
	if apiBundle.StarDeal {
		parts = append(parts, "⭐ Star Deal")
	}
	if apiBundle.Giveaway {
		parts = append(parts, "🎁 FREE")
	}
	
	// DRM info
	if len(apiBundle.DRM) > 0 {
		parts = append(parts, fmt.Sprintf("DRM: %s", strings.Join(apiBundle.DRM, ", ")))
	}
	
	// Operating systems
	if len(apiBundle.OperatingSystems) > 0 {
		parts = append(parts, fmt.Sprintf("OS: %s", strings.Join(apiBundle.OperatingSystems, ", ")))
	}
	
	// Use German description if available
	if len(apiBundle.SDescs) > 0 {
		if germanDesc, exists := apiBundle.SDescs["de"]; exists && germanDesc != "" {
			// Nehme die ersten 100 Zeichen der deutschen Beschreibung
			if len(germanDesc) > 100 {
				germanDesc = germanDesc[:100] + "..."
			}
			parts = append(parts, germanDesc)
		}
	}
	
	if len(parts) == 0 {
		return "Great gaming content with amazing savings"
	}
	
	return strings.Join(parts, " • ")
}

// Alle anderen Funktionen bleiben unverändert...
func convertPromotionsToBundles(promotions *PromotionsResponse, category string) []FanaticalBundle {
	var bundles []FanaticalBundle
	
	// Convert free products to bundles
	for _, freeProduct := range promotions.FreeProducts {
		if !freeProduct.Public {
			continue
		}
		
		validUntil, err := time.Parse(time.RFC3339, freeProduct.ValidUntil)
		if err != nil || validUntil.Before(time.Now()) {
			continue
		}
		
		validFrom, err := time.Parse(time.RFC3339, freeProduct.ValidFrom)
		if err != nil {
			validFrom = time.Now()
		}
		
		for _, product := range freeProduct.Products {
			if !product.IsVisible {
				continue
			}
			
			bundle := FanaticalBundle{
				ID:          product.ID,
				Title:       product.Name,
				Description: createFreeProductDescription(freeProduct, product),
				URL:         fmt.Sprintf("/en/game/%s", product.Slug),
				Slug:        product.Slug,
				Category:    determineProductCategory(product),
				StartDate:   validFrom,
				EndDate:     validUntil,
				IsActive:    true,
				Price: Price{
					Currency: "USD",
					Amount:   0,
					Original: getPrice(product.Price, "USD"),
					Discount: 100,
				},
			}
			
			bundles = append(bundles, bundle)
		}
	}
	
	// Convert vouchers to special entries (for games category only)
	if category == "games" {
		for _, voucher := range promotions.Vouchers {
			if !voucher.Public || !voucher.Game {
				continue
			}
			
			validUntil, err := time.Parse(time.RFC3339, voucher.ValidUntil)
			if err != nil || validUntil.Before(time.Now()) {
				continue
			}
			
			validFrom, err := time.Parse(time.RFC3339, voucher.ValidFrom)
			if err != nil {
				validFrom = time.Now()
			}
			
			bundle := FanaticalBundle{
				ID:          voucher.ID,
				Title:       fmt.Sprintf("Voucher: %s", voucher.Title),
				Description: createVoucherDescription(voucher),
				URL:         "/en/bundles",
				Slug:        "voucher-" + voucher.Code,
				Category:    "games",
				StartDate:   validFrom,
				EndDate:     validUntil,
				IsActive:    true,
				Price: Price{
					Currency: "USD",
					Amount:   0,
					Original: 0,
					Discount: voucher.Percent,
				},
			}
			
			bundles = append(bundles, bundle)
		}
	}
	
	return bundles
}

func createFreeProductDescription(freeProduct FreeProduct, product PromotionProduct) string {
	parts := []string{"🎁 FREE"}
	
	if product.Mystery {
		parts = append(parts, "Mystery Game")
	}
	
	if freeProduct.PartnerBrand != "" {
		parts = append(parts, fmt.Sprintf("Partner: %s", freeProduct.PartnerBrand))
	}
	
	minSpendUSD := getPrice(freeProduct.MinSpend, "USD")
	if minSpendUSD > 0 {
		parts = append(parts, fmt.Sprintf("Min spend: $%.0f", minSpendUSD))
	}
	
	return strings.Join(parts, " • ")
}

func createVoucherDescription(voucher Voucher) string {
	parts := []string{fmt.Sprintf("💰 %d%% OFF", voucher.Percent)}
	
	parts = append(parts, fmt.Sprintf("Code: %s", voucher.Code))
	
	if voucher.FullPriceOnly {
		parts = append(parts, "Full price only")
	}
	
	minSpendUSD := getPrice(voucher.MinSpend, "USD")
	if minSpendUSD > 0 {
		parts = append(parts, fmt.Sprintf("Min spend: $%.0f", minSpendUSD))
	}
	
	return strings.Join(parts, " • ")
}

func determineProductCategory(product PromotionProduct) string {
	productType := strings.ToLower(product.Type)
	name := strings.ToLower(product.Name)
	
	if productType == "book" || strings.Contains(name, "book") {
		return "books"
	}
	
	if productType == "software" || strings.Contains(name, "software") {
		return "software"
	}
	
	return "games"
}

func getPrice(priceMap map[string]float64, preferredCurrency string) float64 {
	if price, exists := priceMap[preferredCurrency]; exists && price > 0 {
		return price
	}
	
	// Fallback to other currencies
	for _, currency := range []string{"EUR", "GBP", "CAD", "AUD"} {
		if price, exists := priceMap[currency]; exists && price > 0 {
			return price
		}
	}
	
	// If no valid price found, return 0
	return 0
}

func determineBundleCategory(apiBundle FanaticalAPIBundle) string {
	name := strings.ToLower(apiBundle.Name)
	displayType := strings.ToLower(apiBundle.DisplayType)
	bundleType := strings.ToLower(apiBundle.Type)
	
	// Check display type first (most reliable)
	if strings.Contains(displayType, "book") {
		return "books"
	}
	if strings.Contains(displayType, "software") {
		return "software"
	}
	
	// Check bundle type
	if bundleType == "book-bundle" || bundleType == "elearning-bundle" {
		return "books"
	}
	if bundleType == "software-bundle" {
		return "software"
	}
	
	// Check bundle name for category hints
	if strings.Contains(name, "book") || 
	   strings.Contains(name, "rpg") || 
	   strings.Contains(name, "tabletop") ||
	   strings.Contains(name, "certification") ||
	   strings.Contains(name, "learning") {
		return "books"
	}
	
	if strings.Contains(name, "software") || 
	   strings.Contains(name, "app") ||
	   strings.Contains(name, "development") {
		return "software"
	}
	
	// Default to games for most bundles
	return "games"
}

func isExpired(validUntil int64) bool {
	return time.Now().Unix() > validUntil
}

func shouldIncludeBundle(bundle FanaticalBundle, category string) bool {
	bundleCategory := strings.ToLower(bundle.Category)
	targetCategory := strings.ToLower(category)
	
	// Direct category match
	if bundleCategory == targetCategory {
		return true
	}
	
	// For backwards compatibility, check title and description
	title := strings.ToLower(bundle.Title)
	description := strings.ToLower(bundle.Description)
	
	switch targetCategory {
	case "books":
		return strings.Contains(title, "book") || 
		       strings.Contains(title, "rpg") ||
		       strings.Contains(title, "tabletop") ||
		       strings.Contains(description, "book") ||
		       strings.Contains(description, "rpg") ||
		       strings.Contains(title, "certification") ||
		       strings.Contains(title, "learning")
	case "games":
		return !strings.Contains(title, "book") && 
		       !strings.Contains(title, "software") &&
		       !strings.Contains(title, "rpg") &&
		       (bundleCategory == "games" || 
		        strings.Contains(title, "game") || 
		        strings.Contains(title, "steam") ||
		        strings.Contains(description, "game") ||
		        strings.Contains(title, "voucher"))
	case "software":
		return strings.Contains(title, "software") ||
		       strings.Contains(title, "app") ||
		       strings.Contains(description, "software") ||
		       strings.Contains(description, "app")
	default:
		return true
	}
}

func createTestBundle(category string) []FanaticalBundle {
	return []FanaticalBundle{
		{
			ID:          fmt.Sprintf("%s-test", category),
			Title:       fmt.Sprintf("Test %s Bundle", strings.Title(category)),
			Description: fmt.Sprintf("Test bundle for %s category - no active bundles found", category),
			URL:         "/en/bundle/test",
			Category:    category,
			StartDate:   time.Now(),
			EndDate:     time.Now().Add(30 * 24 * time.Hour),
			IsActive:    true,
			Price: Price{
				Currency: "USD",
				Amount:   9.99,
				Original: 49.99,
				Discount: 80,
			},
		},
	}
}

func writeFeedToFile(feed feeds.Feed, category string) error {
	// Ensure docs directory exists
	if err := os.MkdirAll("docs", 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	// Write RSS file to docs directory
	filename := fmt.Sprintf("docs/%s.rss", category)
	f, err := os.OpenFile(
		filename,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to create RSS file %s: %w", filename, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	rss, err := feed.ToRss()
	if err != nil {
		return fmt.Errorf("failed to generate RSS content: %w", err)
	}

	if _, err := w.WriteString(rss); err != nil {
		return fmt.Errorf("failed to write RSS content: %w", err)
	}

	// Manual flush to ensure RSS feeds are created
	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush RSS file: %w", err)
	}

	log.WithFields(log.Fields{
		"category": category,
		"file":     filename,
		"size":     len(rss),
	}).Info("RSS feed written successfully")
	return nil
}
