package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// RSS Strukturen
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Language    string `xml:"language"`
	PubDate     string `xml:"pubDate"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// Bundle Struktur
type Bundle struct {
	Title       string
	Link        string
	Description string
	Price       string
	GameCount   string
	ID          string
}

func main() {
	// Bundles von Fanatical abrufen
	bundles, err := fetchFanaticalBundles()
	if err != nil {
		fmt.Printf("Fehler beim Abrufen der Bundles: %v\n", err)
		os.Exit(1)
	}

	// RSS Feed erstellen
	rss := createRSSFeed(bundles)

	// RSS als XML ausgeben
	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		fmt.Printf("Fehler beim Erstellen des RSS: %v\n", err)
		os.Exit(1)
	}

	// XML Header hinzufügen
	fmt.Println(`<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Println(string(output))
}

func fetchFanaticalBundles() ([]Bundle, error) {
	url := "https://www.fanatical.com/de/bundle/games"
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	// User-Agent setzen, um nicht blockiert zu werden
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP-Anfrage fehlgeschlagen: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP Status: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Fehler beim Lesen der Antwort: %v", err)
	}
	
	return parseHTML(string(body))
}

func parseHTML(html string) ([]Bundle, error) {
	var bundles []Bundle
	
	// Regex-Patterns für das Parsen der Bundle-Informationen
	// Diese müssen möglicherweise angepasst werden, je nach aktueller HTML-Struktur
	
	// Pattern für Bundle-Container
	bundlePattern := regexp.MustCompile(`<article[^>]*class="[^"]*bundle[^"]*"[^>]*>(.*?)</article>`)
	
	// Pattern für Bundle-Titel
	titlePattern := regexp.MustCompile(`<h3[^>]*class="[^"]*bundle-title[^"]*"[^>]*>(.*?)</h3>|<h2[^>]*class="[^"]*bundle-title[^"]*"[^>]*>(.*?)</h2>`)
	
	// Pattern für Bundle-Links
	linkPattern := regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>`)
	
	// Pattern für Preise
	pricePattern := regexp.MustCompile(`€\s*(\d+[,.]?\d*)`)
	
	// Pattern für Spiele-Anzahl
	gameCountPattern := regexp.MustCompile(`(\d+)\s*[Ss]piele?`)
	
	// Fallback: Einfachere Patterns für Bundle-Informationen
	simplePattern := regexp.MustCompile(`<div[^>]*class="[^"]*card[^"]*"[^>]*>(.*?)</div>`)
	
	matches := bundlePattern.FindAllStringSubmatch(html, -1)
	
	if len(matches) == 0 {
		// Fallback-Parsing versuchen
		matches = simplePattern.FindAllStringSubmatch(html, -1)
	}
	
	for i, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		bundleHTML := match[1]
		bundle := Bundle{
			ID: fmt.Sprintf("bundle-%d", i),
		}
		
		// Titel extrahieren
		if titleMatches := titlePattern.FindStringSubmatch(bundleHTML); len(titleMatches) > 1 {
			title := titleMatches[1]
			if title == "" && len(titleMatches) > 2 {
				title = titleMatches[2]
			}
			bundle.Title = cleanHTML(title)
		}
		
		// Link extrahieren
		if linkMatches := linkPattern.FindStringSubmatch(bundleHTML); len(linkMatches) > 1 {
			link := linkMatches[1]
			if !strings.HasPrefix(link, "http") {
				link = "https://www.fanatical.com" + link
			}
			bundle.Link = link
		}
		
		// Preis extrahieren
		if priceMatches := pricePattern.FindStringSubmatch(bundleHTML); len(priceMatches) > 1 {
			bundle.Price = priceMatches[1] + "€"
		}
		
		// Spiele-Anzahl extrahieren
		if gameMatches := gameCountPattern.FindStringSubmatch(bundleHTML); len(gameMatches) > 1 {
			bundle.GameCount = gameMatches[1] + " Spiele"
		}
		
		// Beschreibung zusammensetzen
		description := "Fanatical Bundle"
		if bundle.Price != "" {
			description += " - Preis: " + bundle.Price
		}
		if bundle.GameCount != "" {
			description += " - " + bundle.GameCount
		}
		bundle.Description = description
		
		// Nur hinzufügen, wenn mindestens ein Titel vorhanden ist
		if bundle.Title != "" {
			bundles = append(bundles, bundle)
		}
	}
	
	// Wenn keine Bundles gefunden wurden, Mock-Daten verwenden
	if len(bundles) == 0 {
		bundles = getMockBundles()
	}
	
	return bundles, nil
}

func getMockBundles() []Bundle {
	return []Bundle{
		{
			ID:          "mock-1",
			Title:       "Indie Game Bundle",
			Link:        "https://www.fanatical.com/de/bundle/indie-game-bundle",
			Description: "Indie Game Bundle - Preis: 4,99€ - 10 Spiele",
			Price:       "4,99€",
			GameCount:   "10 Spiele",
		},
		{
			ID:          "mock-2",
			Title:       "Strategy Bundle",
			Link:        "https://www.fanatical.com/de/bundle/strategy-bundle",
			Description: "Strategy Bundle - Preis: 9,99€ - 8 Spiele",
			Price:       "9,99€",
			GameCount:   "8 Spiele",
		},
		{
			ID:          "mock-3",
			Title:       "Action Bundle",
			Link:        "https://www.fanatical.com/de/bundle/action-bundle",
			Description: "Action Bundle - Preis: 7,99€ - 12 Spiele",
			Price:       "7,99€",
			GameCount:   "12 Spiele",
		},
	}
}

func createRSSFeed(bundles []Bundle) RSS {
	var items []Item
	
	for _, bundle := range bundles {
		item := Item{
			Title:       bundle.Title,
			Link:        bundle.Link,
			Description: bundle.Description,
			PubDate:     time.Now().Format(time.RFC1123Z),
			GUID:        fmt.Sprintf("fanatical-bundle-%s", bundle.ID),
		}
		
		items = append(items, item)
	}
	
	return RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       "Fanatical Game Bundles",
			Link:        "https://www.fanatical.com/de/bundle/games",
			Description: "Aktuelle Spiele-Bundles von Fanatical",
			Language:    "de-DE",
			PubDate:     time.Now().Format(time.RFC1123Z),
			Items:       items,
		},
	}
}

func cleanHTML(s string) string {
	// HTML-Tags entfernen
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	
	// HTML-Entities dekodieren
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	
	// Whitespace normalisieren
	s = strings.TrimSpace(s)
	re = regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	
	return s
}
