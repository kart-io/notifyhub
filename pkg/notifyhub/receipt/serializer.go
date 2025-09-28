// Package receipt provides receipt management for NotifyHub
package receipt

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ReceiptFilter defines filters for querying receipts
type ReceiptFilter struct {
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Status     []string   `json:"status,omitempty"`
	Platforms  []string   `json:"platforms,omitempty"`
	MessageIDs []string   `json:"message_ids,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// ReceiptSerializer handles receipt serialization and filtering
type ReceiptSerializer struct{}

// NewReceiptSerializer creates a new receipt serializer
func NewReceiptSerializer() *ReceiptSerializer {
	return &ReceiptSerializer{}
}

// SerializeReceipt serializes a receipt to JSON format
func (s *ReceiptSerializer) SerializeReceipt(receipt *Receipt) ([]byte, error) {
	return json.Marshal(receipt)
}

// DeserializeReceipt deserializes a receipt from JSON format
func (s *ReceiptSerializer) DeserializeReceipt(data []byte) (*Receipt, error) {
	var receipt Receipt
	err := json.Unmarshal(data, &receipt)
	return &receipt, err
}

// ExportReceipts exports receipts to a writer in JSON format
func (s *ReceiptSerializer) ExportReceipts(writer io.Writer, receipts []*Receipt, filter ReceiptFilter) error {
	filteredReceipts := s.filterReceipts(receipts, filter)

	// Serialize and write
	data, err := json.MarshalIndent(filteredReceipts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal receipts: %w", err)
	}

	_, err = writer.Write(data)
	return err
}

// ImportReceipts imports receipts from a reader in JSON format
func (s *ReceiptSerializer) ImportReceipts(reader io.Reader) ([]*Receipt, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read import data: %w", err)
	}

	var receipts []*Receipt
	if err := json.Unmarshal(data, &receipts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal receipts: %w", err)
	}

	return receipts, nil
}

// FilterReceipts applies filters to a list of receipts
func (s *ReceiptSerializer) FilterReceipts(receipts []*Receipt, filter ReceiptFilter) []*Receipt {
	return s.filterReceipts(receipts, filter)
}

// filterReceipts applies the filter logic to receipts
func (s *ReceiptSerializer) filterReceipts(receipts []*Receipt, filter ReceiptFilter) []*Receipt {
	results := make([]*Receipt, 0)

	for _, receipt := range receipts {
		if s.MatchesFilter(receipt, filter) {
			results = append(results, receipt)
		}
	}

	// Apply limit and offset
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(results) {
		results = results[:filter.Limit]
	}

	return results
}

// MatchesFilter checks if a receipt matches the given filter
func (s *ReceiptSerializer) MatchesFilter(receipt *Receipt, filter ReceiptFilter) bool {
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