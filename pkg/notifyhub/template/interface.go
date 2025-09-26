// Package template provides unified template management for NotifyHub
package template

import (
	"context"
	"time"
)

// Manager represents the core template management interface
// This provides unified template registration, rendering, and caching
type Manager interface {
	// RegisterTemplate registers a template with a unique name and content
	RegisterTemplate(name string, content string, engine EngineType) error

	// RenderTemplate renders a template with provided variables
	RenderTemplate(ctx context.Context, name string, variables map[string]interface{}) (string, error)

	// ValidateTemplate validates template syntax without rendering
	ValidateTemplate(name string) error

	// ListTemplates returns all registered template names
	ListTemplates() []string

	// GetTemplate returns template information by name
	GetTemplate(name string) (*Template, error)

	// RemoveTemplate removes a template from the manager
	RemoveTemplate(name string) error

	// Close gracefully shuts down the template manager
	Close() error
}

// Engine represents a template rendering engine interface
// This allows support for multiple template engines (Go, Mustache, Handlebars)
type Engine interface {
	// Name returns the engine name
	Name() string

	// Render renders template content with variables
	Render(ctx context.Context, content string, variables map[string]interface{}) (string, error)

	// Validate validates template syntax
	Validate(content string) error

	// GetCapabilities returns engine capabilities
	GetCapabilities() EngineCapabilities
}

// Cache represents the template caching interface
// This supports multi-level caching (memory, Redis, database)
type Cache interface {
	// Get retrieves cached template content
	Get(ctx context.Context, key string) (string, error)

	// Set stores template content in cache with TTL
	Set(ctx context.Context, key string, content string, ttl time.Duration) error

	// Delete removes cached template content
	Delete(ctx context.Context, key string) error

	// Clear clears all cached templates
	Clear(ctx context.Context) error

	// Close gracefully shuts down the cache
	Close() error
}

// Template represents a template with metadata
type Template struct {
	Name      string                 `json:"name"`
	Content   string                 `json:"content"`
	Engine    EngineType             `json:"engine"`
	Variables []string               `json:"variables"` // Required variables
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata"`
	CacheKey  string                 `json:"cache_key"`
	Compiled  bool                   `json:"compiled"` // Whether template is pre-compiled
}

// EngineType represents supported template engine types
type EngineType string

const (
	EngineGo         EngineType = "go"         // Go text/template
	EngineMustache   EngineType = "mustache"   // Mustache templates
	EngineHandlebars EngineType = "handlebars" // Handlebars templates
)

// EngineCapabilities describes what a template engine can do
type EngineCapabilities struct {
	Name                string   `json:"name"`
	SupportedFeatures   []string `json:"supported_features"` // loops, conditionals, functions
	MaxTemplateSize     int      `json:"max_template_size"`
	SupportsCompilation bool     `json:"supports_compilation"` // Pre-compilation support
	SupportsPartials    bool     `json:"supports_partials"`    // Template inclusion
	SupportsFunctions   bool     `json:"supports_functions"`   // Custom functions
	PerformanceLevel    string   `json:"performance_level"`    // high, medium, low
}

// RenderOptions provides options for template rendering
type RenderOptions struct {
	Timeout         time.Duration          `json:"timeout"`
	UseCache        bool                   `json:"use_cache"`
	CacheTTL        time.Duration          `json:"cache_ttl"`
	Variables       map[string]interface{} `json:"variables"`
	FailOnMissing   bool                   `json:"fail_on_missing"`  // Fail if variable is missing
	StrictMode      bool                   `json:"strict_mode"`      // Strict validation
	CustomFunctions map[string]interface{} `json:"custom_functions"` // Custom template functions
}

// ManagerConfig represents template manager configuration
type ManagerConfig struct {
	DefaultEngine   EngineType             `json:"default_engine"`
	EnableCache     bool                   `json:"enable_cache"`
	CacheConfig     CacheConfig            `json:"cache_config"`
	MultiLayerCache *MultiLayerCacheConfig `json:"multi_layer_cache,omitempty"` // Multi-layer cache configuration
	HotReload       bool                   `json:"hot_reload"`                  // Enable template hot reloading
	WatchPaths      []string               `json:"watch_paths"`                 // Directories to watch for changes
	MaxTemplateSize int                    `json:"max_template_size"`           // Maximum template size in bytes
	RenderTimeout   time.Duration          `json:"render_timeout"`              // Default render timeout
	ValidationMode  ValidationMode         `json:"validation_mode"`             // Template validation strictness
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Type       CacheType     `json:"type"`        // memory, redis, database
	TTL        time.Duration `json:"ttl"`         // Default cache TTL
	MaxEntries int           `json:"max_entries"` // Maximum cache entries
	// Redis specific
	RedisAddr     string `json:"redis_addr,omitempty"`
	RedisPassword string `json:"redis_password,omitempty"`
	RedisDB       int    `json:"redis_db,omitempty"`
	// Database specific
	DatabaseURL string `json:"database_url,omitempty"`
	TableName   string `json:"table_name,omitempty"`
}

// CacheType represents supported cache types
type CacheType string

const (
	CacheMemory   CacheType = "memory"   // In-memory cache
	CacheRedis    CacheType = "redis"    // Redis cache
	CacheDatabase CacheType = "database" // Database cache
)

// ValidationMode represents template validation strictness
type ValidationMode string

const (
	ValidationStrict ValidationMode = "strict" // Fail on any validation error
	ValidationWarn   ValidationMode = "warn"   // Log warnings but continue
	ValidationOff    ValidationMode = "off"    // No validation
)

// RenderResult represents the result of template rendering
type RenderResult struct {
	Content    string                 `json:"content"`
	Engine     EngineType             `json:"engine"`
	Variables  map[string]interface{} `json:"variables"`
	RenderTime time.Duration          `json:"render_time"`
	CacheHit   bool                   `json:"cache_hit"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ValidationResult represents template validation result
type ValidationResult struct {
	Valid     bool     `json:"valid"`
	Errors    []string `json:"errors"`
	Warnings  []string `json:"warnings"`
	Variables []string `json:"variables"` // Detected template variables
}
