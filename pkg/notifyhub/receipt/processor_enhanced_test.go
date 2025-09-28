package receipt

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"
)

// TestProcessorEnhancedFunctionality tests advanced processor features
func TestProcessorEnhancedFunctionality(t *testing.T) {
	t.Run("Concurrent processing safety", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		const numGoroutines = 50
		const receiptsPerGoroutine = 100

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		// Start processor
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		processor.Start(ctx)
		defer processor.Stop()

		// Concurrent processing
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < receiptsPerGoroutine; j++ {
					receipt := &Receipt{
						MessageID: "concurrent-test-" + string(rune(workerID)) + "-" + string(rune(j)),
						Status:    string(StatusCompleted),
						Results: []PlatformResult{
							{Platform: "feishu", Target: "user" + string(rune(j)), Success: j%2 == 0, Timestamp: time.Now()},
						},
						Successful: 1,
						Failed:     0,
						Total:      1,
						Timestamp:  time.Now(),
					}

					if err := processor.ProcessReceipt(receipt); err != nil {
						select {
						case errors <- err:
						default:
						}
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent processing error: %v", err)
		}

		// Verify final count
		stats := processor.GetStats()
		expectedCount := int64(numGoroutines * receiptsPerGoroutine)
		if stats["processed_count"].(int64) != expectedCount {
			t.Errorf("Expected %d processed receipts, got %d", expectedCount, stats["processed_count"])
		}
	})

	t.Run("Memory cleanup and retention", func(t *testing.T) {
		// Set short retention period for testing
		processor := NewProcessor(&MockLogger{}, WithRetentionPeriod(100*time.Millisecond))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		processor.Start(ctx)
		defer processor.Stop()

		// Add old receipt
		oldReceipt := &Receipt{
			MessageID: "old-receipt",
			Status:    string(StatusCompleted),
			Results:   []PlatformResult{{Platform: "feishu", Success: true, Timestamp: time.Now()}},
			Timestamp: time.Now().Add(-200 * time.Millisecond), // Older than retention
		}

		err := processor.ProcessReceipt(oldReceipt)
		if err != nil {
			t.Fatalf("Failed to process old receipt: %v", err)
		}

		// Add new receipt
		newReceipt := &Receipt{
			MessageID: "new-receipt",
			Status:    string(StatusCompleted),
			Results:   []PlatformResult{{Platform: "feishu", Success: true, Timestamp: time.Now()}},
			Timestamp: time.Now(),
		}

		err = processor.ProcessReceipt(newReceipt)
		if err != nil {
			t.Fatalf("Failed to process new receipt: %v", err)
		}

		// Trigger cleanup manually
		processor.performCleanup()

		// Old receipt should be cleaned up
		_, found := processor.GetReceipt("old-receipt")
		if found {
			t.Error("Old receipt should have been cleaned up")
		}

		// New receipt should still exist
		_, found = processor.GetReceipt("new-receipt")
		if !found {
			t.Error("New receipt should still exist after cleanup")
		}
	})

	t.Run("Large batch processing performance", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		// Create large batch
		const batchSize = 5000
		receipts := make([]*Receipt, batchSize)

		start := time.Now()
		for i := 0; i < batchSize; i++ {
			receipts[i] = &Receipt{
				MessageID: "batch-" + string(rune(i)),
				Status:    string(StatusCompleted),
				Results: []PlatformResult{
					{Platform: "feishu", Target: "user" + string(rune(i)), Success: i%4 != 0, Timestamp: time.Now()},
					{Platform: "email", Target: "user" + string(rune(i)) + "@example.com", Success: i%3 != 0, Timestamp: time.Now()},
				},
				Timestamp: time.Now(),
			}
		}
		preparationTime := time.Since(start)

		// Process batch
		start = time.Now()
		err := processor.BatchProcessReceipts(receipts)
		processingTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to batch process receipts: %v", err)
		}

		// Verify all receipts were processed
		stats := processor.GetStats()
		if stats["sync_receipts_count"].(int) != batchSize {
			t.Errorf("Expected %d receipts, got %d", batchSize, stats["sync_receipts_count"])
		}

		// Performance checks
		if preparationTime > time.Second {
			t.Errorf("Preparation took too long: %v", preparationTime)
		}
		if processingTime > 2*time.Second {
			t.Errorf("Batch processing took too long: %v", processingTime)
		}

		t.Logf("Prepared %d receipts in %v, processed in %v", batchSize, preparationTime, processingTime)
	})

	t.Run("Subscription and notification system", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		// Create test subscriber
		received := make(chan *Receipt, 10)
		asyncReceived := make(chan *AsyncReceiptTracker, 10)

		subscriber := &TestSubscriber{
			OnReceiptUpdateFunc: func(receipt *Receipt) {
				received <- receipt
			},
			OnAsyncReceiptUpdateFunc: func(tracker *AsyncReceiptTracker) {
				asyncReceived <- tracker
			},
		}

		// Subscribe to a specific message
		messageID := "subscription-test"
		processor.Subscribe(messageID, subscriber)

		// Process receipt
		receipt := &Receipt{
			MessageID: messageID,
			Status:    string(StatusCompleted),
			Results:   []PlatformResult{{Platform: "feishu", Success: true, Timestamp: time.Now()}},
			Timestamp: time.Now(),
		}

		err := processor.ProcessReceipt(receipt)
		if err != nil {
			t.Fatalf("Failed to process receipt: %v", err)
		}

		// Check notification
		select {
		case receivedReceipt := <-received:
			if receivedReceipt.MessageID != messageID {
				t.Errorf("Expected message ID %s, got %s", messageID, receivedReceipt.MessageID)
			}
		case <-time.After(time.Second):
			t.Error("Did not receive receipt notification")
		}

		// Process async receipt
		asyncReceipt := &AsyncReceipt{
			MessageID: messageID,
			Status:    string(StatusProcessing),
			QueuedAt:  time.Now(),
		}

		err = processor.ProcessAsyncReceipt(asyncReceipt, "handle-123")
		if err != nil {
			t.Fatalf("Failed to process async receipt: %v", err)
		}

		// Check async notification
		select {
		case receivedTracker := <-asyncReceived:
			if receivedTracker.Receipt.MessageID != messageID {
				t.Errorf("Expected message ID %s, got %s", messageID, receivedTracker.Receipt.MessageID)
			}
		case <-time.After(time.Second):
			t.Error("Did not receive async receipt notification")
		}

		// Unsubscribe
		processor.Unsubscribe(messageID, subscriber)

		// Process another receipt - should not receive notification
		receipt2 := &Receipt{
			MessageID: messageID,
			Status:    string(StatusFailed),
			Results:   []PlatformResult{{Platform: "email", Success: false, Error: "timeout", Timestamp: time.Now()}},
			Timestamp: time.Now(),
		}

		err = processor.ProcessReceipt(receipt2)
		if err != nil {
			t.Fatalf("Failed to process receipt after unsubscribe: %v", err)
		}

		// Should not receive notification
		select {
		case <-received:
			t.Error("Should not receive notification after unsubscribe")
		case <-time.After(100 * time.Millisecond):
			// Expected - no notification
		}
	})

	t.Run("Filter-based receipt querying", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		// Add various receipts
		now := time.Now()
		receipts := []*Receipt{
			{
				MessageID: "filter-1",
				Status:    string(StatusCompleted),
				Results:   []PlatformResult{{Platform: "feishu", Success: true}},
				Timestamp: now.Add(-2 * time.Hour),
			},
			{
				MessageID: "filter-2",
				Status:    string(StatusFailed),
				Results:   []PlatformResult{{Platform: "email", Success: false}},
				Timestamp: now.Add(-1 * time.Hour),
			},
			{
				MessageID: "filter-3",
				Status:    string(StatusPartialFailed),
				Results: []PlatformResult{
					{Platform: "feishu", Success: true},
					{Platform: "webhook", Success: false},
				},
				Timestamp: now,
			},
		}

		for _, receipt := range receipts {
			err := processor.ProcessReceipt(receipt)
			if err != nil {
				t.Fatalf("Failed to process receipt %s: %v", receipt.MessageID, err)
			}
		}

		// Test various filters
		tests := []struct {
			name          string
			filter        ReceiptFilter
			expectedCount int
			expectedIDs   []string
		}{
			{
				name:          "No filter",
				filter:        ReceiptFilter{},
				expectedCount: 3,
			},
			{
				name: "Status filter",
				filter: ReceiptFilter{
					Status: []string{string(StatusCompleted), string(StatusPartialFailed)},
				},
				expectedCount: 2,
				expectedIDs:   []string{"filter-1", "filter-3"},
			},
			{
				name: "Platform filter",
				filter: ReceiptFilter{
					Platforms: []string{"feishu"},
				},
				expectedCount: 2,
				expectedIDs:   []string{"filter-1", "filter-3"},
			},
			{
				name: "Time range filter",
				filter: ReceiptFilter{
					StartTime: &[]time.Time{now.Add(-90 * time.Minute)}[0],
				},
				expectedCount: 2,
				expectedIDs:   []string{"filter-2", "filter-3"},
			},
			{
				name: "Message ID filter",
				filter: ReceiptFilter{
					MessageIDs: []string{"filter-1", "filter-3"},
				},
				expectedCount: 2,
				expectedIDs:   []string{"filter-1", "filter-3"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				results := processor.GetReceiptsByFilter(tt.filter)

				if len(results) != tt.expectedCount {
					t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
				}

				if tt.expectedIDs != nil {
					resultIDs := make(map[string]bool)
					for _, result := range results {
						resultIDs[result.MessageID] = true
					}

					for _, expectedID := range tt.expectedIDs {
						if !resultIDs[expectedID] {
							t.Errorf("Expected to find receipt %s in results", expectedID)
						}
					}
				}
			})
		}
	})

	t.Run("Import/Export with complex filters", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		// Add test data
		now := time.Now()
		receipts := []*Receipt{
			{
				MessageID: "export-1",
				Status:    string(StatusCompleted),
				Results:   []PlatformResult{{Platform: "feishu", Success: true}},
				Timestamp: now.Add(-2 * time.Hour),
			},
			{
				MessageID: "export-2",
				Status:    string(StatusFailed),
				Results:   []PlatformResult{{Platform: "email", Success: false, Error: "timeout"}},
				Timestamp: now.Add(-1 * time.Hour),
			},
			{
				MessageID: "export-3",
				Status:    string(StatusCompleted),
				Results:   []PlatformResult{{Platform: "webhook", Success: true}},
				Timestamp: now,
			},
		}

		for _, receipt := range receipts {
			err := processor.ProcessReceipt(receipt)
			if err != nil {
				t.Fatalf("Failed to process receipt: %v", err)
			}
		}

		// Export with filter
		var buffer bytes.Buffer
		filter := ReceiptFilter{
			Status: []string{string(StatusCompleted)},
		}

		err := processor.ExportReceipts(&buffer, filter)
		if err != nil {
			t.Fatalf("Failed to export receipts: %v", err)
		}

		// Clear processor
		processor.receipts = make(map[string]*Receipt)

		// Import
		err = processor.ImportReceipts(&buffer)
		if err != nil {
			t.Fatalf("Failed to import receipts: %v", err)
		}

		// Verify only completed receipts were imported
		imported := processor.GetReceiptsByFilter(ReceiptFilter{})
		if len(imported) != 2 {
			t.Errorf("Expected 2 imported receipts, got %d", len(imported))
		}

		for _, receipt := range imported {
			if receipt.Status != string(StatusCompleted) {
				t.Errorf("Expected only completed receipts, found %s", receipt.Status)
			}
		}
	})

	t.Run("Progress tracking for async receipts", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		messageID := "progress-test"

		// Create async receipt with progress
		asyncReceipt := &AsyncReceipt{
			MessageID: messageID,
			Status:    string(StatusProcessing),
			QueuedAt:  time.Now(),
		}

		progress := &ProgressInfo{
			TotalTargets:        100,
			ProcessedTargets:    25,
			SuccessfulTargets:   20,
			FailedTargets:       5,
			ProgressPercent:     25.0,
			EstimatedCompletion: &[]time.Time{time.Now().Add(2 * time.Minute)}[0],
		}

		tracker := &AsyncReceiptTracker{
			Receipt:   asyncReceipt,
			HandleID:  "handle-progress",
			Progress:  progress,
			UpdatedAt: time.Now(),
		}

		err := processor.ProcessAsyncReceipt(asyncReceipt, "handle-progress")
		if err != nil {
			t.Fatalf("Failed to process async receipt: %v", err)
		}

		// Update progress
		tracker.Progress.ProcessedTargets = 50
		tracker.Progress.SuccessfulTargets = 40
		tracker.Progress.FailedTargets = 10
		tracker.Progress.ProgressPercent = 50.0
		tracker.UpdatedAt = time.Now()

		// Store updated tracker
		processor.mutex.Lock()
		processor.asyncReceipts[messageID] = tracker
		processor.mutex.Unlock()

		// Retrieve and verify
		retrieved, found := processor.GetAsyncReceipt(messageID)
		if !found {
			t.Fatal("Async receipt not found")
		}

		if retrieved.Progress.ProcessedTargets != 50 {
			t.Errorf("Expected 50 processed targets, got %d", retrieved.Progress.ProcessedTargets)
		}
		if retrieved.Progress.ProgressPercent != 50.0 {
			t.Errorf("Expected 50%% progress, got %.2f%%", retrieved.Progress.ProgressPercent)
		}
	})

	t.Run("Error resilience and recovery", func(t *testing.T) {
		// Create failing store
		failingStore := NewFailingStore()
		failingStore.shouldFail = true

		processor := NewProcessor(&MockLogger{}, WithPersistenceStore(failingStore))

		receipt := &Receipt{
			MessageID: "resilience-test",
			Status:    string(StatusCompleted),
			Results:   []PlatformResult{{Platform: "feishu", Success: true, Timestamp: time.Now()}},
			Timestamp: time.Now(),
		}

		// Should not fail even if persistence fails
		err := processor.ProcessReceipt(receipt)
		if err != nil {
			t.Fatalf("Processing should succeed even if persistence fails: %v", err)
		}

		// Should still be in memory
		retrieved, found := processor.GetReceipt("resilience-test")
		if !found {
			t.Error("Receipt should be in memory even if persistence failed")
		}
		if retrieved.MessageID != "resilience-test" {
			t.Error("Retrieved receipt should match original")
		}

		// Enable store and test recovery
		failingStore.shouldFail = false

		err = processor.ProcessReceipt(receipt)
		if err != nil {
			t.Fatalf("Processing should succeed when store is working: %v", err)
		}
	})

	t.Run("Metrics accuracy under load", func(t *testing.T) {
		processor := NewProcessor(&MockLogger{})

		const numReceipts = 1000
		expectedSuccessful := 0
		expectedFailed := 0
		expectedPartial := 0

		start := time.Now()

		for i := 0; i < numReceipts; i++ {
			var status string
			var successful, failed int

			switch i % 3 {
			case 0: // All successful
				status = string(StatusCompleted)
				successful = 2
				failed = 0
				expectedSuccessful++
			case 1: // All failed
				status = string(StatusFailed)
				successful = 0
				failed = 2
				expectedFailed++
			case 2: // Partial
				status = string(StatusPartialFailed)
				successful = 1
				failed = 1
				expectedPartial++
			}

			receipt := &Receipt{
				MessageID:  "metrics-" + string(rune(i)),
				Status:     status,
				Results:    make([]PlatformResult, successful+failed),
				Successful: successful,
				Failed:     failed,
				Total:      successful + failed,
				Timestamp:  time.Now(),
			}

			err := processor.ProcessReceipt(receipt)
			if err != nil {
				t.Fatalf("Failed to process receipt %d: %v", i, err)
			}
		}

		processingDuration := time.Since(start)

		// Check metrics
		metrics := processor.GetMetrics()

		if metrics.ProcessedCount != int64(numReceipts) {
			t.Errorf("Expected %d processed receipts, got %d", numReceipts, metrics.ProcessedCount)
		}

		if metrics.PartialFailures != int64(expectedPartial) {
			t.Errorf("Expected %d partial failures, got %d", expectedPartial, metrics.PartialFailures)
		}

		if metrics.AverageProcessing <= 0 {
			t.Error("Average processing time should be positive")
		}

		// Performance check
		if processingDuration > 5*time.Second {
			t.Errorf("Processing %d receipts took too long: %v", numReceipts, processingDuration)
		}

		t.Logf("Processed %d receipts in %v (avg: %v per receipt)",
			numReceipts, processingDuration, processingDuration/time.Duration(numReceipts))
	})
}

// TestSubscriber implements ReceiptSubscriber for testing
type TestSubscriber struct {
	OnReceiptUpdateFunc      func(*Receipt)
	OnAsyncReceiptUpdateFunc func(*AsyncReceiptTracker)
}

func (ts *TestSubscriber) OnReceiptUpdate(receipt *Receipt) {
	if ts.OnReceiptUpdateFunc != nil {
		ts.OnReceiptUpdateFunc(receipt)
	}
}

func (ts *TestSubscriber) OnAsyncReceiptUpdate(tracker *AsyncReceiptTracker) {
	if ts.OnAsyncReceiptUpdateFunc != nil {
		ts.OnAsyncReceiptUpdateFunc(tracker)
	}
}

// FailingStore implements PersistenceStore with controllable failures
type FailingStore struct {
	*MemoryStore
	shouldFail bool
	mutex      sync.RWMutex
}

func NewFailingStore() *FailingStore {
	return &FailingStore{
		MemoryStore: NewMemoryStore().(*MemoryStore),
		shouldFail:  false,
	}
}

func (fs *FailingStore) Store(receipt *Receipt) error {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if fs.shouldFail {
		return ErrStoreNotAvailable
	}
	return fs.MemoryStore.Store(receipt)
}

func (fs *FailingStore) StoreAsync(tracker *AsyncReceiptTracker) error {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if fs.shouldFail {
		return ErrStoreNotAvailable
	}
	return fs.MemoryStore.StoreAsync(tracker)
}

func (fs *FailingStore) BatchStore(receipts []*Receipt) error {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if fs.shouldFail {
		return ErrStoreNotAvailable
	}
	return fs.MemoryStore.BatchStore(receipts)
}

func (fs *FailingStore) SetShouldFail(shouldFail bool) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	fs.shouldFail = shouldFail
}