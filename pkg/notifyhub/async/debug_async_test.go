// Debug test to understand async processing issues
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

// TestDebugAsyncProcessing helps debug the async processing flow
func TestDebugAsyncProcessing(t *testing.T) {
	testLogger := NewTestLogger()
	mockDispatcher := NewTrackingMockDispatcher()

	// Set a short delay for faster debugging
	mockDispatcher.SetProcessDelay(50 * time.Millisecond)

	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 1
	config.MaxWorkers = 1
	config.TaskBatchSize = 1 // Force immediate processing

	executor := NewAsyncExecutor(10, config, mockDispatcher, testLogger)
	require.NotNil(t, executor)

	// Check executor health before starting
	t.Logf("Executor healthy before start: %v", executor.IsHealthy())

	err := executor.Start()
	require.NoError(t, err)

	defer func() {
		err := executor.Stop(5 * time.Second)
		assert.NoError(t, err)
	}()

	// Check executor health after starting
	t.Logf("Executor healthy after start: %v", executor.IsHealthy())

	// Check worker pool stats
	stats := executor.GetStats()
	t.Logf("Executor stats: %+v", stats)

	// Create a test message
	msg := &message.Message{
		ID:      "debug-msg",
		Title:   "Debug Test",
		Body:    "Testing debug async processing",
		Format:  message.FormatText,
		Targets: []target.Target{{Type: "test", Value: "debug-target"}},
	}

	// Create callbacks
	callbacks := NewCallbackRegistry(testLogger)
	handle := NewAsyncHandle(msg, callbacks)
	require.NotNil(t, handle)

	// Check queue state before enqueue
	t.Logf("Queue size before enqueue: %d", executor.GetQueue().Size())
	t.Logf("Queue empty before enqueue: %v", executor.GetQueue().IsEmpty())

	// Enqueue the message
	err = executor.GetQueue().Enqueue(context.Background(), msg, handle)
	require.NoError(t, err)

	// Check queue state after enqueue
	t.Logf("Queue size after enqueue: %d", executor.GetQueue().Size())
	t.Logf("Queue empty after enqueue: %v", executor.GetQueue().IsEmpty())

	// Check handle status
	status := handle.AsyncStatus()
	t.Logf("Handle status after enqueue: %s", status.Status)

	// Wait a bit and check status progression
	time.Sleep(25 * time.Millisecond)
	status = handle.AsyncStatus()
	t.Logf("Handle status after 25ms: %s", status.Status)

	time.Sleep(50 * time.Millisecond)
	status = handle.AsyncStatus()
	t.Logf("Handle status after 75ms: %s", status.Status)

	time.Sleep(100 * time.Millisecond)
	status = handle.AsyncStatus()
	t.Logf("Handle status after 175ms: %s", status.Status)

	// Check if processing completed
	t.Logf("Dispatcher call count: %d", mockDispatcher.GetCallCount())
	t.Logf("Queue size after processing: %d", executor.GetQueue().Size())

	// Try to wait with a reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := handle.Wait(ctx)
	if err != nil {
		t.Logf("Wait failed with error: %v", err)
		// Check final status
		finalStatus := handle.AsyncStatus()
		t.Logf("Final handle status: %s", finalStatus.Status)
		t.Logf("Final queue size: %d", executor.GetQueue().Size())
		t.Logf("Final dispatcher call count: %d", mockDispatcher.GetCallCount())

		// Check worker pool stats again
		finalStats := executor.GetStats()
		t.Logf("Final executor stats: %+v", finalStats)

		t.Fatalf("Handle wait timed out")
	} else {
		t.Logf("Processing completed successfully")
		t.Logf("Result: %+v", result)
	}
}