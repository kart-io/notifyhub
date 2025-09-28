package receipt

import (
	"testing"
	"time"
)

func TestMemoryStore_Store(t *testing.T) {
	store := NewMemoryStore()

	receipt := &Receipt{
		MessageID: "test-msg-1",
		Status:    "completed",
		Results: []PlatformResult{
			{Platform: "feishu", Target: "user1", Success: true, Timestamp: time.Now()},
		},
		Successful: 1,
		Failed:     0,
		Total:      1,
		Timestamp:  time.Now(),
	}

	err := store.Store(receipt)
	if err != nil {
		t.Fatalf("Failed to store receipt: %v", err)
	}

	// Verify storage
	retrieved, err := store.Get("test-msg-1")
	if err != nil {
		t.Fatalf("Failed to get receipt: %v", err)
	}

	if retrieved.MessageID != receipt.MessageID {
		t.Error("Retrieved receipt doesn't match stored receipt")
	}

	if len(retrieved.Results) != len(receipt.Results) {
		t.Error("Retrieved receipt results don't match stored receipt")
	}
}

func TestMemoryStore_StoreAsync(t *testing.T) {
	store := NewMemoryStore()

	asyncReceipt := &AsyncReceipt{
		MessageID: "async-msg-1",
		Status:    "processing",
		QueuedAt:  time.Now(),
	}

	tracker := &AsyncReceiptTracker{
		Receipt:  asyncReceipt,
		HandleID: "handle-1",
		Progress: &ProgressInfo{
			TotalTargets:      10,
			ProcessedTargets:  5,
			SuccessfulTargets: 4,
			FailedTargets:     1,
			ProgressPercent:   50.0,
		},
		UpdatedAt: time.Now(),
	}

	err := store.StoreAsync(tracker)
	if err != nil {
		t.Fatalf("Failed to store async receipt: %v", err)
	}

	// Verify storage
	retrieved, err := store.GetAsync("async-msg-1")
	if err != nil {
		t.Fatalf("Failed to get async receipt: %v", err)
	}

	if retrieved.Receipt.MessageID != tracker.Receipt.MessageID {
		t.Error("Retrieved async receipt doesn't match stored receipt")
	}

	if retrieved.HandleID != tracker.HandleID {
		t.Error("Retrieved handle ID doesn't match stored handle ID")
	}

	if retrieved.Progress.ProgressPercent != tracker.Progress.ProgressPercent {
		t.Error("Retrieved progress doesn't match stored progress")
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.Get("non-existent")
	if err != ErrReceiptNotFound {
		t.Errorf("Expected ErrReceiptNotFound, got %v", err)
	}
}

func TestMemoryStore_GetAsync_NotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.GetAsync("non-existent")
	if err != ErrReceiptNotFound {
		t.Errorf("Expected ErrReceiptNotFound, got %v", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore()

	now := time.Now()
	receipts := []*Receipt{
		{
			MessageID: "msg-1",
			Status:    "completed",
			Results: []PlatformResult{
				{Platform: "feishu", Success: true},
			},
			Timestamp: now,
		},
		{
			MessageID: "msg-2",
			Status:    "failed",
			Results: []PlatformResult{
				{Platform: "email", Success: false},
			},
			Timestamp: now.Add(1 * time.Hour),
		},
		{
			MessageID: "msg-3",
			Status:    "completed",
			Results: []PlatformResult{
				{Platform: "webhook", Success: true},
			},
			Timestamp: now.Add(2 * time.Hour),
		},
	}

	// Store all receipts
	for _, receipt := range receipts {
		err := store.Store(receipt)
		if err != nil {
			t.Fatalf("Failed to store receipt: %v", err)
		}
	}

	// Test filter by status
	filter := ReceiptFilter{
		Status: []string{"completed"},
	}

	results, err := store.List(filter)
	if err != nil {
		t.Fatalf("Failed to list receipts: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 completed receipts, got %d", len(results))
	}

	// Test filter by platform
	filter = ReceiptFilter{
		Platforms: []string{"feishu"},
	}

	results, err = store.List(filter)
	if err != nil {
		t.Fatalf("Failed to list receipts: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 feishu receipt, got %d", len(results))
	}

	// Test time range filter
	startTime := now.Add(30 * time.Minute)
	filter = ReceiptFilter{
		StartTime: &startTime,
	}

	results, err = store.List(filter)
	if err != nil {
		t.Fatalf("Failed to list receipts: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 receipts after start time, got %d", len(results))
	}

	// Test limit and offset
	filter = ReceiptFilter{
		Limit:  1,
		Offset: 1,
	}

	results, err = store.List(filter)
	if err != nil {
		t.Fatalf("Failed to list receipts: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 receipt with limit/offset, got %d", len(results))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()

	receipt := &Receipt{
		MessageID: "delete-test",
		Status:    "completed",
		Timestamp: time.Now(),
	}

	// Store receipt
	err := store.Store(receipt)
	if err != nil {
		t.Fatalf("Failed to store receipt: %v", err)
	}

	// Verify it exists
	_, err = store.Get("delete-test")
	if err != nil {
		t.Fatalf("Receipt should exist before deletion: %v", err)
	}

	// Delete receipt
	err = store.Delete("delete-test")
	if err != nil {
		t.Fatalf("Failed to delete receipt: %v", err)
	}

	// Verify it's gone
	_, err = store.Get("delete-test")
	if err != ErrReceiptNotFound {
		t.Error("Receipt should not exist after deletion")
	}
}

func TestMemoryStore_BatchStore(t *testing.T) {
	store := NewMemoryStore()

	receipts := []*Receipt{
		{
			MessageID: "batch-1",
			Status:    "completed",
			Timestamp: time.Now(),
		},
		{
			MessageID: "batch-2",
			Status:    "failed",
			Timestamp: time.Now(),
		},
		{
			MessageID: "batch-3",
			Status:    "completed",
			Timestamp: time.Now(),
		},
	}

	err := store.BatchStore(receipts)
	if err != nil {
		t.Fatalf("Failed to batch store receipts: %v", err)
	}

	// Verify all receipts were stored
	for _, receipt := range receipts {
		retrieved, err := store.Get(receipt.MessageID)
		if err != nil {
			t.Errorf("Failed to get batch stored receipt %s: %v", receipt.MessageID, err)
		} else if retrieved.Status != receipt.Status {
			t.Errorf("Batch stored receipt status doesn't match: expected %s, got %s", receipt.Status, retrieved.Status)
		}
	}
}

func TestMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore()

	// Store some data
	receipt := &Receipt{
		MessageID: "close-test",
		Status:    "completed",
		Timestamp: time.Now(),
	}

	err := store.Store(receipt)
	if err != nil {
		t.Fatalf("Failed to store receipt: %v", err)
	}

	// Close store
	err = store.Close()
	if err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	// Verify data is cleared
	_, err = store.Get("close-test")
	if err != ErrReceiptNotFound {
		t.Error("Data should be cleared after close")
	}
}

func TestMemoryStore_DeepCopy(t *testing.T) {
	store := NewMemoryStore()

	originalReceipt := &Receipt{
		MessageID: "copy-test",
		Status:    "completed",
		Results: []PlatformResult{
			{Platform: "feishu", Target: "user1", Success: true},
		},
		Timestamp: time.Now(),
	}

	// Store receipt
	err := store.Store(originalReceipt)
	if err != nil {
		t.Fatalf("Failed to store receipt: %v", err)
	}

	// Get receipt and modify it
	retrieved, err := store.Get("copy-test")
	if err != nil {
		t.Fatalf("Failed to get receipt: %v", err)
	}

	// Modify the retrieved receipt
	retrieved.Status = "modified"
	retrieved.Results[0].Success = false

	// Get receipt again and verify original is unchanged
	retrieved2, err := store.Get("copy-test")
	if err != nil {
		t.Fatalf("Failed to get receipt again: %v", err)
	}

	if retrieved2.Status == "modified" {
		t.Error("Original receipt was modified - deep copy not working")
	}

	if !retrieved2.Results[0].Success {
		t.Error("Original receipt results were modified - deep copy not working")
	}
}