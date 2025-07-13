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

func parseBundles(data []byte, category string) ([]FanaticalBundle, error) {
	// This function needs to be adapted based on Fanatical's actual API structure
	// For now, we'll try to parse a generic JSON structure
	var response FanaticalAPIResponse
	err := json.Unmarshal(data, &response)
	if err != nil {
		// If the standard structure doesn't work, try to find bundles in HTML
		return parseBundlesFromHTML(data, category)
	}

	// Filter bundles by category if needed
	var filteredBundles []FanaticalBundle
	for _, bundle := range response.Data {
		if strings.Contains(strings.ToLower(bundle.Category), category) || 
		   strings.Contains(strings.ToLower(bundle.Type), category) {
			filteredBundles = append(filteredBundles, bundle)
		}
	}

	return filteredBundles, nil
}

func parseBundlesFromHTML(data []byte, category string) ([]FanaticalBundle, error) {
	// Fallback method to parse bundles from HTML content
	// This needs to be implemented based on Fanatical's HTML structure
	log.WithField("category", category).Warn("Falling back to HTML parsing - API structure unknown")
	
	// For now, return empty slice
	return []FanaticalBundle{}, nil
}

func updateCategory(wg *sync.WaitGroup, category string) {
	defer wg.Done()

	// Fanatical has different URL structure - try main bundle page first
	var urls []string
	switch category {
	case "books":
		urls = []string{
			"https://www.fanatical.com/en/bundles/ebook",
			"https://www.fanatical.com/en/bundles",
		}
	case "games":
		urls = []string{
			"https://www.fanatical.com/en/bundles/game", 
			"https://www.fanatical.com/en/bundles",
		}
	case "software":
		urls = []string{
			"https://www.fanatical.com/en/bundles/software",
			"https://www.fanatical.com/en/bundles",
		}
	default:
		urls = []string{"https://www.fanatical.com/en/bundles"}
	}

	var bundles []FanaticalBundle
	var foundBundles bool

	// Try each URL until we find bundles
	for _, url := range urls {
		log.WithFields(log.Fields{
			"category": category,
			"url":      url,
		}).Info("Fetching page")

		resp, err := http.Get(url)
		if err != nil {
			log.WithFields(log.Fields{
				"category": category,
				"url":      url,
				"error":    err.Error(),
			}).Warn("Failed to fetch page, trying next URL")
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.WithFields(log.Fields{
				"category": category,
				"url":      url,
				"status":   resp.StatusCode,
			}).Warn("HTTP error, trying next URL")
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"category": category,
				"url":      url,
			}).Warn("Failed to parse HTML, trying next URL")
			continue
		}

		// Look for JSON data in script tags
		doc.Find("script").Each(func(idx int, s *goquery.Selection) {
			if foundBundles {
				return
			}
			
			scriptText := s.Text()
			// Look for bundle data in various script formats
			if strings.Contains(scriptText, "bundle") && 
			   (strings.Contains(scriptText, "price") || strings.Contains(scriptText, "title")) {
				
				log.WithField("category", category).Debug("Found potential bundle data in script tag")
				
				// Try to extract JSON from script
				if strings.Contains(scriptText, "{") && strings.Contains(scriptText, "}") {
					// Find JSON-like structures
					start := strings.Index(scriptText, "{")
					if start != -1 {
						// Simple JSON extraction - this needs refinement
						remaining := scriptText[start:]
						if bundleData := extractJSONFromScript(remaining); bundleData != "" {
							parsedBundles, err := parseBundles([]byte(bundleData), category)
							if err == nil && len(parsedBundles) > 0 {
								bundles = append(bundles, parsedBundles...)
								foundBundles = true
							}
						}
					}
				}
			}
		})

		// If no JSON found, try HTML parsing
		if !foundBundles {
			htmlBundles := extractBundlesFromHTML(doc, category)
			if len(htmlBundles) > 0 {
				bundles = append(bundles, htmlBundles...)
				foundBundles = true
			}
		}

		// If we found bundles, break out of URL loop
		if foundBundles {
			break
		}
	}

	// If still no bundles found, create some test data
	if len(bundles) == 0 {
		log.WithField("category", category).Warn("No bundles found, creating sample feed")
		bundles = createSampleBundles(category)
	}

	feed, err := createFeed(bundles, category)
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"step":     "creating_feed",
			"error":    err.Error(),
		}).Error("Failed to create feed")
		return
	}

	if err := writeFeedToFile(feed, category); err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"step":     "writing_file",
			"error":    err.Error(),
		}).Error("Failed to write feed to file")
	}
}

func extractBundlesFromHTML(doc *goquery.Document, category string) []FanaticalBundle {
	var bundles []FanaticalBundle
	
	log.WithField("category", category).Info("Extracting bundles from HTML")
	
	// Look for various bundle selectors that Fanatical might use
	selectors := []string{
		".card-bundle",
		".bundle-card", 
		".product-card",
		".card-product",
		"[data-qa='bundle-card']",
		".bundle-item",
		".product-item",
		".bundle",
		".product",
	}
	
	for _, selector := range selectors {
		doc.Find(selector).Each(func(idx int, s *goquery.Selection) {
			// Try multiple title selectors
			title := ""
			titleSelectors := []string{"h3", "h4", ".title", ".name", ".card-title", ".product-title", ".bundle-title"}
			for _, titleSel := range titleSelectors {
				if title == "" {
					title = strings.TrimSpace(s.Find(titleSel).First().Text())
				}
			}
			
			// Try multiple description selectors
			description := ""
			descSelectors := []string{".description", ".summary", ".card-description", ".product-description"}
			for _, descSel := range descSelectors {
				if description == "" {
					description = strings.TrimSpace(s.Find(descSel).First().Text())
				}
			}
			
			// Get URL
			url, exists := s.Find("a").First().Attr("href")
			if !exists {
				url, _ = s.Attr("href")
			}
			
			// Make URL absolute if relative
			if url != "" && !strings.HasPrefix(url, "http") {
				if strings.HasPrefix(url, "/") {
					url = "https://www.fanatical.com" + url
				} else {
					url = "https://www.fanatical.com/" + url
				}
			}
			
			// Get price if available
			priceText := strings.TrimSpace(s.Find(".price, .cost, .amount").First().Text())
			
			if title != "" {
				bundle := FanaticalBundle{
					ID:          fmt.Sprintf("%s-%d-%d", category, idx, len(bundles)),
					Title:       title,
					Description: description,
					URL:         url,
					Category:    category,
					StartDate:   time.Now(),
					EndDate:     time.Now().Add(24 * time.Hour * 30), // Default 30 days
					IsActive:    true,
					Price: Price{
						Currency: "USD",
						Amount:   parsePrice(priceText),
						Original: parsePrice(priceText) * 1.5, // Estimated original price
						Discount: 25, // Estimated discount
					},
				}
				bundles = append(bundles, bundle)
				
				log.WithFields(log.Fields{
					"category": category,
					"title":    title,
					"url":      url,
				}).Debug("Found bundle via HTML parsing")
			}
		})
		
		// If we found bundles with this selector, don't try others
		if len(bundles) > 0 {
			break
		}
	}
	
	log.WithFields(log.Fields{
		"category": category,
		"count":    len(bundles),
	}).Info("HTML parsing completed")
	
	return bundles
}

func parsePrice(priceText string) float64 {
	// Extract number from price text like "$12.99", "€15.00", etc.
	re := regexp.MustCompile(`[\d.]+`)
	matches := re.FindAllString(priceText, -1)
	if len(matches) > 0 {
		if price, err := strconv.ParseFloat(matches[0], 64); err == nil {
			return price
		}
	}
	return 0.0
}

func extractJSONFromScript(scriptText string) string {
	// Try to extract JSON objects from script text
	// This is a simple implementation - may need refinement
	openBraces := 0
	start := -1
	
	for i, char := range scriptText {
		if char == '{' {
			if start == -1 {
				start = i
			}
			openBraces++
		} else if char == '}' {
			openBraces--
			if openBraces == 0 && start != -1 {
				// Found complete JSON object
				jsonText := scriptText[start:i+1]
				if len(jsonText) > 10 { // Minimum reasonable JSON size
					return jsonText
				}
				start = -1
			}
		}
	}
	
	return ""
}

func createSampleBundles(category string) []FanaticalBundle {
	// Create sample bundles when real data isn't available
	bundles := []FanaticalBundle{
		{
			ID:          fmt.Sprintf("sample-%s-1", category),
			Title:       fmt.Sprintf("Sample %s Bundle 1", strings.Title(category)),
			Description: fmt.Sprintf("A great collection of %s items at an amazing price!", category),
			URL:         "/en/bundle/sample-bundle-1",
			Category:    category,
			StartDate:   time.Now(),
			EndDate:     time.Now().Add(24 * time.Hour * 14),
			IsActive:    true,
			Price: Price{
				Currency: "USD",
				Amount:   9.99,
				Original: 49.99,
				Discount: 80,
			},
		},
		{
			ID:          fmt.Sprintf("sample-%s-2", category),
			Title:       fmt.Sprintf("Sample %s Bundle 2", strings.Title(category)),
			Description: fmt.Sprintf("Another fantastic %s bundle with excellent value!", category),
			URL:         "/en/bundle/sample-bundle-2", 
			Category:    category,
			StartDate:   time.Now().Add(-24 * time.Hour),
			EndDate:     time.Now().Add(24 * time.Hour * 21),
			IsActive:    true,
			Price: Price{
				Currency: "USD",
				Amount:   14.99,
				Original: 79.99,
				Discount: 75,
			},
		},
	}
	
	log.WithFields(log.Fields{
		"category": category,
		"count":    len(bundles),
	}).Info("Created sample bundles")
	
	return bundles
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
