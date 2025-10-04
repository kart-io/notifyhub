// Package core provides core processing logic for NotifyHub
package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart/notifyhub/pkg/async"
	"github.com/kart/notifyhub/pkg/errors"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/receipt"
	"github.com/kart/notifyhub/pkg/target"
	"github.com/kart/notifyhub/pkg/utils/logger"
	"github.com/kart/notifyhub/pkg/utils/metrics"
)

// Dispatcher handles message dispatch and coordination
type Dispatcher struct {
	manager    *PublicPlatformManager
	router     Router
	asyncQueue async.Queue
	workerPool *async.WorkerPool
	logger     logger.Logger
	metrics    metrics.Metrics

	// Statistics
	stats      *DispatcherStats
	statsMutex sync.RWMutex
}

// DispatcherStats contains dispatcher statistics
type DispatcherStats struct {
	TotalMessages   int64         `json:"total_messages"`
	SuccessfulSends int64         `json:"successful_sends"`
	FailedSends     int64         `json:"failed_sends"`
	AsyncMessages   int64         `json:"async_messages"`
	BatchMessages   int64         `json:"batch_messages"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastMessageTime time.Time     `json:"last_message_time"`
}

// NewDispatcher creates a new message dispatcher
func NewDispatcher(manager *PublicPlatformManager, router Router, log logger.Logger) (*Dispatcher, error) {
	if manager == nil {
		return nil, fmt.Errorf("manager cannot be nil")
	}
	if router == nil {
		return nil, fmt.Errorf("router cannot be nil")
	}
	if log == nil {
		log = logger.New()
	}

	// Create async queue with default configuration
	var queue async.Queue
	var workerPool *async.WorkerPool

	// Use memory queue with default config
	queue = async.NewMemoryQueue(async.QueueConfig{
		Workers:    4,
		BufferSize: 1000,
		Timeout:    30 * time.Second,
	})

	// Create worker pool with default config
	workerPool = async.NewWorkerPool(async.WorkerPoolConfig{
		MinWorkers: 2,
		MaxWorkers: 4,
	})

	dispatcher := &Dispatcher{
		manager:    manager,
		router:     router,
		asyncQueue: queue,
		workerPool: workerPool,
		logger:     log,
		metrics:    metrics.GetDefaultMetrics(),
		stats:      &DispatcherStats{},
	}

	return dispatcher, nil
}

// Send sends a message synchronously
func (d *Dispatcher) Send(ctx context.Context, msg *message.Message, targets ...target.Target) (*receipt.Receipt, error) {
	startTime := time.Now()

	d.logger.Debug("Dispatching message", map[string]interface{}{
		"message_id": msg.ID,
		"targets":    len(targets),
	})

	// Update statistics
	d.updateStats(func(s *DispatcherStats) {
		s.TotalMessages++
		s.LastMessageTime = time.Now()
	})

	// Record metrics
	timer := metrics.NewTimer(metrics.MetricMessagesDuration, map[string]string{
		"type": "sync",
	})
	defer timer.Stop()

	// Validate message
	if err := d.validateMessage(msg); err != nil {
		d.updateStats(func(s *DispatcherStats) { s.FailedSends++ })
		metrics.RecordMessageFailed("dispatcher", "validation_error", timer.Stop())
		return nil, fmt.Errorf("message validation failed: %w", err)
	}

	// Use provided targets or return error if empty
	if len(targets) == 0 {
		d.updateStats(func(s *DispatcherStats) { s.FailedSends++ })
		metrics.RecordMessageFailed("dispatcher", "no_targets_error", timer.Stop())
		return nil, fmt.Errorf("no targets provided for message")
	}

	// Route targets to platforms
	platformTargets, err := d.router.Route(targets)
	if err != nil {
		d.updateStats(func(s *DispatcherStats) { s.FailedSends++ })
		metrics.RecordMessageFailed("dispatcher", "routing_error", timer.Stop())
		return nil, fmt.Errorf("failed to route message: %w", err)
	}

	// Send to platforms
	receipts := make([]*receipt.Receipt, 0, len(platformTargets))
	var errors []error

	for platformName, platformTargets := range platformTargets {
		platformReceipt, err := d.manager.Send(ctx, platformName, msg, platformTargets)
		if err != nil {
			errors = append(errors, fmt.Errorf("platform %s: %w", platformName, err))
			continue
		}
		receipts = append(receipts, platformReceipt)
	}

	// Check if any sends succeeded
	if len(receipts) == 0 {
		d.updateStats(func(s *DispatcherStats) { s.FailedSends++ })
		metrics.RecordMessageFailed("dispatcher", "all_platforms_failed", timer.Stop())
		return nil, fmt.Errorf("all platforms failed: %v", errors)
	}

	// Create consolidated receipt
	consolidatedReceipt := d.consolidateReceipts(msg, receipts, errors)

	// Update statistics
	d.updateStats(func(s *DispatcherStats) {
		s.SuccessfulSends++
		s.AverageLatency = d.calculateAverageLatency(s.AverageLatency, time.Since(startTime))
	})

	metrics.RecordMessageSent("dispatcher", timer.Stop())

	d.logger.Info("Message dispatched successfully", map[string]interface{}{
		"message_id":  msg.ID,
		"receipt_id":  consolidatedReceipt.MessageID,
		"platforms":   len(platformTargets),
		"duration_ms": time.Since(startTime).Milliseconds(),
	})

	return consolidatedReceipt, nil
}

// SendAsync sends a message asynchronously
func (d *Dispatcher) SendAsync(ctx context.Context, msg *message.Message, targets ...target.Target) (async.Handle, error) {
	if d.asyncQueue == nil {
		return nil, fmt.Errorf("async processing not enabled")
	}

	d.logger.Debug("Dispatching message asynchronously", map[string]interface{}{
		"message_id": msg.ID,
		"targets":    len(targets),
	})

	// Update statistics
	d.updateStats(func(s *DispatcherStats) {
		s.TotalMessages++
		s.AsyncMessages++
		s.LastMessageTime = time.Now()
	})

	// Validate message
	if err := d.validateMessage(msg); err != nil {
		d.updateStats(func(s *DispatcherStats) { s.FailedSends++ })
		return nil, fmt.Errorf("message validation failed: %w", err)
	}

	// Create async task
	// Queue the message for async processing
	handle, err := d.asyncQueue.Enqueue(ctx, msg, targets)
	if err != nil {
		d.updateStats(func(s *DispatcherStats) { s.FailedSends++ })
		return nil, fmt.Errorf("failed to queue message: %w", err)
	}

	d.logger.Info("Message queued for async processing", map[string]interface{}{
		"message_id": msg.ID,
		"handle_id":  handle.ID(),
	})

	return handle, nil
}

// SendBatch sends multiple messages
func (d *Dispatcher) SendBatch(ctx context.Context, messages []*message.Message) ([]*receipt.Receipt, error) {
	startTime := time.Now()

	d.logger.Debug("Dispatching batch messages", map[string]interface{}{
		"batch_size": len(messages),
	})

	// Update statistics
	d.updateStats(func(s *DispatcherStats) {
		s.TotalMessages += int64(len(messages))
		s.BatchMessages += int64(len(messages))
		s.LastMessageTime = time.Now()
	})

	receipts := make([]*receipt.Receipt, 0, len(messages))
	var errors []error

	// Send each message
	for _, msg := range messages {
		receipt, err := d.Send(ctx, msg)
		if err != nil {
			errors = append(errors, fmt.Errorf("message %s: %w", msg.ID, err))
			continue
		}
		receipts = append(receipts, receipt)
	}

	d.logger.Info("Batch messages dispatched", map[string]interface{}{
		"batch_size":  len(messages),
		"successful":  len(receipts),
		"failed":      len(errors),
		"duration_ms": time.Since(startTime).Milliseconds(),
	})

	if len(errors) > 0 {
		d.logger.Warn("Some messages in batch failed", map[string]interface{}{
			"errors": errors,
		})
	}

	return receipts, nil
}

// validateMessage validates a message before dispatch
func (d *Dispatcher) validateMessage(msg *message.Message) error {
	if msg == nil {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message cannot be nil",
		}
	}

	if msg.ID == "" {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message ID cannot be empty",
		}
	}

	if msg.Body == "" && msg.Title == "" {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message must have title or body",
		}
	}

	return nil
}

// consolidateReceipts creates a consolidated receipt from multiple platform receipts
func (d *Dispatcher) consolidateReceipts(msg *message.Message, receipts []*receipt.Receipt, errors []error) *receipt.Receipt {
	// Collect all platform results
	var allResults []receipt.PlatformResult
	totalSuccessful := 0
	totalFailed := 0

	for _, rcpt := range receipts {
		allResults = append(allResults, rcpt.Results...)
		totalSuccessful += rcpt.Successful
		totalFailed += rcpt.Failed
	}

	// Add failed platforms due to errors
	totalFailed += len(errors)

	status := receipt.StatusSuccess
	if totalFailed > 0 {
		if totalSuccessful > 0 {
			status = receipt.StatusPartial
		} else {
			status = receipt.StatusFailed
		}
	}

	consolidatedReceipt := &receipt.Receipt{
		MessageID:  msg.ID,
		Status:     status,
		Results:    allResults,
		Successful: totalSuccessful,
		Failed:     totalFailed,
		Total:      totalSuccessful + totalFailed,
		Timestamp:  time.Now(),
	}

	// Status is already set correctly above based on success/failure counts

	return consolidatedReceipt
}

// updateStats safely updates dispatcher statistics
func (d *Dispatcher) updateStats(updateFunc func(*DispatcherStats)) {
	d.statsMutex.Lock()
	defer d.statsMutex.Unlock()
	updateFunc(d.stats)
}

// calculateAverageLatency calculates rolling average latency
func (d *Dispatcher) calculateAverageLatency(currentAvg, newLatency time.Duration) time.Duration {
	// Simple moving average with weight 0.9 for historical data
	return time.Duration(float64(currentAvg)*0.9 + float64(newLatency)*0.1)
}

// GetStats returns current dispatcher statistics
func (d *Dispatcher) GetStats() *DispatcherStats {
	d.statsMutex.RLock()
	defer d.statsMutex.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := *d.stats
	return &statsCopy
}

// Health checks the health of the dispatcher
func (d *Dispatcher) Health(ctx context.Context) error {
	// Basic health check - verify components exist
	if d.manager == nil {
		return fmt.Errorf("manager is nil")
	}
	if d.router == nil {
		return fmt.Errorf("router is nil")
	}

	// Check if we can list platforms from manager
	platforms := d.manager.ListSenders()
	if len(platforms) == 0 {
		return fmt.Errorf("no platforms available")
	}

	return nil
}

// Close shuts down the dispatcher
func (d *Dispatcher) Close() error {
	d.logger.Info("Shutting down dispatcher")

	// Reset components (they don't have Close methods)
	d.workerPool = nil
	d.asyncQueue = nil

	d.logger.Info("Dispatcher shut down successfully")
	return nil
}
