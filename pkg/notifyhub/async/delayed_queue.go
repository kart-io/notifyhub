// Package async provides delayed queue implementation for scheduled message processing
package async

import (
	"context"
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

// DelayedQueueItem represents an item in the delayed queue
type DelayedQueueItem struct {
	*QueueItem
	ScheduledAt time.Time
	Index       int // index in the heap
}

// DelayedAsyncQueue implements AsyncQueue with scheduling support
type DelayedAsyncQueue struct {
	underlying  AsyncQueue           // The actual queue for ready messages
	heap        *DelayedMessageHeap  // Min-heap for scheduled messages
	mutex       sync.RWMutex
	logger      logger.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	closed      bool
	scheduler   *MessageScheduler
}

// DelayedMessageHeap implements heap.Interface for scheduled messages
type DelayedMessageHeap []*DelayedQueueItem

func (h DelayedMessageHeap) Len() int           { return len(h) }
func (h DelayedMessageHeap) Less(i, j int) bool { return h[i].ScheduledAt.Before(h[j].ScheduledAt) }
func (h DelayedMessageHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *DelayedMessageHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*DelayedQueueItem)
	item.Index = n
	*h = append(*h, item)
}

func (h *DelayedMessageHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*h = old[0 : n-1]
	return item
}

// NewDelayedAsyncQueue creates a new delayed queue wrapping an underlying queue
func NewDelayedAsyncQueue(underlying AsyncQueue, logger logger.Logger) *DelayedAsyncQueue {
	ctx, cancel := context.WithCancel(context.Background())

	heapImpl := &DelayedMessageHeap{}
	heap := heapImpl

	queue := &DelayedAsyncQueue{
		underlying: underlying,
		heap:       heap,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}

	queue.scheduler = &MessageScheduler{
		delayedQueue: queue,
		logger:       logger,
		tickInterval: 100 * time.Millisecond, // Check every 100ms
	}

	// Start the scheduler
	queue.wg.Add(1)
	go queue.scheduler.run(ctx, &queue.wg)

	return queue
}

func (q *DelayedAsyncQueue) Enqueue(ctx context.Context, msg *message.Message, handle AsyncHandle) error {
	return q.EnqueueDelayed(ctx, msg, handle, time.Time{})
}

func (q *DelayedAsyncQueue) EnqueueBatch(ctx context.Context, msgs []*message.Message, batchHandle AsyncBatchHandle) error {
	return q.EnqueueBatchDelayed(ctx, msgs, batchHandle, time.Time{})
}

// EnqueueDelayed adds a message to be processed at a specific time
func (q *DelayedAsyncQueue) EnqueueDelayed(ctx context.Context, msg *message.Message, handle AsyncHandle, scheduledAt time.Time) error {
	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	// If not scheduled for future, enqueue immediately
	if scheduledAt.IsZero() || scheduledAt.Before(time.Now()) {
		return q.underlying.Enqueue(ctx, msg, handle)
	}

	// Add to delayed heap
	q.mutex.Lock()
	defer q.mutex.Unlock()

	item := &DelayedQueueItem{
		QueueItem: &QueueItem{
			Message:  msg,
			Handle:   handle,
			QueuedAt: time.Now(),
			Priority: int(msg.Priority),
		},
		ScheduledAt: scheduledAt,
	}

	heap.Push(q.heap, item)

	q.logger.Debug("Message scheduled for delayed processing",
		"message_id", msg.ID,
		"scheduled_at", scheduledAt,
		"delay", scheduledAt.Sub(time.Now()))

	// Update handle status
	if handleImpl, ok := handle.(*HandleImpl); ok {
		handleImpl.UpdateStatus(StatusPendingOp, 0.0, nil, nil)
	}

	return nil
}

// EnqueueBatchDelayed adds a batch of messages to be processed at a specific time
func (q *DelayedAsyncQueue) EnqueueBatchDelayed(ctx context.Context, msgs []*message.Message, batchHandle AsyncBatchHandle, scheduledAt time.Time) error {
	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	// If not scheduled for future, enqueue immediately
	if scheduledAt.IsZero() || scheduledAt.Before(time.Now()) {
		return q.underlying.EnqueueBatch(ctx, msgs, batchHandle)
	}

	batchID := batchHandle.BatchID()

	// Get individual handles from batch handle
	var handles []AsyncHandle
	if batchImpl, ok := batchHandle.(*BatchHandleImpl); ok {
		handles = batchImpl.GetHandles()
	} else {
		return fmt.Errorf("invalid batch handle type")
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Add all messages to delayed heap
	for i, msg := range msgs {
		if i >= len(handles) {
			return fmt.Errorf("mismatch between messages and handles")
		}

		item := &DelayedQueueItem{
			QueueItem: &QueueItem{
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
			},
			ScheduledAt: scheduledAt,
		}

		heap.Push(q.heap, item)
	}

	q.logger.Debug("Batch scheduled for delayed processing",
		"batch_id", batchID,
		"count", len(msgs),
		"scheduled_at", scheduledAt,
		"delay", scheduledAt.Sub(time.Now()))

	return nil
}

func (q *DelayedAsyncQueue) Dequeue(ctx context.Context) (*QueueItem, error) {
	// Delegate to underlying queue
	return q.underlying.Dequeue(ctx)
}

func (q *DelayedAsyncQueue) Size() int {
	q.mutex.RLock()
	delayedCount := q.heap.Len()
	q.mutex.RUnlock()

	return q.underlying.Size() + delayedCount
}

func (q *DelayedAsyncQueue) IsEmpty() bool {
	return q.Size() == 0
}

func (q *DelayedAsyncQueue) Close() error {
	if q.closed {
		return nil
	}

	q.closed = true
	q.cancel()
	q.wg.Wait()

	q.logger.Info("Delayed async queue closed")

	// Cancel all delayed messages
	q.mutex.Lock()
	for q.heap.Len() > 0 {
		item := heap.Pop(q.heap).(*DelayedQueueItem)
		_ = item.Handle.Cancel() // Ignore cancel errors in cleanup
	}
	q.mutex.Unlock()

	return q.underlying.Close()
}

func (q *DelayedAsyncQueue) Health() QueueHealth {
	underlyingHealth := q.underlying.Health()

	q.mutex.RLock()
	delayedCount := q.heap.Len()
	q.mutex.RUnlock()

	return QueueHealth{
		Healthy:        underlyingHealth.Healthy && !q.closed,
		Size:           underlyingHealth.Size + delayedCount,
		MaxSize:        underlyingHealth.MaxSize,
		ProcessedCount: underlyingHealth.ProcessedCount,
		ErrorCount:     underlyingHealth.ErrorCount,
	}
}

// GetScheduledCount returns the number of scheduled messages
func (q *DelayedAsyncQueue) GetScheduledCount() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return q.heap.Len()
}

// GetNextScheduledTime returns the time of the next scheduled message
func (q *DelayedAsyncQueue) GetNextScheduledTime() *time.Time {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	if q.heap.Len() == 0 {
		return nil
	}

	nextTime := (*q.heap)[0].ScheduledAt
	return &nextTime
}

// Note: MessageScheduler implementation moved to scheduler.go