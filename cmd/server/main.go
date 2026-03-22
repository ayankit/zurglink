package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ayankit/clog"
	"github.com/ayankit/zurg-syms/internal/config"
	"github.com/ayankit/zurg-syms/internal/organise"
	"github.com/ayankit/zurg-syms/internal/tmdb"
)

// WebhookRequest expects JSON like: {"relative_path": "/shows/The.Last.of.Us.S01E01.1080p.mkv"}
type WebhookRequest struct {
	RelativePath string `json:"relative_path"`
}

func main() {
	clog.Init(clog.LevelDebug)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	tmdbClient := tmdb.NewClient(cfg.TMDBToken)

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req WebhookRequest
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
				log.Printf("Error processing %s: %v", path, err)
			}
		}(req.RelativePath)

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Processing started"))
	})

	log.Printf("Starting Symlink Organizer on port %s", cfg.Port)
	log.Printf("Source: %s | Dest: %s", cfg.SourcePath, cfg.DestPath)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
