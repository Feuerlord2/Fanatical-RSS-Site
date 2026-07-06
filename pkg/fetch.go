package gofanatical

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"
)

// bundlesURL is a variable so tests can point it at a stub server.
var bundlesURL = "https://www.fanatical.com/api/algolia/bundles?altRank=false"

const fetchAttempts = 3

// AlgoliaBundle mirrors the fields we use from Fanatical's Algolia API.
type AlgoliaBundle struct {
	Name             string             `json:"name"`
	Slug             string             `json:"slug"`
	Type             string             `json:"type"`
	DisplayType      string             `json:"display_type"`
	Cover            string             `json:"cover"`
	DiscountPercent  int                `json:"discount_percent"`
	OnSale           bool               `json:"on_sale"`
	BestEver         bool               `json:"best_ever"`
	FlashSale        bool               `json:"flash_sale"`
	StarDeal         bool               `json:"star_deal"`
	Giveaway         bool               `json:"giveaway"`
	Price            map[string]float64 `json:"price"`
	FullPrice        map[string]float64 `json:"fullPrice"`
	OperatingSystems []string           `json:"operating_systems"`
	DRM              []string           `json:"drm"`
	ValidFrom        int64              `json:"available_valid_from"`
	ValidUntil       int64              `json:"available_valid_until"`
	GameTotal        int                `json:"game_total"`
	DLCTotal         int                `json:"dlc_total"`
}

// fetchBundles downloads the current bundle list, retrying transient failures.
func fetchBundles() ([]FanaticalBundle, error) {
	var lastErr error
	for attempt := 1; attempt <= fetchAttempts; attempt++ {
		bundles, err := fetchBundlesOnce()
		if err == nil {
			return bundles, nil
		}
		lastErr = err
		slog.Warn("fetch attempt failed", "attempt", attempt, "error", err)
		if attempt < fetchAttempts {
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}
	return nil, lastErr
}

func fetchBundlesOnce() ([]FanaticalBundle, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest(http.MethodGet, bundlesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.fanatical.com/en/bundles")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Algolia API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Algolia API returned status %d", resp.StatusCode)
	}

	var algoliaBundles []AlgoliaBundle
	if err := json.NewDecoder(resp.Body).Decode(&algoliaBundles); err != nil {
		return nil, fmt.Errorf("failed to decode Algolia API response: %w", err)
	}

	slog.Info("fetched bundles from Algolia API", "bundles", len(algoliaBundles))

	return convertAlgoliaBundles(algoliaBundles, time.Now()), nil
}

// convertAlgoliaBundles turns API bundles into internal ones, dropping
// bundles that are unnamed, expired, or not on sale.
func convertAlgoliaBundles(algoliaBundles []AlgoliaBundle, now time.Time) []FanaticalBundle {
	var bundles []FanaticalBundle
	skipped := 0

	for _, ab := range algoliaBundles {
		if ab.Name == "" || !ab.OnSale || now.Unix() > ab.ValidUntil {
			skipped++
			continue
		}

		price, currency := pickPrice(ab.Price)
		// Read the original price in the same currency so the discount and
		// savings math never mixes currencies.
		originalPrice := ab.FullPrice[currency]

		discount := ab.DiscountPercent
		if discount == 0 && originalPrice > price && originalPrice > 0 {
			discount = int(math.Round((originalPrice - price) / originalPrice * 100))
		}

		bundles = append(bundles, FanaticalBundle{
			Title:       ab.Name,
			Slug:        ab.Slug,
			Description: buildDescription(ab),
			Image:       coverImageURL(ab.Cover),
			URL:         bundleURL(ab),
			Category:    categorizeBundle(ab),
			StartDate:   time.Unix(ab.ValidFrom, 0),
			EndDate:     time.Unix(ab.ValidUntil, 0),
			Price: Price{
				Currency: currency,
				Amount:   price,
				Original: originalPrice,
				Discount: discount,
			},
		})
	}

	slog.Info("bundle conversion completed", "total", len(algoliaBundles), "active", len(bundles), "skipped", skipped)

	return bundles
}

// pickPrice returns the USD price when available, otherwise the first
// positive price from a fixed list of fallback currencies.
func pickPrice(priceMap map[string]float64) (float64, string) {
	if price, ok := priceMap["USD"]; ok && price > 0 {
		return price, "USD"
	}
	for _, currency := range []string{"EUR", "GBP", "CAD", "AUD"} {
		if price, ok := priceMap[currency]; ok && price > 0 {
			return price, currency
		}
	}
	return 0, "USD"
}

func bundleURL(ab AlgoliaBundle) string {
	switch ab.Type {
	case "pick-and-mix":
		return fmt.Sprintf("/en/pick-and-mix/%s", ab.Slug)
	case "game":
		return fmt.Sprintf("/en/game/%s", ab.Slug)
	default:
		return fmt.Sprintf("/en/bundle/%s", ab.Slug)
	}
}

// coverImageURL turns a bare cover filename into a full imgix URL.
func coverImageURL(cover string) string {
	if cover == "" || strings.HasPrefix(cover, "http") {
		return cover
	}
	return "https://fanatical.imgix.net/product/original/" + cover
}

func buildDescription(ab AlgoliaBundle) string {
	var parts []string

	switch ab.Type {
	case "pick-and-mix":
		parts = append(parts, "🎯 Build your own bundle")
	case "bundle":
		parts = append(parts, "📦 Bundle")
	}

	if ab.GameTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d items", ab.GameTotal))
	}
	if ab.DLCTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d DLC", ab.DLCTotal))
	}

	if ab.BestEver {
		parts = append(parts, "🏆 Best price ever!")
	}
	if ab.FlashSale {
		parts = append(parts, "⚡ Flash sale!")
	}
	if ab.StarDeal {
		parts = append(parts, "⭐ Star Deal")
	}
	if ab.Giveaway {
		parts = append(parts, "🎁 FREE")
	}

	if len(ab.DRM) > 0 {
		parts = append(parts, "DRM: "+strings.Join(ab.DRM, ", "))
	}
	if len(ab.OperatingSystems) > 0 {
		parts = append(parts, "OS: "+strings.Join(ab.OperatingSystems, ", "))
	}

	if len(parts) == 0 {
		return "Great content with amazing savings"
	}

	return strings.Join(parts, " • ")
}
