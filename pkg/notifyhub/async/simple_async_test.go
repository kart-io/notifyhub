// Simple test to verify basic async functionality works
package async

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// TestBasicAsyncFunctionality tests the core async behavior
func TestBasicAsyncFunctionality(t *testing.T) {
	t.Parallel()

	testLogger := NewTestLogger()
	mockDispatcher := NewTrackingMockDispatcher()

	// Set a delay to make async behavior observable
	mockDispatcher.SetProcessDelay(100 * time.Millisecond)

	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 1
	config.MaxWorkers = 1
	config.TaskBatchSize = 1

	executor := NewAsyncExecutor(10, config, mockDispatcher, testLogger)
	require.NotNil(t, executor)

	err := executor.Start()
	require.NoError(t, err)
	defer func() {
		err := executor.Stop(5 * time.Second)
		assert.NoError(t, err)
	}()

	// Create a test message
	msg := &message.Message{
		ID:      "test-async-msg",
		Title:   "Test Async",
		Body:    "Testing async processing",
		Format:  message.FormatText,
		Targets: []target.Target{{Type: "test", Value: "test-target"}},
	}

	// Create callbacks
	callbacks := NewCallbackRegistry(testLogger)
	handle := NewAsyncHandle(msg, callbacks)
	require.NotNil(t, handle)

	// Record start time
	startTime := time.Now()

	// Enqueue the message (should be fast)
	err = executor.GetQueue().Enqueue(context.Background(), msg, handle)
	require.NoError(t, err)

	enqueueTime := time.Since(startTime)
	t.Logf("Enqueue time: %v", enqueueTime)

	// Enqueuing should be very fast
	assert.Less(t, enqueueTime, 50*time.Millisecond, "Enqueue should be immediate")

	// Wait for async processing to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := handle.Wait(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)

	totalTime := time.Since(startTime)
	t.Logf("Total processing time: %v", totalTime)

	// Should have taken at least the processing delay
	assert.GreaterOrEqual(t, totalTime, 90*time.Millisecond, "Should wait for async processing")
	assert.Less(t, totalTime, 1*time.Second, "Should not take too long")

	// Verify the dispatcher was called
	assert.Equal(t, 1, mockDispatcher.GetCallCount(), "Dispatcher should be called exactly once")

	// Verify queue is empty
	assert.True(t, executor.GetQueue().IsEmpty(), "Queue should be empty after processing")
}

// TestAsyncStatusProgression tests that status changes happen correctly
func TestAsyncStatusProgression(t *testing.T) {
	t.Parallel()

	testLogger := NewTestLogger()
	mockDispatcher := NewTrackingMockDispatcher()

	// Set a delay to observe status changes
	mockDispatcher.SetProcessDelay(200 * time.Millisecond)

	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 1
	config.MaxWorkers = 1
	config.TaskBatchSize = 1

	executor := NewAsyncExecutor(10, config, mockDispatcher, testLogger)
	require.NotNil(t, executor)

	err := executor.Start()
	require.NoError(t, err)
	defer func() {
		err := executor.Stop(5 * time.Second)
		assert.NoError(t, err)
	}()

	// Create a test message
	msg := &message.Message{
		ID:      "status-test-msg",
		Title:   "Status Test",
		Body:    "Testing status progression",
		Format:  message.FormatText,
		Targets: []target.Target{{Type: "test", Value: "status-target"}},
	}

	// Create callbacks
	callbacks := NewCallbackRegistry(testLogger)
	handle := NewAsyncHandle(msg, callbacks)
	require.NotNil(t, handle)

	// Check initial status
	initialStatus := handle.AsyncStatus()
	t.Logf("Initial status: %s", initialStatus.Status)

	// Enqueue the message
	err = executor.GetQueue().Enqueue(context.Background(), msg, handle)
	require.NoError(t, err)

	// Wait a bit for processing to start
	time.Sleep(50 * time.Millisecond)

	// Check status during processing
	processingStatus := handle.AsyncStatus()
	t.Logf("During processing status: %s", processingStatus.Status)

	// Wait for completion
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := handle.Wait(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check final status
	finalStatus := handle.AsyncStatus()
	t.Logf("Final status: %s", finalStatus.Status)

	// Verify status progression
	assert.Equal(t, StatusPendingOp, initialStatus.Status, "Should start as pending")
	assert.Equal(t, StatusCompleted, finalStatus.Status, "Should end as completed")
}

// TestAsyncVsSyncPerformance tests that async provides performance benefits
func TestAsyncVsSyncPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	testLogger := NewTestLogger()
	const messageCount = 10
	const processingDelay = 50 * time.Millisecond

	// Test synchronous processing time
	syncDispatcher := NewTrackingMockDispatcher()
	syncDispatcher.SetProcessDelay(processingDelay)

	syncStartTime := time.Now()
	for i := 0; i < messageCount; i++ {
		msg := &message.Message{
			ID:      "sync-msg-" + string(rune(i)),
			Title:   "Sync Test",
			Body:    "Testing sync processing",
			Format:  message.FormatText,
			Targets: []target.Target{{Type: "test", Value: "sync-target"}},
		}
		_, err := syncDispatcher.Dispatch(context.Background(), msg)
		require.NoError(t, err)
	}
	syncDuration := time.Since(syncStartTime)
	t.Logf("Sync processing time for %d messages: %v", messageCount, syncDuration)

	// Test asynchronous processing time
	asyncDispatcher := NewTrackingMockDispatcher()
	asyncDispatcher.SetProcessDelay(processingDelay)

	config := &WorkerPoolConfig{
		MinWorkers:      3,
		MaxWorkers:      3,
		TargetLoad:      0.7,
		ScaleUpDelay:    5 * time.Second,
		ScaleDownDelay:  30 * time.Second,
		HealthCheckTime: 10 * time.Second,
		MaxIdleTime:     60 * time.Second,
		TaskBatchSize:   1,
	}

	executor := NewAsyncExecutor(messageCount*2, config, asyncDispatcher, testLogger)
	require.NotNil(t, executor)

	err := executor.Start()
	require.NoError(t, err)
	defer func() {
		err := executor.Stop(10 * time.Second)
		assert.NoError(t, err)
	}()

	// Create messages and handles
	handles := make([]AsyncHandle, 0, messageCount)
	callbacks := NewCallbackRegistry(testLogger)

	asyncStartTime := time.Now()

	// Enqueue all messages (should be fast)
	for i := 0; i < messageCount; i++ {
		msg := &message.Message{
			ID:      "async-msg-" + string(rune(i)),
			Title:   "Async Test",
			Body:    "Testing async processing",
			Format:  message.FormatText,
			Targets: []target.Target{{Type: "test", Value: "async-target"}},
		}

		handle := NewAsyncHandle(msg, callbacks)
		require.NotNil(t, handle)

		err := executor.GetQueue().Enqueue(context.Background(), msg, handle)
		require.NoError(t, err)

		handles = append(handles, handle)
	}

	enqueueTime := time.Since(asyncStartTime)
	t.Logf("Async enqueue time for %d messages: %v", messageCount, enqueueTime)

	// Wait for all to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, handle := range handles {
		_, err := handle.Wait(ctx)
		require.NoError(t, err)
	}

	asyncDuration := time.Since(asyncStartTime)
	t.Logf("Async total time for %d messages: %v", messageCount, asyncDuration)

	// Verify async is faster than sync
	expectedSyncTime := time.Duration(messageCount) * processingDelay
	t.Logf("Expected sync time: %v, Actual sync time: %v", expectedSyncTime, syncDuration)
	t.Logf("Async improvement: %v faster than sync", syncDuration-asyncDuration)

	// Async should be significantly faster due to parallel processing
	assert.Less(t, asyncDuration, syncDuration*2/3, "Async should be significantly faster than sync")

	// Enqueuing should be very fast
	assert.Less(t, enqueueTime, processingDelay, "Enqueuing should be much faster than processing")
}
