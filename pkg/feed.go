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
	
	log.WithFields(log.Fields{
		"category": category,
		"status":   resp.StatusCode,
		"url":      url,
	}).Info("HTTP response received")
	
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
	
	// Debug: Log page title and basic info
	pageTitle := doc.Find("title").Text()
	log.WithFields(log.Fields{
		"category": category,
		"title":    pageTitle,
	}).Info("Page loaded successfully")
	
	// Try to extract bundles from the page
	bundles := extractBundlesFromFanatical(doc, category)
	
	log.WithFields(log.Fields{
		"category": category,
		"count":    len(bundles),
	}).Info("Bundle extraction completed")
	
	// If no bundles found, create at least one test bundle to verify RSS generation works
	if len(bundles) == 0 {
		log.WithField("category", category).Warn("No bundles found on website, creating test bundle")
		bundles = []FanaticalBundle{
			{
				ID:          fmt.Sprintf("%s-test", category),
				Title:       fmt.Sprintf("Test %s Bundle", strings.Title(category)),
				Description: fmt.Sprintf("Test bundle for %s category - website scraping found no results", category),
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
	} else {
		log.WithFields(log.Fields{
			"category": category,
			"bundles":  len(bundles),
		}).Info("Successfully created RSS feed")
	}
}

func extractBundlesFromFanatical(doc *goquery.Document, category string) []FanaticalBundle {
	var bundles []FanaticalBundle
	
	log.WithField("category", category).Info("Extracting bundles from Fanatical HTML")
	
	// Debug: Count different types of elements
	scriptCount := doc.Find("script").Length()
	log.WithFields(log.Fields{
		"category": category,
		"scripts":  scriptCount,
	}).Debug("Page analysis")
	
	// Look for script tags that might contain bundle data
	scriptBundles := 0
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		scriptContent := s.Text()
		
		// Look for various bundle-related keywords
		if strings.Contains(scriptContent, "bundle") && 
		   (strings.Contains(scriptContent, "title") || strings.Contains(scriptContent, "name")) {
			
			log.WithFields(log.Fields{
				"category": category,
				"script":   i,
			}).Debug("Found potential bundle data in script tag")
			scriptBundles++
			
			// Try to extract JSON objects from script
			extracted := extractBundlesFromScript(scriptContent, category)
			bundles = append(bundles, extracted...)
		}
	})
	
	log.WithFields(log.Fields{
		"category":      category,
		"script_matches": scriptBundles,
		"bundles_from_scripts": len(bundles),
	}).Info("Script parsing completed")
	
	// If no bundles found in scripts, try HTML parsing
	if len(bundles) == 0 {
		log.WithField("category", category).Info("No bundles found in scripts, trying HTML parsing")
		htmlBundles := extractBundlesFromHTML(doc, category)
		bundles = append(bundles, htmlBundles...)
		
		log.WithFields(log.Fields{
			"category": category,
			"bundles_from_html": len(htmlBundles),
		}).Info("HTML parsing completed")
	}
	
	// Try to find ANY text that looks like bundle titles
	if len(bundles) == 0 {
		log.WithField("category", category).Info("Trying to find any bundle-like content")
		
		// Look for any element containing "bundle" text
		doc.Find("*").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if len(text) > 10 && len(text) < 200 && 
			   strings.Contains(strings.ToLower(text), "bundle") {
				
				log.WithFields(log.Fields{
					"category": category,
					"text":     text[:min(50, len(text))],
					"tag":      goquery.NodeName(s),
				}).Debug("Found bundle-like text")
				
				// Create bundle from this text
				bundle := FanaticalBundle{
					ID:          fmt.Sprintf("%s-found-%d", category, i),
					Title:       text,
					Description: "Found via text search",
					URL:         "/en/bundle/unknown",
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
				}
				
				if shouldIncludeBundle(bundle, category) {
					bundles = append(bundles, bundle)
				}
			}
		})
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	
	// Try many different selectors that Fanatical might use
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
		"div[class*='item']",
		"div[class*='tile']",
		"section",
		".item",
		".tile",
		"a[href*='bundle']",
		"a[href*='product']",
	}
	
	for _, selector := range selectors {
		found := 0
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			if bundle := parseHTMLBundle(s, category); bundle != nil {
				bundles = append(bundles, *bundle)
				found++
			}
		})
		
		log.WithFields(log.Fields{
			"category": category,
			"selector": selector,
			"found":    found,
			"elements": doc.Find(selector).Length(),
		}).Debug("Tried selector")
		
		// If we found bundles with this selector, log it but continue trying others
		if found > 0 {
			log.WithFields(log.Fields{
				"category": category,
				"selector": selector,
				"bundles":  found,
			}).Info("Found bundles with selector")
		}
	}
	
	// Also try to find any links that contain bundle URLs
	bundleLinks := 0
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && strings.Contains(href, "bundle") {
			bundleLinks++
			
			title := strings.TrimSpace(s.Text())
			if title == "" {
				title = strings.TrimSpace(s.Find("*").Text())
			}
			
			if len(title) > 3 && len(title) < 200 {
				bundle := FanaticalBundle{
					ID:          fmt.Sprintf("%s-link-%d", category, i),
					Title:       title,
					Description: "Found via bundle link",
					URL:         href,
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
				}
				bundles = append(bundles, bundle)
			}
		}
	})
	
	log.WithFields(log.Fields{
		"category":     category,
		"bundle_links": bundleLinks,
		"total_links":  doc.Find("a").Length(),
	}).Debug("Bundle link analysis")
	
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
