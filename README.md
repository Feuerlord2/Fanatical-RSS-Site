# Fanatical Bundle RSS Generator

🎮 An automated RSS feed generator for game bundles from [Fanatical.com](https://www.fanatical.com/de/bundle/games)

## 📋 Overview

This project automatically creates an RSS feed with current game bundles from Fanatical. The feed is updated daily at 17:00 UTC and served via GitHub Pages.

## ✨ Features

- 🔄 **Automatic Updates**: Daily at 17:00 UTC via GitHub Actions
- 📱 **RSS 2.0 Compatible**: Works with all popular RSS readers
- 🌐 **Web Interface**: User-friendly website to subscribe to the feed
- 🎯 **Detailed Information**: Bundle titles, prices, game counts, and links
- 🚀 **GitHub Pages**: Free hosting via GitHub
- 🔧 **Robust**: Fallback to mock data when scraping fails

## 🚀 Quick Start

### 1. Repository Setup

```bash
# Fork or clone the repository
git clone https://github.com/username/fanatical-bundle-rss.git
cd fanatical-bundle-rss

# Install Go dependencies
go mod download
```

### 2. Local Development

```bash
# Generate RSS feed
go run cmd/generator/main.go > docs/rss.xml

# With file output for testing
go run cmd/generator/main.go --file
```

### 3. Enable GitHub Pages

1. Go to repository **Settings** → **Pages**
2. Set Source to **GitHub Actions**
3. The workflow will be automatically activated

### 4. Subscribe to RSS Feed

```
https://username.github.io/fanatical-bundle-rss/rss.xml
```

## 📁 Project Structure

```
fanatical-bundle-rss/
├── .github/workflows/
│   └── update-rss.yml          # GitHub Actions Workflow
├── cmd/generator/
│   └── main.go                 # Main application
├── internal/
│   ├── models/
│   │   └── bundle.go          # Data models
│   ├── scraper/
│   │   └── scraper.go         # Web scraping logic
│   └── rss/
│       └── generator.go       # RSS generation
├── web/
│   ├── index.html             # Web interface
│   └── style.css              # Styling
├── docs/
│   ├── rss.xml               # Generated RSS feed
│   └── last_updated.txt      # Update timestamp
├── go.mod                    # Go module definition
├── go.sum                    # Go dependencies checksums
├── README.md                 # This file
├── LICENSE                   # MIT License
└── .gitignore               # Git ignore rules
```

## 🔧 Configuration

### GitHub Actions Workflow

The workflow (`.github/workflows/update-rss.yml`) runs:
- **Automatically**: Daily at 17:00 UTC
- **Manually**: Via GitHub Actions UI
- **On Push**: To the main branch

### Environment Variables

No special environment variables required. The workflow uses:
- `GITHUB_TOKEN`: Automatically provided by GitHub
- Standard Go 1.21 environment

### Customizations

**Change RSS feed metadata:**

```go
// In cmd/generator/main.go
generator := rss.NewGenerator()
generator.SetFeedMetadata(
    "My Custom Feed Title",
    "https://example.com",
    "Custom Description",
    "en-US"
)
```

**Change update time:**

```yaml
# In .github/workflows/update-rss.yml
schedule:
  - cron: '0 15 * * *'  # 15:00 UTC instead of 17:00 UTC
```

## 🛠️ Development

### Prerequisites

- Go 1.21 or higher
- Git
- GitHub Account (for Actions and Pages)

### Dependencies

```bash
go get github.com/PuerkitoBio/goquery@latest
go get golang.org/x/net@latest
```

### Local Testing

```bash
# Run tests
go test ./...

# Test RSS feed
go run cmd/generator/main.go --file
cat rss.xml

# Test web interface
cd web && python -m http.server 8000
# Open http://localhost:8000
```

### Code Structure

**Scraper (`internal/scraper/`)**
- `FetchBundles()`: Main scraping function
- Robust HTML parsing with multiple fallback strategies
- Handles anti-bot measures

**RSS Generator (`internal/rss/`)**
- RSS 2.0 compliant XML generation
- HTML-formatted item descriptions
- Automatic categorization

**Models (`internal/models/`)**
- Bundle data structure
- Validation and defaults
- Utility functions

## 📊 Monitoring

### Check Status

```bash
# Feed status via curl
curl -I https://username.github.io/fanatical-bundle-rss/rss.xml

# Last update
curl https://username.github.io/fanatical-bundle-rss/last_updated.txt
```

### GitHub Actions Logs

1. Go to **Actions** tab in repository
2. Click on latest "Update RSS Feed" workflow
3. Check logs for errors

### Error Handling

The scraper has multiple fallback mechanisms:
- Multiple CSS selectors
- Fallback parsing strategies
- Mock data for complete failures

## 🤝 Contributing

### Report Issues

Use GitHub Issues for:
- 🐛 Bug Reports
- 💡 Feature Requests
- 📚 Documentation Improvements

### Pull Requests

1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Create pull request

### Code Standards

- Go fmt for formatting
- Meaningful commit messages
- Tests for new features
- Documentation for public APIs

## 📱 RSS Reader Recommendations

### Desktop
- **Windows**: RSSOwl, QuiteRSS
- **macOS**: NetNewsWire, Reeder
- **Linux**: Akregator, Liferea

### Mobile
- **iOS**: Reeder, NetNewsWire, Unread
- **Android**: Feedly, Inoreader, ReadYou

### Web-based
- **Feedly**: Free with premium features
- **Inoreader**: Very comprehensive
- **The Old Reader**: Simple and clean

## 🔒 Privacy & Security

### Data Collection
- No personal data stored
- Only public bundle information from Fanatical
- No cookies or tracking

### Security
- Read-only access to Fanatical
- No sensitive credentials
- HTTPS for all connections

## ⚖️ Legal

### Fair Use
- Only publicly available information
- No copyright violations
- Transformative use (RSS format)

### Disclaimer
- Not officially affiliated with Fanatical
- No guarantee of availability
- Use at your own responsibility

## 🚨 Troubleshooting

### Common Issues

**RSS feed not updating:**
```bash
# Trigger workflow manually
gh workflow run "Update RSS Feed"

# Or via GitHub UI: Actions → Update RSS Feed → Run workflow
```

**Scraping fails:**
- Fanatical may have changed HTML structure
- Rate limiting or anti-bot measures
- Adjust CSS selectors in `scraper.go`

**GitHub Pages not available:**
- Enable Pages in repository settings
- Check branch on `gh-pages` or Actions deployment
- Wait for DNS propagation (up to 24h)

### Debug Mode

```bash
# Enable verbose logging
export DEBUG=true
go run cmd/generator/main.go --file

# Save HTML output for debugging
curl "https://www.fanatical.com/de/bundle/games" > debug.html
```

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Credits

- [Fanatical](https://www.fanatical.com/) for bundle data
- [GoQuery](https://github.com/PuerkitoBio/goquery) for HTML parsing
- [GitHub Actions](https://github.com/features/actions) for automation
- [GitHub Pages](https://pages.github.com/) for hosting

---

**⭐ If you like this project, give it a star on GitHub!**
