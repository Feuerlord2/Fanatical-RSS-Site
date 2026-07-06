package gofanatical

import (
	"strings"
	"testing"
	"time"
)

func TestCreateRichContentEscapesHTML(t *testing.T) {
	bundle := FanaticalBundle{
		Title:       `Evil <script>alert("x")</script> & Friends`,
		Description: `Deals & <b>discounts</b>`,
		Image:       `https://example.com/cover.jpg?a=1&b=2`,
		URL:         "/en/bundle/evil",
		EndDate:     time.Now().Add(48 * time.Hour),
		Price:       Price{Currency: "USD", Amount: 4.99, Original: 9.99, Discount: 50},
	}

	content := createRichContent(bundle)

	if strings.Contains(content, "<script>") {
		t.Error("title was not HTML-escaped: raw <script> tag in output")
	}
	if !strings.Contains(content, "Evil &lt;script&gt;") {
		t.Error("expected escaped title in output")
	}
	if !strings.Contains(content, "Deals &amp; &lt;b&gt;discounts&lt;/b&gt;") {
		t.Error("expected escaped description in output")
	}
	if !strings.Contains(content, "cover.jpg?a=1&amp;b=2") {
		t.Error("expected escaped image URL in attribute")
	}
}

func TestCreateRichContentPrices(t *testing.T) {
	base := FanaticalBundle{
		Title:   "Bundle",
		URL:     "/en/bundle/x",
		EndDate: time.Now().Add(time.Hour),
	}

	free := base
	free.Price = Price{Currency: "USD", Amount: 0, Original: 0}
	content := createRichContent(free)
	if !strings.Contains(content, "FREE") || !strings.Contains(content, "N/A") {
		t.Error("free bundle should render FREE and N/A")
	}

	euro := base
	euro.Price = Price{Currency: "EUR", Amount: 3.49, Original: 34.99, Discount: 90}
	content = createRichContent(euro)
	if !strings.Contains(content, "€3.49") || !strings.Contains(content, "€34.99") {
		t.Errorf("expected euro prices in output, got: %s", content)
	}
}

func TestImageMIMEType(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://x.com/a.png", "image/png"},
		{"https://x.com/a.PNG?w=100", "image/png"},
		{"https://x.com/a.webp", "image/webp"},
		{"https://x.com/a.gif", "image/gif"},
		{"https://x.com/a.jpeg", "image/jpeg"},
		{"https://x.com/a", "image/jpeg"},
	}
	for _, tt := range tests {
		if got := imageMIMEType(tt.url); got != tt.want {
			t.Errorf("imageMIMEType(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
