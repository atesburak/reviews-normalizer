package tripadvisor

import (
	"fmt"
	"math"
	"math/rand"

	"amsterdam-ratings/internal/config"
	"amsterdam-ratings/internal/sources"
)

// GenerateSynthetic builds a plausible fake TripAdvisor dataset for
// no-network smoke testing of cmd/compare, since there's no live API to hit
// (see this package's doc comment). Bias magnitude and sign mirror the Yelp
// dry-run (internal/sources/yelp.GenerateSynthetic) as a placeholder "also
// less inflated than Google" assumption — treat this as untested until
// checked against a real CSV import, since TripAdvisor's own tourist-heavy
// audience could plausibly cut the other way in Centrum-like areas.
func GenerateSynthetic(seed int64, placesPerCell int) sources.Dataset {
	r := rand.New(rand.NewSource(seed + 2)) // offset from both the Google and Yelp dry-runs' seeds

	taBias := map[string]float64{
		"Centrum":    0.12,
		"Jordaan":    0.08,
		"De Pijp":    0.04,
		"Oud-West":   0.02,
		"Oost":       0.00,
		"West":       -0.02,
		"Zuid":       0.02,
		"Nieuw-West": -0.04,
		"Noord":      -0.07,
		"Zuidoost":   -0.09,
	}
	categoryBase := map[string]float64{
		"restaurant":         4.0,
		"cafe":               4.1,
		"bar":                3.9,
		"museum":             4.4,
		"tourist_attraction": 4.2,
	}

	var ds sources.Dataset
	ds.Source = "tripadvisor"
	ds.GeneratedBy = "dry-run"

	id := 0
	for _, n := range config.Neighborhoods {
		for _, cat := range config.Categories {
			for i := 0; i < placesPerCell; i++ {
				id++
				base := categoryBase[cat]
				bias := taBias[n.Name]
				noise := r.NormFloat64() * 0.25
				rating := clamp(base+bias+noise, 1.0, 5.0)
				reviewCount := int(math.Max(3, r.NormFloat64()*200+200))

				ds.Ratings = append(ds.Ratings, sources.VenueRating{
					Source:       "tripadvisor",
					ExternalID:   fmt.Sprintf("ta-dryrun-%d", id),
					Name:         fmt.Sprintf("%s %s #%d", n.Name, cat, i+1),
					Neighborhood: n.Name,
					Category:     cat,
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
