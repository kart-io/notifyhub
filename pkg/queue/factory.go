// Package queue provides factory for creating queue instances
package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// QueueFactory creates queue instances based on configuration
type QueueFactory struct {
	logger logger.Logger
}

// NewQueueFactory creates a new queue factory
func NewQueueFactory(log logger.Logger) Factory {
	if log == nil {
		log = logger.Discard
	}
	return &QueueFactory{
		logger: log,
	}
}

// CreateQueue creates a new queue instance based on configuration
func (f *QueueFactory) CreateQueue(config *Config) (Queue, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	f.logger.Debug("Creating queue", "type", config.Type, "capacity", config.Capacity)

	var baseQueue Queue

	switch config.Type {
	case "memory", "":
		// Default to memory queue
		capacity := config.Capacity
		if capacity <= 0 {
			capacity = 1000 // Default capacity
		}
		baseQueue = NewMemoryQueue(capacity, f.logger)

	case "redis":
		// Redis queue implementation
		if config.RedisOptions == nil {
			return nil, fmt.Errorf("redis options required for redis queue")
		}
		capacity := config.Capacity
		if capacity <= 0 {
			capacity = 10000 // Default capacity for Redis
		}
		var err error
		baseQueue, err = NewRedisQueue(config.RedisOptions, capacity, f.logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create redis queue: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported queue type: %s", config.Type)
	}

	// Wrap with retry queue if retry policy is configured
	if config.RetryPolicy.MaxRetries > 0 {
		f.logger.Debug("Wrapping queue with retry support",
			"maxRetries", config.RetryPolicy.MaxRetries,
			"initialInterval", config.RetryPolicy.InitialInterval)
		baseQueue = NewRetryQueue(baseQueue, config.RetryPolicy, f.logger)
	}

	f.logger.Info("Queue created successfully",
		"type", config.Type,
		"capacity", config.Capacity,
		"retryEnabled", config.RetryPolicy.MaxRetries > 0)

	return baseQueue, nil
}

// GetSupportedTypes returns supported queue types
func (f *QueueFactory) GetSupportedTypes() []string {
	return []string{"memory", "redis"}
}

// Default configurations for common use cases

// DefaultMemoryQueueConfig returns default configuration for memory queue
func DefaultMemoryQueueConfig() *Config {
	return &Config{
		Type:     "memory",
		Capacity: 1000,
		Workers:  4,
		RetryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			MaxElapsedTime:  5 * time.Minute,
		},
		DeadLetterQueue: true,
	}
}

// DefaultRedisQueueConfig returns default configuration for Redis queue
func DefaultRedisQueueConfig(addr string) *Config {
	return &Config{
		Type:     "redis",
		Capacity: 10000,
		Workers:  8,
		RedisOptions: &RedisOptions{
			Addr:         addr,
			DB:           0,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
			MinIdleConns: 5,
			MaxConnAge:   0,
			KeyPrefix:    "notifyhub:queue:",
		},
		RetryPolicy: RetryPolicy{
			MaxRetries:      5,
			InitialInterval: 1 * time.Second,
			MaxInterval:     60 * time.Second,
			Multiplier:      2.0,
			MaxElapsedTime:  10 * time.Minute,
		},
		DeadLetterQueue: true,
	}
}

// QueueManager manages multiple queues
type QueueManager struct {
	queues  map[string]Queue
	workers map[string]WorkerPool
	factory Factory
	logger  logger.Logger
	mutex   sync.RWMutex
}

// NewQueueManager creates a new queue manager
func NewQueueManager(factory Factory, log logger.Logger) *QueueManager {
	if log == nil {
		log = logger.Discard
	}
	if factory == nil {
		factory = NewQueueFactory(log)
	}
	return &QueueManager{
		queues:  make(map[string]Queue),
		workers: make(map[string]WorkerPool),
		factory: factory,
		logger:  log,
	}
}

// CreateQueue creates and registers a new queue
func (m *QueueManager) CreateQueue(name string, config *Config, handler MessageHandler) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.queues[name]; exists {
		return fmt.Errorf("queue %s already exists", name)
	}

	// Create queue
	queue, err := m.factory.CreateQueue(config)
	if err != nil {
		return fmt.Errorf("failed to create queue %s: %w", name, err)
	}

	m.queues[name] = queue

	// Create worker pool if handler is provided
	if handler != nil {
		workerCount := config.Workers
		if workerCount <= 0 {
			workerCount = 4
		}

		pool := NewWorkerPool(queue, handler, 1, workerCount, m.logger)
		m.workers[name] = pool

		// Start the worker pool
		ctx := context.Background()
		if err := pool.Start(ctx); err != nil {
			m.logger.Error("Failed to start worker pool", "queue", name, "error", err)
			// Don't fail queue creation, workers can be started later
		}
	}

	m.logger.Info("Queue registered", "name", name, "type", config.Type)
	return nil
}

// GetQueue retrieves a queue by name
func (m *QueueManager) GetQueue(name string) (Queue, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	queue, exists := m.queues[name]
	return queue, exists
}

// GetWorkerPool retrieves a worker pool by queue name
func (m *QueueManager) GetWorkerPool(name string) (WorkerPool, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	pool, exists := m.workers[name]
	return pool, exists
}

// ListQueues returns all queue names
func (m *QueueManager) ListQueues() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	names := make([]string, 0, len(m.queues))
	for name := range m.queues {
		names = append(names, name)
	}
	return names
}

// GetStats returns statistics for all queues
func (m *QueueManager) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]interface{})

	for name, queue := range m.queues {
		queueStats := map[string]interface{}{
			"size":     queue.Size(),
			"is_empty": queue.IsEmpty(),
		}

		// Add worker pool stats if available
		if pool, exists := m.workers[name]; exists {
			queueStats["workers"] = pool.GetStats()
		}

		stats[name] = queueStats
	}

	return stats
}

// Close closes all queues and stops all workers
func (m *QueueManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.Info("Closing queue manager")

	var lastError error

	// Stop all worker pools first
	for name, pool := range m.workers {
		if err := pool.Stop(); err != nil {
			m.logger.Error("Failed to stop worker pool", "queue", name, "error", err)
			lastError = err
		}
	}

	// Close all queues
	for name, queue := range m.queues {
		if err := queue.Close(); err != nil {
			m.logger.Error("Failed to close queue", "name", name, "error", err)
			lastError = err
		}
	}

	// Clear maps
	m.queues = make(map[string]Queue)
	m.workers = make(map[string]WorkerPool)

	m.logger.Info("Queue manager closed")
	return lastError
}