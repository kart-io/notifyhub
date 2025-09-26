// Package async provides queue management for asynchronous message processing
package async

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

// AsyncQueue represents a queue for asynchronous message processing
type AsyncQueue interface {
	// Enqueue adds a message to the queue
	Enqueue(ctx context.Context, msg *message.Message, handle AsyncHandle) error

	// EnqueueBatch adds multiple messages to the queue as a batch
	EnqueueBatch(ctx context.Context, msgs []*message.Message, batchHandle AsyncBatchHandle) error

	// Dequeue removes and returns the next message from the queue
	Dequeue(ctx context.Context) (*QueueItem, error)

	// Size returns the current queue size
	Size() int

	// IsEmpty returns whether the queue is empty
	IsEmpty() bool

	// Close shuts down the queue
	Close() error

	// Health returns queue health status
	Health() QueueHealth
}

// QueueItem represents an item in the async queue
type QueueItem struct {
	Message   *message.Message
	Handle    AsyncHandle
	BatchInfo *BatchInfo
	QueuedAt  time.Time
	Priority  int
}

// BatchInfo contains batch-related information for queue items
type BatchInfo struct {
	BatchID     string
	BatchHandle AsyncBatchHandle
	BatchIndex  int
	TotalItems  int
}

// QueueHealth represents the health status of a queue
type QueueHealth struct {
	Healthy        bool  `json:"healthy"`
	Size           int   `json:"size"`
	MaxSize        int   `json:"max_size"`
	ProcessedCount int64 `json:"processed_count"`
	ErrorCount     int64 `json:"error_count"`
}

// memoryAsyncQueue implements AsyncQueue using in-memory structures
type memoryAsyncQueue struct {
	items          []*QueueItem
	mutex          sync.Mutex
	cond           *sync.Cond
	maxSize        int
	closed         bool
	processedCount int64
	errorCount     int64
	logger         logger.Logger
}

// NewMemoryAsyncQueue creates a new memory-based async queue
func NewMemoryAsyncQueue(maxSize int, logger logger.Logger) AsyncQueue {
	queue := &memoryAsyncQueue{
		items:   make([]*QueueItem, 0),
		maxSize: maxSize,
		logger:  logger,
	}
	queue.cond = sync.NewCond(&queue.mutex)
	return queue
}

func (q *memoryAsyncQueue) Enqueue(ctx context.Context, msg *message.Message, handle AsyncHandle) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	if len(q.items) >= q.maxSize {
		return fmt.Errorf("queue is full (size: %d)", q.maxSize)
	}

	item := &QueueItem{
		Message:  msg,
		Handle:   handle,
		QueuedAt: time.Now(),
		Priority: int(msg.Priority),
	}

	// Insert based on priority (higher priority first)
	inserted := false
	for i, existing := range q.items {
		if item.Priority > existing.Priority {
			// Insert at position i
			q.items = append(q.items[:i], append([]*QueueItem{item}, q.items[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.items = append(q.items, item)
	}

	q.logger.Debug("Message enqueued", "message_id", msg.ID, "queue_size", len(q.items), "priority", item.Priority)

	// Update handle status
	if HandleImpl, ok := handle.(*HandleImpl); ok {
		HandleImpl.UpdateStatus(StatusPending, 0.0, nil, nil)
	}

	q.cond.Signal()
	return nil
}

func (q *memoryAsyncQueue) EnqueueBatch(ctx context.Context, msgs []*message.Message, batchHandle AsyncBatchHandle) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	if len(q.items)+len(msgs) > q.maxSize {
		return fmt.Errorf("batch would exceed queue capacity")
	}

	batchID := batchHandle.BatchID()

	// Get individual handles from batch handle
	var handles []AsyncHandle
	if batchImpl, ok := batchHandle.(*BatchHandleImpl); ok {
		handles = batchImpl.GetHandles()
	} else {
		return fmt.Errorf("invalid batch handle type")
	}

	// Create queue items for each message
	for i, msg := range msgs {
		if i >= len(handles) {
			return fmt.Errorf("mismatch between messages and handles")
		}

		item := &QueueItem{
			Message:  msg,
			Handle:   handles[i],
			QueuedAt: time.Now(),
			Priority: int(msg.Priority),
			BatchInfo: &BatchInfo{
				BatchID:     batchID,
				BatchHandle: batchHandle,
				BatchIndex:  i,
				TotalItems:  len(msgs),
			},
		}

		// Insert based on priority
		inserted := false
		for j, existing := range q.items {
			if item.Priority > existing.Priority {
				q.items = append(q.items[:j], append([]*QueueItem{item}, q.items[j:]...)...)
				inserted = true
				break
			}
		}
		if !inserted {
			q.items = append(q.items, item)
		}
	}

	q.logger.Debug("Batch enqueued", "batch_id", batchID, "count", len(msgs), "queue_size", len(q.items))
	q.cond.Broadcast()
	return nil
}

func (q *memoryAsyncQueue) Dequeue(ctx context.Context) (*QueueItem, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	for len(q.items) == 0 && !q.closed {
		// Check context cancellation before waiting
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Wait for signal or context cancellation
		// We need to unlock while waiting and relock when woken up
		q.mutex.Unlock()

		// Use a channel to coordinate between wait and context cancellation
		waitDone := make(chan struct{})
		go func() {
			q.mutex.Lock()
			q.cond.Wait() // This unlocks and relocks the mutex
			q.mutex.Unlock()
			close(waitDone)
		}()

		select {
		case <-ctx.Done():
			// Context cancelled, relock and return error
			q.mutex.Lock()
			return nil, ctx.Err()
		case <-waitDone:
			// Wake up from cond.Wait(), relock and continue
			q.mutex.Lock()
		}
	}

	if q.closed && len(q.items) == 0 {
		return nil, fmt.Errorf("queue is closed and empty")
	}

	// Remove first item (highest priority)
	item := q.items[0]
	q.items = q.items[1:]

	q.logger.Debug("Message dequeued", "message_id", item.Message.ID, "queue_size", len(q.items))

	// Update handle status
	if HandleImpl, ok := item.Handle.(*HandleImpl); ok {
		HandleImpl.UpdateStatus(StatusProcessing, 0.1, nil, nil)
	}

	return item, nil
}

func (q *memoryAsyncQueue) Size() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return len(q.items)
}

func (q *memoryAsyncQueue) IsEmpty() bool {
	return q.Size() == 0
}

func (q *memoryAsyncQueue) Close() error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	q.logger.Info("Async queue closed", "remaining_items", len(q.items))

	// Cancel all remaining items
	for _, item := range q.items {
		_ = item.Handle.Cancel() // Ignore cancel errors in cleanup
	}
	q.items = nil

	q.cond.Broadcast()
	return nil
}

func (q *memoryAsyncQueue) Health() QueueHealth {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	return QueueHealth{
		Healthy:        !q.closed,
		Size:           len(q.items),
		MaxSize:        q.maxSize,
		ProcessedCount: q.processedCount,
		ErrorCount:     q.errorCount,
	}
}

// IncrementProcessed increments the processed count
func (q *memoryAsyncQueue) IncrementProcessed() {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.processedCount++
}

// IncrementError increments the error count
func (q *memoryAsyncQueue) IncrementError() {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.errorCount++
}

// QueueFactory creates queues based on configuration
type QueueFactory struct {
	logger logger.Logger
}

// NewQueueFactory creates a new queue factory
func NewQueueFactory(logger logger.Logger) *QueueFactory {
	return &QueueFactory{logger: logger}
}

// CreateQueue creates a queue based on the given configuration
func (f *QueueFactory) CreateQueue(queueType string, config map[string]interface{}) (AsyncQueue, error) {
	switch queueType {
	case "memory":
		maxSize := 1000 // default
		if size, ok := config["max_size"].(int); ok {
			maxSize = size
		}
		return NewMemoryAsyncQueue(maxSize, f.logger), nil

	case "redis":
		// TODO: Implement Redis queue
		return nil, fmt.Errorf("redis queue not yet implemented")

	default:
		return nil, fmt.Errorf("unknown queue type: %s", queueType)
	}
}
