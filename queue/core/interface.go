package core

import (
	"context"
	"time"
)

// Queue defines the interface for message queues
type Queue interface {
	// Enqueue adds a message to the queue
	Enqueue(ctx context.Context, msg *Message) (string, error)

	// Dequeue retrieves a message from the queue
	Dequeue(ctx context.Context) (*Message, error)

	// Ack acknowledges successful processing of a message
	Ack(msgID string) error

	// Nack indicates failed processing and schedules retry
	Nack(msgID string, nextRetry time.Time) error

	// Close closes the queue
	Close() error

	// Size returns the current queue size
	Size() int

	// Health checks queue health
	Health(ctx context.Context) error
}
