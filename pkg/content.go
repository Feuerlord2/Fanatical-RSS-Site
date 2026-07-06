package gofanatical

import (
	"fmt"
	"html"
	"path"
	"strings"
)

// createRichContent renders the HTML body of a feed item. All dynamic
// values are HTML-escaped — bundle titles and descriptions come from an
// external API and must not be trusted as markup.
func createRichContent(bundle FanaticalBundle) string {
	var content strings.Builder

	title := html.EscapeString(bundle.Title)
	description := html.EscapeString(bundle.Description)
	link := html.EscapeString("https://www.fanatical.com" + bundle.URL)

	if bundle.Image != "" {
		content.WriteString(fmt.Sprintf("<img src=\"%s\" alt=\"%s\" style=\"max-width: 100%%; border-radius: 8px; margin-bottom: 10px;\" />\n",
			html.EscapeString(bundle.Image), title))
	}

	content.WriteString(fmt.Sprintf("<h3>%s</h3>\n", title))
	content.WriteString(fmt.Sprintf("<p>%s</p>\n", description))

	symbol := currencySymbol(bundle.Price.Currency)

	currentPrice := fmt.Sprintf("%s%.2f", symbol, bundle.Price.Amount)
	if bundle.Price.Amount == 0 {
		currentPrice = "FREE"
	}

	originalPrice := fmt.Sprintf("%s%.2f", symbol, bundle.Price.Original)
	if bundle.Price.Original == 0 {
		originalPrice = "N/A"
	}

	savings := bundle.Price.Original - bundle.Price.Amount
	savingsText := fmt.Sprintf("%s%.2f", symbol, savings)
	if savings <= 0 {
		savingsText = "N/A"
	}

	content.WriteString("<table border='1' style='border-collapse: collapse; margin: 10px 0;'>\n")
	content.WriteString("<tr style='background-color: #f0f0f0;'><th style='padding: 5px;'>Current Price</th><th style='padding: 5px;'>Original Price</th><th style='padding: 5px;'>Discount</th><th style='padding: 5px;'>You Save</th></tr>\n")
	content.WriteString(fmt.Sprintf("<tr><td style='padding: 5px; text-align: center;'><strong>%s</strong></td><td style='padding: 5px; text-align: center;'>%s</td><td style='padding: 5px; text-align: center;'>%d%%</td><td style='padding: 5px; text-align: center;'>%s</td></tr>\n",
		currentPrice, originalPrice, bundle.Price.Discount, savingsText))
	content.WriteString("</table>\n")

	// No "time remaining" line here: it would be computed from the current
	// time, making the generated XML differ on every run even when nothing
	// changed — which would defeat the only-commit-on-real-changes behavior.
	content.WriteString("<h4>⏰ Availability</h4>\n")
	content.WriteString("<ul>\n")
	content.WriteString(fmt.Sprintf("<li><strong>Ends:</strong> %s</li>\n", bundle.EndDate.UTC().Format("January 2, 2006 15:04 MST")))
	content.WriteString("</ul>\n")

	content.WriteString(fmt.Sprintf("<p><a href='%s' style='background-color: #ff6f00; color: white; padding: 10px 15px; text-decoration: none; border-radius: 5px;'>🛒 Get this deal on Fanatical</a></p>\n", link))

	return content.String()
}

func currencySymbol(code string) string {
	switch code {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "CAD":
		return "CA$"
	case "AUD":
		return "A$"
	}
	return code + " "
}

// imageMIMEType guesses the enclosure MIME type from the image URL.
func imageMIMEType(url string) string {
	if i := strings.IndexAny(url, "?#"); i >= 0 {
		url = url[:i]
	}
	switch strings.ToLower(path.Ext(url)) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
