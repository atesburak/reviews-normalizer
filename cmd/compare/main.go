// Command compare cross-references Google, Yelp, and (optionally)
// TripAdvisor ratings for the same physical venues and reports the average
// rating gap per Amsterdam neighborhood — i.e. how much each platform's
// rating diverges from Google's in each area, which is the cross-platform
// analogue of the within-Google neighborhood normalization matrix that
// cmd/analyze produces.
//
// No-network smoke test (synthesizes all three sources with a known,
// injected bias so you can confirm the matching + aggregation logic works):
//
//	go run ./cmd/compare -dry-run
//
// Real run, after populating data/places.json (cmd/fetch) and
// data/yelp.json (cmd/fetch-yelp), optionally plus a TripAdvisor CSV
// (see internal/sources/tripadvisor for the expected format):
//
//	go run ./cmd/compare -google data/places.json -yelp data/yelp.json -tripadvisor data/tripadvisor.csv
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"amsterdam-ratings/internal/match"
	"amsterdam-ratings/internal/places"
	"amsterdam-ratings/internal/sources"
	"amsterdam-ratings/internal/sources/tripadvisor"
	"amsterdam-ratings/internal/sources/yelp"
)

func main() {
	googlePath := flag.String("google", "data/places.json", "Google dataset JSON path (from cmd/fetch)")
	yelpPath := flag.String("yelp", "data/yelp.json", "Yelp dataset JSON path (from cmd/fetch-yelp)")
	tripadvisorPath := flag.String("tripadvisor", "", "TripAdvisor dataset path: .csv for a manual/partner export, .json for one from -dry-run (optional, comparison runs without it)")
	maxDistanceM := flag.Float64("max-distance-m", 120, "max distance between two listings to still consider them the same venue")
	minNameScore := flag.Float64("min-name-score", 0.3, "min name-token Jaccard similarity to still consider two listings the same venue")
	dryRun := flag.Bool("dry-run", false, "generate synthetic data for all sources instead of reading files")
	limit := flag.Int("limit", 15, "places per neighborhood x category cell (-dry-run only)")
	seed := flag.Int64("seed", 42, "random seed (-dry-run only)")
	flag.Parse()

	var googleRatings []sources.VenueRating
	others := map[string][]sources.VenueRating{}

	if *dryRun {
		googleRatings = fromGoogle(places.GenerateSynthetic(*seed, *limit))
		others["yelp"] = yelp.GenerateSynthetic(*seed, *limit).Ratings
		others["tripadvisor"] = tripadvisor.GenerateSynthetic(*seed, *limit).Ratings
	} else {
		googleDS, err := loadGoogleDataset(*googlePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load google dataset: %v\n", err)
			os.Exit(1)
		}
		googleRatings = fromGoogle(googleDS)

		yelpDS, err := loadSourcesDataset(*yelpPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load yelp dataset: %v\n", err)
			os.Exit(1)
		}
		others["yelp"] = yelpDS.Ratings

		if *tripadvisorPath != "" {
			taDS, err := loadTripAdvisor(*tripadvisorPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "load tripadvisor dataset: %v\n", err)
				os.Exit(1)
			}
			others["tripadvisor"] = taDS.Ratings
		}
	}

	if len(googleRatings) == 0 {
		fmt.Fprintln(os.Stderr, "no google places loaded")
		os.Exit(1)
	}

	matched := match.Match(googleRatings, others, *maxDistanceM, *minNameScore)
	report(matched, others)
}

func fromGoogle(ds places.Dataset) []sources.VenueRating {
	out := make([]sources.VenueRating, 0, len(ds.Places))
	for _, p := range ds.Places {
		out = append(out, sources.VenueRating{
			Source:       "google",
			ExternalID:   p.PlaceID,
			Name:         p.Name,
			Neighborhood: p.Neighborhood,
			Category:     p.Category,
			Lat:          p.Lat,
			Lng:          p.Lng,
			Rating:       p.Rating,
			ReviewCount:  p.UserRatingsTotal,
		})
	}
	return out
}

func loadGoogleDataset(path string) (places.Dataset, error) {
	var ds places.Dataset
	f, err := os.Open(path)
	if err != nil {
		return ds, err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&ds)
	return ds, err
}

func loadSourcesDataset(path string) (sources.Dataset, error) {
	var ds sources.Dataset
	f, err := os.Open(path)
	if err != nil {
		return ds, err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&ds)
	return ds, err
}

func loadTripAdvisor(path string) (sources.Dataset, error) {
	if strings.EqualFold(filepath.Ext(path), ".csv") {
		return tripadvisor.LoadCSV(path)
	}
	return loadSourcesDataset(path)
}

type neighStat struct {
	NGoogle  int
	NMatched map[string]int
	SumDelta map[string]float64 // sum of (google_rating - other_rating) over matched venues
}

func report(matched []match.Matched, others map[string][]sources.VenueRating) {
	otherNames := sortedSourceKeys(others)
	if len(otherNames) == 0 {
		fmt.Println("no secondary sources provided - nothing to compare Google against")
		return
	}

	byN := map[string]*neighStat{}
	for _, m := range matched {
		s, ok := byN[m.Neighborhood]
		if !ok {
			s = &neighStat{NMatched: map[string]int{}, SumDelta: map[string]float64{}}
			byN[m.Neighborhood] = s
		}
		s.NGoogle++
		gRating := m.Ratings["google"]
		for _, name := range otherNames {
			if r, ok := m.Ratings[name]; ok {
				s.NMatched[name]++
				s.SumDelta[name] += gRating - r
			}
		}
	}

	fmt.Println("=== Cross-platform rating gap per neighborhood (Google minus other platform, matched venues only) ===")
	for _, n := range sortedNeighKeys(byN) {
		s := byN[n]
		fmt.Printf("\n%s (n_google=%d)\n", n, s.NGoogle)
		for _, name := range otherNames {
			nm := s.NMatched[name]
			if nm == 0 {
				fmt.Printf("  vs %-12s no matches (0/%d) - can't compute a gap here\n", name, s.NGoogle)
				continue
			}
			avg := s.SumDelta[name] / float64(nm)
			coverage := float64(nm) / float64(s.NGoogle) * 100
			fmt.Printf("  vs %-12s %+.3f stars   (matched %d/%d google venues = %.0f%%)\n", name, avg, nm, s.NGoogle, coverage)
		}
	}

	fmt.Println("\n=== Overall (all neighborhoods pooled) ===")
	for _, name := range otherNames {
		var totalMatched int
		var totalDelta float64
		for _, s := range byN {
			totalMatched += s.NMatched[name]
			totalDelta += s.SumDelta[name]
		}
		if totalMatched == 0 {
			fmt.Printf("vs %-12s no matches at all\n", name)
			continue
		}
		fmt.Printf("vs %-12s %+.3f stars avg   (n=%d matched venues)\n", name, totalDelta/float64(totalMatched), totalMatched)
	}

	fmt.Println("\nNote: unmatched venues are ambiguous, not necessarily absent from the other platform -")
	fmt.Println("they may just differ enough in name/location to fall outside -max-distance-m / -min-name-score.")
	fmt.Println("Low match coverage in a neighborhood means treat its gap estimate as low-confidence.")
}

func sortedSourceKeys(m map[string][]sources.VenueRating) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedNeighKeys(m map[string]*neighStat) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
