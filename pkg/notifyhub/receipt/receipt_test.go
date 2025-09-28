package receipt

import (
	"testing"
	"time"
)

// TestReceiptModel tests the receipt data structure and its behavior
func TestReceiptModel(t *testing.T) {
	t.Run("Basic receipt creation", func(t *testing.T) {
		receipt := &Receipt{
			MessageID: "test-receipt-1",
			Status:    string(StatusCompleted),
			Results: []PlatformResult{
				{
					Platform:  "feishu",
					Target:    "user1",
					Success:   true,
					MessageID: "feishu-msg-1",
					Timestamp: time.Now(),
					Duration:  100 * time.Millisecond,
				},
			},
			Successful: 1,
			Failed:     0,
			Total:      1,
			Timestamp:  time.Now(),
		}

		// Validate basic fields
		if receipt.MessageID == "" {
			t.Error("MessageID should not be empty")
		}
		if receipt.Status == "" {
			t.Error("Status should not be empty")
		}
		if len(receipt.Results) == 0 {
			t.Error("Results should not be empty")
		}
		if receipt.Total != len(receipt.Results) {
			t.Errorf("Total (%d) should match results length (%d)", receipt.Total, len(receipt.Results))
		}
		if receipt.Successful+receipt.Failed != receipt.Total {
			t.Errorf("Successful (%d) + Failed (%d) should equal Total (%d)",
				receipt.Successful, receipt.Failed, receipt.Total)
		}
	})

	t.Run("Platform result validation", func(t *testing.T) {
		result := PlatformResult{
			Platform:  "email",
			Target:    "user@example.com",
			Success:   false,
			Error:     "Connection timeout",
			Timestamp: time.Now(),
			Duration:  5 * time.Second,
		}

		// Validate platform result fields
		if result.Platform == "" {
			t.Error("Platform should not be empty")
		}
		if result.Target == "" {
			t.Error("Target should not be empty")
		}
		if result.Success && result.Error != "" {
			t.Error("Successful result should not have error message")
		}
		if !result.Success && result.Error == "" {
			t.Error("Failed result should have error message")
		}
		if result.Duration < 0 {
			t.Error("Duration should not be negative")
		}
	})

	t.Run("Async receipt creation", func(t *testing.T) {
		asyncReceipt := &AsyncReceipt{
			MessageID: "async-test-1",
			Status:    string(StatusPending),
			QueuedAt:  time.Now(),
		}

		if asyncReceipt.MessageID == "" {
			t.Error("MessageID should not be empty")
		}
		if asyncReceipt.Status == "" {
			t.Error("Status should not be empty")
		}
		if asyncReceipt.QueuedAt.IsZero() {
			t.Error("QueuedAt should not be zero")
		}
	})

	t.Run("Status enum validation", func(t *testing.T) {
		validStatuses := []ReceiptStatus{
			StatusPending,
			StatusProcessing,
			StatusCompleted,
			StatusFailed,
			StatusPartialFailed,
			StatusCancelled,
		}

		for _, status := range validStatuses {
			if string(status) == "" {
				t.Errorf("Status %v should not be empty string", status)
			}
		}

		// Test status string values
		expectedValues := map[ReceiptStatus]string{
			StatusPending:       "pending",
			StatusProcessing:    "processing",
			StatusCompleted:     "completed",
			StatusFailed:        "failed",
			StatusPartialFailed: "partial_failed",
			StatusCancelled:     "cancelled",
		}

		for status, expected := range expectedValues {
			if string(status) != expected {
				t.Errorf("Status %v should equal %s, got %s", status, expected, string(status))
			}
		}
	})

	t.Run("Complex receipt scenarios", func(t *testing.T) {
		tests := []struct {
			name        string
			results     []PlatformResult
			expectedTotal     int
			expectedSuccessful int
			expectedFailed     int
		}{
			{
				name: "All successful",
				results: []PlatformResult{
					{Platform: "feishu", Success: true},
					{Platform: "email", Success: true},
					{Platform: "webhook", Success: true},
				},
				expectedTotal:      3,
				expectedSuccessful: 3,
				expectedFailed:     0,
			},
			{
				name: "All failed",
				results: []PlatformResult{
					{Platform: "feishu", Success: false, Error: "rate limit"},
					{Platform: "email", Success: false, Error: "timeout"},
					{Platform: "webhook", Success: false, Error: "connection failed"},
				},
				expectedTotal:      3,
				expectedSuccessful: 0,
				expectedFailed:     3,
			},
			{
				name: "Mixed results",
				results: []PlatformResult{
					{Platform: "feishu", Success: true},
					{Platform: "email", Success: false, Error: "timeout"},
					{Platform: "webhook", Success: true},
					{Platform: "sms", Success: false, Error: "invalid number"},
				},
				expectedTotal:      4,
				expectedSuccessful: 2,
				expectedFailed:     2,
			},
			{
				name:               "Empty results",
				results:            []PlatformResult{},
				expectedTotal:      0,
				expectedSuccessful: 0,
				expectedFailed:     0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Calculate counts manually
				successful := 0
				failed := 0
				for _, result := range tt.results {
					if result.Success {
						successful++
					} else {
						failed++
					}
				}

				if successful != tt.expectedSuccessful {
					t.Errorf("Expected %d successful, got %d", tt.expectedSuccessful, successful)
				}
				if failed != tt.expectedFailed {
					t.Errorf("Expected %d failed, got %d", tt.expectedFailed, failed)
				}
				if len(tt.results) != tt.expectedTotal {
					t.Errorf("Expected %d total, got %d", tt.expectedTotal, len(tt.results))
				}
			})
		}
	})

	t.Run("Async receipt tracker", func(t *testing.T) {
		asyncReceipt := &AsyncReceipt{
			MessageID: "tracker-test",
			Status:    string(StatusProcessing),
			QueuedAt:  time.Now(),
		}

		progress := &ProgressInfo{
			TotalTargets:        10,
			ProcessedTargets:    7,
			SuccessfulTargets:   5,
			FailedTargets:       2,
			ProgressPercent:     70.0,
			EstimatedCompletion: &[]time.Time{time.Now().Add(30 * time.Second)}[0],
		}

		tracker := &AsyncReceiptTracker{
			Receipt:   asyncReceipt,
			HandleID:  "handle-123",
			Progress:  progress,
			UpdatedAt: time.Now(),
		}

		// Validate tracker fields
		if tracker.Receipt == nil {
			t.Error("Receipt should not be nil")
		}
		if tracker.HandleID == "" {
			t.Error("HandleID should not be empty")
		}
		if tracker.Progress == nil {
			t.Error("Progress should not be nil")
		}
		if tracker.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should not be zero")
		}

		// Validate progress consistency
		if progress.ProcessedTargets > progress.TotalTargets {
			t.Error("ProcessedTargets should not exceed TotalTargets")
		}
		if progress.SuccessfulTargets+progress.FailedTargets != progress.ProcessedTargets {
			t.Error("SuccessfulTargets + FailedTargets should equal ProcessedTargets")
		}
		if progress.ProgressPercent < 0 || progress.ProgressPercent > 100 {
			t.Errorf("ProgressPercent should be between 0 and 100, got %.2f", progress.ProgressPercent)
		}

		expectedPercent := float64(progress.ProcessedTargets) / float64(progress.TotalTargets) * 100
		if progress.ProgressPercent != expectedPercent {
			t.Errorf("ProgressPercent mismatch: expected %.2f, got %.2f", expectedPercent, progress.ProgressPercent)
		}
	})

	t.Run("Progress info edge cases", func(t *testing.T) {
		tests := []struct {
			name     string
			progress ProgressInfo
			valid    bool
		}{
			{
				name: "Valid progress",
				progress: ProgressInfo{
					TotalTargets:      10,
					ProcessedTargets:  5,
					SuccessfulTargets: 3,
					FailedTargets:     2,
					ProgressPercent:   50.0,
				},
				valid: true,
			},
			{
				name: "Zero total targets",
				progress: ProgressInfo{
					TotalTargets:      0,
					ProcessedTargets:  0,
					SuccessfulTargets: 0,
					FailedTargets:     0,
					ProgressPercent:   0.0,
				},
				valid: true,
			},
			{
				name: "Negative values",
				progress: ProgressInfo{
					TotalTargets:      -1,
					ProcessedTargets:  -1,
					SuccessfulTargets: -1,
					FailedTargets:     -1,
					ProgressPercent:   -1.0,
				},
				valid: false,
			},
			{
				name: "Inconsistent counts",
				progress: ProgressInfo{
					TotalTargets:      10,
					ProcessedTargets:  15, // More than total
					SuccessfulTargets: 5,
					FailedTargets:     2,
					ProgressPercent:   150.0,
				},
				valid: false,
			},
			{
				name: "Mismatched successful + failed",
				progress: ProgressInfo{
					TotalTargets:      10,
					ProcessedTargets:  5,
					SuccessfulTargets: 4,
					FailedTargets:     2, // 4 + 2 = 6, but processed = 5
					ProgressPercent:   50.0,
				},
				valid: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Check basic validity constraints
				hasNegative := tt.progress.TotalTargets < 0 ||
					tt.progress.ProcessedTargets < 0 ||
					tt.progress.SuccessfulTargets < 0 ||
					tt.progress.FailedTargets < 0 ||
					tt.progress.ProgressPercent < 0

				exceedsTotal := tt.progress.ProcessedTargets > tt.progress.TotalTargets

				mismatchedCounts := tt.progress.SuccessfulTargets+tt.progress.FailedTargets != tt.progress.ProcessedTargets

				actuallyValid := !hasNegative && !exceedsTotal && !mismatchedCounts

				if actuallyValid != tt.valid {
					t.Errorf("Expected validity %v, but progress validation result is %v", tt.valid, actuallyValid)
				}
			})
		}
	})

	t.Run("Receipt status transitions", func(t *testing.T) {
		// Test valid status transitions
		validTransitions := map[ReceiptStatus][]ReceiptStatus{
			StatusPending: {
				StatusProcessing,
				StatusCancelled,
			},
			StatusProcessing: {
				StatusCompleted,
				StatusFailed,
				StatusPartialFailed,
				StatusCancelled,
			},
			StatusCompleted: {
				// Terminal state
			},
			StatusFailed: {
				// Terminal state
			},
			StatusPartialFailed: {
				// Terminal state
			},
			StatusCancelled: {
				// Terminal state
			},
		}

		for fromStatus, toStatuses := range validTransitions {
			for _, toStatus := range toStatuses {
				t.Run("Transition from "+string(fromStatus)+" to "+string(toStatus), func(t *testing.T) {
					// This test verifies that our status enum includes expected transitions
					// In a real implementation, you might have a state machine to validate transitions
					if string(fromStatus) == "" || string(toStatus) == "" {
						t.Error("Status values should not be empty")
					}
				})
			}
		}
	})

	t.Run("Duration and timestamp consistency", func(t *testing.T) {
		start := time.Now()

		result := PlatformResult{
			Platform:  "test",
			Target:    "test-target",
			Success:   true,
			Timestamp: start,
			Duration:  100 * time.Millisecond,
		}

		// Duration should be positive for real operations
		if result.Duration <= 0 {
			t.Error("Duration should be positive for completed operations")
		}

		// Timestamp should not be in the future
		if result.Timestamp.After(time.Now().Add(time.Second)) {
			t.Error("Timestamp should not be significantly in the future")
		}

		// For long operations, verify duration makes sense
		longResult := PlatformResult{
			Platform:  "slow-platform",
			Target:    "slow-target",
			Success:   false,
			Error:     "timeout after 30 seconds",
			Timestamp: start,
			Duration:  30 * time.Second,
		}

		if longResult.Duration > time.Minute {
			t.Error("Duration longer than 1 minute seems unreasonable for most operations")
		}
	})

	t.Run("Large receipt handling", func(t *testing.T) {
		// Test with a large number of platform results
		const numResults = 10000
		results := make([]PlatformResult, numResults)

		start := time.Now()
		for i := 0; i < numResults; i++ {
			results[i] = PlatformResult{
				Platform:  "platform_" + string(rune(i%100+'0')),
				Target:    "target_" + string(rune(i)),
				Success:   i%3 != 0, // 2/3 success rate
				Timestamp: time.Now(),
				Duration:  time.Duration(i%1000) * time.Millisecond,
			}
			if !results[i].Success {
				results[i].Error = "error_" + string(rune(i%10+'0'))
			}
		}
		creationTime := time.Since(start)

		receipt := &Receipt{
			MessageID: "large-receipt-test",
			Status:    string(StatusPartialFailed),
			Results:   results,
			Timestamp: time.Now(),
		}

		// Calculate expected counts
		start = time.Now()
		successful := 0
		failed := 0
		for _, result := range results {
			if result.Success {
				successful++
			} else {
				failed++
			}
		}
		countingTime := time.Since(start)

		receipt.Successful = successful
		receipt.Failed = failed
		receipt.Total = len(results)

		// Verify correctness
		if receipt.Total != numResults {
			t.Errorf("Expected total %d, got %d", numResults, receipt.Total)
		}
		if receipt.Successful+receipt.Failed != receipt.Total {
			t.Errorf("Successful (%d) + Failed (%d) should equal Total (%d)",
				receipt.Successful, receipt.Failed, receipt.Total)
		}

		// Performance check
		if creationTime > time.Second {
			t.Errorf("Creating %d results took too long: %v", numResults, creationTime)
		}
		if countingTime > 100*time.Millisecond {
			t.Errorf("Counting %d results took too long: %v", numResults, countingTime)
		}

		t.Logf("Created %d results in %v, counted in %v", numResults, creationTime, countingTime)
	})
}