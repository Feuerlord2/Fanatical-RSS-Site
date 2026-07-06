package gofanatical

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/gorilla/feeds"
)

var categories = []string{"books", "games", "software"}

// Run fetches all bundles once, then writes one RSS feed per category.
// It returns a non-nil error if fetching fails or any feed cannot be
// written, so the caller can exit non-zero and CI turns red instead of
// silently serving stale feeds.
func Run() error {
	configureLogging()

	bundles, err := fetchBundles()
	if err != nil {
		return fmt.Errorf("failed to fetch bundles: %w", err)
	}

	bundles = removeDuplicateBundles(bundles)

	var errs []error
	for _, category := range categories {
		var filtered []FanaticalBundle
		for _, bundle := range bundles {
			if bundle.Category == category {
				filtered = append(filtered, bundle)
			}
		}

		if len(filtered) == 0 {
			slog.Warn("no bundles found for category, creating empty feed", "category", category)
		}

		feed := createFeed(filtered, category)
		if err := writeFeedToFile(feed, category); err != nil {
			errs = append(errs, fmt.Errorf("category %s: %w", category, err))
			continue
		}
		slog.Info("successfully created RSS feed", "category", category, "bundles", len(filtered))
	}

	return errors.Join(errs...)
}

func configureLogging() {
	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}

func createFeed(bundles []FanaticalBundle, category string) feeds.Feed {
	feed := feeds.Feed{
		Title:       fmt.Sprintf("Fanatical RSS %s Bundles", strings.ToUpper(category[:1])+category[1:]),
		Link:        &feeds.Link{Href: "https://feuerlord2.github.io/Fanatical-RSS-Site/"},
		Description: fmt.Sprintf("Latest Fanatical %s bundles with amazing deals and discounts!", category),
		Author:      &feeds.Author{Name: "Daniel Winter", Email: "DanielWinterEmsdetten+rss@gmail.com"},
	}

	// Newest bundles first.
	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].StartDate.After(bundles[j].StartDate)
	})

	feed.Items = make([]*feeds.Item, len(bundles))
	for idx, bundle := range bundles {
		item := &feeds.Item{
			Title:       bundle.Title,
			Link:        &feeds.Link{Href: "https://www.fanatical.com" + bundle.URL},
			Content:     createRichContent(bundle),
			Created:     bundle.StartDate,
			Description: bundle.Description,
			// The GUID format must stay stable across releases — changing it
			// makes every feed reader re-deliver all items as new.
			Id: fmt.Sprintf("fanatical-%s-%d", bundle.Slug, bundle.StartDate.Unix()),
		}

		// Cover image as enclosure for Discord embed support.
		if bundle.Image != "" {
			item.Enclosure = &feeds.Enclosure{
				Url:    bundle.Image,
				Type:   imageMIMEType(bundle.Image),
				Length: "0",
			}
		}

		feed.Items[idx] = item
	}

	// Derive the feed timestamp from the newest bundle instead of time.Now()
	// so the generated XML only changes when the content changes. This keeps
	// the auto-commit workflow from committing on every run.
	if len(bundles) > 0 {
		feed.Created = bundles[0].StartDate
	}

	return feed
}

func removeDuplicateBundles(bundles []FanaticalBundle) []FanaticalBundle {
	seen := make(map[string]bool)
	var unique []FanaticalBundle

	for _, bundle := range bundles {
		key := fmt.Sprintf("%s-%d", bundle.Slug, bundle.StartDate.Unix())
		if seen[key] {
			slog.Debug("duplicate bundle removed", "bundle_title", bundle.Title, "duplicate_key", key)
			continue
		}
		seen[key] = true
		unique = append(unique, bundle)
	}

	if removed := len(bundles) - len(unique); removed > 0 {
		slog.Info("duplicate removal completed", "original_count", len(bundles), "duplicates_removed", removed)
	}

	return unique
}

func writeFeedToFile(feed feeds.Feed, category string) error {
	if err := os.MkdirAll("docs", 0o755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	filename := fmt.Sprintf("docs/%s.rss", category)
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create RSS file %s: %w", filename, err)
	}
	defer f.Close()

	rss, err := feed.ToRss()
	if err != nil {
		return fmt.Errorf("failed to generate RSS content: %w", err)
	}

	w := bufio.NewWriter(f)
	if _, err := w.WriteString(rss); err != nil {
		return fmt.Errorf("failed to write RSS content: %w", err)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush RSS file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close RSS file: %w", err)
	}

	slog.Info("RSS feed written", "category", category, "file", filename, "size", len(rss))
	return nil
}
