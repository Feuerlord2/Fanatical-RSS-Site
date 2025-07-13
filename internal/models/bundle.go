package models

import (
	"fmt"
	"time"
)

// Bundle represents a game bundle from Fanatical
type Bundle struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	GameCount   string    `json:"game_count"`
	ImageURL    string    `json:"image_url"`
	Tier        string    `json:"tier"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewBundle creates a new bundle with current timestamps
func NewBundle(id, title string) *Bundle {
	now := time.Now()
	return &Bundle{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GetFullDescription returns a complete description of the bundle
func (b *Bundle) GetFullDescription() string {
	description := b.Description
	
	if b.Price != "" {
		description += fmt.Sprintf(" - Price: %s", b.Price)
	}
	
	if b.GameCount != "" {
		description += fmt.Sprintf(" - %s", b.GameCount)
	}
	
	if b.Tier != "" {
		description += fmt.Sprintf(" - Tier: %s", b.Tier)
	}
	
	return description
}

// GetGUID returns a unique GUID for the bundle
func (b *Bundle) GetGUID() string {
	return fmt.Sprintf("fanatical-bundle-%s", b.ID)
}

// IsValid checks if the bundle is valid
func (b *Bundle) IsValid() bool {
	return b.Title != "" && b.Link != ""
}

// SetDefaults sets default values for missing fields
func (b *Bundle) SetDefaults() {
	if b.Description == "" {
		b.Description = "Fanatical Bundle"
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

// BundleList represents a list of bundles with metadata
type BundleList struct {
	Bundles     []Bundle  `json:"bundles"`
	TotalCount  int       `json:"total_count"`
	LastUpdated time.Time `json:"last_updated"`
	Source      string    `json:"source"`
}

// NewBundleList creates a new BundleList
func NewBundleList() *BundleList {
	return &BundleList{
		Bundles:     make([]Bundle, 0),
		LastUpdated: time.Now(),
		Source:      "https://www.fanatical.com/de/bundle/games",
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
