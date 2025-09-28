package async

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

// Use shared test logger from test_common.go

func createTestMessage() *message.Message {
	return &message.Message{
		ID:    "test-message-1",
		Title: "Test Message",
		Body:  "This is a test message",
	}
}

func createTestReceipt() *receipt.Receipt {
	return &receipt.Receipt{
		MessageID:  "test-message-1",
		Status:     "success",
		Successful: 1,
		Failed:     0,
		Total:      1,
		Timestamp:  time.Now(),
	}
}

func TestHandleCreation(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()

	handle := NewHandle(msg, callbacks)

	if handle.ID() != msg.ID {
		t.Errorf("Expected handle ID %s, got %s", msg.ID, handle.ID())
	}

	if handle.Status() != StatusPending {
		t.Errorf("Expected status %s, got %s", StatusPending, handle.Status())
	}

	result, err := handle.Result()
	if result != nil || err != nil {
		t.Errorf("Expected nil result for pending operation, got result: %v, err: %v", result, err)
	}
}

func TestHandleStatusTransitions(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks).(*HandleImpl)

	// Test pending to running
	handle.SetRunning()
	if handle.Status() != StatusRunning {
		t.Errorf("Expected status %s, got %s", StatusRunning, handle.Status())
	}

	// Test running to success
	testReceipt := createTestReceipt()
	handle.UpdateStatus(StatusCompleted, 1.0, testReceipt, nil)
	if handle.Status() != StatusSuccess {
		t.Errorf("Expected status %s, got %s", StatusSuccess, handle.Status())
	}

	result, err := handle.Result()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Errorf("Expected result, got nil")
	}
	if result.MessageID != testReceipt.MessageID {
		t.Errorf("Expected message ID %s, got %s", testReceipt.MessageID, result.MessageID)
	}
}

func TestHandleCancellation(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks)

	// Cancel the handle
	err := handle.Cancel()
	if err != nil {
		t.Errorf("Expected no error on cancel, got %v", err)
	}

	if handle.Status() != StatusCancelled {
		t.Errorf("Expected status %s, got %s", StatusCancelled, handle.Status())
	}

	// Try to cancel again
	err = handle.Cancel()
	if err != nil {
		t.Errorf("Expected no error on double cancel, got %v", err)
	}
}

func TestHandleCannotCancelCompleted(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks).(*HandleImpl)

	// Complete the handle first
	testReceipt := createTestReceipt()
	handle.UpdateStatus(StatusCompleted, 1.0, testReceipt, nil)

	// Try to cancel completed handle
	err := handle.Cancel()
	if err == nil {
		t.Errorf("Expected error when cancelling completed operation")
	}
}

func TestHandleWaitBlocking(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks).(*HandleImpl)

	var wg sync.WaitGroup
	var result *receipt.Receipt
	var err error

	// Start waiting in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err = handle.Wait(context.Background())
	}()

	// Give the waiter time to start
	time.Sleep(10 * time.Millisecond)

	// Complete the operation
	testReceipt := createTestReceipt()
	handle.UpdateStatus(StatusCompleted, 1.0, testReceipt, nil)

	// Wait for completion
	wg.Wait()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Errorf("Expected result, got nil")
	}
	if result.MessageID != testReceipt.MessageID {
		t.Errorf("Expected message ID %s, got %s", testReceipt.MessageID, result.MessageID)
	}
}

func TestHandleWaitTimeout(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := handle.Wait(ctx)

	if err == nil {
		t.Errorf("Expected timeout error")
	}
	if result != nil {
		t.Errorf("Expected nil result on timeout, got %v", result)
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestHandleMultipleConcurrentWaiters(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks).(*HandleImpl)

	const numWaiters = 10
	var wg sync.WaitGroup
	results := make([]*receipt.Receipt, numWaiters)
	errors := make([]error, numWaiters)

	// Start multiple waiters
	for i := 0; i < numWaiters; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index], errors[index] = handle.Wait(context.Background())
		}(i)
	}

	// Give waiters time to start
	time.Sleep(10 * time.Millisecond)

	// Verify waiters are registered
	if handle.GetWaiterCount() != numWaiters {
		t.Errorf("Expected %d waiters, got %d", numWaiters, handle.GetWaiterCount())
	}

	// Complete the operation
	testReceipt := createTestReceipt()
	handle.UpdateStatus(StatusCompleted, 1.0, testReceipt, nil)

	// Wait for all waiters to complete
	wg.Wait()

	// Verify all waiters got the result
	for i := 0; i < numWaiters; i++ {
		if errors[i] != nil {
			t.Errorf("Waiter %d got error: %v", i, errors[i])
		}
		if results[i] == nil {
			t.Errorf("Waiter %d got nil result", i)
		} else if results[i].MessageID != testReceipt.MessageID {
			t.Errorf("Waiter %d got wrong message ID: %s", i, results[i].MessageID)
		}
	}

	// Verify waiters are cleaned up
	if handle.GetWaiterCount() != 0 {
		t.Errorf("Expected 0 waiters after completion, got %d", handle.GetWaiterCount())
	}
}

func TestHandleCallbackChaining(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks)

	var completionCalled bool
	var errorCalled bool

	// Chain callbacks using fluent interface
	handle.OnComplete(func(r *receipt.Receipt) {
		completionCalled = true
	}).OnError(func(m *message.Message, err error) {
		errorCalled = true
	}).OnProgress(func(completed, total int) {
		// Progress callback for chaining test
	})

	// Complete successfully
	testReceipt := createTestReceipt()
	handleImpl := handle.(*HandleImpl)
	handleImpl.UpdateStatus(StatusCompleted, 1.0, testReceipt, nil)

	// Give callbacks time to execute
	time.Sleep(10 * time.Millisecond)

	if !completionCalled {
		t.Errorf("Expected completion callback to be called")
	}
	if errorCalled {
		t.Errorf("Expected error callback not to be called")
	}
}

func TestHandleErrorCallback(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks)

	var errorCalled bool
	var receivedError error

	handle.OnError(func(m *message.Message, err error) {
		errorCalled = true
		receivedError = err
	})

	// Fail the operation
	testError := fmt.Errorf("test error")
	handleImpl := handle.(*HandleImpl)
	handleImpl.UpdateStatus(StatusFailedOp, 0.0, nil, testError)

	// Give callback time to execute
	time.Sleep(10 * time.Millisecond)

	if !errorCalled {
		t.Errorf("Expected error callback to be called")
	}
	if receivedError == nil || receivedError.Error() != testError.Error() {
		t.Errorf("Expected error %v, got %v", testError, receivedError)
	}
}

func TestHandleProgressCallback(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks)

	var progressCalled bool
	var lastCompleted, lastTotal int

	handle.OnProgress(func(completed, total int) {
		progressCalled = true
		lastCompleted = completed
		lastTotal = total
	})

	// Update progress
	handleImpl := handle.(*HandleImpl)
	handleImpl.UpdateStatus(StatusProcessing, 0.5, nil, nil)

	// Give callback time to execute
	time.Sleep(10 * time.Millisecond)

	if !progressCalled {
		t.Errorf("Expected progress callback to be called")
	}
	if lastCompleted != 50 || lastTotal != 100 {
		t.Errorf("Expected progress 50/100, got %d/%d", lastCompleted, lastTotal)
	}
}

func TestHandleRegistry(t *testing.T) {
	registry := NewHandleRegistry(10, 100*time.Millisecond)
	defer registry.Shutdown()

	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks).(*HandleImpl)

	// Test registration
	err := registry.Register(handle)
	if err != nil {
		t.Errorf("Expected no error on registration, got %v", err)
	}

	if registry.GetActiveCount() != 1 {
		t.Errorf("Expected 1 active handle, got %d", registry.GetActiveCount())
	}

	// Test retrieval
	retrievedHandle, exists := registry.Get(msg.ID)
	if !exists {
		t.Errorf("Expected handle to exist in registry")
	}
	if retrievedHandle.messageID != handle.messageID {
		t.Errorf("Expected same handle, got different one")
	}

	// Test removal
	registry.Remove(msg.ID)
	if registry.GetActiveCount() != 0 {
		t.Errorf("Expected 0 active handles after removal, got %d", registry.GetActiveCount())
	}
}

func TestHandleRegistryCapacityLimit(t *testing.T) {
	registry := NewHandleRegistry(2, 100*time.Millisecond)
	defer registry.Shutdown()

	callbacks := NewCallbackRegistry(getTestLogger())

	// Register maximum handles
	for i := 0; i < 2; i++ {
		msg := &message.Message{ID: fmt.Sprintf("msg-%d", i)}
		handle := NewHandle(msg, callbacks).(*HandleImpl)
		err := registry.Register(handle)
		if err != nil {
			t.Errorf("Expected no error registering handle %d, got %v", i, err)
		}
	}

	// Try to register one more (should fail)
	msg := &message.Message{ID: "msg-overflow"}
	handle := NewHandle(msg, callbacks).(*HandleImpl)
	err := registry.Register(handle)
	if err == nil {
		t.Errorf("Expected error when exceeding registry capacity")
	}
}

func TestHandleRegistryGarbageCollection(t *testing.T) {
	registry := NewHandleRegistry(10, 50*time.Millisecond)
	defer registry.Shutdown()

	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks).(*HandleImpl)

	// Register and complete handle
	registry.Register(handle)
	testReceipt := createTestReceipt()
	handle.UpdateStatus(StatusCompleted, 1.0, testReceipt, nil)

	// Wait for GC cycle
	time.Sleep(100 * time.Millisecond)

	// Handle should be cleaned up
	if registry.GetActiveCount() > 0 {
		t.Errorf("Expected handles to be garbage collected, but %d still active", registry.GetActiveCount())
	}
}

func TestHandleTimeout(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks).(*HandleImpl)

	// Handle should not be timed out initially
	if handle.IsTimeout(1 * time.Second) {
		t.Errorf("Expected handle not to be timed out")
	}

	// Wait a bit and check timeout with very short duration
	time.Sleep(10 * time.Millisecond)
	if !handle.IsTimeout(1 * time.Millisecond) {
		t.Errorf("Expected handle to be timed out")
	}

	// Complete the handle
	testReceipt := createTestReceipt()
	handle.UpdateStatus(StatusCompleted, 1.0, testReceipt, nil)

	// Completed handle should not be considered timed out
	if handle.IsTimeout(1 * time.Millisecond) {
		t.Errorf("Expected completed handle not to be timed out")
	}
}

func TestHandleCleanup(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()
	handle := NewHandle(msg, callbacks).(*HandleImpl)

	// Add some waiters
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		handle.Wait(ctx)
	}()

	// Give waiter time to register
	time.Sleep(10 * time.Millisecond)

	// Verify waiter is registered
	if handle.GetWaiterCount() != 1 {
		t.Errorf("Expected 1 waiter, got %d", handle.GetWaiterCount())
	}

	// Cleanup handle
	handle.Cleanup()

	// Cancel context to unblock the waiter
	cancel()
	wg.Wait()

	// Verify waiters are cleaned up
	if handle.GetWaiterCount() != 0 {
		t.Errorf("Expected 0 waiters after cleanup, got %d", handle.GetWaiterCount())
	}
}

func TestHandleBackwardCompatibility(t *testing.T) {
	callbacks := NewCallbackRegistry(getTestLogger())
	msg := createTestMessage()

	// Test AsyncHandle interface
	asyncHandle := NewAsyncHandle(msg, callbacks)

	if asyncHandle.MessageID() != msg.ID {
		t.Errorf("Expected message ID %s, got %s", msg.ID, asyncHandle.MessageID())
	}

	// Test AsyncStatus method
	status := asyncHandle.AsyncStatus()
	if status.MessageID != msg.ID {
		t.Errorf("Expected message ID %s in status, got %s", msg.ID, status.MessageID)
	}
	if status.Status != StatusPendingOp {
		t.Errorf("Expected status %s, got %s", StatusPendingOp, status.Status)
	}
}