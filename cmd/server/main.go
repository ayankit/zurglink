package main

import (
	"encoding/json"
	"net/http"

	"github.com/ayankit/clog"
	"github.com/ayankit/zurglink/internal/config"
	"github.com/ayankit/zurglink/internal/organise"
	"github.com/ayankit/zurglink/internal/tmdb"
)

// RequestData expects JSON like: {"relative_path": "/shows/The.Last.of.Us.S01E01.1080p.mkv"}
type RequestData struct {
	RelativePath string `json:"relative_path"`
}

func main() {
	clog.Init(clog.LevelDebug)
	cfg, err := config.Load()
	if err != nil {
		clog.Fatal("Configuration load failed", clog.Err(err))
	}

	tmdbClient := tmdb.NewClient(cfg.TMDBToken)

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RequestData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request body", http.StatusBadRequest)
			return
		}

		if req.RelativePath == "" {
			http.Error(w, "relative_path is required", http.StatusBadRequest)
			return
		}

		// Process asynchronously so we don't block the webhook caller
		go func(path string) {
			err := organise.ProcessPath(cfg.SourcePath, cfg.DestPath, path, tmdbClient)
			if err != nil {
				clog.Error("Error during processing", "path", path, clog.Err(err))
			}
		}(req.RelativePath)

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
