package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type Client struct {
	Token      string
	HTTPClient *http.Client
	cache      map[string]*MediaInfo
	mu         sync.Mutex
}

func NewClient(token string) *Client {
	return &Client{
		Token:      token,
		HTTPClient: http.DefaultClient,
		cache:      make(map[string]*MediaInfo),
	}
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

func sanitizeTitle(name string) string {
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", " -")
	name = strings.ReplaceAll(name, "?", "")
	name = strings.ReplaceAll(name, "<", "")
	name = strings.ReplaceAll(name, ">", "")
	name = strings.ReplaceAll(name, "|", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "*", "")
	return strings.TrimSpace(name)
}

// Search searches for a TV show or Movie and returns the top result
func (c *Client) Search(title string, year int, isMovie bool) (*MediaInfo, error) {
	cacheKey := fmt.Sprintf("%s|%d|%t", title, year, isMovie)

	// Check cache
	c.mu.Lock()
	if val, ok := c.cache[cacheKey]; ok {
		c.mu.Unlock()
		if val == nil {
			return nil, fmt.Errorf("cached: no tmdb results found for %s", title)
		}
		return val, nil
	}
	c.mu.Unlock()

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

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb api error: %d", resp.StatusCode)
	}

	var result tmdbResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		c.mu.Lock()
		c.cache[cacheKey] = nil
		c.mu.Unlock()
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

	info := &MediaInfo{
		ID:    first.ID,
		Title: sanitizeTitle(officialTitle),
		Year:  officialYear,
	}

	// Save to cache
	c.mu.Lock()
	c.cache[cacheKey] = info
	c.mu.Unlock()

	return info, nil
}
