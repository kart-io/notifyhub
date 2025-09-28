package receipt

import (
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// TestReceiptBuilders tests receipt builder functionality
func TestReceiptBuilders(t *testing.T) {
	t.Run("Receipt builder basic functionality", func(t *testing.T) {
		builder := NewReceiptBuilder("test-message-1")

		// Test initial state
		if builder.messageID != "test-message-1" {
			t.Errorf("Expected message ID 'test-message-1', got '%s'", builder.messageID)
		}
		if len(builder.results) != 0 {
			t.Errorf("Expected empty results, got %d", len(builder.results))
		}
		if builder.timestamp.IsZero() {
			t.Error("Timestamp should be set")
		}

		// Add single result
		builder.AddSingleResult("feishu", "user1", true, "feishu-msg-1", "", 100*time.Millisecond)

		if len(builder.results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(builder.results))
		}

		result := builder.results[0]
		if result.Platform != "feishu" {
			t.Errorf("Expected platform 'feishu', got '%s'", result.Platform)
		}
		if result.Target != "user1" {
			t.Errorf("Expected target 'user1', got '%s'", result.Target)
		}
		if !result.Success {
			t.Error("Expected success to be true")
		}
		if result.MessageID != "feishu-msg-1" {
			t.Errorf("Expected message ID 'feishu-msg-1', got '%s'", result.MessageID)
		}
		if result.Duration != 100*time.Millisecond {
			t.Errorf("Expected duration 100ms, got %v", result.Duration)
		}

		// Build final receipt
		receipt := builder.Build()
		if receipt.MessageID != "test-message-1" {
			t.Errorf("Expected message ID 'test-message-1', got '%s'", receipt.MessageID)
		}
		if receipt.Successful != 1 {
			t.Errorf("Expected 1 successful, got %d", receipt.Successful)
		}
		if receipt.Failed != 0 {
			t.Errorf("Expected 0 failed, got %d", receipt.Failed)
		}
		if receipt.Total != 1 {
			t.Errorf("Expected total 1, got %d", receipt.Total)
		}
		if receipt.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", receipt.Status)
		}
	})

	t.Run("Receipt builder with platform results", func(t *testing.T) {
		builder := NewReceiptBuilder("test-message-2")

		// Mock platform send results
		sendResults := []*platform.SendResult{
			{
				Target: target.Target{
					Type:     "user",
					Value:    "user1",
					Platform: "feishu",
				},
				Success:   true,
				MessageID: "feishu-msg-1",
				Metadata: map[string]interface{}{
					"duration": int64(150), // milliseconds
				},
			},
			{
				Target: target.Target{
					Type:     "user",
					Value:    "user2",
					Platform: "feishu",
				},
				Success:   false,
				Error:     "rate limit exceeded",
				MessageID: "",
				Metadata: map[string]interface{}{
					"duration": int64(500), // milliseconds
				},
			},
		}

		builder.AddPlatformResults("feishu", sendResults)

		if len(builder.results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(builder.results))
		}

		// Check first result
		result1 := builder.results[0]
		if result1.Platform != "feishu" {
			t.Errorf("Expected platform 'feishu', got '%s'", result1.Platform)
		}
		if result1.Target != "user1" {
			t.Errorf("Expected target 'user1', got '%s'", result1.Target)
		}
		if !result1.Success {
			t.Error("Expected first result to be successful")
		}
		if result1.Duration != 150*time.Millisecond {
			t.Errorf("Expected duration 150ms, got %v", result1.Duration)
		}

		// Check second result
		result2 := builder.results[1]
		if result2.Success {
			t.Error("Expected second result to be failed")
		}
		if result2.Error != "rate limit exceeded" {
			t.Errorf("Expected error 'rate limit exceeded', got '%s'", result2.Error)
		}

		// Build and verify receipt
		receipt := builder.Build()
		if receipt.Successful != 1 {
			t.Errorf("Expected 1 successful, got %d", receipt.Successful)
		}
		if receipt.Failed != 1 {
			t.Errorf("Expected 1 failed, got %d", receipt.Failed)
		}
		if receipt.Status != "partial" {
			t.Errorf("Expected status 'partial', got '%s'", receipt.Status)
		}
	})

	t.Run("Receipt builder status calculation", func(t *testing.T) {
		tests := []struct {
			name            string
			successResults  int
			failedResults   int
			expectedStatus  string
		}{
			{
				name:           "All successful",
				successResults: 3,
				failedResults:  0,
				expectedStatus: "success",
			},
			{
				name:           "All failed",
				successResults: 0,
				failedResults:  3,
				expectedStatus: "failed",
			},
			{
				name:           "Mixed results",
				successResults: 2,
				failedResults:  1,
				expectedStatus: "partial",
			},
			{
				name:           "No results",
				successResults: 0,
				failedResults:  0,
				expectedStatus: "success",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				builder := NewReceiptBuilder("status-test-" + tt.name)

				// Add successful results
				for i := 0; i < tt.successResults; i++ {
					builder.AddSingleResult("platform1", "target"+string(rune(i+'0')), true, "msg"+string(rune(i+'0')), "", 100*time.Millisecond)
				}

				// Add failed results
				for i := 0; i < tt.failedResults; i++ {
					builder.AddSingleResult("platform2", "target"+string(rune(i+'0')), false, "", "error"+string(rune(i+'0')), 200*time.Millisecond)
				}

				receipt := builder.Build()
				if receipt.Status != tt.expectedStatus {
					t.Errorf("Expected status '%s', got '%s'", tt.expectedStatus, receipt.Status)
				}
				if receipt.Successful != tt.successResults {
					t.Errorf("Expected %d successful, got %d", tt.successResults, receipt.Successful)
				}
				if receipt.Failed != tt.failedResults {
					t.Errorf("Expected %d failed, got %d", tt.failedResults, receipt.Failed)
				}
			})
		}
	})

	t.Run("Async receipt builder", func(t *testing.T) {
		builder := NewAsyncReceiptBuilder("async-test-1")

		if builder.messageID != "async-test-1" {
			t.Errorf("Expected message ID 'async-test-1', got '%s'", builder.messageID)
		}
		if builder.queuedAt.IsZero() {
			t.Error("QueuedAt should be set")
		}

		// Test different status builders
		tests := []struct {
			name           string
			buildFunc      func() *AsyncReceipt
			expectedStatus string
		}{
			{
				name:           "Pending",
				buildFunc:      builder.BuildPending,
				expectedStatus: string(StatusPending),
			},
			{
				name:           "Processing",
				buildFunc:      builder.BuildProcessing,
				expectedStatus: string(StatusProcessing),
			},
			{
				name:           "Completed",
				buildFunc:      builder.BuildCompleted,
				expectedStatus: string(StatusCompleted),
			},
			{
				name:           "Failed",
				buildFunc:      builder.BuildFailed,
				expectedStatus: string(StatusFailed),
			},
			{
				name:           "Cancelled",
				buildFunc:      builder.BuildCancelled,
				expectedStatus: string(StatusCancelled),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				asyncReceipt := tt.buildFunc()
				if asyncReceipt.MessageID != "async-test-1" {
					t.Errorf("Expected message ID 'async-test-1', got '%s'", asyncReceipt.MessageID)
				}
				if asyncReceipt.Status != tt.expectedStatus {
					t.Errorf("Expected status '%s', got '%s'", tt.expectedStatus, asyncReceipt.Status)
				}
				if asyncReceipt.QueuedAt != builder.queuedAt {
					t.Error("QueuedAt should match builder timestamp")
				}
			})
		}
	})

	t.Run("Progress to receipt conversion", func(t *testing.T) {
		tests := []struct {
			name            string
			messageID       string
			processed       int
			failed          int
			successful      int
			total           int
			expectedStatus  string
			expectError     bool
		}{
			{
				name:           "Complete success",
				messageID:      "progress-test-1",
				processed:      10,
				failed:         0,
				successful:     10,
				total:          10,
				expectedStatus: "success",
				expectError:    false,
			},
			{
				name:           "Complete failure",
				messageID:      "progress-test-2",
				processed:      5,
				failed:         5,
				successful:     0,
				total:          5,
				expectedStatus: "failed",
				expectError:    false,
			},
			{
				name:           "Partial completion",
				messageID:      "progress-test-3",
				processed:      7,
				failed:         2,
				successful:     5,
				total:          10,
				expectedStatus: "processing",
				expectError:    false,
			},
			{
				name:           "Mixed results complete",
				messageID:      "progress-test-4",
				processed:      8,
				failed:         3,
				successful:     5,
				total:          8,
				expectedStatus: "partial",
				expectError:    false,
			},
			{
				name:        "Empty message ID",
				messageID:   "",
				processed:   5,
				failed:      2,
				successful:  3,
				total:       5,
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				startTime := time.Now().Add(-1 * time.Minute)
				receipt, err := ConvertProgressToReceipt(tt.messageID, tt.processed, tt.failed, tt.successful, tt.total, startTime)

				if tt.expectError {
					if err == nil {
						t.Error("Expected error but got none")
					}
					return
				}

				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				if receipt.MessageID != tt.messageID {
					t.Errorf("Expected message ID '%s', got '%s'", tt.messageID, receipt.MessageID)
				}
				if receipt.Status != tt.expectedStatus {
					t.Errorf("Expected status '%s', got '%s'", tt.expectedStatus, receipt.Status)
				}
				if receipt.Successful != tt.successful {
					t.Errorf("Expected %d successful, got %d", tt.successful, receipt.Successful)
				}
				if receipt.Failed != tt.failed {
					t.Errorf("Expected %d failed, got %d", tt.failed, receipt.Failed)
				}
				if receipt.Total != tt.total {
					t.Errorf("Expected %d total, got %d", tt.total, receipt.Total)
				}

				// Check that duration was calculated
				if len(receipt.Results) > 0 {
					duration := receipt.Results[0].Duration
					if duration <= 0 {
						t.Error("Expected positive duration for async batch result")
					}
				}
			})
		}
	})

	t.Run("Factory processor creation", func(t *testing.T) {
		logger := &MockLogger{}
		factory := NewFactory(logger)

		// Test basic creation
		processor := factory.CreateProcessor(nil)
		if processor == nil {
			t.Fatal("Expected processor to be created")
		}

		// Test with configuration
		config := &ProcessorConfig{
			RetentionPeriod: 48 * time.Hour,
			CleanupInterval: 30 * time.Minute,
		}

		processor = factory.CreateProcessor(config)
		if processor.GetRetentionPeriod() != 48*time.Hour {
			t.Errorf("Expected retention period 48h, got %v", processor.GetRetentionPeriod())
		}
		if processor.cleanupInterval != 30*time.Minute {
			t.Errorf("Expected cleanup interval 30m, got %v", processor.cleanupInterval)
		}
	})

	t.Run("Default processor config", func(t *testing.T) {
		config := DefaultProcessorConfig()

		if config.RetentionPeriod != 24*time.Hour {
			t.Errorf("Expected default retention period 24h, got %v", config.RetentionPeriod)
		}
		if config.CleanupInterval != time.Hour {
			t.Errorf("Expected default cleanup interval 1h, got %v", config.CleanupInterval)
		}
	})

	t.Run("Logging subscriber", func(t *testing.T) {
		logger := &MockLogger{}
		subscriber := NewLoggingSubscriber(logger)

		// Test receipt update logging
		receipt := &Receipt{
			MessageID:  "log-test-1",
			Status:     string(StatusCompleted),
			Successful: 2,
			Failed:     1,
			Total:      3,
		}

		// This should not panic
		subscriber.OnReceiptUpdate(receipt)

		// Test async receipt update logging
		asyncReceipt := &AsyncReceiptTracker{
			Receipt: &AsyncReceipt{
				MessageID: "async-log-test-1",
				Status:    string(StatusProcessing),
				QueuedAt:  time.Now(),
			},
			HandleID: "handle-123",
			Progress: &ProgressInfo{
				TotalTargets:      10,
				ProcessedTargets:  6,
				SuccessfulTargets: 4,
				FailedTargets:     2,
				ProgressPercent:   60.0,
			},
			UpdatedAt: time.Now(),
		}

		// This should not panic
		subscriber.OnAsyncReceiptUpdate(asyncReceipt)

		// Test with nil progress
		asyncReceiptNoProgress := &AsyncReceiptTracker{
			Receipt: &AsyncReceipt{
				MessageID: "async-log-test-2",
				Status:    string(StatusPending),
				QueuedAt:  time.Now(),
			},
			HandleID:  "handle-124",
			Progress:  nil,
			UpdatedAt: time.Now(),
		}

		// This should not panic
		subscriber.OnAsyncReceiptUpdate(asyncReceiptNoProgress)
	})

	t.Run("Builder method chaining", func(t *testing.T) {
		// Test that builder methods return the builder for chaining
		builder := NewReceiptBuilder("chain-test")

		result := builder.
			AddSingleResult("platform1", "target1", true, "msg1", "", 100*time.Millisecond).
			AddSingleResult("platform2", "target2", false, "", "error", 200*time.Millisecond)

		if result != builder {
			t.Error("Builder methods should return the builder for chaining")
		}

		receipt := builder.Build()
		if len(receipt.Results) != 2 {
			t.Errorf("Expected 2 results from chained calls, got %d", len(receipt.Results))
		}
	})

	t.Run("Large batch builder performance", func(t *testing.T) {
		builder := NewReceiptBuilder("performance-test")

		// Add many results
		const numResults = 10000
		start := time.Now()

		for i := 0; i < numResults; i++ {
			builder.AddSingleResult(
				"platform_"+string(rune(i%10+'0')),
				"target_"+string(rune(i)),
				i%3 != 0, // 2/3 success rate
				"msg_"+string(rune(i)),
				"",
				time.Duration(i%1000)*time.Millisecond,
			)
		}

		buildTime := time.Since(start)

		start = time.Now()
		receipt := builder.Build()
		finalizeTime := time.Since(start)

		// Verify correctness
		if len(receipt.Results) != numResults {
			t.Errorf("Expected %d results, got %d", numResults, len(receipt.Results))
		}

		expectedSuccessful := 0
		for i := 0; i < numResults; i++ {
			if i%3 != 0 {
				expectedSuccessful++
			}
		}

		if receipt.Successful != expectedSuccessful {
			t.Errorf("Expected %d successful, got %d", expectedSuccessful, receipt.Successful)
		}

		// Performance checks
		if buildTime > time.Second {
			t.Errorf("Building %d results took too long: %v", numResults, buildTime)
		}
		if finalizeTime > 100*time.Millisecond {
			t.Errorf("Finalizing receipt took too long: %v", finalizeTime)
		}

		t.Logf("Built %d results in %v, finalized in %v", numResults, buildTime, finalizeTime)
	})

	t.Run("Builder error handling edge cases", func(t *testing.T) {
		// Test with empty message ID
		builder := NewReceiptBuilder("")
		receipt := builder.Build()
		if receipt.MessageID != "" {
			t.Error("Builder should preserve empty message ID")
		}

		// Test with empty platform/target
		builder = NewReceiptBuilder("edge-test")
		builder.AddSingleResult("", "", true, "", "", 0)
		receipt = builder.Build()

		if len(receipt.Results) != 1 {
			t.Error("Should still add result even with empty fields")
		}

		result := receipt.Results[0]
		if result.Platform != "" || result.Target != "" {
			t.Error("Should preserve empty platform and target")
		}

		// Test with negative duration
		builder = NewReceiptBuilder("negative-duration-test")
		builder.AddSingleResult("platform", "target", true, "msg", "", -100*time.Millisecond)
		receipt = builder.Build()

		if receipt.Results[0].Duration >= 0 {
			t.Error("Should preserve negative duration for error analysis")
		}
	})
}