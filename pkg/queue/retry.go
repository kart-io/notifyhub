// Package queue provides retry queue functionality
package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// retryQueue wraps a queue with retry functionality
type retryQueue struct {
	Queue
	deadLetterQueue Queue
	retryPolicy     RetryPolicy
	mutex           sync.RWMutex
	logger          logger.Logger
}

// NewRetryQueue creates a new queue with retry support
func NewRetryQueue(baseQueue Queue, retryPolicy RetryPolicy, log logger.Logger) RetryQueue {
	if log == nil {
		log = logger.Discard
	}

	// Create dead letter queue with same capacity
	var capacity int
	if memQueue, ok := baseQueue.(*memoryQueue); ok {
		capacity = memQueue.capacity
	} else {
		capacity = 1000 // Default capacity
	}

	dlq := NewMemoryQueue(capacity, log)

	return &retryQueue{
		Queue:           baseQueue,
		deadLetterQueue: dlq,
		retryPolicy:     retryPolicy,
		logger:          log,
	}
}

// Retry moves a message back to the queue with incremented retry count
func (rq *retryQueue) Retry(ctx context.Context, msg *QueueMessage) error {
	if msg == nil {
		return ErrInvalidMessage
	}

	rq.mutex.Lock()
	defer rq.mutex.Unlock()

	msg.RetryCount++
	msg.UpdatedAt = time.Now()

	// Check if max retries exceeded
	if msg.MaxRetries == 0 {
		msg.MaxRetries = rq.retryPolicy.MaxRetries
	}

	if msg.RetryCount > msg.MaxRetries {
		rq.logger.Warn("Max retries exceeded, moving to dead letter queue",
			"messageID", msg.ID,
			"retryCount", msg.RetryCount,
			"maxRetries", msg.MaxRetries)
		return rq.MoveToDeadLetter(ctx, msg)
	}

	// Calculate delay based on retry policy
	delay := rq.calculateRetryDelay(msg.RetryCount)

	// Schedule for retry
	scheduledAt := time.Now().Add(delay)
	msg.ScheduledAt = &scheduledAt

	rq.logger.Info("Scheduling message for retry",
		"messageID", msg.ID,
		"retryCount", msg.RetryCount,
		"scheduledAt", scheduledAt)

	return rq.Enqueue(ctx, msg)
}

// MoveToDeadLetter moves a message to the dead letter queue
func (rq *retryQueue) MoveToDeadLetter(ctx context.Context, msg *QueueMessage) error {
	if msg == nil {
		return ErrInvalidMessage
	}

	// Add metadata about why it's in DLQ
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["dlq_reason"] = "max_retries_exceeded"
	msg.Metadata["dlq_timestamp"] = time.Now()
	msg.Metadata["original_retry_count"] = msg.RetryCount

	rq.logger.Info("Moving message to dead letter queue",
		"messageID", msg.ID,
		"retryCount", msg.RetryCount)

	return rq.deadLetterQueue.Enqueue(ctx, msg)
}

// GetDeadLetterQueue returns the dead letter queue
func (rq *retryQueue) GetDeadLetterQueue() Queue {
	return rq.deadLetterQueue
}

// calculateRetryDelay calculates the delay for the next retry
func (rq *retryQueue) calculateRetryDelay(retryCount int) time.Duration {
	if retryCount <= 0 {
		return rq.retryPolicy.InitialInterval
	}

	// Exponential backoff with jitter
	delay := rq.retryPolicy.InitialInterval
	for i := 1; i < retryCount; i++ {
		delay = time.Duration(float64(delay) * rq.retryPolicy.Multiplier)
		if delay > rq.retryPolicy.MaxInterval {
			delay = rq.retryPolicy.MaxInterval
			break
		}
	}

	// Add jitter (±10%)
	jitter := time.Duration(float64(delay) * 0.1)
	delay = delay + time.Duration(time.Now().UnixNano()%int64(jitter))

	return delay
}

// RetryableWorker processes messages with automatic retry
type RetryableWorker struct {
	id          string
	queue       RetryQueue
	handler     MessageHandler
	running     bool
	mutex       sync.RWMutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
	logger      logger.Logger
	maxRetries  int
	concurrency int
}

// NewRetryableWorker creates a new worker with retry support
func NewRetryableWorker(id string, queue RetryQueue, handler MessageHandler, concurrency int, log logger.Logger) Worker {
	if log == nil {
		log = logger.Discard
	}

	if concurrency <= 0 {
		concurrency = 1
	}

	return &RetryableWorker{
		id:          id,
		queue:       queue,
		handler:     handler,
		concurrency: concurrency,
		stopCh:      make(chan struct{}),
		logger:      log,
		maxRetries:  3,
	}
}

// Start begins processing messages
func (w *RetryableWorker) Start(ctx context.Context) error {
	w.mutex.Lock()
	if w.running {
		w.mutex.Unlock()
		return fmt.Errorf("worker %s is already running", w.id)
	}
	w.running = true
	w.mutex.Unlock()

	w.logger.Info("Starting retryable worker", "workerID", w.id, "concurrency", w.concurrency)

	// Start multiple goroutines for concurrent processing
	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.processMessages(ctx, i)
	}

	return nil
}

// Stop stops processing messages
func (w *RetryableWorker) Stop() error {
	w.mutex.Lock()
	if !w.running {
		w.mutex.Unlock()
		return fmt.Errorf("worker %s is not running", w.id)
	}
	w.running = false
	w.mutex.Unlock()

	w.logger.Info("Stopping retryable worker", "workerID", w.id)

	close(w.stopCh)
	w.wg.Wait()

	w.logger.Info("Retryable worker stopped", "workerID", w.id)
	return nil
}

// Process handles a single message
func (w *RetryableWorker) Process(ctx context.Context, msg *QueueMessage) error {
	if msg == nil {
		return ErrInvalidMessage
	}

	w.logger.Debug("Processing message", "workerID", w.id, "messageID", msg.ID)

	// Process with timeout
	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := w.handler(processCtx, msg)
	if err != nil {
		w.logger.Error("Failed to process message",
			"workerID", w.id,
			"messageID", msg.ID,
			"error", err)

		// Retry the message
		if retryErr := w.queue.Retry(ctx, msg); retryErr != nil {
			w.logger.Error("Failed to retry message",
				"workerID", w.id,
				"messageID", msg.ID,
				"error", retryErr)
		}
		return err
	}

	w.logger.Debug("Message processed successfully", "workerID", w.id, "messageID", msg.ID)
	return nil
}

// GetID returns the worker ID
func (w *RetryableWorker) GetID() string {
	return w.id
}

// IsRunning returns true if the worker is running
func (w *RetryableWorker) IsRunning() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.running
}

// processMessages continuously processes messages from the queue
func (w *RetryableWorker) processMessages(ctx context.Context, goroutineID int) {
	defer w.wg.Done()

	w.logger.Debug("Worker goroutine started", "workerID", w.id, "goroutineID", goroutineID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
			// Try to get a message
			msg, err := w.queue.Dequeue(ctx)
			if err != nil {
				if err == ErrQueueEmpty {
					// Queue is empty, wait a bit
					time.Sleep(100 * time.Millisecond)
					continue
				}
				w.logger.Error("Failed to dequeue message",
					"workerID", w.id,
					"goroutineID", goroutineID,
					"error", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Process the message
			if err := w.Process(ctx, msg); err != nil {
				w.logger.Error("Message processing failed",
					"workerID", w.id,
					"goroutineID", goroutineID,
					"messageID", msg.ID,
					"error", err)
			}
		}
	}
}
