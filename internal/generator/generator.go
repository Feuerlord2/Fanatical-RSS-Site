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
	bundleType      string
}

// NewGenerator creates a new RSS generator for a specific bundle type
func NewGenerator(bundleType string) *Generator {
	return &Generator{
		feedTitle:       getFeedTitle(bundleType),
		feedLink:        fmt.Sprintf("https://www.fanatical.com/en/bundle/%s", bundleType),
		feedDescription: getFeedDescription(bundleType),
		feedLanguage:    "en-US",
		feedCopyright:   "© 2025 Fanatical Bundle RSS Generator",
		feedCategory:    getFeedCategory(bundleType),
		feedTTL:         60, // 60 minutes
		bundleType:      bundleType,
	}
}

// getFeedTitle returns the appropriate feed title for the bundle type
func getFeedTitle(bundleType string) string {
	switch bundleType {
	case "games":
		return "Fanatical Game Bundles"
	case "books":
		return "Fanatical Book Bundles"
	case "software":
		return "Fanatical Software Bundles"
	default:
		return "Fanatical Bundles"
	}
}

// getFeedDescription returns the appropriate feed description for the bundle type
func getFeedDescription(bundleType string) string {
	switch bundleType {
	case "games":
		return "Current game bundles from Fanatical - Automatically generated"
	case "books":
		return "Current book bundles from Fanatical - Automatically generated"
	case "software":
		return "Current software bundles from Fanatical - Automatically generated"
	default:
		return "Current bundles from Fanatical - Automatically generated"
	}
}

// getFeedCategory returns the appropriate feed category for the bundle type
func getFeedCategory(bundleType string) string {
	switch bundleType {
	case "games":
		return "Gaming"
	case "books":
		return "Books & Literature"
	case "software":
		return "Software & Technology"
	default:
		return "Technology"
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
	
	if bundle.ItemCount != "" {
		itemType := bundle.GetItemTypeName()
		html += fmt.Sprintf(`<p><strong>Number of %s:</strong> %s</p>`, itemType, bundle.ItemCount)
	}
	
	if bundle.Tier != "" {
		html += fmt.Sprintf(`<p><strong>Tier:</strong> %s</p>`, bundle.Tier)
	}
	
	html += fmt.Sprintf(`<p><strong>Type:</strong> %s</p>`, bundle.GetBundleTypeName())
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
	
	// Base category on bundle type
	switch bundle.BundleType {
	case "games":
		return g.getGameCategory(title)
	case "books":
		return g.getBookCategory(title)
	case "software":
		return g.getSoftwareCategory(title)
	default:
		return g.feedCategory
	}
}

// getGameCategory determines game-specific categories
func (g *Generator) getGameCategory(title string) string {
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

// getBookCategory determines book-specific categories
func (g *Generator) getBookCategory(title string) string {
	categories := map[string]string{
		"fiction":     "Fiction",
		"sci-fi":      "Science Fiction",
		"fantasy":     "Fantasy",
		"mystery":     "Mystery",
		"romance":     "Romance",
		"thriller":    "Thriller",
		"biography":   "Biography",
		"history":     "History",
		"programming": "Programming",
		"tech":        "Technology",
		"business":    "Business",
		"self-help":   "Self Help",
		"cooking":     "Cooking",
		"art":         "Art",
	}
	
	for keyword, category := range categories {
		if strings.Contains(title, keyword) {
			return category
		}
	}
	
	return "Books & Literature"
}

// getSoftwareCategory determines software-specific categories
func (g *Generator) getSoftwareCategory(title string) string {
	categories := map[string]string{
		"creative":     "Creative Software",
		"design":       "Design",
		"photo":        "Photo Editing",
		"video":        "Video Editing",
		"audio":        "Audio Production",
		"productivity": "Productivity",
		"office":       "Office Software",
		"security":     "Security",
		"utility":      "Utilities",
		"development":  "Development Tools",
		"programming":  "Programming",
		"game":         "Game Development",
		"3d":           "3D Software",
		"animation":    "Animation",
	}
	
	for keyword, category := range categories {
		if strings.Contains(title, keyword) {
			return category
		}
	}
	
	return "Software & Technology"
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
		g.feedDescription = description
	}
	if language != "" {
		g.feedLanguage = language
	}
}

// SetTTL sets the Time-To-Live for the feed
func (g *Generator) SetTTL(minutes int) {
	g.feedTTL = minutes
}

// ValidateRSS checks if the generated RSS feed is valid
func (g *Generator) ValidateRSS(rssContent string) error {
	// Basic validation
	if !strings.Contains(rssContent, "<?xml") {
		return fmt.Errorf("XML header missing")
	}
	
	if !strings.Contains(rssContent, "<rss") {
		return fmt.Errorf("RSS root element missing")
	}
	
	if !strings.Contains(rssContent, "<channel>") {
		return fmt.Errorf("channel element missing")
	}
	
	// Test XML parsing
	var rss RSS
	err := xml.Unmarshal([]byte(rssContent), &rss)
	if err != nil {
		return fmt.Errorf("XML parsing failed: %w", err)
	}
	
	// Check minimum requirements
	if rss.Channel.Title == "" {
		return fmt.Errorf("channel title missing")
	}
	
	if rss.Channel.Link == "" {
		return fmt.Errorf("channel link missing")
	}
	
	return nil
}
