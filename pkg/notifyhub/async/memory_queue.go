// Package async provides memory-based queue implementation for async message processing
package async

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

// memoryAsyncQueue implements AsyncQueue using in-memory structures
type memoryAsyncQueue struct {
	items          []*QueueItem
	mutex          sync.Mutex
	cond           *sync.Cond
	maxSize        int
	closed         bool
	processedCount int64
	errorCount     int64
	enqueuedCount  int64
	dequeuedCount  int64
	startTime      time.Time
	lastProcessed  time.Time
	logger         logger.Logger
}

// NewMemoryAsyncQueue creates a new memory-based async queue
func NewMemoryAsyncQueue(maxSize int, logger logger.Logger) AsyncQueue {
	queue := &memoryAsyncQueue{
		items:     make([]*QueueItem, 0),
		maxSize:   maxSize,
		logger:    logger,
		startTime: time.Now(),
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

	// Update statistics
	q.enqueuedCount++

	// Update handle status
	if HandleImpl, ok := handle.(*HandleImpl); ok {
		HandleImpl.UpdateStatus(StatusPendingOp, 0.0, nil, nil)
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

	// Update statistics
	q.enqueuedCount += int64(len(msgs))

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

	// Update statistics
	q.dequeuedCount++
	q.lastProcessed = time.Now()

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

// DequeueBatch removes and returns multiple messages from the queue (up to maxItems)
func (q *memoryAsyncQueue) DequeueBatch(ctx context.Context, maxItems int) ([]*QueueItem, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed && len(q.items) == 0 {
		return nil, fmt.Errorf("queue is closed and empty")
	}

	if len(q.items) == 0 {
		return []*QueueItem{}, nil
	}

	// Determine how many items to dequeue
	count := maxItems
	if count > len(q.items) {
		count = len(q.items)
	}

	// Extract items
	items := make([]*QueueItem, count)
	copy(items, q.items[:count])
	q.items = q.items[count:]

	// Update statistics
	q.dequeuedCount += int64(count)
	q.lastProcessed = time.Now()

	q.logger.Debug("Batch dequeued", "count", count, "queue_size", len(q.items))

	// Update handle statuses
	for _, item := range items {
		if handleImpl, ok := item.Handle.(*HandleImpl); ok {
			handleImpl.UpdateStatus(StatusProcessing, 0.1, nil, nil)
		}
	}

	return items, nil
}

// EnqueueBulk adds multiple queue items efficiently
func (q *memoryAsyncQueue) EnqueueBulk(ctx context.Context, items []*QueueItem) error {
	if len(items) == 0 {
		return nil
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	if len(q.items)+len(items) > q.maxSize {
		return fmt.Errorf("bulk operation would exceed queue capacity")
	}

	// Insert all items based on priority
	for _, item := range items {
		inserted := false
		for i, existing := range q.items {
			if item.Priority > existing.Priority {
				q.items = append(q.items[:i], append([]*QueueItem{item}, q.items[i:]...)...)
				inserted = true
				break
			}
		}
		if !inserted {
			q.items = append(q.items, item)
		}

		// Update handle status
		if handleImpl, ok := item.Handle.(*HandleImpl); ok {
			handleImpl.UpdateStatus(StatusPendingOp, 0.0, nil, nil)
		}
	}

	// Update statistics
	q.enqueuedCount += int64(len(items))

	q.logger.Debug("Bulk enqueued", "count", len(items), "queue_size", len(q.items))
	q.cond.Broadcast()
	return nil
}

// GetStatistics returns detailed queue performance metrics
func (q *memoryAsyncQueue) GetStatistics() QueueStatistics {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	uptime := time.Since(q.startTime).Seconds()
	priorityDist := make(map[int]int)

	// Calculate priority distribution
	for _, item := range q.items {
		priorityDist[item.Priority]++
	}

	// Calculate average wait time (simplified - would need more tracking in real implementation)
	avgWaitTime := 0.0
	if len(q.items) > 0 && q.dequeuedCount > 0 {
		avgWaitTime = uptime / float64(q.dequeuedCount) * 1000 // Convert to ms
	}

	lastProcessed := int64(0)
	if !q.lastProcessed.IsZero() {
		lastProcessed = q.lastProcessed.Unix()
	}

	return QueueStatistics{
		TotalEnqueued:        q.enqueuedCount,
		TotalDequeued:        q.dequeuedCount,
		TotalProcessed:       q.processedCount,
		TotalErrors:          q.errorCount,
		CurrentSize:          len(q.items),
		MaxSize:              q.maxSize,
		ScheduledCount:       0, // Memory queue doesn't have scheduling
		AverageWaitTime:      avgWaitTime,
		AverageProcessTime:   0.0, // Would need process time tracking
		PriorityDistribution: priorityDist,
		LastProcessedAt:      lastProcessed,
		Uptime:               int64(uptime),
	}
}

// GetThroughputMetrics returns throughput-related metrics
func (q *memoryAsyncQueue) GetThroughputMetrics() ThroughputMetrics {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	uptime := time.Since(q.startTime).Seconds()
	if uptime == 0 {
		uptime = 1 // Prevent division by zero
	}

	totalMessages := float64(q.processedCount)
	totalErrors := float64(q.errorCount)

	errorRate := 0.0
	successRate := 100.0
	if totalMessages+totalErrors > 0 {
		errorRate = (totalErrors / (totalMessages + totalErrors)) * 100
		successRate = (totalMessages / (totalMessages + totalErrors)) * 100
	}

	return ThroughputMetrics{
		MessagesPerSecond: totalMessages / uptime,
		MessagesPerMinute: totalMessages / uptime * 60,
		MessagesPerHour:   totalMessages / uptime * 3600,
		ErrorRate:         errorRate,
		SuccessRate:       successRate,
		P50ProcessTime:    0.0, // Would need detailed process time tracking
		P95ProcessTime:    0.0,
		P99ProcessTime:    0.0,
	}
}

// GetQueueByPriority returns the count of messages by priority
func (q *memoryAsyncQueue) GetQueueByPriority() map[int]int {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	priorities := make(map[int]int)
	for _, item := range q.items {
		priorities[item.Priority]++
	}
	return priorities
}

// EnqueueWithPriority adds a message with a specific priority (overriding message priority)
func (q *memoryAsyncQueue) EnqueueWithPriority(ctx context.Context, msg *message.Message, handle AsyncHandle, priority int) error {
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
		Priority: priority, // Use provided priority instead of message priority
	}

	// Insert based on priority (higher priority first)
	inserted := false
	for i, existing := range q.items {
		if item.Priority > existing.Priority {
			q.items = append(q.items[:i], append([]*QueueItem{item}, q.items[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.items = append(q.items, item)
	}

	q.logger.Debug("Message enqueued with custom priority", "message_id", msg.ID, "priority", priority, "queue_size", len(q.items))

	// Update statistics
	q.enqueuedCount++

	// Update handle status
	if handleImpl, ok := handle.(*HandleImpl); ok {
		handleImpl.UpdateStatus(StatusPendingOp, 0.0, nil, nil)
	}

	q.cond.Signal()
	return nil
}

// GetHighestPriorityMessage returns the highest priority message without removing it
func (q *memoryAsyncQueue) GetHighestPriorityMessage(ctx context.Context) (*QueueItem, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if len(q.items) == 0 {
		return nil, nil
	}

	// Return a copy of the first item (highest priority)
	item := q.items[0]
	return &QueueItem{
		Message:   item.Message,
		Handle:    item.Handle,
		BatchInfo: item.BatchInfo,
		QueuedAt:  item.QueuedAt,
		Priority:  item.Priority,
	}, nil
}