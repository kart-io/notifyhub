// Package async provides asynchronous processing capabilities for NotifyHub
package async

import (
	"context"
	"sync"
	"time"

	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/receipt"
)

// Handle represents an asynchronous operation handle
type Handle interface {
	// Status query
	ID() string
	Status() Status
	Result() <-chan Result

	// Control operations
	Cancel() error
	Wait(ctx context.Context) (*receipt.Receipt, error)

	// Callback management
	OnComplete(callback CompletionCallback) Handle
	OnError(callback ErrorCallback) Handle
	OnProgress(callback ProgressCallback) Handle
}

// BatchHandle represents a batch asynchronous operation handle
type BatchHandle interface {
	// Status query
	BatchID() string
	Status() BatchStatus
	Results() <-chan Result
	Progress() <-chan BatchProgress

	// Control operations
	Cancel() error
	Wait(ctx context.Context) ([]*receipt.Receipt, error)
}

// Status represents the status of an async operation
type Status struct {
	ID        string                 `json:"id"`
	State     OperationState         `json:"state"`
	Progress  float64                `json:"progress"` // 0.0 to 1.0
	StartedAt time.Time              `json:"started_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// BatchStatus represents the status of a batch async operation
type BatchStatus struct {
	BatchID   string         `json:"batch_id"`
	State     OperationState `json:"state"`
	Total     int            `json:"total"`
	Completed int            `json:"completed"`
	Failed    int            `json:"failed"`
	Progress  float64        `json:"progress"`
	StartedAt time.Time      `json:"started_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// BatchProgress represents progress information for batch operations
type BatchProgress struct {
	Completed int     `json:"completed"`
	Total     int     `json:"total"`
	Failed    int     `json:"failed"`
	Progress  float64 `json:"progress"`
}

// OperationState represents the state of an operation
type OperationState string

const (
	StatePending    OperationState = "pending"
	StateProcessing OperationState = "processing"
	StateCompleted  OperationState = "completed"
	StateFailed     OperationState = "failed"
	StateCancelled  OperationState = "cancelled"
)

// Result represents the result of an async operation
type Result struct {
	Receipt *receipt.Receipt `json:"receipt,omitempty"`
	Error   error            `json:"error,omitempty"`
}

// Callback function types are defined in callback.go

// Option represents an async operation option
type Option func(*Options) error

// Options represents async operation options
type Options struct {
	Priority    int                    `json:"priority"`
	Timeout     time.Duration          `json:"timeout"`
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
	Callbacks   *CallbackConfig        `json:"callbacks,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CallbackConfig holds callback configurations
type CallbackConfig struct {
	OnComplete CompletionCallback `json:"-"`
	OnError    ErrorCallback      `json:"-"`
	OnProgress ProgressCallback   `json:"-"`
}

// RetryPolicy is defined in queue.go

// Async option functions
func WithPriority(priority int) Option {
	return func(o *Options) error {
		o.Priority = priority
		return nil
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) error {
		o.Timeout = timeout
		return nil
	}
}

func WithRetryPolicy(policy RetryPolicy) Option {
	return func(o *Options) error {
		o.RetryPolicy = &policy
		return nil
	}
}

func OnComplete(callback CompletionCallback) Option {
	return func(o *Options) error {
		if o.Callbacks == nil {
			o.Callbacks = &CallbackConfig{}
		}
		o.Callbacks.OnComplete = callback
		return nil
	}
}

func OnError(callback ErrorCallback) Option {
	return func(o *Options) error {
		if o.Callbacks == nil {
			o.Callbacks = &CallbackConfig{}
		}
		o.Callbacks.OnError = callback
		return nil
	}
}

func OnProgress(callback ProgressCallback) Option {
	return func(o *Options) error {
		if o.Callbacks == nil {
			o.Callbacks = &CallbackConfig{}
		}
		o.Callbacks.OnProgress = callback
		return nil
	}
}

func WithMetadata(key string, value interface{}) Option {
	return func(o *Options) error {
		if o.Metadata == nil {
			o.Metadata = make(map[string]interface{})
		}
		o.Metadata[key] = value
		return nil
	}
}

// MemoryHandle implements Handle using in-memory storage
type MemoryHandle struct {
	id          string
	status      Status
	statusMutex sync.RWMutex
	result      chan Result
	cancel      chan bool
	manager     *CallbackManager
}

// NewMemoryHandle creates a new memory handle
func NewMemoryHandle(id string) *MemoryHandle {
	return &MemoryHandle{
		id:      id,
		status:  Status{ID: id, State: StatePending, StartedAt: time.Now(), UpdatedAt: time.Now()},
		result:  make(chan Result, 1),
		cancel:  make(chan bool, 1),
		manager: NewCallbackManager(),
	}
}

// ID returns the handle ID
func (h *MemoryHandle) ID() string {
	return h.id
}

// Status returns the current status
func (h *MemoryHandle) Status() Status {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()

	statusCopy := h.status
	statusCopy.UpdatedAt = time.Now()
	return statusCopy
}

// Result returns the result channel
func (h *MemoryHandle) Result() <-chan Result {
	return h.result
}

// Cancel cancels the operation
func (h *MemoryHandle) Cancel() error {
	h.statusMutex.Lock()
	h.status.State = StateCancelled
	h.status.UpdatedAt = time.Now()
	h.statusMutex.Unlock()

	select {
	case h.cancel <- true:
	default:
	}
	return nil
}

// Wait waits for the operation to complete
func (h *MemoryHandle) Wait(ctx context.Context) (*receipt.Receipt, error) {
	select {
	case result := <-h.result:
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Receipt, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// OnComplete sets completion callback
func (h *MemoryHandle) OnComplete(callback CompletionCallback) Handle {
	h.manager.OnComplete(callback)
	return h
}

// OnError sets error callback
func (h *MemoryHandle) OnError(callback ErrorCallback) Handle {
	// Use the callback directly since signatures match
	h.manager.OnError(callback)
	return h
}

// OnProgress sets progress callback
func (h *MemoryHandle) OnProgress(callback ProgressCallback) Handle {
	h.manager.OnProgress(callback)
	return h
}

// SetResult sets the result for the handle
func (h *MemoryHandle) SetResult(result Result) {
	h.statusMutex.Lock()
	h.status.State = StateCompleted
	if result.Error != nil {
		h.status.State = StateFailed
	}
	h.status.UpdatedAt = time.Now()
	h.statusMutex.Unlock()

	select {
	case h.result <- result:
	default:
		// Channel might be full or closed
	}
}

// SetResultWithCallback sets the result and triggers callbacks
func (h *MemoryHandle) SetResultWithCallback(result Result, msg *message.Message) {
	h.SetResult(result)

	// Trigger callbacks if manager exists
	if h.manager != nil {
		if result.Error != nil {
			h.manager.TriggerError(msg, result.Error)
		} else {
			h.manager.TriggerComplete(result.Receipt)
		}
	}
}

// MemoryBatchHandle implements BatchHandle using in-memory storage
type MemoryBatchHandle struct {
	batchID     string
	handles     []Handle
	status      BatchStatus
	statusMutex sync.RWMutex
	results     chan Result
	progress    chan BatchProgress
}

// NewBatchHandle creates a new batch handle
func NewBatchHandle(handles []Handle) *MemoryBatchHandle {
	batchID := time.Now().Format("20060102150405") // Simple ID generation
	return &MemoryBatchHandle{
		batchID: batchID,
		handles: handles,
		status: BatchStatus{
			BatchID:   batchID,
			State:     StatePending,
			Total:     len(handles),
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		results:  make(chan Result, len(handles)),
		progress: make(chan BatchProgress, 10),
	}
}

// BatchID returns the batch ID
func (bh *MemoryBatchHandle) BatchID() string {
	return bh.batchID
}

// Status returns the batch status
func (bh *MemoryBatchHandle) Status() BatchStatus {
	bh.statusMutex.RLock()
	defer bh.statusMutex.RUnlock()

	statusCopy := bh.status
	statusCopy.UpdatedAt = time.Now()
	return statusCopy
}

// Results returns the results channel
func (bh *MemoryBatchHandle) Results() <-chan Result {
	return bh.results
}

// Progress returns the progress channel
func (bh *MemoryBatchHandle) Progress() <-chan BatchProgress {
	return bh.progress
}

// Cancel cancels all operations in the batch
func (bh *MemoryBatchHandle) Cancel() error {
	for _, handle := range bh.handles {
		if err := handle.Cancel(); err != nil {
			// Log error but continue cancelling other handles
			continue
		}
	}

	bh.statusMutex.Lock()
	bh.status.State = StateCancelled
	bh.status.UpdatedAt = time.Now()
	bh.statusMutex.Unlock()

	return nil
}

// Wait waits for all operations to complete
func (bh *MemoryBatchHandle) Wait(ctx context.Context) ([]*receipt.Receipt, error) {
	receipts := make([]*receipt.Receipt, len(bh.handles))

	for i, handle := range bh.handles {
		receipt, err := handle.Wait(ctx)
		if err != nil {
			receipts[i] = nil
		} else {
			receipts[i] = receipt
		}
	}

	return receipts, nil
}

// AddResult adds a result to the batch handle
func (bh *MemoryBatchHandle) AddResult(result Result) {
	select {
	case bh.results <- result:
		// Update status (thread-safe update needed)
		bh.statusMutex.Lock()
		if result.Error != nil {
			bh.status.Failed++
		} else {
			bh.status.Completed++
		}
		bh.status.Progress = float64(bh.status.Completed+bh.status.Failed) / float64(bh.status.Total)
		bh.status.UpdatedAt = time.Now()

		// Update state based on progress
		if bh.status.Completed+bh.status.Failed >= bh.status.Total {
			if bh.status.Failed == 0 {
				bh.status.State = StateCompleted
			} else {
				bh.status.State = StateFailed
			}
		} else {
			bh.status.State = StateProcessing
		}
		bh.statusMutex.Unlock()

		// Send progress update
		progress := BatchProgress{
			Completed: bh.status.Completed,
			Total:     bh.status.Total,
			Failed:    bh.status.Failed,
			Progress:  bh.status.Progress,
		}
		select {
		case bh.progress <- progress:
		default:
			// Progress channel might be full
		}
	default:
		// Results channel might be full
	}
}
