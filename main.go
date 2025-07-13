package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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
	LastBuildDate string `xml:"lastBuildDate"`
	Items         []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// Bundle represents a Fanatical bundle
type Bundle struct {
	Title       string
	Description string
	Link        string
	Price       string
	ImageURL    string
	PubDate     time.Time
}

// BundleType represents the type of bundle
type BundleType struct {
	Name string
	URL  string
	File string
}

var bundleTypes = []BundleType{
	{"Games", "https://www.fanatical.com/de/bundle/games", "games.rss"},
	{"Books", "https://www.fanatical.com/de/bundle/books", "books.rss"},
	{"Software", "https://www.fanatical.com/de/bundle/software", "software.rss"},
}

func main() {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll("docs", 0755); err != nil {
		log.Fatal(err)
	}

	// Generate RSS feeds for each bundle type
	for _, bundleType := range bundleTypes {
		log.Printf("Generating RSS feed for %s...", bundleType.Name)
		
		bundles, err := scrapeBundles(bundleType.URL)
		if err != nil {
			log.Printf("Error scraping %s bundles: %v", bundleType.Name, err)
			continue
		}

		rss := generateRSS(bundleType.Name, bundles)
		
		outputPath := filepath.Join("docs", bundleType.File)
		if err := writeRSSFile(outputPath, rss); err != nil {
			log.Printf("Error writing RSS file for %s: %v", bundleType.Name, err)
			continue
		}

		log.Printf("Successfully generated %s with %d items", bundleType.File, len(bundles))
	}

	// Copy index.html to docs directory
	if err := copyIndexFile(); err != nil {
		log.Printf("Error copying index.html: %v", err)
	}

	log.Println("RSS generation complete!")
}

func scrapeBundles(url string) ([]Bundle, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set user agent to avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var bundles []Bundle
	
	// Scrape bundle information - adjust selectors based on Fanatical's structure
	doc.Find(".bundle-item, .card-bundle, .bundle-card").Each(func(i int, s *goquery.Selection) {
		bundle := Bundle{}
		
		// Extract title
		titleEl := s.Find(".bundle-title, .card-title, h3, h4").First()
		bundle.Title = strings.TrimSpace(titleEl.Text())
		
		// Extract link
		linkEl := s.Find("a").First()
		if href, exists := linkEl.Attr("href"); exists {
			if strings.HasPrefix(href, "/") {
				bundle.Link = "https://www.fanatical.com" + href
			} else {
				bundle.Link = href
			}
		}
		
		// Extract description
		descEl := s.Find(".bundle-description, .card-description, .description").First()
		bundle.Description = strings.TrimSpace(descEl.Text())
		
		// Extract price
		priceEl := s.Find(".price, .bundle-price, .card-price").First()
		bundle.Price = strings.TrimSpace(priceEl.Text())
		
		// Extract image URL
		imgEl := s.Find("img").First()
		if src, exists := imgEl.Attr("src"); exists {
			if strings.HasPrefix(src, "/") {
				bundle.ImageURL = "https://www.fanatical.com" + src
			} else {
				bundle.ImageURL = src
			}
		}
		
		bundle.PubDate = time.Now()
		
		// Only add bundle if it has a title and link
		if bundle.Title != "" && bundle.Link != "" {
			bundles = append(bundles, bundle)
		}
	})

	return bundles, nil
}

func generateRSS(bundleType string, bundles []Bundle) RSS {
	channel := Channel{
		Title:         fmt.Sprintf("Fanatical %s Bundles", bundleType),
		Link:          "https://www.fanatical.com/de/bundle/" + strings.ToLower(bundleType),
		Description:   fmt.Sprintf("Latest %s bundles from Fanatical", strings.ToLower(bundleType)),
		Language:      "de-DE",
		LastBuildDate: time.Now().Format(time.RFC1123Z),
	}

	for _, bundle := range bundles {
		description := bundle.Description
		if bundle.Price != "" {
			description += fmt.Sprintf(" - Price: %s", bundle.Price)
		}
		if bundle.ImageURL != "" {
			description += fmt.Sprintf(`<br><img src="%s" alt="%s" style="max-width: 300px;">`, bundle.ImageURL, bundle.Title)
		}

		item := Item{
			Title:       bundle.Title,
			Link:        bundle.Link,
			Description: description,
			PubDate:     bundle.PubDate.Format(time.RFC1123Z),
			GUID:        bundle.Link,
		}
		channel.Items = append(channel.Items, item)
	}

	return RSS{
		Version: "2.0",
		Channel: channel,
	}
}

func writeRSSFile(filename string, rss RSS) error {
	xmlData, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return err
	}

	xmlString := xml.Header + string(xmlData)
	return ioutil.WriteFile(filename, []byte(xmlString), 0644)
}

func copyIndexFile() error {
	// Check if index.html exists in current directory
	if _, err := os.Stat("index.html"); os.IsNotExist(err) {
		// Create a basic index.html if it doesn't exist
		return createIndexFile()
	}

	// Copy existing index.html to docs directory
	input, err := ioutil.ReadFile("index.html")
	if err != nil {
		return err
	}

	return ioutil.WriteFile("docs/index.html", input, 0644)
}

func createIndexFile() error {
	indexContent := `<!DOCTYPE html>
<html lang="de">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Fanatical RSS Feeds</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: #333;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: rgba(255, 255, 255, 0.95);
            padding: 30px;
            border-radius: 15px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        }
        h1 {
            color: #2c3e50;
            text-align: center;
            margin-bottom: 30px;
        }
        .feed-section {
            margin: 20px 0;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 10px;
            border-left: 4px solid #007bff;
        }
        .feed-title {
            font-size: 1.2em;
            font-weight: bold;
            color: #2c3e50;
            margin-bottom: 10px;
        }
        .feed-link {
            display: inline-block;
            padding: 8px 16px;
            background: #007bff;
            color: white;
            text-decoration: none;
            border-radius: 5px;
            transition: background 0.3s;
        }
        .feed-link:hover {
            background: #0056b3;
        }
        .description {
            margin: 15px 0;
            color: #666;
        }
        .footer {
            text-align: center;
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #eee;
            color: #777;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎮 Fanatical RSS Feeds</h1>
        <p class="description">
            Bleiben Sie auf dem Laufenden über die neuesten Bundles von Fanatical! 
            Fügen Sie diese RSS-Feeds zu Ihrem bevorzugten RSS-Reader hinzu.
        </p>

        <div class="feed-section">
            <div class="feed-title">📚 Books</div>
            <div class="description">E-Book Bundles von Fanatical</div>
            <a href="books.rss" class="feed-link">RSS Feed</a>
        </div>

        <div class="feed-section">
            <div class="feed-title">🎮 Games</div>
            <div class="description">Spiele-Bundles von Fanatical</div>
            <a href="games.rss" class="feed-link">RSS Feed</a>
        </div>

        <div class="feed-section">
            <div class="feed-title">💻 Software</div>
            <div class="description">Software-Bundles von Fanatical</div>
            <a href="software.rss" class="feed-link">RSS Feed</a>
        </div>

        <div class="footer">
            <p>Diese Website steht in keiner Verbindung zu Fanatical. Alle Markenzeichen gehören ihren jeweiligen Eigentümern.</p>
        </div>
    </div>
</body>
</html>`

	return ioutil.WriteFile("docs/index.html", []byte(indexContent), 0644)
}
