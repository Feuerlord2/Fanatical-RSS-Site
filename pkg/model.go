package gofanatical

import "time"

// FanaticalBundle is the internal representation of a Fanatical deal.
type FanaticalBundle struct {
	Title       string
	Slug        string
	Description string
	Image       string
	URL         string
	Category    string
	StartDate   time.Time
	EndDate     time.Time
	Price       Price
}

// Price holds pricing information for a bundle.
type Price struct {
	Currency string
	Amount   float64
	Original float64
	Discount int
}
