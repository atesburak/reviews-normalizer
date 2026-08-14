// Package sources defines the source-agnostic rating shape used to compare
// the same physical venue across multiple review platforms (Google, Yelp,
// TripAdvisor, ...). Each platform-specific package (internal/sources/yelp,
// internal/sources/tripadvisor) produces a Dataset in this shape; the
// existing Google pipeline (internal/places) predates this package and is
// adapted into it at the point of use (cmd/compare) rather than rewritten,
// since it also feeds the unrelated within-Google OLS pipeline in cmd/analyze.
package sources

// VenueRating is one venue's rating on one platform.
type VenueRating struct {
	Source       string  `json:"source"` // "google", "yelp", "tripadvisor"
	ExternalID   string  `json:"external_id"`
	Name         string  `json:"name"`
	Neighborhood string  `json:"neighborhood"`
	Category     string  `json:"category"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	Rating       float64 `json:"rating"`
	ReviewCount  int     `json:"review_count"`
}

// Dataset is the on-disk JSON shape written by cmd/fetch-yelp and the
// TripAdvisor loader, and read by cmd/compare.
type Dataset struct {
	Source      string        `json:"source"`
	GeneratedBy string        `json:"generated_by"` // "live" or "dry-run"
	Ratings     []VenueRating `json:"ratings"`
}
