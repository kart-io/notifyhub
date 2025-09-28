// Package async provides batch handle implementation for batch asynchronous operations
package async

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

// BatchHandleImpl implements AsyncBatchHandle for real batch asynchronous operations
type BatchHandleImpl struct {
	batchID     string
	handles     []AsyncHandle
	progressCh  chan BatchProgress
	resultCh    chan AsyncResult
	statusMutex sync.RWMutex
	status      OperationStatus
	total       int
	completed   int
	failed      int
	progress    float64
	startedAt   int64
	updatedAt   int64
	cancelled   bool
	cancelCh    chan struct{}
	callbacks   *CallbackRegistry
}

// NewAsyncBatchHandle creates a new async batch handle
func NewAsyncBatchHandle(batchID string, messages []*message.Message, callbacks *CallbackRegistry) AsyncBatchHandle {
	now := time.Now().Unix()
	handles := make([]AsyncHandle, len(messages))

	for i, msg := range messages {
		handles[i] = NewAsyncHandle(msg, callbacks)
	}

	handle := &BatchHandleImpl{
		batchID:    batchID,
		handles:    handles,
		progressCh: make(chan BatchProgress, 10),
		resultCh:   make(chan AsyncResult, len(messages)),
		status:     StatusPendingOp,
		total:      len(messages),
		completed:  0,
		failed:     0,
		progress:   0.0,
		startedAt:  now,
		updatedAt:  now,
		cancelCh:   make(chan struct{}),
		callbacks:  callbacks,
	}

	return handle
}

func (h *BatchHandleImpl) BatchID() string {
	return h.batchID
}

func (h *BatchHandleImpl) Wait(ctx context.Context) ([]*receipt.Receipt, error) {
	results := make([]*receipt.Receipt, 0, h.total)
	var lastError error

	for _, handle := range h.handles {
		result, err := handle.Wait(ctx)
		if err != nil {
			lastError = err
		}
		if result != nil {
			results = append(results, result)
		}
	}

	return results, lastError
}

func (h *BatchHandleImpl) Status() AsyncBatchStatus {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()

	return AsyncBatchStatus{
		BatchID:   h.batchID,
		Status:    h.status,
		Total:     h.total,
		Completed: h.completed,
		Failed:    h.failed,
		Progress:  h.progress,
		StartedAt: h.startedAt,
		UpdatedAt: h.updatedAt,
	}
}

func (h *BatchHandleImpl) Cancel() error {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()

	if h.status == StatusCompleted || h.status == StatusFailedOp {
		return fmt.Errorf("cannot cancel completed batch operation")
	}

	if h.status == StatusCancelledOp {
		return nil // Already cancelled
	}

	h.cancelled = true
	h.status = StatusCancelledOp
	h.updatedAt = time.Now().Unix()

	// Cancel all individual handles
	for _, handle := range h.handles {
		_ = handle.Cancel() // Ignore cancel errors in cleanup
	}

	close(h.cancelCh)
	return nil
}

func (h *BatchHandleImpl) Progress() <-chan BatchProgress {
	return h.progressCh
}

func (h *BatchHandleImpl) Results() <-chan AsyncResult {
	return h.resultCh
}

// UpdateProgress updates batch progress (internal method)
func (h *BatchHandleImpl) UpdateProgress(completed, failed int) {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()

	if h.cancelled {
		return
	}

	h.completed = completed
	h.failed = failed
	h.progress = float64(completed+failed) / float64(h.total)
	h.updatedAt = time.Now().Unix()

	// Determine status
	if completed+failed == h.total {
		if failed == h.total {
			h.status = StatusFailedOp
		} else {
			h.status = StatusCompleted
		}

		// Trigger completion callback
		if h.callbacks != nil {
			summary := &BatchSummary{
				BatchID:     h.batchID,
				Total:       h.total,
				Successful:  completed,
				Failed:      failed,
				CompletedAt: h.updatedAt,
			}
			h.callbacks.TriggerComplete(summary)
		}
	} else {
		h.status = StatusProcessing

		// Trigger progress callback
		if h.callbacks != nil {
			h.callbacks.TriggerProgress(completed, h.total)
		}
	}

	// Send progress update
	select {
	case h.progressCh <- BatchProgress{
		BatchID:   h.batchID,
		Total:     h.total,
		Completed: completed,
		Failed:    failed,
		Progress:  h.progress,
	}:
	default:
		// Channel is full, skip this update
	}
}

// GetHandles returns the individual handles (for workers)
func (h *BatchHandleImpl) GetHandles() []AsyncHandle {
	return h.handles
}

