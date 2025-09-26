// Package template provides configuration options for template management
package template

import (
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// Option represents a configuration option function for template manager
type Option func(*ManagerConfig) error

// WithDefaultEngine sets the default template engine
func WithDefaultEngine(engine EngineType) Option {
	return func(config *ManagerConfig) error {
		config.DefaultEngine = engine
		return nil
	}
}

// WithCache enables template caching with the specified configuration
func WithCache(cacheType CacheType, ttl time.Duration, maxEntries int) Option {
	return func(config *ManagerConfig) error {
		config.EnableCache = true
		config.CacheConfig = CacheConfig{
			Type:       cacheType,
			TTL:        ttl,
			MaxEntries: maxEntries,
		}
		return nil
	}
}

// WithMemoryCache enables in-memory template caching
func WithMemoryCache(ttl time.Duration, maxEntries int) Option {
	return func(config *ManagerConfig) error {
		config.EnableCache = true
		config.CacheConfig = CacheConfig{
			Type:       CacheMemory,
			TTL:        ttl,
			MaxEntries: maxEntries,
		}
		return nil
	}
}

// WithRedisCache enables Redis template caching
func WithRedisCache(addr, password string, db int, ttl time.Duration) Option {
	return func(config *ManagerConfig) error {
		config.EnableCache = true
		config.CacheConfig = CacheConfig{
			Type:          CacheRedis,
			TTL:           ttl,
			RedisAddr:     addr,
			RedisPassword: password,
			RedisDB:       db,
		}
		return nil
	}
}

// WithDatabaseCache enables database template caching
func WithDatabaseCache(dbURL, tableName string, ttl time.Duration) Option {
	return func(config *ManagerConfig) error {
		config.EnableCache = true
		config.CacheConfig = CacheConfig{
			Type:        CacheDatabase,
			TTL:         ttl,
			DatabaseURL: dbURL,
			TableName:   tableName,
		}
		return nil
	}
}

// WithMultiLayerCache enables 3-layer caching system (Memory → Redis → Database)
func WithMultiLayerCache(memoryTTL, redisTTL, databaseTTL time.Duration) Option {
	return func(config *ManagerConfig) error {
		config.EnableCache = true
		multiConfig := DefaultMultiLayerCacheConfig()
		multiConfig.MemoryTTL = memoryTTL
		multiConfig.RedisTTL = redisTTL
		multiConfig.DatabaseTTL = databaseTTL
		config.MultiLayerCache = &multiConfig
		return nil
	}
}

// WithMultiLayerCacheConfig enables multi-layer cache with custom configuration
func WithMultiLayerCacheConfig(multiConfig MultiLayerCacheConfig) Option {
	return func(config *ManagerConfig) error {
		config.EnableCache = true
		config.MultiLayerCache = &multiConfig
		return nil
	}
}

// WithWriteThroughCache enables write-through caching for multi-layer cache
func WithWriteThroughCache() Option {
	return func(config *ManagerConfig) error {
		if config.MultiLayerCache == nil {
			// Initialize with defaults if not set
			multiConfig := DefaultMultiLayerCacheConfig()
			config.MultiLayerCache = &multiConfig
		}
		config.MultiLayerCache.WriteThrough = true
		return nil
	}
}

// WithReadThroughCache enables read-through caching for multi-layer cache
func WithReadThroughCache() Option {
	return func(config *ManagerConfig) error {
		if config.MultiLayerCache == nil {
			// Initialize with defaults if not set
			multiConfig := DefaultMultiLayerCacheConfig()
			config.MultiLayerCache = &multiConfig
		}
		config.MultiLayerCache.ReadThrough = true
		return nil
	}
}

// WithHotReload enables template hot reloading
func WithHotReload(watchPaths ...string) Option {
	return func(config *ManagerConfig) error {
		config.HotReload = true
		config.WatchPaths = watchPaths
		return nil
	}
}

// WithMaxTemplateSize sets the maximum template size
func WithMaxTemplateSize(size int) Option {
	return func(config *ManagerConfig) error {
		config.MaxTemplateSize = size
		return nil
	}
}

// WithRenderTimeout sets the default template rendering timeout
func WithRenderTimeout(timeout time.Duration) Option {
	return func(config *ManagerConfig) error {
		config.RenderTimeout = timeout
		return nil
	}
}

// WithValidationMode sets the template validation mode
func WithValidationMode(mode ValidationMode) Option {
	return func(config *ManagerConfig) error {
		config.ValidationMode = mode
		return nil
	}
}

// WithStrictValidation enables strict template validation
func WithStrictValidation() Option {
	return func(config *ManagerConfig) error {
		config.ValidationMode = ValidationStrict
		return nil
	}
}

// WithNoValidation disables template validation
func WithNoValidation() Option {
	return func(config *ManagerConfig) error {
		config.ValidationMode = ValidationOff
		return nil
	}
}

// DefaultManagerConfig returns a sensible default configuration
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		DefaultEngine: EngineGo,
		EnableCache:   true,
		CacheConfig: CacheConfig{
			Type:       CacheMemory,
			TTL:        30 * time.Minute,
			MaxEntries: 1000,
		},
		HotReload:       false,
		WatchPaths:      []string{},
		MaxTemplateSize: 1024 * 1024, // 1MB
		RenderTimeout:   30 * time.Second,
		ValidationMode:  ValidationWarn,
	}
}

// NewManagerWithOptions creates a new template manager with configuration options
func NewManagerWithOptions(logger logger.Logger, opts ...Option) (Manager, error) {
	config := DefaultManagerConfig()

	for _, opt := range opts {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}

	return NewManager(config, logger)
}

// TestConfig returns configuration optimized for testing
func TestConfig() ManagerConfig {
	return ManagerConfig{
		DefaultEngine:   EngineGo,
		EnableCache:     false, // Disable caching for tests
		HotReload:       false,
		WatchPaths:      []string{},
		MaxTemplateSize: 64 * 1024, // 64KB for tests
		RenderTimeout:   5 * time.Second,
		ValidationMode:  ValidationStrict, // Strict validation in tests
	}
}

// PerformanceConfig returns configuration optimized for performance
func PerformanceConfig() ManagerConfig {
	return ManagerConfig{
		DefaultEngine: EngineGo, // Go templates are fastest
		EnableCache:   true,
		CacheConfig: CacheConfig{
			Type:       CacheMemory,
			TTL:        time.Hour, // Longer cache for performance
			MaxEntries: 10000,     // More cache entries
		},
		HotReload:       false, // No hot reload in production
		WatchPaths:      []string{},
		MaxTemplateSize: 2 * 1024 * 1024, // 2MB
		RenderTimeout:   10 * time.Second,
		ValidationMode:  ValidationOff, // No validation for performance
	}
}

// DevelopmentConfig returns configuration optimized for development
func DevelopmentConfig(watchPaths ...string) ManagerConfig {
	return ManagerConfig{
		DefaultEngine:   EngineGo,
		EnableCache:     false, // Disable cache for development
		HotReload:       true,  // Enable hot reload
		WatchPaths:      watchPaths,
		MaxTemplateSize: 1024 * 1024, // 1MB
		RenderTimeout:   30 * time.Second,
		ValidationMode:  ValidationWarn, // Warn but continue
	}
}
