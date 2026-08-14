// Package match cross-references the same physical venue across review
// platforms that use different IDs, name formatting, and coordinate
// precision for it. There's no shared key (Google place_id, Yelp business
// id, and a TripAdvisor CSV row have nothing in common), so matching is
// probabilistic: nearest by distance, filtered by name-token overlap.
package match

import (
	"math"
	"strings"
	"unicode"

	"amsterdam-ratings/internal/sources"
)

// Matched holds one physical venue's rating across whichever sources found
// a corresponding match, anchored on the primary source's identity (name,
// neighborhood, category). A venue with no match in a given secondary
// source simply has no entry for it in Ratings/ReviewCounts — callers use
// that to compute match coverage per neighborhood, not just deltas among
// the venues that did match.
type Matched struct {
	Neighborhood string
	Category     string
	PrimaryName  string
	Ratings      map[string]float64 // source -> rating
	ReviewCounts map[string]int     // source -> review_count
}

// stopwords are stripped before comparing name tokens: generic words
// (category labels, "amsterdam") that don't help distinguish one venue from
// another and would otherwise inflate the similarity score between two
// unrelated venues of the same category.
var stopwords = map[string]bool{
	"amsterdam":  true,
	"restaurant": true,
	"restaurants": true,
	"cafe":       true,
	"cafes":      true,
	"bar":        true,
	"bars":       true,
	"museum":     true,
	"museums":    true,
	"landmarks":  true,
	"the":        true,
	"de":         true,
	"het":        true,
	"van":        true,
}

// Match anchors on each entry in primary and, for every source in others,
// finds the best candidate by name-token Jaccard similarity among those
// within maxDistanceM, keeping it only if that similarity is >= minNameScore.
func Match(primary []sources.VenueRating, others map[string][]sources.VenueRating, maxDistanceM, minNameScore float64) []Matched {
	out := make([]Matched, 0, len(primary))

	for _, p := range primary {
		m := Matched{
			Neighborhood: p.Neighborhood,
			Category:     p.Category,
			PrimaryName:  p.Name,
			Ratings:      map[string]float64{p.Source: p.Rating},
			ReviewCounts: map[string]int{p.Source: p.ReviewCount},
		}
		pTokens := tokenize(p.Name)

		for srcName, candidates := range others {
			bestIdx := -1
			bestScore := -1.0
			for i, c := range candidates {
				if p.Neighborhood != "" && c.Neighborhood != "" && p.Neighborhood != c.Neighborhood {
					continue
				}
				if haversineMeters(p.Lat, p.Lng, c.Lat, c.Lng) > maxDistanceM {
					continue
				}
				score := jaccard(pTokens, tokenize(c.Name))
				if score < minNameScore {
					continue
				}
				if score > bestScore {
					bestScore = score
					bestIdx = i
				}
			}
			if bestIdx >= 0 {
				best := candidates[bestIdx]
				m.Ratings[srcName] = best.Rating
				m.ReviewCounts[srcName] = best.ReviewCount
			}
		}

		out = append(out, m)
	}

	return out
}

func tokenize(name string) map[string]bool {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	tokens := map[string]bool{}
	for _, w := range strings.Fields(b.String()) {
		if stopwords[w] {
			continue
		}
		tokens[w] = true
	}
	return tokens
}

func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * c
}
