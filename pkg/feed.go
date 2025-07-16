func shouldIncludeBundle(bundle FanaticalBundle, category string) bool {
	bundleCategory := strings.ToLower(bundle.Category)
	targetCategory := strings.ToLower(category)
	
	// DEBUG: Log bundle info für problematische Kategorien
	if targetCategory == "books" || targetCategory == "software" {
		log.WithFields(log.Fields{
			"bundle_title":    bundle.Title,
			"bundle_category": bundleCategory,
			"target_category": targetCategory,
		}).Info("Checking bundle for category")
	}
	
	// Direct category match - this should be the primary filter
	if bundleCategory == targetCategory {
		return true
	}
	
	// For games category, be more restrictive and exclude books/software content
	if targetCategory == "games" {
		title := strings.ToLower(bundle.Title)
		description := strings.ToLower(bundle.Description)
		
		// EXCLUDE these patterns that clearly indicate books/software
		excludePatterns := []string{
			"certification", "learning", "elearning", "training", "course",
			"programming", "coding", "development", "python", "c#",
			"graphics and design", "business computing", "network",
			"robotics", "digital life", "software", "app", "excel",
			"zenva", "machine learning", "cloud", "security",
		}
		
		for _, pattern := range excludePatterns {
			if strings.Contains(title, pattern) || strings.Contains(description, pattern) {
				log.WithFields(log.Fields{
					"bundle_title": bundle.Title,
					"pattern":      pattern,
					"category":     targetCategory,
				}).Info("GAMES: Bundle excluded due to pattern match")
				return false
			}
		}
		
		// Only include if it's clearly a game or has gaming indicators
		gameIndicators := []string{
			"game", "rpg", "fantasy", "strategy", "capcom", "brutal",
			"chillout", "favorites", "point and click", "steam",
			"voucher", "bundle", "pick-and-mix",
		}
		
		// If bundle category is already set to games, include it
		if bundleCategory == "games" {
			return true
		}
		
		// Check for game indicators
		for _, indicator := range gameIndicators {
			if strings.Contains(title, indicator) || strings.Contains(description, indicator) {
				return true
			}
		}
		
		// Only include bundles with empty/unknown categories if they don't match exclude patterns
		if bundleCategory == "" || bundle.Category == "" {
			return true
		}
		
		return false
	}
	
	// For books and software, use the existing logic but make it more precise
	title := strings.ToLower(bundle.Title)
	description := strings.ToLower(bundle.Description)
	
	switch targetCategory {
	case "books":
		// WICHTIG: Explizit Gaming-bezogene RPG Bundles ausschließen!
		if strings.Contains(title, "rpg and fantasy") || 
		   strings.Contains(title, "game") ||
		   strings.Contains(title, "gaming") {
			return false // Das sind Games, nicht Books!
		}
		
		// Books sind nur echte Bücher und Zertifizierungen
		shouldInclude := strings.Contains(title, "book") ||
		       strings.Contains(title, "certification")
		       
		if shouldInclude {
			log.WithField("bundle_title", bundle.Title).Info("BOOKS: Bundle matched!")
		}
		return shouldInclude
		
	case "software":
		shouldInclude := strings.Contains(title, "software") ||
		       strings.Contains(title, "app") ||
		       strings.Contains(description, "software") ||
		       strings.Contains(description, "app") ||
		       strings.Contains(title, "excel") ||
		       strings.Contains(title, "zenva")
		       
		if shouldInclude {
			log.WithField("bundle_title", bundle.Title).Info("SOFTWARE: Bundle matched!")
		}
		return shouldInclude
		
	default:
		return true
	}
}
