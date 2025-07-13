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

	// Try the bundle page first
	url := fmt.Sprintf("https://www.fanatical.com/en/bundle/%s", category)
	resp, err := http.Get(url)
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"url":      url,
			"error":    err.Error(),
		}).Error("Failed to fetch category page")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"category": category,
			"status":   resp.StatusCode,
		}).Error("HTTP error when fetching page")
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.WithField("category", category).Error("Failed to parse HTML")
		return
	}

	// Look for JSON data in script tags (similar to other bundle sites)
	var bundles []FanaticalBundle
	found := false

	// Try to find JSON data in script tags
	doc.Find("script[type='application/json']").Each(func(idx int, s *goquery.Selection) {
		if found {
			return
		}
		
		data := s.Text()
		if data != "" {
			parsedBundles, err := parseBundles([]byte(data), category)
			if err != nil {
				log.WithFields(log.Fields{
					"category": category,
					"step":     "parsing_json",
					"error":    err.Error(),
				}).Debug("Failed to parse JSON data")
				return
			}
			bundles = parsedBundles
			found = true
		}
	})

	// If no JSON found, try to extract bundle information from HTML
	if !found {
		bundles = extractBundlesFromHTML(doc, category)
	}

	if len(bundles) == 0 {
		log.WithField("category", category).Warn("No bundles found")
		// Create empty feed to maintain RSS structure
		bundles = []FanaticalBundle{}
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
	
	// This is a placeholder implementation
	// You'll need to inspect Fanatical's HTML structure and update this accordingly
	log.WithField("category", category).Info("Extracting bundles from HTML - this needs implementation")
	
	// Example: Look for bundle cards or similar elements
	doc.Find(".bundle-card, .product-card, [data-bundle]").Each(func(idx int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find("h3, h4, .title, .name").First().Text())
		description := strings.TrimSpace(s.Find(".description, .summary").First().Text())
		url, _ := s.Find("a").First().Attr("href")
		
		if title != "" {
			bundle := FanaticalBundle{
				ID:          fmt.Sprintf("%s-%d", category, idx),
				Title:       title,
				Description: description,
				URL:         url,
				Category:    category,
				StartDate:   time.Now(),
				EndDate:     time.Now().Add(24 * time.Hour * 30), // Default 30 days
				IsActive:    true,
			}
			bundles = append(bundles, bundle)
		}
	})
	
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
