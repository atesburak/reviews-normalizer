package yelp

import (
	"fmt"
	"math"
	"math/rand"

	"amsterdam-ratings/internal/config"
	"amsterdam-ratings/internal/sources"
)

// GenerateSynthetic builds a plausible fake Yelp dataset with a *known*,
// smaller injected neighborhood bias than the Google dry-run
// (internal/places.GenerateSynthetic's trueBias), modeling Yelp's
// closer-to-true-quality baseline per the cross-platform review-inflation
// literature this project is built around (see README). Venue names and
// positions mirror places.GenerateSynthetic's naming convention 1:1 so
// cmd/compare's name+distance matching has something realistic to find.
func GenerateSynthetic(seed int64, placesPerCell int) sources.Dataset {
	r := rand.New(rand.NewSource(seed + 1)) // offset so noise isn't identical to the Google dry-run's

	// ~20-30% of the Google dry-run's injected bias magnitude, same sign.
	yelpBias := map[string]float64{
		"Centrum":    0.15,
		"Jordaan":    0.10,
		"De Pijp":    0.05,
		"Oud-West":   0.03,
		"Oost":       0.00,
		"West":       -0.02,
		"Zuid":       0.02,
		"Nieuw-West": -0.05,
		"Noord":      -0.08,
		"Zuidoost":   -0.10,
	}
	categoryBase := map[string]float64{
		"restaurant":         3.9,
		"cafe":               4.0,
		"bar":                3.8,
		"museum":             4.3,
		"tourist_attraction": 3.9,
	}

	var ds sources.Dataset
	ds.Source = "yelp"
	ds.GeneratedBy = "dry-run"

	id := 0
	for _, n := range config.Neighborhoods {
		for _, cat := range config.Categories {
			alias := config.YelpCategoryAlias[cat]
			for i := 0; i < placesPerCell; i++ {
				id++
				base := categoryBase[cat]
				bias := yelpBias[n.Name]
				noise := r.NormFloat64() * 0.25
				rating := clamp(base+bias+noise, 1.0, 5.0)
				reviewCount := int(math.Max(3, r.NormFloat64()*150+150))

				ds.Ratings = append(ds.Ratings, sources.VenueRating{
					Source:       "yelp",
					ExternalID:   fmt.Sprintf("yelp-dryrun-%d", id),
					Name:         fmt.Sprintf("%s %s #%d", n.Name, cat, i+1),
					Neighborhood: n.Name,
					Category:     alias,
					Lat:          n.Lat,
					Lng:          n.Lng,
					Rating:       round1(rating),
					ReviewCount:  reviewCount,
				})
			}
		}
	}
	return ds
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
