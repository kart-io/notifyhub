// Package queue provides worker pool implementation
package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// workerPool manages multiple workers for processing queue messages
type workerPool struct {
	queue        Queue
	handler      MessageHandler
	workers      map[string]Worker
	workerCount  int32
	minWorkers   int
	maxWorkers   int
	scalePolicy  *ScalePolicy
	running      bool
	metrics      *MetricsCollector
	healthErrors []HealthError
	mutex        sync.RWMutex
	stopCh       chan struct{}
	wg           sync.WaitGroup
	logger       logger.Logger
	stats        *PoolStats
	startTime    time.Time
}

// PoolStats contains worker pool statistics
type PoolStats struct {
	WorkerCount      int           `json:"worker_count"`
	ActiveWorkers    int           `json:"active_workers"`
	ProcessedCount   int64         `json:"processed_count"`
	ErrorCount       int64         `json:"error_count"`
	AverageTime      time.Duration `json:"average_time"`
	TotalProcessTime time.Duration `json:"total_process_time"`
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(queue Queue, handler MessageHandler, minWorkers, maxWorkers int, log logger.Logger) WorkerPool {
	if log == nil {
		log = logger.Discard
	}

	if minWorkers <= 0 {
		minWorkers = 1
	}
	if maxWorkers < minWorkers {
		maxWorkers = minWorkers
	}

	return &workerPool{
		queue:      queue,
		handler:    handler,
		workers:    make(map[string]Worker),
		minWorkers: minWorkers,
		maxWorkers: maxWorkers,
		scalePolicy: &ScalePolicy{
			ScaleUpThreshold:   50,
			ScaleDownThreshold: 10,
			ScaleUpStep:        2,
			ScaleDownStep:      1,
			ScaleInterval:      10 * time.Second,
		},
		stopCh:    make(chan struct{}),
		logger:    log,
		stats:     &PoolStats{},
		metrics:   NewMetricsCollector(),
		startTime: time.Now(),
	}
}

// Start starts all workers
func (p *workerPool) Start(ctx context.Context) error {
	p.mutex.Lock()
	if p.running {
		p.mutex.Unlock()
		return fmt.Errorf("worker pool is already running")
	}
	p.running = true
	p.mutex.Unlock()

	p.logger.Info("Starting worker pool", "minWorkers", p.minWorkers, "maxWorkers", p.maxWorkers)

	// Start minimum number of workers
	for i := 0; i < p.minWorkers; i++ {
		if err := p.addWorker(ctx); err != nil {
			p.logger.Error("Failed to add worker", "error", err)
			// Continue with other workers
		}
	}

	// Start auto-scaler
	p.wg.Add(1)
	go p.autoScale(ctx)

	p.logger.Info("Worker pool started", "workerCount", atomic.LoadInt32(&p.workerCount))
	return nil
}

// Stop stops all workers
func (p *workerPool) Stop() error {
	p.mutex.Lock()
	if !p.running {
		p.mutex.Unlock()
		return fmt.Errorf("worker pool is not running")
	}
	p.running = false
	p.mutex.Unlock()

	p.logger.Info("Stopping worker pool")

	close(p.stopCh)

	// Stop all workers
	p.mutex.Lock()
	for id, worker := range p.workers {
		if err := worker.Stop(); err != nil {
			p.logger.Error("Failed to stop worker", "workerID", id, "error", err)
		}
	}
	p.workers = make(map[string]Worker)
	p.mutex.Unlock()

	p.wg.Wait()

	atomic.StoreInt32(&p.workerCount, 0)
	p.logger.Info("Worker pool stopped")
	return nil
}

// Scale adjusts the number of workers
func (p *workerPool) Scale(count int) error {
	if count < p.minWorkers || count > p.maxWorkers {
		return fmt.Errorf("worker count must be between %d and %d", p.minWorkers, p.maxWorkers)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	currentCount := int(atomic.LoadInt32(&p.workerCount))

	p.logger.Info("Scaling worker pool", "currentCount", currentCount, "targetCount", count)

	if count > currentCount {
		// Add workers
		ctx := context.Background()
		for i := currentCount; i < count; i++ {
			if err := p.addWorker(ctx); err != nil {
				p.logger.Error("Failed to add worker during scaling", "error", err)
				break
			}
		}
	} else if count < currentCount {
		// Remove workers
		toRemove := currentCount - count
		removed := 0
		for id, worker := range p.workers {
			if removed >= toRemove {
				break
			}
			if err := worker.Stop(); err != nil {
				p.logger.Error("Failed to stop worker during scaling", "workerID", id, "error", err)
				continue
			}
			delete(p.workers, id)
			atomic.AddInt32(&p.workerCount, -1)
			removed++
			p.logger.Debug("Worker removed", "workerID", id)
		}
	}

	p.logger.Info("Worker pool scaled", "newCount", atomic.LoadInt32(&p.workerCount))
	return nil
}

// GetWorkerCount returns the number of workers
func (p *workerPool) GetWorkerCount() int {
	return int(atomic.LoadInt32(&p.workerCount))
}

// GetStats returns worker pool statistics
func (p *workerPool) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	activeWorkers := 0
	for _, worker := range p.workers {
		if worker.IsRunning() {
			activeWorkers++
		}
	}

	avgTime := time.Duration(0)
	if p.stats.ProcessedCount > 0 {
		avgTime = p.stats.TotalProcessTime / time.Duration(p.stats.ProcessedCount)
	}

	return map[string]interface{}{
		"worker_count":       atomic.LoadInt32(&p.workerCount),
		"active_workers":     activeWorkers,
		"processed_count":    p.stats.ProcessedCount,
		"error_count":        p.stats.ErrorCount,
		"average_time":       avgTime,
		"total_process_time": p.stats.TotalProcessTime,
		"queue_size":         p.queue.Size(),
	}
}

// GetMetrics returns detailed worker metrics
func (p *workerPool) GetMetrics() *WorkerMetrics {
	return p.metrics.GetWorkerMetrics()
}

// SetScalePolicy updates scaling policy
func (p *workerPool) SetScalePolicy(policy *ScalePolicy) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.scalePolicy = policy
}

// GetHealth returns health status
func (p *workerPool) GetHealth() *HealthStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	status := "healthy"
	queueSize := p.queue.Size()
	errorCount := p.stats.ErrorCount
	processedCount := p.stats.ProcessedCount

	// Calculate error rate
	var errorRate float64
	if processedCount > 0 {
		errorRate = float64(errorCount) / float64(processedCount)
	}

	// Determine status
	if errorRate > 0.2 || queueSize > int(p.scalePolicy.ScaleUpThreshold*2) {
		status = "unhealthy"
	} else if errorRate > 0.05 || queueSize > int(p.scalePolicy.ScaleUpThreshold) {
		status = "degraded"
	}

	// Copy errors
	errors := make([]HealthError, len(p.healthErrors))
	copy(errors, p.healthErrors)

	return &HealthStatus{
		Status:      status,
		QueueSize:   queueSize,
		Workers:     int(atomic.LoadInt32(&p.workerCount)),
		LastProcess: time.Now(), // Simplified
		Errors:      errors,
		Uptime:      time.Since(p.startTime),
	}
}

// addWorker adds a new worker to the pool
func (p *workerPool) addWorker(ctx context.Context) error {
	workerID := fmt.Sprintf("worker-%d-%d", time.Now().Unix(), atomic.AddInt32(&p.workerCount, 0))

	// Create worker based on queue type
	var worker Worker
	if retryQueue, ok := p.queue.(RetryQueue); ok {
		worker = NewRetryableWorker(workerID, retryQueue, p.wrapHandler(), 1, p.logger)
	} else {
		worker = &basicWorker{
			id:      workerID,
			queue:   p.queue,
			handler: p.wrapHandler(),
			stopCh:  make(chan struct{}),
			logger:  p.logger,
		}
	}

	if err := worker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker %s: %w", workerID, err)
	}

	p.workers[workerID] = worker
	atomic.AddInt32(&p.workerCount, 1)

	p.logger.Debug("Worker added to pool", "workerID", workerID, "totalWorkers", atomic.LoadInt32(&p.workerCount))
	return nil
}

// wrapHandler wraps the message handler with statistics tracking
func (p *workerPool) wrapHandler() MessageHandler {
	return func(ctx context.Context, msg *Message) error {
		startTime := time.Now()

		err := p.handler(ctx, msg)

		duration := time.Since(startTime)
		atomic.AddInt64(&p.stats.ProcessedCount, 1)
		p.stats.TotalProcessTime += duration

		if err != nil {
			atomic.AddInt64(&p.stats.ErrorCount, 1)
		}

		return err
	}
}

// autoScale automatically scales workers based on queue size
func (p *workerPool) autoScale(ctx context.Context) {
	defer p.wg.Done()

	p.mutex.RLock()
	scaleInterval := p.scalePolicy.ScaleInterval
	p.mutex.RUnlock()

	ticker := time.NewTicker(scaleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.checkAndScale(ctx)
		}
	}
}

// checkAndScale checks queue size and scales workers according to policy
func (p *workerPool) checkAndScale(ctx context.Context) {
	queueSize := p.queue.Size()
	currentWorkers := int(atomic.LoadInt32(&p.workerCount))

	p.mutex.RLock()
	scalePolicy := p.scalePolicy
	p.mutex.RUnlock()

	// Calculate desired worker count based on scaling policy and queue size
	var desiredWorkers int

	// Apply scale-up rules
	if queueSize >= scalePolicy.ScaleUpThreshold {
		// Scale up by configured step
		desiredWorkers = currentWorkers + scalePolicy.ScaleUpStep
	} else if queueSize <= scalePolicy.ScaleDownThreshold && currentWorkers > p.minWorkers {
		// Scale down by configured step
		desiredWorkers = currentWorkers - scalePolicy.ScaleDownStep
	} else {
		// No scaling needed
		desiredWorkers = currentWorkers
	}

	// Ensure within bounds
	if desiredWorkers < p.minWorkers {
		desiredWorkers = p.minWorkers
	} else if desiredWorkers > p.maxWorkers {
		desiredWorkers = p.maxWorkers
	}

	// Scale if needed and enough time has passed since last scaling
	if desiredWorkers != currentWorkers {
		p.logger.Debug("Auto-scaling workers",
			"queueSize", queueSize,
			"currentWorkers", currentWorkers,
			"desiredWorkers", desiredWorkers,
			"scaleUpThreshold", scalePolicy.ScaleUpThreshold,
			"scaleDownThreshold", scalePolicy.ScaleDownThreshold)

		if err := p.Scale(desiredWorkers); err != nil {
			p.logger.Error("Failed to auto-scale workers", "error", err)
			// Record health error
			p.recordHealthError("scaling_failed", err.Error(), "error")
		}
	}

	// Update metrics with current worker status
	p.updateWorkerMetrics()
}

// updateWorkerMetrics updates worker metrics for monitoring
func (p *workerPool) updateWorkerMetrics() {
	activeWorkers := 0
	idleWorkers := 0

	p.mutex.RLock()
	for _, worker := range p.workers {
		if worker.IsRunning() {
			activeWorkers++
		} else {
			idleWorkers++
		}
	}
	p.mutex.RUnlock()

	p.metrics.UpdateWorkerCounts(activeWorkers, idleWorkers)
}

// recordHealthError records a health error for monitoring
func (p *workerPool) recordHealthError(code, message, severity string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	error := HealthError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Severity:  severity,
	}

	// Keep only last 50 errors
	if len(p.healthErrors) >= 50 {
		p.healthErrors = p.healthErrors[1:]
	}

	p.healthErrors = append(p.healthErrors, error)
}

// basicWorker is a simple worker implementation
type basicWorker struct {
	id      string
	queue   Queue
	handler MessageHandler
	running bool
	mutex   sync.RWMutex
	stopCh  chan struct{}
	wg      sync.WaitGroup
	logger  logger.Logger
}

// Start begins processing messages
func (w *basicWorker) Start(ctx context.Context) error {
	w.mutex.Lock()
	if w.running {
		w.mutex.Unlock()
		return fmt.Errorf("worker %s is already running", w.id)
	}
	w.running = true
	w.mutex.Unlock()

	w.logger.Debug("Starting basic worker", "workerID", w.id)

	w.wg.Add(1)
	go w.processMessages(ctx)

	return nil
}

// Stop stops processing messages
func (w *basicWorker) Stop() error {
	w.mutex.Lock()
	if !w.running {
		w.mutex.Unlock()
		return fmt.Errorf("worker %s is not running", w.id)
	}
	w.running = false
	w.mutex.Unlock()

	w.logger.Debug("Stopping basic worker", "workerID", w.id)

	close(w.stopCh)
	w.wg.Wait()

	w.logger.Debug("Basic worker stopped", "workerID", w.id)
	return nil
}

// Process handles a single message
func (w *basicWorker) Process(ctx context.Context, msg *Message) error {
	return w.handler(ctx, msg)
}

// GetID returns the worker ID
func (w *basicWorker) GetID() string {
	return w.id
}

// IsRunning returns true if the worker is running
func (w *basicWorker) IsRunning() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.running
}

// processMessages continuously processes messages from the queue
func (w *basicWorker) processMessages(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
			msg, err := w.queue.Dequeue(ctx)
			if err != nil {
				if err == ErrQueueEmpty {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				w.logger.Error("Failed to dequeue message",
					"workerID", w.id,
					"error", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if err := w.Process(ctx, msg); err != nil {
				w.logger.Error("Message processing failed",
					"workerID", w.id,
					"messageID", msg.ID,
					"error", err)
			}
		}
	}
}
