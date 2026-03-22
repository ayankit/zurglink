package main

import (
	"encoding/json"
	"net/http"

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
		go func(path string) {
			err := manager.ProcessPath(path)
			if err != nil {
				clog.Error("Error during processing", "path", path, clog.Err(err))
			}
		}(req.Path)

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
