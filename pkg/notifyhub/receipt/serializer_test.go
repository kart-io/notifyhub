package receipt

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestReceiptSerializer tests comprehensive serialization functionality
func TestReceiptSerializer(t *testing.T) {
	serializer := NewReceiptSerializer()

	t.Run("Basic serialization and deserialization", func(t *testing.T) {
		receipt := &Receipt{
			MessageID: "test-serialize-1",
			Status:    string(StatusCompleted),
			Results: []PlatformResult{
				{
					Platform:  "feishu",
					Target:    "user1",
					Success:   true,
					MessageID: "feishu-msg-1",
					Timestamp: time.Now().Round(time.Second), // Round to avoid precision issues
					Duration:  150 * time.Millisecond,
				},
				{
					Platform:  "email",
					Target:    "user@example.com",
					Success:   false,
					Error:     "timeout",
					Timestamp: time.Now().Round(time.Second),
					Duration:  5 * time.Second,
				},
			},
			Successful: 1,
			Failed:     1,
			Total:      2,
			Timestamp:  time.Now().Round(time.Second),
		}

		// Test serialization
		data, err := serializer.SerializeReceipt(receipt)
		if err != nil {
			t.Fatalf("Failed to serialize receipt: %v", err)
		}

		// Verify JSON structure
		var jsonData map[string]interface{}
		if err := json.Unmarshal(data, &jsonData); err != nil {
			t.Fatalf("Serialized data is not valid JSON: %v", err)
		}

		// Check required fields
		requiredFields := []string{"message_id", "status", "results", "successful", "failed", "total", "timestamp"}
		for _, field := range requiredFields {
			if _, ok := jsonData[field]; !ok {
				t.Errorf("Missing required field in serialized data: %s", field)
			}
		}

		// Test deserialization
		deserialized, err := serializer.DeserializeReceipt(data)
		if err != nil {
			t.Fatalf("Failed to deserialize receipt: %v", err)
		}

		// Verify data integrity
		if deserialized.MessageID != receipt.MessageID {
			t.Errorf("MessageID mismatch: expected %s, got %s", receipt.MessageID, deserialized.MessageID)
		}
		if deserialized.Status != receipt.Status {
			t.Errorf("Status mismatch: expected %s, got %s", receipt.Status, deserialized.Status)
		}
		if len(deserialized.Results) != len(receipt.Results) {
			t.Errorf("Results length mismatch: expected %d, got %d", len(receipt.Results), len(deserialized.Results))
		}
		if deserialized.Successful != receipt.Successful {
			t.Errorf("Successful count mismatch: expected %d, got %d", receipt.Successful, deserialized.Successful)
		}
		if deserialized.Failed != receipt.Failed {
			t.Errorf("Failed count mismatch: expected %d, got %d", receipt.Failed, deserialized.Failed)
		}
		if deserialized.Total != receipt.Total {
			t.Errorf("Total count mismatch: expected %d, got %d", receipt.Total, deserialized.Total)
		}

		// Verify platform results
		for i, original := range receipt.Results {
			if i >= len(deserialized.Results) {
				t.Fatalf("Missing result at index %d", i)
			}
			result := deserialized.Results[i]
			if result.Platform != original.Platform {
				t.Errorf("Platform mismatch at index %d: expected %s, got %s", i, original.Platform, result.Platform)
			}
			if result.Target != original.Target {
				t.Errorf("Target mismatch at index %d: expected %s, got %s", i, original.Target, result.Target)
			}
			if result.Success != original.Success {
				t.Errorf("Success mismatch at index %d: expected %v, got %v", i, original.Success, result.Success)
			}
			if result.Error != original.Error {
				t.Errorf("Error mismatch at index %d: expected %s, got %s", i, original.Error, result.Error)
			}
		}
	})

	t.Run("Large receipt serialization performance", func(t *testing.T) {
		// Create a large receipt with many results
		results := make([]PlatformResult, 1000)
		for i := 0; i < 1000; i++ {
			results[i] = PlatformResult{
				Platform:  "platform_" + string(rune(i%10+'0')),
				Target:    "target_" + string(rune(i)),
				Success:   i%3 != 0,
				MessageID: "msg_" + string(rune(i)),
				Error:     "",
				Timestamp: time.Now(),
				Duration:  time.Duration(i) * time.Millisecond,
			}
			if !results[i].Success {
				results[i].Error = "error_" + string(rune(i%5+'0'))
			}
		}

		receipt := &Receipt{
			MessageID:  "large-receipt",
			Status:     string(StatusPartialFailed),
			Results:    results,
			Successful: 667, // Approximately 2/3 success
			Failed:     333,
			Total:      1000,
			Timestamp:  time.Now(),
		}

		// Test serialization performance
		start := time.Now()
		data, err := serializer.SerializeReceipt(receipt)
		serializationDuration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to serialize large receipt: %v", err)
		}

		// Test deserialization performance
		start = time.Now()
		deserialized, err := serializer.DeserializeReceipt(data)
		deserializationDuration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to deserialize large receipt: %v", err)
		}

		// Verify correctness
		if len(deserialized.Results) != 1000 {
			t.Errorf("Expected 1000 results, got %d", len(deserialized.Results))
		}

		// Performance assertions
		if serializationDuration > 500*time.Millisecond {
			t.Errorf("Serialization took too long: %v", serializationDuration)
		}
		if deserializationDuration > 500*time.Millisecond {
			t.Errorf("Deserialization took too long: %v", deserializationDuration)
		}

		t.Logf("Serialized 1000 results in %v, deserialized in %v", serializationDuration, deserializationDuration)
		t.Logf("Serialized size: %d bytes", len(data))
	})

	t.Run("Export and import functionality", func(t *testing.T) {
		receipts := []*Receipt{
			{
				MessageID: "export-test-1",
				Status:    string(StatusCompleted),
				Results: []PlatformResult{
					{Platform: "feishu", Target: "user1", Success: true, Timestamp: time.Now()},
				},
				Successful: 1,
				Failed:     0,
				Total:      1,
				Timestamp:  time.Now(),
			},
			{
				MessageID: "export-test-2",
				Status:    string(StatusFailed),
				Results: []PlatformResult{
					{Platform: "email", Target: "user@example.com", Success: false, Error: "timeout", Timestamp: time.Now()},
				},
				Successful: 0,
				Failed:     1,
				Total:      1,
				Timestamp:  time.Now(),
			},
		}

		// Test export
		var buffer bytes.Buffer
		filter := ReceiptFilter{} // No filter

		err := serializer.ExportReceipts(&buffer, receipts, filter)
		if err != nil {
			t.Fatalf("Failed to export receipts: %v", err)
		}

		// Verify exported JSON is valid and properly formatted
		exportedData := buffer.String()
		if !strings.HasPrefix(exportedData, "[\n") {
			t.Error("Exported data should start with formatted JSON array")
		}

		var exportedReceipts []*Receipt
		if err := json.Unmarshal(buffer.Bytes(), &exportedReceipts); err != nil {
			t.Fatalf("Exported data is not valid JSON: %v", err)
		}

		if len(exportedReceipts) != 2 {
			t.Errorf("Expected 2 exported receipts, got %d", len(exportedReceipts))
		}

		// Test import
		importedReceipts, err := serializer.ImportReceipts(&buffer)
		if err != nil {
			t.Fatalf("Failed to import receipts: %v", err)
		}

		if len(importedReceipts) != 2 {
			t.Errorf("Expected 2 imported receipts, got %d", len(importedReceipts))
		}

		// Verify data integrity
		for i, original := range receipts {
			imported := importedReceipts[i]
			if imported.MessageID != original.MessageID {
				t.Errorf("Import failed: MessageID mismatch at index %d", i)
			}
			if imported.Status != original.Status {
				t.Errorf("Import failed: Status mismatch at index %d", i)
			}
		}
	})

	t.Run("Filter functionality", func(t *testing.T) {
		now := time.Now()
		receipts := []*Receipt{
			{
				MessageID: "filter-test-1",
				Status:    string(StatusCompleted),
				Results: []PlatformResult{
					{Platform: "feishu", Target: "user1", Success: true},
				},
				Timestamp: now.Add(-2 * time.Hour),
			},
			{
				MessageID: "filter-test-2",
				Status:    string(StatusFailed),
				Results: []PlatformResult{
					{Platform: "email", Target: "user@example.com", Success: false},
				},
				Timestamp: now.Add(-1 * time.Hour),
			},
			{
				MessageID: "filter-test-3",
				Status:    string(StatusPartialFailed),
				Results: []PlatformResult{
					{Platform: "webhook", Target: "http://example.com", Success: false},
					{Platform: "sms", Target: "1234567890", Success: true},
				},
				Timestamp: now,
			},
		}

		tests := []struct {
			name           string
			filter         ReceiptFilter
			expectedCount  int
			expectedIDs    []string
		}{
			{
				name:          "No filter",
				filter:        ReceiptFilter{},
				expectedCount: 3,
				expectedIDs:   []string{"filter-test-1", "filter-test-2", "filter-test-3"},
			},
			{
				name: "Time range filter",
				filter: ReceiptFilter{
					StartTime: &[]time.Time{now.Add(-90 * time.Minute)}[0],
					EndTime:   &[]time.Time{now.Add(-30 * time.Minute)}[0],
				},
				expectedCount: 1,
				expectedIDs:   []string{"filter-test-2"},
			},
			{
				name: "Status filter",
				filter: ReceiptFilter{
					Status: []string{string(StatusCompleted), string(StatusPartialFailed)},
				},
				expectedCount: 2,
				expectedIDs:   []string{"filter-test-1", "filter-test-3"},
			},
			{
				name: "Platform filter",
				filter: ReceiptFilter{
					Platforms: []string{"email", "webhook"},
				},
				expectedCount: 2,
				expectedIDs:   []string{"filter-test-2", "filter-test-3"},
			},
			{
				name: "Message ID filter",
				filter: ReceiptFilter{
					MessageIDs: []string{"filter-test-1", "filter-test-3"},
				},
				expectedCount: 2,
				expectedIDs:   []string{"filter-test-1", "filter-test-3"},
			},
			{
				name: "Limit and offset",
				filter: ReceiptFilter{
					Limit:  2,
					Offset: 1,
				},
				expectedCount: 2,
				expectedIDs:   []string{"filter-test-2", "filter-test-3"},
			},
			{
				name: "Complex filter - multiple conditions",
				filter: ReceiptFilter{
					Status:    []string{string(StatusCompleted), string(StatusFailed)},
					StartTime: &[]time.Time{now.Add(-3 * time.Hour)}[0],
					Limit:     1,
				},
				expectedCount: 1,
				expectedIDs:   []string{"filter-test-1"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				filtered := serializer.FilterReceipts(receipts, tt.filter)

				if len(filtered) != tt.expectedCount {
					t.Errorf("Expected %d filtered receipts, got %d", tt.expectedCount, len(filtered))
				}

				// Verify expected IDs
				for i, expectedID := range tt.expectedIDs {
					if i >= len(filtered) {
						t.Errorf("Missing expected receipt %s at index %d", expectedID, i)
						continue
					}
					if filtered[i].MessageID != expectedID {
						t.Errorf("Expected receipt ID %s at index %d, got %s", expectedID, i, filtered[i].MessageID)
					}
				}
			})
		}
	})

	t.Run("Edge cases and error handling", func(t *testing.T) {
		// Test empty receipt serialization
		emptyReceipt := &Receipt{
			MessageID: "empty",
			Results:   []PlatformResult{},
			Timestamp: time.Now(),
		}

		data, err := serializer.SerializeReceipt(emptyReceipt)
		if err != nil {
			t.Fatalf("Failed to serialize empty receipt: %v", err)
		}

		deserialized, err := serializer.DeserializeReceipt(data)
		if err != nil {
			t.Fatalf("Failed to deserialize empty receipt: %v", err)
		}

		if len(deserialized.Results) != 0 {
			t.Errorf("Expected empty results, got %d", len(deserialized.Results))
		}

		// Test nil receipt handling - json.Marshal(nil) returns "null" which is valid JSON
		data, err = serializer.SerializeReceipt(nil)
		if err != nil {
			t.Errorf("Unexpected error when serializing nil receipt: %v", err)
		}
		if string(data) != "null" {
			t.Errorf("Expected 'null' for nil receipt, got %s", string(data))
		}

		// Test invalid JSON deserialization
		invalidJSON := []byte(`{"invalid": json}`)
		_, err = serializer.DeserializeReceipt(invalidJSON)
		if err == nil {
			t.Error("Expected error when deserializing invalid JSON")
		}

		// Test empty data deserialization
		_, err = serializer.DeserializeReceipt([]byte{})
		if err == nil {
			t.Error("Expected error when deserializing empty data")
		}

		// Test malformed JSON
		malformedJSON := []byte(`{"message_id": "test", "results": [invalid]}`)
		_, err = serializer.DeserializeReceipt(malformedJSON)
		if err == nil {
			t.Error("Expected error when deserializing malformed JSON")
		}
	})

	t.Run("Filter matching edge cases", func(t *testing.T) {
		receipt := &Receipt{
			MessageID: "edge-case-test",
			Status:    string(StatusCompleted),
			Results: []PlatformResult{
				{Platform: "feishu", Success: true},
			},
			Timestamp: time.Now(),
		}

		// Test with empty filters
		emptyFilters := []ReceiptFilter{
			{Status: []string{}},
			{Platforms: []string{}},
			{MessageIDs: []string{}},
		}

		for i, filter := range emptyFilters {
			t.Run("Empty filter "+string(rune(i+'0')), func(t *testing.T) {
				matches := serializer.MatchesFilter(receipt, filter)
				if !matches {
					t.Error("Receipt should match empty filter")
				}
			})
		}

		// Test with non-matching filters
		nonMatchingFilters := []ReceiptFilter{
			{Status: []string{"invalid_status"}},
			{Platforms: []string{"invalid_platform"}},
			{MessageIDs: []string{"invalid_id"}},
		}

		for i, filter := range nonMatchingFilters {
			t.Run("Non-matching filter "+string(rune(i+'0')), func(t *testing.T) {
				matches := serializer.MatchesFilter(receipt, filter)
				if matches {
					t.Error("Receipt should not match non-matching filter")
				}
			})
		}

		// Test time edge cases
		now := time.Now()
		timeFilters := []struct {
			name     string
			filter   ReceiptFilter
			expected bool
		}{
			{
				name: "Exact start time",
				filter: ReceiptFilter{
					StartTime: &receipt.Timestamp,
				},
				expected: true,
			},
			{
				name: "Exact end time",
				filter: ReceiptFilter{
					EndTime: &receipt.Timestamp,
				},
				expected: true,
			},
			{
				name: "Future start time",
				filter: ReceiptFilter{
					StartTime: &[]time.Time{now.Add(1 * time.Hour)}[0],
				},
				expected: false,
			},
			{
				name: "Past end time",
				filter: ReceiptFilter{
					EndTime: &[]time.Time{now.Add(-1 * time.Hour)}[0],
				},
				expected: false,
			},
		}

		for _, tt := range timeFilters {
			t.Run(tt.name, func(t *testing.T) {
				matches := serializer.MatchesFilter(receipt, tt.filter)
				if matches != tt.expected {
					t.Errorf("Expected filter match %v, got %v", tt.expected, matches)
				}
			})
		}
	})

	t.Run("Concurrent serialization safety", func(t *testing.T) {
		receipt := &Receipt{
			MessageID: "concurrent-test",
			Status:    string(StatusCompleted),
			Results: []PlatformResult{
				{Platform: "feishu", Target: "user1", Success: true, Timestamp: time.Now()},
			},
			Successful: 1,
			Failed:     0,
			Total:      1,
			Timestamp:  time.Now(),
		}

		// Test concurrent serialization
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer func() { done <- true }()

				for j := 0; j < 100; j++ {
					data, err := serializer.SerializeReceipt(receipt)
					if err != nil {
						errors <- err
						return
					}

					_, err = serializer.DeserializeReceipt(data)
					if err != nil {
						errors <- err
						return
					}
				}
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Success
			case err := <-errors:
				t.Fatalf("Concurrent serialization failed: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("Concurrent serialization test timed out")
			}
		}
	})
}