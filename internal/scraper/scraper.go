package scraper

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/Feuerlord2/Fanatical-RSS-Site/internal/models"
)

// Scraper for Fanatical bundle pages
type Scraper struct {
	client     *http.Client
	baseURL    string
	bundleType string
}

// NewScraper creates a new scraper instance for a specific bundle type
func NewScraper(bundleType string) *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    "https://www.fanatical.com",
		bundleType: bundleType,
	}
}

// FetchBundles retrieves all bundles from the Fanatical page
func (s *Scraper) FetchBundles() ([]models.Bundle, error) {
	url := fmt.Sprintf("%s/de/bundle/%s", s.baseURL, s.bundleType)
	
	log.Printf("Loading %s bundles from: %s", s.bundleType, url)
	
	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}
	
	// Set headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,de;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	
	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status: %d", resp.StatusCode)
	}
	
	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}
	
	return s.parseDocument(doc)
}

// parseDocument parses the HTML document and extracts bundle information
func (s *Scraper) parseDocument(doc *goquery.Document) ([]models.Bundle, error) {
	var bundles []models.Bundle
	
	// Try different selectors for bundle cards
	selectors := []string{
		"article.bundle-card",
		".bundle-card",
		"[data-product-type='bundle']",
		".card.bundle",
		".product-card",
		"article[class*='bundle']",
		".grid-item",
		".card",
		"[class*='product']",
	}
	
	var bundleElements *goquery.Selection
	
	for _, selector := range selectors {
		bundleElements = doc.Find(selector)
		if bundleElements.Length() > 0 {
			log.Printf("Found with selector: %s (%d elements)", selector, bundleElements.Length())
			break
		}
	}
	
	if bundleElements.Length() == 0 {
		log.Println("No bundle elements found, using fallback parsing")
		return s.fallbackParsing(doc)
	}
	
	// Extract bundle information
	bundleElements.Each(func(i int, sel *goquery.Selection) {
		bundle := s.extractBundle(i, sel)
		if bundle.IsValid() {
			bundles = append(bundles, bundle)
		}
	})
	
	log.Printf("Extracted %s bundles: %d", s.bundleType, len(bundles))
	
	return bundles, nil
}

// extractBundle extracts bundle information from an HTML element
func (s *Scraper) extractBundle(index int, sel *goquery.Selection) models.Bundle {
	bundle := models.Bundle{
		ID:         fmt.Sprintf("%s-bundle-%d-%d", s.bundleType, time.Now().Unix(), index),
		BundleType: s.bundleType,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	// Extract title
	titleSelectors := []string{
		"h3.bundle-title",
		"h2.bundle-title",
		".bundle-title",
		"h3",
		"h2",
		".title",
		"[class*='title']",
		".card-title",
		".product-title",
	}
	
	for _, selector := range titleSelectors {
		if title := sel.Find(selector).First().Text(); title != "" {
			bundle.Title = strings.TrimSpace(title)
			break
		}
	}
	
	// Extract link
	if href, exists := sel.Find("a").First().Attr("href"); exists {
		if !strings.HasPrefix(href, "http") {
			href = s.baseURL + href
		}
		bundle.Link = href
	}
	
	// Extract price
	priceSelectors := []string{
		".price",
		"[class*='price']",
		".bundle-price",
		"span[class*='price']",
		".cost",
		"[class*='cost']",
	}
	
	for _, selector := range priceSelectors {
		if price := sel.Find(selector).First().Text(); price != "" {
			bundle.Price = s.cleanPrice(price)
			break
		}
	}
	
	// Extract item count (games/books/software)
	itemCountSelectors := []string{
		".game-count",
		".item-count",
		"[class*='count']",
		".items",
		"[class*='items']",
		".quantity",
	}
	
	for _, selector := range itemCountSelectors {
		if count := sel.Find(selector).First().Text(); count != "" {
			if itemCount := s.extractItemCount(count); itemCount != "" {
				bundle.ItemCount = itemCount
				break
			}
		}
	}
	
	// Extract image URL
	if img := sel.Find("img").First(); img.Length() > 0 {
		if src, exists := img.Attr("src"); exists {
			if !strings.HasPrefix(src, "http") {
				src = s.baseURL + src
			}
			bundle.ImageURL = src
		}
	}
	
	// Extract tier (if available)
	if tier := sel.Find(".tier, [class*='tier']").First().Text(); tier != "" {
		bundle.Tier = strings.TrimSpace(tier)
	}
	
	// Generate description
	bundle.Description = s.generateDescription(bundle)
	
	return bundle
}

// fallbackParsing as fallback when normal selectors don't work
func (s *Scraper) fallbackParsing(doc *goquery.Document) ([]models.Bundle, error) {
	log.Printf("Using fallback parsing for %s", s.bundleType)
	
	var bundles []models.Bundle
	
	// Search for links pointing to bundle pages
	doc.Find(fmt.Sprintf("a[href*='/bundle/%s']", s.bundleType)).Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists {
			return
		}
		
		if !strings.HasPrefix(href, "http") {
			href = s.baseURL + href
		}
		
		title := sel.Text()
		if title == "" {
			title = sel.Find("img").AttrOr("alt", "")
		}
		
		if title != "" {
			bundle := models.Bundle{
				ID:          fmt.Sprintf("%s-fallback-%d", s.bundleType, i),
				Title:       strings.TrimSpace(title),
				Link:        href,
				BundleType:  s.bundleType,
				Description: fmt.Sprintf("Fanatical %s Bundle", strings.Title(s.bundleType)),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			
			if bundle.IsValid() {
				bundles = append(bun
