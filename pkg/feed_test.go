package gofanatical

import (
	"strings"
	"testing"
	"time"
)

func testBundle(slug string, start time.Time) FanaticalBundle {
	return FanaticalBundle{
		Title:     "Bundle " + slug,
		Slug:      slug,
		URL:       "/en/bundle/" + slug,
		StartDate: start,
		EndDate:   start.Add(14 * 24 * time.Hour),
		Price:     Price{Currency: "USD", Amount: 4.99, Original: 9.99, Discount: 50},
	}
}

func TestRemoveDuplicateBundles(t *testing.T) {
	start := time.Unix(1000, 0)
	bundles := []FanaticalBundle{
		testBundle("alpha", start),
		testBundle("alpha", start),                // exact duplicate
		testBundle("alpha", start.Add(time.Hour)), // same slug, new start = new deal
		testBundle("beta", start),
	}

	got := removeDuplicateBundles(bundles)
	if len(got) != 3 {
		t.Fatalf("expected 3 unique bundles, got %d", len(got))
	}
}

func TestCreateFeedSortsNewestFirst(t *testing.T) {
	old := testBundle("old", time.Unix(1000, 0))
	newer := testBundle("newer", time.Unix(2000, 0))

	feed := createFeed([]FanaticalBundle{old, newer}, "games")

	if len(feed.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(feed.Items))
	}
	if feed.Items[0].Title != "Bundle newer" {
		t.Errorf("newest bundle should be first, got %q", feed.Items[0].Title)
	}
}

func TestCreateFeedDeterministicOutput(t *testing.T) {
	sameStart := time.Unix(1000, 0)
	makeBundles := func(order []string) []FanaticalBundle {
		var bs []FanaticalBundle
		for _, slug := range order {
			bs = append(bs, testBundle(slug, sameStart))
		}
		return bs
	}

	// Same bundles, different input order (e.g. Algolia re-ranking) must
	// produce byte-identical RSS, otherwise CI commits phantom changes.
	feed1 := createFeed(makeBundles([]string{"zeta", "alpha", "mid"}), "games")
	rss1, err := feed1.ToRss()
	if err != nil {
		t.Fatal(err)
	}
	feed2 := createFeed(makeBundles([]string{"mid", "zeta", "alpha"}), "games")
	rss2, err := feed2.ToRss()
	if err != nil {
		t.Fatal(err)
	}
	if rss1 != rss2 {
		t.Error("same content in different input order produced different XML")
	}
}

func TestCreateFeedStableTimestamp(t *testing.T) {
	newest := time.Unix(2000, 0)
	feed := createFeed([]FanaticalBundle{
		testBundle("a", time.Unix(1000, 0)),
		testBundle("b", newest),
	}, "games")

	// The feed timestamp must derive from content, not from time.Now(),
	// so unchanged content produces byte-identical XML across runs.
	if !feed.Created.Equal(newest) {
		t.Errorf("feed.Created = %v, want %v", feed.Created, newest)
	}
}

func TestCreateFeedGUIDStability(t *testing.T) {
	start := time.Unix(1234, 0)
	feed := createFeed([]FanaticalBundle{testBundle("my-slug", start)}, "games")

	// This exact GUID format is what existing subscribers' readers have
	// stored. Never change it, or every item re-delivers as new.
	want := "fanatical-my-slug-1234"
	if feed.Items[0].Id != want {
		t.Errorf("GUID = %q, want %q", feed.Items[0].Id, want)
	}
}

func TestCreateFeedRendersValidRSS(t *testing.T) {
	bundle := testBundle("render-me", time.Unix(1000, 0))
	bundle.Image = "https://example.com/cover.png"
	feed := createFeed([]FanaticalBundle{bundle}, "software")

	rss, err := feed.ToRss()
	if err != nil {
		t.Fatalf("ToRss failed: %v", err)
	}
	for _, want := range []string{"<rss", "Fanatical RSS Software Bundles", "fanatical-render-me-1000", `type="image/png"`} {
		if !strings.Contains(rss, want) {
			t.Errorf("RSS output missing %q", want)
		}
	}
}

func TestCreateFeedEmptyCategory(t *testing.T) {
	feed := createFeed(nil, "books")
	if _, err := feed.ToRss(); err != nil {
		t.Fatalf("empty feed must still render: %v", err)
	}
}
