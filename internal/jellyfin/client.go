package jellyfin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
)

// Client is a client for the Jellyfin TargetedScans API.
type Client struct {
	token      string
	baseURL    string
	mountPath  string
	httpClient *http.Client
}

// NewClient creates a new Jellyfin client.
// Returns nil if either server or token is empty, making the service optional.
func NewClient(server, token, mountPath string) *Client {
	if server == "" || token == "" {
		return nil
	}
	return &Client{
		token:      token,
		baseURL:    server,
		mountPath:  mountPath,
		httpClient: &http.Client{},
	}
}

type ScanPathRequest struct {
	Path string `json:"Path"`
}

type ScanPathResponse struct {
	ItemId   string `json:"ItemId"`
	ItemName string `json:"ItemName"`
	Status   string `json:"Status"`
	Message  string `json:"Message"`
}

type ScanPathsRequest struct {
	Paths []string `json:"Paths"`
}

type ScanPathsResponse struct {
	Results []ScanPathResponse `json:"Results"`
}

// ScanPath scans a single path. It is a no-op if the client is nil.
func (c *Client) ScanPath(path string) (*ScanPathResponse, error) {
	if c == nil {
		return nil, nil // No-op if not configured
	}

	reqBody := ScanPathRequest{Path: filepath.Join(c.mountPath, path)}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint, err := url.JoinPath(c.baseURL, "Library/ScanPath")
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf(`MediaBrowser Token="%s"`, c.token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var scanResp ScanPathResponse
	if err := json.NewDecoder(resp.Body).Decode(&scanResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &scanResp, nil
}

// ScanPaths scans multiple paths in a single batch. It is a no-op if the client is nil.
func (c *Client) ScanPaths(paths []string) (*ScanPathsResponse, error) {
	if c == nil {
		return nil, nil // No-op if not configured
	}

	for i, p := range paths {
		paths[i] = filepath.Join(c.mountPath, p)
	}

	reqBody := ScanPathsRequest{Paths: paths}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint, err := url.JoinPath(c.baseURL, "Library/ScanPaths")
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf(`MediaBrowser Token="%s"`, c.token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var scanResp ScanPathsResponse
	if err := json.NewDecoder(resp.Body).Decode(&scanResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &scanResp, nil
}
