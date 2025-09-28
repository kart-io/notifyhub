// Package async provides worker pool for asynchronous message processing
package async

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
)

// WorkerPoolConfig defines configuration for worker pool
type WorkerPoolConfig struct {
	MinWorkers      int           `json:"min_workers"`      // Minimum number of workers
	MaxWorkers      int           `json:"max_workers"`      // Maximum number of workers
	TargetLoad      float64       `json:"target_load"`      // Target load percentage (0.0-1.0)
	ScaleUpDelay    time.Duration `json:"scale_up_delay"`   // Delay before scaling up
	ScaleDownDelay  time.Duration `json:"scale_down_delay"` // Delay before scaling down
	HealthCheckTime time.Duration `json:"health_check_time"` // Health check interval
	MaxIdleTime     time.Duration `json:"max_idle_time"`     // Max idle time before worker removal
	TaskBatchSize   int           `json:"task_batch_size"`   // Tasks per batch for batch processing
}

// DefaultWorkerPoolConfig returns default configuration
func DefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		MinWorkers:      2,
		MaxWorkers:      runtime.NumCPU() * 2,
		TargetLoad:      0.7,
		ScaleUpDelay:    5 * time.Second,
		ScaleDownDelay:  30 * time.Second,
		HealthCheckTime: 10 * time.Second,
		MaxIdleTime:     60 * time.Second,
		TaskBatchSize:   10,
	}
}

// WorkerPool manages a pool of workers for processing async messages with dynamic scaling
type WorkerPool struct {
	config       *WorkerPoolConfig
	workers      map[int]*Worker
	queue        AsyncQueue
	dispatcher   core.Dispatcher
	callbacks    *CallbackRegistry
	logger       logger.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	stopped      bool
	mutex        sync.RWMutex
	nextWorkerID int32
	loadBalancer *LoadBalancer
	monitor      *WorkerMonitor
	scaler       *WorkerScaler
	startTime    time.Time
}

// WorkerState represents the current state of a worker
type WorkerState int

const (
	WorkerStateIdle WorkerState = iota
	WorkerStateProcessing
	WorkerStateShuttingDown
	WorkerStateStopped
)

// String returns string representation of worker state
func (s WorkerState) String() string {
	switch s {
	case WorkerStateIdle:
		return "idle"
	case WorkerStateProcessing:
		return "processing"
	case WorkerStateShuttingDown:
		return "shutting_down"
	case WorkerStateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// Worker represents a single worker in the pool with enhanced monitoring
type Worker struct {
	id            int
	pool          *WorkerPool
	queue         AsyncQueue
	dispatcher    core.Dispatcher
	callbacks     *CallbackRegistry
	logger        logger.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	processed     int64
	errors        int64
	startTime     time.Time
	lastActivity  time.Time
	state         WorkerState
	stateMutex    sync.RWMutex
	currentTask   *QueueItem
	taskMutex     sync.RWMutex
	shutdownCh    chan struct{}
	heartbeatCh   chan time.Time
	affinity      WorkerAffinity
	performance   *WorkerPerformance
}

// WorkerAffinity defines worker specialization
type WorkerAffinity struct {
	Platforms    []string    `json:"platforms"`    // Preferred platforms
	MessageTypes []string    `json:"message_types"` // Preferred message types
	Priorities   []int       `json:"priorities"`    // Preferred priorities
	Specialized  bool        `json:"specialized"`   // Whether worker is specialized
}

// WorkerPerformance tracks worker performance metrics
type WorkerPerformance struct {
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	Throughput        float64       `json:"throughput"` // messages per second
	ErrorRate         float64       `json:"error_rate"`
	LastUpdateTime    time.Time     `json:"last_update_time"`
	mutex             sync.RWMutex
}

// WorkerStats represents comprehensive statistics for a worker
type WorkerStats struct {
	ID               int               `json:"id"`
	State            string            `json:"state"`
	Processed        int64             `json:"processed"`
	Errors           int64             `json:"errors"`
	Uptime           int64             `json:"uptime_seconds"`
	LastActivity     time.Time         `json:"last_activity"`
	CurrentTask      string            `json:"current_task,omitempty"`
	Affinity         WorkerAffinity    `json:"affinity"`
	Performance      *WorkerPerformance `json:"performance"`
	CPUUsage         float64           `json:"cpu_usage"`
	MemoryUsage      uint64            `json:"memory_usage_bytes"`
}

// LoadBalancer manages task distribution among workers
type LoadBalancer struct {
	strategy   LoadBalanceStrategy
	workerPool *WorkerPool
	logger     logger.Logger
	mutex      sync.RWMutex
}

// LoadBalanceStrategy defines how tasks are distributed
type LoadBalanceStrategy int

const (
	RoundRobin LoadBalanceStrategy = iota
	LeastConnections
	WeightedRoundRobin
	AffinityBased
)

// WorkerMonitor provides health monitoring and status tracking
type WorkerMonitor struct {
	pool           *WorkerPool
	logger         logger.Logger
	monitoringCh   chan WorkerEvent
	heartbeatTimer *time.Ticker
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// WorkerEvent represents worker lifecycle events
type WorkerEvent struct {
	WorkerID  int
	EventType WorkerEventType
	Timestamp time.Time
	Data      map[string]interface{}
}

// WorkerEventType defines types of worker events
type WorkerEventType int

const (
	WorkerStarted WorkerEventType = iota
	WorkerStopped
	WorkerIdle
	WorkerBusy
	WorkerError
	WorkerTimeout
	WorkerHeartbeat
)

// WorkerScaler handles dynamic scaling of worker pool
type WorkerScaler struct {
	pool          *WorkerPool
	logger        logger.Logger
	scalingTimer  *time.Ticker
	lastScaleUp   time.Time
	lastScaleDown time.Time
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	loadHistory   []float64
	mutex         sync.RWMutex
}

// NewWorkerPool creates a new enhanced worker pool
func NewWorkerPool(config *WorkerPoolConfig, queue AsyncQueue, dispatcher core.Dispatcher, callbacks *CallbackRegistry, logger logger.Logger) *WorkerPool {
	if config == nil {
		config = DefaultWorkerPoolConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		config:       config,
		workers:      make(map[int]*Worker),
		queue:        queue,
		dispatcher:   dispatcher,
		callbacks:    callbacks,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
		nextWorkerID: 0,
		startTime:    time.Now(),
	}

	// Create load balancer
	pool.loadBalancer = NewLoadBalancer(LeastConnections, pool, logger)

	// Create monitor
	pool.monitor = NewWorkerMonitor(pool, logger)

	// Create scaler
	pool.scaler = NewWorkerScaler(pool, logger)

	// Create initial workers
	for i := 0; i < config.MinWorkers; i++ {
		worker := pool.createWorker()
		pool.workers[worker.id] = worker
	}

	return pool
}

// createWorker creates a new worker with enhanced features
func (p *WorkerPool) createWorker() *Worker {
	id := int(atomic.AddInt32(&p.nextWorkerID, 1))
	ctx, cancel := context.WithCancel(p.ctx)

	worker := &Worker{
		id:           id,
		pool:         p,
		queue:        p.queue,
		dispatcher:   p.dispatcher,
		callbacks:    p.callbacks,
		logger:       p.logger,
		ctx:          ctx,
		cancel:       cancel,
		startTime:    time.Now(),
		lastActivity: time.Now(),
		state:        WorkerStateIdle,
		shutdownCh:   make(chan struct{}),
		heartbeatCh:  make(chan time.Time, 1),
		affinity:     WorkerAffinity{},
		performance:  &WorkerPerformance{LastUpdateTime: time.Now()},
	}

	return worker
}

// Start starts the worker pool with graceful staging
func (p *WorkerPool) Start() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.stopped {
		return fmt.Errorf("cannot start stopped worker pool")
	}

	p.logger.Info("Starting enhanced worker pool",
		"initial_workers", len(p.workers),
		"min_workers", p.config.MinWorkers,
		"max_workers", p.config.MaxWorkers)

	// Start monitor first
	if err := p.monitor.Start(); err != nil {
		return fmt.Errorf("failed to start monitor: %w", err)
	}

	// Start scaler
	if err := p.scaler.Start(); err != nil {
		p.monitor.Stop(5 * time.Second)
		return fmt.Errorf("failed to start scaler: %w", err)
	}

	// Start workers in stages to avoid resource spikes
	workerList := make([]*Worker, 0, len(p.workers))
	for _, worker := range p.workers {
		workerList = append(workerList, worker)
	}

	// Start workers in batches
	batchSize := 2
	for i := 0; i < len(workerList); i += batchSize {
		end := i + batchSize
		if end > len(workerList) {
			end = len(workerList)
		}

		// Start batch of workers
		for j := i; j < end; j++ {
			worker := workerList[j]
			p.wg.Add(1)
			go worker.start(&p.wg)
			p.logger.Debug("Started worker", "worker_id", worker.id)
		}

		// Brief pause between batches
		if end < len(workerList) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	p.logger.Info("Worker pool started successfully", "active_workers", len(p.workers))
	return nil
}

// Stop gracefully stops the worker pool with staged shutdown
func (p *WorkerPool) Stop(timeout time.Duration) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.stopped {
		return nil
	}

	p.logger.Info("Stopping enhanced worker pool gracefully",
		"timeout", timeout,
		"active_workers", len(p.workers))
	p.stopped = true

	// Stop scaler first
	if err := p.scaler.Stop(5 * time.Second); err != nil {
		p.logger.Warn("Failed to stop scaler cleanly", "error", err)
	}

	// Stop accepting new work - signal all workers
	p.cancel()

	// Allow workers to finish current tasks
	drainTimeout := timeout / 3
	p.logger.Info("Allowing workers to drain current tasks", "drain_timeout", drainTimeout)

	// Wait for workers with staged timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("All workers stopped gracefully")
	case <-time.After(drainTimeout):
		p.logger.Warn("Worker drain timeout, forcing shutdown")
		// Force shutdown remaining workers
		p.forceStopWorkers()
	}

	// Stop monitor last
	if err := p.monitor.Stop(5 * time.Second); err != nil {
		p.logger.Warn("Failed to stop monitor cleanly", "error", err)
	}

	p.logger.Info("Worker pool shutdown completed")
	return nil
}

// forceStopWorkers forcefully stops remaining workers
func (p *WorkerPool) forceStopWorkers() {
	for _, worker := range p.workers {
		select {
		case worker.shutdownCh <- struct{}{}:
			p.logger.Debug("Sent force shutdown signal", "worker_id", worker.id)
		default:
			// Channel full or closed, worker likely already stopping
		}
	}
}

// GetStats returns comprehensive statistics for all workers
func (p *WorkerPool) GetStats() []WorkerStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	stats := make([]WorkerStats, 0, len(p.workers))
	for _, worker := range p.workers {
		stats = append(stats, worker.getStats())
	}
	return stats
}

// GetTotalStats returns aggregated statistics
func (p *WorkerPool) GetTotalStats() (processed int64, errors int64) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, worker := range p.workers {
		processed += atomic.LoadInt64(&worker.processed)
		errors += atomic.LoadInt64(&worker.errors)
	}
	return
}

// GetDetailedStats returns comprehensive pool statistics
func (p *WorkerPool) GetDetailedStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	processed, errors := p.GetTotalStats()
	queueHealth := p.queue.Health()

	// Calculate load and performance metrics
	load := p.calculateCurrentLoad()
	throughput := p.calculateThroughput()

	stats := map[string]interface{}{
		"pool_info": map[string]interface{}{
			"active_workers":     len(p.workers),
			"min_workers":        p.config.MinWorkers,
			"max_workers":        p.config.MaxWorkers,
			"target_load":        p.config.TargetLoad,
			"current_load":       load,
			"uptime_seconds":     int64(time.Since(p.startTime).Seconds()),
		},
		"performance": map[string]interface{}{
			"total_processed":    processed,
			"total_errors":       errors,
			"error_rate":         float64(errors) / float64(processed+1),
			"throughput":         throughput,
		},
		"queue_info": map[string]interface{}{
			"size":      queueHealth.Size,
			"max_size":  queueHealth.MaxSize,
			"healthy":   queueHealth.Healthy,
		},
		"worker_stats": p.GetStats(),
	}

	return stats
}

// calculateCurrentLoad calculates current system load
func (p *WorkerPool) calculateCurrentLoad() float64 {
	busyWorkers := 0
	for _, worker := range p.workers {
		worker.stateMutex.RLock()
		if worker.state == WorkerStateProcessing {
			busyWorkers++
		}
		worker.stateMutex.RUnlock()
	}

	if len(p.workers) == 0 {
		return 0.0
	}
	return float64(busyWorkers) / float64(len(p.workers))
}

// calculateThroughput calculates messages per second
func (p *WorkerPool) calculateThroughput() float64 {
	processed, _ := p.GetTotalStats()
	uptime := time.Since(p.startTime).Seconds()
	if uptime <= 0 {
		return 0.0
	}
	return float64(processed) / uptime
}

// IsHealthy returns whether the worker pool is healthy
func (p *WorkerPool) IsHealthy() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.stopped {
		return false
	}

	// Check if we have minimum workers
	if len(p.workers) < p.config.MinWorkers {
		return false
	}

	// Check worker health
	healthyWorkers := 0
	for _, worker := range p.workers {
		if worker.isHealthy() {
			healthyWorkers++
		}
	}

	// At least 50% of workers should be healthy
	return float64(healthyWorkers)/float64(len(p.workers)) >= 0.5
}

// AddWorker dynamically adds a new worker to the pool
func (p *WorkerPool) AddWorker() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.workers) >= p.config.MaxWorkers {
		return fmt.Errorf("cannot add worker: at maximum capacity %d", p.config.MaxWorkers)
	}

	worker := p.createWorker()
	p.workers[worker.id] = worker

	p.wg.Add(1)
	go worker.start(&p.wg)

	p.logger.Info("Added new worker", "worker_id", worker.id, "total_workers", len(p.workers))
	return nil
}

// RemoveWorker gracefully removes a worker from the pool
func (p *WorkerPool) RemoveWorker(workerID int) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.workers) <= p.config.MinWorkers {
		return fmt.Errorf("cannot remove worker: at minimum capacity %d", p.config.MinWorkers)
	}

	worker, exists := p.workers[workerID]
	if !exists {
		return fmt.Errorf("worker %d not found", workerID)
	}

	// Signal worker to stop
	worker.cancel()
	delete(p.workers, workerID)

	p.logger.Info("Removed worker", "worker_id", workerID, "total_workers", len(p.workers))
	return nil
}

// GetWorkerByID returns a specific worker by ID
func (p *WorkerPool) GetWorkerByID(workerID int) (*Worker, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	worker, exists := p.workers[workerID]
	return worker, exists
}

// start runs a worker with enhanced monitoring and batching
func (w *Worker) start(wg *sync.WaitGroup) {
	defer wg.Done()
	defer w.cleanup()

	w.updateState(WorkerStateIdle)
	w.logger.Info("Enhanced worker started", "worker_id", w.id)

	// Start heartbeat goroutine
	go w.heartbeat()

	batchBuffer := make([]*QueueItem, 0, w.pool.config.TaskBatchSize)
	batchTimer := time.NewTimer(time.Second)
	defer batchTimer.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("Worker stopping gracefully", "worker_id", w.id)
			w.updateState(WorkerStateShuttingDown)
			// Process remaining batch before stopping
			if len(batchBuffer) > 0 {
				w.processBatch(batchBuffer)
			}
			return

		case <-w.shutdownCh:
			w.logger.Warn("Worker force shutdown", "worker_id", w.id)
			w.updateState(WorkerStateStopped)
			return

		case <-batchTimer.C:
			// Process accumulated batch on timer
			if len(batchBuffer) > 0 {
				w.processBatch(batchBuffer)
				batchBuffer = batchBuffer[:0] // Reset buffer
			}
			batchTimer.Reset(time.Second)

		default:
			// Try to get work from load balancer
			item := w.pool.loadBalancer.GetNextTask(w)
			if item == nil {
				// No work available, try queue directly
				var err error
				item, err = w.queue.Dequeue(w.ctx)
				if err != nil {
					if w.ctx.Err() != nil {
						return // Context cancelled
					}
					// No work available, brief idle period
					w.updateState(WorkerStateIdle)
					time.Sleep(50 * time.Millisecond)
					continue
				}
			}

			// Add to batch or process immediately
			if w.pool.config.TaskBatchSize > 1 {
				batchBuffer = append(batchBuffer, item)
				if len(batchBuffer) >= w.pool.config.TaskBatchSize {
					w.processBatch(batchBuffer)
					batchBuffer = batchBuffer[:0] // Reset buffer
					batchTimer.Reset(time.Second)
				}
			} else {
				// Process immediately for single-task batches
				w.processMessage(item)
			}
		}
	}
}

// heartbeat sends periodic heartbeat signals
func (w *Worker) heartbeat() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			select {
			case w.heartbeatCh <- time.Now():
			default:
				// Channel full, skip this heartbeat
			}
		}
	}
}

// cleanup handles worker cleanup
func (w *Worker) cleanup() {
	w.updateState(WorkerStateStopped)
	w.setCurrentTask(nil)
	w.logger.Debug("Worker cleanup completed", "worker_id", w.id)
}

// updateState safely updates worker state
func (w *Worker) updateState(state WorkerState) {
	w.stateMutex.Lock()
	defer w.stateMutex.Unlock()
	w.state = state
	w.lastActivity = time.Now()
}

// setCurrentTask safely sets current task
func (w *Worker) setCurrentTask(item *QueueItem) {
	w.taskMutex.Lock()
	defer w.taskMutex.Unlock()
	w.currentTask = item
}

// isHealthy checks if worker is healthy
func (w *Worker) isHealthy() bool {
	w.stateMutex.RLock()
	defer w.stateMutex.RUnlock()

	// Worker is unhealthy if it hasn't been active recently
	return time.Since(w.lastActivity) < w.pool.config.MaxIdleTime &&
		w.state != WorkerStateStopped
}

// processBatch processes a batch of messages
func (w *Worker) processBatch(batch []*QueueItem) {
	if len(batch) == 0 {
		return
	}

	w.updateState(WorkerStateProcessing)
	w.logger.Debug("Processing message batch", "worker_id", w.id, "batch_size", len(batch))

	for _, item := range batch {
		w.processMessage(item)
	}

	w.updateState(WorkerStateIdle)
}

// processMessage processes a single message with enhanced monitoring
func (w *Worker) processMessage(item *QueueItem) {
	startTime := time.Now()
	w.updateState(WorkerStateProcessing)
	w.setCurrentTask(item)

	w.logger.Debug("Processing message", "worker_id", w.id, "message_id", item.Message.ID)

	// Check if operation was cancelled
	if HandleImpl, ok := item.Handle.(*HandleImpl); ok {
		if HandleImpl.IsCancelled() {
			w.logger.Debug("Message was cancelled", "worker_id", w.id, "message_id", item.Message.ID)
			w.setCurrentTask(nil)
			w.updateState(WorkerStateIdle)
			return
		}

		// Update status to processing
		HandleImpl.SetRunning()
	}

	// Process the message through dispatcher
	receipt, err := w.dispatcher.Dispatch(w.ctx, item.Message)
	duration := time.Since(startTime)

	// Update performance metrics
	w.updatePerformanceMetrics(duration, err == nil)

	if err != nil {
		w.logger.Error("Message processing failed",
			"worker_id", w.id,
			"message_id", item.Message.ID,
			"error", err,
			"duration", duration)
		atomic.AddInt64(&w.errors, 1)

		// Update handle with error
		if HandleImpl, ok := item.Handle.(*HandleImpl); ok {
			HandleImpl.UpdateStatus(StatusFailedOp, 1.0, nil, err)
		}

		// Update batch progress if applicable
		if item.BatchInfo != nil {
			w.updateBatchProgress(item.BatchInfo, false)
		}

		// Increment queue error count
		if memQueue, ok := w.queue.(*memoryAsyncQueue); ok {
			memQueue.IncrementError()
		}
	} else {
		w.logger.Debug("Message processed successfully",
			"worker_id", w.id,
			"message_id", item.Message.ID,
			"duration", duration,
			"successful", receipt.Successful,
			"failed", receipt.Failed)
		atomic.AddInt64(&w.processed, 1)

		// Update handle with success
		if HandleImpl, ok := item.Handle.(*HandleImpl); ok {
			HandleImpl.UpdateStatus(StatusCompleted, 1.0, receipt, nil)
		}

		// Update batch progress if applicable
		if item.BatchInfo != nil {
			w.updateBatchProgress(item.BatchInfo, true)
		}

		// Increment queue processed count
		if memQueue, ok := w.queue.(*memoryAsyncQueue); ok {
			memQueue.IncrementProcessed()
		}
	}

	w.setCurrentTask(nil)
	w.updateState(WorkerStateIdle)
}

// updatePerformanceMetrics updates worker performance tracking
func (w *Worker) updatePerformanceMetrics(duration time.Duration, success bool) {
	w.performance.mutex.Lock()
	defer w.performance.mutex.Unlock()

	// Update average processing time (exponential moving average)
	alpha := 0.1 // Smoothing factor
	if w.performance.AvgProcessingTime == 0 {
		w.performance.AvgProcessingTime = duration
	} else {
		w.performance.AvgProcessingTime = time.Duration(
			float64(w.performance.AvgProcessingTime)*(1-alpha) + float64(duration)*alpha,
		)
	}

	// Update throughput (messages per second)
	processed := atomic.LoadInt64(&w.processed)
	uptime := time.Since(w.startTime).Seconds()
	if uptime > 0 {
		w.performance.Throughput = float64(processed) / uptime
	}

	// Update error rate
	errors := atomic.LoadInt64(&w.errors)
	if processed+errors > 0 {
		w.performance.ErrorRate = float64(errors) / float64(processed+errors)
	}

	w.performance.LastUpdateTime = time.Now()
}

// updateBatchProgress updates progress for batch operations
func (w *Worker) updateBatchProgress(batchInfo *BatchInfo, success bool) {
	if batchHandle, ok := batchInfo.BatchHandle.(*BatchHandleImpl); ok {
		// This is a simplified approach - in a real implementation,
		// you'd need thread-safe tracking of batch completion
		status := batchHandle.Status()
		completed := status.Completed
		failed := status.Failed

		if success {
			completed++
		} else {
			failed++
		}

		batchHandle.UpdateProgress(completed, failed)

		w.logger.Debug("Updated batch progress",
			"worker_id", w.id,
			"batch_id", batchInfo.BatchID,
			"completed", completed,
			"failed", failed,
			"total", batchInfo.TotalItems)
	}
}

// getStats returns comprehensive statistics for this worker
func (w *Worker) getStats() WorkerStats {
	w.stateMutex.RLock()
	state := w.state.String()
	lastActivity := w.lastActivity
	w.stateMutex.RUnlock()

	w.taskMutex.RLock()
	currentTask := ""
	if w.currentTask != nil {
		currentTask = w.currentTask.Message.ID
	}
	w.taskMutex.RUnlock()

	w.performance.mutex.RLock()
	performance := &WorkerPerformance{
		AvgProcessingTime: w.performance.AvgProcessingTime,
		Throughput:        w.performance.Throughput,
		ErrorRate:         w.performance.ErrorRate,
		LastUpdateTime:    w.performance.LastUpdateTime,
	}
	w.performance.mutex.RUnlock()

	// Get system resource usage (simplified)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return WorkerStats{
		ID:           w.id,
		State:        state,
		Processed:    atomic.LoadInt64(&w.processed),
		Errors:       atomic.LoadInt64(&w.errors),
		Uptime:       int64(time.Since(w.startTime).Seconds()),
		LastActivity: lastActivity,
		CurrentTask:  currentTask,
		Affinity:     w.affinity,
		Performance:  performance,
		CPUUsage:     w.getCPUUsage(),
		MemoryUsage:  m.Alloc, // Simplified, should be per-worker
	}
}

// getCPUUsage gets worker CPU usage (simplified implementation)
func (w *Worker) getCPUUsage() float64 {
	// This is a simplified implementation
	// In production, you'd want proper per-goroutine CPU tracking
	processed := atomic.LoadInt64(&w.processed)
	uptime := time.Since(w.startTime).Seconds()
	if uptime > 0 {
		return float64(processed) / uptime / 100.0 // Normalized CPU usage estimate
	}
	return 0.0
}

// SetAffinity sets worker affinity for specialized processing
func (w *Worker) SetAffinity(affinity WorkerAffinity) {
	w.affinity = affinity
	w.logger.Info("Worker affinity updated", "worker_id", w.id, "affinity", affinity)
}

// AsyncExecutor manages the complete asynchronous execution system
type AsyncExecutor struct {
	queue      AsyncQueue
	workerPool *WorkerPool
	callbacks  *CallbackRegistry
	dispatcher core.Dispatcher
	logger     logger.Logger
	started    bool
	mutex      sync.RWMutex
	config     *WorkerPoolConfig
}

// NewAsyncExecutor creates a new async executor with enhanced configuration
func NewAsyncExecutor(queueSize int, config *WorkerPoolConfig, dispatcher core.Dispatcher, logger logger.Logger) *AsyncExecutor {
	if config == nil {
		config = DefaultWorkerPoolConfig()
	}

	// Create queue
	queue := NewMemoryAsyncQueue(queueSize, logger)

	// Create callback registry
	callbacks := NewCallbackRegistry(logger)

	// Create worker pool
	workerPool := NewWorkerPool(config, queue, dispatcher, callbacks, logger)

	return &AsyncExecutor{
		queue:      queue,
		workerPool: workerPool,
		callbacks:  callbacks,
		dispatcher: dispatcher,
		logger:     logger,
		config:     config,
	}
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(strategy LoadBalanceStrategy, pool *WorkerPool, logger logger.Logger) *LoadBalancer {
	return &LoadBalancer{
		strategy:   strategy,
		workerPool: pool,
		logger:     logger,
	}
}

// GetNextTask gets the next task for a worker based on load balancing strategy
func (lb *LoadBalancer) GetNextTask(worker *Worker) *QueueItem {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	// For now, return nil to use direct queue access
	// In a full implementation, this would implement different strategies
	return nil
}

// SelectWorker selects the best worker for a task
func (lb *LoadBalancer) SelectWorker(item *QueueItem) *Worker {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	switch lb.strategy {
	case LeastConnections:
		return lb.selectLeastBusyWorker()
	case AffinityBased:
		return lb.selectAffinityBasedWorker(item)
	case WeightedRoundRobin:
		return lb.selectWeightedWorker()
	default:
		return lb.selectRoundRobinWorker()
	}
}

// selectLeastBusyWorker selects worker with least current load
func (lb *LoadBalancer) selectLeastBusyWorker() *Worker {
	var bestWorker *Worker
	leastBusy := float64(1.1) // Start above max possible load

	for _, worker := range lb.workerPool.workers {
		worker.stateMutex.RLock()
		load := 0.0
		if worker.state == WorkerStateProcessing {
			load = 1.0
		}
		worker.stateMutex.RUnlock()

		if load < leastBusy && worker.isHealthy() {
			leastBusy = load
			bestWorker = worker
		}
	}

	return bestWorker
}

// selectAffinityBasedWorker selects worker based on affinity
func (lb *LoadBalancer) selectAffinityBasedWorker(item *QueueItem) *Worker {
	// Check for workers with matching affinity
	for _, worker := range lb.workerPool.workers {
		if worker.affinity.Specialized && worker.isHealthy() {
			// Check platform affinity
			for _, platform := range worker.affinity.Platforms {
				if item.Message.Targets[0].Platform == platform {
					return worker
				}
			}
		}
	}

	// Fallback to least connections
	return lb.selectLeastBusyWorker()
}

// selectRoundRobinWorker selects worker in round-robin fashion
func (lb *LoadBalancer) selectRoundRobinWorker() *Worker {
	// Simple round-robin implementation
	workers := make([]*Worker, 0, len(lb.workerPool.workers))
	for _, worker := range lb.workerPool.workers {
		if worker.isHealthy() {
			workers = append(workers, worker)
		}
	}

	if len(workers) == 0 {
		return nil
	}

	// Use worker ID as simple counter for round-robin
	index := int(time.Now().UnixNano()) % len(workers)
	return workers[index]
}

// selectWeightedWorker selects worker based on performance weights
func (lb *LoadBalancer) selectWeightedWorker() *Worker {
	// Simple implementation: prefer workers with better performance
	var bestWorker *Worker
	bestThroughput := float64(-1)

	for _, worker := range lb.workerPool.workers {
		if !worker.isHealthy() {
			continue
		}

		worker.performance.mutex.RLock()
		throughput := worker.performance.Throughput
		worker.performance.mutex.RUnlock()

		if throughput > bestThroughput {
			bestThroughput = throughput
			bestWorker = worker
		}
	}

	return bestWorker
}

// contains checks if slice contains string
func contains(item string, slice []string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Start starts the enhanced async executor
func (e *AsyncExecutor) Start() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.started {
		return fmt.Errorf("executor already started")
	}

	e.logger.Info("Starting enhanced async executor")
	if err := e.workerPool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	e.started = true
	return nil
}

// Stop stops the async executor
func (e *AsyncExecutor) Stop(timeout time.Duration) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.started {
		return nil
	}

	e.logger.Info("Stopping async executor")

	// Stop worker pool
	if err := e.workerPool.Stop(timeout); err != nil {
		e.logger.Error("Failed to stop worker pool", "error", err)
	}

	// Close queue
	if err := e.queue.Close(); err != nil {
		e.logger.Error("Failed to close queue", "error", err)
	}

	e.started = false
	return nil
}

// GetQueue returns the queue
func (e *AsyncExecutor) GetQueue() AsyncQueue {
	return e.queue
}

// GetCallbackRegistry returns the callback registry
func (e *AsyncExecutor) GetCallbackRegistry() *CallbackRegistry {
	return e.callbacks
}

// IsHealthy returns whether the executor is healthy
func (e *AsyncExecutor) IsHealthy() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return e.started && e.workerPool.IsHealthy() && e.queue.Health().Healthy
}

// GetStats returns comprehensive executor statistics
func (e *AsyncExecutor) GetStats() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return map[string]interface{}{
		"started":      e.started,
		"healthy":      e.IsHealthy(),
		"worker_pool":  e.workerPool.GetDetailedStats(),
		"queue_health": e.queue.Health(),
	}
}

// GetWorkerPoolConfig returns the worker pool configuration
func (e *AsyncExecutor) GetWorkerPoolConfig() *WorkerPoolConfig {
	return e.config
}

// ScaleWorkers manually scales the worker pool
func (e *AsyncExecutor) ScaleWorkers(targetWorkers int) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.started {
		return fmt.Errorf("executor not started")
	}

	currentWorkers := len(e.workerPool.workers)
	if targetWorkers > currentWorkers {
		// Scale up
		for i := 0; i < targetWorkers-currentWorkers; i++ {
			if err := e.workerPool.AddWorker(); err != nil {
				return fmt.Errorf("failed to add worker: %w", err)
			}
		}
	} else if targetWorkers < currentWorkers {
		// Scale down
		workersToRemove := currentWorkers - targetWorkers
		for workerID := range e.workerPool.workers {
			if workersToRemove <= 0 {
				break
			}
			if err := e.workerPool.RemoveWorker(workerID); err != nil {
				return fmt.Errorf("failed to remove worker %d: %w", workerID, err)
			}
			workersToRemove--
		}
	}

	e.logger.Info("Worker pool scaled", "from", currentWorkers, "to", targetWorkers)
	return nil
}

// NewWorkerMonitor creates a new worker monitor
func NewWorkerMonitor(pool *WorkerPool, logger logger.Logger) *WorkerMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerMonitor{
		pool:           pool,
		logger:         logger,
		monitoringCh:   make(chan WorkerEvent, 100),
		heartbeatTimer: time.NewTicker(pool.config.HealthCheckTime),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start starts the worker monitor
func (wm *WorkerMonitor) Start() error {
	wm.logger.Info("Starting worker monitor")

	wm.wg.Add(1)
	go wm.monitorLoop()

	return nil
}

// Stop stops the worker monitor
func (wm *WorkerMonitor) Stop(timeout time.Duration) error {
	wm.logger.Info("Stopping worker monitor")

	wm.cancel()
	wm.heartbeatTimer.Stop()

	done := make(chan struct{})
	go func() {
		wm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("monitor stop timeout")
	}
}

// monitorLoop runs the main monitoring loop
func (wm *WorkerMonitor) monitorLoop() {
	defer wm.wg.Done()

	for {
		select {
		case <-wm.ctx.Done():
			return

		case <-wm.heartbeatTimer.C:
			wm.checkWorkerHealth()

		case event := <-wm.monitoringCh:
			wm.handleWorkerEvent(event)
		}
	}
}

// checkWorkerHealth performs periodic health checks
func (wm *WorkerMonitor) checkWorkerHealth() {
	wm.pool.mutex.RLock()
	defer wm.pool.mutex.RUnlock()

	for _, worker := range wm.pool.workers {
		if !worker.isHealthy() {
			wm.logger.Warn("Unhealthy worker detected",
				"worker_id", worker.id,
				"last_activity", worker.lastActivity)

			// Emit unhealthy event
			event := WorkerEvent{
				WorkerID:  worker.id,
				EventType: WorkerTimeout,
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"last_activity": worker.lastActivity},
			}

			select {
			case wm.monitoringCh <- event:
			default:
				// Channel full, skip event
			}
		}
	}
}

// handleWorkerEvent handles worker events
func (wm *WorkerMonitor) handleWorkerEvent(event WorkerEvent) {
	wm.logger.Debug("Worker event",
		"worker_id", event.WorkerID,
		"event_type", event.EventType,
		"timestamp", event.Timestamp)

	// Handle specific event types
	switch event.EventType {
	case WorkerTimeout:
		// Consider restarting worker or scaling
		wm.logger.Warn("Worker timeout detected", "worker_id", event.WorkerID)
	case WorkerError:
		// Log error for analysis
		wm.logger.Error("Worker error", "worker_id", event.WorkerID, "data", event.Data)
	}
}

// NewWorkerScaler creates a new worker scaler
func NewWorkerScaler(pool *WorkerPool, logger logger.Logger) *WorkerScaler {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerScaler{
		pool:          pool,
		logger:        logger,
		scalingTimer:  time.NewTicker(pool.config.ScaleUpDelay),
		lastScaleUp:   time.Now(),
		lastScaleDown: time.Now(),
		ctx:           ctx,
		cancel:        cancel,
		loadHistory:   make([]float64, 0, 10),
	}
}

// Start starts the worker scaler
func (ws *WorkerScaler) Start() error {
	ws.logger.Info("Starting worker scaler")

	ws.wg.Add(1)
	go ws.scalingLoop()

	return nil
}

// Stop stops the worker scaler
func (ws *WorkerScaler) Stop(timeout time.Duration) error {
	ws.logger.Info("Stopping worker scaler")

	ws.cancel()
	ws.scalingTimer.Stop()

	done := make(chan struct{})
	go func() {
		ws.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("scaler stop timeout")
	}
}

// scalingLoop runs the main scaling loop
func (ws *WorkerScaler) scalingLoop() {
	defer ws.wg.Done()

	for {
		select {
		case <-ws.ctx.Done():
			return

		case <-ws.scalingTimer.C:
			ws.evaluateScaling()
		}
	}
}

// evaluateScaling evaluates whether to scale up or down
func (ws *WorkerScaler) evaluateScaling() {
	currentLoad := ws.pool.calculateCurrentLoad()
	queueHealth := ws.pool.queue.Health()

	// Add to load history
	ws.mutex.Lock()
	ws.loadHistory = append(ws.loadHistory, currentLoad)
	if len(ws.loadHistory) > 10 {
		ws.loadHistory = ws.loadHistory[1:]
	}
	ws.mutex.Unlock()

	// Calculate average load
	avgLoad := ws.calculateAverageLoad()

	ws.logger.Debug("Scaling evaluation",
		"current_load", currentLoad,
		"avg_load", avgLoad,
		"target_load", ws.pool.config.TargetLoad,
		"queue_size", queueHealth.Size,
		"workers", len(ws.pool.workers))

	// Scale up if load is consistently high
	if avgLoad > ws.pool.config.TargetLoad && len(ws.pool.workers) < ws.pool.config.MaxWorkers {
		if time.Since(ws.lastScaleUp) > ws.pool.config.ScaleUpDelay {
			ws.scaleUp()
		}
	}

	// Scale down if load is consistently low
	if avgLoad < ws.pool.config.TargetLoad*0.5 && len(ws.pool.workers) > ws.pool.config.MinWorkers {
		if time.Since(ws.lastScaleDown) > ws.pool.config.ScaleDownDelay {
			ws.scaleDown()
		}
	}
}

// calculateAverageLoad calculates average load over history
func (ws *WorkerScaler) calculateAverageLoad() float64 {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	if len(ws.loadHistory) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, load := range ws.loadHistory {
		sum += load
	}
	return sum / float64(len(ws.loadHistory))
}

// scaleUp adds a new worker
func (ws *WorkerScaler) scaleUp() {
	ws.logger.Info("Scaling up worker pool")

	if err := ws.pool.AddWorker(); err != nil {
		ws.logger.Error("Failed to scale up", "error", err)
		return
	}

	ws.lastScaleUp = time.Now()
}

// scaleDown removes a worker
func (ws *WorkerScaler) scaleDown() {
	ws.logger.Info("Scaling down worker pool")

	// Find least busy worker to remove
	ws.pool.mutex.RLock()
	var targetWorker *Worker
	for _, worker := range ws.pool.workers {
		worker.stateMutex.RLock()
		if worker.state == WorkerStateIdle {
			targetWorker = worker
			worker.stateMutex.RUnlock()
			break
		}
		worker.stateMutex.RUnlock()
	}
	ws.pool.mutex.RUnlock()

	if targetWorker != nil {
		if err := ws.pool.RemoveWorker(targetWorker.id); err != nil {
			ws.logger.Error("Failed to scale down", "error", err)
			return
		}
		ws.lastScaleDown = time.Now()
	}
}

// Additional utility methods for the enhanced worker pool

// GetWorkerAffinity returns a worker's affinity settings
func (p *WorkerPool) GetWorkerAffinity(workerID int) (WorkerAffinity, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	worker, exists := p.workers[workerID]
	if !exists {
		return WorkerAffinity{}, fmt.Errorf("worker %d not found", workerID)
	}

	return worker.affinity, nil
}

// SetWorkerAffinity sets a worker's affinity settings
func (p *WorkerPool) SetWorkerAffinity(workerID int, affinity WorkerAffinity) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	worker, exists := p.workers[workerID]
	if !exists {
		return fmt.Errorf("worker %d not found", workerID)
	}

	worker.SetAffinity(affinity)
	return nil
}

// GetLoadBalancingStrategy returns current load balancing strategy
func (p *WorkerPool) GetLoadBalancingStrategy() LoadBalanceStrategy {
	p.loadBalancer.mutex.RLock()
	defer p.loadBalancer.mutex.RUnlock()
	return p.loadBalancer.strategy
}

// SetLoadBalancingStrategy sets the load balancing strategy
func (p *WorkerPool) SetLoadBalancingStrategy(strategy LoadBalanceStrategy) {
	p.loadBalancer.mutex.Lock()
	defer p.loadBalancer.mutex.Unlock()
	p.loadBalancer.strategy = strategy
	p.logger.Info("Load balancing strategy updated", "strategy", strategy)
}

// GetBestWorkerForTask returns the best worker for a specific task
func (p *WorkerPool) GetBestWorkerForTask(item *QueueItem) *Worker {
	return p.loadBalancer.SelectWorker(item)
}
