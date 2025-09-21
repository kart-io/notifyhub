package core

import (
	"context"
	"fmt"
	"sync"
)

// MemoryQueue is a simple in-memory queue implementation
type MemoryQueue struct {
	messages chan []byte
	closed   bool
	mu       sync.RWMutex
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue(capacity int) *MemoryQueue {
	return &MemoryQueue{
		messages: make(chan []byte, capacity),
		closed:   false,
	}
}

// Enqueue adds a message to the queue
func (q *MemoryQueue) Enqueue(ctx context.Context, data []byte) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	select {
	case q.messages <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("queue is full")
	}
}

// Dequeue retrieves a message from the queue
func (q *MemoryQueue) Dequeue(ctx context.Context) ([]byte, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	select {
	case data := <-q.messages:
		return data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Health checks if the queue is healthy
func (q *MemoryQueue) Health(ctx context.Context) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}
	return nil
}

// Close gracefully closes the queue
func (q *MemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	close(q.messages)
	return nil
}

// Size returns the current number of messages in the queue
func (q *MemoryQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return 0
	}
	return len(q.messages)
}

// MemoryQueueFactory creates memory queue instances
type MemoryQueueFactory struct{}

// Name returns the factory name
func (f *MemoryQueueFactory) Name() string {
	return "memory"
}

// Create creates a new memory queue instance
func (f *MemoryQueueFactory) Create(config map[string]interface{}) (Queue, error) {
	capacity := 1000 // default capacity

	if c, ok := config["capacity"]; ok {
		if cap, ok := c.(int); ok {
			capacity = cap
		}
	}

	return NewMemoryQueue(capacity), nil
}

// ValidateConfig validates the memory queue configuration
func (f *MemoryQueueFactory) ValidateConfig(config map[string]interface{}) error {
	if config == nil {
		return nil // memory queue doesn't require configuration
	}

	if c, ok := config["capacity"]; ok {
		if cap, ok := c.(int); !ok || cap <= 0 {
			return fmt.Errorf("capacity must be a positive integer")
		}
	}

	return nil
}
