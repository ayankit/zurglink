package config

import (
	"fmt"
	"os"

	"github.com/ayankit/clog"
)

type AppConfig struct {
	Port        string
	TMDBToken   string // TMDB Read Access Token (Bearer)
	SourcePath  string // The base path where Zurg is mounted
	DestPath    string // The base path for organized media
	JFToken     string // The API token from Jellyfin
	JFServer    string // The address of Jellyfin Server to push on changes
	JFMountPath string // The base path that is used in Jellyfin Container
}

func Load() (*AppConfig, error) {
	cfg := &AppConfig{
		Port:        getEnvOrDefault("PORT", "8080"),
		TMDBToken:   os.Getenv("TMDB_TOKEN"),
		SourcePath:  os.Getenv("SOURCE_DIR"),
		DestPath:    os.Getenv("DEST_DIR"),
		JFToken:     os.Getenv("JF_TOKEN"),
		JFServer:    os.Getenv("JF_SERVER"),
		JFMountPath: os.Getenv("JF_MOUNT_DIR"),
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
