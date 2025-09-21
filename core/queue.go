package core

import (
	"context"
	"time"
)

// Queue defines a simplified interface for message queues
type Queue interface {
	// Enqueue adds a message to the queue
	Enqueue(ctx context.Context, data []byte) error

	// Dequeue retrieves a message from the queue
	// Returns nil if no message is available within the context timeout
	Dequeue(ctx context.Context) ([]byte, error)

	// Health checks if the queue is healthy
	Health(ctx context.Context) error

	// Close gracefully closes the queue
	Close() error
}

// QueueFactory creates queue instances
type QueueFactory interface {
	// Name returns the queue type name
	Name() string

	// Create creates a queue instance with the given configuration
	Create(config map[string]interface{}) (Queue, error)

	// ValidateConfig validates the configuration
	ValidateConfig(config map[string]interface{}) error
}

// QueueMessage represents a message in the queue
type QueueMessage struct {
	ID        string            `json:"id"`
	Data      []byte            `json:"data"`
	Headers   map[string]string `json:"headers,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	Retries   int               `json:"retries"`
}

// NewQueueMessage creates a new queue message
func NewQueueMessage(data []byte) *QueueMessage {
	return &QueueMessage{
		ID:        generateID(),
		Data:      data,
		Headers:   make(map[string]string),
		CreatedAt: time.Now(),
		Retries:   0,
	}
}

// AddHeader adds a header to the message
func (m *QueueMessage) AddHeader(key, value string) {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
}

// GetHeader gets a header value
func (m *QueueMessage) GetHeader(key string) (string, bool) {
	if m.Headers == nil {
		return "", false
	}
	value, exists := m.Headers[key]
	return value, exists
}

// IncrementRetries increments the retry count
func (m *QueueMessage) IncrementRetries() {
	m.Retries++
}

// generateID generates a unique ID for messages
func generateID() string {
	// Simple timestamp-based ID generation
	// In production, use a more robust ID generation method
	return time.Now().Format("20060102150405.000000")
}
