package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ayankit/clog"
	"github.com/ayankit/zurglink/internal/config"
	"github.com/ayankit/zurglink/internal/jellyfin"
	"github.com/ayankit/zurglink/internal/organise"
	"github.com/ayankit/zurglink/internal/tmdb"
)

// RequestBody expects JSON with path property
type RequestBody struct {
	Path string `json:"path"`
}

func main() {
	clog.Init(clog.LevelDebug)
	cfg, err := config.Load()
	if err != nil {
		clog.Fatal("Configuration load failed", clog.Err(err))
	}

	// If jellyfin mount point is not provided, use dest path by default
	mountPath := cfg.JFMountPath
	if mountPath == "" {
		mountPath = cfg.DestPath
	}

	tmdbClient := tmdb.NewClient(cfg.TMDBToken)
	jellyfinClient := jellyfin.NewClient(cfg.JFServer, cfg.JFToken, mountPath)
	manager, err := organise.NewManager(cfg.SourcePath, cfg.DestPath, tmdbClient, jellyfinClient)
	if err != nil {
		clog.Fatal("Manager init failed", clog.Err(err))
	}

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RequestBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request body", http.StatusBadRequest)
			return
		}

		if req.Path == "" {
			http.Error(w, "path is required", http.StatusBadRequest)
			return
		}

		// Process asynchronously so we don't block the webhook caller
		go processWithRetry(req.Path, manager, 3)

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Processing started"))
	})

	clog.Info("Starting ZurgLink...", "port", cfg.Port)
	clog.Info("Path config:", "source", cfg.SourcePath, "destination", cfg.DestPath)
	clog.Info("Jellyfin config:", "server", cfg.JFServer, "mountpath", cfg.JFMountPath)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		clog.Fatal("Server failed to start", clog.Err(err))
	}
}

// processWithRetry attempts to process a path with exponential backoff on failure.
func processWithRetry(path string, manager *organise.Manager, maxRetries int) {
	var err error
	delay := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = manager.ProcessPath(path)
		if err == nil {
			// Success
			return
		}

		if attempt < maxRetries {
			clog.Warn("Process failed, retrying...", "path", path, "attempt", attempt, "next_delay", delay.String(), clog.Err(err))
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}
	}

	// All retries exhausted
	clog.Error("Failed to process path after retries", "path", path, "retries", maxRetries, clog.Err(err))
}
