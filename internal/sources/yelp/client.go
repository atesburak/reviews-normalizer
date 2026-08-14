// Package yelp implements a minimal client for the Yelp Fusion Business
// Search API, producing sources.VenueRating records for cross-platform
// comparison against the Google Places pipeline.
package yelp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"amsterdam-ratings/internal/sources"
)

const searchURL = "https://api.yelp.com/v3/businesses/search"

// pageLimit is Yelp Fusion's max results per page.
const pageLimit = 50

// Client wraps the Yelp Fusion API. Requires a Yelp API key (Fusion is
// free-tier but does require signup): https://docs.developer.yelp.com/docs/fusion-authentication
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

type searchResponse struct {
	Businesses []struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Rating      float64 `json:"rating"`
		ReviewCount int     `json:"review_count"`
		Coordinates struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"coordinates"`
	} `json:"businesses"`
	Total int `json:"total"`
}

// Search fetches up to maxResults businesses near lat/lng for a Yelp
// category alias (see internal/config.YelpCategoryAlias), paging through
// Yelp's 50-per-request limit.
func (c *Client) Search(lat, lng float64, radiusM int, categoryAlias string, maxResults int) ([]sources.VenueRating, error) {
	var out []sources.VenueRating
	offset := 0

	for len(out) < maxResults {
		limit := pageLimit
		if remaining := maxResults - len(out); remaining < limit {
			limit = remaining
		}

		q := url.Values{}
		q.Set("latitude", fmt.Sprintf("%f", lat))
		q.Set("longitude", fmt.Sprintf("%f", lng))
		q.Set("radius", fmt.Sprintf("%d", radiusM)) // Yelp caps this at 40000
		q.Set("categories", categoryAlias)
		q.Set("limit", fmt.Sprintf("%d", limit))
		q.Set("offset", fmt.Sprintf("%d", offset))

		req, err := http.NewRequest(http.MethodGet, searchURL+"?"+q.Encode(), nil)
		if err != nil {
			return out, fmt.Errorf("yelp search build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return out, fmt.Errorf("yelp search request: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return out, fmt.Errorf("yelp search read body: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return out, fmt.Errorf("yelp search status %d: %s", resp.StatusCode, string(body))
		}

		var parsed searchResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return out, fmt.Errorf("yelp search unmarshal: %w", err)
		}
		if len(parsed.Businesses) == 0 {
			break
		}

		for _, b := range parsed.Businesses {
			out = append(out, sources.VenueRating{
				Source:      "yelp",
				ExternalID:  b.ID,
				Name:        b.Name,
				Category:    categoryAlias,
				Lat:         b.Coordinates.Latitude,
				Lng:         b.Coordinates.Longitude,
				Rating:      b.Rating,
				ReviewCount: b.ReviewCount,
			})
			if len(out) >= maxResults {
				break
			}
		}

		offset += len(parsed.Businesses)
		if offset >= parsed.Total {
			break
		}
		time.Sleep(150 * time.Millisecond) // stay well under QPS limits
	}

	return out, nil
}
