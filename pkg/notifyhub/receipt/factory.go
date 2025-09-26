// Package receipt provides factory functions for creating receipt processors
package receipt

import (
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// Factory creates and configures receipt processors
type Factory struct {
	logger logger.Logger
}

// NewFactory creates a new receipt factory
func NewFactory(logger logger.Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateProcessor creates a new receipt processor with the given configuration
func (f *Factory) CreateProcessor(config *ProcessorConfig) *Processor {
	processor := NewProcessor(f.logger)

	if config != nil {
		if config.RetentionPeriod > 0 {
			processor.SetRetentionPeriod(config.RetentionPeriod)
		}
		if config.CleanupInterval > 0 {
			processor.cleanupInterval = config.CleanupInterval
		}
	}

	return processor
}

// ProcessorConfig represents configuration for receipt processor
type ProcessorConfig struct {
	RetentionPeriod time.Duration `json:"retention_period"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// DefaultProcessorConfig returns default processor configuration
func DefaultProcessorConfig() *ProcessorConfig {
	return &ProcessorConfig{
		RetentionPeriod: 24 * time.Hour,
		CleanupInterval: time.Hour,
	}
}

// ReceiptBuilder helps build receipts from platform results
type ReceiptBuilder struct {
	messageID string
	timestamp time.Time
	results   []PlatformResult
}

// NewReceiptBuilder creates a new receipt builder
func NewReceiptBuilder(messageID string) *ReceiptBuilder {
	return &ReceiptBuilder{
		messageID: messageID,
		timestamp: time.Now(),
		results:   make([]PlatformResult, 0),
	}
}

// AddPlatformResults adds results from platform send operations
func (rb *ReceiptBuilder) AddPlatformResults(platformName string, sendResults []*platform.SendResult) *ReceiptBuilder {
	for _, result := range sendResults {
		platformResult := PlatformResult{
			Platform:  platformName,
			Target:    result.Target.Value,
			Success:   result.Success,
			MessageID: result.MessageID,
			Timestamp: time.Now(),
			Duration:  time.Duration(result.Metadata["duration"].(int64)) * time.Millisecond,
		}

		if result.Error != "" {
			platformResult.Error = result.Error
		}

		rb.results = append(rb.results, platformResult)
	}

	return rb
}

// AddSingleResult adds a single platform result
func (rb *ReceiptBuilder) AddSingleResult(platformName, target string, success bool, messageID, errorMsg string, duration time.Duration) *ReceiptBuilder {
	result := PlatformResult{
		Platform:  platformName,
		Target:    target,
		Success:   success,
		MessageID: messageID,
		Error:     errorMsg,
		Timestamp: time.Now(),
		Duration:  duration,
	}

	rb.results = append(rb.results, result)
	return rb
}

// Build creates the final receipt
func (rb *ReceiptBuilder) Build() *Receipt {
	successful := 0
	failed := 0
	var overallError error

	for _, result := range rb.results {
		if result.Success {
			successful++
		} else {
			failed++
			if overallError == nil && result.Error != "" {
				overallError = fmt.Errorf("%s", result.Error)
			}
		}
	}

	status := "success"
	if failed > 0 && successful == 0 {
		status = "failed"
	} else if failed > 0 {
		status = "partial"
	}

	return &Receipt{
		MessageID:  rb.messageID,
		Status:     status,
		Results:    rb.results,
		Successful: successful,
		Failed:     failed,
		Total:      len(rb.results),
		Timestamp:  rb.timestamp,
		Error:      overallError,
	}
}

// AsyncReceiptBuilder helps build async receipts
type AsyncReceiptBuilder struct {
	messageID string
	queuedAt  time.Time
}

// NewAsyncReceiptBuilder creates a new async receipt builder
func NewAsyncReceiptBuilder(messageID string) *AsyncReceiptBuilder {
	return &AsyncReceiptBuilder{
		messageID: messageID,
		queuedAt:  time.Now(),
	}
}

// BuildPending creates a pending async receipt
func (arb *AsyncReceiptBuilder) BuildPending() *AsyncReceipt {
	return &AsyncReceipt{
		MessageID: arb.messageID,
		Status:    string(StatusPending),
		QueuedAt:  arb.queuedAt,
	}
}

// BuildProcessing creates a processing async receipt
func (arb *AsyncReceiptBuilder) BuildProcessing() *AsyncReceipt {
	return &AsyncReceipt{
		MessageID: arb.messageID,
		Status:    string(StatusProcessing),
		QueuedAt:  arb.queuedAt,
	}
}

// BuildCompleted creates a completed async receipt
func (arb *AsyncReceiptBuilder) BuildCompleted() *AsyncReceipt {
	return &AsyncReceipt{
		MessageID: arb.messageID,
		Status:    string(StatusCompleted),
		QueuedAt:  arb.queuedAt,
	}
}

// BuildFailed creates a failed async receipt
func (arb *AsyncReceiptBuilder) BuildFailed() *AsyncReceipt {
	return &AsyncReceipt{
		MessageID: arb.messageID,
		Status:    string(StatusFailed),
		QueuedAt:  arb.queuedAt,
	}
}

// BuildCancelled creates a cancelled async receipt
func (arb *AsyncReceiptBuilder) BuildCancelled() *AsyncReceipt {
	return &AsyncReceipt{
		MessageID: arb.messageID,
		Status:    string(StatusCancelled),
		QueuedAt:  arb.queuedAt,
	}
}

// Integration helpers

// ConvertProgressToReceipt converts progress information to receipt format
// This function accepts progress data instead of async handle to avoid circular dependencies
func ConvertProgressToReceipt(messageID string, processed, failed, successful, total int, startTime time.Time) (*Receipt, error) {
	if messageID == "" {
		return nil, fmt.Errorf("message ID cannot be empty")
	}

	builder := NewReceiptBuilder(messageID)

	status := "processing"
	if processed == total {
		if failed == 0 {
			status = "success"
		} else if successful == 0 {
			status = "failed"
		} else {
			status = "partial"
		}
	}

	// Create a summary platform result representing the async operation
	builder.AddSingleResult(
		"async_batch",
		fmt.Sprintf("%d_targets", total),
		failed == 0,
		messageID,
		"",
		time.Since(startTime),
	)

	receipt := builder.Build()
	receipt.Status = status
	receipt.Successful = successful
	receipt.Failed = failed
	receipt.Total = total

	return receipt, nil
}

// Simple subscriber implementation for logging
type LoggingSubscriber struct {
	logger logger.Logger
}

// NewLoggingSubscriber creates a subscriber that logs receipt updates
func NewLoggingSubscriber(logger logger.Logger) *LoggingSubscriber {
	return &LoggingSubscriber{logger: logger}
}

// OnReceiptUpdate logs receipt updates
func (ls *LoggingSubscriber) OnReceiptUpdate(receipt *Receipt) {
	ls.logger.Info("Receipt updated",
		"message_id", receipt.MessageID,
		"status", receipt.Status,
		"successful", receipt.Successful,
		"failed", receipt.Failed,
		"total", receipt.Total)
}

// OnAsyncReceiptUpdate logs async receipt updates
func (ls *LoggingSubscriber) OnAsyncReceiptUpdate(asyncReceipt *AsyncReceiptTracker) {
	fields := []interface{}{
		"message_id", asyncReceipt.Receipt.MessageID,
		"status", asyncReceipt.Receipt.Status,
		"queued_at", asyncReceipt.Receipt.QueuedAt.Format(time.RFC3339),
	}

	if asyncReceipt.Progress != nil {
		fields = append(fields,
			"progress_percent", asyncReceipt.Progress.ProgressPercent,
			"processed", asyncReceipt.Progress.ProcessedTargets,
			"total", asyncReceipt.Progress.TotalTargets)
	}

	ls.logger.Info("Async receipt updated", fields...)
}
