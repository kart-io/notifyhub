// Package async provides worker pool for asynchronous message processing
package async

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
)

// WorkerPool manages a pool of workers for processing async messages
type WorkerPool struct {
	workers    []*Worker
	queue      AsyncQueue
	dispatcher core.Dispatcher
	callbacks  *CallbackRegistry
	logger     logger.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	stopped    bool
	mutex      sync.RWMutex
}

// Worker represents a single worker in the pool
type Worker struct {
	id         int
	pool       *WorkerPool
	queue      AsyncQueue
	dispatcher core.Dispatcher
	callbacks  *CallbackRegistry
	logger     logger.Logger
	ctx        context.Context
	processed  int64
	errors     int64
	startTime  time.Time
}

// WorkerStats represents statistics for a worker
type WorkerStats struct {
	ID        int   `json:"id"`
	Processed int64 `json:"processed"`
	Errors    int64 `json:"errors"`
	Uptime    int64 `json:"uptime_seconds"`
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workerCount int, queue AsyncQueue, dispatcher core.Dispatcher, callbacks *CallbackRegistry, logger logger.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		workers:    make([]*Worker, workerCount),
		queue:      queue,
		dispatcher: dispatcher,
		callbacks:  callbacks,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Create workers
	for i := 0; i < workerCount; i++ {
		worker := &Worker{
			id:         i,
			pool:       pool,
			queue:      queue,
			dispatcher: dispatcher,
			callbacks:  callbacks,
			logger:     logger,
			ctx:        ctx,
			startTime:  time.Now(),
		}
		pool.workers[i] = worker
	}

	return pool
}

// Start starts all workers in the pool
func (p *WorkerPool) Start() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.stopped {
		p.logger.Error("Cannot start stopped worker pool")
		return
	}

	p.logger.Info("Starting worker pool", "worker_count", len(p.workers))

	for _, worker := range p.workers {
		p.wg.Add(1)
		go worker.start(&p.wg)
	}

	p.logger.Info("Worker pool started")
}

// Stop gracefully stops all workers
func (p *WorkerPool) Stop(timeout time.Duration) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.stopped {
		return nil
	}

	p.logger.Info("Stopping worker pool", "timeout", timeout)
	p.stopped = true

	// Signal all workers to stop
	p.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("Worker pool stopped gracefully")
		return nil
	case <-time.After(timeout):
		p.logger.Warn("Worker pool stop timeout exceeded")
		return fmt.Errorf("timeout waiting for workers to stop")
	}
}

// GetStats returns statistics for all workers
func (p *WorkerPool) GetStats() []WorkerStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	stats := make([]WorkerStats, len(p.workers))
	for i, worker := range p.workers {
		stats[i] = worker.getStats()
	}
	return stats
}

// GetTotalStats returns aggregated statistics
func (p *WorkerPool) GetTotalStats() (processed int64, errors int64) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, worker := range p.workers {
		workerStats := worker.getStats()
		processed += workerStats.Processed
		errors += workerStats.Errors
	}
	return
}

// IsHealthy returns whether the worker pool is healthy
func (p *WorkerPool) IsHealthy() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return !p.stopped
}

// start runs a single worker
func (w *Worker) start(wg *sync.WaitGroup) {
	defer wg.Done()

	w.logger.Info("Worker started", "worker_id", w.id)

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("Worker stopping", "worker_id", w.id)
			return
		default:
			// Try to get work from queue
			item, err := w.queue.Dequeue(w.ctx)
			if err != nil {
				if w.ctx.Err() != nil {
					// Context cancelled, normal shutdown
					return
				}
				w.logger.Error("Failed to dequeue message", "worker_id", w.id, "error", err)
				w.errors++
				time.Sleep(100 * time.Millisecond) // Brief pause before retry
				continue
			}

			// Process the message
			w.processMessage(item)
		}
	}
}

// processMessage processes a single message
func (w *Worker) processMessage(item *QueueItem) {
	startTime := time.Now()
	w.logger.Debug("Processing message", "worker_id", w.id, "message_id", item.Message.ID)

	// Check if operation was cancelled
	if HandleImpl, ok := item.Handle.(*HandleImpl); ok {
		if HandleImpl.IsCancelled() {
			w.logger.Debug("Message was cancelled", "worker_id", w.id, "message_id", item.Message.ID)
			return
		}

		// Update status to processing
		HandleImpl.UpdateStatus(StatusProcessing, 0.2, nil, nil)
	}

	// Process the message through dispatcher
	receipt, err := w.dispatcher.Dispatch(w.ctx, item.Message)
	duration := time.Since(startTime)

	if err != nil {
		w.logger.Error("Message processing failed",
			"worker_id", w.id,
			"message_id", item.Message.ID,
			"error", err,
			"duration", duration)
		w.errors++

		// Update handle with error
		if HandleImpl, ok := item.Handle.(*HandleImpl); ok {
			HandleImpl.UpdateStatus(StatusFailed, 1.0, nil, err)
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
		w.processed++

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

// getStats returns statistics for this worker
func (w *Worker) getStats() WorkerStats {
	return WorkerStats{
		ID:        w.id,
		Processed: w.processed,
		Errors:    w.errors,
		Uptime:    int64(time.Since(w.startTime).Seconds()),
	}
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
}

// NewAsyncExecutor creates a new async executor
func NewAsyncExecutor(queueSize int, workerCount int, dispatcher core.Dispatcher, logger logger.Logger) *AsyncExecutor {
	// Create queue
	queue := NewMemoryAsyncQueue(queueSize, logger)

	// Create callback registry
	callbacks := NewCallbackRegistry(logger)

	// Create worker pool
	workerPool := NewWorkerPool(workerCount, queue, dispatcher, callbacks, logger)

	return &AsyncExecutor{
		queue:      queue,
		workerPool: workerPool,
		callbacks:  callbacks,
		dispatcher: dispatcher,
		logger:     logger,
	}
}

// Start starts the async executor
func (e *AsyncExecutor) Start() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.started {
		return fmt.Errorf("executor already started")
	}

	e.logger.Info("Starting async executor")
	e.workerPool.Start()
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

// GetStats returns executor statistics
func (e *AsyncExecutor) GetStats() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	processed, errors := e.workerPool.GetTotalStats()
	queueHealth := e.queue.Health()

	return map[string]interface{}{
		"started":         e.started,
		"healthy":         e.IsHealthy(),
		"queue_size":      queueHealth.Size,
		"queue_max_size":  queueHealth.MaxSize,
		"total_processed": processed,
		"total_errors":    errors,
		"worker_stats":    e.workerPool.GetStats(),
	}
}
