package receipt

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// TestReceiptModelIntegration tests complete receipt model integration scenarios
func TestReceiptModelIntegration(t *testing.T) {
	t.Run("Complete workflow: Builder -> Processor -> Aggregation -> Serialization", func(t *testing.T) {
		// Setup
		logger := &MockLogger{}
		store := NewMemoryStore()
		config := AggregationConfig{
			PartialFailureThreshold: 0.6,
			FailureTolerance:        2,
			RequiredPlatforms:       []string{"feishu", "email"},
		}

		processor := NewProcessor(logger,
			WithPersistenceStore(store),
			WithAggregationConfig(config),
		)

		// Step 1: Build receipts using builders
		messageID := "integration-test-1"
		builder := NewReceiptBuilder(messageID)

		// Simulate platform results
		platformResults := []*platform.SendResult{
			{
				Target: target.Target{
					Type:     "user",
					Value:    "user1",
					Platform: "feishu",
				},
				Success:   true,
				MessageID: "feishu-msg-1",
				Metadata: map[string]interface{}{
					"duration": int64(120),
				},
			},
			{
				Target: target.Target{
					Type:     "email",
					Value:    "user1@example.com",
					Platform: "email",
				},
				Success:   false,
				Error:     "SMTP timeout",
				MessageID: "",
				Metadata: map[string]interface{}{
					"duration": int64(5000),
				},
			},
			{
				Target: target.Target{
					Type:     "webhook",
					Value:    "https://webhook.example.com",
					Platform: "webhook",
				},
				Success:   true,
				MessageID: "webhook-msg-1",
				Metadata: map[string]interface{}{
					"duration": int64(200),
				},
			},
		}

		// Add platform results
		builder.AddPlatformResults("feishu", platformResults[:1])
		builder.AddPlatformResults("email", platformResults[1:2])
		builder.AddPlatformResults("webhook", platformResults[2:3])

		// Build receipt
		receipt := builder.Build()

		// Step 2: Process through processor with aggregation
		err := processor.ProcessReceipt(receipt)
		if err != nil {
			t.Fatalf("Failed to process receipt: %v", err)
		}

		// Step 3: Verify aggregation applied
		processed, found := processor.GetReceipt(messageID)
		if !found {
			t.Fatal("Processed receipt not found")
		}

		// Check aggregation results
		if processed.Total != 3 {
			t.Errorf("Expected total 3, got %d", processed.Total)
		}
		if processed.Successful != 2 {
			t.Errorf("Expected successful 2, got %d", processed.Successful)
		}
		if processed.Failed != 1 {
			t.Errorf("Expected failed 1, got %d", processed.Failed)
		}

		// Check status calculation (should be partial failure due to required platform failure)
		if processed.Status != string(StatusFailed) {
			t.Errorf("Expected status failed (email required platform failed), got %s", processed.Status)
		}

		// Step 4: Serialize and verify
		data, err := processor.SerializeReceipt(processed)
		if err != nil {
			t.Fatalf("Failed to serialize receipt: %v", err)
		}

		deserialized, err := processor.DeserializeReceipt(data)
		if err != nil {
			t.Fatalf("Failed to deserialize receipt: %v", err)
		}

		// Verify serialization integrity
		if deserialized.MessageID != processed.MessageID {
			t.Error("Serialization lost message ID")
		}
		if deserialized.Status != processed.Status {
			t.Error("Serialization lost status")
		}
		if len(deserialized.Results) != len(processed.Results) {
			t.Error("Serialization lost platform results")
		}

		// Step 5: Verify persistence
		_, err = store.Get(messageID)
		if err != nil {
			t.Errorf("Receipt not persisted: %v", err)
		}
	})

	t.Run("Async receipt lifecycle management", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		// Step 1: Create pending async receipt
		messageID := "async-lifecycle-test"
		asyncBuilder := NewAsyncReceiptBuilder(messageID)
		pendingReceipt := asyncBuilder.BuildPending()

		err := processor.ProcessAsyncReceipt(pendingReceipt, "handle-123")
		if err != nil {
			t.Fatalf("Failed to process pending receipt: %v", err)
		}

		// Step 2: Update to processing
		processingReceipt := asyncBuilder.BuildProcessing()
		err = processor.ProcessAsyncReceipt(processingReceipt, "handle-123")
		if err != nil {
			t.Fatalf("Failed to process processing receipt: %v", err)
		}

		// Step 3: Add progress tracking
		tracker, found := processor.GetAsyncReceipt(messageID)
		if !found {
			t.Fatal("Async receipt tracker not found")
		}

		tracker.Progress = &ProgressInfo{
			TotalTargets:      10,
			ProcessedTargets:  5,
			SuccessfulTargets: 4,
			FailedTargets:     1,
			ProgressPercent:   50.0,
		}
		tracker.UpdatedAt = time.Now()

		// Manually update in processor
		processor.mutex.Lock()
		processor.asyncReceipts[messageID] = tracker
		processor.mutex.Unlock()

		// Step 4: Complete the async operation
		completedReceipt := asyncBuilder.BuildCompleted()
		err = processor.ProcessAsyncReceipt(completedReceipt, "handle-123")
		if err != nil {
			t.Fatalf("Failed to process completed receipt: %v", err)
		}

		// Step 5: Convert to final receipt
		finalReceipt, err := ConvertProgressToReceipt(
			messageID,
			tracker.Progress.ProcessedTargets,
			tracker.Progress.FailedTargets,
			tracker.Progress.SuccessfulTargets,
			tracker.Progress.TotalTargets,
			tracker.Receipt.QueuedAt,
		)
		if err != nil {
			t.Fatalf("Failed to convert progress to receipt: %v", err)
		}

		// Process final receipt
		err = processor.ProcessReceipt(finalReceipt)
		if err != nil {
			t.Fatalf("Failed to process final receipt: %v", err)
		}

		// Verify final state
		final, found := processor.GetReceipt(messageID)
		if !found {
			t.Fatal("Final receipt not found")
		}

		if final.Successful != 4 {
			t.Errorf("Expected 4 successful in final receipt, got %d", final.Successful)
		}
		if final.Failed != 1 {
			t.Errorf("Expected 1 failed in final receipt, got %d", final.Failed)
		}
	})

	t.Run("Multi-platform failure pattern analysis", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		// Create receipts with various failure patterns
		patterns := []struct {
			platforms []string
			successes []bool
			errors    []string
		}{
			{
				platforms: []string{"feishu", "email", "webhook"},
				successes: []bool{true, true, true},
				errors:    []string{"", "", ""},
			},
			{
				platforms: []string{"feishu", "email"},
				successes: []bool{false, false},
				errors:    []string{"rate limit exceeded", "SMTP timeout"},
			},
			{
				platforms: []string{"feishu", "email", "sms"},
				successes: []bool{true, false, false},
				errors:    []string{"", "authentication failed", "invalid number"},
			},
			{
				platforms: []string{"webhook"},
				successes: []bool{false},
				errors:    []string{"service unavailable"},
			},
			{
				platforms: []string{"feishu", "webhook"},
				successes: []bool{false, true},
				errors:    []string{"network timeout", ""},
			},
		}

		// Process all patterns
		for i, pattern := range patterns {
			results := make([]PlatformResult, len(pattern.platforms))
			for j, platform := range pattern.platforms {
				results[j] = PlatformResult{
					Platform:  platform,
					Target:    "target" + string(rune(j)),
					Success:   pattern.successes[j],
					Error:     pattern.errors[j],
					Timestamp: time.Now(),
				}
			}

			receipt := &Receipt{
				MessageID: "pattern-" + string(rune(i)),
				Results:   results,
				Timestamp: time.Now(),
			}

			// Calculate counts
			successful := 0
			failed := 0
			for _, success := range pattern.successes {
				if success {
					successful++
				} else {
					failed++
				}
			}

			receipt.Successful = successful
			receipt.Failed = failed
			receipt.Total = len(results)

			if failed > 0 && successful == 0 {
				receipt.Status = string(StatusFailed)
			} else if failed > 0 {
				receipt.Status = string(StatusPartialFailed)
			} else {
				receipt.Status = string(StatusCompleted)
			}

			err := processor.ProcessReceipt(receipt)
			if err != nil {
				t.Fatalf("Failed to process pattern receipt %d: %v", i, err)
			}
		}

		// Analyze failure patterns
		analysis := processor.AnalyzeFailurePatterns(ReceiptFilter{})

		// Verify analysis results
		if analysis["total_receipts"].(int) != len(patterns) {
			t.Errorf("Expected %d total receipts, got %v", len(patterns), analysis["total_receipts"])
		}

		// Check specific error categories
		errorsByType := analysis["failures_by_error"].(map[string]int)
		expectedErrors := map[string]int{
			"rate_limit_errors":          1,
			"timeout_errors":             2, // SMTP timeout + network timeout
			"authentication_errors":      1,
			"validation_errors":          1, // invalid number
			"service_unavailable_errors": 1,
		}

		for errorType, expectedCount := range expectedErrors {
			if errorsByType[errorType] != expectedCount {
				t.Errorf("Expected %d %s, got %d", expectedCount, errorType, errorsByType[errorType])
			}
		}

		// Check platform failures
		platformFailures := analysis["failures_by_platform"].(map[string]int)
		expectedPlatformFailures := map[string]int{
			"feishu":  2,
			"email":   2,
			"webhook": 1,
			"sms":     1,
		}

		for platform, expectedCount := range expectedPlatformFailures {
			if platformFailures[platform] != expectedCount {
				t.Errorf("Expected %d failures for %s, got %d", expectedCount, platform, platformFailures[platform])
			}
		}
	})

	t.Run("High-throughput concurrent processing", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		processor.Start(ctx)
		defer processor.Stop()

		const numWorkers = 20
		const receiptsPerWorker = 200
		const totalReceipts = numWorkers * receiptsPerWorker

		var wg sync.WaitGroup
		errors := make(chan error, numWorkers)
		startTime := time.Now()

		// Launch concurrent workers
		for workerID := 0; workerID < numWorkers; workerID++ {
			wg.Add(1)
			go func(wID int) {
				defer wg.Done()

				for i := 0; i < receiptsPerWorker; i++ {
					messageID := "concurrent-" + string(rune(wID)) + "-" + string(rune(i))

					// Vary the complexity of receipts
					numPlatforms := (i % 5) + 1
					results := make([]PlatformResult, numPlatforms)

					successful := 0
					failed := 0

					for p := 0; p < numPlatforms; p++ {
						success := (p+i)%3 != 0 // Roughly 2/3 success rate
						results[p] = PlatformResult{
							Platform:  "platform" + string(rune(p+'0')),
							Target:    "target" + string(rune(p)),
							Success:   success,
							Timestamp: time.Now(),
							Duration:  time.Duration((p+1)*50) * time.Millisecond,
						}

						if success {
							successful++
							results[p].MessageID = "msg-" + string(rune(p))
						} else {
							failed++
							results[p].Error = "error-" + string(rune(p%3+'0'))
						}
					}

					receipt := &Receipt{
						MessageID:  messageID,
						Results:    results,
						Successful: successful,
						Failed:     failed,
						Total:      numPlatforms,
						Timestamp:  time.Now(),
					}

					if err := processor.ProcessReceipt(receipt); err != nil {
						select {
						case errors <- err:
						default:
						}
						return
					}

					// Occasionally process async receipts too
					if i%10 == 0 {
						asyncReceipt := &AsyncReceipt{
							MessageID: "async-" + messageID,
							Status:    string(StatusProcessing),
							QueuedAt:  time.Now(),
						}

						if err := processor.ProcessAsyncReceipt(asyncReceipt, "handle-"+string(rune(wID))); err != nil {
							select {
							case errors <- err:
							default:
							}
							return
						}
					}
				}
			}(workerID)
		}

		// Wait for completion
		wg.Wait()
		close(errors)

		processingDuration := time.Since(startTime)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent processing error: %v", err)
		}

		// Verify final state
		stats := processor.GetStats()
		processedCount := stats["processed_count"].(int64)

		if processedCount != int64(totalReceipts) {
			t.Errorf("Expected %d processed receipts, got %d", totalReceipts, processedCount)
		}

		asyncCount := stats["async_receipts_count"].(int)
		expectedAsyncCount := numWorkers * (receiptsPerWorker / 10) // Every 10th receipt

		if asyncCount != expectedAsyncCount {
			t.Errorf("Expected %d async receipts, got %d", expectedAsyncCount, asyncCount)
		}

		// Performance verification
		avgPerReceipt := processingDuration / time.Duration(totalReceipts)
		if avgPerReceipt > 5*time.Millisecond {
			t.Errorf("Average processing time per receipt too slow: %v", avgPerReceipt)
		}

		t.Logf("Processed %d receipts in %v (avg: %v per receipt)",
			totalReceipts, processingDuration, avgPerReceipt)
		t.Logf("Throughput: %.2f receipts/second",
			float64(totalReceipts)/processingDuration.Seconds())
	})

	t.Run("Export/Import with large dataset", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		// Generate large dataset
		const datasetSize = 5000
		startTime := time.Now()

		for i := 0; i < datasetSize; i++ {
			receipt := &Receipt{
				MessageID: "export-import-" + string(rune(i)),
				Status:    []string{string(StatusCompleted), string(StatusFailed), string(StatusPartialFailed)}[i%3],
				Results: []PlatformResult{
					{Platform: "feishu", Success: i%2 == 0, Timestamp: time.Now()},
					{Platform: "email", Success: i%3 != 0, Timestamp: time.Now()},
				},
				Successful: func() int {
					count := 0
					if i%2 == 0 {
						count++
					}
					if i%3 != 0 {
						count++
					}
					return count
				}(),
				Failed: func() int {
					count := 0
					if i%2 != 0 {
						count++
					}
					if i%3 == 0 {
						count++
					}
					return count
				}(),
				Total:     2,
				Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			}

			err := processor.ProcessReceipt(receipt)
			if err != nil {
				t.Fatalf("Failed to process receipt %d: %v", i, err)
			}
		}

		generationTime := time.Since(startTime)

		// Export with selective filter
		var buffer bytes.Buffer
		filter := ReceiptFilter{
			Status: []string{string(StatusCompleted), string(StatusPartialFailed)},
			Limit:  1000, // Only export first 1000 matching
		}

		startTime = time.Now()
		err := processor.ExportReceipts(&buffer, filter)
		exportTime := time.Since(startTime)

		if err != nil {
			t.Fatalf("Failed to export receipts: %v", err)
		}

		// Clear processor
		processor.receipts = make(map[string]*Receipt)

		// Import
		startTime = time.Now()
		err = processor.ImportReceipts(&buffer)
		importTime := time.Since(startTime)

		if err != nil {
			t.Fatalf("Failed to import receipts: %v", err)
		}

		// Verify imported data
		imported := processor.GetReceiptsByFilter(ReceiptFilter{})
		if len(imported) != 1000 {
			t.Errorf("Expected 1000 imported receipts, got %d", len(imported))
		}

		// Verify only specified statuses were imported
		for _, receipt := range imported {
			if receipt.Status != string(StatusCompleted) && receipt.Status != string(StatusPartialFailed) {
				t.Errorf("Unexpected status in imported data: %s", receipt.Status)
			}
		}

		// Performance checks
		if generationTime > 10*time.Second {
			t.Errorf("Data generation took too long: %v", generationTime)
		}
		if exportTime > 2*time.Second {
			t.Errorf("Export took too long: %v", exportTime)
		}
		if importTime > 2*time.Second {
			t.Errorf("Import took too long: %v", importTime)
		}

		t.Logf("Generated %d receipts in %v", datasetSize, generationTime)
		t.Logf("Exported 1000 receipts in %v, imported in %v", exportTime, importTime)
		t.Logf("Export size: %d bytes", buffer.Len())
	})

	t.Run("Memory efficiency and cleanup verification", func(t *testing.T) {
		// Set very short retention for aggressive cleanup testing
		processor := NewProcessor(&MockLogger{}, WithRetentionPeriod(50*time.Millisecond))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		processor.Start(ctx)
		defer processor.Stop()

		// Add many receipts over time
		const numBatches = 10
		const receiptsPerBatch = 100

		totalAdded := 0

		for batch := 0; batch < numBatches; batch++ {
			// Add batch of receipts
			for i := 0; i < receiptsPerBatch; i++ {
				receipt := &Receipt{
					MessageID: "cleanup-" + string(rune(batch)) + "-" + string(rune(i)),
					Status:    string(StatusCompleted),
					Results:   []PlatformResult{{Platform: "test", Success: true, Timestamp: time.Now()}},
					Timestamp: time.Now(),
				}

				err := processor.ProcessReceipt(receipt)
				if err != nil {
					t.Fatalf("Failed to process receipt: %v", err)
				}
				totalAdded++
			}

			// Wait for cleanup to occur
			time.Sleep(100 * time.Millisecond)

			// Force cleanup
			processor.performCleanup()

			// Check memory usage
			stats := processor.GetStats()
			currentCount := stats["sync_receipts_count"].(int)

			// After cleanup, should have significantly fewer than total added
			if currentCount > receiptsPerBatch*2 {
				t.Logf("Batch %d: Current count %d, total added %d", batch, currentCount, totalAdded)
			}
		}

		// Final verification
		stats := processor.GetStats()
		finalCount := stats["sync_receipts_count"].(int)

		// Should have cleaned up most receipts
		if finalCount > receiptsPerBatch {
			t.Errorf("Expected cleanup to reduce memory usage, final count: %d", finalCount)
		}

		t.Logf("Added %d receipts total, %d remaining after cleanup", totalAdded, finalCount)
	})
}