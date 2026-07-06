package gofanatical

import (
	"regexp"
	"strings"
)

var (
	bookKeywords     = []string{"book", "comic", "certification", "learning", "training", "course"}
	softwareKeywords = []string{"software", "excel", "beats and vibes", "global beats"}
	// "app" needs word boundaries so titles like "Happy Farm" don't match.
	appWordPattern = regexp.MustCompile(`\bapps?\b`)
)

// categorizeBundle assigns each bundle to exactly one of "books", "games"
// or "software". The display_type field is the most reliable signal; title
// keywords catch bundles the API only labels with the generic "bundle" type.
func categorizeBundle(ab AlgoliaBundle) string {
	switch strings.ToLower(ab.DisplayType) {
	case "book-bundle", "comic-bundle":
		return "books"
	case "software-bundle", "audio-bundle", "elearning-bundle":
		return "software"
	}

	title := strings.ToLower(ab.Name)
	for _, keyword := range bookKeywords {
		if strings.Contains(title, keyword) {
			return "books"
		}
	}
	for _, keyword := range softwareKeywords {
		if strings.Contains(title, keyword) {
			return "software"
		}
	}
	if appWordPattern.MatchString(title) {
		return "software"
	}

	return "games"
}
