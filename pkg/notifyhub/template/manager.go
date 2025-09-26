// Package template provides template management implementation
package template

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// ManagerImpl implements the Manager interface
type ManagerImpl struct {
	config      ManagerConfig
	engines     map[EngineType]Engine
	templates   map[string]*Template
	cache       Cache
	hotReloader *HotReloader
	logger      logger.Logger
	mutex       sync.RWMutex
}

// NewManager creates a new template manager with the given configuration
func NewManager(config ManagerConfig, logger logger.Logger) (Manager, error) {
	manager := &ManagerImpl{
		config:    config,
		engines:   make(map[EngineType]Engine),
		templates: make(map[string]*Template),
		logger:    logger,
	}

	// Initialize default engines
	if err := manager.initializeEngines(); err != nil {
		return nil, fmt.Errorf("failed to initialize template engines: %w", err)
	}

	// Initialize cache if enabled
	if config.EnableCache {
		// Check if multi-layer cache is configured
		if config.MultiLayerCache != nil {
			multiCache, err := NewMultiLayerCache(
				*config.MultiLayerCache,
				map[CacheType]CacheConfig{
					CacheMemory:   config.CacheConfig,
					CacheRedis:    config.CacheConfig,
					CacheDatabase: config.CacheConfig,
				},
				logger)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize multi-layer template cache: %w", err)
			}
			manager.cache = multiCache
		} else {
			// Single-layer cache
			cache, err := NewCache(config.CacheConfig, logger)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize template cache: %w", err)
			}
			manager.cache = cache
		}
	}

	// Initialize hot reloader if enabled
	if config.HotReload && len(config.WatchPaths) > 0 {
		hotReloadConfig := DefaultHotReloadConfig(config.WatchPaths...)

		// Set up reload callbacks
		hotReloadConfig.OnReload = func(templateName string, engine EngineType, reloadErr error) {
			if reloadErr != nil {
				logger.Error("Template reload failed", "template", templateName, "error", reloadErr)
			} else {
				logger.Info("Template reloaded successfully", "template", templateName, "engine", engine)

				// Clear cache for the reloaded template if cache is enabled
				if manager.cache != nil {
					cacheKey := fmt.Sprintf("template:%s:%s", templateName, engine)
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					if err := manager.cache.Delete(ctx, cacheKey); err != nil {
						logger.Warn("Failed to clear cache for reloaded template", "template", templateName, "error", err)
					}
				}
			}
		}

		hotReloadConfig.OnError = func(source string, err error) {
			logger.Error("Hot reload error", "source", source, "error", err)
		}

		hotReloader, err := NewHotReloader(manager, hotReloadConfig, logger)
		if err != nil {
			logger.Error("Failed to initialize hot reloader", "error", err)
			// Don't fail manager creation, just disable hot reload
		} else {
			manager.hotReloader = hotReloader
		}
	}

	manager.logger.Info("Template manager initialized",
		"engines", len(manager.engines),
		"cache_enabled", config.EnableCache,
		"hot_reload", manager.hotReloader != nil)

	return manager, nil
}

// RegisterTemplate registers a template with the manager
func (m *ManagerImpl) RegisterTemplate(name string, content string, engine EngineType) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Validate template name
	if name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	// Validate template size
	if len(content) > m.config.MaxTemplateSize {
		return fmt.Errorf("template size %d exceeds maximum %d", len(content), m.config.MaxTemplateSize)
	}

	// Get or use default engine
	if engine == "" {
		engine = m.config.DefaultEngine
	}

	// Check if engine is supported
	templateEngine, exists := m.engines[engine]
	if !exists {
		return fmt.Errorf("unsupported template engine: %s", engine)
	}

	// Validate template syntax
	if m.config.ValidationMode != ValidationOff {
		if err := templateEngine.Validate(content); err != nil {
			if m.config.ValidationMode == ValidationStrict {
				return fmt.Errorf("template validation failed: %w", err)
			}
			m.logger.Warn("Template validation warning", "template", name, "error", err)
		}
	}

	// Extract template variables
	variables, err := m.extractVariables(content, engine)
	if err != nil {
		m.logger.Warn("Failed to extract template variables", "template", name, "error", err)
	}

	// Create template
	template := &Template{
		Name:      name,
		Content:   content,
		Engine:    engine,
		Variables: variables,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
		CacheKey:  fmt.Sprintf("template:%s:%s", name, engine),
		Compiled:  false,
	}

	// Store template
	m.templates[name] = template

	// Cache template if caching is enabled
	if m.cache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := m.cache.Set(ctx, template.CacheKey, content, m.config.CacheConfig.TTL); err != nil {
			m.logger.Warn("Failed to cache template", "template", name, "error", err)
		}
	}

	m.logger.Debug("Template registered", "name", name, "engine", engine, "variables", len(variables))

	return nil
}

// RenderTemplate renders a template with provided variables
func (m *ManagerImpl) RenderTemplate(ctx context.Context, name string, variables map[string]interface{}) (string, error) {
	m.mutex.RLock()
	template, exists := m.templates[name]
	m.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("template not found: %s", name)
	}

	// Get template engine
	engine, exists := m.engines[template.Engine]
	if !exists {
		return "", fmt.Errorf("template engine not available: %s", template.Engine)
	}

	// Check cache first if enabled
	var content string
	var cacheHit bool

	if m.cache != nil {
		if cached, err := m.cache.Get(ctx, template.CacheKey); err == nil {
			content = cached
			cacheHit = true
		}
	}

	// Use template content if not cached
	if content == "" {
		content = template.Content
	}

	// Set rendering timeout
	renderCtx := ctx
	if m.config.RenderTimeout > 0 {
		var cancel context.CancelFunc
		renderCtx, cancel = context.WithTimeout(ctx, m.config.RenderTimeout)
		defer cancel()
	}

	// Render template
	start := time.Now()
	result, err := engine.Render(renderCtx, content, variables)
	renderTime := time.Since(start)

	if err != nil {
		m.logger.Error("Template rendering failed", "template", name, "engine", template.Engine, "error", err)
		return "", fmt.Errorf("template rendering failed: %w", err)
	}

	m.logger.Debug("Template rendered", "template", name, "render_time", renderTime, "cache_hit", cacheHit)

	return result, nil
}

// ValidateTemplate validates template syntax
func (m *ManagerImpl) ValidateTemplate(name string) error {
	m.mutex.RLock()
	template, exists := m.templates[name]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("template not found: %s", name)
	}

	engine, exists := m.engines[template.Engine]
	if !exists {
		return fmt.Errorf("template engine not available: %s", template.Engine)
	}

	return engine.Validate(template.Content)
}

// ListTemplates returns all registered template names
func (m *ManagerImpl) ListTemplates() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	names := make([]string, 0, len(m.templates))
	for name := range m.templates {
		names = append(names, name)
	}

	return names
}

// GetTemplate returns template information by name
func (m *ManagerImpl) GetTemplate(name string) (*Template, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	template, exists := m.templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	// Return a copy to prevent external modifications
	templateCopy := *template
	return &templateCopy, nil
}

// RemoveTemplate removes a template from the manager
func (m *ManagerImpl) RemoveTemplate(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	template, exists := m.templates[name]
	if !exists {
		return fmt.Errorf("template not found: %s", name)
	}

	// Remove from cache
	if m.cache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := m.cache.Delete(ctx, template.CacheKey); err != nil {
			m.logger.Warn("Failed to remove template from cache", "template", name, "error", err)
		}
	}

	// Remove from memory
	delete(m.templates, name)

	m.logger.Debug("Template removed", "name", name)

	return nil
}

// Close gracefully shuts down the template manager
func (m *ManagerImpl) Close() error {
	m.logger.Info("Closing template manager")

	// Stop hot reloader if enabled
	if m.hotReloader != nil {
		if err := m.hotReloader.Stop(); err != nil {
			m.logger.Error("Failed to stop hot reloader", "error", err)
		}
	}

	// Close cache if enabled
	if m.cache != nil {
		if err := m.cache.Close(); err != nil {
			m.logger.Error("Failed to close template cache", "error", err)
		}
	}

	// Clear templates
	m.mutex.Lock()
	m.templates = make(map[string]*Template)
	m.mutex.Unlock()

	m.logger.Info("Template manager closed")

	return nil
}

// initializeEngines initializes the supported template engines
func (m *ManagerImpl) initializeEngines() error {
	// Initialize Go text/template engine
	goEngine := NewGoEngine(m.logger)
	m.engines[EngineGo] = goEngine

	// Initialize Mustache template engine
	mustacheEngine := NewMustacheEngine(m.logger)
	m.engines[EngineMustache] = mustacheEngine

	// Initialize Handlebars template engine
	handlebarsEngine := NewHandlebarsEngine(m.logger)
	m.engines[EngineHandlebars] = handlebarsEngine

	m.logger.Debug("Template engines initialized",
		"count", len(m.engines),
		"engines", []string{"go", "mustache", "handlebars"})

	return nil
}

// extractVariables extracts template variables from content using the appropriate engine
func (m *ManagerImpl) extractVariables(content string, engine EngineType) ([]string, error) {
	// Get the appropriate engine and use its variable extraction
	templateEngine, exists := m.engines[engine]
	if !exists {
		// Fallback to simple extraction
		return m.simpleExtractVariables(content, engine), nil
	}

	// Use engine-specific extraction if available
	switch eng := templateEngine.(type) {
	case *GoEngine:
		return eng.ExtractVariables(content)
	case *MustacheEngine:
		return eng.ExtractVariables(content)
	case *HandlebarsEngine:
		return eng.ExtractVariables(content)
	default:
		// Fallback to simple extraction
		return m.simpleExtractVariables(content, engine), nil
	}
}

// simpleExtractVariables provides basic variable extraction as fallback
func (m *ManagerImpl) simpleExtractVariables(content string, engine EngineType) []string {
	var variables []string

	switch engine {
	case EngineGo:
		variables = extractGoTemplateVariables(content)
	case EngineMustache:
		variables = extractMustacheVariables(content)
	case EngineHandlebars:
		variables = extractHandlebarsVariables(content)
	}

	return variables
}

// GetHotReloadStats returns hot reload statistics and watched templates
func (m *ManagerImpl) GetHotReloadStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["enabled"] = m.hotReloader != nil

	if m.hotReloader != nil {
		watchedTemplates := m.hotReloader.GetWatchedTemplates()
		stats["watched_templates"] = len(watchedTemplates)
		stats["watch_paths"] = m.config.WatchPaths

		// Template details
		templates := make([]map[string]interface{}, 0, len(watchedTemplates))
		for _, template := range watchedTemplates {
			templates = append(templates, map[string]interface{}{
				"name":      template.Name,
				"file_path": template.FilePath,
				"engine":    string(template.Engine),
				"mod_time":  template.ModTime,
				"size":      template.Size,
			})
		}
		stats["templates"] = templates
	} else {
		stats["watched_templates"] = 0
		stats["watch_paths"] = []string{}
		stats["templates"] = []interface{}{}
	}

	return stats
}

// IsHotReloadEnabled returns whether hot reload is enabled
func (m *ManagerImpl) IsHotReloadEnabled() bool {
	return m.hotReloader != nil
}

// Helper functions for variable extraction (simplified implementations)
func extractGoTemplateVariables(content string) []string {
	// Simplified implementation - would use actual Go template parser
	return []string{}
}

func extractMustacheVariables(content string) []string {
	// Simplified implementation - would use Mustache parser
	return []string{}
}

func extractHandlebarsVariables(content string) []string {
	// Simplified implementation - would use Handlebars parser
	return []string{}
}
