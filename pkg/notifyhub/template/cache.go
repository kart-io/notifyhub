// Package template provides template caching implementation
package template

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// NewCache creates a new cache based on the configuration
func NewCache(config CacheConfig, logger logger.Logger) (Cache, error) {
	switch config.Type {
	case CacheMemory:
		return NewMemoryCache(config, logger), nil
	case CacheRedis:
		return NewRedisCache(config, logger)
	case CacheDatabase:
		return NewDatabaseCache(config, logger)
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", config.Type)
	}
}

// MemoryCache implements Cache interface using in-memory storage
type MemoryCache struct {
	config    CacheConfig
	logger    logger.Logger
	entries   map[string]*CacheEntry
	mutex     sync.RWMutex
	stopCh    chan struct{}
	cleanupWG sync.WaitGroup
}

// CacheEntry represents a cached template entry
type CacheEntry struct {
	Content   string
	ExpiresAt time.Time
	CreatedAt time.Time
	HitCount  int64
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(config CacheConfig, logger logger.Logger) *MemoryCache {
	cache := &MemoryCache{
		config:  config,
		logger:  logger,
		entries: make(map[string]*CacheEntry),
		stopCh:  make(chan struct{}),
	}

	// Start cleanup goroutine
	cache.startCleanup()

	logger.Debug("Memory cache initialized", "max_entries", config.MaxEntries, "ttl", config.TTL)

	return cache
}

// Get retrieves cached template content
func (c *MemoryCache) Get(ctx context.Context, key string) (string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return "", fmt.Errorf("cache miss: key not found")
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired entry (will be cleaned up by cleanup goroutine)
		return "", fmt.Errorf("cache miss: entry expired")
	}

	// Update hit count
	entry.HitCount++

	c.logger.Debug("Cache hit", "key", key, "hit_count", entry.HitCount)

	return entry.Content, nil
}

// Set stores template content in cache with TTL
func (c *MemoryCache) Set(ctx context.Context, key string, content string, ttl time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check cache size limit
	if len(c.entries) >= c.config.MaxEntries {
		// Remove oldest entry
		c.evictOldest()
	}

	// Use provided TTL or default
	if ttl == 0 {
		ttl = c.config.TTL
	}

	// Create cache entry
	entry := &CacheEntry{
		Content:   content,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
		HitCount:  0,
	}

	c.entries[key] = entry

	c.logger.Debug("Cache set", "key", key, "ttl", ttl, "size", len(content))

	return nil
}

// Delete removes cached template content
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.entries[key]; !exists {
		return fmt.Errorf("cache key not found: %s", key)
	}

	delete(c.entries, key)

	c.logger.Debug("Cache delete", "key", key)

	return nil
}

// Clear clears all cached templates
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = make(map[string]*CacheEntry)

	c.logger.Debug("Cache cleared")

	return nil
}

// Close gracefully shuts down the cache
func (c *MemoryCache) Close() error {
	c.logger.Debug("Closing memory cache")

	// Stop cleanup goroutine
	close(c.stopCh)
	c.cleanupWG.Wait()

	// Clear entries
	c.mutex.Lock()
	c.entries = make(map[string]*CacheEntry)
	c.mutex.Unlock()

	c.logger.Debug("Memory cache closed")

	return nil
}

// evictOldest removes the oldest cache entry
func (c *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.logger.Debug("Cache eviction", "key", oldestKey, "created_at", oldestTime)
	}
}

// startCleanup starts a background goroutine to clean up expired entries
func (c *MemoryCache) startCleanup() {
	c.cleanupWG.Add(1)

	go func() {
		defer c.cleanupWG.Done()

		ticker := time.NewTicker(time.Minute) // Cleanup every minute
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.cleanupExpired()
			case <-c.stopCh:
				return
			}
		}
	}()
}

// cleanupExpired removes expired cache entries
func (c *MemoryCache) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(c.entries, key)
	}

	if len(expiredKeys) > 0 {
		c.logger.Debug("Cache cleanup", "expired_entries", len(expiredKeys))
	}
}

// RedisCache implements Cache interface using Redis
type RedisCache struct {
	config CacheConfig
	logger logger.Logger
	client *redis.Client
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(config CacheConfig, logger logger.Logger) (*RedisCache, error) {
	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Debug("Redis cache initialized", "addr", config.RedisAddr, "db", config.RedisDB)

	return &RedisCache{
		config: config,
		logger: logger,
		client: rdb,
	}, nil
}

// Get retrieves cached template content from Redis
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	result, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("cache miss: key not found in Redis")
		}
		c.logger.Error("Redis GET failed", "key", key, "error", err)
		return "", fmt.Errorf("redis get error: %w", err)
	}

	c.logger.Debug("Redis cache hit", "key", key, "size", len(result))

	return result, nil
}

// Set stores template content in Redis with TTL
func (c *RedisCache) Set(ctx context.Context, key string, content string, ttl time.Duration) error {
	// Use provided TTL or default
	if ttl == 0 {
		ttl = c.config.TTL
	}

	err := c.client.Set(ctx, key, content, ttl).Err()
	if err != nil {
		c.logger.Error("Redis SET failed", "key", key, "error", err)
		return fmt.Errorf("redis set error: %w", err)
	}

	c.logger.Debug("Redis cache set", "key", key, "ttl", ttl, "size", len(content))

	return nil
}

// Delete removes cached template content from Redis
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	result := c.client.Del(ctx, key)
	if err := result.Err(); err != nil {
		c.logger.Error("Redis DEL failed", "key", key, "error", err)
		return fmt.Errorf("redis delete error: %w", err)
	}

	deleted := result.Val()
	c.logger.Debug("Redis cache delete", "key", key, "deleted", deleted)

	return nil
}

// Clear clears all cached templates from Redis
func (c *RedisCache) Clear(ctx context.Context) error {
	// Use pattern matching to delete only template keys
	keys, err := c.client.Keys(ctx, "template:*").Result()
	if err != nil {
		c.logger.Error("Redis KEYS failed", "error", err)
		return fmt.Errorf("redis keys error: %w", err)
	}

	if len(keys) > 0 {
		result := c.client.Del(ctx, keys...)
		if err := result.Err(); err != nil {
			c.logger.Error("Redis bulk DEL failed", "error", err)
			return fmt.Errorf("redis bulk delete error: %w", err)
		}

		deleted := result.Val()
		c.logger.Debug("Redis cache cleared", "keys_deleted", deleted)
	}

	return nil
}

// Close gracefully shuts down the Redis cache
func (c *RedisCache) Close() error {
	c.logger.Debug("Closing Redis cache")

	err := c.client.Close()
	if err != nil {
		c.logger.Error("Failed to close Redis connection", "error", err)
		return fmt.Errorf("redis close error: %w", err)
	}

	c.logger.Debug("Redis cache closed")

	return nil
}

// DatabaseCache implements Cache interface using database storage
type DatabaseCache struct {
	config CacheConfig
	logger logger.Logger
	// In a real implementation, this would have a database connection
	// For now, we'll implement a file-based approach as a database simulation
	entries map[string]*DatabaseCacheEntry
	mutex   sync.RWMutex
}

// DatabaseCacheEntry represents a database cache entry
type DatabaseCacheEntry struct {
	Key       string    `json:"key"`
	Content   string    `json:"content"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewDatabaseCache creates a new database cache
func NewDatabaseCache(config CacheConfig, logger logger.Logger) (*DatabaseCache, error) {
	// In a real implementation, this would initialize database connection
	// For this simulation, we use an in-memory map
	cache := &DatabaseCache{
		config:  config,
		logger:  logger,
		entries: make(map[string]*DatabaseCacheEntry),
	}

	logger.Debug("Database cache initialized (simulated)", "table", config.TableName)

	return cache, nil
}

// Get retrieves cached template content from database
func (c *DatabaseCache) Get(ctx context.Context, key string) (string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return "", fmt.Errorf("cache miss: key not found in database")
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		return "", fmt.Errorf("cache miss: entry expired in database")
	}

	c.logger.Debug("Database cache hit", "key", key, "size", len(entry.Content))

	return entry.Content, nil
}

// Set stores template content in database with TTL
func (c *DatabaseCache) Set(ctx context.Context, key string, content string, ttl time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Use provided TTL or default
	if ttl == 0 {
		ttl = c.config.TTL
	}

	now := time.Now()
	entry := &DatabaseCacheEntry{
		Key:       key,
		Content:   content,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// In a real implementation, this would be an SQL INSERT/UPDATE
	c.entries[key] = entry

	c.logger.Debug("Database cache set", "key", key, "ttl", ttl, "size", len(content))

	return nil
}

// Delete removes cached template content from database
func (c *DatabaseCache) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.entries[key]; !exists {
		return fmt.Errorf("cache key not found in database: %s", key)
	}

	// In a real implementation, this would be an SQL DELETE
	delete(c.entries, key)

	c.logger.Debug("Database cache delete", "key", key)

	return nil
}

// Clear clears all cached templates from database
func (c *DatabaseCache) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// In a real implementation, this would be SQL: DELETE FROM template_cache
	c.entries = make(map[string]*DatabaseCacheEntry)

	c.logger.Debug("Database cache cleared")

	return nil
}

// Close gracefully shuts down the database cache
func (c *DatabaseCache) Close() error {
	c.logger.Debug("Closing database cache")

	// In a real implementation, would close database connection
	c.mutex.Lock()
	c.entries = make(map[string]*DatabaseCacheEntry)
	c.mutex.Unlock()

	c.logger.Debug("Database cache closed")

	return nil
}
