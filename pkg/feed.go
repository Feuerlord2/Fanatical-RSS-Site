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

// Algolia API Bundle Struktur
type AlgoliaBundle struct {
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
	OperatingSystems []string         `json:"operating_systems"`
	DRM             []string          `json:"drm"`
	Features        []string          `json:"features"`
	Categories      []string          `json:"categories"`
	ValidFrom       int64             `json:"available_valid_from"`
	ValidUntil      int64             `json:"available_valid_until"`
	Position        int               `json:"position"`
	ReleaseDate     int64             `json:"release_date"`
	Giveaway        bool              `json:"giveaway"`
	GameTotal       int               `json:"game_total"`
	DLCTotal        int               `json:"dlc_total"`
	BundleCovers    []BundleGame      `json:"bundle_covers"`
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

func Run() {
	wg := sync.WaitGroup{}
	// Nur noch 3 Kategorien - kein Fallback mehr
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
		content := createRichContent(bundle)
		title := bundle.Title // Nur der Bundle-Name

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

func removeDuplicateBundles(bundles []FanaticalBundle) []FanaticalBundle {
	seen := make(map[string]bool)
	var uniqueBundles []FanaticalBundle
	
	for _, bundle := range bundles {
		key := fmt.Sprintf("%s-%d", bundle.Slug, bundle.StartDate.Unix())
		
		if !seen[key] {
			seen[key] = true
			uniqueBundles = append(uniqueBundles, bundle)
		} else {
			log.WithFields(log.Fields{
				"bundle_title": bundle.Title,
				"bundle_slug":  bundle.Slug,
				"duplicate_key": key,
			}).Info("Duplicate bundle removed")
		}
	}
	
	log.WithFields(log.Fields{
		"original_count": len(bundles),
		"unique_count":   len(uniqueBundles),
		"duplicates_removed": len(bundles) - len(uniqueBundles),
	}).Info("Duplicate removal completed")
	
	return uniqueBundles
}

func updateCategory(wg *sync.WaitGroup, category string) {
	defer wg.Done()

	log.WithField("category", category).Info("Fetching data from Fanatical Algolia API")
	
	// Nur noch Algolia API
	bundles, err := fetchBundlesFromAlgoliaAPI()
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to fetch bundles from Algolia API")
		return
	}
	
	log.WithField("total_bundles_before_dedup", len(bundles)).Info("Total bundles collected from Algolia API")
	
	// Entferne Duplikate
	bundles = removeDuplicateBundles(bundles)
	log.WithField("total_bundles_after_dedup", len(bundles)).Info("Total bundles after duplicate removal")
	
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
	
	// If no bundles found, create test bundle
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

func fetchBundlesFromAlgoliaAPI() ([]FanaticalBundle, error) {
	url := "https://www.fanatical.com/api/algolia/bundles?altRank=false"
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Headers ohne Accept-Encoding (verhindert Compression)
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
		return nil, fmt.Errorf("failed to fetch Algolia API: %w", err)
	}
	defer resp.Body.Close()
	
	log.WithField("status", resp.StatusCode).Info("Algolia API response received")
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Algolia API returned status %d", resp.StatusCode)
	}
	
	var algoliBundles []AlgoliaBundle
	if err := json.NewDecoder(resp.Body).Decode(&algoliBundles); err != nil {
		return nil, fmt.Errorf("failed to decode Algolia API response: %w", err)
	}
	
	log.WithField("bundles", len(algoliBundles)).Info("Successfully fetched bundles from Algolia API")
	
	return convertAlgoliaBundlesToInternal(algoliBundles), nil
}

func convertAlgoliaBundlesToInternal(algoliBundles []AlgoliaBundle) []FanaticalBundle {
	var bundles []FanaticalBundle
	var skippedCount int
	
	for _, algoliaBundle := range algoliBundles {
		// Skip invalid bundles
		if algoliaBundle.Name == "" {
			skippedCount++
			continue
		}
		
		// Skip expired bundles
		if isExpired(algoliaBundle.ValidUntil) {
			skippedCount++
			continue
		}
		
		// Get USD price, fallback to other currencies
		price := getPrice(algoliaBundle.Price, "USD")
		originalPrice := getPrice(algoliaBundle.FullPrice, "USD")
		
		// Calculate discount if not provided
		discount := algoliaBundle.DiscountPercent
		if discount == 0 && originalPrice > price && originalPrice > 0 {
			discount = int(((originalPrice - price) / originalPrice) * 100)
		}
		
		// Create enhanced description
		description := createEnhancedBundleDescription(algoliaBundle)
		
		// Determine URL based on type
		var url string
		switch algoliaBundle.Type {
		case "pick-and-mix":
			url = fmt.Sprintf("/en/pick-and-mix/%s", algoliaBundle.Slug)
		case "game":
			url = fmt.Sprintf("/en/game/%s", algoliaBundle.Slug)
		case "bundle":
			url = fmt.Sprintf("/en/bundle/%s", algoliaBundle.Slug)
		default:
			url = fmt.Sprintf("/en/bundle/%s", algoliaBundle.Slug)
		}
		
		bundle := FanaticalBundle{
			ID:          algoliaBundle.ProductID,
			Title:       algoliaBundle.Name,
			Description: description,
			URL:         url,
			Slug:        algoliaBundle.Slug,
			Category:    determineBundleCategory(algoliaBundle),
			StartDate:   time.Unix(algoliaBundle.ValidFrom, 0),
			EndDate:     time.Unix(algoliaBundle.ValidUntil, 0),
			IsActive:    algoliaBundle.OnSale && !isExpired(algoliaBundle.ValidUntil),
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
		"total":    len(algoliBundles),
		"active":   len(bundles),
		"skipped":  skippedCount,
	}).Info("Bundle conversion completed")
	
	return bundles
}

func createEnhancedBundleDescription(algoliaBundle AlgoliaBundle) string {
	parts := []string{}
	
	// Type-specific descriptions
	switch algoliaBundle.Type {
	case "pick-and-mix":
		parts = append(parts, "🎯 Build your own bundle")
	case "bundle":
		parts = append(parts, "📦 Bundle")
	case "game":
		if algoliaBundle.StarDeal {
			parts = append(parts, "⭐ Star Deal")
		}
	}
	
	// Game/DLC count
	if algoliaBundle.GameTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d items", algoliaBundle.GameTotal))
	}
	if algoliaBundle.DLCTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d DLC", algoliaBundle.DLCTotal))
	}
	
	// Special indicators
	if algoliaBundle.BestEver {
		parts = append(parts, "🏆 Best price ever!")
	}
	if algoliaBundle.FlashSale {
		parts = append(parts, "⚡ Flash sale!")
	}
	if algoliaBundle.StarDeal {
		parts = append(parts, "⭐ Star Deal")
	}
	if algoliaBundle.Giveaway {
		parts = append(parts, "🎁 FREE")
	}
	
	// DRM info
	if len(algoliaBundle.DRM) > 0 {
		parts = append(parts, fmt.Sprintf("DRM: %s", strings.Join(algoliaBundle.DRM, ", ")))
	}
	
	// Operating systems
	if len(algoliaBundle.OperatingSystems) > 0 {
		parts = append(parts, fmt.Sprintf("OS: %s", strings.Join(algoliaBundle.OperatingSystems, ", ")))
	}
	
	if len(parts) == 0 {
		return "Great content with amazing savings"
	}
	
	return strings.Join(parts, " • ")
}

func determineBundleCategory(algoliaBundle AlgoliaBundle) string {
	name := strings.ToLower(algoliaBundle.Name)
	bundleType := strings.ToLower(algoliaBundle.Type)
	displayType := strings.ToLower(algoliaBundle.DisplayType)
	
	log.WithFields(log.Fields{
		"bundle_name":    algoliaBundle.Name,
		"bundle_type":    algoliaBundle.Type,
		"display_type":   algoliaBundle.DisplayType,
	}).Debug("Determining bundle category")
	
	// Check display_type FIRST (most reliable)
	switch displayType {
	case "book-bundle":
		return "books"
	case "comic-bundle":
		return "books"
	case "software-bundle":
		return "software"
	case "audio-bundle":
		return "software"  // Audio = Software
	case "elearning-bundle":
		return "software"  // E-Learning = Software
	case "bundle":
		// Normale Game Bundles
		return "games"
	}
	
	// Check bundle type as fallback
	switch bundleType {
	case "bundle":
		// Schaue in den Namen für spezielle Kategorien
		if strings.Contains(name, "software") || strings.Contains(name, "excel") {
			return "software"
		}
		if strings.Contains(name, "book") || strings.Contains(name, "certification") || strings.Contains(name, "learning") {
			return "books"
		}
		return "games" // Default für normale bundles
	case "pick-and-mix":
		return "games" // Pick-and-mix sind meist Games
	case "game":
		return "games"
	default:
		return "games" // Default
	}
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
	
	return 0
}

func isExpired(validUntil int64) bool {
	return time.Now().Unix() > validUntil
}

// VEREINFACHTE shouldIncludeBundle - basiert hauptsächlich auf display_type
func shouldIncludeBundle(bundle FanaticalBundle, category string) bool {
	bundleCategory := strings.ToLower(bundle.Category)
	targetCategory := strings.ToLower(category)
	
	log.WithFields(log.Fields{
		"bundle_title":    bundle.Title,
		"bundle_category": bundleCategory,
		"target_category": targetCategory,
	}).Debug("Checking bundle for category")
	
	// Direct category match (hauptsächlich basierend auf display_type)
	if bundleCategory == targetCategory {
		return true
	}
	
	title := strings.ToLower(bundle.Title)
	
	switch targetCategory {
	case "books":
		// WICHTIG: Software-Bundles explizit ausschließen
		if bundleCategory == "software" {
			return false
		}
		
		// Spezielle Titel-basierte Checks für Books
		shouldInclude := strings.Contains(title, "certification") ||
		       (strings.Contains(title, "learning") && bundleCategory != "software") || // Learning nur wenn nicht bereits Software
		       (strings.Contains(title, "training") && bundleCategory != "software") ||  // Training nur wenn nicht bereits Software
		       (strings.Contains(title, "course") && bundleCategory != "software") ||    // Course nur wenn nicht bereits Software
		       strings.Contains(title, "comic") ||
		       strings.Contains(title, "collection") && strings.Contains(title, "comic")
		       
		if shouldInclude {
			log.WithField("bundle_title", bundle.Title).Info("BOOKS: Bundle matched by title!")
		}
		return shouldInclude
		
	case "software":
		// Spezielle Titel-basierte Checks für Software
		shouldInclude := strings.Contains(title, "software") ||
		       strings.Contains(title, "app") ||
		       strings.Contains(title, "excel") ||
		       strings.Contains(title, "beats and vibes") ||
		       strings.Contains(title, "global beats")
		       
		if shouldInclude {
			log.WithField("bundle_title", bundle.Title).Info("SOFTWARE: Bundle matched by title!")
		}
		return shouldInclude
		
	case "games":
		// Games ist der Default - alles was nicht Books oder Software ist
		isBooks := bundleCategory == "books" ||
		          strings.Contains(title, "certification") ||
		          strings.Contains(title, "learning") ||
		          strings.Contains(title, "training") ||
		          strings.Contains(title, "course") ||
		          strings.Contains(title, "comic")
		          
		isSoftware := bundleCategory == "software" ||
		             strings.Contains(title, "software") ||
		             strings.Contains(title, "app") ||
		             strings.Contains(title, "excel") ||
		             strings.Contains(title, "beats and vibes") ||
		             strings.Contains(title, "global beats")
		
		shouldInclude := !isBooks && !isSoftware
		return shouldInclude
		
	default:
		return false
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
