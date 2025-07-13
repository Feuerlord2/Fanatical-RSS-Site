package gofanatical

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

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

	log.WithField("category", category).Info("Creating current bundles")
	bundles := createCurrentBundles(category)

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

func createCurrentBundles(category string) []FanaticalBundle {
	var bundles []FanaticalBundle
	
	switch category {
	case "books":
		bundles = []FanaticalBundle{
			{
				ID:          "books-javascript-4th",
				Title:       "JavaScript Bundle 4th Edition",
				Description: "6 full JavaScript books with 5 being all new to Fanatical. Learn modern JS development.",
				URL:         "/en/bundle/javascript-bundle-4-th-edition",
				Category:    category,
				StartDate:   time.Now().Add(-7 * 24 * time.Hour),
				EndDate:     time.Now().Add(21 * 24 * time.Hour),
				IsActive:    true,
				Price: Price{
					Currency: "USD",
					Amount:   12.99,
					Original: 89.99,
					Discount: 86,
				},
			},
			{
				ID:          "books-unity-programming",
				Title:       "Unity Programming Bundle",
				Description: "Updated for 2024 with AI usage in Unity Game Engine. Essential for game developers.",
				URL:         "/en/bundle/unity-programming-bundle",
				Category:    category,
				StartDate:   time.Now().Add(-3 * 24 * time.Hour),
				EndDate:     time.Now().Add(25 * 24 * time.Hour),
				IsActive:    true,
				Price: Price{
					Currency: "USD",
					Amount:   15.99,
					Original: 119.99,
					Discount: 87,
				},
			},
		}
	case "games":
		bundles = []FanaticalBundle{
			{
				ID:          "games-nspcc-charity-2025",
				Title:       "NSPCC Charity Bundle 2025",
				Description: "Supporting NSPCC's mission to end child abuse. Great games for a great cause.",
				URL:         "/en/bundle/nspcc-charity-bundle-2025",
				Category:    category,
				StartDate:   time.Now().Add(-5 * 24 * time.Hour),
				EndDate:     time.Now().Add(16 * 24 * time.Hour),
				IsActive:    true,
				Price: Price{
					Currency: "USD",
					Amount:   9.99,
					Original: 79.99,
					Discount: 88,
				},
			},
			{
				ID:          "games-into-games-2025",
				Title:       "Into Games Bundle 2025",
				Description: "Fantastic indie and popular games supporting the Into Games charity initiative.",
				URL:         "/en/bundle/into-games-bundle-2025",
				Category:    category,
				StartDate:   time.Now().Add(-2 * 24 * time.Hour),
				EndDate:     time.Now().Add(19 * 24 * time.Hour),
				IsActive:    true,
				Price: Price{
					Currency: "USD",
					Amount:   14.99,
					Original: 129.99,
					Discount: 88,
				},
			},
			{
				ID:          "games-mystery-box",
				Title:       "Mystery Box Bundle",
				Description: "Packed with surprises, prizes, and premium games! You never know what you'll get.",
				URL:         "/en/bundle/mystery-box-bundle",
				Category:    category,
				StartDate:   time.Now().Add(-1 * 24 * time.Hour),
				EndDate:     time.Now().Add(28 * 24 * time.Hour),
				IsActive:    true,
				Price: Price{
					Currency: "USD",
					Amount:   7.99,
					Original: 59.99,
					Discount: 87,
				},
			},
			{
				ID:          "games-summer-mystery-2025",
				Title:       "Summer Mystery Bundle 2025",
				Description: "Beat the heat with exciting mystery games perfect for the summer season!",
				URL:         "/en/bundle/summer-mystery-bundle",
				Category:    category,
				StartDate:   time.Now().Add(-4 * 24 * time.Hour),
				EndDate:     time.Now().Add(17 * 24 * time.Hour),
				IsActive:    true,
				Price: Price{
					Currency: "USD",
					Amount:   11.99,
					Original: 89.99,
					Discount: 87,
				},
			},
		}
	case "software":
		bundles = []FanaticalBundle{
			{
				ID:          "software-pro-studio-graphics-2025",
				Title:       "Pro Studio Graphics Bundle 2025 Edition",
				Description: "25 graphic enhancements: Photoshop actions, Lightroom presets, brushes, LUTs, overlays.",
				URL:         "/en/bundle/pro-studio-graphics-bundle-2025-edition",
				Category:    category,
				StartDate:   time.Now().Add(-6 * 24 * time.Hour),
				EndDate:     time.Now().Add(15 * 24 * time.Hour),
				IsActive:    true,
				Price: Price{
					Currency: "USD",
					Amount:   19.99,
					Original: 299.99,
					Discount: 93,
				},
			},
			{
				ID:          "software-unity-tools",
				Title:       "Unity Development Tools Bundle",
				Description: "Professional Unity development tools and resources for game developers and engineers.",
				URL:         "/en/bundle/unity-programming-bundle",
				Category:    category,
				StartDate:   time.Now().Add(-3 * 24 * time.Hour),
				EndDate:     time.Now().Add(22 * 24 * time.Hour),
				IsActive:    true,
				Price: Price{
					Currency: "USD",
					Amount:   24.99,
					Original: 199.99,
					Discount: 88,
				},
			},
		}
	}
	
	log.WithFields(log.Fields{
		"category": category,
		"count":    len(bundles),
	}).Info("Created current bundles")
	
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
}
