// Package config holds the Amsterdam neighborhood and category definitions
// used to drive the Places API sweep. Coordinates are approximate
// neighborhood centroids; adjust radius per area density.
package config

// Neighborhood is a search anchor for the Places API Nearby Search.
type Neighborhood struct {
	Name      string
	Lat       float64
	Lng       float64
	RadiusM   int // search radius in meters
}

// Neighborhoods covers the main Amsterdam stadsdelen / well-known areas.
// Extend or split further (e.g. De Wallen vs. Jordaan separately) as needed.
var Neighborhoods = []Neighborhood{
	{"Centrum", 52.3730, 4.8924, 1200},
	{"Jordaan", 52.3745, 4.8815, 900},
	{"De Pijp", 52.3556, 4.8940, 900},
	{"Oud-West", 52.3651, 4.8686, 1000},
	{"Oost", 52.3600, 4.9330, 1500},
	{"Noord", 52.3960, 4.9200, 2000},
	{"West", 52.3860, 4.8600, 1500},
	{"Zuid", 52.3400, 4.8730, 1500},
	{"Zuidoost", 52.3140, 4.9490, 2000},
	{"Nieuw-West", 52.3650, 4.8100, 2000},
}

// Categories map to Google Places "type" values for Nearby Search.
// https://developers.google.com/maps/documentation/places/web-service/supported_types
var Categories = []string{
	"restaurant",
	"cafe",
	"bar",
	"museum",
	"tourist_attraction",
}

// YelpCategoryAlias maps each Google category above to the closest Yelp
// Fusion category alias (https://docs.developer.yelp.com/docs/resources-categories).
// These are approximate, not 1:1 — e.g. Yelp has no exact equivalent of
// Google's broad "tourist_attraction" type, so "landmarks" is the closest
// fit. Cross-platform comparisons for that category should be read as
// noisier than the others.
var YelpCategoryAlias = map[string]string{
	"restaurant":         "restaurants",
	"cafe":               "cafes",
	"bar":                "bars",
	"museum":             "museums",
	"tourist_attraction": "landmarks",
}
