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

// Handle represents a handle to a true asynchronous operation
// This interface follows the design document specification for async handle management
type Handle interface {
	// Status query methods
	ID() string
	Status() Status
	Result() (*receipt.Receipt, error)

	// Control operations
	Cancel() error
	Wait(ctx context.Context) (*receipt.Receipt, error)

	// Callback management (fluent interface)
	OnComplete(callback CompletionCallback) Handle
	OnError(callback ErrorCallback) Handle
	OnProgress(callback ProgressCallback) Handle
}

// AsyncHandle represents a handle to a true asynchronous operation
// This replaces the pseudo-async implementation identified in the architecture analysis
type AsyncHandle interface {
	MessageID() string
	Wait(ctx context.Context) (*receipt.Receipt, error)
	AsyncStatus() AsyncStatus  // Renamed to avoid conflict with Handle.Status()
	Cancel() error
	ResultChannel() <-chan AsyncResult  // Renamed to avoid conflict with Handle.Result()
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

// Status represents the status of an asynchronous operation (design document spec)
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// OperationStatus represents the status of an operation (backward compatibility)
type OperationStatus string

const (
	StatusPendingOp    OperationStatus = "pending"
	StatusProcessing   OperationStatus = "processing"
	StatusCompleted    OperationStatus = "completed"
	StatusFailedOp     OperationStatus = "failed"
	StatusCancelledOp  OperationStatus = "cancelled"
)

// Callback function types as specified in design document
type CompletionCallback func(*receipt.Receipt)
type ErrorCallback func(*message.Message, error)
type ProgressCallback func(completed, total int)

// BatchSummary represents a summary of batch operation completion
type BatchSummary struct {
	BatchID     string `json:"batch_id"`
	Total       int    `json:"total"`
	Successful  int    `json:"successful"`
	Failed      int    `json:"failed"`
	CompletedAt int64  `json:"completed_at"`
}

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

// HandleImpl implements both Handle and AsyncHandle interfaces for real asynchronous operations
type HandleImpl struct {
	messageID   string
	message     *message.Message
	resultCh    chan AsyncResult
	statusMutex sync.RWMutex
	status      Status                // Using new Status type for Handle interface
	oldStatus   OperationStatus       // Backward compatibility for AsyncHandle
	progress    float64
	error       error
	result      *receipt.Receipt
	startedAt   int64
	updatedAt   int64
	cancelled   bool
	cancelCh    chan struct{}
	callbacks   *CallbackRegistry

	// Handle-specific callback fields
	completionCallback CompletionCallback
	errorCallback      ErrorCallback
	progressCallback   ProgressCallback

	// Handle lifecycle management
	waiters     []chan bool
	waitersMux  sync.Mutex
	done        bool
	doneCh      chan struct{}
}

// NewHandle creates a new handle for a message following design document specification
func NewHandle(msg *message.Message, callbacks *CallbackRegistry) Handle {
	now := time.Now().Unix()
	handle := &HandleImpl{
		messageID:  msg.ID,
		message:    msg,
		resultCh:   make(chan AsyncResult, 1),
		status:     StatusPending,
		oldStatus:  StatusPendingOp,
		progress:   0.0,
		startedAt:  now,
		updatedAt:  now,
		cancelCh:   make(chan struct{}),
		doneCh:     make(chan struct{}),
		callbacks:  callbacks,
		waiters:    make([]chan bool, 0),
		done:       false,
	}
	return handle
}

// NewAsyncHandle creates a new async handle for a message (backward compatibility)
func NewAsyncHandle(msg *message.Message, callbacks *CallbackRegistry) AsyncHandle {
	// Reuse the new HandleImpl since it implements both interfaces
	return NewHandle(msg, callbacks).(*HandleImpl)
}

// Handle interface implementation

// ID returns the unique identifier for this operation
func (h *HandleImpl) ID() string {
	return h.messageID
}

// Status returns the current status of the operation
func (h *HandleImpl) Status() Status {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()
	return h.status
}

// Result returns the current result if available, or nil if not yet completed
func (h *HandleImpl) Result() (*receipt.Receipt, error) {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()

	if h.status == StatusSuccess || h.status == StatusFailed || h.status == StatusCancelled {
		return h.result, h.error
	}
	return nil, nil // Not yet completed
}

// OnComplete registers a completion callback and returns the handle for chaining
func (h *HandleImpl) OnComplete(callback CompletionCallback) Handle {
	h.statusMutex.Lock()
	h.completionCallback = callback
	h.statusMutex.Unlock()

	// If already completed, trigger callback immediately
	if h.isCompleted() {
		if result, err := h.Result(); err == nil && result != nil {
			go h.safeCallbackExecution(func() {
				callback(result)
			})
		}
	}

	return h
}

// OnError registers an error callback and returns the handle for chaining
func (h *HandleImpl) OnError(callback ErrorCallback) Handle {
	h.statusMutex.Lock()
	h.errorCallback = callback
	h.statusMutex.Unlock()

	// If already failed, trigger callback immediately
	if h.status == StatusFailed || h.status == StatusCancelled {
		if h.error != nil {
			go h.safeCallbackExecution(func() {
				callback(h.message, h.error)
			})
		}
	}

	return h
}

// OnProgress registers a progress callback and returns the handle for chaining
func (h *HandleImpl) OnProgress(callback ProgressCallback) Handle {
	h.statusMutex.Lock()
	h.progressCallback = callback
	h.statusMutex.Unlock()
	return h
}

// AsyncHandle interface implementation (backward compatibility)

func (h *HandleImpl) MessageID() string {
	return h.messageID
}

// Wait blocks until the operation completes or the context is cancelled
// This method supports multiple concurrent waiters
func (h *HandleImpl) Wait(ctx context.Context) (*receipt.Receipt, error) {
	// Check if already completed
	h.statusMutex.RLock()
	if h.done {
		defer h.statusMutex.RUnlock()
		return h.result, h.error
	}
	h.statusMutex.RUnlock()

	// Register as a waiter
	waiterCh := make(chan bool, 1)
	h.waitersMux.Lock()
	if h.done {
		// Completed while we were acquiring lock
		h.waitersMux.Unlock()
		h.statusMutex.RLock()
		defer h.statusMutex.RUnlock()
		return h.result, h.error
	}
	h.waiters = append(h.waiters, waiterCh)
	h.waitersMux.Unlock()

	// Wait for completion or cancellation
	select {
	case <-waiterCh:
		// Operation completed
		h.statusMutex.RLock()
		defer h.statusMutex.RUnlock()
		return h.result, h.error
	case <-ctx.Done():
		// Context cancelled - remove from waiters
		h.removeWaiter(waiterCh)
		return nil, ctx.Err()
	case <-h.cancelCh:
		// Operation cancelled
		return nil, fmt.Errorf("operation cancelled")
	}
}

// AsyncStatus returns the status in AsyncStatus format for backward compatibility
func (h *HandleImpl) AsyncStatus() AsyncStatus {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()

	errorStr := ""
	if h.error != nil {
		errorStr = h.error.Error()
	}

	return AsyncStatus{
		MessageID: h.messageID,
		Status:    h.oldStatus,
		Progress:  h.progress,
		Error:     errorStr,
		StartedAt: h.startedAt,
		UpdatedAt: h.updatedAt,
	}
}

// Cancel cancels the operation if it's not already completed
func (h *HandleImpl) Cancel() error {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()

	if h.status == StatusSuccess || h.status == StatusFailed {
		return fmt.Errorf("cannot cancel completed operation")
	}

	if h.status == StatusCancelled {
		return nil // Already cancelled
	}

	h.cancelled = true
	h.status = StatusCancelled
	h.oldStatus = StatusCancelledOp
	h.error = fmt.Errorf("operation cancelled")
	h.updatedAt = time.Now().Unix()

	// Mark as done and notify waiters
	if !h.done {
		h.done = true
		h.notifyWaiters()
		close(h.doneCh)
	}

	// Close cancel channel (only if not already closed)
	select {
	case <-h.cancelCh:
		// Already closed
	default:
		close(h.cancelCh)
	}

	// Trigger callbacks
	if h.callbacks != nil {
		h.callbacks.TriggerError(h.message, h.error)
	}

	if h.errorCallback != nil {
		go h.safeCallbackExecution(func() {
			h.errorCallback(h.message, h.error)
		})
	}

	return nil
}

// ResultChannel returns the result channel for AsyncHandle interface (backward compatibility)
func (h *HandleImpl) ResultChannel() <-chan AsyncResult {
	return h.resultCh
}

// UpdateStatus updates the handle status (internal method for workers)
func (h *HandleImpl) UpdateStatus(status OperationStatus, progress float64, result *receipt.Receipt, err error) {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()

	if h.cancelled {
		return // Don't update if cancelled
	}

	// Convert old status to new status
	h.oldStatus = status
	switch status {
	case StatusPendingOp:
		h.status = StatusPending
	case StatusProcessing:
		h.status = StatusRunning
	case StatusCompleted:
		h.status = StatusSuccess
	case StatusFailedOp:
		h.status = StatusFailed
	case StatusCancelledOp:
		h.status = StatusCancelled
	}

	h.progress = progress
	h.result = result
	h.error = err
	h.updatedAt = time.Now().Unix()

	// Send result to channel if completed
	if status == StatusCompleted || status == StatusFailedOp {
		select {
		case h.resultCh <- AsyncResult{
			MessageID: h.messageID,
			Receipt:   result,
			Error:     err,
		}:
		default:
			// Channel already has result or is closed
		}

		// Mark as done and notify waiters
		if !h.done {
			h.done = true
			h.notifyWaiters()
			close(h.doneCh)
		}

		// Trigger callbacks
		if h.callbacks != nil {
			if err != nil {
				h.callbacks.TriggerError(h.message, err)
			} else if result != nil {
				h.callbacks.TriggerResult(result)
			}
		}

		// Trigger handle-specific callbacks
		if err != nil && h.errorCallback != nil {
			go h.safeCallbackExecution(func() {
				h.errorCallback(h.message, err)
			})
		} else if result != nil && h.completionCallback != nil {
			go h.safeCallbackExecution(func() {
				h.completionCallback(result)
			})
		}
	}

	// Trigger progress callback if set
	if h.progressCallback != nil && h.progress > 0 {
		go h.safeCallbackExecution(func() {
			h.progressCallback(int(h.progress*100), 100)
		})
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

// Helper methods for handle lifecycle management

// isCompleted checks if the operation is in a completed state
func (h *HandleImpl) isCompleted() bool {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()
	return h.status == StatusSuccess || h.status == StatusFailed || h.status == StatusCancelled
}

// notifyWaiters notifies all waiting goroutines that the operation is complete
func (h *HandleImpl) notifyWaiters() {
	h.waitersMux.Lock()
	defer h.waitersMux.Unlock()

	for _, waiterCh := range h.waiters {
		select {
		case waiterCh <- true:
		default:
			// Channel might be closed or full, skip
		}
	}
	h.waiters = h.waiters[:0] // Clear the waiters slice
}

// removeWaiter removes a specific waiter from the waiters list
func (h *HandleImpl) removeWaiter(waiterCh chan bool) {
	h.waitersMux.Lock()
	defer h.waitersMux.Unlock()

	for i, wch := range h.waiters {
		if wch == waiterCh {
			h.waiters = append(h.waiters[:i], h.waiters[i+1:]...)
			close(waiterCh)
			return
		}
	}
}

// safeCallbackExecution executes a callback with panic recovery
func (h *HandleImpl) safeCallbackExecution(callback func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			// Log panic but don't crash the handle
			if h.callbacks != nil && h.callbacks.logger != nil {
				h.callbacks.logger.Error("Handle callback panicked",
					"message_id", h.messageID,
					"panic", recovered)
			}
		}
	}()

	callback()
}

// Cleanup performs handle cleanup and resource disposal
func (h *HandleImpl) Cleanup() {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()

	// Clean up callbacks from registry
	if h.callbacks != nil {
		h.callbacks.CleanupMessageCallbacks(h.messageID)
	}

	// Mark as done if not already
	if !h.done {
		h.done = true
		h.notifyWaiters()
		close(h.doneCh)
	}

	// Close channels if not already closed
	select {
	case <-h.resultCh:
	default:
		close(h.resultCh)
	}
}

// GetWaiterCount returns the number of goroutines waiting on this handle (for monitoring)
func (h *HandleImpl) GetWaiterCount() int {
	h.waitersMux.Lock()
	defer h.waitersMux.Unlock()
	return len(h.waiters)
}

// IsTimeout checks if the handle has exceeded its timeout
func (h *HandleImpl) IsTimeout(timeout time.Duration) bool {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()

	if h.done || h.cancelled {
		return false // Already completed or cancelled
	}

	return time.Since(time.Unix(h.startedAt, 0)) > timeout
}

// SetRunning marks the handle as running (internal method for workers)
func (h *HandleImpl) SetRunning() {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()

	if h.cancelled {
		return
	}

	h.status = StatusRunning
	h.oldStatus = StatusProcessing
	h.updatedAt = time.Now().Unix()
}

// HandleRegistry manages active handles for lifecycle management and memory leak prevention
type HandleRegistry struct {
	mutex       sync.RWMutex
	handles     map[string]*HandleImpl
	maxHandles  int
	gcInterval  time.Duration
	stopGC      chan struct{}
	gcRunning   bool
}

// NewHandleRegistry creates a new handle registry with garbage collection
func NewHandleRegistry(maxHandles int, gcInterval time.Duration) *HandleRegistry {
	registry := &HandleRegistry{
		handles:    make(map[string]*HandleImpl),
		maxHandles: maxHandles,
		gcInterval: gcInterval,
		stopGC:     make(chan struct{}),
	}

	// Start garbage collection goroutine
	go registry.startGC()

	return registry
}

// Register adds a handle to the registry
func (r *HandleRegistry) Register(handle *HandleImpl) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.handles) >= r.maxHandles {
		// Try to clean up completed handles first
		r.cleanupCompletedHandles()

		if len(r.handles) >= r.maxHandles {
			return fmt.Errorf("handle registry is full (max: %d)", r.maxHandles)
		}
	}

	r.handles[handle.messageID] = handle
	return nil
}

// Get retrieves a handle by ID
func (r *HandleRegistry) Get(id string) (*HandleImpl, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	handle, exists := r.handles[id]
	return handle, exists
}

// Remove removes a handle from the registry
func (r *HandleRegistry) Remove(id string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if handle, exists := r.handles[id]; exists {
		handle.Cleanup()
		delete(r.handles, id)
	}
}

// GetActiveCount returns the number of active handles
func (r *HandleRegistry) GetActiveCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.handles)
}

// CleanupExpiredHandles removes handles that have exceeded the timeout
func (r *HandleRegistry) CleanupExpiredHandles(timeout time.Duration) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	expired := make([]string, 0)
	for id, handle := range r.handles {
		if handle.IsTimeout(timeout) {
			expired = append(expired, id)
		}
	}

	for _, id := range expired {
		if handle, exists := r.handles[id]; exists {
			handle.Cancel() // Cancel and cleanup expired handle
			handle.Cleanup()
			delete(r.handles, id)
		}
	}

	return len(expired)
}

// cleanupCompletedHandles removes completed handles (internal method)
func (r *HandleRegistry) cleanupCompletedHandles() {
	completed := make([]string, 0)
	for id, handle := range r.handles {
		if handle.isCompleted() {
			completed = append(completed, id)
		}
	}

	for _, id := range completed {
		if handle, exists := r.handles[id]; exists {
			handle.Cleanup()
			delete(r.handles, id)
		}
	}
}

// startGC runs the garbage collection loop
func (r *HandleRegistry) startGC() {
	r.mutex.Lock()
	r.gcRunning = true
	r.mutex.Unlock()

	ticker := time.NewTicker(r.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.mutex.Lock()
			r.cleanupCompletedHandles()
			r.mutex.Unlock()
		case <-r.stopGC:
			return
		}
	}
}

// Shutdown stops the registry and cleans up all handles
func (r *HandleRegistry) Shutdown() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.gcRunning {
		close(r.stopGC)
		r.gcRunning = false
	}

	// Cleanup all remaining handles
	for id, handle := range r.handles {
		handle.Cleanup()
		delete(r.handles, id)
	}
}

// GetStats returns registry statistics
func (r *HandleRegistry) GetStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := make(map[string]int)
	for _, handle := range r.handles {
		status := string(handle.Status())
		stats[status]++
	}

	return map[string]interface{}{
		"total_handles": len(r.handles),
		"max_handles":   r.maxHandles,
		"status_counts": stats,
	}
}

// Note: BatchHandleImpl implementation moved to batch_handle.go
