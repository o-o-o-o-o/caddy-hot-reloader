package hotreloader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	ignore "github.com/sabhiram/go-gitignore"
	"go.uber.org/zap"
)

// FileWatcher watches files for changes
type FileWatcher struct {
	basePath    string
	config      *HotReloader
	logger      *zap.Logger
	watcher     *fsnotify.Watcher
	gitignore   *ignore.GitIgnore
	watchedDirs []string
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(basePath string, config *HotReloader, logger *zap.Logger) (*FileWatcher, error) {
	// Verify base path exists
	if info, err := os.Stat(basePath); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("base path is not a directory or doesn't exist: %s", basePath)
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	fw := &FileWatcher{
		basePath:    basePath,
		config:      config,
		logger:      logger,
		watcher:     fsWatcher,
		watchedDirs: make([]string, 0),
	}

	// Load .gitignore if configured
	if config.RespectGitignore {
		gitignorePath := filepath.Join(basePath, ".gitignore")
		if _, err := os.Stat(gitignorePath); err == nil {
			compiled, compileErr := ignore.CompileIgnoreFile(gitignorePath)
			if compileErr != nil {
				logger.Warn("failed to parse .gitignore",
					zap.String("path", gitignorePath),
					zap.Error(compileErr),
				)
			} else {
				fw.gitignore = compiled
				logger.Debug("loaded .gitignore", zap.String("path", gitignorePath))
			}
		}
	}

	// Add watches for configured patterns
	if err := fw.setupWatches(); err != nil {
		fsWatcher.Close()
		// Log the error but don't fail—let serving continue without hot reload
		logger.Warn("failed to setup file watcher",
			zap.String("basePath", basePath),
			zap.Error(err),
		)
		// Return the watcher anyway so serving doesn't break
		return fw, nil
	}

	return fw, nil
}

// setupWatches sets up file system watches based on configured patterns
func (fw *FileWatcher) setupWatches() error {
	watchedCount := 0

	// For each watch pattern, find matching directories
	for _, pattern := range fw.config.Watch {
		// Handle glob patterns like "site/**"
		basePattern := strings.TrimSuffix(pattern, "/**")
		basePattern = strings.TrimSuffix(basePattern, "**")

		watchPath := filepath.Join(fw.basePath, basePattern)

		// Check if directory exists
		if info, err := os.Stat(watchPath); err == nil && info.IsDir() {
			// Walk directory tree and add watches
			err := filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if !info.IsDir() {
					return nil
				}

				// Check if should be excluded
				if fw.shouldExclude(path) {
					fw.logger.Debug("skipping excluded directory",
						zap.String("path", path),
					)
					return filepath.SkipDir
				}

				// Add watch
				if err := fw.watcher.Add(path); err != nil {
					fw.logger.Warn("failed to watch directory",
						zap.String("path", path),
						zap.Error(err),
					)
					return nil // continue watching other dirs
				}

				fw.watchedDirs = append(fw.watchedDirs, path)
				fw.logger.Debug("watching directory", zap.String("path", path))
				watchedCount++

				return nil
			})

			if err != nil {
				fw.logger.Warn("error walking directory",
					zap.String("path", watchPath),
					zap.Error(err),
				)
				// Don't fail, continue with other patterns
			}
		} else {
			fw.logger.Debug("watch path does not exist (yet)",
				zap.String("pattern", pattern),
				zap.String("path", watchPath),
			)
		}
	}

	// As fallback, watch the base directory itself (if it exists) so we detect when subdirs are created
	if watchedCount == 0 {
		fw.logger.Info("no pattern directories found, watching base directory for dynamic creation",
			zap.String("basePath", fw.basePath),
		)
		if err := fw.watcher.Add(fw.basePath); err != nil {
			fw.logger.Warn("failed to watch base directory",
				zap.String("path", fw.basePath),
				zap.Error(err),
			)
			// Even this failed, just log and return—don't block serving
			return fmt.Errorf("failed to setup any watches for %s: %w", fw.basePath, err)
		}
		fw.watchedDirs = append(fw.watchedDirs, fw.basePath)
		watchedCount = 1
	}

	fw.logger.Info("file watcher setup complete",
		zap.Int("watched_dirs", watchedCount),
	)

	return nil
}

// Watch starts watching for file changes
func (fw *FileWatcher) Watch(broadcastCh chan<- *ReloadMessage, shutdownCh <-chan struct{}) {
	fw.logger.Debug("starting file watch loop")

	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Only care about Write and Create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				fw.handleFileChange(event.Name, broadcastCh)
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fw.logger.Error("file watcher error", zap.Error(err))

		case <-shutdownCh:
			fw.logger.Debug("stopping file watch loop")
			return
		}
	}
}

// handleFileChange processes a file change event
func (fw *FileWatcher) handleFileChange(filePath string, broadcastCh chan<- *ReloadMessage) {
	// Check if file should be excluded
	if fw.shouldExclude(filePath) {
		fw.logger.Debug("ignoring excluded file",
			zap.String("file", filePath),
		)
		return
	}

	// Check if file extension matches
	if !fw.matchesExtension(filePath) {
		return
	}

	// Get relative path
	relPath, err := filepath.Rel(fw.basePath, filePath)
	if err != nil {
		relPath = filepath.Base(filePath)
	}

	// Determine reload type based on file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	msgType := "reload"

	if ext == ".css" || ext == ".scss" || ext == ".sass" {
		msgType = "css"
	}

	fw.logger.Info("file changed",
		zap.String("file", relPath),
		zap.String("type", msgType),
	)

	// Send reload message
	select {
	case broadcastCh <- &ReloadMessage{
		Type: msgType,
		File: relPath,
	}:
	default:
		fw.logger.Warn("broadcast channel full, dropping message")
	}
}

// shouldExclude checks if a path should be excluded
func (fw *FileWatcher) shouldExclude(path string) bool {
	// Check .gitignore
	if fw.gitignore != nil {
		relPath, err := filepath.Rel(fw.basePath, path)
		if err == nil && fw.gitignore.MatchesPath(relPath) {
			return true
		}
	}

	// Check explicit exclude patterns
	for _, pattern := range fw.config.Exclude {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}

		// Also check if pattern matches any part of the path
		if strings.Contains(path, strings.ReplaceAll(pattern, "**", "")) {
			return true
		}
	}

	return false
}

// matchesExtension checks if file extension matches configured extensions
func (fw *FileWatcher) matchesExtension(filePath string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))

	for _, allowedExt := range fw.config.Extensions {
		if ext == strings.ToLower(allowedExt) {
			return true
		}
	}

	return false
}

// Close closes the file watcher
func (fw *FileWatcher) Close() error {
	fw.logger.Debug("closing file watcher")
	return fw.watcher.Close()
}
