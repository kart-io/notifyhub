// Package template provides hot reload functionality for template management
package template

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kart-io/notifyhub/pkg/logger"
)

// HotReloader manages template file watching and hot reloading
type HotReloader struct {
	manager    Manager
	watcher    *fsnotify.Watcher
	watchPaths []string
	templates  map[string]*WatchedTemplate // filename -> template info
	config     HotReloadConfig
	logger     logger.Logger
	mutex      sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// WatchedTemplate represents a template file being watched
type WatchedTemplate struct {
	Name     string     `json:"name"`
	FilePath string     `json:"file_path"`
	Engine   EngineType `json:"engine"`
	Content  string     `json:"content"`
	ModTime  time.Time  `json:"mod_time"`
	Size     int64      `json:"size"`
	Checksum string     `json:"checksum"`
}

// HotReloadConfig configures hot reload behavior
type HotReloadConfig struct {
	WatchPaths     []string                        `json:"watch_paths"`     // Directories to watch
	FileExtensions []string                        `json:"file_extensions"` // File extensions to watch (.tmpl, .tpl, .html)
	EngineMapping  map[string]EngineType           `json:"engine_mapping"`  // Extension to engine mapping
	ReloadDelay    time.Duration                   `json:"reload_delay"`    // Delay before reloading after file change
	MaxFileSize    int64                           `json:"max_file_size"`   // Maximum file size to watch
	RecursiveWatch bool                            `json:"recursive_watch"` // Watch subdirectories
	IgnorePatterns []string                        `json:"ignore_patterns"` // Patterns to ignore (.git, .tmp, etc.)
	OnReload       func(string, EngineType, error) `json:"-"`               // Callback on template reload
	OnError        func(string, error)             `json:"-"`               // Callback on error
}

// ReloadEvent represents a template reload event
type ReloadEvent struct {
	Type         ReloadEventType `json:"type"`
	TemplateName string          `json:"template_name"`
	FilePath     string          `json:"file_path"`
	Engine       EngineType      `json:"engine"`
	Timestamp    time.Time       `json:"timestamp"`
	Error        error           `json:"error,omitempty"`
}

// ReloadEventType represents the type of reload event
type ReloadEventType string

const (
	ReloadEventCreated  ReloadEventType = "created"
	ReloadEventModified ReloadEventType = "modified"
	ReloadEventDeleted  ReloadEventType = "deleted"
	ReloadEventError    ReloadEventType = "error"
)

// NewHotReloader creates a new hot reloader for the template manager
func NewHotReloader(manager Manager, config HotReloadConfig, logger logger.Logger) (*HotReloader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Set default configuration
	if len(config.FileExtensions) == 0 {
		config.FileExtensions = []string{".tmpl", ".tpl", ".html", ".txt", ".md"}
	}

	if config.EngineMapping == nil {
		config.EngineMapping = map[string]EngineType{
			".tmpl": EngineGo,
			".tpl":  EngineGo,
			".html": EngineGo,
			".txt":  EngineGo,
			".md":   EngineGo,
		}
	}

	if config.ReloadDelay == 0 {
		config.ReloadDelay = 100 * time.Millisecond
	}

	if config.MaxFileSize == 0 {
		config.MaxFileSize = 1024 * 1024 // 1MB
	}

	if len(config.IgnorePatterns) == 0 {
		config.IgnorePatterns = []string{".git", ".tmp", ".swp", ".DS_Store", "~"}
	}

	reloader := &HotReloader{
		manager:    manager,
		watcher:    watcher,
		watchPaths: config.WatchPaths,
		templates:  make(map[string]*WatchedTemplate),
		config:     config,
		logger:     logger,
		stopCh:     make(chan struct{}),
	}

	// Initialize watched templates
	if err := reloader.initializeWatchedTemplates(); err != nil {
		_ = watcher.Close() // Ignore close error during cleanup
		return nil, fmt.Errorf("failed to initialize watched templates: %w", err)
	}

	// Start watching
	if err := reloader.startWatching(); err != nil {
		_ = watcher.Close() // Ignore close error during cleanup
		return nil, fmt.Errorf("failed to start watching: %w", err)
	}

	logger.Info("Hot reloader initialized",
		"watch_paths", len(config.WatchPaths),
		"templates", len(reloader.templates),
		"extensions", config.FileExtensions)

	return reloader, nil
}

// initializeWatchedTemplates scans watch paths and registers existing templates
func (hr *HotReloader) initializeWatchedTemplates() error {
	for _, watchPath := range hr.watchPaths {
		if err := hr.scanDirectory(watchPath); err != nil {
			hr.logger.Error("Failed to scan directory", "path", watchPath, "error", err)
			return err
		}
	}

	hr.logger.Debug("Initialized watched templates", "count", len(hr.templates))
	return nil
}

// scanDirectory recursively scans a directory for template files
func (hr *HotReloader) scanDirectory(dirPath string) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			hr.logger.Warn("Error walking directory", "path", path, "error", err)
			return nil // Continue walking
		}

		// Skip directories
		if d.IsDir() {
			// Check if we should ignore this directory
			if hr.shouldIgnore(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file should be watched
		if hr.shouldWatchFile(path) {
			if err := hr.addWatchedTemplate(path); err != nil {
				hr.logger.Error("Failed to add watched template", "path", path, "error", err)
			}
		}

		return nil
	})
}

// shouldWatchFile checks if a file should be watched based on extension and patterns
func (hr *HotReloader) shouldWatchFile(filePath string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	validExt := false
	for _, allowedExt := range hr.config.FileExtensions {
		if ext == allowedExt {
			validExt = true
			break
		}
	}
	if !validExt {
		return false
	}

	// Check ignore patterns
	filename := filepath.Base(filePath)
	if hr.shouldIgnore(filename) {
		return false
	}

	// Check file size
	if stat, err := os.Stat(filePath); err == nil {
		if stat.Size() > hr.config.MaxFileSize {
			hr.logger.Warn("File too large to watch", "path", filePath, "size", stat.Size())
			return false
		}
	}

	return true
}

// shouldIgnore checks if a file/directory should be ignored
func (hr *HotReloader) shouldIgnore(name string) bool {
	for _, pattern := range hr.config.IgnorePatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}
	return false
}

// addWatchedTemplate adds a template file to the watched list
func (hr *HotReloader) addWatchedTemplate(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", filePath, err)
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat template file %s: %w", filePath, err)
	}

	// Determine engine type from extension
	ext := strings.ToLower(filepath.Ext(filePath))
	engine := hr.config.EngineMapping[ext]
	if engine == "" {
		engine = EngineGo // Default to Go templates
	}

	// Generate template name from file path
	templateName := hr.generateTemplateName(filePath)

	watchedTemplate := &WatchedTemplate{
		Name:     templateName,
		FilePath: filePath,
		Engine:   engine,
		Content:  string(content),
		ModTime:  stat.ModTime(),
		Size:     stat.Size(),
		Checksum: hr.calculateChecksum(content),
	}

	hr.mutex.Lock()
	hr.templates[filePath] = watchedTemplate
	hr.mutex.Unlock()

	// Register template with manager
	if err := hr.manager.RegisterTemplate(templateName, string(content), engine); err != nil {
		hr.logger.Error("Failed to register template", "name", templateName, "error", err)
		return err
	}

	hr.logger.Debug("Added watched template", "name", templateName, "path", filePath, "engine", engine)
	return nil
}

// generateTemplateName generates a template name from file path
func (hr *HotReloader) generateTemplateName(filePath string) string {
	// Remove extension and convert path separators to dots
	name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// For nested paths, include directory structure
	for _, watchPath := range hr.watchPaths {
		if strings.HasPrefix(filePath, watchPath) {
			relPath, _ := filepath.Rel(watchPath, filePath)
			relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))
			name = strings.ReplaceAll(relPath, string(filepath.Separator), ".")
			break
		}
	}

	return name
}

// calculateChecksum calculates a simple checksum for content comparison
func (hr *HotReloader) calculateChecksum(content []byte) string {
	// Simple hash based on content length and first/last bytes
	if len(content) == 0 {
		return "empty"
	}

	sum := len(content)
	if len(content) > 0 {
		sum += int(content[0]) * 31
	}
	if len(content) > 1 {
		sum += int(content[len(content)-1]) * 37
	}

	return fmt.Sprintf("%x", sum)
}

// startWatching starts the file system watching
func (hr *HotReloader) startWatching() error {
	// Add watch paths
	for _, watchPath := range hr.watchPaths {
		if err := hr.addWatchPath(watchPath); err != nil {
			return fmt.Errorf("failed to watch path %s: %w", watchPath, err)
		}
	}

	// Start event processing goroutine
	hr.wg.Add(1)
	go hr.processEvents()

	hr.logger.Debug("Started file system watching", "paths", hr.watchPaths)
	return nil
}

// addWatchPath adds a path to the file watcher
func (hr *HotReloader) addWatchPath(path string) error {
	// Add the directory to watcher
	if err := hr.watcher.Add(path); err != nil {
		return fmt.Errorf("failed to add watch path: %w", err)
	}

	// If recursive watching is enabled, add subdirectories
	if hr.config.RecursiveWatch {
		return filepath.WalkDir(path, func(walkPath string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Continue walking
			}

			if d.IsDir() && walkPath != path {
				if !hr.shouldIgnore(d.Name()) {
					if addErr := hr.watcher.Add(walkPath); addErr != nil {
						hr.logger.Warn("Failed to add subdirectory to watcher", "path", walkPath, "error", addErr)
					}
				}
			}

			return nil
		})
	}

	return nil
}

// processEvents processes file system events
func (hr *HotReloader) processEvents() {
	defer hr.wg.Done()

	// Debounce map to avoid multiple rapid events for the same file
	debounceMap := make(map[string]*time.Timer)
	debounceMutex := sync.Mutex{}

	for {
		select {
		case event, ok := <-hr.watcher.Events:
			if !ok {
				return
			}

			// Check if we should handle this file
			if !hr.shouldWatchFile(event.Name) {
				continue
			}

			hr.logger.Debug("File system event", "event", event)

			// Debounce events for the same file
			debounceMutex.Lock()
			if timer, exists := debounceMap[event.Name]; exists {
				timer.Stop()
			}
			debounceMap[event.Name] = time.AfterFunc(hr.config.ReloadDelay, func() {
				hr.handleFileEvent(event)
				debounceMutex.Lock()
				delete(debounceMap, event.Name)
				debounceMutex.Unlock()
			})
			debounceMutex.Unlock()

		case err, ok := <-hr.watcher.Errors:
			if !ok {
				return
			}
			hr.logger.Error("File watcher error", "error", err)
			if hr.config.OnError != nil {
				hr.config.OnError("watcher", err)
			}

		case <-hr.stopCh:
			return
		}
	}
}

// handleFileEvent handles a specific file system event
func (hr *HotReloader) handleFileEvent(event fsnotify.Event) {
	filePath := event.Name

	hr.mutex.Lock()
	watchedTemplate, exists := hr.templates[filePath]
	hr.mutex.Unlock()

	switch {
	case event.Has(fsnotify.Write) || event.Has(fsnotify.Create):
		// File was modified or created
		if err := hr.reloadTemplate(filePath, exists); err != nil {
			hr.logger.Error("Failed to reload template", "path", filePath, "error", err)
			hr.notifyReload("", EngineGo, ReloadEventError, err)
		} else {
			eventType := ReloadEventModified
			if !exists {
				eventType = ReloadEventCreated
			}
			engine := EngineGo
			if watchedTemplate != nil {
				engine = watchedTemplate.Engine
			}
			hr.notifyReload(hr.generateTemplateName(filePath), engine, eventType, nil)
		}

	case event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename):
		// File was deleted or renamed
		if exists {
			if err := hr.removeTemplate(filePath); err != nil {
				hr.logger.Error("Failed to remove template", "path", filePath, "error", err)
			} else {
				hr.notifyReload(watchedTemplate.Name, watchedTemplate.Engine, ReloadEventDeleted, nil)
			}
		}
	}
}

// reloadTemplate reloads a template from disk
func (hr *HotReloader) reloadTemplate(filePath string, existed bool) error {
	// Check if file still exists
	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File was deleted, handle as removal
			if existed {
				return hr.removeTemplate(filePath)
			}
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Check if content actually changed
	newChecksum := hr.calculateChecksum(content)
	hr.mutex.RLock()
	oldTemplate, exists := hr.templates[filePath]
	hr.mutex.RUnlock()

	if exists && oldTemplate.Checksum == newChecksum {
		// Content hasn't changed, skip reload
		return nil
	}

	// Determine engine type
	ext := strings.ToLower(filepath.Ext(filePath))
	engine := hr.config.EngineMapping[ext]
	if engine == "" {
		engine = EngineGo
	}

	// Generate template name
	templateName := hr.generateTemplateName(filePath)

	// Update watched template
	watchedTemplate := &WatchedTemplate{
		Name:     templateName,
		FilePath: filePath,
		Engine:   engine,
		Content:  string(content),
		ModTime:  stat.ModTime(),
		Size:     stat.Size(),
		Checksum: newChecksum,
	}

	hr.mutex.Lock()
	hr.templates[filePath] = watchedTemplate
	hr.mutex.Unlock()

	// Register/update template with manager
	if err := hr.manager.RegisterTemplate(templateName, string(content), engine); err != nil {
		return fmt.Errorf("failed to register template: %w", err)
	}

	hr.logger.Info("Template reloaded", "name", templateName, "path", filePath, "engine", engine)
	return nil
}

// removeTemplate removes a template from watching and manager
func (hr *HotReloader) removeTemplate(filePath string) error {
	hr.mutex.Lock()
	watchedTemplate, exists := hr.templates[filePath]
	if exists {
		delete(hr.templates, filePath)
	}
	hr.mutex.Unlock()

	if exists {
		// Remove from manager
		if err := hr.manager.RemoveTemplate(watchedTemplate.Name); err != nil {
			hr.logger.Error("Failed to remove template from manager", "name", watchedTemplate.Name, "error", err)
		}

		hr.logger.Info("Template removed", "name", watchedTemplate.Name, "path", filePath)
	}

	return nil
}

// notifyReload notifies about a template reload event
func (hr *HotReloader) notifyReload(templateName string, engine EngineType, eventType ReloadEventType, err error) {
	if hr.config.OnReload != nil {
		hr.config.OnReload(templateName, engine, err)
	}

	// Log the event
	switch eventType {
	case ReloadEventCreated:
		hr.logger.Info("Template created", "name", templateName, "engine", engine)
	case ReloadEventModified:
		hr.logger.Info("Template modified", "name", templateName, "engine", engine)
	case ReloadEventDeleted:
		hr.logger.Info("Template deleted", "name", templateName, "engine", engine)
	case ReloadEventError:
		hr.logger.Error("Template reload error", "name", templateName, "error", err)
	}
}

// GetWatchedTemplates returns all currently watched templates
func (hr *HotReloader) GetWatchedTemplates() map[string]*WatchedTemplate {
	hr.mutex.RLock()
	defer hr.mutex.RUnlock()

	result := make(map[string]*WatchedTemplate)
	for k, v := range hr.templates {
		templateCopy := *v
		result[k] = &templateCopy
	}

	return result
}

// Stop stops the hot reloader
func (hr *HotReloader) Stop() error {
	hr.logger.Info("Stopping hot reloader")

	// Signal stop
	close(hr.stopCh)

	// Wait for goroutines to finish
	hr.wg.Wait()

	// Close watcher
	err := hr.watcher.Close()
	if err != nil {
		hr.logger.Error("Failed to close file watcher", "error", err)
	}

	hr.logger.Info("Hot reloader stopped")
	return err
}

// DefaultHotReloadConfig returns default hot reload configuration
func DefaultHotReloadConfig(watchPaths ...string) HotReloadConfig {
	return HotReloadConfig{
		WatchPaths:     watchPaths,
		FileExtensions: []string{".tmpl", ".tpl", ".html", ".txt"},
		EngineMapping: map[string]EngineType{
			".tmpl": EngineGo,
			".tpl":  EngineGo,
			".html": EngineGo,
			".txt":  EngineGo,
		},
		ReloadDelay:    100 * time.Millisecond,
		MaxFileSize:    1024 * 1024, // 1MB
		RecursiveWatch: true,
		IgnorePatterns: []string{".git", ".tmp", ".swp", ".DS_Store", "~", ".backup"},
	}
}
