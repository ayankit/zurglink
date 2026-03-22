package organise

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ayankit/clog"
	"github.com/ayankit/zurglink/internal/jellyfin"
	"github.com/ayankit/zurglink/internal/ptn"
	"github.com/ayankit/zurglink/internal/tmdb"
)

type Manager struct {
	SourcePath string
	DestPath   string
	tmdb       *tmdb.Client
	jellyfin   *jellyfin.Client
}

func NewManager(sourcePath, destPath string, tmdb *tmdb.Client, jellyfin *jellyfin.Client) (*Manager, error) {
	if err := verifyRead(sourcePath); err != nil {
		return nil, fmt.Errorf("source path verification failed: %w", err)
	}
	if err := verifyWrite(destPath); err != nil {
		return nil, fmt.Errorf("destination path verification failed: %w", err)
	}
	if tmdb == nil {
		return nil, fmt.Errorf("tmdb client was nil")
	}
	return &Manager{
		SourcePath: sourcePath,
		DestPath:   destPath,
		tmdb:       tmdb,
		jellyfin:   jellyfin,
	}, nil
}

// ProcessPath handles a directory or a single file recursively.
func (m *Manager) ProcessPath(relativePath string) error {
	absolutePath := relativePath
	if !strings.HasPrefix(relativePath, m.SourcePath+string(filepath.Separator)) {
		absolutePath = filepath.Join(m.SourcePath, relativePath)
	}

	info, err := os.Stat(absolutePath)
	if err != nil {
		return err
	}

	// Paths that we need to send to Jellyfin for update
	scanPaths := []string{}

	if info.IsDir() {
		err := filepath.WalkDir(absolutePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				clog.Error("error walking path", "path", path, clog.Err(err))
				return nil
			}
			if !d.IsDir() && ptn.IsVideoFile(path) {
				if err := m.processSingleFile(path); err != nil {
					clog.Error("failed to process file", "path", path, clog.Err(err))
				}
				scanPaths = append(scanPaths, strings.TrimPrefix(path, m.SourcePath))
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	if ptn.IsVideoFile(absolutePath) {
		if err := m.processSingleFile(absolutePath); err != nil {
			return err
		}
		scanPaths = append(scanPaths, strings.TrimPrefix(absolutePath, m.SourcePath))
	}

	// Notify jellyfin about changes
	m.notifyJellyfin(scanPaths)

	return nil
}

// processSingleFile handles the end-to-end logic for a single video file
func (m *Manager) processSingleFile(absolutePath string) error {
	fileName := filepath.Base(absolutePath)

	// 1. Parse raw filename
	parsed, err := ptn.Parse(fileName)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", fileName, err)
	}

	// 2. Get Official TMDB Data
	tmdbData, err := m.tmdb.Search(parsed.Title, parsed.Year, parsed.IsMovie)
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
		// Jellyfin Movie: movies/Movie Name (2006) [tmdbid-123]/Movie Name (2006) [1080p].mkv
		movieFolder := fmt.Sprintf("%s%s [tmdbid-%d]", tmdbData.Title, yearStr, tmdbData.ID)
		destDir = filepath.Join(m.DestPath, "movies", movieFolder)

		destFileName := fmt.Sprintf("%s%s%s%s", tmdbData.Title, yearStr, resTag, ext)
		destFile = filepath.Join(destDir, destFileName)

	} else {
		// Jellyfin TV: shows/Show Name (2006) [tmdbid-123]/Season 01/Show Name (2006) S01E01 [1080p].mkv
		showFolder := fmt.Sprintf("%s%s [tmdbid-%d]", tmdbData.Title, yearStr, tmdbData.ID)
		seasonFolder := fmt.Sprintf("Season %02d", parsed.Season)
		destDir = filepath.Join(m.DestPath, "shows", showFolder, seasonFolder)

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

	clog.Infof("Successfully organized: %s -> %s", absolutePath, filepath.Base(destFile))
	return nil
}

func (m *Manager) notifyJellyfin(paths []string) {
	if m.jellyfin == nil || len(paths) == 0 {
		clog.Debug("Notify jellyfin skipped", "client", m.jellyfin, "pathCount", len(paths))
		return
	}

	if len(paths) == 1 {
		response, err := m.jellyfin.ScanPath(paths[0])
		if err != nil {
			clog.Warn("Failure in notifying jellyfin for update", clog.Err(err))
			return
		}
		clog.Debug("Jellyfin notify path successful", "response", response)
		return
	}

	response, err := m.jellyfin.ScanPaths(paths)
	if err != nil {
		clog.Warn("Failure in notifying jellyfin for bulk update", clog.Err(err))
		return
	}
	clog.Debug("Jellyfin notify bulk path successful", "response", response)
}

func verifyRead(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist path=%s", path)
		}
		return fmt.Errorf("stat failed path=%s err=%w", path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory path=%s", path)
	}

	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("read failed, permission denied path=%s err=%w", path, err)
	}
	dir.Close()

	return nil
}

func verifyWrite(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0644); err != nil {
				return fmt.Errorf("create failed path=%s err=%w", path, err)
			}

			info, err = os.Stat(path)
			if err != nil {
				return fmt.Errorf("created dir but stat failed path=%s err=%w", path, err)
			}
		}
		return fmt.Errorf("stat failed path=%s err=%w", path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory path=%s", path)
	}

	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("read failed, permission denied path=%s err=%w", path, err)
	}
	dir.Close()

	// Verify write by creating a temp file
	tempFile, err := os.CreateTemp(path, ".write_test_*")
	if err != nil {
		return fmt.Errorf("write failed path=%s err=%w", path, err)
	}
	tempFile.Close()
	os.Remove(tempFile.Name())

	return nil
}
