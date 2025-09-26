// Package async provides true asynchronous processing capabilities for NotifyHub
package async

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

// AsyncHandle represents a handle to a true asynchronous operation
// This replaces the pseudo-async implementation identified in the architecture analysis
type AsyncHandle interface {
	MessageID() string
	Wait(ctx context.Context) (*receipt.Receipt, error)
	Status() AsyncStatus
	Cancel() error
	Result() <-chan AsyncResult
}

// AsyncBatchHandle represents a handle to a batch asynchronous operation
type AsyncBatchHandle interface {
	BatchID() string
	Wait(ctx context.Context) ([]*receipt.Receipt, error)
	Status() AsyncBatchStatus
	Cancel() error
	Progress() <-chan BatchProgress
	Results() <-chan AsyncResult
}

// AsyncStatus represents the status of an asynchronous operation
type AsyncStatus struct {
	MessageID string          `json:"message_id"`
	Status    OperationStatus `json:"status"`
	Progress  float64         `json:"progress"`
	Error     string          `json:"error,omitempty"`
	StartedAt int64           `json:"started_at"`
	UpdatedAt int64           `json:"updated_at"`
}

// AsyncBatchStatus represents the status of a batch asynchronous operation
type AsyncBatchStatus struct {
	BatchID   string          `json:"batch_id"`
	Status    OperationStatus `json:"status"`
	Total     int             `json:"total"`
	Completed int             `json:"completed"`
	Failed    int             `json:"failed"`
	Progress  float64         `json:"progress"`
	StartedAt int64           `json:"started_at"`
	UpdatedAt int64           `json:"updated_at"`
}

// OperationStatus represents the status of an operation
type OperationStatus string

const (
	StatusPending    OperationStatus = "pending"
	StatusProcessing OperationStatus = "processing"
	StatusCompleted  OperationStatus = "completed"
	StatusFailed     OperationStatus = "failed"
	StatusCancelled  OperationStatus = "cancelled"
)

// AsyncResult represents the result of an asynchronous operation
type AsyncResult struct {
	MessageID string           `json:"message_id"`
	Receipt   *receipt.Receipt `json:"receipt,omitempty"`
	Error     error            `json:"error,omitempty"`
}

// BatchProgress represents the progress of a batch operation
type BatchProgress struct {
	BatchID   string  `json:"batch_id"`
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Failed    int     `json:"failed"`
	Progress  float64 `json:"progress"`
}

// HandleImpl implements AsyncHandle for real asynchronous operations
type HandleImpl struct {
	messageID   string
	message     *message.Message
	resultCh    chan AsyncResult
	statusMutex sync.RWMutex
	status      OperationStatus
	progress    float64
	error       error
	result      *receipt.Receipt
	startedAt   int64
	updatedAt   int64
	cancelled   bool
	cancelCh    chan struct{}
	callbacks   *CallbackRegistry
}

// NewAsyncHandle creates a new async handle for a message
func NewAsyncHandle(msg *message.Message, callbacks *CallbackRegistry) AsyncHandle {
	now := time.Now().Unix()
	handle := &HandleImpl{
		messageID: msg.ID,
		message:   msg,
		resultCh:  make(chan AsyncResult, 1),
		status:    StatusPending,
		progress:  0.0,
		startedAt: now,
		updatedAt: now,
		cancelCh:  make(chan struct{}),
		callbacks: callbacks,
	}
	return handle
}

func (h *HandleImpl) MessageID() string {
	return h.messageID
}

func (h *HandleImpl) Wait(ctx context.Context) (*receipt.Receipt, error) {
	h.statusMutex.RLock()
	if h.status == StatusCompleted || h.status == StatusFailed || h.status == StatusCancelled {
		defer h.statusMutex.RUnlock()
		return h.result, h.error
	}
	h.statusMutex.RUnlock()

	select {
	case result := <-h.resultCh:
		return result.Receipt, result.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.cancelCh:
		return nil, fmt.Errorf("operation cancelled")
	}
}

func (h *HandleImpl) Status() AsyncStatus {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()

	errorStr := ""
	if h.error != nil {
		errorStr = h.error.Error()
	}

	return AsyncStatus{
		MessageID: h.messageID,
		Status:    h.status,
		Progress:  h.progress,
		Error:     errorStr,
		StartedAt: h.startedAt,
		UpdatedAt: h.updatedAt,
	}
}

func (h *HandleImpl) Cancel() error {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()

	if h.status == StatusCompleted || h.status == StatusFailed {
		return fmt.Errorf("cannot cancel completed operation")
	}

	if h.status == StatusCancelled {
		return nil // Already cancelled
	}

	h.cancelled = true
	h.status = StatusCancelled
	h.updatedAt = time.Now().Unix()

	close(h.cancelCh)

	// Trigger error callback
	if h.callbacks != nil {
		h.callbacks.TriggerError(h.message, fmt.Errorf("operation cancelled"))
	}

	return nil
}

func (h *HandleImpl) Result() <-chan AsyncResult {
	return h.resultCh
}

// UpdateStatus updates the handle status (internal method for workers)
func (h *HandleImpl) UpdateStatus(status OperationStatus, progress float64, result *receipt.Receipt, err error) {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()

	if h.cancelled {
		return // Don't update if cancelled
	}

	h.status = status
	h.progress = progress
	h.result = result
	h.error = err
	h.updatedAt = time.Now().Unix()

	// Send result to channel if completed
	if status == StatusCompleted || status == StatusFailed {
		select {
		case h.resultCh <- AsyncResult{
			MessageID: h.messageID,
			Receipt:   result,
			Error:     err,
		}:
		default:
			// Channel already has result or is closed
		}

		// Trigger callbacks
		if h.callbacks != nil {
			if err != nil {
				h.callbacks.TriggerError(h.message, err)
			} else if result != nil {
				h.callbacks.TriggerResult(result)
			}
		}
	}
}

// IsCancelled returns whether the operation was cancelled
func (h *HandleImpl) IsCancelled() bool {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()
	return h.cancelled
}

// GetCancelChannel returns the cancel channel for workers
func (h *HandleImpl) GetCancelChannel() <-chan struct{} {
	return h.cancelCh
}

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
		status:     StatusPending,
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

	if h.status == StatusCompleted || h.status == StatusFailed {
		return fmt.Errorf("cannot cancel completed batch operation")
	}

	if h.status == StatusCancelled {
		return nil // Already cancelled
	}

	h.cancelled = true
	h.status = StatusCancelled
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
			h.status = StatusFailed
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

// BatchSummary represents a summary of batch operation results
type BatchSummary struct {
	BatchID     string `json:"batch_id"`
	Total       int    `json:"total"`
	Successful  int    `json:"successful"`
	Failed      int    `json:"failed"`
	CompletedAt int64  `json:"completed_at"`
}

// generateBatchID generates a unique batch ID
// generateBatchID is removed as unused
// func generateBatchID() string {
// 	return fmt.Sprintf("batch_%d", time.Now().UnixNano())
// }
