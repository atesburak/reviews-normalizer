package match

import (
	"testing"

	"amsterdam-ratings/internal/sources"
)

func TestMatchFindsCloseSimilarlyNamedVenue(t *testing.T) {
	primary := []sources.VenueRating{
		{Source: "google", Name: "Cafe De Jordaan", Neighborhood: "Jordaan", Lat: 52.3745, Lng: 4.8815, Rating: 4.6},
	}
	others := map[string][]sources.VenueRating{
		"yelp": {
			{Source: "yelp", Name: "Café de Jordaan", Neighborhood: "Jordaan", Lat: 52.37455, Lng: 4.88152, Rating: 4.1},
		},
	}

	got := Match(primary, others, 100, 0.3)
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	yelpRating, ok := got[0].Ratings["yelp"]
	if !ok {
		t.Fatalf("expected a yelp match, got none: %+v", got[0])
	}
	if yelpRating != 4.1 {
		t.Errorf("expected matched yelp rating 4.1, got %v", yelpRating)
	}
}

func TestMatchRejectsFarAwayVenue(t *testing.T) {
	primary := []sources.VenueRating{
		{Source: "google", Name: "Cafe De Jordaan", Neighborhood: "Jordaan", Lat: 52.3745, Lng: 4.8815, Rating: 4.6},
	}
	others := map[string][]sources.VenueRating{
		"yelp": {
			// Same name, but ~2km away -> should not match at a 100m threshold.
			{Source: "yelp", Name: "Cafe De Jordaan", Neighborhood: "Jordaan", Lat: 52.39, Lng: 4.90, Rating: 3.0},
		},
	}

	got := Match(primary, others, 100, 0.3)
	if _, ok := got[0].Ratings["yelp"]; ok {
		t.Errorf("expected no yelp match for a venue 2km away, but got one")
	}
}

func TestMatchRejectsDissimilarName(t *testing.T) {
	primary := []sources.VenueRating{
		{Source: "google", Name: "Cafe De Jordaan", Neighborhood: "Jordaan", Lat: 52.3745, Lng: 4.8815, Rating: 4.6},
	}
	others := map[string][]sources.VenueRating{
		"yelp": {
			// Right next door, but a totally different name -> a different venue.
			{Source: "yelp", Name: "Bloemenwinkel Amsterdam", Neighborhood: "Jordaan", Lat: 52.37451, Lng: 4.88151, Rating: 4.9},
		},
	}

	got := Match(primary, others, 100, 0.3)
	if _, ok := got[0].Ratings["yelp"]; ok {
		t.Errorf("expected no yelp match for a dissimilarly named venue, but got one")
	}
}
