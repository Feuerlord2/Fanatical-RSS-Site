package gofanatical

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func validBundle(name string) AlgoliaBundle {
	return AlgoliaBundle{
		Name:       name,
		Slug:       "test-slug",
		Type:       "bundle",
		OnSale:     true,
		Price:      map[string]float64{"USD": 4.99},
		FullPrice:  map[string]float64{"USD": 49.99},
		ValidFrom:  1000,
		ValidUntil: 2000,
	}
}

func TestConvertAlgoliaBundlesFiltering(t *testing.T) {
	now := time.Unix(1500, 0)

	expired := validBundle("Expired Bundle")
	expired.ValidUntil = 1000

	notOnSale := validBundle("Hidden Bundle")
	notOnSale.OnSale = false

	unnamed := validBundle("")

	got := convertAlgoliaBundles([]AlgoliaBundle{validBundle("Good Bundle"), expired, notOnSale, unnamed}, now)

	if len(got) != 1 {
		t.Fatalf("expected 1 bundle, got %d", len(got))
	}
	if got[0].Title != "Good Bundle" {
		t.Errorf("unexpected bundle survived: %q", got[0].Title)
	}
}

func TestConvertAlgoliaBundlesDiscountRounded(t *testing.T) {
	now := time.Unix(1500, 0)

	b := validBundle("Deal")
	b.DiscountPercent = 0
	b.Price = map[string]float64{"USD": 25.10}
	b.FullPrice = map[string]float64{"USD": 100.00}

	got := convertAlgoliaBundles([]AlgoliaBundle{b}, now)
	if len(got) != 1 {
		t.Fatalf("expected 1 bundle, got %d", len(got))
	}
	// 74.9% must round to 75, not truncate to 74.
	if got[0].Price.Discount != 75 {
		t.Errorf("discount = %d, want 75", got[0].Price.Discount)
	}
}

func TestConvertAlgoliaBundlesCurrencyConsistency(t *testing.T) {
	now := time.Unix(1500, 0)

	b := validBundle("Euro Deal")
	b.Price = map[string]float64{"EUR": 10.00}
	b.FullPrice = map[string]float64{"EUR": 40.00, "USD": 999.99}

	got := convertAlgoliaBundles([]AlgoliaBundle{b}, now)
	if len(got) != 1 {
		t.Fatalf("expected 1 bundle, got %d", len(got))
	}
	p := got[0].Price
	if p.Currency != "EUR" || p.Amount != 10.00 || p.Original != 40.00 {
		t.Errorf("price = %+v, want EUR 10.00 / 40.00", p)
	}
	if p.Discount != 75 {
		t.Errorf("discount = %d, want 75", p.Discount)
	}
}

func TestPickPrice(t *testing.T) {
	tests := []struct {
		name         string
		prices       map[string]float64
		wantAmount   float64
		wantCurrency string
	}{
		{"prefers USD", map[string]float64{"USD": 5, "EUR": 4}, 5, "USD"},
		{"falls back to EUR", map[string]float64{"EUR": 4, "GBP": 3}, 4, "EUR"},
		{"skips zero USD", map[string]float64{"USD": 0, "GBP": 3}, 3, "GBP"},
		{"empty map", map[string]float64{}, 0, "USD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, currency := pickPrice(tt.prices)
			if amount != tt.wantAmount || currency != tt.wantCurrency {
				t.Errorf("pickPrice() = %v %s, want %v %s", amount, currency, tt.wantAmount, tt.wantCurrency)
			}
		})
	}
}

func TestBundleURL(t *testing.T) {
	tests := []struct {
		bundleType string
		want       string
	}{
		{"pick-and-mix", "/en/pick-and-mix/test-slug"},
		{"game", "/en/game/test-slug"},
		{"bundle", "/en/bundle/test-slug"},
		{"unknown", "/en/bundle/test-slug"},
	}

	for _, tt := range tests {
		b := AlgoliaBundle{Slug: "test-slug", Type: tt.bundleType}
		if got := bundleURL(b); got != tt.want {
			t.Errorf("bundleURL(type=%s) = %q, want %q", tt.bundleType, got, tt.want)
		}
	}
}

func TestCoverImageURL(t *testing.T) {
	if got := coverImageURL("abc.jpeg"); got != "https://fanatical.imgix.net/product/original/abc.jpeg" {
		t.Errorf("bare filename not prefixed: %q", got)
	}
	if got := coverImageURL("https://example.com/x.png"); got != "https://example.com/x.png" {
		t.Errorf("absolute URL must pass through: %q", got)
	}
	if got := coverImageURL(""); got != "" {
		t.Errorf("empty cover must stay empty: %q", got)
	}
}

func TestFetchBundlesOnceAgainstStubServer(t *testing.T) {
	future := time.Now().Add(72 * time.Hour).Unix()
	body := fmt.Sprintf(`[
		{
			"name": "Killer Bundle 42",
			"slug": "killer-bundle-42",
			"type": "bundle",
			"display_type": "bundle",
			"cover": "cover.jpeg",
			"on_sale": true,
			"discount_percent": 90,
			"price": {"USD": 4.99},
			"fullPrice": {"USD": 49.99},
			"available_valid_from": 1000,
			"available_valid_until": %d,
			"game_total": 10
		},
		{
			"name": "Coding Course Collection",
			"slug": "coding-course",
			"type": "bundle",
			"display_type": "elearning-bundle",
			"on_sale": true,
			"price": {"USD": 9.99},
			"fullPrice": {"USD": 99.99},
			"available_valid_from": 1000,
			"available_valid_until": %d
		}
	]`, future, future)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer server.Close()

	oldURL := bundlesURL
	bundlesURL = server.URL
	defer func() { bundlesURL = oldURL }()

	bundles, err := fetchBundlesOnce()
	if err != nil {
		t.Fatalf("fetchBundlesOnce failed: %v", err)
	}
	if len(bundles) != 2 {
		t.Fatalf("expected 2 bundles, got %d", len(bundles))
	}
	if bundles[0].Category != "games" || bundles[1].Category != "software" {
		t.Errorf("unexpected categories: %s, %s", bundles[0].Category, bundles[1].Category)
	}
	if bundles[0].Image != "https://fanatical.imgix.net/product/original/cover.jpeg" {
		t.Errorf("cover URL not built: %q", bundles[0].Image)
	}
}

func TestFetchBundlesOnceServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	oldURL := bundlesURL
	bundlesURL = server.URL
	defer func() { bundlesURL = oldURL }()

	if _, err := fetchBundlesOnce(); err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}
}
