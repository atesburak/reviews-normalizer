// Package features turns a raw places.Dataset into (a) neighborhood-level
// review-effort diagnostics and (b) a regression-ready design matrix.
package features

import (
	"math"
	"sort"

	"amsterdam-ratings/internal/places"
)

// NeighborhoodStats summarizes review-effort signals per neighborhood —
// these are the "is this area's rating inflation driven by low-effort
// reviewing?" diagnostics discussed in the methodology.
type NeighborhoodStats struct {
	Neighborhood     string
	NPlaces          int
	NReviews         int
	AvgRating        float64
	AvgReviewLenChar float64
	ShareNoText      float64
}

func ComputeNeighborhoodStats(ds places.Dataset) []NeighborhoodStats {
	type acc struct {
		nPlaces, nReviews, noText int
		sumRating, sumLen         float64
	}
	byN := map[string]*acc{}

	for _, p := range ds.Places {
		a, ok := byN[p.Neighborhood]
		if !ok {
			a = &acc{}
			byN[p.Neighborhood] = a
		}
		a.nPlaces++
		a.sumRating += p.Rating
		for _, r := range p.Reviews {
			a.nReviews++
			l := len([]rune(r.Text))
			a.sumLen += float64(l)
			if l == 0 {
				a.noText++
			}
		}
	}

	var out []NeighborhoodStats
	for name, a := range byN {
		s := NeighborhoodStats{
			Neighborhood: name,
			NPlaces:      a.nPlaces,
			NReviews:     a.nReviews,
			AvgRating:    a.sumRating / float64(a.nPlaces),
		}
		if a.nReviews > 0 {
			s.AvgReviewLenChar = a.sumLen / float64(a.nReviews)
			s.ShareNoText = float64(a.noText) / float64(a.nReviews)
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AvgRating > out[j].AvgRating })
	return out
}

// DesignMatrix holds X, y and the human-readable name of every column in X,
// so regression coefficients can be reported meaningfully.
type DesignMatrix struct {
	X          [][]float64
	Y          []float64
	ColNames   []string
	BaseNeighb string // reference/dropped neighborhood dummy
	BaseCat    string // reference/dropped category dummy
}

// BuildDesignMatrix constructs:
//
//	rating ~ intercept + neighborhood dummies (baseline dropped) +
//	         category dummies (baseline dropped) + log(review_count+1)
//
// The first neighborhood/category encountered (alphabetically) is treated
// as the baseline/reference level, so all reported neighborhood
// coefficients are "stars relative to <baseline>, holding category and
// review volume constant" — i.e. the normalization matrix.
func BuildDesignMatrix(ds places.Dataset) DesignMatrix {
	neighSet := map[string]bool{}
	catSet := map[string]bool{}
	for _, p := range ds.Places {
		neighSet[p.Neighborhood] = true
		catSet[p.Category] = true
	}
	neighborhoods := sortedKeys(neighSet)
	categories := sortedKeys(catSet)

	baseN, baseC := neighborhoods[0], categories[0]

	colNames := []string{"intercept"}
	for _, n := range neighborhoods[1:] {
		colNames = append(colNames, "neighborhood:"+n)
	}
	for _, c := range categories[1:] {
		colNames = append(colNames, "category:"+c)
	}
	colNames = append(colNames, "log_review_count")

	var X [][]float64
	var Y []float64

	for _, p := range ds.Places {
		row := make([]float64, len(colNames))
		row[0] = 1 // intercept
		idx := 1
		for _, n := range neighborhoods[1:] {
			if p.Neighborhood == n {
				row[idx] = 1
			}
			idx++
		}
		for _, c := range categories[1:] {
			if p.Category == c {
				row[idx] = 1
			}
			idx++
		}
		row[len(colNames)-1] = math.Log(float64(p.UserRatingsTotal) + 1)

		X = append(X, row)
		Y = append(Y, p.Rating)
	}

	return DesignMatrix{X: X, Y: Y, ColNames: colNames, BaseNeighb: baseN, BaseCat: baseC}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
