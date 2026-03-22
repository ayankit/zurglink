package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Client struct {
	Token string
}

func NewClient(token string) *Client {
	return &Client{Token: token}
}

type tmdbResponse struct {
	Results []struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`           // TV Name
		Title        string `json:"title"`          // Movie Title
		FirstAirDate string `json:"first_air_date"` // TV Date
		ReleaseDate  string `json:"release_date"`   // Movie Date
	} `json:"results"`
}

type MediaInfo struct {
	ID    int
	Title string
	Year  string
}

// Search searches for a TV show or Movie and returns the top result
func (c *Client) Search(title string, year int, isMovie bool) (*MediaInfo, error) {
	baseURL := "https://api.themoviedb.org/3/search/movie"
	if !isMovie {
		baseURL = "https://api.themoviedb.org/3/search/tv"
	}

	reqURL, _ := url.Parse(baseURL)
	q := reqURL.Query()
	q.Add("query", title)
	if year > 0 {
		if isMovie {
			q.Add("year", fmt.Sprintf("%d", year))
		} else {
			q.Add("first_air_date_year", fmt.Sprintf("%d", year))
		}
	}
	reqURL.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", reqURL.String(), nil)
	req.Header.Add("Authorization", "Bearer "+c.Token)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result tmdbResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no tmdb results found for %s", title)
	}

	first := result.Results[0]

	// Handle differences between TV and Movie JSON responses
	officialTitle := first.Name
	rawDate := first.FirstAirDate
	if isMovie {
		officialTitle = first.Title
		rawDate = first.ReleaseDate
	}

	// Extract Year from YYYY-MM-DD
	officialYear := ""
	if len(rawDate) >= 4 {
		officialYear = rawDate[:4]
	}

	return &MediaInfo{
		ID:    first.ID,
		Title: officialTitle,
		Year:  officialYear,
	}, nil
}
