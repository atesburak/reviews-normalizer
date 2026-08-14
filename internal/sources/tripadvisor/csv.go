// Package tripadvisor loads TripAdvisor ratings for cross-platform
// comparison against the Google Places pipeline.
//
// Unlike Google Places and Yelp Fusion, TripAdvisor's Content API is
// partner-gated: as of 2026 there is no self-serve API key for new
// developers, only an approved-partner program. There is therefore no
// live-fetch client here (nothing to build against). Instead this package
// loads a CSV you've compiled by hand, exported from an approved partner
// account, or otherwise licensed — see LoadCSV for the expected format.
package tripadvisor

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"amsterdam-ratings/internal/sources"
)

// LoadCSV reads a TripAdvisor ratings export with header row:
//
//	name,neighborhood,category,lat,lng,rating,review_count,external_id
//
// Column order doesn't matter (matched by header name); extra columns are
// ignored. This is intentionally schema-first rather than API-first, since
// there's no live TripAdvisor API most developers can call — populate the
// CSV from whatever access you do have (manual lookup, an approved partner
// export, a licensed dataset).
func LoadCSV(path string) (sources.Dataset, error) {
	var ds sources.Dataset
	ds.Source = "tripadvisor"
	ds.GeneratedBy = "live"

	f, err := os.Open(path)
	if err != nil {
		return ds, fmt.Errorf("open tripadvisor csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return ds, fmt.Errorf("read tripadvisor csv header: %w", err)
	}
	col := make(map[string]int, len(header))
	for i, h := range header {
		col[h] = i
	}
	required := []string{"name", "neighborhood", "category", "lat", "lng", "rating", "review_count", "external_id"}
	for _, c := range required {
		if _, ok := col[c]; !ok {
			return ds, fmt.Errorf("tripadvisor csv missing required column %q", c)
		}
	}

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ds, fmt.Errorf("read tripadvisor csv row: %w", err)
		}

		lat, err := strconv.ParseFloat(row[col["lat"]], 64)
		if err != nil {
			return ds, fmt.Errorf("parse lat %q: %w", row[col["lat"]], err)
		}
		lng, err := strconv.ParseFloat(row[col["lng"]], 64)
		if err != nil {
			return ds, fmt.Errorf("parse lng %q: %w", row[col["lng"]], err)
		}
		rating, err := strconv.ParseFloat(row[col["rating"]], 64)
		if err != nil {
			return ds, fmt.Errorf("parse rating %q: %w", row[col["rating"]], err)
		}
		reviewCount, err := strconv.Atoi(row[col["review_count"]])
		if err != nil {
			return ds, fmt.Errorf("parse review_count %q: %w", row[col["review_count"]], err)
		}

		ds.Ratings = append(ds.Ratings, sources.VenueRating{
			Source:       "tripadvisor",
			ExternalID:   row[col["external_id"]],
			Name:         row[col["name"]],
			Neighborhood: row[col["neighborhood"]],
			Category:     row[col["category"]],
			Lat:          lat,
			Lng:          lng,
			Rating:       rating,
			ReviewCount:  reviewCount,
		})
	}

	return ds, nil
}
