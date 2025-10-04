// Package template provides template caching functionality
package template

import (
	"sync"
	"time"
)

// Cache interface for template caching
type Cache interface {
	// Get retrieves a cached template
	Get(key string) (interface{}, bool)

	// Set stores a template in cache
	Set(key string, value interface{}, ttl time.Duration)

	// Delete removes a template from cache
	Delete(key string)

	// Clear removes all cached templates
	Clear()

	// Size returns the number of cached templates
	Size() int
}

// MemoryCache implements in-memory template caching
type MemoryCache struct {
	cache map[string]*cacheItem
	mutex sync.RWMutex
}

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		cache: make(map[string]*cacheItem),
	}
}

// Get retrieves a cached template
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// Check expiration
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		delete(c.cache, key)
		return nil, false
	}

	return item.value, true
}

// Set stores a template in cache
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.cache[key] = &cacheItem{
		value:     value,
		expiresAt: expiresAt,
	}
}

// Delete removes a template from cache
func (c *MemoryCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.cache, key)
}

// Clear removes all cached templates
func (c *MemoryCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache = make(map[string]*cacheItem)
}

// Size returns the number of cached templates
func (c *MemoryCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.cache)
}
