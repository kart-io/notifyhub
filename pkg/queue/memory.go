// Package queue provides memory-based queue implementation
package queue

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// memoryQueue implements an in-memory queue with priority and delay support
type memoryQueue struct {
	messages        chan *Message
	priorityHeap    *priorityHeap
	delayedMessages *delayHeap
	capacity        int
	closed          int32
	stats           *QueueStats
	deadLetterQueue Queue
	retryPolicy     RetryPolicy
	metrics         *MetricsCollector
	mutex           sync.RWMutex
	stopCh          chan struct{}
	wg              sync.WaitGroup
	logger          logger.Logger
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue(capacity int, log logger.Logger) Queue {
	if log == nil {
		log = logger.Discard
	}

	q := &memoryQueue{
		messages:        make(chan *Message, capacity),
		priorityHeap:    newPriorityHeap(),
		delayedMessages: newDelayHeap(),
		capacity:        capacity,
		stats: &QueueStats{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		stopCh:  make(chan struct{}),
		logger:  log,
		metrics: NewMetricsCollector(),
		retryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			MaxElapsedTime:  5 * time.Minute,
		},
	}

	// Start background processor for delayed messages
	q.wg.Add(1)
	go q.processDelayedMessages()

	log.Info("Memory queue created", "capacity", capacity)
	return q
}

// Enqueue adds a message to the queue
func (q *memoryQueue) Enqueue(ctx context.Context, msg *Message) error {
	if atomic.LoadInt32(&q.closed) == 1 {
		q.logger.Error("Attempted to enqueue to closed queue")
		return ErrQueueClosed
	}

	if msg == nil {
		q.logger.Error("Attempted to enqueue nil message")
		return ErrInvalidMessage
	}

	// Set timestamps if not set
	now := time.Now()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	msg.UpdatedAt = now

	// Handle delayed messages
	if msg.ScheduledAt != nil && msg.ScheduledAt.After(now) {
		q.mutex.Lock()
		heap.Push(q.delayedMessages, msg)
		q.mutex.Unlock()
		atomic.AddInt64(&q.stats.EnqueuedCount, 1)
		q.logger.Debug("Message enqueued for delayed processing", "messageID", msg.ID, "scheduledAt", msg.ScheduledAt)
		return nil
	}

	// Handle priority messages
	if msg.Priority > PriorityNormal {
		q.mutex.Lock()
		heap.Push(q.priorityHeap, msg)
		q.mutex.Unlock()
		atomic.AddInt64(&q.stats.EnqueuedCount, 1)
		q.logger.Debug("High priority message enqueued", "messageID", msg.ID, "priority", msg.Priority)
		return nil
	}

	// Regular enqueue
	select {
	case q.messages <- msg:
		atomic.AddInt64(&q.stats.EnqueuedCount, 1)
		q.logger.Debug("Message enqueued", "messageID", msg.ID)
		q.metrics.RecordEnqueue(msg)
		q.metrics.UpdateQueueSize(q.Size())
		return nil
	case <-ctx.Done():
		q.logger.Warn("Enqueue cancelled", "messageID", msg.ID)
		return ctx.Err()
	default:
		q.logger.Error("Queue is full", "capacity", q.capacity)
		return ErrQueueFull
	}
}

// EnqueueBatch adds multiple messages to the queue
func (q *memoryQueue) EnqueueBatch(ctx context.Context, msgs []*Message) error {
	if atomic.LoadInt32(&q.closed) == 1 {
		return ErrQueueClosed
	}

	q.logger.Debug("Enqueueing batch", "count", len(msgs))

	var errors []error
	for _, msg := range msgs {
		if err := q.Enqueue(ctx, msg); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		q.logger.Error("Batch enqueue had errors", "errorCount", len(errors))
		return fmt.Errorf("failed to enqueue %d messages", len(errors))
	}

	return nil
}

// Dequeue retrieves and removes a message from the queue
func (q *memoryQueue) Dequeue(ctx context.Context) (*Message, error) {
	if atomic.LoadInt32(&q.closed) == 1 {
		return nil, ErrQueueClosed
	}

	// Check priority queue first
	q.mutex.Lock()
	if q.priorityHeap.Len() > 0 {
		msg := heap.Pop(q.priorityHeap).(*Message)
		q.mutex.Unlock()
		atomic.AddInt64(&q.stats.DequeuedCount, 1)
		q.logger.Debug("Priority message dequeued", "messageID", msg.ID, "priority", msg.Priority)
		return msg, nil
	}
	q.mutex.Unlock()

	// Regular dequeue
	select {
	case msg := <-q.messages:
		atomic.AddInt64(&q.stats.DequeuedCount, 1)
		q.logger.Debug("Message dequeued", "messageID", msg.ID)
		q.metrics.RecordDequeue(msg)
		q.metrics.UpdateQueueSize(q.Size())
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, ErrQueueEmpty
	}
}

// DequeueBatch retrieves and removes multiple messages from the queue
func (q *memoryQueue) DequeueBatch(ctx context.Context, count int) ([]*Message, error) {
	if atomic.LoadInt32(&q.closed) == 1 {
		return nil, ErrQueueClosed
	}

	q.logger.Debug("Dequeueing batch", "requestedCount", count)

	messages := make([]*Message, 0, count)
	for i := 0; i < count; i++ {
		msg, err := q.Dequeue(ctx)
		if err != nil {
			if err == ErrQueueEmpty && len(messages) > 0 {
				// Partial batch is okay
				break
			}
			if len(messages) == 0 {
				return nil, err
			}
			break
		}
		messages = append(messages, msg)
	}

	q.logger.Debug("Batch dequeued", "count", len(messages))
	return messages, nil
}

// Peek retrieves a message without removing it from the queue
func (q *memoryQueue) Peek(ctx context.Context) (*Message, error) {
	if atomic.LoadInt32(&q.closed) == 1 {
		return nil, ErrQueueClosed
	}

	// Check priority queue first
	q.mutex.RLock()
	if q.priorityHeap.Len() > 0 {
		msg := (*q.priorityHeap)[0]
		q.mutex.RUnlock()
		return msg, nil
	}
	q.mutex.RUnlock()

	// Peek at regular queue
	select {
	case msg := <-q.messages:
		// Put it back
		select {
		case q.messages <- msg:
			return msg, nil
		default:
			// Queue became full in the meantime, should not happen
			return msg, nil
		}
	default:
		return nil, ErrQueueEmpty
	}
}

// Size returns the number of messages in the queue
func (q *memoryQueue) Size() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.messages) + q.priorityHeap.Len() + q.delayedMessages.Len()
}

// IsEmpty returns true if the queue has no messages
func (q *memoryQueue) IsEmpty() bool {
	return q.Size() == 0
}

// Clear removes all messages from the queue
func (q *memoryQueue) Clear() error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Clear regular queue
	for len(q.messages) > 0 {
		<-q.messages
	}

	// Clear priority heap
	q.priorityHeap = newPriorityHeap()

	// Clear delayed messages
	q.delayedMessages = newDelayHeap()

	q.logger.Info("Queue cleared")
	return nil
}

// Close closes the queue and releases resources
func (q *memoryQueue) Close() error {
	if !atomic.CompareAndSwapInt32(&q.closed, 0, 1) {
		return nil // Already closed
	}

	q.logger.Info("Closing memory queue")

	close(q.stopCh)
	q.wg.Wait()

	close(q.messages)

	if q.deadLetterQueue != nil {
		_ = q.deadLetterQueue.Close()
	}

	q.logger.Info("Memory queue closed")
	return nil
}

// processDelayedMessages processes delayed messages in the background
func (q *memoryQueue) processDelayedMessages() {
	defer q.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopCh:
			return
		case <-ticker.C:
			q.processReadyDelayedMessages()
		}
	}
}

// processReadyDelayedMessages moves ready delayed messages to the main queue
func (q *memoryQueue) processReadyDelayedMessages() {
	now := time.Now()
	q.mutex.Lock()
	defer q.mutex.Unlock()

	for q.delayedMessages.Len() > 0 {
		msg := (*q.delayedMessages)[0]
		if msg.ScheduledAt != nil && msg.ScheduledAt.After(now) {
			break // Not ready yet
		}

		heap.Pop(q.delayedMessages)
		msg.ScheduledAt = nil // Clear scheduled time

		// Move to appropriate queue based on priority
		if msg.Priority > PriorityNormal {
			heap.Push(q.priorityHeap, msg)
		} else {
			select {
			case q.messages <- msg:
				q.logger.Debug("Delayed message ready for processing", "messageID", msg.ID)
			default:
				// Queue is full, put back in delayed queue
				msg.ScheduledAt = &now
				heap.Push(q.delayedMessages, msg)
				q.logger.Warn("Queue full, delayed message postponed", "messageID", msg.ID)
				break
			}
		}
	}
}

// Priority heap implementation
type priorityHeap []*Message

func newPriorityHeap() *priorityHeap {
	h := &priorityHeap{}
	heap.Init(h)
	return h
}

func (h priorityHeap) Len() int { return len(h) }

func (h priorityHeap) Less(i, j int) bool {
	// Higher priority first
	if h[i].Priority != h[j].Priority {
		return h[i].Priority > h[j].Priority
	}
	// Earlier timestamp first for same priority
	return h[i].Timestamp.Before(h[j].Timestamp)
}

func (h priorityHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *priorityHeap) Push(x interface{}) {
	*h = append(*h, x.(*Message))
}

func (h *priorityHeap) Pop() interface{} {
	old := *h
	n := len(old)
	msg := old[n-1]
	*h = old[0 : n-1]
	return msg
}

// Delay heap implementation
type delayHeap []*Message

func newDelayHeap() *delayHeap {
	h := &delayHeap{}
	heap.Init(h)
	return h
}

func (h delayHeap) Len() int { return len(h) }

func (h delayHeap) Less(i, j int) bool {
	// Earlier scheduled time first
	if h[i].ScheduledAt == nil {
		return true
	}
	if h[j].ScheduledAt == nil {
		return false
	}
	return h[i].ScheduledAt.Before(*h[j].ScheduledAt)
}

func (h delayHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *delayHeap) Push(x interface{}) {
	*h = append(*h, x.(*Message))
}

func (h *delayHeap) Pop() interface{} {
	old := *h
	n := len(old)
	msg := old[n-1]
	*h = old[0 : n-1]
	return msg
}

// GetStats returns queue statistics (ObservableQueue interface)
func (q *memoryQueue) GetStats() *QueueStats {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	return &QueueStats{
		Size:            q.Size(),
		EnqueuedCount:   atomic.LoadInt64(&q.stats.EnqueuedCount),
		DequeuedCount:   atomic.LoadInt64(&q.stats.DequeuedCount),
		RetryCount:      q.stats.RetryCount,
		DeadLetterCount: q.stats.DeadLetterCount,
		AverageWaitTime: q.stats.AverageWaitTime,
		CreatedAt:       q.stats.CreatedAt,
		UpdatedAt:       time.Now(),
	}
}

// GetMetrics returns detailed queue metrics (ObservableQueue interface)
func (q *memoryQueue) GetMetrics() *QueueMetrics {
	return q.metrics.GetQueueMetrics()
}

// Subscribe registers a callback for queue events (ObservableQueue interface)
func (q *memoryQueue) Subscribe(event string, callback func(msg *Message)) {
	q.metrics.Subscribe(event, callback)
}

// Unsubscribe removes a callback for queue events (ObservableQueue interface)
func (q *memoryQueue) Unsubscribe(event string) {
	q.metrics.Unsubscribe(event)
}