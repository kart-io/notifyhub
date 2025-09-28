// Package receipt provides receipt management for NotifyHub
package receipt

import (
	"sync"
)

// MemoryStore provides an in-memory implementation of PersistenceStore
// This is suitable for testing and simple deployments
type MemoryStore struct {
	receipts      map[string]*Receipt
	asyncReceipts map[string]*AsyncReceiptTracker
	mutex         sync.RWMutex
}

// NewMemoryStore creates a new in-memory persistence store
func NewMemoryStore() PersistenceStore {
	return &MemoryStore{
		receipts:      make(map[string]*Receipt),
		asyncReceipts: make(map[string]*AsyncReceiptTracker),
	}
}

// Store stores a receipt in memory
func (m *MemoryStore) Store(receipt *Receipt) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Create a deep copy to avoid reference issues
	receiptCopy := *receipt
	receiptCopy.Results = make([]PlatformResult, len(receipt.Results))
	copy(receiptCopy.Results, receipt.Results)

	m.receipts[receipt.MessageID] = &receiptCopy
	return nil
}

// StoreAsync stores an async receipt tracker in memory
func (m *MemoryStore) StoreAsync(asyncReceipt *AsyncReceiptTracker) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Create a deep copy
	trackerCopy := *asyncReceipt
	if asyncReceipt.Receipt != nil {
		receiptCopy := *asyncReceipt.Receipt
		trackerCopy.Receipt = &receiptCopy
	}
	if asyncReceipt.Progress != nil {
		progressCopy := *asyncReceipt.Progress
		trackerCopy.Progress = &progressCopy
	}

	m.asyncReceipts[asyncReceipt.Receipt.MessageID] = &trackerCopy
	return nil
}

// Get retrieves a receipt by message ID
func (m *MemoryStore) Get(messageID string) (*Receipt, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if receipt, ok := m.receipts[messageID]; ok {
		// Return a copy to avoid reference issues
		receiptCopy := *receipt
		receiptCopy.Results = make([]PlatformResult, len(receipt.Results))
		copy(receiptCopy.Results, receipt.Results)
		return &receiptCopy, nil
	}

	return nil, ErrReceiptNotFound
}

// GetAsync retrieves an async receipt tracker by message ID
func (m *MemoryStore) GetAsync(messageID string) (*AsyncReceiptTracker, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if tracker, ok := m.asyncReceipts[messageID]; ok {
		// Return a copy
		trackerCopy := *tracker
		if tracker.Receipt != nil {
			receiptCopy := *tracker.Receipt
			trackerCopy.Receipt = &receiptCopy
		}
		if tracker.Progress != nil {
			progressCopy := *tracker.Progress
			trackerCopy.Progress = &progressCopy
		}
		return &trackerCopy, nil
	}

	return nil, ErrReceiptNotFound
}

// List retrieves receipts matching the given filter
func (m *MemoryStore) List(filter ReceiptFilter) ([]*Receipt, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	results := make([]*Receipt, 0)

	for _, receipt := range m.receipts {
		if m.matchesFilter(receipt, filter) {
			// Create a copy
			receiptCopy := *receipt
			receiptCopy.Results = make([]PlatformResult, len(receipt.Results))
			copy(receiptCopy.Results, receipt.Results)
			results = append(results, &receiptCopy)
		}
	}

	// Apply limit and offset
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(results) {
		results = results[:filter.Limit]
	}

	return results, nil
}

// Delete removes a receipt by message ID
func (m *MemoryStore) Delete(messageID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.receipts, messageID)
	delete(m.asyncReceipts, messageID)
	return nil
}

// BatchStore stores multiple receipts in a batch
func (m *MemoryStore) BatchStore(receipts []*Receipt) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, receipt := range receipts {
		// Create a deep copy
		receiptCopy := *receipt
		receiptCopy.Results = make([]PlatformResult, len(receipt.Results))
		copy(receiptCopy.Results, receipt.Results)

		m.receipts[receipt.MessageID] = &receiptCopy
	}

	return nil
}

// Close closes the memory store (no-op for memory store)
func (m *MemoryStore) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Clear all data
	m.receipts = make(map[string]*Receipt)
	m.asyncReceipts = make(map[string]*AsyncReceiptTracker)

	return nil
}

// matchesFilter checks if a receipt matches the given filter
func (m *MemoryStore) matchesFilter(receipt *Receipt, filter ReceiptFilter) bool {
	// Time range filter
	if filter.StartTime != nil && receipt.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && receipt.Timestamp.After(*filter.EndTime) {
		return false
	}

	// Status filter
	if len(filter.Status) > 0 {
		statusMatch := false
		for _, status := range filter.Status {
			if receipt.Status == status {
				statusMatch = true
				break
			}
		}
		if !statusMatch {
			return false
		}
	}

	// Platform filter
	if len(filter.Platforms) > 0 {
		platformMatch := false
		for _, filterPlatform := range filter.Platforms {
			for _, result := range receipt.Results {
				if result.Platform == filterPlatform {
					platformMatch = true
					break
				}
			}
			if platformMatch {
				break
			}
		}
		if !platformMatch {
			return false
		}
	}

	// Message ID filter
	if len(filter.MessageIDs) > 0 {
		messageIDMatch := false
		for _, messageID := range filter.MessageIDs {
			if receipt.MessageID == messageID {
				messageIDMatch = true
				break
			}
		}
		if !messageIDMatch {
			return false
		}
	}

	return true
}