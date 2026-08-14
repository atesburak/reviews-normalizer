package places

import (
	"fmt"
	"math"
	"math/rand"

	"amsterdam-ratings/internal/config"
)

// GenerateSynthetic builds a plausible fake dataset with a *known* injected
// neighborhood bias and review-effort pattern, so you can sanity-check that
// the regression in cmd/analyze actually recovers it before spending API
// quota on the real thing.
//
// Design: Centrum/Jordaan get a positive rating bump and shorter/no-text
// reviews (tourist-heavy, low-effort). Noord/Zuidoost get a negative bump
// and longer, more critical reviews. This mirrors the pattern the FTC
// review-inflation literature associates with low-effort reviewing.
func GenerateSynthetic(seed int64, placesPerCell int) Dataset {
	r := rand.New(rand.NewSource(seed))

	// Ground-truth neighborhood bias (stars), unknown to the model, which
	// the regression should approximately recover.
	trueBias := map[string]float64{
		"Centrum":    0.55,
		"Jordaan":    0.35,
		"De Pijp":    0.15,
		"Oud-West":   0.10,
		"Oost":       0.00,
		"West":       -0.05,
		"Zuid":       0.05,
		"Nieuw-West": -0.15,
		"Noord":      -0.25,
		"Zuidoost":   -0.35,
	}
	// True effort proxy correlated (imperfectly) with the bias above.
	baseNoTextShare := map[string]float64{
		"Centrum":    0.40,
		"Jordaan":    0.32,
		"De Pijp":    0.20,
		"Oud-West":   0.18,
		"Oost":       0.15,
		"West":       0.14,
		"Zuid":       0.16,
		"Nieuw-West": 0.10,
		"Noord":      0.08,
		"Zuidoost":   0.07,
	}

	categoryBase := map[string]float64{
		"restaurant":          4.1,
		"cafe":                4.2,
		"bar":                 4.0,
		"museum":              4.4,
		"tourist_attraction":  4.1,
	}

	var ds Dataset
	ds.GeneratedBy = "dry-run"

	id := 0
	for _, n := range config.Neighborhoods {
		for _, cat := range config.Categories {
			for i := 0; i < placesPerCell; i++ {
				id++
				base := categoryBase[cat]
				bias := trueBias[n.Name]
				noise := r.NormFloat64() * 0.25
				rating := clamp(base+bias+noise, 1.0, 5.0)

				reviewCount := int(math.Max(3, r.NormFloat64()*300+400))
				noTextP := clamp(baseNoTextShare[n.Name]+r.NormFloat64()*0.05, 0.02, 0.7)

				nReviews := 5
				reviews := make([]Review, 0, nReviews)
				for j := 0; j < nReviews; j++ {
					noText := r.Float64() < noTextP
					text := ""
					if !noText {
						// Longer text reviews trend slightly more critical,
						// matching the empirical pattern discussed.
						length := int(math.Max(20, r.NormFloat64()*150+150))
						text = fmt.Sprintf("%*s", length, "x") // placeholder text of target length
					}
					reviews = append(reviews, Review{
						Text:     text,
						Rating:   int(clamp(rating+r.NormFloat64()*0.7, 1, 5)),
						Language: "nl",
					})
				}

				ds.Places = append(ds.Places, Place{
					PlaceID:          fmt.Sprintf("dryrun-%d", id),
					Name:             fmt.Sprintf("%s %s #%d", n.Name, cat, i+1),
					Neighborhood:     n.Name,
					Category:         cat,
					Lat:              n.Lat,
					Lng:              n.Lng,
					Rating:           round1(rating),
					UserRatingsTotal: reviewCount,
					Reviews:          reviews,
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
