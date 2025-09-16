package queue_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/notifiers"
	"github.com/kart-io/notifyhub/queue"
)

func TestSimpleQueue(t *testing.T) {
	// Test queue creation
	q := queue.NewSimple(10)
	if q == nil {
		t.Fatal("Queue should not be nil")
	}

	// Test queue enqueue
	ctx := context.Background()
	message := &queue.Message{
		Message: &notifiers.Message{
			Title: "Test Message",
			Body:  "Test Body",
		},
	}

	taskID, err := q.Enqueue(ctx, message)
	if err != nil {
		t.Fatalf("Failed to enqueue message: %v", err)
	}

	if taskID == "" {
		t.Error("Task ID should not be empty")
	}

	// Test queue dequeue
	dequeuedMessage, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Failed to dequeue message: %v", err)
	}

	if dequeuedMessage == nil {
		t.Fatal("Dequeued message should not be nil")
	}

	if dequeuedMessage.Message.Title != "Test Message" {
		t.Error("Dequeued message title should match")
	}

	// Test empty queue dequeue (should block, so use context with timeout)
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err = q.Dequeue(ctxTimeout)
	if err == nil {
		t.Error("Dequeue from empty queue should timeout")
	}

	// Test queue close
	err = q.Close()
	if err != nil {
		t.Errorf("Failed to close queue: %v", err)
	}

	// Test enqueue after close should fail
	_, err = q.Enqueue(ctx, message)
	if err == nil {
		t.Error("Enqueue after close should fail")
	}
}

func TestRetryPolicy(t *testing.T) {
	// Test default retry policy
	policy := queue.DefaultRetryPolicy()
	if policy == nil {
		t.Fatal("Default retry policy should not be nil")
	}

	if policy.MaxRetries != 3 {
		t.Error("Default retry policy should have 3 max retries")
	}

	if policy.InitialInterval != 30*time.Second {
		t.Error("Default retry policy should have 30s initial interval")
	}

	if policy.Multiplier != 2.0 {
		t.Error("Default retry policy should have 2.0 multiplier")
	}

	// Test should retry logic
	if !policy.ShouldRetry(0) {
		t.Error("Should retry on attempt 0")
	}

	if !policy.ShouldRetry(2) {
		t.Error("Should retry on attempt 2")
	}

	if policy.ShouldRetry(3) {
		t.Error("Should not retry on attempt 3 (max retries reached)")
	}

	// Test backoff calculation through NextRetry (with jitter tolerance)
	nextRetry1 := policy.NextRetry(0)
	expectedTime1 := time.Now().Add(30 * time.Second)
	// Allow for jitter - policy has 5s max jitter
	if nextRetry1.Before(expectedTime1.Add(-1*time.Second)) || nextRetry1.After(expectedTime1.Add(6*time.Second)) {
		t.Errorf("First retry should be around 30s from now (got %v, expected around %v)", nextRetry1, expectedTime1)
	}

	nextRetry2 := policy.NextRetry(1)
	expectedTime2 := time.Now().Add(60 * time.Second) // 30s * 2
	// Allow for jitter - policy has 5s max jitter
	if nextRetry2.Before(expectedTime2.Add(-1*time.Second)) || nextRetry2.After(expectedTime2.Add(6*time.Second)) {
		t.Errorf("Second retry should be around 60s from now (got %v, expected around %v)", nextRetry2, expectedTime2)
	}

	// Test custom retry policy
	customPolicy := queue.ExponentialBackoffPolicy(5, 500*time.Millisecond, 1.5)
	if customPolicy.MaxRetries != 5 {
		t.Error("Custom policy should have 5 max retries")
	}

	if customPolicy.InitialInterval != 500*time.Millisecond {
		t.Error("Custom policy should have 500ms initial interval")
	}

	if customPolicy.Multiplier != 1.5 {
		t.Error("Custom policy should have 1.5 multiplier")
	}

	// Test no retry policy
	noRetryPolicy := queue.NoRetryPolicy()
	if noRetryPolicy.ShouldRetry(0) {
		t.Error("No retry policy should not retry")
	}
}

func TestCallbackOptions(t *testing.T) {
	// Test callback options creation
	options := &queue.CallbackOptions{
		WebhookURL:      "https://example.com/webhook",
		WebhookSecret:   "webhook-secret",
		CallbackTimeout: 30 * time.Second,
	}

	// Test adding callbacks
	sentCallback := queue.NewCallbackFunc("sent-logger", func(ctx context.Context, callbackCtx *queue.CallbackContext) error {
		if callbackCtx.Event != queue.CallbackEventSent {
			t.Error("Callback event should be sent")
		}
		return nil
	})

	failedCallback := queue.NewLoggingCallback("failed-logger", func(format string, v ...interface{}) {
		// Simple logging function
	})

	options.AddCallback(queue.CallbackEventSent, sentCallback)
	options.AddCallback(queue.CallbackEventFailed, failedCallback)

	// Test callback retrieval through direct field access
	if len(options.OnSent) != 1 {
		t.Error("Should have one sent callback")
	}

	if len(options.OnFailed) != 1 {
		t.Error("Should have one failed callback")
	}

	if len(options.OnRetry) != 0 {
		t.Error("Should have no retry callbacks")
	}
}

func TestCallbackEvents(t *testing.T) {
	// Test callback event constants
	if queue.CallbackEventSent != "sent" {
		t.Error("queue.CallbackEventSent should be 'sent'")
	}

	if queue.CallbackEventFailed != "failed" {
		t.Error("queue.CallbackEventFailed should be 'failed'")
	}

	if queue.CallbackEventRetry != "retry" {
		t.Error("queue.CallbackEventRetry should be 'retry'")
	}

	if queue.CallbackEventMaxRetries != "max_retries" {
		t.Error("queue.CallbackEventMaxRetries should be 'max_retries'")
	}
}

func TestCallbackContext(t *testing.T) {
	// Test callback context creation
	now := time.Now()
	message := &notifiers.Message{
		Title: "Test Message",
		Body:  "Test Body",
	}

	results := []*notifiers.SendResult{
		{
			Platform: "test",
			Success:  true,
			SentAt:   now,
		},
	}

	ctx := &queue.CallbackContext{
		MessageID:  "test-123",
		Event:      queue.CallbackEventSent,
		Message:    message,
		Results:    results,
		Error:      nil,
		Attempts:   2,
		ExecutedAt: now,
		Duration:   100 * time.Millisecond,
	}

	if ctx.MessageID != "test-123" {
		t.Error("queue.CallbackContext MessageID should be test-123")
	}

	if ctx.Event != queue.CallbackEventSent {
		t.Error("queue.CallbackContext Event should be sent")
	}

	if ctx.Message.Title != "Test Message" {
		t.Error("queue.CallbackContext Message title should match")
	}

	if len(ctx.Results) != 1 {
		t.Error("queue.CallbackContext should have one result")
	}

	if ctx.Error != nil {
		t.Error("queue.CallbackContext Error should be nil")
	}

	if ctx.Attempts != 2 {
		t.Error("queue.CallbackContext Attempts should be 2")
	}

	if ctx.Duration != 100*time.Millisecond {
		t.Error("queue.CallbackContext Duration should be 100ms")
	}
}

// Mock message sender for testing worker
type mockMessageSender struct {
	sendCalled    bool
	shouldFail    bool
	callbackCount int
}

func (m *mockMessageSender) SendSync(ctx context.Context, message *notifiers.Message, options interface{}) ([]*notifiers.SendResult, error) {
	m.sendCalled = true
	m.callbackCount++

	if m.shouldFail {
		return []*notifiers.SendResult{
			{
				Platform: "mock",
				Success:  false,
				Error:    "mock failure",
				SentAt:   time.Now(),
			},
		}, fmt.Errorf("mock failure")
	}

	return []*notifiers.SendResult{
		{
			Platform: "mock",
			Success:  true,
			SentAt:   time.Now(),
		},
	}, nil
}

func TestWorkerBasic(t *testing.T) {
	// Create test queue and mock sender
	q := queue.NewSimple(10)
	sender := &mockMessageSender{}
	policy := queue.NoRetryPolicy() // No retry for simplicity

	// Create worker
	worker := queue.NewWorker(q, sender, policy, 1)
	if worker == nil {
		t.Fatal("Worker should not be nil")
	}

	ctx := context.Background()

	// Start worker
	err := worker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Enqueue a message
	message := &queue.Message{
		Message: &notifiers.Message{
			Title: "Worker Test",
			Body:  "Worker Test Body",
		},
	}

	_, err = q.Enqueue(ctx, message)
	if err != nil {
		t.Fatalf("Failed to enqueue message: %v", err)
	}

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop worker
	worker.Stop()

	// Check that message was processed
	if !sender.sendCalled {
		t.Error("Worker should have called SendSync")
	}

	// Close queue
	q.Close()
}

func TestCallbackFunc(t *testing.T) {
	// Test callback function creation
	called := false
	callback := queue.NewCallbackFunc("test-callback", func(ctx context.Context, callbackCtx *queue.CallbackContext) error {
		called = true
		return nil
	})

	if callback.Name() != "test-callback" {
		t.Error("Callback name should be test-callback")
	}

	// Test callback execution
	ctx := context.Background()
	callbackCtx := &queue.CallbackContext{
		Event: queue.CallbackEventSent,
	}

	err := callback.Execute(ctx, callbackCtx)
	if err != nil {
		t.Errorf("Callback execution should not fail: %v", err)
	}

	if !called {
		t.Error("Callback function should have been called")
	}
}

func TestLoggingCallback(t *testing.T) {
	// Test logging callback creation
	logged := ""
	logFunc := func(format string, v ...interface{}) {
		logged = format
	}

	callback := queue.NewLoggingCallback("logging-callback", logFunc)
	if callback.Name() != "logging-callback" {
		t.Error("Logging callback name should be logging-callback")
	}

	// Test callback execution
	ctx := context.Background()
	callbackCtx := &queue.CallbackContext{
		Event:     queue.CallbackEventSent,
		MessageID: "test-123",
	}

	err := callback.Execute(ctx, callbackCtx)
	if err != nil {
		t.Errorf("Logging callback execution should not fail: %v", err)
	}

	if logged == "" {
		t.Error("Logging callback should have logged something")
	}
}

func TestEnhancedQueue(t *testing.T) {
	// Test enhanced queue creation
	baseQueue := queue.NewSimple(10)
	enhancedQueue := queue.NewEnhancedQueue(baseQueue)
	if enhancedQueue == nil {
		t.Fatal("Enhanced queue should not be nil")
	}

	ctx := context.Background()

	// Test immediate message (no delay)
	immediateMsg := &queue.Message{
		Message: &notifiers.Message{
			Title: "Immediate Message",
			Body:  "No delay",
			Delay: 0,
		},
	}

	taskID, err := enhancedQueue.Enqueue(ctx, immediateMsg)
	if err != nil {
		t.Fatalf("Failed to enqueue immediate message: %v", err)
	}

	if taskID == "" {
		t.Error("Task ID should not be empty")
	}

	// Test dequeue immediate message
	dequeuedMsg, err := enhancedQueue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Failed to dequeue message: %v", err)
	}

	if dequeuedMsg.Message.Title != "Immediate Message" {
		t.Error("Dequeued message should match")
	}

	// Test delayed message (should not be immediately available)
	delayedMsg := &queue.Message{
		Message: &notifiers.Message{
			Title: "Delayed Message",
			Body:  "With delay",
			Delay: 100 * time.Millisecond,
		},
	}

	_, err = enhancedQueue.Enqueue(ctx, delayedMsg)
	if err != nil {
		t.Fatalf("Failed to enqueue delayed message: %v", err)
	}

	// Immediate dequeue should timeout (message is scheduled)
	ctxTimeout, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	_, err = enhancedQueue.Dequeue(ctxTimeout)
	if err == nil {
		t.Error("Dequeue should timeout for delayed message")
	}

	// Wait for delay to pass, then check if message becomes available
	time.Sleep(150 * time.Millisecond)

	dequeuedDelayedMsg, err := enhancedQueue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Failed to dequeue delayed message after delay: %v", err)
	}

	if dequeuedDelayedMsg.Message.Title != "Delayed Message" {
		t.Error("Delayed message should be available after delay")
	}

	// Test queue health
	err = enhancedQueue.Health(ctx)
	if err != nil {
		t.Errorf("Enhanced queue health check should pass: %v", err)
	}

	// Test queue size
	size := enhancedQueue.Size()
	if size < 0 {
		t.Error("Queue size should not be negative")
	}

	// Close enhanced queue
	err = enhancedQueue.Close()
	if err != nil {
		t.Errorf("Failed to close enhanced queue: %v", err)
	}
}
