package gofanatical

import "time"

// FanaticalBundle represents a bundle from Fanatical
type FanaticalBundle struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Image       string    `json:"image"`
	URL         string    `json:"url"`
	Price       Price     `json:"price"`
	StartDate   time.Time `json:"startDate"`
	EndDate     time.Time `json:"endDate"`
	Type        string    `json:"type"`
	Category    string    `json:"category"`
	Items       []Item    `json:"items"`
	IsActive    bool      `json:"isActive"`
}

// Price represents pricing information
type Price struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
	Original float64 `json:"original"`
	Discount int     `json:"discount"`
}

// Item represents an item in a bundle
type Item struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Type        string `json:"type"`
	Value       Price  `json:"value"`
}

// FanaticalAPIResponse represents the API response structure
// Note: This is a placeholder structure that needs to be updated 
// based on actual Fanatical API response format
type FanaticalAPIResponse struct {
	Success bool              `json:"success"`
	Data    []FanaticalBundle `json:"data"`
	Meta    struct {
		Total       int `json:"total"`
		CurrentPage int `json:"currentPage"`
		PerPage     int `json:"perPage"`
	} `json:"meta"`
}
