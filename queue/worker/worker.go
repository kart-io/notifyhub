package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/notifiers"
	"github.com/kart-io/notifyhub/queue/callbacks"
	"github.com/kart-io/notifyhub/queue/core"
	"github.com/kart-io/notifyhub/queue/retry"
)

// MessageSender defines interface for sending messages
type MessageSender interface {
	SendSync(ctx context.Context, message *notifiers.Message, options interface{}) ([]*notifiers.SendResult, error)
}

// Worker processes messages from the queue
type Worker struct {
	queue       core.Queue
	sender      MessageSender
	retryPolicy *retry.RetryPolicy
	concurrency int
	stopCh      chan struct{}
	workers     []chan struct{}
	wg          sync.WaitGroup
	callbacks   *callbacks.CallbackExecutor
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewWorker creates a new queue worker
func NewWorker(queue core.Queue, sender MessageSender, retryPolicy *retry.RetryPolicy, concurrency int) *Worker {
	if retryPolicy == nil {
		retryPolicy = retry.DefaultRetryPolicy()
	}
	if concurrency <= 0 {
		concurrency = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		queue:       queue,
		sender:      sender,
		retryPolicy: retryPolicy,
		concurrency: concurrency,
		stopCh:      make(chan struct{}),
		callbacks:   callbacks.NewCallbackExecutor(),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the queue worker
func (w *Worker) Start(ctx context.Context) error {
	// Start worker goroutines
	for i := 0; i < w.concurrency; i++ {
		workerStop := make(chan struct{})
		w.workers = append(w.workers, workerStop)

		w.wg.Add(1)
		// Use the worker's context that can be cancelled during Stop()
		go w.worker(w.ctx, workerStop, i)
	}
	return nil
}

// Stop stops the queue worker
func (w *Worker) Stop() {
	// Cancel the worker context for graceful shutdown
	w.cancel()

	close(w.stopCh)

	// Stop all workers
	for _, workerStop := range w.workers {
		close(workerStop)
	}

	w.wg.Wait()
}

// worker is the main worker loop
func (w *Worker) worker(ctx context.Context, stopCh chan struct{}, workerID int) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-stopCh:
			return
		default:
			w.processMessage(ctx)
		}
	}
}

// processMessage processes a single message from the queue
func (w *Worker) processMessage(ctx context.Context) {
	// Dequeue message with timeout
	msgCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	queueMsg, err := w.queue.Dequeue(msgCtx)
	cancel()

	if err != nil {
		return // Timeout or context cancelled
	}

	start := time.Now()

	// Process the message
	sendCtx, sendCancel := context.WithTimeout(ctx, 30*time.Second)
	results, err := w.sender.SendSync(sendCtx, queueMsg.Message, queueMsg.Options)
	sendCancel()

	duration := time.Since(start)
	queueMsg.LastResults = results

	hasFailures := err != nil || w.hasFailures(results)

	if hasFailures {
		// Handle failure
		queueMsg.LastError = w.getErrorMessage(err, results)

		// Execute failure callbacks
		w.callbacks.ExecuteCallbacks(ctx, callbacks.CallbackEventFailed, queueMsg, results, err, duration)

		if w.retryPolicy.ShouldRetry(queueMsg.Attempts) {
			// Execute retry callbacks
			w.callbacks.ExecuteCallbacks(ctx, callbacks.CallbackEventRetry, queueMsg, results, err, duration)

			// Schedule retry
			nextRetry := w.retryPolicy.NextRetry(queueMsg.Attempts)
			w.queue.Nack(queueMsg.ID, nextRetry)

			// Re-enqueue after delay (simplified implementation)
			go func() {
				time.Sleep(time.Until(nextRetry))
				w.queue.Enqueue(context.Background(), queueMsg)
			}()
		} else {
			// Max retries exceeded
			w.callbacks.ExecuteCallbacks(ctx, callbacks.CallbackEventMaxRetries, queueMsg, results, err, duration)
			w.queue.Ack(queueMsg.ID)
		}
	} else {
		// Success
		w.callbacks.ExecuteCallbacks(ctx, callbacks.CallbackEventSent, queueMsg, results, nil, duration)
		w.queue.Ack(queueMsg.ID)
	}
}

// hasFailures checks if any send result failed
func (w *Worker) hasFailures(results []*notifiers.SendResult) bool {
	for _, result := range results {
		if !result.Success {
			return true
		}
	}
	return false
}

// getErrorMessage extracts error message from results
func (w *Worker) getErrorMessage(err error, results []*notifiers.SendResult) string {
	if err != nil {
		return err.Error()
	}

	var errors []string
	for _, result := range results {
		if !result.Success {
			errors = append(errors, fmt.Sprintf("%s: %s", result.Platform, result.Error))
		}
	}

	if len(errors) > 0 {
		return fmt.Sprintf("send failures: %v", errors)
	}

	return "unknown error"
}
