// Command fetch sweeps Amsterdam neighborhoods x categories via the Google
// Places API and writes a Dataset JSON file for cmd/analyze to consume.
//
// Real run (needs a billed Google Cloud API key with Places API enabled):
//
//	export GOOGLE_MAPS_API_KEY=...
//	go run ./cmd/fetch -out data/places.json -with-reviews -limit 20
//
// No-network smoke test:
//
//	go run ./cmd/fetch -dry-run -out data/places.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"amsterdam-ratings/internal/config"
	"amsterdam-ratings/internal/places"
)

func main() {
	out := flag.String("out", "data/places.json", "output JSON path")
	limit := flag.Int("limit", 20, "max places per neighborhood x category cell")
	withReviews := flag.Bool("with-reviews", false, "also fetch Place Details reviews (extra billed calls, 1 per place)")
	dryRun := flag.Bool("dry-run", false, "generate synthetic data instead of calling the live API")
	seed := flag.Int64("seed", 42, "random seed for -dry-run")
	flag.Parse()

	if *dryRun {
		ds := places.GenerateSynthetic(*seed, *limit)
		writeDataset(*out, ds)
		fmt.Printf("wrote %d synthetic places to %s\n", len(ds.Places), *out)
		return
	}

	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "GOOGLE_MAPS_API_KEY is not set (or pass -dry-run to test without the API)")
		os.Exit(1)
	}
	client := places.NewClient(apiKey)

	var ds places.Dataset
	ds.GeneratedBy = "live"

	for _, n := range config.Neighborhoods {
		for _, cat := range config.Categories {
			found, err := client.NearbySearch(n.Lat, n.Lng, n.RadiusM, cat, *limit)
			if err != nil {
				fmt.Fprintf(os.Stderr, "nearby search failed for %s/%s: %v\n", n.Name, cat, err)
				continue
			}
			for i := range found {
				found[i].Neighborhood = n.Name
			}
			fmt.Printf("%-12s %-20s -> %d places\n", n.Name, cat, len(found))

			if *withReviews {
				for i := range found {
					reviews, err := client.PlaceDetails(found[i].PlaceID)
					if err != nil {
						fmt.Fprintf(os.Stderr, "  place details failed for %s: %v\n", found[i].Name, err)
						continue
					}
					found[i].Reviews = reviews
					time.Sleep(150 * time.Millisecond) // stay well under QPS limits
				}
			}
			ds.Places = append(ds.Places, found...)
		}
	}

	writeDataset(*out, ds)
	fmt.Printf("wrote %d places to %s\n", len(ds.Places), *out)
}

func writeDataset(path string, ds places.Dataset) {
	if dir := dirOf(path); dir != "" {
		os.MkdirAll(dir, 0o755)
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(ds); err != nil {
		fmt.Fprintf(os.Stderr, "encode output: %v\n", err)
		os.Exit(1)
	}
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return ""
}
