// Package async provides worker pool for async processing
package async

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart/notifyhub/pkg/utils/logger"
)

// Worker processes queue items
type Worker struct {
	id        int
	items     <-chan *QueueItem
	logger    logger.Logger
	quit      chan bool
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// NewWorker creates a new worker
func NewWorker(id int, items <-chan *QueueItem) *Worker {
	return &Worker{
		id:     id,
		items:  items,
		logger: logger.New(),
		quit:   make(chan bool),
	}
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) {
	w.wg.Add(1)
	defer w.wg.Done()

	w.logger.Info("Worker started", "worker_id", w.id)

	for {
		select {
		case item, ok := <-w.items:
			if !ok {
				w.logger.Info("Worker channel closed, stopping", "worker_id", w.id)
				return
			}
			w.processItem(ctx, item)

		case <-w.quit:
			w.logger.Info("Worker stopping", "worker_id", w.id)
			return

		case <-ctx.Done():
			w.logger.Info("Worker context cancelled", "worker_id", w.id)
			return
		}
	}
}

// Stop stops the worker
func (w *Worker) Stop() {
	w.closeOnce.Do(func() {
		close(w.quit)
	})
	w.wg.Wait()
}

// processItem processes a single queue item
func (w *Worker) processItem(ctx context.Context, item *QueueItem) {
	w.logger.Debug("Processing item", "worker_id", w.id, "item_id", item.ID)

	var result Result

	// Execute the item's processor function if available
	if item.Processor != nil {
		result = item.Processor(ctx, item.Message, item.Targets)
	} else {
		// Handle items without processor (create error result)
		w.logger.Error("No processor function for item", "worker_id", w.id, "item_id", item.ID)
		result = Result{
			Receipt: nil,
			Error:   fmt.Errorf("no processor function available for queue item %s", item.ID),
		}
	}

	// Send result to the handle if available
	if item.Handle != nil {
		if memHandle, ok := item.Handle.(*MemoryHandle); ok {
			memHandle.SetResultWithCallback(result, item.Message)
		}
	}

	w.logger.Debug("Item processed", "worker_id", w.id, "item_id", item.ID)
}

// WorkerPool manages a pool of workers
type WorkerPool struct {
	workers []*Worker
	config  WorkerPoolConfig
	logger  logger.Logger
	mu      sync.Mutex
	items   <-chan *QueueItem
	ctx     context.Context
}

// WorkerPoolConfig configures the worker pool
type WorkerPoolConfig struct {
	MinWorkers int `json:"min_workers"`
	MaxWorkers int `json:"max_workers"`
	ScaleUp    int `json:"scale_up_threshold"`
	ScaleDown  int `json:"scale_down_threshold"`
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(config WorkerPoolConfig) *WorkerPool {
	if config.MinWorkers <= 0 {
		config.MinWorkers = 2
	}
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 10
	}

	return &WorkerPool{
		config: config,
		logger: logger.New(),
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start(ctx context.Context, items <-chan *QueueItem) error {
	wp.ctx = ctx
	wp.items = items

	wp.logger.Info("Starting worker pool", "min_workers", wp.config.MinWorkers)

	// Start initial workers
	wp.workers = make([]*Worker, wp.config.MinWorkers)
	for i := 0; i < wp.config.MinWorkers; i++ {
		worker := NewWorker(i, items)
		wp.workers[i] = worker
		go worker.Start(ctx)
	}

	return nil
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.logger.Info("Stopping worker pool", "worker_count", len(wp.workers))

	for i := range wp.workers {
		wp.workers[i].Stop()
	}
}

// Scale scales the worker pool up or down
func (wp *WorkerPool) Scale(targetWorkers int) error {
	currentWorkers := len(wp.workers)

	if targetWorkers > wp.config.MaxWorkers {
		targetWorkers = wp.config.MaxWorkers
	}
	if targetWorkers < wp.config.MinWorkers {
		targetWorkers = wp.config.MinWorkers
	}

	if targetWorkers == currentWorkers {
		return nil
	}

	wp.logger.Info("Scaling worker pool", "current", currentWorkers, "target", targetWorkers)

	// Implement scaling logic
	if targetWorkers > currentWorkers {
		// Scale up: add new workers
		return wp.scaleUp(targetWorkers - currentWorkers)
	}

	// Scale down: remove excess workers
	return wp.scaleDown(currentWorkers - targetWorkers)
}

// scaleUp adds new workers to the pool
func (wp *WorkerPool) scaleUp(count int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.logger.Info("Scaling up worker pool", "additional_workers", count)

	currentCount := len(wp.workers)

	for i := 0; i < count; i++ {
		workerID := currentCount + i + 1
		worker := NewWorker(workerID, wp.items)

		// Start worker in a goroutine
		go worker.Start(wp.ctx)

		wp.workers = append(wp.workers, worker)
		wp.logger.Debug("Added new worker", "worker_id", workerID, "total_workers", len(wp.workers))
	}

	wp.logger.Info("Worker pool scaled up successfully",
		"added", count,
		"total_workers", len(wp.workers))

	return nil
}

// scaleDown removes workers from the pool
func (wp *WorkerPool) scaleDown(count int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if count <= 0 {
		return nil
	}

	wp.logger.Info("Scaling down worker pool", "workers_to_remove", count)

	currentCount := len(wp.workers)
	if count > currentCount {
		count = currentCount
	}

	// Stop the last 'count' workers
	workersToStop := wp.workers[currentCount-count:]
	wp.workers = wp.workers[:currentCount-count]

	// Stop workers gracefully
	for _, worker := range workersToStop {
		worker.Stop()
		wp.logger.Debug("Stopped worker", "worker_id", worker.id)
	}

	wp.logger.Info("Worker pool scaled down successfully",
		"removed", count,
		"remaining_workers", len(wp.workers))

	return nil
}

// GetStats returns worker pool statistics
func (wp *WorkerPool) GetStats() WorkerPoolStats {
	return WorkerPoolStats{
		ActiveWorkers: len(wp.workers),
		MinWorkers:    wp.config.MinWorkers,
		MaxWorkers:    wp.config.MaxWorkers,
		UpdatedAt:     time.Now(),
	}
}

// WorkerPoolStats provides worker pool statistics
type WorkerPoolStats struct {
	ActiveWorkers int       `json:"active_workers"`
	MinWorkers    int       `json:"min_workers"`
	MaxWorkers    int       `json:"max_workers"`
	UpdatedAt     time.Time `json:"updated_at"`
}
