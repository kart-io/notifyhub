// Package async provides callback management for asynchronous operations
package async

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)


// CallbackRegistry manages callbacks for asynchronous operations with enhanced features
type CallbackRegistry struct {
	mutex            sync.RWMutex
	globalCallbacks  *Callbacks
	messageCallbacks map[string]*Callbacks // per-message callbacks
	logger           logger.Logger

	// Enhanced callback management
	executor         *CallbackExecutor
	tracker          *CallbackTracker
	errorRecovery    *ErrorRecoveryManager
	performanceTracker *PerformanceTracker
}

// Callbacks represents the set of callback functions with enhanced metadata
type Callbacks struct {
	OnResult   func(*receipt.Receipt)
	OnError    func(*message.Message, error)
	OnProgress func(completed, total int)
	OnComplete func(*BatchSummary)

	// Enhanced callback features
	Priority    int                    `json:"priority"`
	Timeout     time.Duration         `json:"timeout"`
	RetryPolicy *CallbackRetryPolicy  `json:"retry_policy,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	Chain       []*CallbackChain      `json:"chain,omitempty"`
}

// NewCallbackRegistry creates a new enhanced callback registry
func NewCallbackRegistry(logger logger.Logger) *CallbackRegistry {
	registry := &CallbackRegistry{
		globalCallbacks:  &Callbacks{},
		messageCallbacks: make(map[string]*Callbacks),
		logger:           logger,
	}

	// Initialize enhanced components
	registry.executor = NewCallbackExecutor(logger)
	registry.tracker = NewCallbackTracker(logger)
	registry.errorRecovery = NewErrorRecoveryManager(logger)
	registry.performanceTracker = NewPerformanceTracker(logger)

	return registry
}

// Enhanced callback types and interfaces

// Note: The callback types are already defined in handle.go to maintain compatibility
// We extend them here with additional metadata and features through the Callbacks struct

// CallbackRetryPolicy defines retry behavior for failed callbacks
type CallbackRetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialInterval time.Duration `json:"initial_interval"`
	MaxInterval     time.Duration `json:"max_interval"`
	Multiplier      float64       `json:"multiplier"`
	Jitter          bool          `json:"jitter"`
}

// CallbackChain represents a chain of callbacks for composition
type CallbackChain struct {
	Name        string                 `json:"name"`
	Callback    interface{}            `json:"-"` // Function pointer
	Condition   func() bool            `json:"-"` // Conditional execution
	Priority    int                    `json:"priority"`
	Timeout     time.Duration         `json:"timeout"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CallbackExecution represents a callback execution instance
type CallbackExecution struct {
	ID          string                 `json:"id"`
	CallbackID  string                 `json:"callback_id"`
	MessageID   string                 `json:"message_id"`
	Type        CallbackType           `json:"type"`
	Status      CallbackStatus         `json:"status"`
	StartedAt   time.Time             `json:"started_at"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	Duration    time.Duration         `json:"duration"`
	Error       string                `json:"error,omitempty"`
	RetryCount  int                   `json:"retry_count"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CallbackType represents the type of callback
type CallbackType string

const (
	CallbackTypeResult   CallbackType = "result"
	CallbackTypeError    CallbackType = "error"
	CallbackTypeProgress CallbackType = "progress"
	CallbackTypeComplete CallbackType = "complete"
)

// CallbackStatus represents the status of callback execution
type CallbackStatus string

const (
	CallbackStatusPending   CallbackStatus = "pending"
	CallbackStatusRunning   CallbackStatus = "running"
	CallbackStatusSuccess   CallbackStatus = "success"
	CallbackStatusFailed    CallbackStatus = "failed"
	CallbackStatusTimeout   CallbackStatus = "timeout"
	CallbackStatusCancelled CallbackStatus = "cancelled"
)

// CallbackExecutor manages asynchronous callback execution with worker pool
type CallbackExecutor struct {
	logger       logger.Logger
	workerPool   chan struct{} // Semaphore for limiting concurrent callbacks
	shutdownCh   chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	executionCh  chan *CallbackExecution
	maxWorkers   int
	queueSize    int

	// Statistics
	totalExecuted   int64
	totalFailed     int64
	totalTimeout    int64
	totalCancelled  int64
	averageLatency  int64 // in nanoseconds
}

// NewCallbackExecutor creates a new callback executor with worker pool
func NewCallbackExecutor(logger logger.Logger) *CallbackExecutor {
	ctx, cancel := context.WithCancel(context.Background())

	executor := &CallbackExecutor{
		logger:      logger,
		workerPool:  make(chan struct{}, 10), // Max 10 concurrent callbacks
		shutdownCh:  make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
		executionCh: make(chan *CallbackExecution, 100), // Queue up to 100 callbacks
		maxWorkers:  10,
		queueSize:   100,
	}

	// Start worker goroutines
	for i := 0; i < executor.maxWorkers; i++ {
		executor.wg.Add(1)
		go executor.worker()
	}

	return executor
}

// worker is the callback execution worker goroutine
func (e *CallbackExecutor) worker() {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			return
		case execution := <-e.executionCh:
			e.executeCallback(execution)
		}
	}
}

// ExecuteAsync executes a callback asynchronously
func (e *CallbackExecutor) ExecuteAsync(callbackType CallbackType, messageID string, callback interface{}, args ...interface{}) *CallbackExecution {
	execution := &CallbackExecution{
		ID:         fmt.Sprintf("%s_%d", messageID, time.Now().UnixNano()),
		CallbackID: fmt.Sprintf("%s_%s", callbackType, messageID),
		MessageID:  messageID,
		Type:       callbackType,
		Status:     CallbackStatusPending,
		StartedAt:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// Store callback function and arguments in metadata
	execution.Metadata["callback"] = callback
	execution.Metadata["args"] = args

	// Try to queue the execution
	select {
	case e.executionCh <- execution:
		e.logger.Debug("Callback queued for execution",
			"execution_id", execution.ID,
			"type", callbackType,
			"message_id", messageID)
		return execution
	default:
		// Queue is full, execute callback with degraded mode
		execution.Status = CallbackStatusFailed
		execution.Error = "callback queue is full"
		e.logger.Warn("Callback queue is full, dropping callback",
			"type", callbackType,
			"message_id", messageID)
		atomic.AddInt64(&e.totalFailed, 1)
		return execution
	}
}

// executeCallback executes a single callback with error recovery and tracking
func (e *CallbackExecutor) executeCallback(execution *CallbackExecution) {
	if execution == nil {
		e.logger.Error("Attempted to execute nil callback execution")
		return
	}

	startTime := time.Now()
	execution.Status = CallbackStatusRunning

	// Acquire worker slot
	e.workerPool <- struct{}{}
	defer func() { <-e.workerPool }()

	defer func() {
		execution.Duration = time.Since(startTime)
		completedAt := time.Now()
		execution.CompletedAt = &completedAt

		// Update statistics
		atomic.AddInt64(&e.totalExecuted, 1)
		atomic.StoreInt64(&e.averageLatency, execution.Duration.Nanoseconds())

		// Panic recovery
		if recovered := recover(); recovered != nil {
			execution.Status = CallbackStatusFailed
			execution.Error = fmt.Sprintf("callback panicked: %v", recovered)
			atomic.AddInt64(&e.totalFailed, 1)

			e.logger.Error("Callback panicked",
				"execution_id", execution.ID,
				"type", execution.Type,
				"message_id", execution.MessageID,
				"panic", recovered,
				"duration", execution.Duration)
		}
	}()

	// Get callback and arguments from metadata
	if execution.Metadata == nil {
		execution.Status = CallbackStatusFailed
		execution.Error = "callback metadata is nil"
		return
	}

	callback := execution.Metadata["callback"]
	if callback == nil {
		execution.Status = CallbackStatusFailed
		execution.Error = "callback function is nil"
		return
	}

	argsInterface, exists := execution.Metadata["args"]
	if !exists {
		execution.Status = CallbackStatusFailed
		execution.Error = "callback arguments not found"
		return
	}

	args, ok := argsInterface.([]interface{})
	if !ok {
		execution.Status = CallbackStatusFailed
		execution.Error = "invalid callback arguments format"
		return
	}

	// Execute callback based on type
	switch execution.Type {
	case CallbackTypeResult:
		if cb, ok := callback.(func(*receipt.Receipt)); ok && len(args) >= 1 {
			if receipt, ok := args[0].(*receipt.Receipt); ok {
				cb(receipt)
				execution.Status = CallbackStatusSuccess
			} else {
				execution.Status = CallbackStatusFailed
				execution.Error = "invalid receipt argument"
			}
		} else {
			execution.Status = CallbackStatusFailed
			execution.Error = "invalid result callback or arguments"
		}

	case CallbackTypeError:
		if cb, ok := callback.(func(*message.Message, error)); ok && len(args) >= 2 {
			if msg, ok1 := args[0].(*message.Message); ok1 {
				if err, ok2 := args[1].(error); ok2 {
					cb(msg, err)
					execution.Status = CallbackStatusSuccess
				} else {
					execution.Status = CallbackStatusFailed
					execution.Error = "invalid error argument"
				}
			} else {
				execution.Status = CallbackStatusFailed
				execution.Error = "invalid message argument"
			}
		} else {
			execution.Status = CallbackStatusFailed
			execution.Error = "invalid error callback or arguments"
		}

	case CallbackTypeProgress:
		if cb, ok := callback.(func(completed, total int)); ok && len(args) >= 2 {
			if completed, ok1 := args[0].(int); ok1 {
				if total, ok2 := args[1].(int); ok2 {
					cb(completed, total)
					execution.Status = CallbackStatusSuccess
				} else {
					execution.Status = CallbackStatusFailed
					execution.Error = "invalid total argument"
				}
			} else {
				execution.Status = CallbackStatusFailed
				execution.Error = "invalid completed argument"
			}
		} else {
			execution.Status = CallbackStatusFailed
			execution.Error = "invalid progress callback or arguments"
		}

	case CallbackTypeComplete:
		if cb, ok := callback.(func(*BatchSummary)); ok && len(args) >= 1 {
			if summary, ok := args[0].(*BatchSummary); ok {
				cb(summary)
				execution.Status = CallbackStatusSuccess
			} else {
				execution.Status = CallbackStatusFailed
				execution.Error = "invalid batch summary argument"
			}
		} else {
			execution.Status = CallbackStatusFailed
			execution.Error = "invalid complete callback or arguments"
		}

	default:
		execution.Status = CallbackStatusFailed
		execution.Error = fmt.Sprintf("unknown callback type: %s", execution.Type)
	}

	if execution.Status == CallbackStatusFailed {
		atomic.AddInt64(&e.totalFailed, 1)
		e.logger.Error("Callback execution failed",
			"execution_id", execution.ID,
			"type", execution.Type,
			"message_id", execution.MessageID,
			"error", execution.Error,
			"duration", execution.Duration)
	} else {
		e.logger.Debug("Callback executed successfully",
			"execution_id", execution.ID,
			"type", execution.Type,
			"message_id", execution.MessageID,
			"duration", execution.Duration)
	}
}

// Shutdown gracefully shuts down the callback executor
func (e *CallbackExecutor) Shutdown(timeout time.Duration) error {
	e.logger.Info("Shutting down callback executor")

	// Stop accepting new callbacks
	e.cancel()
	close(e.executionCh)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.logger.Info("Callback executor shut down gracefully")
		return nil
	case <-time.After(timeout):
		e.logger.Warn("Callback executor shutdown timeout exceeded")
		return fmt.Errorf("timeout waiting for callback workers to finish")
	}
}

// GetStats returns executor statistics
func (e *CallbackExecutor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"max_workers":       e.maxWorkers,
		"queue_size":        e.queueSize,
		"total_executed":    atomic.LoadInt64(&e.totalExecuted),
		"total_failed":      atomic.LoadInt64(&e.totalFailed),
		"total_timeout":     atomic.LoadInt64(&e.totalTimeout),
		"total_cancelled":   atomic.LoadInt64(&e.totalCancelled),
		"average_latency_ns": atomic.LoadInt64(&e.averageLatency),
	}
}

// CallbackTracker tracks callback execution status and performance
type CallbackTracker struct {
	logger     logger.Logger
	executions map[string]*CallbackExecution
	mutex      sync.RWMutex

	// Performance metrics
	totalExecutions    int64
	successfulExecutions int64
	failedExecutions   int64
	averageLatency     time.Duration
	maxLatency         time.Duration
	minLatency         time.Duration
}

// NewCallbackTracker creates a new callback tracker
func NewCallbackTracker(logger logger.Logger) *CallbackTracker {
	return &CallbackTracker{
		logger:     logger,
		executions: make(map[string]*CallbackExecution),
		minLatency: time.Hour, // Initialize with high value
	}
}

// TrackExecution tracks a callback execution
func (t *CallbackTracker) TrackExecution(execution *CallbackExecution) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.executions[execution.ID] = execution
	atomic.AddInt64(&t.totalExecutions, 1)

	if execution.Status == CallbackStatusSuccess {
		atomic.AddInt64(&t.successfulExecutions, 1)
	} else if execution.Status == CallbackStatusFailed {
		atomic.AddInt64(&t.failedExecutions, 1)
	}

	// Update latency metrics
	if execution.Duration > 0 {
		if execution.Duration > t.maxLatency {
			t.maxLatency = execution.Duration
		}
		if execution.Duration < t.minLatency || t.minLatency == time.Hour {
			t.minLatency = execution.Duration
		}

		// Simple moving average for now
		total := atomic.LoadInt64(&t.totalExecutions)
		if total > 0 {
			t.averageLatency = time.Duration((int64(t.averageLatency)*(total-1) + execution.Duration.Nanoseconds()) / total)
		}
	}

	t.logger.Debug("Callback execution tracked",
		"execution_id", execution.ID,
		"status", execution.Status,
		"duration", execution.Duration)
}

// GetExecution retrieves a callback execution by ID
func (t *CallbackTracker) GetExecution(id string) (*CallbackExecution, bool) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	execution, exists := t.executions[id]
	return execution, exists
}

// GetExecutionHistory returns the history of executions for a message
func (t *CallbackTracker) GetExecutionHistory(messageID string) []*CallbackExecution {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var history []*CallbackExecution
	for _, execution := range t.executions {
		if execution.MessageID == messageID {
			history = append(history, execution)
		}
	}
	return history
}

// GetStats returns tracker statistics
func (t *CallbackTracker) GetStats() map[string]interface{} {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	successRate := float64(0)
	if t.totalExecutions > 0 {
		successRate = float64(t.successfulExecutions) / float64(t.totalExecutions) * 100
	}

	return map[string]interface{}{
		"total_executions":     atomic.LoadInt64(&t.totalExecutions),
		"successful_executions": atomic.LoadInt64(&t.successfulExecutions),
		"failed_executions":    atomic.LoadInt64(&t.failedExecutions),
		"success_rate_percent": successRate,
		"average_latency":      t.averageLatency.String(),
		"max_latency":          t.maxLatency.String(),
		"min_latency":          t.minLatency.String(),
		"active_executions":    len(t.executions),
	}
}

// CleanupOldExecutions removes old execution records to prevent memory leaks
func (t *CallbackTracker) CleanupOldExecutions(maxAge time.Duration) int {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, execution := range t.executions {
		if execution.StartedAt.Before(cutoff) {
			delete(t.executions, id)
			removed++
		}
	}

	if removed > 0 {
		t.logger.Debug("Cleaned up old callback executions",
			"removed_count", removed,
			"remaining_count", len(t.executions))
	}

	return removed
}

// ErrorRecoveryManager handles callback failure recovery and retry mechanisms
type ErrorRecoveryManager struct {
	logger       logger.Logger
	retryPolicies map[string]*CallbackRetryPolicy
	failedCallbacks map[string]*FailedCallback
	mutex        sync.RWMutex

	// Dead letter queue for permanently failed callbacks
	deadLetterQueue chan *FailedCallback
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// FailedCallback represents a failed callback with retry information
type FailedCallback struct {
	Execution   *CallbackExecution
	RetryPolicy *CallbackRetryPolicy
	NextRetry   time.Time
	RetryCount  int
	MaxRetries  int
	LastError   error
	CreatedAt   time.Time
}

// NewErrorRecoveryManager creates a new error recovery manager
func NewErrorRecoveryManager(logger logger.Logger) *ErrorRecoveryManager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &ErrorRecoveryManager{
		logger:          logger,
		retryPolicies:   make(map[string]*CallbackRetryPolicy),
		failedCallbacks: make(map[string]*FailedCallback),
		deadLetterQueue: make(chan *FailedCallback, 1000),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Start retry worker
	manager.wg.Add(1)
	go manager.retryWorker()

	// Start dead letter processor
	manager.wg.Add(1)
	go manager.deadLetterProcessor()

	return manager
}

// SetRetryPolicy sets a retry policy for a callback type
func (m *ErrorRecoveryManager) SetRetryPolicy(callbackType CallbackType, policy *CallbackRetryPolicy) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.retryPolicies[string(callbackType)] = policy
	m.logger.Debug("Retry policy set",
		"callback_type", callbackType,
		"max_retries", policy.MaxRetries,
		"initial_interval", policy.InitialInterval)
}

// HandleFailure handles a failed callback execution
func (m *ErrorRecoveryManager) HandleFailure(execution *CallbackExecution, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Get retry policy for this callback type
	policy, exists := m.retryPolicies[string(execution.Type)]
	if !exists {
		// Use default policy
		policy = &CallbackRetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			Jitter:          true,
		}
	}

	// Check if we should retry
	if execution.RetryCount >= policy.MaxRetries {
		// Send to dead letter queue
		failedCallback := &FailedCallback{
			Execution:   execution,
			RetryPolicy: policy,
			RetryCount:  execution.RetryCount,
			MaxRetries:  policy.MaxRetries,
			LastError:   err,
			CreatedAt:   time.Now(),
		}

		select {
		case m.deadLetterQueue <- failedCallback:
			m.logger.Warn("Callback sent to dead letter queue",
				"execution_id", execution.ID,
				"retry_count", execution.RetryCount,
				"max_retries", policy.MaxRetries)
		default:
			m.logger.Error("Dead letter queue is full, dropping failed callback",
				"execution_id", execution.ID)
		}
		return
	}

	// Calculate next retry time
	interval := policy.InitialInterval
	for i := 0; i < execution.RetryCount; i++ {
		interval = time.Duration(float64(interval) * policy.Multiplier)
		if interval > policy.MaxInterval {
			interval = policy.MaxInterval
			break
		}
	}

	// Add jitter if enabled
	if policy.Jitter {
		jitter := time.Duration(float64(interval) * 0.1) // 10% jitter
		jitterSign := int64(1)
		if time.Now().UnixNano()%2 == 0 {
			jitterSign = -1
		}
		interval += time.Duration(float64(jitter) * float64(jitterSign))
	}

	failedCallback := &FailedCallback{
		Execution:   execution,
		RetryPolicy: policy,
		NextRetry:   time.Now().Add(interval),
		RetryCount:  execution.RetryCount + 1,
		MaxRetries:  policy.MaxRetries,
		LastError:   err,
		CreatedAt:   time.Now(),
	}

	m.failedCallbacks[execution.ID] = failedCallback

	m.logger.Info("Callback scheduled for retry",
		"execution_id", execution.ID,
		"retry_count", failedCallback.RetryCount,
		"next_retry", failedCallback.NextRetry,
		"interval", interval)
}

// retryWorker processes failed callbacks for retry
func (m *ErrorRecoveryManager) retryWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Second) // Check every second
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.processRetries()
		}
	}
}

// processRetries processes callbacks that are ready for retry
func (m *ErrorRecoveryManager) processRetries() {
	m.mutex.Lock()
	now := time.Now()
	toRetry := make([]*FailedCallback, 0)

	for id, failedCallback := range m.failedCallbacks {
		if now.After(failedCallback.NextRetry) {
			toRetry = append(toRetry, failedCallback)
			delete(m.failedCallbacks, id)
		}
	}
	m.mutex.Unlock()

	// Process retries outside of lock
	for _, failedCallback := range toRetry {
		m.logger.Info("Retrying failed callback",
			"execution_id", failedCallback.Execution.ID,
			"retry_count", failedCallback.RetryCount)

		// Update execution for retry
		execution := failedCallback.Execution
		execution.RetryCount = failedCallback.RetryCount
		execution.Status = CallbackStatusPending
		execution.Error = ""

		// TODO: Re-queue the callback for execution
		// This would integrate with the CallbackExecutor to retry the callback
	}
}

// deadLetterProcessor processes permanently failed callbacks
func (m *ErrorRecoveryManager) deadLetterProcessor() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case failedCallback := <-m.deadLetterQueue:
			m.logger.Error("Callback permanently failed",
				"execution_id", failedCallback.Execution.ID,
				"callback_type", failedCallback.Execution.Type,
				"message_id", failedCallback.Execution.MessageID,
				"retry_count", failedCallback.RetryCount,
				"last_error", failedCallback.LastError)

			// TODO: Implement dead letter handling (e.g., store to database, send alerts)
		}
	}
}

// GetFailedCallbacks returns the list of currently failed callbacks
func (m *ErrorRecoveryManager) GetFailedCallbacks() []*FailedCallback {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	callbacks := make([]*FailedCallback, 0, len(m.failedCallbacks))
	for _, callback := range m.failedCallbacks {
		callbacks = append(callbacks, callback)
	}
	return callbacks
}

// GetStats returns error recovery statistics
func (m *ErrorRecoveryManager) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"failed_callbacks_count":    len(m.failedCallbacks),
		"retry_policies_count":      len(m.retryPolicies),
		"dead_letter_queue_size":    len(m.deadLetterQueue),
		"dead_letter_queue_capacity": cap(m.deadLetterQueue),
	}
}

// Shutdown gracefully shuts down the error recovery manager
func (m *ErrorRecoveryManager) Shutdown(timeout time.Duration) error {
	m.logger.Info("Shutting down error recovery manager")

	m.cancel()
	close(m.deadLetterQueue)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info("Error recovery manager shut down gracefully")
		return nil
	case <-time.After(timeout):
		m.logger.Warn("Error recovery manager shutdown timeout exceeded")
		return fmt.Errorf("timeout waiting for error recovery workers to finish")
	}
}

// PerformanceTracker tracks callback performance metrics
type PerformanceTracker struct {
	logger  logger.Logger
	metrics map[string]*CallbackMetrics
	mutex   sync.RWMutex
}

// CallbackMetrics represents performance metrics for a callback type
type CallbackMetrics struct {
	CallbackType       CallbackType  `json:"callback_type"`
	TotalExecutions    int64         `json:"total_executions"`
	SuccessfulExecutions int64       `json:"successful_executions"`
	FailedExecutions   int64         `json:"failed_executions"`
	AverageLatency     time.Duration `json:"average_latency"`
	MaxLatency         time.Duration `json:"max_latency"`
	MinLatency         time.Duration `json:"min_latency"`
	P95Latency         time.Duration `json:"p95_latency"`
	P99Latency         time.Duration `json:"p99_latency"`
	LastUpdated        time.Time     `json:"last_updated"`
	LatencyHistory     []time.Duration `json:"-"` // Store recent latencies for percentile calculation
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker(logger logger.Logger) *PerformanceTracker {
	return &PerformanceTracker{
		logger:  logger,
		metrics: make(map[string]*CallbackMetrics),
	}
}

// RecordExecution records a callback execution for performance tracking
func (p *PerformanceTracker) RecordExecution(execution *CallbackExecution) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	key := string(execution.Type)
	metrics, exists := p.metrics[key]
	if !exists {
		metrics = &CallbackMetrics{
			CallbackType:   execution.Type,
			MinLatency:     time.Hour, // Initialize with high value
			LatencyHistory: make([]time.Duration, 0, 100), // Keep last 100 latencies
		}
		p.metrics[key] = metrics
	}

	// Update counters
	metrics.TotalExecutions++
	if execution.Status == CallbackStatusSuccess {
		metrics.SuccessfulExecutions++
	} else if execution.Status == CallbackStatusFailed {
		metrics.FailedExecutions++
	}

	// Update latency metrics
	if execution.Duration > 0 {
		// Update min/max
		if execution.Duration > metrics.MaxLatency {
			metrics.MaxLatency = execution.Duration
		}
		if execution.Duration < metrics.MinLatency || metrics.MinLatency == time.Hour {
			metrics.MinLatency = execution.Duration
		}

		// Update average
		metrics.AverageLatency = time.Duration((int64(metrics.AverageLatency)*(metrics.TotalExecutions-1) + execution.Duration.Nanoseconds()) / metrics.TotalExecutions)

		// Add to latency history for percentile calculation
		metrics.LatencyHistory = append(metrics.LatencyHistory, execution.Duration)
		if len(metrics.LatencyHistory) > 100 {
			// Keep only the most recent 100 latencies
			metrics.LatencyHistory = metrics.LatencyHistory[1:]
		}

		// Calculate percentiles
		p.calculatePercentiles(metrics)
	}

	metrics.LastUpdated = time.Now()

	p.logger.Debug("Callback performance recorded",
		"callback_type", execution.Type,
		"duration", execution.Duration,
		"total_executions", metrics.TotalExecutions)
}

// calculatePercentiles calculates P95 and P99 latencies
func (p *PerformanceTracker) calculatePercentiles(metrics *CallbackMetrics) {
	if len(metrics.LatencyHistory) == 0 {
		return
	}

	// Sort latencies (simple bubble sort for small datasets)
	sorted := make([]time.Duration, len(metrics.LatencyHistory))
	copy(sorted, metrics.LatencyHistory)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate P95 and P99
	n := len(sorted)
	if n > 0 {
		p95Index := int(float64(n) * 0.95)
		if p95Index >= n {
			p95Index = n - 1
		}
		metrics.P95Latency = sorted[p95Index]

		p99Index := int(float64(n) * 0.99)
		if p99Index >= n {
			p99Index = n - 1
		}
		metrics.P99Latency = sorted[p99Index]
	}
}

// GetMetrics returns performance metrics for a callback type
func (p *PerformanceTracker) GetMetrics(callbackType CallbackType) (*CallbackMetrics, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	metrics, exists := p.metrics[string(callbackType)]
	return metrics, exists
}

// GetAllMetrics returns all performance metrics
func (p *PerformanceTracker) GetAllMetrics() map[string]*CallbackMetrics {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*CallbackMetrics)
	for k, v := range p.metrics {
		result[k] = v
	}
	return result
}

// GetStats returns performance tracker statistics
func (p *PerformanceTracker) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	stats := make(map[string]interface{})
	for callbackType, metrics := range p.metrics {
		successRate := float64(0)
		if metrics.TotalExecutions > 0 {
			successRate = float64(metrics.SuccessfulExecutions) / float64(metrics.TotalExecutions) * 100
		}

		stats[callbackType] = map[string]interface{}{
			"total_executions":      metrics.TotalExecutions,
			"successful_executions": metrics.SuccessfulExecutions,
			"failed_executions":     metrics.FailedExecutions,
			"success_rate_percent":  successRate,
			"average_latency":       metrics.AverageLatency.String(),
			"max_latency":           metrics.MaxLatency.String(),
			"min_latency":           metrics.MinLatency.String(),
			"p95_latency":           metrics.P95Latency.String(),
			"p99_latency":           metrics.P99Latency.String(),
			"last_updated":          metrics.LastUpdated,
		}
	}

	return stats
}

// RegisterGlobalCallbacks registers global callbacks for all operations
func (r *CallbackRegistry) RegisterGlobalCallbacks(callbacks *Callbacks) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if callbacks.OnResult != nil {
		r.globalCallbacks.OnResult = callbacks.OnResult
	}
	if callbacks.OnError != nil {
		r.globalCallbacks.OnError = callbacks.OnError
	}
	if callbacks.OnProgress != nil {
		r.globalCallbacks.OnProgress = callbacks.OnProgress
	}
	if callbacks.OnComplete != nil {
		r.globalCallbacks.OnComplete = callbacks.OnComplete
	}

	r.logger.Debug("Registered global callbacks")
}

// RegisterMessageCallbacks registers callbacks for a specific message
func (r *CallbackRegistry) RegisterMessageCallbacks(messageID string, callbacks *Callbacks) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.messageCallbacks[messageID] = callbacks
	r.logger.Debug("Registered message callbacks", "message_id", messageID)
}

// TriggerResult triggers result callbacks using the enhanced execution engine
func (r *CallbackRegistry) TriggerResult(result *receipt.Receipt) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.logger.Debug("Triggering result callbacks", "message_id", result.MessageID)

	// Trigger message-specific callback
	if callbacks, exists := r.messageCallbacks[result.MessageID]; exists && callbacks.OnResult != nil {
		execution := r.executor.ExecuteAsync(CallbackTypeResult, result.MessageID, callbacks.OnResult, result)
		r.tracker.TrackExecution(execution)
		r.performanceTracker.RecordExecution(execution)
	}

	// Trigger global callback
	if r.globalCallbacks.OnResult != nil {
		execution := r.executor.ExecuteAsync(CallbackTypeResult, result.MessageID, r.globalCallbacks.OnResult, result)
		r.tracker.TrackExecution(execution)
		r.performanceTracker.RecordExecution(execution)
	}
}

// TriggerError triggers error callbacks using the enhanced execution engine
func (r *CallbackRegistry) TriggerError(msg *message.Message, err error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.logger.Debug("Triggering error callbacks", "message_id", msg.ID, "error", err)

	// Trigger message-specific callback
	if callbacks, exists := r.messageCallbacks[msg.ID]; exists && callbacks.OnError != nil {
		execution := r.executor.ExecuteAsync(CallbackTypeError, msg.ID, callbacks.OnError, msg, err)
		r.tracker.TrackExecution(execution)
		r.performanceTracker.RecordExecution(execution)
	}

	// Trigger global callback
	if r.globalCallbacks.OnError != nil {
		execution := r.executor.ExecuteAsync(CallbackTypeError, msg.ID, r.globalCallbacks.OnError, msg, err)
		r.tracker.TrackExecution(execution)
		r.performanceTracker.RecordExecution(execution)
	}
}

// TriggerProgress triggers progress callbacks using the enhanced execution engine
func (r *CallbackRegistry) TriggerProgress(completed, total int) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.logger.Debug("Triggering progress callbacks", "completed", completed, "total", total)

	// Only global progress callback makes sense for batch operations
	if r.globalCallbacks.OnProgress != nil {
		execution := r.executor.ExecuteAsync(CallbackTypeProgress, "progress", r.globalCallbacks.OnProgress, completed, total)
		r.tracker.TrackExecution(execution)
		r.performanceTracker.RecordExecution(execution)
	}
}

// TriggerComplete triggers completion callbacks using the enhanced execution engine
func (r *CallbackRegistry) TriggerComplete(summary *BatchSummary) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.logger.Debug("Triggering completion callbacks", "batch_id", summary.BatchID)

	// Only global completion callback makes sense for batch operations
	if r.globalCallbacks.OnComplete != nil {
		execution := r.executor.ExecuteAsync(CallbackTypeComplete, summary.BatchID, r.globalCallbacks.OnComplete, summary)
		r.tracker.TrackExecution(execution)
		r.performanceTracker.RecordExecution(execution)
	}
}

// CleanupMessageCallbacks removes callbacks for a completed message
func (r *CallbackRegistry) CleanupMessageCallbacks(messageID string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.messageCallbacks, messageID)
	r.logger.Debug("Cleaned up message callbacks", "message_id", messageID)
}

// safeCallback executes a callback function with error recovery
func (r *CallbackRegistry) safeCallback(callback func(), callbackType, messageID string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			r.logger.Error("Callback panicked",
				"callback_type", callbackType,
				"message_id", messageID,
				"panic", recovered)
		}
	}()

	callback()
}

// GetCallbackCount returns the number of registered message callbacks (for monitoring)
func (r *CallbackRegistry) GetCallbackCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.messageCallbacks)
}

// HasGlobalCallbacks returns whether global callbacks are registered
func (r *CallbackRegistry) HasGlobalCallbacks() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.globalCallbacks.OnResult != nil ||
		r.globalCallbacks.OnError != nil ||
		r.globalCallbacks.OnProgress != nil ||
		r.globalCallbacks.OnComplete != nil
}

// ClearGlobalCallbacks clears all global callbacks
func (r *CallbackRegistry) ClearGlobalCallbacks() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.globalCallbacks = &Callbacks{}
	r.logger.Debug("Cleared global callbacks")
}

// ClearAllCallbacks clears all callbacks (global and message-specific)
func (r *CallbackRegistry) ClearAllCallbacks() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.globalCallbacks = &Callbacks{}
	r.messageCallbacks = make(map[string]*Callbacks)
	r.logger.Debug("Cleared all callbacks")
}

// Enhanced registry methods for the callback management system

// SetCallbackTimeout sets timeout for callback execution
func (r *CallbackRegistry) SetCallbackTimeout(messageID string, timeout time.Duration) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if callbacks, exists := r.messageCallbacks[messageID]; exists {
		callbacks.Timeout = timeout
		r.logger.Debug("Set callback timeout", "message_id", messageID, "timeout", timeout)
	}
}

// SetCallbackPriority sets priority for callback execution
func (r *CallbackRegistry) SetCallbackPriority(messageID string, priority int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if callbacks, exists := r.messageCallbacks[messageID]; exists {
		callbacks.Priority = priority
		r.logger.Debug("Set callback priority", "message_id", messageID, "priority", priority)
	}
}

// SetCallbackRetryPolicy sets retry policy for callback execution
func (r *CallbackRegistry) SetCallbackRetryPolicy(messageID string, policy *CallbackRetryPolicy) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if callbacks, exists := r.messageCallbacks[messageID]; exists {
		callbacks.RetryPolicy = policy
		r.logger.Debug("Set callback retry policy", "message_id", messageID, "max_retries", policy.MaxRetries)
	}
}

// AddCallbackMetadata adds metadata to callback
func (r *CallbackRegistry) AddCallbackMetadata(messageID string, key string, value interface{}) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if callbacks, exists := r.messageCallbacks[messageID]; exists {
		if callbacks.Metadata == nil {
			callbacks.Metadata = make(map[string]interface{})
		}
		callbacks.Metadata[key] = value
		r.logger.Debug("Added callback metadata", "message_id", messageID, "key", key)
	}
}

// RegisterCallbackChain registers a chain of callbacks for composition
func (r *CallbackRegistry) RegisterCallbackChain(messageID string, chain []*CallbackChain) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if callbacks, exists := r.messageCallbacks[messageID]; exists {
		callbacks.Chain = chain
		r.logger.Debug("Registered callback chain", "message_id", messageID, "chain_length", len(chain))
	}
}

// ExecuteCallbackChain executes a chain of callbacks in order
func (r *CallbackRegistry) ExecuteCallbackChain(messageID string, result *receipt.Receipt, err error) {
	r.mutex.RLock()
	callbacks, exists := r.messageCallbacks[messageID]
	r.mutex.RUnlock()

	if !exists || callbacks.Chain == nil {
		return
	}

	// Sort chain by priority
	chain := make([]*CallbackChain, len(callbacks.Chain))
	copy(chain, callbacks.Chain)
	for i := 0; i < len(chain); i++ {
		for j := i + 1; j < len(chain); j++ {
			if chain[i].Priority < chain[j].Priority {
				chain[i], chain[j] = chain[j], chain[i]
			}
		}
	}

	// Execute callbacks in priority order
	for _, cb := range chain {
		// Check condition if present
		if cb.Condition != nil && !cb.Condition() {
			r.logger.Debug("Skipping callback due to condition",
				"message_id", messageID,
				"callback_name", cb.Name)
			continue
		}

		// Execute callback based on type
		switch callback := cb.Callback.(type) {
		case func(*receipt.Receipt):
			if result != nil {
				execution := r.executor.ExecuteAsync(CallbackTypeResult, messageID, callback, result)
				r.tracker.TrackExecution(execution)
				r.performanceTracker.RecordExecution(execution)
			}
		case func(*message.Message, error):
			if err != nil {
				msg := &message.Message{ID: messageID} // Simplified for chain execution
				execution := r.executor.ExecuteAsync(CallbackTypeError, messageID, callback, msg, err)
				r.tracker.TrackExecution(execution)
				r.performanceTracker.RecordExecution(execution)
			}
		}
	}
}

// GetCallbackStats returns comprehensive callback statistics
func (r *CallbackRegistry) GetCallbackStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	executorStats := r.executor.GetStats()
	trackerStats := r.tracker.GetStats()
	recoveryStats := r.errorRecovery.GetStats()
	performanceStats := r.performanceTracker.GetStats()

	return map[string]interface{}{
		"registry": map[string]interface{}{
			"global_callbacks_count":  r.getGlobalCallbackCount(),
			"message_callbacks_count": len(r.messageCallbacks),
			"has_global_callbacks":    r.HasGlobalCallbacks(),
		},
		"executor":    executorStats,
		"tracker":     trackerStats,
		"recovery":    recoveryStats,
		"performance": performanceStats,
	}
}

// getGlobalCallbackCount returns the number of global callbacks registered
func (r *CallbackRegistry) getGlobalCallbackCount() int {
	count := 0
	if r.globalCallbacks.OnResult != nil {
		count++
	}
	if r.globalCallbacks.OnError != nil {
		count++
	}
	if r.globalCallbacks.OnProgress != nil {
		count++
	}
	if r.globalCallbacks.OnComplete != nil {
		count++
	}
	return count
}

// GetExecutionHistory returns the execution history for a message
func (r *CallbackRegistry) GetExecutionHistory(messageID string) []*CallbackExecution {
	return r.tracker.GetExecutionHistory(messageID)
}

// GetFailedCallbacks returns the list of failed callbacks
func (r *CallbackRegistry) GetFailedCallbacks() []*FailedCallback {
	return r.errorRecovery.GetFailedCallbacks()
}

// SetRetryPolicy sets a retry policy for a callback type
func (r *CallbackRegistry) SetRetryPolicy(callbackType CallbackType, policy *CallbackRetryPolicy) {
	r.errorRecovery.SetRetryPolicy(callbackType, policy)
}

// PerformMaintenance performs maintenance tasks like cleanup and optimization
func (r *CallbackRegistry) PerformMaintenance() {
	r.logger.Debug("Performing callback registry maintenance")

	// Cleanup old execution records (older than 1 hour)
	removed := r.tracker.CleanupOldExecutions(1 * time.Hour)
	if removed > 0 {
		r.logger.Info("Cleaned up old callback executions", "removed_count", removed)
	}

	// Get current statistics for monitoring
	stats := r.GetCallbackStats()
	r.logger.Info("Callback registry maintenance completed", "stats", stats)
}

// Shutdown gracefully shuts down the callback registry and all components
func (r *CallbackRegistry) Shutdown(timeout time.Duration) error {
	r.logger.Info("Shutting down callback registry")

	// Shutdown components in order
	var errs []error

	if err := r.executor.Shutdown(timeout); err != nil {
		errs = append(errs, fmt.Errorf("executor shutdown failed: %w", err))
	}

	if err := r.errorRecovery.Shutdown(timeout); err != nil {
		errs = append(errs, fmt.Errorf("error recovery shutdown failed: %w", err))
	}

	// Clear all callbacks
	r.ClearAllCallbacks()

	if len(errs) > 0 {
		return fmt.Errorf("callback registry shutdown errors: %v", errs)
	}

	r.logger.Info("Callback registry shut down successfully")
	return nil
}

// IsHealthy returns whether the callback registry is healthy
func (r *CallbackRegistry) IsHealthy() bool {
	// Check if all components are functioning
	stats := r.GetCallbackStats()

	// Basic health checks
	executorStats := stats["executor"].(map[string]interface{})
	if executorStats["total_failed"].(int64) > executorStats["total_executed"].(int64)/2 {
		return false // More than 50% failure rate
	}

	recoveryStats := stats["recovery"].(map[string]interface{})
	if recoveryStats["dead_letter_queue_size"].(int) >= recoveryStats["dead_letter_queue_capacity"].(int) {
		return false // Dead letter queue is full
	}

	return true
}

// GetHealthStatus returns detailed health status
func (r *CallbackRegistry) GetHealthStatus() map[string]interface{} {
	stats := r.GetCallbackStats()
	healthy := r.IsHealthy()

	return map[string]interface{}{
		"healthy":         healthy,
		"timestamp":       time.Now(),
		"component_stats": stats,
	}
}
