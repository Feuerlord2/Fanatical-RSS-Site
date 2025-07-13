package models

import (
	"fmt"
	"time"
)

// Bundle represents a bundle from Fanatical (games, books, or software)
type Bundle struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	ItemCount   string    `json:"item_count"`   // Previously GameCount, now generic
	ImageURL    string    `json:"image_url"`
	Tier        string    `json:"tier"`
	BundleType  string    `json:"bundle_type"` // games, books, software
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewBundle creates a new bundle with current timestamps
func NewBundle(id, title, bundleType string) *Bundle {
	now := time.Now()
	return &Bundle{
		ID:         id,
		Title:      title,
		BundleType: bundleType,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// GetFullDescription returns a complete description of the bundle
func (b *Bundle) GetFullDescription() string {
	description := b.Description
	
	if b.Price != "" {
		description += fmt.Sprintf(" - Price: %s", b.Price)
	}
	
	if b.ItemCount != "" {
		description += fmt.Sprintf(" - %s", b.ItemCount)
	}
	
	if b.Tier != "" {
		description += fmt.Sprintf(" - Tier: %s", b.Tier)
	}
	
	return description
}

// GetGUID returns a unique GUID for the bundle
func (b *Bundle) GetGUID() string {
	return fmt.Sprintf("fanatical-%s-bundle-%s", b.BundleType, b.ID)
}

// IsValid checks if the bundle is valid
func (b *Bundle) IsValid() bool {
	return b.Title != "" && b.Link != "" && b.BundleType != ""
}

// SetDefaults sets default values for missing fields
func (b *Bundle) SetDefaults() {
	if b.Description == "" {
		bundleTypeName := "Bundle"
		switch b.BundleType {
		case "games":
			bundleTypeName = "Game Bundle"
		case "books":
			bundleTypeName = "Book Bundle"
		case "software":
			bundleTypeName = "Software Bundle"
		}
		b.Description = fmt.Sprintf("Fanatical %s", bundleTypeName)
	}
	
	if b.Link != "" && b.Link[0] == '/' {
		b.Link = "https://www.fanatical.com" + b.Link
	}
	
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = time.Now()
	}
}

// GetBundleTypeName returns a human-readable bundle type name
func (b *Bundle) GetBundleTypeName() string {
	switch b.BundleType {
	case "games":
		return "Game Bundles"
	case "books":
		return "Book Bundles"
	case "software":
		return "Software Bundles"
	default:
		return "Bundles"
	}
}

// GetItemTypeName returns the correct item type name for the bundle
func (b *Bundle) GetItemTypeName() string {
	switch b.BundleType {
	case "games":
		return "Games"
	case "books":
		return "Books"
	case "software":
		return "Software"
	default:
		return "Items"
	}
}

// BundleList represents a list of bundles with metadata
type BundleList struct {
	Bundles     []Bundle  `json:"bundles"`
	TotalCount  int       `json:"total_count"`
	LastUpdated time.Time `json:"last_updated"`
	Source      string    `json:"source"`
	BundleType  string    `json:"bundle_type"`
}

// NewBundleList creates a new BundleList for a specific bundle type
func NewBundleList(bundleType string) *BundleList {
	return &BundleList{
		Bundles:     make([]Bundle, 0),
		LastUpdated: time.Now(),
		Source:      fmt.Sprintf("https://www.fanatical.com/de/bundle/%s", bundleType),
		BundleType:  bundleType,
	}
}

// AddBundle adds a bundle to the list
func (bl *BundleList) AddBundle(bundle Bundle) {
	bundle.SetDefaults()
	if bundle.IsValid() {
		bl.Bundles = append(bl.Bundles, bundle)
		bl.TotalCount = len(bl.Bundles)
		bl.LastUpdated = time.Now()
	}
}

// GetValidBundles returns only valid bundles
func (bl *BundleList) GetValidBundles() []Bundle {
	var validBundles []Bundle
	for _, bundle := range bl.Bundles {
		if bundle.IsValid() {
			validBundles = append(validBundles, bundle)
		}
	}
	return validBundles
}

// GetBundleTypeName returns a human-readable name for the bundle type
func (bl *BundleList) GetBundleTypeName() string {
	switch bl.BundleType {
	case "games":
		return "Game Bundles"
	case "books":
		return "Book Bundles"
	case "software":
		return "Software Bundles"
	default:
		return "Bundles"
	}
}
