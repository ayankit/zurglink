package tmdb

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// RoundTripFunc allows mocking the http.Client
type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Normal Title", "Normal Title"},
		{"Attack on Titan: Final Season", "Attack on Titan - Final Season"},
		{"Fate/Stay Night", "Fate-Stay Night"},
		{"CSI: Miami?", "CSI - Miami"},
		{"<Invalid>|Title*", "InvalidTitle"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := sanitizeTitle(test.input)
			if result != test.expected {
				t.Errorf("expected %q, got %q", test.expected, result)
			}
		})
	}
}

func TestTMDB_Search_Movie(t *testing.T) {
	mockResponse := `{
		"results": [
			{
				"id": 12345,
				"title": "The Matrix: Reloaded",
				"release_date": "2003-05-15"
			}
		]
	}`

	httpClient := NewTestClient(func(req *http.Request) *http.Response {
		if !strings.Contains(req.URL.Path, "/search/movie") {
			t.Errorf("Expected movie search, got %s", req.URL.Path)
		}
		if req.URL.Query().Get("query") != "The Matrix" {
			t.Errorf("Expected query 'The Matrix', got %s", req.URL.Query().Get("query"))
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			Header:     make(http.Header),
		}
	})

	client := NewClient("dummy_token")
	client.HTTPClient = httpClient

	info, err := client.Search("The Matrix", 2003, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if info.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", info.ID)
	}
	if info.Title != "The Matrix - Reloaded" { // Note the sanitization
		t.Errorf("Expected title 'The Matrix - Reloaded', got %q", info.Title)
	}
	if info.Year != "2003" {
		t.Errorf("Expected year '2003', got %q", info.Year)
	}

	// Test caching
	httpClientCacheMiss := NewTestClient(func(req *http.Request) *http.Response {
		t.Fatal("Expected cache hit, but HTTP request was made")
		return nil
	})
	client.HTTPClient = httpClientCacheMiss
	cachedInfo, err := client.Search("The Matrix", 2003, true)
	if err != nil {
		t.Fatalf("Unexpected error from cache: %v", err)
	}
	if cachedInfo.ID != 12345 {
		t.Errorf("Cache mismatch, expected ID 12345, got %d", cachedInfo.ID)
	}
}

func TestTMDB_Search_TV(t *testing.T) {
	mockResponse := `{
		"results": [
			{
				"id": 67890,
				"name": "Breaking Bad",
				"first_air_date": "2008-01-20"
			}
		]
	}`

	httpClient := NewTestClient(func(req *http.Request) *http.Response {
		if !strings.Contains(req.URL.Path, "/search/tv") {
			t.Errorf("Expected tv search, got %s", req.URL.Path)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			Header:     make(http.Header),
		}
	})

	client := NewClient("dummy_token")
	client.HTTPClient = httpClient

	info, err := client.Search("Breaking Bad", 2008, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if info.Title != "Breaking Bad" {
		t.Errorf("Expected title 'Breaking Bad', got %q", info.Title)
	}
	if info.Year != "2008" {
		t.Errorf("Expected year '2008', got %q", info.Year)
	}
}

func TestTMDB_Search_NoResults(t *testing.T) {
	mockResponse := `{"results": []}`

	httpClient := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			Header:     make(http.Header),
		}
	})

	client := NewClient("dummy_token")
	client.HTTPClient = httpClient

	_, err := client.Search("Unknown Show XYZ", 0, false)
	if err == nil {
		t.Fatalf("Expected error for no results, got nil")
	}
	if !strings.Contains(err.Error(), "no tmdb results found") {
		t.Errorf("Unexpected error message: %v", err)
	}

	// Test negative caching
	httpClientCacheMiss := NewTestClient(func(req *http.Request) *http.Response {
		t.Fatal("Expected negative cache hit, but HTTP request was made")
		return nil
	})
	client.HTTPClient = httpClientCacheMiss

	_, errCache := client.Search("Unknown Show XYZ", 0, false)
	if errCache == nil {
		t.Fatalf("Expected error from negative cache, got nil")
	}
	if !strings.Contains(errCache.Error(), "cached: no tmdb results found") {
		t.Errorf("Unexpected cache error message: %v", errCache)
	}
}
