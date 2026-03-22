package jellyfin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_ScanPath(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Library/ScanPath" {
			t.Errorf("expected path /Library/ScanPath, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != `MediaBrowser Token="test-token"` {
			t.Errorf("expected auth header MediaBrowser Token=\"test-token\", got %s", auth)
		}

		var reqBody ScanPathRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatal(err)
		}
		if reqBody.Path != "/mount/path/media/TV/Test/S01E01.mkv" {
			t.Errorf("expected path /mount/path/media/TV/Test/S01E01.mkv, got %s", reqBody.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ScanPathResponse{
			ItemId:   "abc1234",
			ItemName: "S01E01",
			Status:   "Created",
			Message:  "Item created",
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "/mount/path")
	resp, err := client.ScanPath("media/TV/Test/S01E01.mkv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ItemId != "abc1234" || resp.Status != "Created" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestClient_ScanPaths(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Library/ScanPaths" {
			t.Errorf("expected path /Library/ScanPaths, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		var reqBody ScanPathsRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatal(err)
		}
		if len(reqBody.Paths) != 2 {
			t.Errorf("expected 2 paths, got %d", len(reqBody.Paths))
		}
		if reqBody.Paths[0] != "/mount/path/media/TV/Test/S01E01.mkv" {
			t.Errorf("expected path /mount/path/media/TV/Test/S01E01.mkv, got %s", reqBody.Paths[0])
		}
		if reqBody.Paths[1] != "/mount/path/media/TV/Test/S01E02.mkv" {
			t.Errorf("expected path /mount/path/media/TV/Test/S01E02.mkv, got %s", reqBody.Paths[1])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ScanPathsResponse{
			Results: []ScanPathResponse{
				{
					ItemId:   "1",
					ItemName: "S01E01",
					Status:   "Created",
				},
				{
					ItemId:   "2",
					ItemName: "S01E02",
					Status:   "Created",
				},
			},
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-token", "/mount/path")
	resp, err := client.ScanPaths([]string{
		"media/TV/Test/S01E01.mkv",
		"media/TV/Test/S01E02.mkv",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Errorf("unexpected results length: %d", len(resp.Results))
	}
}

func TestClient_Optional(t *testing.T) {
	client := NewClient("", "", "")
	if client != nil {
		t.Errorf("expected nil client for empty config")
	}

	// Calling methods on nil client should not panic
	resp1, err1 := client.ScanPath("/test")
	if err1 != nil || resp1 != nil {
		t.Errorf("expected nil, nil for nil client ScanPath")
	}

	resp2, err2 := client.ScanPaths([]string{"/test"})
	if err2 != nil || resp2 != nil {
		t.Errorf("expected nil, nil for nil client ScanPaths")
	}
}
