// Package receipt provides comprehensive receipt processing for NotifyHub
package receipt

import (
	"context"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// ReceiptStatus represents the status of a receipt
type ReceiptStatus string

const (
	StatusPending    ReceiptStatus = "pending"
	StatusProcessing ReceiptStatus = "processing"
	StatusCompleted  ReceiptStatus = "completed"
	StatusFailed     ReceiptStatus = "failed"
	StatusCancelled  ReceiptStatus = "cancelled"
)

// Processor handles receipt collection, aggregation, and reporting
type Processor struct {
	receipts      map[string]*Receipt
	asyncReceipts map[string]*AsyncReceiptTracker
	subscribers   map[string][]ReceiptSubscriber
	mutex         sync.RWMutex
	logger        logger.Logger

	// Configuration
	maxRetention    time.Duration
	cleanupInterval time.Duration

	// Background cleanup
	stopCh  chan struct{}
	started bool
}

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

// ReceiptReport represents aggregated receipt statistics
type ReceiptReport struct {
	PeriodStart     time.Time                 `json:"period_start"`
	PeriodEnd       time.Time                 `json:"period_end"`
	TotalMessages   int                       `json:"total_messages"`
	TotalTargets    int                       `json:"total_targets"`
	SuccessRate     float64                   `json:"success_rate"`
	FailureRate     float64                   `json:"failure_rate"`
	ByPlatform      map[string]*PlatformStats `json:"by_platform"`
	ByStatus        map[string]int            `json:"by_status"`
	AverageDuration time.Duration             `json:"average_duration"`
	Receipts        []*Receipt                `json:"receipts,omitempty"`
}

// PlatformStats represents statistics for a specific platform
type PlatformStats struct {
	Platform      string        `json:"platform"`
	TotalMessages int           `json:"total_messages"`
	Successful    int           `json:"successful"`
	Failed        int           `json:"failed"`
	SuccessRate   float64       `json:"success_rate"`
	AvgDuration   time.Duration `json:"average_duration"`
}

// NewProcessor creates a new receipt processor
func NewProcessor(logger logger.Logger) *Processor {
	return &Processor{
		receipts:        make(map[string]*Receipt),
		asyncReceipts:   make(map[string]*AsyncReceiptTracker),
		subscribers:     make(map[string][]ReceiptSubscriber),
		logger:          logger,
		maxRetention:    24 * time.Hour, // Keep receipts for 24 hours by default
		cleanupInterval: time.Hour,      // Cleanup every hour
		stopCh:          make(chan struct{}),
	}
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

// ProcessReceipt processes a synchronous receipt
func (p *Processor) ProcessReceipt(receipt *Receipt) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Store receipt
	p.receipts[receipt.MessageID] = receipt

	// Notify subscribers
	if subscribers, ok := p.subscribers[receipt.MessageID]; ok {
		for _, sub := range subscribers {
			go sub.OnReceiptUpdate(receipt)
		}
	}

	p.logger.Debug("Processed receipt",
		"message_id", receipt.MessageID,
		"status", receipt.Status,
		"successful", receipt.Successful,
		"failed", receipt.Failed)
}

// ProcessAsyncReceipt processes an asynchronous receipt
func (p *Processor) ProcessAsyncReceipt(asyncReceipt *AsyncReceipt, handleID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	tracker := &AsyncReceiptTracker{
		Receipt:   asyncReceipt,
		HandleID:  handleID,
		UpdatedAt: time.Now(),
	}

	// Progress information would need to be provided separately to avoid circular dependencies
	// For now, we just track the receipt without detailed progress

	// Store async receipt tracker
	p.asyncReceipts[asyncReceipt.MessageID] = tracker

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

// GetReceipt retrieves a receipt by message ID
func (p *Processor) GetReceipt(messageID string) (*Receipt, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	receipt, ok := p.receipts[messageID]
	return receipt, ok
}

// GetAsyncReceipt retrieves an async receipt tracker by message ID
func (p *Processor) GetAsyncReceipt(messageID string) (*AsyncReceiptTracker, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tracker, ok := p.asyncReceipts[messageID]
	return tracker, ok
}

// GenerateReport generates a receipt report for a specific time period
func (p *Processor) GenerateReport(start, end time.Time, includePlatformStats bool) *ReceiptReport {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	report := &ReceiptReport{
		PeriodStart: start,
		PeriodEnd:   end,
		ByPlatform:  make(map[string]*PlatformStats),
		ByStatus:    make(map[string]int),
		Receipts:    make([]*Receipt, 0),
	}

	var totalDuration time.Duration
	var totalMessages, totalTargets, totalSuccessful int

	// Process all receipts in the time period
	for _, receipt := range p.receipts {
		if receipt.Timestamp.After(start) && receipt.Timestamp.Before(end) {
			report.Receipts = append(report.Receipts, receipt)
			report.ByStatus[receipt.Status]++

			totalMessages++
			totalTargets += receipt.Total
			totalSuccessful += receipt.Successful

			// Calculate platform statistics
			if includePlatformStats {
				for _, result := range receipt.Results {
					if _, ok := report.ByPlatform[result.Platform]; !ok {
						report.ByPlatform[result.Platform] = &PlatformStats{
							Platform: result.Platform,
						}
					}

					stats := report.ByPlatform[result.Platform]
					stats.TotalMessages++
					if result.Success {
						stats.Successful++
					} else {
						stats.Failed++
					}

					// Add to average duration calculation
					totalDuration += result.Duration
				}
			}
		}
	}

	// Calculate summary statistics
	report.TotalMessages = totalMessages
	report.TotalTargets = totalTargets
	if totalTargets > 0 {
		report.SuccessRate = float64(totalSuccessful) / float64(totalTargets) * 100
		report.FailureRate = 100 - report.SuccessRate
	}

	if totalTargets > 0 {
		report.AverageDuration = totalDuration / time.Duration(totalTargets)
	}

	// Calculate platform success rates
	for _, stats := range report.ByPlatform {
		if stats.TotalMessages > 0 {
			stats.SuccessRate = float64(stats.Successful) / float64(stats.TotalMessages) * 100
		}
	}

	p.logger.Info("Generated receipt report",
		"period_start", start.Format(time.RFC3339),
		"period_end", end.Format(time.RFC3339),
		"total_messages", report.TotalMessages,
		"success_rate", report.SuccessRate)

	return report
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

// SetRetentionPeriod sets the receipt retention period
func (p *Processor) SetRetentionPeriod(retention time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.maxRetention = retention
}

// GetStats returns current processor statistics
func (p *Processor) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return map[string]interface{}{
		"sync_receipts_count":  len(p.receipts),
		"async_receipts_count": len(p.asyncReceipts),
		"subscribers_count":    len(p.subscribers),
		"retention_period":     p.maxRetention.String(),
		"cleanup_interval":     p.cleanupInterval.String(),
		"started":              p.started,
	}
}
