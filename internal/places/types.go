package places

// Review is a single user review as returned by the Place Details API
// (Google only returns up to ~5 "most relevant" reviews per place per call).
type Review struct {
	Text          string `json:"text"`
	Rating        int    `json:"rating"`
	RelativeTime  string `json:"relative_time_description,omitempty"`
	Language      string `json:"language,omitempty"`
	OriginalLang  string `json:"original_language,omitempty"`
}

// Place is one business/venue with the fields we need for normalization.
type Place struct {
	PlaceID          string   `json:"place_id"`
	Name             string   `json:"name"`
	Neighborhood     string   `json:"neighborhood"`
	Category         string   `json:"category"` // the Places "type" we searched under
	Lat              float64  `json:"lat"`
	Lng              float64  `json:"lng"`
	Rating           float64  `json:"rating"`
	UserRatingsTotal int      `json:"user_ratings_total"`
	Reviews          []Review `json:"reviews,omitempty"`
}

// Dataset is the on-disk JSON shape written by cmd/fetch and read by cmd/analyze.
type Dataset struct {
	GeneratedBy string  `json:"generated_by"` // "live" or "dry-run"
	Places      []Place `json:"places"`
}
