package rss

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/Feuerlord2/Fanatical-RSS-Site/internal/models"
)

// RSS structures
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	Description   string `xml:"description"`
	Language      string `xml:"language"`
	Copyright     string `xml:"copyright"`
	ManagingEditor string `xml:"managingEditor"`
	WebMaster     string `xml:"webMaster"`
	PubDate       string `xml:"pubDate"`
	LastBuildDate string `xml:"lastBuildDate"`
	Category      string `xml:"category"`
	Generator     string `xml:"generator"`
	TTL           int    `xml:"ttl"`
	Image         Image  `xml:"image"`
	Items         []Item `xml:"item"`
}

type Image struct {
	URL    string `xml:"url"`
	Title  string `xml:"title"`
	Link   string `xml:"link"`
	Width  int    `xml:"width"`
	Height int    `xml:"height"`
}

type Item struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
	Author      string     `xml:"author"`
	Category    string     `xml:"category"`
	Comments    string     `xml:"comments"`
	Enclosure   *Enclosure `xml:"enclosure,omitempty"`
	GUID        string     `xml:"guid"`
	PubDate     string     `xml:"pubDate"`
	Source      string     `xml:"source"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length int    `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// Generator for RSS feeds
type Generator struct {
	feedTitle       string
	feedLink        string
	feedDescription string
	feedLanguage    string
	feedCopyright   string
	feedCategory    string
	feedTTL         int
}

// NewGenerator creates a new RSS generator
func NewGenerator() *Generator {
	return &Generator{
		feedTitle:       "Fanatical Game Bundles",
		feedLink:        "https://www.fanatical.com/en/bundle/games",
		feedDescription: "Current game bundles from Fanatical - Automatically generated",
		feedLanguage:    "en-US",
		feedCopyright:   "© 2025 Fanatical Bundle RSS Generator",
		feedCategory:    "Gaming",
		feedTTL:         60, // 60 minutes
	}
}

// GenerateRSS creates an RSS feed from bundle data
func (g *Generator) GenerateRSS(bundles []models.Bundle) (string, error) {
	now := time.Now()
	
	// Create RSS items
	var items []Item
	for _, bundle := range bundles {
		if !bundle.IsValid() {
			continue
		}
		
		item := Item{
			Title:       g.cleanTitle(bundle.Title),
			Link:        bundle.Link,
			Description: g.createItemDescription(bundle),
			Author:      "rss@fanatical.com",
			Category:    g.getCategoryFromBundle(bundle),
			GUID:        bundle.GetGUID(),
			PubDate:     bundle.UpdatedAt.Format(time.RFC1123Z),
			Source:      g.feedLink,
		}
		
		// Add enclosure for image if available
		if bundle.ImageURL != "" {
			item.Enclosure = &Enclosure{
				URL:    bundle.ImageURL,
				Length: 0, // Unknown
				Type:   "image/jpeg",
			}
		}
		
		items = append(items, item)
	}
	
	// Build RSS structure
	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:         g.feedTitle,
			Link:          g.feedLink,
			Description:   g.feedDescription,
			Language:      g.feedLanguage,
			Copyright:     g.feedCopyright,
			ManagingEditor: "rss@fanatical.com",
			WebMaster:     "webmaster@fanatical.com",
			PubDate:       now.Format(time.RFC1123Z),
			LastBuildDate: now.Format(time.RFC1123Z),
			Category:      g.feedCategory,
			Generator:     "Fanatical Bundle RSS Generator v1.0",
			TTL:           g.feedTTL,
			Image: Image{
				URL:    "https://www.fanatical.com/favicon.ico",
				Title:  g.feedTitle,
				Link:   g.feedLink,
				Width:  32,
				Height: 32,
			},
			Items: items,
		},
	}
	
	// Generate XML
	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error generating XML: %w", err)
	}
	
	// Add XML header
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + string(output)
	
	return xmlContent, nil
}

// createItemDescription creates a detailed description for an RSS item
func (g *Generator) createItemDescription(bundle models.Bundle) string {
	var parts []string
	
	// Base description
	if bundle.Description != "" {
		parts = append(parts, bundle.Description)
	}
	
	// Create HTML-formatted description
	html := fmt.Sprintf(`<div style="font-family: Arial, sans-serif;">`)
	
	if bundle.ImageURL != "" {
		html += fmt.Sprintf(`<img src="%s" alt="%s" style="max-width: 300px; height: auto; margin-bottom: 10px;" />`, 
			bundle.ImageURL, bundle.Title)
	}
	
	html += fmt.Sprintf(`<h3>%s</h3>`, bundle.Title)
	
	if bundle.Price != "" {
		html += fmt.Sprintf(`<p><strong>Price:</strong> %s</p>`, bundle.Price)
	}
	
	if bundle.GameCount != "" {
		html += fmt.Sprintf(`<p><strong>Number of Games:</strong> %s</p>`, bundle.GameCount)
	}
	
	if bundle.Tier != "" {
		html += fmt.Sprintf(`<p><strong>Tier:</strong> %s</p>`, bundle.Tier)
	}
	
	html += fmt.Sprintf(`<p><a href="%s" target="_blank">View Bundle →</a></p>`, bundle.Link)
	html += `</div>`
	
	return html
}

// cleanTitle cleans the title from unwanted characters
func (g *Generator) cleanTitle(title string) string {
	// Remove HTML tags
	title = strings.ReplaceAll(title, "<", "&lt;")
	title = strings.ReplaceAll(title, ">", "&gt;")
	title = strings.ReplaceAll(title, "&", "&amp;")
	
	// Normalize whitespace
	title = strings.TrimSpace(title)
	
	return title
}

// getCategoryFromBundle determines the category based on bundle properties
func (g *Generator) getCategoryFromBundle(bundle models.Bundle) string {
	title := strings.ToLower(bundle.Title)
	
	// Determine category based on title
	categories := map[string]string{
		"indie":      "Indie Games",
		"strategy":   "Strategy",
		"action":     "Action",
		"rpg":        "RPG",
		"adventure":  "Adventure",
		"puzzle":     "Puzzle",
		"horror":     "Horror",
		"simulation": "Simulation",
		"racing":     "Racing",
		"sports":     "Sports",
		"shooter":    "Shooter",
		"platformer": "Platformer",
		"roguelike":  "Roguelike",
	}
	
	for keyword, category := range categories {
		if strings.Contains(title, keyword) {
			return category
		}
	}
	
	return "Gaming"
}

// SetFeedMetadata allows customizing feed metadata
func (g *Generator) SetFeedMetadata(title, link, description, language string) {
	if title != "" {
		g.feedTitle = title
	}
	if link != "" {
		g.feedLink = link
	}
	if description != "" {
		g.feedDescription =
