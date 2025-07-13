package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Feuerlord2/Fanatical-RSS-Site/internal/rss"
	"github.com/Feuerlord2/Fanatical-RSS-Site/internal/scraper"
)

func main() {
	// Configure logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	// Determine which type to scrape based on command line argument
	bundleType := "games" // default
	if len(os.Args) > 1 {
		bundleType = strings.ToLower(os.Args[1])
	}
	
	// Validate bundle type
	validTypes := map[string]bool{
		"games":    true,
		"books":    true,
		"software": true,
	}
	
	if !validTypes[bundleType] {
		log.Fatalf("Invalid bundle type: %s. Valid types: games, books, software", bundleType)
	}
	
	log.Printf("Generating RSS feed for: %s", bundleType)
	
	// Initialize scraper for specific type
	s := scraper.NewScraper(bundleType)
	
	// Fetch bundles from Fanatical
	bundles, err := s.FetchBundles()
	if err != nil {
		log.Printf("Error fetching %s bundles: %v", bundleType, err)
		// Exit with error - no fallback to mock data
		os.Exit(1)
	}
	
	if len(bundles) == 0 {
		log.Printf("No %s bundles found on Fanatical", bundleType)
		// Create empty RSS feed instead of mock data
		bundles = []models.Bundle{}
	}
	
	log.Printf("Found %s bundles: %d", bundleType, len(bundles))
	
	// Initialize RSS generator for specific type
	generator := rss.NewGenerator(bundleType)
	
	// Generate RSS feed
	rssContent, err := generator.GenerateRSS(bundles)
	if err != nil {
		log.Fatalf("Error generating RSS feed: %v", err)
	}
	
	// Output RSS to stdout
	fmt.Print(rssContent)
	
	// Optional: Write to file for local testing
	if len(os.Args) > 2 && os.Args[2] == "--file" {
		filename := fmt.Sprintf("rss_%s.xml", bundleType)
		err = os.WriteFile(filename, []byte(rssContent), 0644)
		if err != nil {
			log.Printf("Error writing RSS file: %v", err)
		} else {
			log.Printf("RSS file successfully created: %s", filename)
		}
	}
}
