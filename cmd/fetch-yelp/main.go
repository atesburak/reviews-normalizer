// Command fetch-yelp sweeps the same Amsterdam neighborhoods x categories as
// cmd/fetch, but against the Yelp Fusion API, writing a sources.Dataset JSON
// for cmd/compare to diff against the Google dataset.
//
// Real run (needs a free Yelp Fusion API key: https://docs.developer.yelp.com/docs/fusion-authentication):
//
//	export YELP_API_KEY=...
//	go run ./cmd/fetch-yelp -out data/yelp.json -limit 20
//
// No-network smoke test:
//
//	go run ./cmd/fetch-yelp -dry-run -out data/yelp.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"amsterdam-ratings/internal/config"
	"amsterdam-ratings/internal/sources"
	"amsterdam-ratings/internal/sources/yelp"
)

func main() {
	out := flag.String("out", "data/yelp.json", "output JSON path")
	limit := flag.Int("limit", 20, "max businesses per neighborhood x category cell")
	dryRun := flag.Bool("dry-run", false, "generate synthetic data instead of calling the live API")
	seed := flag.Int64("seed", 42, "random seed for -dry-run")
	flag.Parse()

	if *dryRun {
		ds := yelp.GenerateSynthetic(*seed, *limit)
		writeDataset(*out, ds)
		fmt.Printf("wrote %d synthetic yelp ratings to %s\n", len(ds.Ratings), *out)
		return
	}

	apiKey := os.Getenv("YELP_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "YELP_API_KEY is not set (or pass -dry-run to test without the API)")
		os.Exit(1)
	}
	client := yelp.NewClient(apiKey)

	var ds sources.Dataset
	ds.Source = "yelp"
	ds.GeneratedBy = "live"

	for _, n := range config.Neighborhoods {
		for _, cat := range config.Categories {
			alias, ok := config.YelpCategoryAlias[cat]
			if !ok {
				fmt.Fprintf(os.Stderr, "no Yelp category alias for %q, skipping\n", cat)
				continue
			}
			found, err := client.Search(n.Lat, n.Lng, n.RadiusM, alias, *limit)
			if err != nil {
				fmt.Fprintf(os.Stderr, "yelp search failed for %s/%s: %v\n", n.Name, alias, err)
				continue
			}
			for i := range found {
				found[i].Neighborhood = n.Name
			}
			fmt.Printf("%-12s %-20s -> %d businesses\n", n.Name, alias, len(found))
			ds.Ratings = append(ds.Ratings, found...)
		}
	}

	writeDataset(*out, ds)
	fmt.Printf("wrote %d yelp ratings to %s\n", len(ds.Ratings), *out)
}

func writeDataset(path string, ds sources.Dataset) {
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
