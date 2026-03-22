package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ayankit/clog"
)

type AppConfig struct {
	Port        string
	TMDBToken   string // TMDB Read Access Token (Bearer)
	JFToken     string // The API token from Jellyfin
	JFServer    string // The address of Jellyfin Server to push on changes
	JFMountPath string // The base path that is used in Jellyfin Container
	SourcePath  string // The base path where Zurg is mounted
	DestPath    string // The base path for organized media
}

func Load() (*AppConfig, error) {
	cfg := &AppConfig{
		Port:        getEnvOrDefault("PORT", "8080"),
		TMDBToken:   os.Getenv("TMDB_TOKEN"),
		JFToken:     os.Getenv("JF_TOKEN"),
		JFServer:    os.Getenv("JF_SERVER"),
		JFMountPath: filepath.Clean(os.Getenv("JF_MOUNT_DIR")),
		SourcePath:  filepath.Clean(os.Getenv("SOURCE_DIR")),
		DestPath:    filepath.Clean(os.Getenv("DEST_DIR")),
	}

	if cfg.TMDBToken == "" {
		return nil, fmt.Errorf("TMDB_TOKEN environment variable is required")
	}
	if cfg.SourcePath == "" || cfg.DestPath == "" {
		return nil, fmt.Errorf("SOURCE_DIR and DEST_DIR environment variables are required")
	}

	if cfg.JFServer == "" || cfg.JFToken == "" {
		clog.Info("Jellyfin server or token not provided. The update service will not run.")
		if cfg.JFMountPath == "" {
			clog.Warn("Jellyfin Mount Path not specified, the destination path will be used.")
		}
	}

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return fallback
}
