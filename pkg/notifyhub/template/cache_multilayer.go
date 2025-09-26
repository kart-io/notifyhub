// Package template provides multi-layer cache implementation
package template

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// MultiLayerCache implements a 3-layer caching strategy: Memory → Redis → Database
type MultiLayerCache struct {
	memoryCache   Cache
	redisCache    Cache
	databaseCache Cache
	config        MultiLayerCacheConfig
	logger        logger.Logger
}

// MultiLayerCacheConfig configures the multi-layer cache behavior
type MultiLayerCacheConfig struct {
	EnableMemory   bool          `json:"enable_memory"`
	EnableRedis    bool          `json:"enable_redis"`
	EnableDatabase bool          `json:"enable_database"`
	MemoryTTL      time.Duration `json:"memory_ttl"`    // Shortest TTL for memory
	RedisTTL       time.Duration `json:"redis_ttl"`     // Medium TTL for Redis
	DatabaseTTL    time.Duration `json:"database_ttl"`  // Longest TTL for database
	WriteThrough   bool          `json:"write_through"` // Write to all layers immediately
	ReadThrough    bool          `json:"read_through"`  // Populate upper layers on cache miss
}

// NewMultiLayerCache creates a new multi-layer cache with the specified layers
func NewMultiLayerCache(config MultiLayerCacheConfig, cacheConfigs map[CacheType]CacheConfig, logger logger.Logger) (*MultiLayerCache, error) {
	cache := &MultiLayerCache{
		config: config,
		logger: logger,
	}

	// Initialize memory cache if enabled
	if config.EnableMemory {
		if memoryCfg, exists := cacheConfigs[CacheMemory]; exists {
			memCache := NewMemoryCache(memoryCfg, logger)
			cache.memoryCache = memCache
		}
	}

	// Initialize Redis cache if enabled
	if config.EnableRedis {
		if redisCfg, exists := cacheConfigs[CacheRedis]; exists {
			redisCache, err := NewRedisCache(redisCfg, logger)
			if err != nil {
				logger.Warn("Failed to initialize Redis cache, disabling Redis layer", "error", err)
			} else {
				cache.redisCache = redisCache
			}
		}
	}

	// Initialize database cache if enabled
	if config.EnableDatabase {
		if dbCfg, exists := cacheConfigs[CacheDatabase]; exists {
			dbCache, err := NewDatabaseCache(dbCfg, logger)
			if err != nil {
				logger.Warn("Failed to initialize database cache, disabling database layer", "error", err)
			} else {
				cache.databaseCache = dbCache
			}
		}
	}

	logger.Info("Multi-layer cache initialized",
		"memory", cache.memoryCache != nil,
		"redis", cache.redisCache != nil,
		"database", cache.databaseCache != nil,
		"write_through", config.WriteThrough,
		"read_through", config.ReadThrough)

	return cache, nil
}

// Get retrieves cached content from the multi-layer cache
func (c *MultiLayerCache) Get(ctx context.Context, key string) (string, error) {
	var content string
	var err error
	var hitLayer string

	// Try memory cache first (L1)
	if c.memoryCache != nil {
		content, err = c.memoryCache.Get(ctx, key)
		if err == nil {
			hitLayer = "memory"
			c.logger.Debug("Multi-layer cache hit", "key", key, "layer", hitLayer)
			return content, nil
		}
	}

	// Try Redis cache (L2)
	if c.redisCache != nil {
		content, err = c.redisCache.Get(ctx, key)
		if err == nil {
			hitLayer = "redis"
			c.logger.Debug("Multi-layer cache hit", "key", key, "layer", hitLayer)

			// Populate memory cache if read-through is enabled
			if c.config.ReadThrough && c.memoryCache != nil {
				if setErr := c.memoryCache.Set(ctx, key, content, c.config.MemoryTTL); setErr != nil {
					c.logger.Warn("Failed to populate memory cache", "key", key, "error", setErr)
				}
			}

			return content, nil
		}
	}

	// Try database cache (L3)
	if c.databaseCache != nil {
		content, err = c.databaseCache.Get(ctx, key)
		if err == nil {
			hitLayer = "database"
			c.logger.Debug("Multi-layer cache hit", "key", key, "layer", hitLayer)

			// Populate upper layers if read-through is enabled
			if c.config.ReadThrough {
				if c.redisCache != nil {
					if setErr := c.redisCache.Set(ctx, key, content, c.config.RedisTTL); setErr != nil {
						c.logger.Warn("Failed to populate Redis cache", "key", key, "error", setErr)
					}
				}
				if c.memoryCache != nil {
					if setErr := c.memoryCache.Set(ctx, key, content, c.config.MemoryTTL); setErr != nil {
						c.logger.Warn("Failed to populate memory cache", "key", key, "error", setErr)
					}
				}
			}

			return content, nil
		}
	}

	// Cache miss in all layers
	return "", fmt.Errorf("cache miss: key not found in any layer")
}

// Set stores content in the multi-layer cache
func (c *MultiLayerCache) Set(ctx context.Context, key string, content string, ttl time.Duration) error {
	var errors []error

	if c.config.WriteThrough {
		// Write-through: write to all enabled layers
		if c.memoryCache != nil {
			memTTL := c.config.MemoryTTL
			if ttl > 0 && ttl < memTTL {
				memTTL = ttl // Use shorter TTL if provided
			}
			if err := c.memoryCache.Set(ctx, key, content, memTTL); err != nil {
				errors = append(errors, fmt.Errorf("memory cache set failed: %w", err))
			}
		}

		if c.redisCache != nil {
			redisTTL := c.config.RedisTTL
			if ttl > 0 && ttl < redisTTL {
				redisTTL = ttl
			}
			if err := c.redisCache.Set(ctx, key, content, redisTTL); err != nil {
				errors = append(errors, fmt.Errorf("redis cache set failed: %w", err))
			}
		}

		if c.databaseCache != nil {
			dbTTL := c.config.DatabaseTTL
			if ttl > 0 && ttl < dbTTL {
				dbTTL = ttl
			}
			if err := c.databaseCache.Set(ctx, key, content, dbTTL); err != nil {
				errors = append(errors, fmt.Errorf("database cache set failed: %w", err))
			}
		}
	} else {
		// Write-back: write to memory first, others later
		if c.memoryCache != nil {
			if err := c.memoryCache.Set(ctx, key, content, c.config.MemoryTTL); err != nil {
				errors = append(errors, fmt.Errorf("memory cache set failed: %w", err))
			}
		}
	}

	c.logger.Debug("Multi-layer cache set", "key", key, "write_through", c.config.WriteThrough, "errors", len(errors))

	if len(errors) > 0 {
		// Return first error but log all
		for _, err := range errors {
			c.logger.Error("Cache layer set error", "error", err)
		}
		return errors[0]
	}

	return nil
}

// Delete removes content from all cache layers
func (c *MultiLayerCache) Delete(ctx context.Context, key string) error {
	var errors []error

	if c.memoryCache != nil {
		if err := c.memoryCache.Delete(ctx, key); err != nil {
			errors = append(errors, fmt.Errorf("memory cache delete failed: %w", err))
		}
	}

	if c.redisCache != nil {
		if err := c.redisCache.Delete(ctx, key); err != nil {
			errors = append(errors, fmt.Errorf("redis cache delete failed: %w", err))
		}
	}

	if c.databaseCache != nil {
		if err := c.databaseCache.Delete(ctx, key); err != nil {
			errors = append(errors, fmt.Errorf("database cache delete failed: %w", err))
		}
	}

	c.logger.Debug("Multi-layer cache delete", "key", key, "errors", len(errors))

	if len(errors) > 0 {
		// Return first error but log all
		for _, err := range errors {
			c.logger.Error("Cache layer delete error", "error", err)
		}
		return errors[0]
	}

	return nil
}

// Clear clears all cache layers
func (c *MultiLayerCache) Clear(ctx context.Context) error {
	var errors []error

	if c.memoryCache != nil {
		if err := c.memoryCache.Clear(ctx); err != nil {
			errors = append(errors, fmt.Errorf("memory cache clear failed: %w", err))
		}
	}

	if c.redisCache != nil {
		if err := c.redisCache.Clear(ctx); err != nil {
			errors = append(errors, fmt.Errorf("redis cache clear failed: %w", err))
		}
	}

	if c.databaseCache != nil {
		if err := c.databaseCache.Clear(ctx); err != nil {
			errors = append(errors, fmt.Errorf("database cache clear failed: %w", err))
		}
	}

	c.logger.Debug("Multi-layer cache cleared", "errors", len(errors))

	if len(errors) > 0 {
		// Return first error but log all
		for _, err := range errors {
			c.logger.Error("Cache layer clear error", "error", err)
		}
		return errors[0]
	}

	return nil
}

// Close gracefully shuts down all cache layers
func (c *MultiLayerCache) Close() error {
	var errors []error

	c.logger.Debug("Closing multi-layer cache")

	if c.memoryCache != nil {
		if err := c.memoryCache.Close(); err != nil {
			errors = append(errors, fmt.Errorf("memory cache close failed: %w", err))
		}
	}

	if c.redisCache != nil {
		if err := c.redisCache.Close(); err != nil {
			errors = append(errors, fmt.Errorf("redis cache close failed: %w", err))
		}
	}

	if c.databaseCache != nil {
		if err := c.databaseCache.Close(); err != nil {
			errors = append(errors, fmt.Errorf("database cache close failed: %w", err))
		}
	}

	c.logger.Debug("Multi-layer cache closed", "errors", len(errors))

	if len(errors) > 0 {
		// Return first error but log all
		for _, err := range errors {
			c.logger.Error("Cache layer close error", "error", err)
		}
		return errors[0]
	}

	return nil
}

// GetStats returns statistics for each cache layer
func (c *MultiLayerCache) GetStats(ctx context.Context) map[string]interface{} {
	stats := make(map[string]interface{})

	stats["layers"] = map[string]bool{
		"memory":   c.memoryCache != nil,
		"redis":    c.redisCache != nil,
		"database": c.databaseCache != nil,
	}

	stats["config"] = map[string]interface{}{
		"write_through": c.config.WriteThrough,
		"read_through":  c.config.ReadThrough,
		"memory_ttl":    c.config.MemoryTTL,
		"redis_ttl":     c.config.RedisTTL,
		"database_ttl":  c.config.DatabaseTTL,
	}

	// In a real implementation, would collect hit/miss ratios and other metrics
	return stats
}

// DefaultMultiLayerCacheConfig returns a sensible default configuration
func DefaultMultiLayerCacheConfig() MultiLayerCacheConfig {
	return MultiLayerCacheConfig{
		EnableMemory:   true,
		EnableRedis:    true,
		EnableDatabase: true,
		MemoryTTL:      5 * time.Minute,  // Fast expiry for memory
		RedisTTL:       30 * time.Minute, // Medium expiry for Redis
		DatabaseTTL:    2 * time.Hour,    // Long expiry for database
		WriteThrough:   true,             // Write to all layers
		ReadThrough:    true,             // Populate upper layers on miss
	}
}
