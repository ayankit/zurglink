package organise

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ayankit/zurg-syms/internal/ptn"
	"github.com/ayankit/zurg-syms/internal/tmdb"
)

// ProcessFile handles the end-to-end logic for a single file
func ProcessFile(sourceBase, destBase, relPath string, tmdbClient *tmdb.Client) error {
	fullSourcePath := filepath.Join(sourceBase, relPath)

	// 1. Parse raw filename
	parsed, err := ptn.Parse(fullSourcePath)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", relPath, err)
	}

	// 2. Get Official TMDB Data
	tmdbData, err := tmdbClient.Search(parsed.Title, parsed.Year, parsed.IsMovie)
	if err != nil {
		return fmt.Errorf("tmdb search failed: %w", err)
	}

	// 3. Construct Jellyfin paths
	var destDir, destFile string
	resTag := ""
	if parsed.Resolution != "" {
		resTag = fmt.Sprintf(" [%s]", parsed.Resolution)
	}

	if parsed.IsMovie {
		// Jellyfin Movie: Movies/Movie Name (2006) [tmdbid-123]/Movie Name (2006) [1080p].mkv
		movieFolder := fmt.Sprintf("%s (%s) [tmdbid-%d]", tmdbData.Title, tmdbData.Year, tmdbData.ID)
		destDir = filepath.Join(destBase, "Movies", movieFolder)

		fileName := fmt.Sprintf("%s (%s)%s%s", tmdbData.Title, tmdbData.Year, resTag, parsed.Container)
		destFile = filepath.Join(destDir, fileName)

	} else {
		// Jellyfin TV: Shows/Show Name (2006) [tmdbid-123]/Season 01/Show Name (2006) S01E01 [1080p].mkv
		showFolder := fmt.Sprintf("%s (%s) [tmdbid-%d]", tmdbData.Title, tmdbData.Year, tmdbData.ID)
		seasonFolder := fmt.Sprintf("Season %02d", parsed.Season)
		destDir = filepath.Join(destBase, "Shows", showFolder, seasonFolder)

		fileName := fmt.Sprintf("%s (%s) S%02dE%02d%s%s", tmdbData.Title, tmdbData.Year, parsed.Season, parsed.Episode, resTag, parsed.Container)
		destFile = filepath.Join(destDir, fileName)
	}

	// 4. Create Directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	// 5. Create Symlink (Remove if exists to overwrite)
	os.Remove(destFile)
	if err := os.Symlink(fullSourcePath, destFile); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	log.Printf("Successfully organized: %s -> %s", parsed.Original, filepath.Base(destFile))
	return nil
}
