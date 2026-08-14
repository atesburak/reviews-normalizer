// Command analyze reads a Dataset JSON (from cmd/fetch) and produces:
//
//  1. A neighborhood normalization matrix: how many stars to add/subtract
//     per neighborhood, holding category mix and review volume constant.
//  2. Review-effort diagnostics per neighborhood (avg review length, share
//     of no-text reviews) to sanity-check *why* a neighborhood runs hot/cold.
//
// Usage:
//
//	go run ./cmd/analyze -in data/places.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"amsterdam-ratings/internal/features"
	"amsterdam-ratings/internal/places"
	"amsterdam-ratings/internal/regress"
)

func main() {
	in := flag.String("in", "data/places.json", "input dataset JSON path (from cmd/fetch)")
	flag.Parse()

	f, err := os.Open(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	var ds places.Dataset
	if err := json.NewDecoder(f).Decode(&ds); err != nil {
		fmt.Fprintf(os.Stderr, "decode input: %v\n", err)
		os.Exit(1)
	}
	if len(ds.Places) == 0 {
		fmt.Fprintln(os.Stderr, "no places in dataset")
		os.Exit(1)
	}
	fmt.Printf("loaded %d places (%s)\n\n", len(ds.Places), ds.GeneratedBy)

	dm := features.BuildDesignMatrix(ds)
	beta, err := regress.Fit(dm.X, dm.Y)
	if err != nil {
		fmt.Fprintf(os.Stderr, "regression failed: %v\n", err)
		os.Exit(1)
	}
	yHat := regress.Predict(dm.X, beta)
	r2 := regress.RSquared(dm.Y, yHat)

	fmt.Println("=== Neighborhood normalization matrix ===")
	fmt.Printf("(stars relative to baseline neighborhood %q, holding category and log(review count) constant)\n\n", dm.BaseNeighb)

	type coefRow struct {
		Name string
		Coef float64
	}
	var neighCoefs []coefRow
	neighCoefs = append(neighCoefs, coefRow{dm.BaseNeighb, 0.0}) // baseline reference
	for i, name := range dm.ColNames {
		if len(name) > 13 && name[:13] == "neighborhood:" {
			neighCoefs = append(neighCoefs, coefRow{name[13:], beta[i]})
		}
	}
	sort.Slice(neighCoefs, func(i, j int) bool { return neighCoefs[i].Coef > neighCoefs[j].Coef })

	fmt.Printf("%-14s %10s   %s\n", "Neighborhood", "Δ stars", "Suggested correction (subtract Δ to normalize)")
	for _, c := range neighCoefs {
		fmt.Printf("%-14s %+10.3f   %+.3f\n", c.Name, c.Coef, -c.Coef)
	}

	fmt.Println("\n=== Category effects (reference:", dm.BaseCat, ") ===")
	for i, name := range dm.ColNames {
		if len(name) > 9 && name[:9] == "category:" {
			fmt.Printf("%-25s %+.3f\n", name[9:], beta[i])
		}
	}

	for i, name := range dm.ColNames {
		if name == "log_review_count" {
			fmt.Printf("\nlog(review_count+1) coefficient: %+.3f (stars per e-fold increase in review count)\n", beta[i])
		}
	}
	fmt.Printf("R^2: %.3f  (n=%d)\n", r2, len(dm.Y))

	fmt.Println("\n=== Review-effort diagnostics (from sampled review text) ===")
	stats := features.ComputeNeighborhoodStats(ds)
	if stats[0].NReviews == 0 {
		fmt.Println("(no review text in dataset — run fetch with -with-reviews to populate this section)")
		return
	}
	fmt.Printf("%-14s %8s %10s %14s %12s\n", "Neighborhood", "#places", "avg rating", "avg review len", "% no-text")
	for _, s := range stats {
		fmt.Printf("%-14s %8d %10.2f %14.0f %11.1f%%\n", s.Neighborhood, s.NPlaces, s.AvgRating, s.AvgReviewLenChar, s.ShareNoText*100)
	}
	fmt.Println("\nIf a neighborhood's Δ stars above is large AND positive AND it has short/no-text reviews,")
	fmt.Println("that's consistent with the low-effort-review inflation pattern — same mechanism, different axis, as the")
	fmt.Println("cross-platform rating inflation literature (Raval 2024).")
}
