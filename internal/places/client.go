package places

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	nearbySearchURL = "https://maps.googleapis.com/maps/api/place/nearbysearch/json"
	placeDetailsURL = "https://maps.googleapis.com/maps/api/place/details/json"
)

// Client wraps the Google Places Web Service. Requires billing enabled on
// the associated Google Cloud project.
type Client struct {
	APIKey string
	HTTP   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
		HTTP:   &http.Client{Timeout: 15 * time.Second},
	}
}

type nearbySearchResponse struct {
	Results []struct {
		PlaceID string `json:"place_id"`
		Name    string `json:"name"`
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
		Rating           float64 `json:"rating"`
		UserRatingsTotal int     `json:"user_ratings_total"`
	} `json:"results"`
	NextPageToken string `json:"next_page_token"`
	Status        string `json:"status"`
	ErrorMessage  string `json:"error_message"`
}

// NearbySearch fetches up to 3 pages (60 results) for one lat/lng/radius/type
// combination. Google requires a short delay before a next_page_token becomes
// valid, which this handles internally.
func (c *Client) NearbySearch(lat, lng float64, radiusM int, placeType string, maxResults int) ([]Place, error) {
	var out []Place
	pageToken := ""

	for {
		q := url.Values{}
		q.Set("key", c.APIKey)
		q.Set("location", fmt.Sprintf("%f,%f", lat, lng))
		q.Set("radius", fmt.Sprintf("%d", radiusM))
		q.Set("type", placeType)
		if pageToken != "" {
			q.Set("pagetoken", pageToken)
		}

		resp, err := c.HTTP.Get(nearbySearchURL + "?" + q.Encode())
		if err != nil {
			return out, fmt.Errorf("nearby search request: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return out, fmt.Errorf("nearby search read body: %w", err)
		}

		var parsed nearbySearchResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return out, fmt.Errorf("nearby search unmarshal: %w", err)
		}
		if parsed.Status != "OK" && parsed.Status != "ZERO_RESULTS" {
			return out, fmt.Errorf("nearby search status %s: %s", parsed.Status, parsed.ErrorMessage)
		}

		for _, r := range parsed.Results {
			out = append(out, Place{
				PlaceID:          r.PlaceID,
				Name:             r.Name,
				Category:         placeType,
				Lat:              r.Geometry.Location.Lat,
				Lng:              r.Geometry.Location.Lng,
				Rating:           r.Rating,
				UserRatingsTotal: r.UserRatingsTotal,
			})
			if len(out) >= maxResults {
				return out, nil
			}
		}

		if parsed.NextPageToken == "" {
			return out, nil
		}
		pageToken = parsed.NextPageToken
		// Google's next_page_token has a short activation delay.
		time.Sleep(2 * time.Second)
	}
}

type placeDetailsResponse struct {
	Result struct {
		Reviews []struct {
			Text                    string `json:"text"`
			Rating                  int    `json:"rating"`
			RelativeTimeDescription string `json:"relative_time_description"`
			Language                string `json:"language"`
			OriginalLanguage        string `json:"original_language"`
		} `json:"reviews"`
	} `json:"result"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
}

// PlaceDetails fetches the (up to 5) "most relevant" reviews for a place.
// This is a separate billed call per place — use sparingly / sample.
func (c *Client) PlaceDetails(placeID string) ([]Review, error) {
	q := url.Values{}
	q.Set("key", c.APIKey)
	q.Set("place_id", placeID)
	q.Set("fields", "review")

	resp, err := c.HTTP.Get(placeDetailsURL + "?" + q.Encode())
	if err != nil {
		return nil, fmt.Errorf("place details request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("place details read body: %w", err)
	}

	var parsed placeDetailsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("place details unmarshal: %w", err)
	}
	if parsed.Status != "OK" {
		return nil, fmt.Errorf("place details status %s: %s", parsed.Status, parsed.ErrorMessage)
	}

	var reviews []Review
	for _, r := range parsed.Result.Reviews {
		reviews = append(reviews, Review{
			Text:         r.Text,
			Rating:       r.Rating,
			RelativeTime: r.RelativeTimeDescription,
			Language:     r.Language,
			OriginalLang: r.OriginalLanguage,
		})
	}
	return reviews, nil
}
