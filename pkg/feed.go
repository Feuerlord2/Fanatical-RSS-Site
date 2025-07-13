package gofanatical

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/feeds"
	log "github.com/sirupsen/logrus"
)

// API Response structures for bundles API
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

// API Response structures for promotions API
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
		content := fmt.Sprintf("%s\n\nPrice: %s %.2f (Original: %.2f)\nDiscount: %d%%\nValid until: %s",
			bundle.Description,
			bundle.Price.Currency,
			bundle.Price.Amount,
			bundle.Price.Original,
			bundle.Price.Discount,
			bundle.EndDate.Format("2006-01-02 15:04"))

		if bundle.GameTotal > 0 {
			content += fmt.Sprintf("\nGames: %d", bundle.GameTotal)
		}

		feed.Items[idx] = &feeds.Item{
			Title:       bundle.Title,
			Link:        &feeds.Link{Href: fmt.Sprintf("https://www.fanatical.com/en/bundle/%s", bundle.Slug)},
			Content:     content,
			Created:     bundle.StartDate,
			Description: bundle.Description,
		}
	}

	// Sort items so that latest bundles are on the top
	sort.Slice(feed.Items, func(i, j int) bool { 
		return feed.Items[i].Created.After(feed.Items[j].Created) 
	})

	return feed, nil
}

func updateCategory(wg *sync.WaitGroup, category string) {
	defer wg.Done()

	log.WithField("category", category).Info("Fetching data from Fanatical APIs")
	
	// Fetch from both APIs
	bundles, err := fetchBundlesFromAPI()
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to fetch bundles from API")
		bundles = []FanaticalBundle{}
	}

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

func fetchBundlesFromAPI() ([]FanaticalBundle, error) {
	url := "https://www.fanatical.com/api/algolia/bundles?altRank=false"
	
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
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.fanatical.com/en/bundles")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bundles API: %w", err)
	}
	defer resp.Body.Close()
	
	log.WithField("status", resp.StatusCode).Info("Bundles API response received")
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bundles API returned status %d", resp.StatusCode)
	}
	
	var apiBundles []FanaticalAPIBundle
	if err := json.NewDecoder(resp.Body).Decode(&apiBundles); err != nil {
		return nil, fmt.Errorf("failed to decode bundles API response: %w", err)
	}
	
	log.WithField("bundles", len(apiBundles)).Info("Successfully fetched bundles from API")
	
	return convertAPIBundlesToInternal(apiBundles), nil
}

func fetchPromotionsFromAPI() (*PromotionsResponse, error) {
	url := "https://www.fanatical.com/api/all-promotions/en"
	
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
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
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
	
	for _, apiBundle := range apiBundles {
		// Skip giveaways (free bundles) unless they have meaningful content
		if apiBundle.Giveaway && apiBundle.GameTotal < 5 {
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
		
		// Create description from bundle info
		description := createBundleDescription(apiBundle)
		
		bundle := FanaticalBundle{
			ID:          apiBundle.ProductID,
			Title:       apiBundle.Name,
			Description: description,
			URL:         fmt.Sprintf("/en/bundle/%s", apiBundle.Slug),
			Slug:        apiBundle.Slug,
			Category:    determineBundleCategory(apiBundle),
			StartDate:   time.Unix(apiBundle.ValidFrom, 0),
			EndDate:     time.Unix(apiBundle.ValidUntil, 0),
			IsActive:    apiBundle.OnSale && !isExpired(apiBundle.ValidUntil),
			GameTotal:   apiBundle.GameTotal,
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
		}
	}
	
	return bundles
}

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
				GameTotal:   1,
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
				GameTotal:   0,
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

func createBundleDescription(apiBundle FanaticalAPIBundle) string {
	parts := []string{}
	
	if apiBundle.Type == "pick-and-mix" {
		parts = append(parts, "Build your own bundle")
	}
	
	if apiBundle.GameTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d games", apiBundle.GameTotal))
	}
	
	if apiBundle.DLCTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d DLC", apiBundle.DLCTotal))
	}
	
	if apiBundle.BestEver {
		parts = append(parts, "Best price ever!")
	}
	
	if apiBundle.FlashSale {
		parts = append(parts, "Flash sale!")
	}
	
	if apiBundle.StarDeal {
		parts = append(parts, "⭐ Star Deal")
	}
	
	if len(apiBundle.DRM) > 0 {
		parts = append(parts, fmt.Sprintf("DRM: %s", strings.Join(apiBundle.DRM, ", ")))
	}
	
	if len(parts) == 0 {
		return "Game bundle with great savings"
	}
	
	return strings.Join(parts, " • ")
}

func determineBundleCategory(apiBundle FanaticalAPIBundle) string {
	name := strings.ToLower(apiBundle.Name)
	displayType := strings.ToLower(apiBundle.DisplayType)
	
	// Check display type first
	if strings.Contains(displayType, "book") {
		return "books"
	}
	
	// Check bundle name for category hints
	if strings.Contains(name, "book") || strings.Contains(name, "rpg") || 
	   strings.Contains(name, "tabletop") {
		return "books"
	}
	
	if strings.Contains(name, "software") || strings.Contains(name, "app") {
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
		       strings.Contains(description, "rpg")
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
			GameTotal:   5,
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
	}).Info("RSS feed written successfully")
	return nil
}
