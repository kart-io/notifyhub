package async

import (
	"context"
	"testing"
	"time"

	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/receipt"
	"github.com/kart/notifyhub/pkg/target"
)

func TestMemoryHandle_Status(t *testing.T) {
	handle := NewMemoryHandle("test-123")

	status := handle.Status()

	if status.ID != "test-123" {
		t.Errorf("Status.ID = %v, want %v", status.ID, "test-123")
	}
	if status.State != StatePending {
		t.Errorf("Status.State = %v, want %v", status.State, StatePending)
	}
}

func TestMemoryHandle_SetResult(t *testing.T) {
	handle := NewMemoryHandle("test-123")

	testReceipt := &receipt.Receipt{
		MessageID: "test-123",
		Status:    receipt.StatusSuccess,
	}

	result := Result{
		Receipt: testReceipt,
		Error:   nil,
	}

	handle.SetResult(result)

	status := handle.Status()
	if status.State != StateCompleted {
		t.Errorf("Status.State = %v, want %v", status.State, StateCompleted)
	}
}

func TestMemoryHandle_SetResultWithError(t *testing.T) {
	handle := NewMemoryHandle("test-123")

	result := Result{
		Receipt: nil,
		Error:   ErrTestError,
	}

	handle.SetResult(result)

	status := handle.Status()
	if status.State != StateFailed {
		t.Errorf("Status.State = %v, want %v", status.State, StateFailed)
	}
}

var ErrTestError = &testError{"test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestMemoryHandle_Wait(t *testing.T) {
	handle := NewMemoryHandle("test-123")

	testReceipt := &receipt.Receipt{
		MessageID: "test-123",
		Status:    receipt.StatusSuccess,
	}

	// Set result in goroutine
	go func() {
		time.Sleep(50 * time.Millisecond)
		handle.SetResult(Result{Receipt: testReceipt})
	}()

	ctx := context.Background()
	gotReceipt, err := handle.Wait(ctx)

	if err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}
	if gotReceipt == nil {
		t.Fatal("Wait() receipt is nil")
	}
	if gotReceipt.MessageID != "test-123" {
		t.Errorf("Receipt.MessageID = %v, want %v", gotReceipt.MessageID, "test-123")
	}
}

func TestMemoryHandle_WaitTimeout(t *testing.T) {
	handle := NewMemoryHandle("test-123")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := handle.Wait(ctx)

	if err == nil {
		t.Error("Wait() should timeout but got nil error")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Wait() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestMemoryHandle_Cancel(t *testing.T) {
	handle := NewMemoryHandle("test-123")

	err := handle.Cancel()
	if err != nil {
		t.Errorf("Cancel() error = %v, want nil", err)
	}

	status := handle.Status()
	if status.State != StateCancelled {
		t.Errorf("Status.State = %v, want %v", status.State, StateCancelled)
	}
}

func TestMemoryQueue_NewMemoryQueue(t *testing.T) {
	config := QueueConfig{
		Workers:    4,
		BufferSize: 100,
		Timeout:    30 * time.Second,
	}

	queue := NewMemoryQueue(config)

	if queue == nil {
		t.Fatal("NewMemoryQueue() returned nil")
	}
	if queue.config.Workers != 4 {
		t.Errorf("Queue workers = %v, want %v", queue.config.Workers, 4)
	}
	if queue.config.BufferSize != 100 {
		t.Errorf("Queue buffer size = %v, want %v", queue.config.BufferSize, 100)
	}
}

func TestMemoryQueue_NewMemoryQueue_DefaultValues(t *testing.T) {
	config := QueueConfig{}
	queue := NewMemoryQueue(config)

	if queue.config.Workers != 4 {
		t.Errorf("Default workers = %v, want %v", queue.config.Workers, 4)
	}
	if queue.config.BufferSize != 1000 {
		t.Errorf("Default buffer size = %v, want %v", queue.config.BufferSize, 1000)
	}
}

func TestMemoryQueue_Enqueue(t *testing.T) {
	queue := NewMemoryQueue(QueueConfig{Workers: 2, BufferSize: 10})

	msg := &message.Message{
		ID:    "test-msg",
		Title: "Test",
		Body:  "Test message",
	}

	ctx := context.Background()
	handle, err := queue.Enqueue(ctx, msg, []target.Target{}, nil)

	if err != nil {
		t.Errorf("Enqueue() error = %v, want nil", err)
	}
	if handle == nil {
		t.Fatal("Enqueue() handle is nil")
	}
	if handle.ID() != "test-msg" {
		t.Errorf("Handle.ID() = %v, want %v", handle.ID(), "test-msg")
	}

	stats := queue.GetStats()
	if stats.Pending != 1 {
		t.Errorf("Queue pending = %v, want %v", stats.Pending, 1)
	}
}

func TestMemoryQueue_EnqueueAfterClose(t *testing.T) {
	queue := NewMemoryQueue(QueueConfig{Workers: 2, BufferSize: 10})

	ctx := context.Background()
	_ = queue.Stop(ctx)

	msg := &message.Message{
		ID:    "test-msg",
		Title: "Test",
		Body:  "Test message",
	}

	_, err := queue.Enqueue(ctx, msg, []target.Target{}, nil)

	if err == nil {
		t.Error("Enqueue() after Stop should return error")
	}
}

func TestMemoryQueue_StartStop(t *testing.T) {
	queue := NewMemoryQueue(QueueConfig{Workers: 2, BufferSize: 10})

	ctx := context.Background()

	// Start queue
	err := queue.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v, want nil", err)
	}

	// Stop queue
	err = queue.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}

	// Stop again should not error
	err = queue.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() second time error = %v, want nil", err)
	}
}

func TestMemoryQueue_GetStats(t *testing.T) {
	queue := NewMemoryQueue(QueueConfig{Workers: 2, BufferSize: 10})

	stats := queue.GetStats()

	if stats.Workers != 0 {
		t.Errorf("Stats.Workers = %v, want %v (not started)", stats.Workers, 0)
	}
	if stats.Pending != 0 {
		t.Errorf("Stats.Pending = %v, want %v", stats.Pending, 0)
	}

	ctx := context.Background()
	_ = queue.Start(ctx)

	stats = queue.GetStats()
	if stats.Workers != 2 {
		t.Errorf("Stats.Workers after start = %v, want %v", stats.Workers, 2)
	}
}

func TestBatchHandle_Status(t *testing.T) {
	handles := []Handle{
		NewMemoryHandle("msg-1"),
		NewMemoryHandle("msg-2"),
		NewMemoryHandle("msg-3"),
	}

	batchHandle := NewBatchHandle(handles)

	status := batchHandle.Status()

	if status.Total != 3 {
		t.Errorf("BatchStatus.Total = %v, want %v", status.Total, 3)
	}
	if status.State != StatePending {
		t.Errorf("BatchStatus.State = %v, want %v", status.State, StatePending)
	}
}

func TestBatchHandle_AddResult(t *testing.T) {
	handles := []Handle{
		NewMemoryHandle("msg-1"),
		NewMemoryHandle("msg-2"),
	}

	batchHandle := NewBatchHandle(handles)

	// Add successful result
	batchHandle.AddResult(Result{
		Receipt: &receipt.Receipt{MessageID: "msg-1"},
		Error:   nil,
	})

	status := batchHandle.Status()
	if status.Completed != 1 {
		t.Errorf("BatchStatus.Completed = %v, want %v", status.Completed, 1)
	}
	if status.Progress != 0.5 {
		t.Errorf("BatchStatus.Progress = %v, want %v", status.Progress, 0.5)
	}

	// Add failed result
	batchHandle.AddResult(Result{
		Receipt: nil,
		Error:   ErrTestError,
	})

	status = batchHandle.Status()
	if status.Failed != 1 {
		t.Errorf("BatchStatus.Failed = %v, want %v", status.Failed, 1)
	}
	if status.Progress != 1.0 {
		t.Errorf("BatchStatus.Progress = %v, want %v", status.Progress, 1.0)
	}
	if status.State != StateFailed {
		t.Errorf("BatchStatus.State = %v, want %v", status.State, StateFailed)
	}
}

func TestBatchHandle_Cancel(t *testing.T) {
	handles := []Handle{
		NewMemoryHandle("msg-1"),
		NewMemoryHandle("msg-2"),
	}

	batchHandle := NewBatchHandle(handles)

	err := batchHandle.Cancel()
	if err != nil {
		t.Errorf("Cancel() error = %v, want nil", err)
	}

	status := batchHandle.Status()
	if status.State != StateCancelled {
		t.Errorf("BatchStatus.State = %v, want %v", status.State, StateCancelled)
	}

	// Check individual handles
	for _, h := range handles {
		if h.Status().State != StateCancelled {
			t.Errorf("Individual handle state = %v, want %v", h.Status().State, StateCancelled)
		}
	}
}

func TestCallbackManager_TriggerCallbacks(t *testing.T) {
	manager := NewCallbackManager()

	completeCalled := false
	errorCalled := false
	progressCalled := false

	manager.OnComplete(func(r *receipt.Receipt) {
		completeCalled = true
	})

	manager.OnError(func(m *message.Message, e error) {
		errorCalled = true
	})

	manager.OnProgress(func(completed, total int) {
		progressCalled = true
	})

	// Trigger callbacks
	manager.TriggerComplete(&receipt.Receipt{})
	manager.TriggerError(&message.Message{}, ErrTestError)
	manager.TriggerProgress(1, 10)

	if !completeCalled {
		t.Error("OnComplete callback was not called")
	}
	if !errorCalled {
		t.Error("OnError callback was not called")
	}
	if !progressCalled {
		t.Error("OnProgress callback was not called")
	}
}

func TestCallbackManager_HasCallbacks(t *testing.T) {
	manager := NewCallbackManager()

	if manager.HasCallbacks() {
		t.Error("HasCallbacks() = true, want false (no callbacks set)")
	}

	manager.OnComplete(func(r *receipt.Receipt) {})

	if !manager.HasCallbacks() {
		t.Error("HasCallbacks() = false, want true (callback set)")
	}
}
