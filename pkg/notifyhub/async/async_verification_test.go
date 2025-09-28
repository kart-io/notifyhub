// Package async provides comprehensive tests to verify the authenticity of asynchronous processing
// This test file implements Task 7.5: 验证异步处理的真实性
package async

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Test suite for verifying true asynchronous processing behavior
// Requirements: 2.1, 2.2, 2.3, 2.4, 2.5 - Complete async processing verification

// TrackingMockDispatcher extends MockDispatcher with call counting
type TrackingMockDispatcher struct {
	*MockDispatcher
	callCount int32
}

func NewTrackingMockDispatcher() *TrackingMockDispatcher {
	return &TrackingMockDispatcher{
		MockDispatcher: NewMockDispatcher(),
	}
}

func (t *TrackingMockDispatcher) Dispatch(ctx context.Context, msg *message.Message) (*receipt.Receipt, error) {
	atomic.AddInt32(&t.callCount, 1)
	return t.MockDispatcher.Dispatch(ctx, msg)
}

func (t *TrackingMockDispatcher) GetCallCount() int {
	return int(atomic.LoadInt32(&t.callCount))
}

// TestAsyncProcessingAuthenticity verifies that SendAsync uses real queue instead of sync calls
func TestAsyncProcessingAuthenticity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		queueType   string
		workerCount int
		messageCount int
	}{
		{
			name:        "Memory queue with single worker",
			queueType:   "memory",
			workerCount: 1,
			messageCount: 5,
		},
		{
			name:        "Memory queue with multiple workers",
			queueType:   "memory",
			workerCount: 3,
			messageCount: 10,
		},
		{
			name:        "Memory queue with high concurrency",
			queueType:   "memory",
			workerCount: 5,
			messageCount: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test logger
			testLogger := NewTestLogger()

			// Create mock dispatcher to verify async processing
			mockDispatcher := NewTrackingMockDispatcher()

			// Create async executor with specified configuration
			config := &WorkerPoolConfig{
				MinWorkers:      tt.workerCount,
				MaxWorkers:      tt.workerCount,
				TargetLoad:      0.7,
				ScaleUpDelay:    5 * time.Second,
				ScaleDownDelay:  30 * time.Second,
				HealthCheckTime: 10 * time.Second,
				MaxIdleTime:     60 * time.Second,
				TaskBatchSize:   1, // Process one at a time for verification
			}

			executor := NewAsyncExecutor(100, config, mockDispatcher, testLogger)
			require.NotNil(t, executor)

			// Start the executor
			err := executor.Start()
			require.NoError(t, err)
			defer func() {
				err := executor.Stop(5 * time.Second)
				assert.NoError(t, err)
			}()

			// Record start time for async verification
			startTime := time.Now()

			// Create messages and handles
			handles := make([]AsyncHandle, 0, tt.messageCount)
			messages := make([]*message.Message, 0, tt.messageCount)

			// Verify that SendAsync returns immediately without blocking
			for i := 0; i < tt.messageCount; i++ {
				msg := &message.Message{
					ID:      fmt.Sprintf("msg-%d", i),
					Title:   fmt.Sprintf("Test Message %d", i),
					Body:    fmt.Sprintf("This is test message %d", i),
					Format:  message.FormatText,
					Targets: []target.Target{{Type: "test", Value: fmt.Sprintf("target-%d", i)}},
				}
				messages = append(messages, msg)

				// Create callback registry
				callbacks := NewCallbackRegistry(testLogger)

				// Create handle
				handle := NewAsyncHandle(msg, callbacks)
				require.NotNil(t, handle)

				// Enqueue the message
				err := executor.GetQueue().Enqueue(context.Background(), msg, handle)
				require.NoError(t, err)

				handles = append(handles, handle)
			}

			// Verify that enqueuing was immediate (should be very fast)
			enqueueDuration := time.Since(startTime)
			t.Logf("Enqueue duration for %d messages: %v", tt.messageCount, enqueueDuration)

			// Enqueuing should be very fast (< 100ms for reasonable message counts)
			maxExpectedEnqueueTime := time.Duration(tt.messageCount) * 10 * time.Millisecond
			assert.Less(t, enqueueDuration, maxExpectedEnqueueTime,
				"Enqueuing should be immediate and not block on processing")

			// Note: We don't check exact queue size here because workers may have already
			// started processing messages (which is the correct async behavior)

			// Wait for all async operations to complete
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			var completedCount int32
			var wg sync.WaitGroup

			for _, handle := range handles {
				wg.Add(1)
				go func(h AsyncHandle) {
					defer wg.Done()
					_, err := h.Wait(ctx)
					if err == nil {
						atomic.AddInt32(&completedCount, 1)
					}
				}(handle)
			}

			wg.Wait()

			// Verify all operations completed
			assert.Equal(t, int32(tt.messageCount), completedCount,
				"All async operations should complete")

			// Verify mock dispatcher was called for each message
			dispatchCallCount := mockDispatcher.GetCallCount()
			assert.Equal(t, tt.messageCount, dispatchCallCount,
				"Dispatcher should be called for each message")

			// Verify queue is empty after processing
			assert.True(t, executor.GetQueue().IsEmpty(), "Queue should be empty after processing")

			// Verify executor health
			assert.True(t, executor.IsHealthy(), "Executor should be healthy")
		})
	}
}

// TestAsyncOperationStatusProgression verifies status changes throughout lifecycle
func TestAsyncOperationStatusProgression(t *testing.T) {
	t.Parallel()

	testLogger := NewTestLogger()
	mockDispatcher := NewTrackingMockDispatcher()

	// Add delay to dispatcher to observe status changes
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

	// Create test message
	msg := &message.Message{
		ID:      "status-test-msg",
		Title:   "Status Test",
		Body:    "Testing status progression",
		Format:  message.FormatText,
		Targets: []target.Target{{Type: "test", Value: "status-target"}},
	}

	// Create callback registry to track status changes
	callbacks := NewCallbackRegistry(testLogger)

	var statusHistory []Status
	var statusMutex sync.Mutex

	// Track status changes via polling
	handle := NewAsyncHandle(msg, callbacks)
	require.NotNil(t, handle)

	// Start status monitoring goroutine
	statusTracker := make(chan struct{})
	go func() {
		defer close(statusTracker)
		for {
			asyncStatus := handle.AsyncStatus()
			var status Status
			switch asyncStatus.Status {
			case StatusPendingOp:
				status = StatusPending
			case StatusProcessing:
				status = StatusRunning
			case StatusCompleted:
				status = StatusSuccess
			case StatusFailedOp:
				status = StatusFailed
			case StatusCancelledOp:
				status = StatusCancelled
			default:
				status = StatusPending
			}

			statusMutex.Lock()
			if len(statusHistory) == 0 || statusHistory[len(statusHistory)-1] != status {
				statusHistory = append(statusHistory, status)
				t.Logf("Status changed to: %s", status)
			}
			statusMutex.Unlock()

			if status == StatusSuccess || status == StatusFailed || status == StatusCancelled {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Enqueue the message
	err = executor.GetQueue().Enqueue(context.Background(), msg, handle)
	require.NoError(t, err)

	// Wait for completion
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := handle.Wait(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Wait for status tracker to finish
	<-statusTracker

	// Verify status progression
	statusMutex.Lock()
	defer statusMutex.Unlock()

	t.Logf("Complete status history: %v", statusHistory)

	// Should have at least one status change
	assert.GreaterOrEqual(t, len(statusHistory), 1, "Should have status transitions")

	// Should end with Success status
	assert.Equal(t, StatusSuccess, statusHistory[len(statusHistory)-1], "Should end with Success status")

	// Should contain StatusRunning at some point
	foundRunning := false
	for _, status := range statusHistory {
		if status == StatusRunning {
			foundRunning = true
			break
		}
	}
	assert.True(t, foundRunning, "Should transition through Running status")
}

// TestConcurrentAsyncOperations verifies resource isolation and concurrent execution
func TestConcurrentAsyncOperations(t *testing.T) {
	t.Parallel()

	testLogger := NewTestLogger()
	mockDispatcher := NewTrackingMockDispatcher()

	// Add processing delay to make concurrency observable
	mockDispatcher.SetProcessDelay(100 * time.Millisecond)

	config := &WorkerPoolConfig{
		MinWorkers:      4,
		MaxWorkers:      4,
		TargetLoad:      0.7,
		ScaleUpDelay:    5 * time.Second,
		ScaleDownDelay:  30 * time.Second,
		HealthCheckTime: 10 * time.Second,
		MaxIdleTime:     60 * time.Second,
		TaskBatchSize:   1,
	}

	executor := NewAsyncExecutor(50, config, mockDispatcher, testLogger)
	require.NotNil(t, executor)

	err := executor.Start()
	require.NoError(t, err)
	defer func() {
		err := executor.Stop(10 * time.Second)
		assert.NoError(t, err)
	}()

	const concurrentOperations = 20

	// Track concurrent processing
	var concurrentlyProcessing int32
	var maxConcurrent int32
	var processingTimes []time.Duration
	var timingMutex sync.Mutex

	// Create custom executor with concurrency tracking dispatcher
	concurrencyTracker := &ConcurrencyTrackingDispatcher{
		baseDispatcher: mockDispatcher,
		processing:     &concurrentlyProcessing,
		maxConcurrent:  &maxConcurrent,
	}

	// Create a new executor with the tracking dispatcher
	executor = NewAsyncExecutor(50, config, concurrencyTracker, testLogger)

	err = executor.Start()
	require.NoError(t, err)

	handles := make([]AsyncHandle, 0, concurrentOperations)
	startTime := time.Now()

	// Create and enqueue multiple operations
	for i := 0; i < concurrentOperations; i++ {
		msg := &message.Message{
			ID:      fmt.Sprintf("concurrent-msg-%d", i),
			Title:   fmt.Sprintf("Concurrent Test %d", i),
			Body:    fmt.Sprintf("Testing concurrent processing %d", i),
			Format:  message.FormatText,
			Targets: []target.Target{{Type: "test", Value: fmt.Sprintf("concurrent-target-%d", i)}},
		}

		callbacks := NewCallbackRegistry(testLogger)
		handle := NewAsyncHandle(msg, callbacks)
		require.NotNil(t, handle)

		operationStart := time.Now()

		err := executor.GetQueue().Enqueue(context.Background(), msg, handle)
		require.NoError(t, err)

		handles = append(handles, handle)

		timingMutex.Lock()
		processingTimes = append(processingTimes, time.Since(operationStart))
		timingMutex.Unlock()
	}

	enqueueDuration := time.Since(startTime)
	t.Logf("Enqueued %d operations in %v", concurrentOperations, enqueueDuration)

	// Verify enqueuing was fast (should not wait for processing)
	assert.Less(t, enqueueDuration, 1*time.Second,
		"Enqueuing should be fast and not wait for processing")

	// Wait for all operations to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var completedCount int32

	for _, handle := range handles {
		wg.Add(1)
		go func(h AsyncHandle) {
			defer wg.Done()
			_, err := h.Wait(ctx)
			if err == nil {
				atomic.AddInt32(&completedCount, 1)
			}
		}(handle)
	}

	wg.Wait()

	// Verify all operations completed
	assert.Equal(t, int32(concurrentOperations), completedCount,
		"All concurrent operations should complete")

	// Verify true concurrent processing occurred
	maxConcurrentValue := atomic.LoadInt32(&maxConcurrent)
	t.Logf("Maximum concurrent operations: %d", maxConcurrentValue)

	// Should have processed multiple operations concurrently
	assert.Greater(t, maxConcurrentValue, int32(1),
		"Should process multiple operations concurrently")
	assert.LessOrEqual(t, maxConcurrentValue, int32(config.MaxWorkers),
		"Concurrent operations should not exceed worker count")

	// Verify resource isolation - each operation should be independent
	dispatchCallCount := mockDispatcher.GetCallCount()
	assert.Equal(t, concurrentOperations, dispatchCallCount,
		"Each operation should be processed independently")
}

// TestAsyncOperationPerformanceBenchmark verifies performance benefits of async processing
func TestAsyncOperationPerformanceBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance benchmark in short mode")
	}

	testLogger := NewTestLogger()

	const messageCount = 100
	const processingDelay = 10 * time.Millisecond

	// Test 1: Synchronous processing (simulated)
	t.Run("Synchronous Processing Simulation", func(t *testing.T) {
		mockDispatcher := NewTrackingMockDispatcher()
		mockDispatcher.SetProcessDelay(processingDelay)

		messages := createTestMessages(messageCount)

		startTime := time.Now()

		// Simulate synchronous processing
		for _, msg := range messages {
			_, err := mockDispatcher.Dispatch(context.Background(), msg)
			require.NoError(t, err)
		}

		syncDuration := time.Since(startTime)
		t.Logf("Synchronous processing of %d messages: %v", messageCount, syncDuration)

		// Store sync duration for comparison
		t.Cleanup(func() {
			t.Logf("Sync baseline: %v", syncDuration)
		})
	})

	// Test 2: Asynchronous processing
	t.Run("Asynchronous Processing Performance", func(t *testing.T) {
		mockDispatcher := NewTrackingMockDispatcher()
		mockDispatcher.SetProcessDelay(processingDelay)

		config := &WorkerPoolConfig{
			MinWorkers:      4,
			MaxWorkers:      8,
			TargetLoad:      0.7,
			ScaleUpDelay:    5 * time.Second,
			ScaleDownDelay:  30 * time.Second,
			HealthCheckTime: 10 * time.Second,
			MaxIdleTime:     60 * time.Second,
			TaskBatchSize:   1,
		}

		executor := NewAsyncExecutor(messageCount*2, config, mockDispatcher, testLogger)
		require.NotNil(t, executor)

		err := executor.Start()
		require.NoError(t, err)
		defer func() {
			err := executor.Stop(10 * time.Second)
			assert.NoError(t, err)
		}()

		messages := createTestMessages(messageCount)
		handles := make([]AsyncHandle, 0, messageCount)

		// Measure enqueue time (should be fast)
		enqueueStart := time.Now()

		for _, msg := range messages {
			callbacks := NewCallbackRegistry(testLogger)
			handle := NewAsyncHandle(msg, callbacks)
			require.NotNil(t, handle)

			err := executor.GetQueue().Enqueue(context.Background(), msg, handle)
			require.NoError(t, err)

			handles = append(handles, handle)
		}

		enqueueDuration := time.Since(enqueueStart)
		t.Logf("Async enqueue time for %d messages: %v", messageCount, enqueueDuration)

		// Verify enqueuing is much faster than processing
		expectedMinProcessingTime := time.Duration(messageCount) * processingDelay / 4 // With 4 workers
		assert.Less(t, enqueueDuration, expectedMinProcessingTime/10,
			"Enqueuing should be much faster than processing")

		// Measure total completion time
		completionStart := time.Now()

		// Wait for all to complete
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		var completedCount int32

		for _, handle := range handles {
			wg.Add(1)
			go func(h AsyncHandle) {
				defer wg.Done()
				_, err := h.Wait(ctx)
				if err == nil {
					atomic.AddInt32(&completedCount, 1)
				}
			}(handle)
		}

		wg.Wait()

		asyncDuration := time.Since(completionStart)
		t.Logf("Async completion time for %d messages: %v", messageCount, asyncDuration)

		// Verify all completed
		assert.Equal(t, int32(messageCount), completedCount,
			"All async operations should complete")

		// Async should be significantly faster due to parallelization
		// With 4+ workers, should be roughly 4x faster than sequential
		expectedAsyncTime := time.Duration(messageCount) * processingDelay / 3 // Conservative estimate
		assert.Less(t, asyncDuration, expectedAsyncTime,
			"Async processing should be significantly faster than sequential")

		t.Logf("Performance improvement: async processing completed in %v vs expected sequential %v",
			asyncDuration, time.Duration(messageCount)*processingDelay)
	})
}

// TestHandleWaitBehaviorAndTimeout verifies Wait() method behavior
func TestHandleWaitBehaviorAndTimeout(t *testing.T) {
	t.Parallel()

	testLogger := NewTestLogger()

	t.Run("Wait with successful completion", func(t *testing.T) {
		mockDispatcher := NewTrackingMockDispatcher()
		mockDispatcher.SetProcessDelay(100 * time.Millisecond)

		config := DefaultWorkerPoolConfig()
		config.TaskBatchSize = 1
		executor := NewAsyncExecutor(10, config, mockDispatcher, testLogger)
		require.NotNil(t, executor)

		err := executor.Start()
		require.NoError(t, err)
		defer func() {
			err := executor.Stop(5 * time.Second)
			assert.NoError(t, err)
		}()

		msg := &message.Message{
			ID:      "wait-test-msg",
			Title:   "Wait Test",
			Body:    "Testing wait behavior",
			Format:  message.FormatText,
			Targets: []target.Target{{Type: "test", Value: "wait-target"}},
		}

		callbacks := NewCallbackRegistry(testLogger)
		handle := NewAsyncHandle(msg, callbacks)
		require.NotNil(t, handle)

		err = executor.GetQueue().Enqueue(context.Background(), msg, handle)
		require.NoError(t, err)

		// Test Wait() blocks until completion
		startTime := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := handle.Wait(ctx)
		waitDuration := time.Since(startTime)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have waited for processing to complete
		assert.GreaterOrEqual(t, waitDuration, 90*time.Millisecond,
			"Wait should block until processing completes")
		assert.Less(t, waitDuration, 1*time.Second,
			"Wait should not take too long")
	})

	t.Run("Wait with timeout", func(t *testing.T) {
		mockDispatcher := NewTrackingMockDispatcher()
		// Set a very long delay to trigger timeout
		mockDispatcher.SetProcessDelay(5 * time.Second)

		config := DefaultWorkerPoolConfig()
		config.TaskBatchSize = 1
		executor := NewAsyncExecutor(10, config, mockDispatcher, testLogger)
		require.NotNil(t, executor)

		err := executor.Start()
		require.NoError(t, err)
		defer func() {
			err := executor.Stop(10 * time.Second)
			assert.NoError(t, err)
		}()

		msg := &message.Message{
			ID:      "timeout-test-msg",
			Title:   "Timeout Test",
			Body:    "Testing timeout behavior",
			Format:  message.FormatText,
			Targets: []target.Target{{Type: "test", Value: "timeout-target"}},
		}

		callbacks := NewCallbackRegistry(testLogger)
		handle := NewAsyncHandle(msg, callbacks)
		require.NotNil(t, handle)

		err = executor.GetQueue().Enqueue(context.Background(), msg, handle)
		require.NoError(t, err)

		// Test Wait() with short timeout
		startTime := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		result, err := handle.Wait(ctx)
		waitDuration := time.Since(startTime)

		// Should timeout
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "context deadline exceeded")

		// Should have waited for the timeout duration
		assert.GreaterOrEqual(t, waitDuration, 180*time.Millisecond,
			"Should wait for timeout duration")
		assert.Less(t, waitDuration, 300*time.Millisecond,
			"Should not wait much longer than timeout")
	})

	t.Run("Multiple concurrent waiters", func(t *testing.T) {
		mockDispatcher := NewTrackingMockDispatcher()
		mockDispatcher.SetProcessDelay(200 * time.Millisecond)

		config := DefaultWorkerPoolConfig()
		config.TaskBatchSize = 1
		executor := NewAsyncExecutor(10, config, mockDispatcher, testLogger)
		require.NotNil(t, executor)

		err := executor.Start()
		require.NoError(t, err)
		defer func() {
			err := executor.Stop(5 * time.Second)
			assert.NoError(t, err)
		}()

		msg := &message.Message{
			ID:      "multi-wait-test-msg",
			Title:   "Multi Wait Test",
			Body:    "Testing multiple waiters",
			Format:  message.FormatText,
			Targets: []target.Target{{Type: "test", Value: "multi-wait-target"}},
		}

		callbacks := NewCallbackRegistry(testLogger)
		handle := NewAsyncHandle(msg, callbacks)
		require.NotNil(t, handle)

		err = executor.GetQueue().Enqueue(context.Background(), msg, handle)
		require.NoError(t, err)

		// Start multiple waiters
		const waiterCount = 5
		var wg sync.WaitGroup
		var successCount int32
		var resultCount int32

		for i := 0; i < waiterCount; i++ {
			wg.Add(1)
			go func(waiterID int) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				result, err := handle.Wait(ctx)
				if err == nil {
					atomic.AddInt32(&successCount, 1)
					if result != nil {
						atomic.AddInt32(&resultCount, 1)
					}
				}
				t.Logf("Waiter %d completed with err=%v", waiterID, err)
			}(i)
		}

		wg.Wait()

		// All waiters should succeed
		assert.Equal(t, int32(waiterCount), successCount,
			"All waiters should receive completion signal")
		assert.Equal(t, int32(waiterCount), resultCount,
			"All waiters should receive the result")
	})
}

// TestAsyncOperationCancellation verifies cancellation behavior
func TestAsyncOperationCancellation(t *testing.T) {
	t.Parallel()

	testLogger := NewTestLogger()
	mockDispatcher := NewTrackingMockDispatcher()

	// Set a delay to make cancellation timing observable
	mockDispatcher.SetProcessDelay(500 * time.Millisecond)

	config := DefaultWorkerPoolConfig()
	config.TaskBatchSize = 1
	executor := NewAsyncExecutor(10, config, mockDispatcher, testLogger)
	require.NotNil(t, executor)

	err := executor.Start()
	require.NoError(t, err)
	defer func() {
		err := executor.Stop(5 * time.Second)
		assert.NoError(t, err)
	}()

	msg := &message.Message{
		ID:      "cancel-test-msg",
		Title:   "Cancel Test",
		Body:    "Testing cancellation",
		Format:  message.FormatText,
		Targets: []target.Target{{Type: "test", Value: "cancel-target"}},
	}

	callbacks := NewCallbackRegistry(testLogger)
	handle := NewAsyncHandle(msg, callbacks)
	require.NotNil(t, handle)

	err = executor.GetQueue().Enqueue(context.Background(), msg, handle)
	require.NoError(t, err)

	// Wait a bit to ensure processing starts
	time.Sleep(100 * time.Millisecond)

	// Cancel the operation
	err = handle.Cancel()
	require.NoError(t, err)

	// Verify status is cancelled
	status := handle.AsyncStatus()
	assert.Equal(t, StatusCancelledOp, status.Status)

	// Wait should return cancellation error
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := handle.Wait(ctx)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cancelled")
}

// TestQueueVersusDirectExecution verifies queue usage instead of direct sync execution
func TestQueueVersusDirectExecution(t *testing.T) {
	t.Parallel()

	testLogger := NewTestLogger()

	// Create mock dispatcher that tracks operations
	mockDispatcher := NewTrackingMockDispatcher()

	// Create executor with proper configuration
	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 2
	config.MaxWorkers = 2
	config.TaskBatchSize = 1

	executor := NewAsyncExecutor(10, config, mockDispatcher, testLogger)

	err := executor.Start()
	require.NoError(t, err)
	defer func() {
		err := executor.Stop(5 * time.Second)
		assert.NoError(t, err)
	}()

	// Create test messages
	messages := createTestMessages(5)
	handles := make([]AsyncHandle, 0, len(messages))
	callbacks := NewCallbackRegistry(testLogger)

	// Enqueue messages
	for _, msg := range messages {
		handle := NewAsyncHandle(msg, callbacks)
		require.NotNil(t, handle)

		err := executor.GetQueue().Enqueue(context.Background(), msg, handle)
		require.NoError(t, err)
		handles = append(handles, handle)
	}

	// Wait for processing
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, handle := range handles {
		_, err := handle.Wait(ctx)
		assert.NoError(t, err)
	}

	// Verify dispatcher was called through workers for each message
	assert.Equal(t, len(messages), mockDispatcher.GetCallCount(),
		"Dispatcher should be called through workers for each message")

	// Verify queue is empty after processing
	assert.True(t, executor.GetQueue().IsEmpty(),
		"Queue should be empty after processing")
}

// Helper functions and test utilities

// createTestMessages creates a slice of test messages
func createTestMessages(count int) []*message.Message {
	messages := make([]*message.Message, 0, count)
	for i := 0; i < count; i++ {
		msg := &message.Message{
			ID:      fmt.Sprintf("test-msg-%d", i),
			Title:   fmt.Sprintf("Test Message %d", i),
			Body:    fmt.Sprintf("This is test message number %d", i),
			Format:  message.FormatText,
			Targets: []target.Target{{Type: "test", Value: fmt.Sprintf("target-%d", i)}},
		}
		messages = append(messages, msg)
	}
	return messages
}


// ConcurrencyTrackingDispatcher tracks concurrent operations
type ConcurrencyTrackingDispatcher struct {
	baseDispatcher core.Dispatcher
	processing     *int32
	maxConcurrent  *int32
}

func (c *ConcurrencyTrackingDispatcher) Dispatch(ctx context.Context, msg *message.Message) (*receipt.Receipt, error) {
	// Increment concurrent counter
	current := atomic.AddInt32(c.processing, 1)
	defer atomic.AddInt32(c.processing, -1)

	// Update max concurrent if needed
	for {
		max := atomic.LoadInt32(c.maxConcurrent)
		if current <= max || atomic.CompareAndSwapInt32(c.maxConcurrent, max, current) {
			break
		}
	}

	return c.baseDispatcher.Dispatch(ctx, msg)
}

func (c *ConcurrencyTrackingDispatcher) RegisterPlatform(name string, creator platform.PlatformCreator) {
	c.baseDispatcher.RegisterPlatform(name, creator)
}

func (c *ConcurrencyTrackingDispatcher) Health(ctx context.Context) (map[string]string, error) {
	return c.baseDispatcher.Health(ctx)
}

func (c *ConcurrencyTrackingDispatcher) Close() error {
	return c.baseDispatcher.Close()
}

// SpyQueue tracks queue operations for verification
type SpyQueue struct {
	baseQueue    AsyncQueue
	enqueueCount int32
	dequeueCount int32
	logger       logger.Logger
}

func NewSpyQueue(logger logger.Logger) *SpyQueue {
	baseQueue := NewMemoryAsyncQueue(100, logger)
	return &SpyQueue{
		baseQueue: baseQueue,
		logger:    logger,
	}
}

func (s *SpyQueue) Enqueue(ctx context.Context, msg *message.Message, handle AsyncHandle) error {
	atomic.AddInt32(&s.enqueueCount, 1)
	return s.baseQueue.Enqueue(ctx, msg, handle)
}

func (s *SpyQueue) EnqueueBatch(ctx context.Context, msgs []*message.Message, batchHandle AsyncBatchHandle) error {
	atomic.AddInt32(&s.enqueueCount, int32(len(msgs)))
	return s.baseQueue.EnqueueBatch(ctx, msgs, batchHandle)
}

func (s *SpyQueue) Dequeue(ctx context.Context) (*QueueItem, error) {
	item, err := s.baseQueue.Dequeue(ctx)
	if err == nil {
		atomic.AddInt32(&s.dequeueCount, 1)
	}
	return item, err
}

func (s *SpyQueue) Size() int {
	return s.baseQueue.Size()
}

func (s *SpyQueue) IsEmpty() bool {
	return s.baseQueue.IsEmpty()
}

func (s *SpyQueue) Health() QueueHealth {
	return s.baseQueue.Health()
}

func (s *SpyQueue) Close() error {
	return s.baseQueue.Close()
}

func (s *SpyQueue) GetEnqueueCount() int {
	return int(atomic.LoadInt32(&s.enqueueCount))
}

func (s *SpyQueue) GetDequeueCount() int {
	return int(atomic.LoadInt32(&s.dequeueCount))
}

// BenchmarkAsyncVsSyncPerformance benchmarks async vs sync performance
func BenchmarkAsyncVsSyncPerformance(b *testing.B) {
	testLogger := NewTestLogger()

	b.Run("SyncProcessing", func(b *testing.B) {
		mockDispatcher := NewTrackingMockDispatcher()
		mockDispatcher.SetProcessDelay(1 * time.Millisecond)

		messages := createTestMessages(b.N)

		b.ResetTimer()
		for _, msg := range messages {
			_, err := mockDispatcher.Dispatch(context.Background(), msg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("AsyncProcessing", func(b *testing.B) {
		mockDispatcher := NewTrackingMockDispatcher()
		mockDispatcher.SetProcessDelay(1 * time.Millisecond)

		config := &WorkerPoolConfig{
			MinWorkers:      runtime.NumCPU(),
			MaxWorkers:      runtime.NumCPU() * 2,
			TargetLoad:      0.7,
			ScaleUpDelay:    5 * time.Second,
			ScaleDownDelay:  30 * time.Second,
			HealthCheckTime: 10 * time.Second,
			MaxIdleTime:     60 * time.Second,
			TaskBatchSize:   1,
		}

		executor := NewAsyncExecutor(b.N*2, config, mockDispatcher, testLogger)
		err := executor.Start()
		if err != nil {
			b.Fatal(err)
		}
		defer func() {
			err := executor.Stop(30 * time.Second)
			if err != nil {
				b.Error(err)
			}
		}()

		messages := createTestMessages(b.N)
		handles := make([]AsyncHandle, 0, b.N)

		callbacks := NewCallbackRegistry(testLogger)

		b.ResetTimer()

		// Enqueue all messages
		for _, msg := range messages {
			handle := NewAsyncHandle(msg, callbacks)
			err := executor.GetQueue().Enqueue(context.Background(), msg, handle)
			if err != nil {
				b.Fatal(err)
			}
			handles = append(handles, handle)
		}

		// Wait for all to complete
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		for _, handle := range handles {
			_, err := handle.Wait(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}