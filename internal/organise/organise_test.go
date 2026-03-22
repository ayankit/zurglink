package organise

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayankit/zurg-syms/internal/tmdb"
)

// mockTMDBResponse helps create a fake TMDB response for testing
func mockTMDBResponse(mockJSON string) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(mockJSON)),
				Header:     make(http.Header),
			}
		}),
	}
}

// RoundTripFunc allows mocking the http.Client
type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestProcessPath_SingleMovie(t *testing.T) {
	// Setup test directories
	testDir := t.TempDir()
	sourceDir := filepath.Join(testDir, "source")
	destDir := filepath.Join(testDir, "dest")
	os.MkdirAll(sourceDir, 0755)

	// Create a dummy video file
	movieFileName := "Inception.1999.1080p.mkv"
	moviePath := filepath.Join(sourceDir, movieFileName)
	os.WriteFile(moviePath, []byte("fake video content"), 0644)

	// Mock TMDB
	mockJSON := `{
		"results": [
			{
				"id": 27205,
				"title": "Inception",
				"release_date": "1999-07-16"
			}
		]
	}`
	tmdbClient := tmdb.NewClient("dummy")
	tmdbClient.HTTPClient = mockTMDBResponse(mockJSON)

	// Run process
	err := ProcessPath(sourceDir, destDir, movieFileName, tmdbClient)
	if err != nil {
		t.Fatalf("ProcessPath failed: %v", err)
	}

	// Verify Symlink
	expectedSymlink := filepath.Join(destDir, "Movies", "Inception (1999) [tmdbid-27205]", "Inception (1999) - [1080p].mkv")
	info, err := os.Lstat(expectedSymlink)
	if err != nil {
		t.Fatalf("Expected symlink not found at %s: %v", expectedSymlink, err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected %s to be a symlink, got mode %v", expectedSymlink, info.Mode())
	}
}

func TestProcessPath_TVShow(t *testing.T) {
	// Setup test directories
	testDir := t.TempDir()
	sourceDir := filepath.Join(testDir, "source")
	destDir := filepath.Join(testDir, "dest")
	os.MkdirAll(sourceDir, 0755)

	// Create a dummy video file
	tvFileName := "Breaking.Bad.S01E01.720p.HDTV.mkv"
	tvPath := filepath.Join(sourceDir, tvFileName)
	os.WriteFile(tvPath, []byte("fake video content"), 0644)

	// Mock TMDB
	mockJSON := `{
		"results": [
			{
				"id": 1396,
				"name": "Breaking Bad",
				"first_air_date": "2008-01-20"
			}
		]
	}`
	tmdbClient := tmdb.NewClient("dummy")
	tmdbClient.HTTPClient = mockTMDBResponse(mockJSON)

	// Run process
	err := ProcessPath(sourceDir, destDir, tvFileName, tmdbClient)
	if err != nil {
		t.Fatalf("ProcessPath failed: %v", err)
	}

	// Verify Symlink
	expectedSymlink := filepath.Join(destDir, "Shows", "Breaking Bad (2008) [tmdbid-1396]", "Season 01", "Breaking Bad (2008) S01E01 - [720p].mkv")
	info, err := os.Lstat(expectedSymlink)
	if err != nil {
		t.Fatalf("Expected symlink not found at %s: %v", expectedSymlink, err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected %s to be a symlink", expectedSymlink)
	}
}

func TestProcessPath_Directory(t *testing.T) {
	// Setup test directories
	testDir := t.TempDir()
	sourceDir := filepath.Join(testDir, "source")
	destDir := filepath.Join(testDir, "dest")
	showDir := filepath.Join(sourceDir, "MyShow")
	os.MkdirAll(showDir, 0755)

	// Create dummy video files and one non-video file
	os.WriteFile(filepath.Join(showDir, "MyShow.S02E01.mkv"), []byte("vid1"), 0644)
	os.WriteFile(filepath.Join(showDir, "MyShow.S02E02.mp4"), []byte("vid2"), 0644)
	os.WriteFile(filepath.Join(showDir, "info.txt"), []byte("text"), 0644) // Should be ignored

	// Mock TMDB
	mockJSON := `{
		"results": [
			{
				"id": 999,
				"name": "MyShow",
				"first_air_date": "2010-01-01"
			}
		]
	}`
	tmdbClient := tmdb.NewClient("dummy")
	tmdbClient.HTTPClient = mockTMDBResponse(mockJSON)

	// Run process on the *directory*
	err := ProcessPath(sourceDir, destDir, "MyShow", tmdbClient)
	if err != nil {
		t.Fatalf("ProcessPath failed: %v", err)
	}

	// Verify Symlinks
	ep1 := filepath.Join(destDir, "Shows", "MyShow (2010) [tmdbid-999]", "Season 02", "MyShow (2010) S02E01.mkv")
	ep2 := filepath.Join(destDir, "Shows", "MyShow (2010) [tmdbid-999]", "Season 02", "MyShow (2010) S02E02.mp4")

	if _, err := os.Lstat(ep1); err != nil {
		t.Errorf("Expected symlink missing: %s", ep1)
	}
	if _, err := os.Lstat(ep2); err != nil {
		t.Errorf("Expected symlink missing: %s", ep2)
	}

	// Make sure info.txt wasn't processed
	err = filepath.WalkDir(destDir, func(path string, d os.DirEntry, err error) error {
		if strings.HasSuffix(path, ".txt") {
			t.Errorf("Found text file in destination which should have been ignored: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
