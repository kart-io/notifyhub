// Package receipt provides receipt management for NotifyHub
package receipt

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// ReceiptStatus represents the status of a receipt
type ReceiptStatus string

const (
	StatusPending       ReceiptStatus = "pending"
	StatusProcessing    ReceiptStatus = "processing"
	StatusCompleted     ReceiptStatus = "completed"
	StatusFailed        ReceiptStatus = "failed"
	StatusPartialFailed ReceiptStatus = "partial_failed"
	StatusCancelled     ReceiptStatus = "cancelled"
)

// AsyncReceiptTracker tracks async operation receipts
type AsyncReceiptTracker struct {
	Receipt   *AsyncReceipt `json:"receipt"`
	HandleID  string        `json:"handle_id"` // Store handle ID instead of handle
	Progress  *ProgressInfo `json:"progress,omitempty"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// ProgressInfo represents progress information for async operations
type ProgressInfo struct {
	TotalTargets        int        `json:"total_targets"`
	ProcessedTargets    int        `json:"processed_targets"`
	SuccessfulTargets   int        `json:"successful_targets"`
	FailedTargets       int        `json:"failed_targets"`
	ProgressPercent     float64    `json:"progress_percent"`
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`
}

// ReceiptSubscriber defines the interface for receipt event subscribers
type ReceiptSubscriber interface {
	OnReceiptUpdate(receipt *Receipt)
	OnAsyncReceiptUpdate(asyncReceipt *AsyncReceiptTracker)
}

// Processor handles receipt collection, aggregation, and reporting
type Processor struct {
	receipts      map[string]*Receipt
	asyncReceipts map[string]*AsyncReceiptTracker
	subscribers   map[string][]ReceiptSubscriber
	mutex         sync.RWMutex
	logger        logger.Logger

	// Components
	aggregator  *ResultAggregator
	serializer  *ReceiptSerializer
	metrics     *MetricsTracker

	// Configuration
	maxRetention    time.Duration
	cleanupInterval time.Duration

	// Persistence
	store PersistenceStore

	// Background cleanup
	stopCh  chan struct{}
	started bool
}

// ProcessorOption defines functional options for processor configuration
type ProcessorOption func(*Processor)

// WithPersistenceStore sets the persistence store
func WithPersistenceStore(store PersistenceStore) ProcessorOption {
	return func(p *Processor) {
		p.store = store
	}
}

// WithAggregationConfig sets the aggregation configuration
func WithAggregationConfig(config AggregationConfig) ProcessorOption {
	return func(p *Processor) {
		p.aggregator = NewResultAggregator(config, p.logger)
	}
}

// WithRetentionPeriod sets the retention period
func WithRetentionPeriod(retention time.Duration) ProcessorOption {
	return func(p *Processor) {
		p.maxRetention = retention
	}
}

// NewProcessor creates a new receipt processor
func NewProcessor(logger logger.Logger, opts ...ProcessorOption) *Processor {
	defaultConfig := AggregationConfig{
		PartialFailureThreshold: 0.5, // 50% failure threshold
		FailureTolerance:        0,
	}

	p := &Processor{
		receipts:        make(map[string]*Receipt),
		asyncReceipts:   make(map[string]*AsyncReceiptTracker),
		subscribers:     make(map[string][]ReceiptSubscriber),
		logger:          logger,
		maxRetention:    24 * time.Hour, // Keep receipts for 24 hours by default
		cleanupInterval: time.Hour,      // Cleanup every hour
		stopCh:          make(chan struct{}),
		aggregator:      NewResultAggregator(defaultConfig, logger),
		serializer:      NewReceiptSerializer(),
		metrics:         NewMetricsTracker(),
	}

	// Apply options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Start starts the receipt processor background tasks
func (p *Processor) Start(ctx context.Context) {
	p.mutex.Lock()
	if p.started {
		p.mutex.Unlock()
		return
	}
	p.started = true
	p.mutex.Unlock()

	p.logger.Info("Starting receipt processor")

	// Start cleanup goroutine
	go p.runCleanup(ctx)
}

// Stop stops the receipt processor
func (p *Processor) Stop() {
	p.mutex.Lock()
	if !p.started {
		p.mutex.Unlock()
		return
	}
	p.started = false
	p.mutex.Unlock()

	p.logger.Info("Stopping receipt processor")
	close(p.stopCh)
}

// ProcessReceipt processes a synchronous receipt with enhanced multi-platform aggregation
func (p *Processor) ProcessReceipt(receipt *Receipt) error {
	start := time.Now()
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Enhanced multi-platform result aggregation
	enhancedReceipt := p.aggregator.AggregateMultiPlatformResults(receipt)

	// Calculate partial failure status
	enhancedReceipt.Status = p.aggregator.CalculateOverallStatus(enhancedReceipt)

	// Store receipt in memory
	p.receipts[enhancedReceipt.MessageID] = enhancedReceipt

	// Persist to store if available
	if p.store != nil {
		if err := p.store.Store(enhancedReceipt); err != nil {
			p.logger.Error("Failed to persist receipt", "error", err, "message_id", enhancedReceipt.MessageID)
			// Don't fail the processing, just log the error
		}
	}

	// Update metrics
	p.metrics.UpdateMetrics(enhancedReceipt, time.Since(start))

	// Notify subscribers
	if subscribers, ok := p.subscribers[enhancedReceipt.MessageID]; ok {
		for _, sub := range subscribers {
			go sub.OnReceiptUpdate(enhancedReceipt)
		}
	}

	p.logger.Debug("Processed receipt",
		"message_id", enhancedReceipt.MessageID,
		"status", enhancedReceipt.Status,
		"successful", enhancedReceipt.Successful,
		"failed", enhancedReceipt.Failed,
		"platforms", p.aggregator.ExtractPlatforms(enhancedReceipt.Results))

	return nil
}

// ProcessAsyncReceipt processes an asynchronous receipt
func (p *Processor) ProcessAsyncReceipt(asyncReceipt *AsyncReceipt, handleID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	tracker := &AsyncReceiptTracker{
		Receipt:   asyncReceipt,
		HandleID:  handleID,
		UpdatedAt: time.Now(),
	}

	// Store async receipt tracker
	p.asyncReceipts[asyncReceipt.MessageID] = tracker

	// Persist to store if available
	if p.store != nil {
		if err := p.store.StoreAsync(tracker); err != nil {
			p.logger.Error("Failed to persist async receipt", "error", err, "message_id", asyncReceipt.MessageID)
		}
	}

	// Notify subscribers
	if subscribers, ok := p.subscribers[asyncReceipt.MessageID]; ok {
		for _, sub := range subscribers {
			go sub.OnAsyncReceiptUpdate(tracker)
		}
	}

	p.logger.Debug("Processed async receipt",
		"message_id", asyncReceipt.MessageID,
		"status", asyncReceipt.Status,
		"progress", tracker.Progress)

	return nil
}

// BatchProcessReceipts processes multiple receipts in a batch for improved performance
func (p *Processor) BatchProcessReceipts(receipts []*Receipt) error {
	if len(receipts) == 0 {
		return nil
	}

	start := time.Now()
	p.mutex.Lock()
	defer p.mutex.Unlock()

	processedReceipts := make([]*Receipt, 0, len(receipts))

	// Process each receipt with aggregation
	for _, receipt := range receipts {
		// Enhanced multi-platform result aggregation
		enhancedReceipt := p.aggregator.AggregateMultiPlatformResults(receipt)

		// Calculate partial failure status
		enhancedReceipt.Status = p.aggregator.CalculateOverallStatus(enhancedReceipt)

		// Store receipt in memory
		p.receipts[enhancedReceipt.MessageID] = enhancedReceipt
		processedReceipts = append(processedReceipts, enhancedReceipt)

		// Update metrics
		p.metrics.UpdateMetrics(enhancedReceipt, time.Since(start)/time.Duration(len(receipts)))
	}

	// Batch persist to store if available
	if p.store != nil {
		if err := p.store.BatchStore(processedReceipts); err != nil {
			p.logger.Error("Failed to batch persist receipts", "error", err, "count", len(processedReceipts))
			// Don't fail the processing, just log the error
		}
	}

	// Notify subscribers for each receipt
	for _, receipt := range processedReceipts {
		if subscribers, ok := p.subscribers[receipt.MessageID]; ok {
			for _, sub := range subscribers {
				go sub.OnReceiptUpdate(receipt)
			}
		}
	}

	p.logger.Info("Batch processed receipts",
		"count", len(processedReceipts),
		"duration", time.Since(start).String())

	return nil
}

// GetReceipt retrieves a receipt by message ID
func (p *Processor) GetReceipt(messageID string) (*Receipt, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	receipt, ok := p.receipts[messageID]
	if !ok && p.store != nil {
		// Try to load from persistence store
		if storedReceipt, err := p.store.Get(messageID); err == nil {
			return storedReceipt, true
		}
	}
	return receipt, ok
}

// GetAsyncReceipt retrieves an async receipt tracker by message ID
func (p *Processor) GetAsyncReceipt(messageID string) (*AsyncReceiptTracker, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tracker, ok := p.asyncReceipts[messageID]
	if !ok && p.store != nil {
		// Try to load from persistence store
		if storedTracker, err := p.store.GetAsync(messageID); err == nil {
			return storedTracker, true
		}
	}
	return tracker, ok
}

// GetReceiptsByFilter retrieves receipts matching the given filter
func (p *Processor) GetReceiptsByFilter(filter ReceiptFilter) []*Receipt {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var allReceipts []*Receipt
	for _, receipt := range p.receipts {
		allReceipts = append(allReceipts, receipt)
	}

	return p.serializer.FilterReceipts(allReceipts, filter)
}

// Subscribe subscribes to receipt updates for a specific message
func (p *Processor) Subscribe(messageID string, subscriber ReceiptSubscriber) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, ok := p.subscribers[messageID]; !ok {
		p.subscribers[messageID] = make([]ReceiptSubscriber, 0)
	}
	p.subscribers[messageID] = append(p.subscribers[messageID], subscriber)

	p.logger.Debug("Added receipt subscriber", "message_id", messageID)
}

// Unsubscribe removes a subscriber for a specific message
func (p *Processor) Unsubscribe(messageID string, subscriber ReceiptSubscriber) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if subscribers, ok := p.subscribers[messageID]; ok {
		// Remove subscriber from slice
		for i, sub := range subscribers {
			if sub == subscriber {
				p.subscribers[messageID] = append(subscribers[:i], subscribers[i+1:]...)
				break
			}
		}

		// Clean up empty subscriber lists
		if len(p.subscribers[messageID]) == 0 {
			delete(p.subscribers, messageID)
		}
	}

	p.logger.Debug("Removed receipt subscriber", "message_id", messageID)
}

// SerializeReceipt serializes a receipt to JSON format
func (p *Processor) SerializeReceipt(receipt *Receipt) ([]byte, error) {
	return p.serializer.SerializeReceipt(receipt)
}

// DeserializeReceipt deserializes a receipt from JSON format
func (p *Processor) DeserializeReceipt(data []byte) (*Receipt, error) {
	return p.serializer.DeserializeReceipt(data)
}

// ExportReceipts exports receipts to a writer in JSON format
func (p *Processor) ExportReceipts(writer io.Writer, filter ReceiptFilter) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var allReceipts []*Receipt
	for _, receipt := range p.receipts {
		allReceipts = append(allReceipts, receipt)
	}

	return p.serializer.ExportReceipts(writer, allReceipts, filter)
}

// ImportReceipts imports receipts from a reader in JSON format
func (p *Processor) ImportReceipts(reader io.Reader) error {
	receipts, err := p.serializer.ImportReceipts(reader)
	if err != nil {
		return fmt.Errorf("failed to import receipts: %w", err)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Import receipts
	for _, receipt := range receipts {
		p.receipts[receipt.MessageID] = receipt

		// Persist if store is available
		if p.store != nil {
			if err := p.store.Store(receipt); err != nil {
				p.logger.Error("Failed to persist imported receipt", "error", err, "message_id", receipt.MessageID)
			}
		}
	}

	p.logger.Info("Imported receipts", "count", len(receipts))
	return nil
}

// AnalyzeFailurePatterns analyzes failure patterns in receipts
func (p *Processor) AnalyzeFailurePatterns(filter ReceiptFilter) map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var allReceipts []*Receipt
	for _, receipt := range p.receipts {
		allReceipts = append(allReceipts, receipt)
	}

	filteredReceipts := p.serializer.FilterReceipts(allReceipts, filter)
	return p.aggregator.AnalyzeFailurePatterns(filteredReceipts)
}

// GetMetrics returns detailed processor metrics
func (p *Processor) GetMetrics() ReceiptMetrics {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.metrics.GetMetrics()
}

// GetStats returns current processor statistics
func (p *Processor) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	metrics := p.metrics.GetMetrics()

	return map[string]interface{}{
		"sync_receipts_count":  len(p.receipts),
		"async_receipts_count": len(p.asyncReceipts),
		"subscribers_count":    len(p.subscribers),
		"retention_period":     p.maxRetention.String(),
		"cleanup_interval":     p.cleanupInterval.String(),
		"started":              p.started,
		"processed_count":      metrics.ProcessedCount,
		"error_count":          metrics.ErrorCount,
		"partial_failures":     metrics.PartialFailures,
		"average_processing":   metrics.AverageProcessing.String(),
		"last_processed":       metrics.LastProcessed.Format(time.RFC3339),
	}
}

// GetAggregationConfig returns the current aggregation configuration
func (p *Processor) GetAggregationConfig() AggregationConfig {
	return p.aggregator.Config
}

// GetRetentionPeriod returns the current retention period
func (p *Processor) GetRetentionPeriod() time.Duration {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.maxRetention
}

// SetRetentionPeriod sets the receipt retention period
func (p *Processor) SetRetentionPeriod(retention time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.maxRetention = retention
}

// Close closes the processor and its persistence store
func (p *Processor) Close() error {
	p.Stop()

	if p.store != nil {
		return p.store.Close()
	}
	return nil
}

// runCleanup performs periodic cleanup of old receipts
func (p *Processor) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.performCleanup()
		}
	}
}

// performCleanup removes old receipts beyond retention period
func (p *Processor) performCleanup() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	cutoff := time.Now().Add(-p.maxRetention)
	cleaned := 0

	// Clean up sync receipts
	for id, receipt := range p.receipts {
		if receipt.Timestamp.Before(cutoff) {
			// Delete from persistence store first if available
			if p.store != nil {
				if err := p.store.Delete(id); err != nil {
					p.logger.Error("Failed to delete receipt from store", "error", err, "message_id", id)
				}
			}

			delete(p.receipts, id)
			delete(p.subscribers, id) // Clean up subscribers too
			cleaned++
		}
	}

	// Clean up async receipts
	for id, tracker := range p.asyncReceipts {
		if tracker.Receipt.QueuedAt.Before(cutoff) {
			delete(p.asyncReceipts, id)
			cleaned++
		}
	}

	if cleaned > 0 {
		p.logger.Debug("Cleaned up old receipts", "count", cleaned, "cutoff", cutoff)
	}
}