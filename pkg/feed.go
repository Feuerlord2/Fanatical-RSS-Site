package gofanatical

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
	log "github.com/sirupsen/logrus"
)

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

		feed.Items[idx] = &feeds.Item{
			Title:       bundle.Title,
			Link:        &feeds.Link{Href: fmt.Sprintf("https://www.fanatical.com%s", bundle.URL)},
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

	// Try to fetch bundles from Fanatical website
	log.WithField("category", category).Info("Fetching bundles from Fanatical website")
	
	url := "https://www.fanatical.com/en/bundles"
	
	// Create HTTP client with headers to avoid blocking
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to create request")
		return
	}
	
	// Add headers to appear like a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to fetch bundles page")
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"category": category,
			"status":   resp.StatusCode,
		}).Error("HTTP error when fetching bundles")
		return
	}
	
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.WithField("category", category).Error("Failed to parse HTML")
		return
	}
	
	// Try to extract bundles from the page
	bundles := extractBundlesFromFanatical(doc, category)
	
	if len(bundles) == 0 {
		log.WithField("category", category).Warn("No bundles found on website")
		return
	}
	
	feed, err := createFeed(bundles, category)
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
	}
}

func extractBundlesFromFanatical(doc *goquery.Document, category string) []FanaticalBundle {
	var bundles []FanaticalBundle
	
	log.WithField("category", category).Info("Extracting bundles from Fanatical HTML")
	
	// Look for script tags that might contain bundle data (similar to Humble Bundle approach)
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		scriptContent := s.Text()
		
		// Look for JSON-like data that contains bundle information
		if strings.Contains(scriptContent, "bundle") && strings.Contains(scriptContent, "title") {
			log.WithField("category", category).Debug("Found potential bundle data in script tag")
			
			// Try to extract JSON objects from script
			bundles = append(bundles, extractBundlesFromScript(scriptContent, category)...)
		}
	})
	
	// If no bundles found in scripts, try HTML parsing
	if len(bundles) == 0 {
		log.WithField("category", category).Info("No bundles found in scripts, trying HTML parsing")
		bundles = extractBundlesFromHTML(doc, category)
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
	}).Info("Bundle extraction completed")
	
	return filteredBundles
}

func extractBundlesFromScript(scriptContent, category string) []FanaticalBundle {
	var bundles []FanaticalBundle
	
	// Try to find JSON objects in the script
	re := regexp.MustCompile(`\{[^{}]*"[^"]*bundle[^"]*"[^{}]*\}`)
	matches := re.FindAllString(scriptContent, -1)
	
	for _, match := range matches {
		// Try to parse as JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(match), &data); err == nil {
			if bundle := parseJSONBundle(data, category); bundle != nil {
				bundles = append(bundles, *bundle)
			}
		}
	}
	
	return bundles
}

func extractBundlesFromHTML(doc *goquery.Document, category string) []FanaticalBundle {
	var bundles []FanaticalBundle
	
	// Try different selectors that Fanatical might use for bundle cards
	selectors := []string{
		"[data-qa*='bundle']",
		"[data-testid*='bundle']", 
		"[data-testid*='product']",
		".bundle-card",
		".product-card",
		".card",
		"article",
		"[class*='bundle']",
		"[class*='product']",
		"[class*='card']",
	}
	
	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			if bundle := parseHTMLBundle(s, category); bundle != nil {
				bundles = append(bundles, *bundle)
			}
		})
		
		// If we found bundles with this selector, use them
		if len(bundles) > 0 {
			break
		}
	}
	
	return bundles
}

func parseJSONBundle(data map[string]interface{}, category string) *FanaticalBundle {
	// Extract bundle information from JSON data
	title, hasTitle := data["title"].(string)
	if !hasTitle || title == "" {
		return nil
	}
	
	bundle := &FanaticalBundle{
		ID:        fmt.Sprintf("%s-%d", category, time.Now().Unix()),
		Title:     title,
		Category:  category,
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now().Add(30 * 24 * time.Hour),
		IsActive:  true,
		Price: Price{
			Currency: "USD",
			Amount:   9.99,
			Original: 49.99,
			Discount: 80,
		},
	}
	
	// Try to extract additional fields
	if desc, hasDesc := data["description"].(string); hasDesc {
		bundle.Description = desc
	}
	
	if url, hasURL := data["url"].(string); hasURL {
		bundle.URL = url
	}
	
	if price, hasPrice := data["price"].(float64); hasPrice {
		bundle.Price.Amount = price
	}
	
	return bundle
}

func parseHTMLBundle(s *goquery.Selection, category string) *FanaticalBundle {
	// Extract title from various possible elements
	title := ""
	titleSelectors := []string{"h1", "h2", "h3", "h4", ".title", ".name", "[data-qa*='title']"}
	
	for _, sel := range titleSelectors {
		if title == "" {
			title = strings.TrimSpace(s.Find(sel).First().Text())
		}
	}
	
	// Skip if no meaningful title found
	if title == "" || len(title) < 3 {
		return nil
	}
	
	// Get URL
	url, _ := s.Find("a").First().Attr("href")
	if url != "" && !strings.HasPrefix(url, "http") {
		if strings.HasPrefix(url, "/") {
			url = "https://www.fanatical.com" + url
		}
	}
	
	// Get description
	description := strings.TrimSpace(s.Find(".description, .summary, p").First().Text())
	
	// Get price
	priceText := strings.TrimSpace(s.Find("[class*='price'], .cost, .amount").First().Text())
	price := parsePrice(priceText)
	
	bundle := &FanaticalBundle{
		ID:          fmt.Sprintf("%s-%d", category, time.Now().UnixNano()),
		Title:       title,
		Description: description,
		URL:         url,
		Category:    category,
		StartDate:   time.Now().Add(-24 * time.Hour),
		EndDate:     time.Now().Add(30 * 24 * time.Hour),
		IsActive:    true,
		Price: Price{
			Currency: "USD",
			Amount:   price,
			Original: price * 2.0,
			Discount: 50,
		},
	}
	
	return bundle
}

func parsePrice(priceText string) float64 {
	// Extract price from text like "$12.99", "€15.00", etc.
	re := regexp.MustCompile(`[\d.]+`)
	matches := re.FindAllString(priceText, -1)
	if len(matches) > 0 {
		if price, err := strconv.ParseFloat(matches[0], 64); err == nil {
			return price
		}
	}
	return 9.99 // Default price
}

func shouldIncludeBundle(bundle FanaticalBundle, category string) bool {
	title := strings.ToLower(bundle.Title)
	description := strings.ToLower(bundle.Description)
	
	switch category {
	case "books":
		return strings.Contains(title, "book") || 
		       strings.Contains(title, "ebook") ||
		       strings.Contains(description, "book")
	case "games":
		return !strings.Contains(title, "book") && 
		       !strings.Contains(title, "software") &&
		       (strings.Contains(title, "game") || 
		        strings.Contains(title, "steam") ||
		        strings.Contains(description, "game"))
	case "software":
		return strings.Contains(title, "software") ||
		       strings.Contains(title, "app") ||
		       strings.Contains(description, "software")
	default:
		return true
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
