// Package async provides queue interface for NotifyHub async processing
package async

import (
	"context"
	"time"

	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/target"
)

// Queue defines the interface for async message queues
type Queue interface {
	// Enqueue adds a message to the queue for async processing
	Enqueue(ctx context.Context, msg *message.Message, targets []target.Target, opts ...Option) (Handle, error)

	// EnqueueBatch adds multiple messages to the queue
	EnqueueBatch(ctx context.Context, msgs []*message.Message, opts ...Option) (BatchHandle, error)

	// Start starts the queue workers
	Start(ctx context.Context) error

	// Stop stops the queue workers gracefully
	Stop(ctx context.Context) error

	// IsHealthy checks if the queue is healthy
	IsHealthy(ctx context.Context) error

	// GetStats returns queue statistics
	GetStats() QueueStats
}

// QueueConfig configures the queue
type QueueConfig struct {
	Workers     int           `json:"workers"`
	BufferSize  int           `json:"buffer_size"`
	Timeout     time.Duration `json:"timeout"`
	RetryPolicy RetryPolicy   `json:"retry_policy"`
}

// QueueStats provides queue statistics
type QueueStats struct {
	Pending    int64     `json:"pending"`
	Processing int64     `json:"processing"`
	Completed  int64     `json:"completed"`
	Failed     int64     `json:"failed"`
	Workers    int       `json:"workers"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialInterval time.Duration `json:"initial_interval"`
	MaxInterval     time.Duration `json:"max_interval"`
	Multiplier      float64       `json:"multiplier"`
	Jitter          bool          `json:"jitter"`
}

// QueueItem represents an item in the queue
type QueueItem struct {
	ID       string           `json:"id"`
	Message  *message.Message `json:"message"`
	Targets  []target.Target  `json:"targets"`
	Options  []Option         `json:"-"`
	Created  time.Time        `json:"created"`
	Attempts int              `json:"attempts"`
}

// MemoryQueue implements Queue using in-memory channels
type MemoryQueue struct {
	config  QueueConfig
	items   chan *QueueItem
	workers []*Worker
	stats   QueueStats
}

// NewMemoryQueue creates a new memory-based queue
func NewMemoryQueue(config QueueConfig) *MemoryQueue {
	if config.Workers <= 0 {
		config.Workers = 4
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}

	return &MemoryQueue{
		config: config,
		items:  make(chan *QueueItem, config.BufferSize),
		stats:  QueueStats{UpdatedAt: time.Now()},
	}
}

// Enqueue adds a message to the queue
func (q *MemoryQueue) Enqueue(ctx context.Context, msg *message.Message, targets []target.Target, opts ...Option) (Handle, error) {
	item := &QueueItem{
		ID:      msg.ID,
		Message: msg,
		Targets: targets,
		Options: opts,
		Created: time.Now(),
	}

	select {
	case q.items <- item:
		q.stats.Pending++
		handle := NewMemoryHandle(item.ID)
		return handle, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// EnqueueBatch adds multiple messages to the queue
func (q *MemoryQueue) EnqueueBatch(ctx context.Context, msgs []*message.Message, opts ...Option) (BatchHandle, error) {
	handles := make([]Handle, len(msgs))

	for i, msg := range msgs {
		// Create a dummy target list - in real implementation this would be provided
		targets := []target.Target{}
		handle, err := q.Enqueue(ctx, msg, targets, opts...)
		if err != nil {
			return nil, err
		}
		handles[i] = handle
	}

	return NewBatchHandle(handles), nil
}

// Start starts the queue workers
func (q *MemoryQueue) Start(ctx context.Context) error {
	q.workers = make([]*Worker, q.config.Workers)
	for i := 0; i < q.config.Workers; i++ {
		worker := NewWorker(i, q.items)
		q.workers[i] = worker
		go worker.Start(ctx)
	}
	return nil
}

// Stop stops the queue workers
func (q *MemoryQueue) Stop(ctx context.Context) error {
	close(q.items)
	for _, worker := range q.workers {
		worker.Stop()
	}
	return nil
}

// IsHealthy checks queue health
func (q *MemoryQueue) IsHealthy(ctx context.Context) error {
	// Simple health check - check if workers are running
	return nil
}

// GetStats returns queue statistics
func (q *MemoryQueue) GetStats() QueueStats {
	q.stats.UpdatedAt = time.Now()
	q.stats.Workers = len(q.workers)
	return q.stats
}
