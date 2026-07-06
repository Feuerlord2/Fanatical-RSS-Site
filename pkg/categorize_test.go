package gofanatical

import "testing"

func TestCategorizeBundle(t *testing.T) {
	tests := []struct {
		name   string
		bundle AlgoliaBundle
		want   string
	}{
		{"book display type", AlgoliaBundle{Name: "Fantasy Mega Bundle", DisplayType: "book-bundle"}, "books"},
		{"comic display type", AlgoliaBundle{Name: "Super Heroes", DisplayType: "comic-bundle"}, "books"},
		{"software display type", AlgoliaBundle{Name: "Dev Tools", DisplayType: "software-bundle"}, "software"},
		{"audio display type", AlgoliaBundle{Name: "Beats Pack", DisplayType: "audio-bundle"}, "software"},
		{"elearning display type", AlgoliaBundle{Name: "Coding 101", DisplayType: "elearning-bundle"}, "software"},
		{"plain game bundle", AlgoliaBundle{Name: "Killer Bundle 30", DisplayType: "bundle", Type: "bundle"}, "games"},
		{"pick and mix", AlgoliaBundle{Name: "Build Your Own Slayer Bundle", Type: "pick-and-mix"}, "games"},
		{"star deal game", AlgoliaBundle{Name: "ELDEN RING", Type: "game", StarDeal: true}, "games"},
		{"course by title", AlgoliaBundle{Name: "Complete Python Course Bundle", Type: "bundle"}, "books"},
		{"certification by title", AlgoliaBundle{Name: "IT Certification Vault", Type: "bundle"}, "books"},
		{"training by title", AlgoliaBundle{Name: "AWS Training Collection", Type: "bundle"}, "books"},
		{"comic by title", AlgoliaBundle{Name: "2000AD Comic Collection", Type: "bundle"}, "books"},
		{"software by title", AlgoliaBundle{Name: "Essential Software Toolkit", Type: "bundle"}, "software"},
		{"excel by title", AlgoliaBundle{Name: "Excel Master Bundle", Type: "bundle"}, "software"},
		{"audio by title", AlgoliaBundle{Name: "Global Beats: House Edition", Type: "bundle"}, "software"},
		{"app as whole word", AlgoliaBundle{Name: "Productivity App Bundle", Type: "bundle"}, "software"},
		{"app inside another word stays games", AlgoliaBundle{Name: "Happy Farm Adventures", Type: "bundle"}, "games"},
		{"apple stays games", AlgoliaBundle{Name: "Golden Apple Quest", Type: "bundle"}, "games"},
		{"book as whole word", AlgoliaBundle{Name: "Big Book Bundle", Type: "bundle"}, "books"},
		{"ebook as whole word", AlgoliaBundle{Name: "eBook Mega Collection", Type: "bundle"}, "books"},
		{"book inside another word stays games", AlgoliaBundle{Name: "Bookworm Adventures", Type: "bundle"}, "games"},
		{"display type beats title keywords", AlgoliaBundle{Name: "Learning Curve", DisplayType: "book-bundle"}, "books"},
		{"unknown everything defaults to games", AlgoliaBundle{Name: "Mystery Box"}, "games"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := categorizeBundle(tt.bundle); got != tt.want {
				t.Errorf("categorizeBundle(%q) = %q, want %q", tt.bundle.Name, got, tt.want)
			}
		})
	}
}
