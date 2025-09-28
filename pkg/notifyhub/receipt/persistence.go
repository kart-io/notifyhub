// Package receipt provides receipt management for NotifyHub
package receipt

import "time"

// PersistenceStore defines the interface for receipt persistence
type PersistenceStore interface {
	Store(receipt *Receipt) error
	StoreAsync(asyncReceipt *AsyncReceiptTracker) error
	Get(messageID string) (*Receipt, error)
	GetAsync(messageID string) (*AsyncReceiptTracker, error)
	List(filter ReceiptFilter) ([]*Receipt, error)
	Delete(messageID string) error
	BatchStore(receipts []*Receipt) error
	Close() error
}

// ReceiptMetrics tracks receipt processing metrics
type ReceiptMetrics struct {
	ProcessedCount    int64         `json:"processed_count"`
	ErrorCount        int64         `json:"error_count"`
	PartialFailures   int64         `json:"partial_failures"`
	AverageProcessing time.Duration `json:"average_processing"`
	LastProcessed     time.Time     `json:"last_processed"`
}

// MetricsTracker handles receipt processing metrics
type MetricsTracker struct {
	metrics ReceiptMetrics
}

// NewMetricsTracker creates a new metrics tracker
func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{
		metrics: ReceiptMetrics{},
	}
}

// UpdateMetrics updates internal processing metrics
func (m *MetricsTracker) UpdateMetrics(receipt *Receipt, processingTime time.Duration) {
	m.metrics.ProcessedCount++
	m.metrics.LastProcessed = time.Now()

	if receipt.Error != nil {
		m.metrics.ErrorCount++
	}

	if receipt.Status == string(StatusPartialFailed) {
		m.metrics.PartialFailures++
	}

	// Update average processing time (simple moving average)
	if m.metrics.ProcessedCount == 1 {
		m.metrics.AverageProcessing = processingTime
	} else {
		m.metrics.AverageProcessing = (m.metrics.AverageProcessing + processingTime) / 2
	}
}

// GetMetrics returns current metrics
func (m *MetricsTracker) GetMetrics() ReceiptMetrics {
	return m.metrics
}

// Reset resets all metrics
func (m *MetricsTracker) Reset() {
	m.metrics = ReceiptMetrics{}
}