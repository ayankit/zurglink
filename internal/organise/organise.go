package organise

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ayankit/zurglink/internal/ptn"
	"github.com/ayankit/zurglink/internal/tmdb"
)

// ProcessPath handles a directory or a single file recursively.
func ProcessPath(sourceBase, destBase, relPath string, tmdbClient *tmdb.Client) error {
	fullSourcePath := filepath.Join(sourceBase, relPath)

	info, err := os.Stat(fullSourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", fullSourcePath, err)
	}

	if info.IsDir() {
		return filepath.WalkDir(fullSourcePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				log.Printf("error walking path %s: %v", path, err)
				return nil
			}
			if !d.IsDir() && ptn.IsVideoFile(path) {
				if err := processSingleFile(destBase, path, tmdbClient); err != nil {
					log.Printf("failed to process file %s: %v", path, err)
				}
			}
			return nil
		})
	}

	if ptn.IsVideoFile(fullSourcePath) {
		return processSingleFile(destBase, fullSourcePath, tmdbClient)
	}

	log.Printf("skipped non-video file: %s", relPath)
	return nil
}

// processSingleFile handles the end-to-end logic for a single video file
func processSingleFile(base, absolutePath string, tmdbClient *tmdb.Client) error {
	fileName := filepath.Base(absolutePath)

	// 1. Parse raw filename
	parsed, err := ptn.Parse(fileName)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", fileName, err)
	}

	// 2. Get Official TMDB Data
	tmdbData, err := tmdbClient.Search(parsed.Title, parsed.Year, parsed.IsMovie)
	if err != nil {
		return fmt.Errorf("tmdb search failed for '%s': %w", parsed.Title, err)
	}

	// 3. Construct Jellyfin paths
	var destDir, destFile string
	resTag := ""
	if parsed.Resolution != "" {
		resTag = fmt.Sprintf(" - [%s]", parsed.Resolution)
	}

	ext := filepath.Ext(fileName)

	yearStr := ""
	if tmdbData.Year != "" {
		yearStr = fmt.Sprintf(" (%s)", tmdbData.Year)
	}

	if parsed.IsMovie {
		// Jellyfin Movie: Movies/Movie Name (2006) [tmdbid-123]/Movie Name (2006) [1080p].mkv
		movieFolder := fmt.Sprintf("%s%s [tmdbid-%d]", tmdbData.Title, yearStr, tmdbData.ID)
		destDir = filepath.Join(base, "Movies", movieFolder)

		destFileName := fmt.Sprintf("%s%s%s%s", tmdbData.Title, yearStr, resTag, ext)
		destFile = filepath.Join(destDir, destFileName)

	} else {
		// Jellyfin TV: Shows/Show Name (2006) [tmdbid-123]/Season 01/Show Name (2006) S01E01 [1080p].mkv
		showFolder := fmt.Sprintf("%s%s [tmdbid-%d]", tmdbData.Title, yearStr, tmdbData.ID)
		seasonFolder := fmt.Sprintf("Season %02d", parsed.Season)
		destDir = filepath.Join(base, "Shows", showFolder, seasonFolder)

		destFileName := fmt.Sprintf("%s%s S%02dE%02d%s%s", tmdbData.Title, yearStr, parsed.Season, parsed.Episode, resTag, ext)
		destFile = filepath.Join(destDir, destFileName)
	}

	// 4. Create Directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	// 5. Create Symlink (Remove if exists to overwrite)
	os.Remove(destFile)
	if err := os.Symlink(absolutePath, destFile); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	log.Printf("Successfully organized: %s -> %s", absolutePath, filepath.Base(destFile))
	return nil
}
