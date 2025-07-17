// 1. ERWEITERE Run() Funktion um fallback kategorie
func Run() {
	wg := sync.WaitGroup{}
	for _, category := range []string{"books", "games", "software", "fallback"} {
		wg.Add(1)
		go updateCategory(&wg, category)
	}
	wg.Wait()
}

// 2. ÜBERARBEITE shouldIncludeBundle für saubere Kategorisierung + Fallback
func shouldIncludeBundle(bundle FanaticalBundle, category string) bool {
	bundleCategory := strings.ToLower(bundle.Category)
	targetCategory := strings.ToLower(category)
	
	// DEBUG: Log bundle info für problematische Kategorien
	if targetCategory == "books" || targetCategory == "software" || targetCategory == "fallback" {
		log.WithFields(log.Fields{
			"bundle_title":    bundle.Title,
			"bundle_category": bundleCategory,
			"target_category": targetCategory,
			"bundle_type":     strings.ToLower(bundle.Type),
		}).Info("Checking bundle for category")
	}
	
	// Direct category match
	if bundleCategory == targetCategory {
		return true
	}
	
	title := strings.ToLower(bundle.Title)
	description := strings.ToLower(bundle.Description)
	
	// Hilfsfunktionen für bessere Kategorisierung
	isBooks := strings.Contains(title, "certification") ||
	          strings.Contains(title, "learning") ||
	          strings.Contains(title, "elearning") ||
	          strings.Contains(title, "training") ||
	          strings.Contains(title, "course") ||
	          (strings.Contains(title, "development") && !strings.Contains(title, "game")) ||
	          strings.Contains(title, "programming") ||
	          strings.Contains(title, "coding") ||
	          strings.Contains(title, "security") ||
	          strings.Contains(title, "cloud") ||
	          strings.Contains(title, "machine learning") ||
	          (strings.Contains(title, "python") && !strings.Contains(title, "game")) ||
	          strings.Contains(title, "c#") ||
	          strings.Contains(title, "graphics and design") ||
	          strings.Contains(title, "business computing") ||
	          strings.Contains(title, "network") ||
	          strings.Contains(title, "robotics") ||
	          strings.Contains(title, "digital life")
	
	isGames := bundleCategory == "games" ||
	          strings.Contains(title, "game") ||
	          strings.Contains(title, "rpg") ||
	          strings.Contains(title, "fantasy") ||
	          strings.Contains(title, "strategy") ||
	          strings.Contains(title, "capcom") ||
	          strings.Contains(title, "brutal") ||
	          strings.Contains(title, "chillout") ||
	          strings.Contains(title, "favorites") ||
	          strings.Contains(title, "point and click") ||
	          strings.Contains(title, "steam") ||
	          strings.Contains(description, "game") ||
	          strings.Contains(title, "voucher")
	
	isSoftware := strings.Contains(title, "software") ||
	             strings.Contains(title, "app") ||
	             strings.Contains(description, "software") ||
	             strings.Contains(description, "app") ||
	             strings.Contains(title, "excel") ||
	             strings.Contains(title, "zenva")
	
	switch targetCategory {
	case "books":
		// Explizit Gaming-bezogene RPG Bundles ausschließen!
		if strings.Contains(title, "rpg and fantasy") || 
		   strings.Contains(title, "game") ||
		   strings.Contains(title, "gaming") {
			return false
		}
		
		if isBooks {
			log.WithField("bundle_title", bundle.Title).Info("BOOKS: Bundle matched!")
		}
		return isBooks
		
	case "games":
		// Exclusions für Games
		if strings.Contains(title, "certification") || 
		   strings.Contains(title, "learning") ||
		   strings.Contains(title, "training") ||
		   strings.Contains(title, "course") ||
		   strings.Contains(title, "software") {
			return false
		}
		
		return isGames
		
	case "software":
		if isSoftware {
			log.WithField("bundle_title", bundle.Title).Info("SOFTWARE: Bundle matched!")
		}
		return isSoftware
		
	case "fallback":
		// Fallback: Alles was NICHT in die anderen drei Kategorien gehört
		shouldInclude := !isBooks && !isGames && !isSoftware
		if shouldInclude {
			log.WithField("bundle_title", bundle.Title).Warn("FALLBACK: Bundle doesn't match any category!")
		}
		return shouldInclude
		
	default:
		return true
	}
}

// 3. FÜGE Duplikat-Entfernung hinzu - NEUE Funktion
func removeDuplicateBundles(bundles []FanaticalBundle) []FanaticalBundle {
	seen := make(map[string]bool)
	var uniqueBundles []FanaticalBundle
	
	for _, bundle := range bundles {
		// Erstelle einen einzigartigen Key basierend auf Slug + StartDate
		key := fmt.Sprintf("%s-%d", bundle.Slug, bundle.StartDate.Unix())
		
		if !seen[key] {
			seen[key] = true
			uniqueBundles = append(uniqueBundles, bundle)
		} else {
			log.WithFields(log.Fields{
				"bundle_title": bundle.Title,
				"bundle_slug":  bundle.Slug,
				"duplicate_key": key,
			}).Info("Duplicate bundle removed")
		}
	}
	
	log.WithFields(log.Fields{
		"original_count": len(bundles),
		"unique_count":   len(uniqueBundles),
		"duplicates_removed": len(bundles) - len(uniqueBundles),
	}).Info("Duplicate removal completed")
	
	return uniqueBundles
}

// 4. ÜBERARBEITE updateCategory um Duplikate zu entfernen
func updateCategory(wg *sync.WaitGroup, category string) {
	defer wg.Done()

	log.WithField("category", category).Info("Fetching data from Fanatical APIs")
	
	var allBundles []FanaticalBundle
	
	// Fetch from /api/all/de (Pick-and-Mix + StarDeals)
	newApiBundles, err := fetchBundlesFromNewAPI()
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to fetch bundles from /api/all/de")
	} else {
		allBundles = append(allBundles, newApiBundles...)
		log.WithField("new_api_bundles", len(newApiBundles)).Info("Added bundles from /api/all/de")
	}
	
	// Fetch from algolia API (normale Bundles) - mit Compression-Fix
	algoliaApiBundles, err := fetchBundlesFromAlgoliaAPI()
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to fetch bundles from algolia API")
	} else {
		allBundles = append(allBundles, algoliaApiBundles...)
		log.WithField("algolia_bundles", len(algoliaApiBundles)).Info("Added bundles from algolia API")
	}

	// Also fetch promotions
	promotions, err := fetchPromotionsFromAPI()
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to fetch promotions from API")
	} else {
		// Convert promotions to bundles and add them
		promotionBundles := convertPromotionsToBundles(promotions, category)
		allBundles = append(allBundles, promotionBundles...)
		log.WithField("promotion_bundles", len(promotionBundles)).Info("Added promotion bundles")
	}
	
	log.WithField("total_bundles_before_dedup", len(allBundles)).Info("Total bundles collected from all APIs")
	
	// *** NEU: Entferne Duplikate BEVOR gefiltert wird ***
	allBundles = removeDuplicateBundles(allBundles)
	log.WithField("total_bundles_after_dedup", len(allBundles)).Info("Total bundles after duplicate removal")
	
	// Filter bundles by category
	var filteredBundles []FanaticalBundle
	for _, bundle := range allBundles {
		if shouldIncludeBundle(bundle, category) {
			filteredBundles = append(filteredBundles, bundle)
		}
	}
	
	log.WithFields(log.Fields{
		"category": category,
		"total":    len(allBundles),
		"filtered": len(filteredBundles),
	}).Info("Bundle filtering completed")
	
	// If no bundles found after filtering, create a test bundle (außer für fallback)
	if len(filteredBundles) == 0 && category != "fallback" {
		log.WithField("category", category).Warn("No bundles found for category, creating test bundle")
		filteredBundles = createTestBundle(category)
	}
	
	// Für fallback: Wenn leer, dann erstelle eine leere RSS oder überspringe
	if len(filteredBundles) == 0 && category == "fallback" {
		log.WithField("category", category).Info("No fallback bundles found - creating empty feed")
		// Erstelle leeren Feed für fallback
		feed := feeds.Feed{
			Title:       "Fanatical RSS Fallback Bundles",
			Link:        &feeds.Link{Href: "https://feuerlord2.github.io/Fanatical-RSS-Site/"},
			Description: "Bundles that don't fit into any other category",
			Author:      &feeds.Author{Name: "Daniel Winter", Email: "DanielWinterEmsdetten+rss@gmail.com"},
			Created:     time.Now(),
			Items:       []*feeds.Item{}, // Leere Items
		}
		
		if err := writeFeedToFile(feed, category); err != nil {
			log.WithFields(log.Fields{
				"category": category,
				"error":    err.Error(),
			}).Error("Failed to write empty fallback feed to file")
		} else {
			log.WithField("category", category).Info("Successfully created empty fallback RSS feed")
		}
		return
	}
	
	feed, err := createFeed(filteredBundles, category)
	if err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to create feed")
		return
	}

	if err := writeFeedToFile(feed, category); err != nil {
		log.WithFields(log.Fields{
			"category": category,
			"error":    err.Error(),
		}).Error("Failed to write feed to file")
	} else {
		log.WithFields(log.Fields{
			"category": category,
			"bundles":  len(filteredBundles),
		}).Info("Successfully created RSS feed")
	}
}
