# Fanatical-RSS-Site

RSS feeds for Fanatical bundle deals. Queries the Fanatical Algolia API every 6 hours and generates three RSS feeds — books, games, software.

**Live:** https://feuerlord2.github.io/Fanatical-RSS-Site/

## Feeds

```
https://feuerlord2.github.io/Fanatical-RSS-Site/books.rss
https://feuerlord2.github.io/Fanatical-RSS-Site/games.rss
https://feuerlord2.github.io/Fanatical-RSS-Site/software.rss
```

Add these to any RSS reader, Discord bot, or news aggregator. Each item includes current price, original price, discount percentage, and a direct link to the deal.

## How it works

A Go program fetches Fanatical's public Algolia API endpoint once (with retries), deduplicates the bundles, assigns each one to exactly one category (books/games/software, based on `display_type` with title-keyword fallbacks), and writes one RSS 2.0 file per category. GitHub Actions runs this on a schedule, commits changed feeds, and deploys `docs/` to GitHub Pages.

Feed timestamps are derived from the newest bundle rather than the current time, so unchanged content produces byte-identical XML and the workflow only commits when there are actual new deals. If the API is unreachable, the program exits non-zero and the workflow run fails visibly instead of silently serving stale feeds.

## Running locally

```
make run        # build and generate feeds into docs/
make test       # run the test suite
make dev        # run with debug logging
make serve      # preview docs/ at http://localhost:8080
```

Requires Go 1.24+. Only external dependency is [gorilla/feeds](https://github.com/gorilla/feeds); logging uses the standard library `log/slog`.

## Project structure

```
cmd/gofanatical.go   Entry point (exit code 1 on failure)
pkg/fetch.go         API fetching with retries, conversion to internal types
pkg/categorize.go    Category assignment (books/games/software)
pkg/content.go       HTML item content (escaped), currency/MIME helpers
pkg/feed.go          Run() orchestration, RSS generation, file output
pkg/model.go         Data types (FanaticalBundle, Price)
pkg/*_test.go        Unit tests incl. a stub-server fetch test
docs/                GitHub Pages output (HTML + RSS files)
Makefile             Build, run, test, serve targets
```

Note: the RSS item GUID format (`fanatical-<slug>-<start-unix>`) is a stable contract with subscribers' feed readers — changing it would re-deliver every item as new.

## Disclaimer

Not affiliated with Fanatical. This is a private hobby project.

## License

MIT — see [LICENSE](LICENSE).
