package gofanatical

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRunEndToEnd drives the full pipeline against a stub API server:
// fetch → categorize → dedup → write docs/*.rss. It also runs the pipeline
// twice to guarantee unchanged input produces byte-identical files — the
// property the CI workflow relies on to only commit real changes.
func TestRunEndToEnd(t *testing.T) {
	future := time.Now().Add(72 * time.Hour).Unix()
	body := fmt.Sprintf(`[
		{"name": "Killer Bundle 42", "slug": "killer-42", "type": "bundle", "display_type": "bundle",
		 "on_sale": true, "price": {"USD": 4.99}, "fullPrice": {"USD": 49.99},
		 "available_valid_from": 1000, "available_valid_until": %d},
		{"name": "Fantasy Book Library", "slug": "fantasy-books", "type": "bundle", "display_type": "book-bundle",
		 "on_sale": true, "price": {"USD": 9.99}, "fullPrice": {"USD": 99.99},
		 "available_valid_from": 2000, "available_valid_until": %d},
		{"name": "Excel Toolkit", "slug": "excel-kit", "type": "bundle", "display_type": "software-bundle",
		 "on_sale": true, "price": {"USD": 14.99}, "fullPrice": {"USD": 29.99},
		 "available_valid_from": 3000, "available_valid_until": %d},
		{"name": "Killer Bundle 42", "slug": "killer-42", "type": "bundle", "display_type": "bundle",
		 "on_sale": true, "price": {"USD": 4.99}, "fullPrice": {"USD": 49.99},
		 "available_valid_from": 1000, "available_valid_until": %d}
	]`, future, future, future, future)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer server.Close()

	oldURL := bundlesURL
	bundlesURL = server.URL
	defer func() { bundlesURL = oldURL }()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	if err := Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	firstRun := map[string]string{}
	for _, category := range []string{"books", "games", "software"} {
		path := filepath.Join("docs", category+".rss")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("missing feed file %s: %v", path, err)
		}
		// Every feed must be well-formed XML.
		var doc struct{}
		if err := xml.Unmarshal(data, &doc); err != nil {
			t.Errorf("%s is not well-formed XML: %v", path, err)
		}
		firstRun[category] = string(data)
	}

	// The duplicate "killer-42" entry must be deduplicated.
	if got := strings.Count(firstRun["games"], "fanatical-killer-42-1000"); got != 1 {
		t.Errorf("expected exactly 1 killer-42 GUID in games feed, got %d", got)
	}
	if !strings.Contains(firstRun["books"], "Fantasy Book Library") {
		t.Error("books feed missing the book bundle")
	}
	if !strings.Contains(firstRun["software"], "Excel Toolkit") {
		t.Error("software feed missing the software bundle")
	}
	if strings.Contains(firstRun["games"], "Fantasy Book Library") {
		t.Error("book bundle leaked into games feed")
	}

	// Second run with identical input must produce byte-identical files.
	if err := Run(); err != nil {
		t.Fatalf("second Run failed: %v", err)
	}
	for category, before := range firstRun {
		after, err := os.ReadFile(filepath.Join("docs", category+".rss"))
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != before {
			t.Errorf("%s.rss changed between runs with identical input", category)
		}
	}
}
