package hotreloader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	ignore "github.com/sabhiram/go-gitignore"
	"go.uber.org/zap"
)

// debounceInterval is the quiet period after the last file event before a
// coalesced reload message is broadcast. Editor saves and build steps emit
// bursts of events; without settling, clients reload repeatedly and may
// fetch half-written files.
const debounceInterval = 100 * time.Millisecond

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
	// This allows flexible watching regardless of project structure
	if watchedCount == 0 {
		fw.logger.Info("watching entire base directory (no pattern directories found or no patterns specified)",
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

	debounce := time.NewTimer(debounceInterval)
	if !debounce.Stop() {
		<-debounce.C
	}
	timerActive := false
	pending := make(map[string]string) // relative path -> message type

	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Only care about Write and Create events
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			if event.Op&fsnotify.Create != 0 {
				fw.watchIfNewDir(event.Name, pending)
			}

			if msg := fw.classifyChange(event.Name); msg != nil {
				pending[msg.File] = msg.Type
			}

			if len(pending) > 0 {
				if timerActive && !debounce.Stop() {
					<-debounce.C
				}
				debounce.Reset(debounceInterval)
				timerActive = true
			}

		case <-debounce.C:
			timerActive = false
			fw.flushPending(pending, broadcastCh)
			pending = make(map[string]string)

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

// watchIfNewDir adds watches for a newly created directory tree so files
// inside it trigger reloads. Files already present when the walk runs (e.g.
// from a recursive copy) are recorded in pending, since their own events may
// have fired before the watch existed.
func (fw *FileWatcher) watchIfNewDir(path string, pending map[string]string) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}

	walkErr := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if fw.shouldExclude(p) {
				return filepath.SkipDir
			}
			if addErr := fw.watcher.Add(p); addErr != nil {
				fw.logger.Warn("failed to watch new directory",
					zap.String("path", p),
					zap.Error(addErr),
				)
				return nil
			}
			fw.watchedDirs = append(fw.watchedDirs, p)
			fw.logger.Debug("watching new directory", zap.String("path", p))
			return nil
		}
		if msg := fw.classifyChange(p); msg != nil {
			pending[msg.File] = msg.Type
		}
		return nil
	})
	if walkErr != nil {
		fw.logger.Warn("error walking new directory",
			zap.String("path", path),
			zap.Error(walkErr),
		)
	}
}

// classifyChange decides whether a changed path should trigger a reload and
// returns the message to coalesce, or nil to ignore the event.
func (fw *FileWatcher) classifyChange(filePath string) *ReloadMessage {
	if isEditorJunk(filepath.Base(filePath)) {
		return nil
	}

	if fw.shouldExclude(filePath) {
		fw.logger.Debug("ignoring excluded file",
			zap.String("file", filePath),
		)
		return nil
	}

	if !fw.matchesExtension(filePath) {
		return nil
	}

	// Directory creations get watches via watchIfNewDir but are not changes
	if info, err := os.Stat(filePath); err == nil && info.IsDir() {
		return nil
	}

	relPath, err := filepath.Rel(fw.basePath, filePath)
	if err != nil {
		relPath = filepath.Base(filePath)
	}

	msgType := "reload"
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".css" || ext == ".scss" || ext == ".sass" {
		msgType = "css"
	}

	fw.logger.Debug("file changed",
		zap.String("file", relPath),
		zap.String("type", msgType),
	)

	return &ReloadMessage{Type: msgType, File: relPath}
}

// flushPending broadcasts one coalesced message for all changes that settled
// within the debounce window.
func (fw *FileWatcher) flushPending(pending map[string]string, broadcastCh chan<- *ReloadMessage) {
	msg := coalescePending(pending)
	if msg == nil {
		return
	}

	fw.logger.Info("broadcasting change",
		zap.String("type", msg.Type),
		zap.String("file", msg.File),
		zap.Int("changed_files", len(pending)),
	)

	select {
	case broadcastCh <- msg:
	default:
		fw.logger.Warn("broadcast channel full, dropping message")
	}
}

// coalescePending reduces a batch of settled changes to a single message.
// Any non-CSS change forces a full reload; multiple CSS changes are sent with
// an empty file so the client refreshes every stylesheet.
func coalescePending(pending map[string]string) *ReloadMessage {
	if len(pending) == 0 {
		return nil
	}

	var cssFile string
	for file, msgType := range pending {
		if msgType == "reload" {
			return &ReloadMessage{Type: "reload", File: file}
		}
		cssFile = file
	}

	if len(pending) == 1 {
		return &ReloadMessage{Type: "css", File: cssFile}
	}
	return &ReloadMessage{Type: "css"}
}

// isEditorJunk reports whether a file name is a transient editor artifact:
// Vim's 4913 probe file, ~ backups and .sw? swap files, Emacs autosaves and
// lock files, and .DS_Store (kept here even though it is in the default
// excludes, so custom exclude lists don't reintroduce it).
func isEditorJunk(name string) bool {
	if name == "4913" || name == ".DS_Store" {
		return true
	}
	if strings.HasSuffix(name, "~") {
		return true
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".swp", ".swo", ".swx":
		return true
	}
	if strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#") {
		return true
	}
	if strings.HasPrefix(name, ".#") {
		return true
	}
	return false
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

// matchesExtension checks if file extension matches configured extensions.
// An empty list means every extension triggers a reload.
func (fw *FileWatcher) matchesExtension(filePath string) bool {
	if len(fw.config.Extensions) == 0 {
		return true
	}

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
