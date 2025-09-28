package async

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger implements logger.Logger for testing
type mockLogger struct{}

func (m *mockLogger) LogMode(level logger.LogLevel) logger.Logger { return m }
func (m *mockLogger) Debug(msg string, args ...any) {}
func (m *mockLogger) Info(msg string, args ...any)  {}
func (m *mockLogger) Warn(msg string, args ...any)  {}
func (m *mockLogger) Error(msg string, args ...any) {}

func TestMemoryAsyncQueue_BasicOperations(t *testing.T) {
	logger := &mockLogger{}
	queue := NewMemoryAsyncQueue(10, logger)
	defer queue.Close()

	// Test initial state
	assert.True(t, queue.IsEmpty())
	assert.Equal(t, 0, queue.Size())

	// Create test message and handle
	msg := &message.Message{
		ID:       "test-1",
		Title:    "Test Message",
		Body:     "Test Body",
		Priority: message.PriorityNormal,
	}

	callbacks := NewCallbackRegistry(logger)
	handle := NewAsyncHandle(msg, callbacks)

	// Test enqueue
	ctx := context.Background()
	err := queue.Enqueue(ctx, msg, handle)
	require.NoError(t, err)

	assert.False(t, queue.IsEmpty())
	assert.Equal(t, 1, queue.Size())

	// Test dequeue
	item, err := queue.Dequeue(ctx)
	require.NoError(t, err)
	require.NotNil(t, item)

	assert.Equal(t, msg.ID, item.Message.ID)
	assert.Equal(t, int(msg.Priority), item.Priority)
	assert.True(t, queue.IsEmpty())
}

func TestMemoryAsyncQueue_PriorityOrdering(t *testing.T) {
	logger := &mockLogger{}
	queue := NewMemoryAsyncQueue(10, logger)
	defer queue.Close()

	callbacks := NewCallbackRegistry(logger)
	ctx := context.Background()

	// Create messages with different priorities
	messages := []*message.Message{
		{ID: "low", Priority: message.PriorityLow},
		{ID: "urgent", Priority: message.PriorityUrgent},
		{ID: "normal", Priority: message.PriorityNormal},
		{ID: "high", Priority: message.PriorityHigh},
	}

	// Enqueue in random order
	for _, msg := range messages {
		handle := NewAsyncHandle(msg, callbacks)
		err := queue.Enqueue(ctx, msg, handle)
		require.NoError(t, err)
	}

	// Dequeue should return in priority order (urgent, high, normal, low)
	expectedOrder := []string{"urgent", "high", "normal", "low"}

	for i, expectedID := range expectedOrder {
		item, err := queue.Dequeue(ctx)
		require.NoError(t, err, "Failed to dequeue item %d", i)
		assert.Equal(t, expectedID, item.Message.ID, "Wrong priority order at position %d", i)
	}
}

func TestMemoryAsyncQueue_BatchOperations(t *testing.T) {
	logger := &mockLogger{}
	queue := NewMemoryAsyncQueue(10, logger)
	defer queue.Close()

	callbacks := NewCallbackRegistry(logger)
	ctx := context.Background()

	// Create batch messages
	messages := []*message.Message{
		{ID: "batch-1", Priority: message.PriorityNormal},
		{ID: "batch-2", Priority: message.PriorityNormal},
		{ID: "batch-3", Priority: message.PriorityNormal},
	}

	batchHandle := NewAsyncBatchHandle("batch-123", messages, callbacks)

	// Test batch enqueue
	err := queue.EnqueueBatch(ctx, messages, batchHandle)
	require.NoError(t, err)

	assert.Equal(t, 3, queue.Size())

	// Test batch dequeue
	items, err := queue.(interface{ DequeueBatch(context.Context, int) ([]*QueueItem, error) }).DequeueBatch(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, 1, queue.Size())

	// Dequeue remaining
	items, err = queue.(interface{ DequeueBatch(context.Context, int) ([]*QueueItem, error) }).DequeueBatch(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.True(t, queue.IsEmpty())
}

func TestMemoryAsyncQueue_CapacityLimits(t *testing.T) {
	logger := &mockLogger{}
	queue := NewMemoryAsyncQueue(2, logger) // Small capacity
	defer queue.Close()

	callbacks := NewCallbackRegistry(logger)
	ctx := context.Background()

	// Fill queue to capacity
	for i := 0; i < 2; i++ {
		msg := &message.Message{ID: "msg-" + string(rune('1'+i)), Priority: message.PriorityNormal}
		handle := NewAsyncHandle(msg, callbacks)
		err := queue.Enqueue(ctx, msg, handle)
		require.NoError(t, err)
	}

	// Next enqueue should fail
	msg := &message.Message{ID: "overflow", Priority: message.PriorityNormal}
	handle := NewAsyncHandle(msg, callbacks)
	err := queue.Enqueue(ctx, msg, handle)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue is full")
}

func TestMemoryAsyncQueue_ConcurrentAccess(t *testing.T) {
	logger := &mockLogger{}
	queue := NewMemoryAsyncQueue(100, logger)
	defer queue.Close()

	callbacks := NewCallbackRegistry(logger)
	ctx := context.Background()

	const numProducers = 5
	const numConsumers = 3
	const messagesPerProducer = 10

	var wg sync.WaitGroup
	producedCount := int32(0)
	consumedCount := int32(0)

	// Start producers
	for p := 0; p < numProducers; p++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			for i := 0; i < messagesPerProducer; i++ {
				msg := &message.Message{
					ID:       fmt.Sprintf("producer-%d-msg-%d", producerID, i),
					Priority: message.PriorityNormal,
				}
				handle := NewAsyncHandle(msg, callbacks)
				err := queue.Enqueue(ctx, msg, handle)
				if err != nil {
					t.Errorf("Producer %d failed to enqueue: %v", producerID, err)
					return
				}
				atomic.AddInt32(&producedCount, 1)
			}
		}(p)
	}

	// Start consumers
	for c := 0; c < numConsumers; c++ {
		wg.Add(1)
		go func(consumerID int) {
			defer wg.Done()
			for {
				item, err := queue.Dequeue(ctx)
				if err != nil {
					if queue.IsEmpty() && atomic.LoadInt32(&producedCount) == numProducers*messagesPerProducer {
						return // All done
					}
					time.Sleep(10 * time.Millisecond)
					continue
				}
				if item != nil {
					atomic.AddInt32(&consumedCount, 1)
				}
			}
		}(c)
	}

	// Wait for all producers to finish
	wg.Wait()

	// Wait for all messages to be consumed
	for atomic.LoadInt32(&consumedCount) < numProducers*messagesPerProducer {
		time.Sleep(10 * time.Millisecond)
	}

	assert.Equal(t, int32(numProducers*messagesPerProducer), atomic.LoadInt32(&producedCount))
	assert.Equal(t, int32(numProducers*messagesPerProducer), atomic.LoadInt32(&consumedCount))
}

func TestMemoryAsyncQueue_Statistics(t *testing.T) {
	logger := &mockLogger{}
	queue := NewMemoryAsyncQueue(10, logger)
	defer queue.Close()

	callbacks := NewCallbackRegistry(logger)
	ctx := context.Background()

	// Enqueue some messages
	for i := 0; i < 3; i++ {
		msg := &message.Message{ID: fmt.Sprintf("msg-%d", i), Priority: message.PriorityNormal}
		handle := NewAsyncHandle(msg, callbacks)
		err := queue.Enqueue(ctx, msg, handle)
		require.NoError(t, err)
	}

	// Dequeue one message
	_, err := queue.Dequeue(ctx)
	require.NoError(t, err)

	// Check statistics
	stats := queue.(interface{ GetStatistics() QueueStatistics }).GetStatistics()
	assert.Equal(t, int64(3), stats.TotalEnqueued)
	assert.Equal(t, int64(1), stats.TotalDequeued)
	assert.Equal(t, 2, stats.CurrentSize)
	assert.Equal(t, 10, stats.MaxSize)
	assert.True(t, stats.Uptime > 0)

	// Check throughput metrics
	metrics := queue.(interface{ GetThroughputMetrics() ThroughputMetrics }).GetThroughputMetrics()
	assert.True(t, metrics.MessagesPerSecond >= 0)
	assert.Equal(t, 100.0, metrics.SuccessRate) // No errors yet
}

func TestMemoryAsyncQueue_PriorityOperations(t *testing.T) {
	logger := &mockLogger{}
	queue := NewMemoryAsyncQueue(10, logger)
	defer queue.Close()

	callbacks := NewCallbackRegistry(logger)
	ctx := context.Background()

	// Test custom priority enqueue
	msg := &message.Message{ID: "test", Priority: message.PriorityNormal}
	handle := NewAsyncHandle(msg, callbacks)

	priorityQueue := queue.(interface {
		EnqueueWithPriority(context.Context, *message.Message, AsyncHandle, int) error
	})

	err := priorityQueue.EnqueueWithPriority(ctx, msg, handle, 999) // Very high priority
	require.NoError(t, err)

	// Add normal priority message
	msg2 := &message.Message{ID: "normal", Priority: message.PriorityHigh}
	handle2 := NewAsyncHandle(msg2, callbacks)
	err = queue.Enqueue(ctx, msg2, handle2)
	require.NoError(t, err)

	// The custom priority message should come first
	item, err := queue.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test", item.Message.ID)
	assert.Equal(t, 999, item.Priority)

	// Check priority distribution
	priorityDist := queue.(interface{ GetQueueByPriority() map[int]int }).GetQueueByPriority()
	assert.Equal(t, 1, priorityDist[int(message.PriorityHigh)])
}

func TestMemoryAsyncQueue_HealthCheck(t *testing.T) {
	logger := &mockLogger{}
	queue := NewMemoryAsyncQueue(10, logger)

	health := queue.Health()
	assert.True(t, health.Healthy)
	assert.Equal(t, 0, health.Size)
	assert.Equal(t, 10, health.MaxSize)
	assert.Equal(t, int64(0), health.ProcessedCount)
	assert.Equal(t, int64(0), health.ErrorCount)

	// Close queue and check health
	queue.Close()
	health = queue.Health()
	assert.False(t, health.Healthy)
}

func TestDelayedAsyncQueue_ScheduledMessages(t *testing.T) {
	logger := &mockLogger{}
	memQueue := NewMemoryAsyncQueue(10, logger)
	delayedQueue := NewDelayedAsyncQueue(memQueue, logger)
	defer delayedQueue.Close()

	callbacks := NewCallbackRegistry(logger)
	ctx := context.Background()

	// Schedule a message for 100ms in the future
	msg := &message.Message{ID: "delayed", Priority: message.PriorityNormal}
	handle := NewAsyncHandle(msg, callbacks)
	scheduledAt := time.Now().Add(100 * time.Millisecond)

	err := delayedQueue.EnqueueDelayed(ctx, msg, handle, scheduledAt)
	require.NoError(t, err)

	// Message should not be immediately available
	assert.Equal(t, 0, memQueue.Size())
	assert.Equal(t, 1, delayedQueue.GetScheduledCount())

	// Wait for message to be scheduled
	time.Sleep(150 * time.Millisecond)

	// Now message should be available in underlying queue
	assert.Equal(t, 1, memQueue.Size())
	assert.Equal(t, 0, delayedQueue.GetScheduledCount())

	// Dequeue should work
	item, err := delayedQueue.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, "delayed", item.Message.ID)
}

func TestDelayedAsyncQueue_ImmediateMessages(t *testing.T) {
	logger := &mockLogger{}
	memQueue := NewMemoryAsyncQueue(10, logger)
	delayedQueue := NewDelayedAsyncQueue(memQueue, logger)
	defer delayedQueue.Close()

	callbacks := NewCallbackRegistry(logger)
	ctx := context.Background()

	// Schedule a message for the past (should be immediate)
	msg := &message.Message{ID: "immediate", Priority: message.PriorityNormal}
	handle := NewAsyncHandle(msg, callbacks)
	scheduledAt := time.Now().Add(-1 * time.Hour) // Past time

	err := delayedQueue.EnqueueDelayed(ctx, msg, handle, scheduledAt)
	require.NoError(t, err)

	// Message should be immediately available
	assert.Equal(t, 1, memQueue.Size())
	assert.Equal(t, 0, delayedQueue.GetScheduledCount())

	item, err := delayedQueue.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, "immediate", item.Message.ID)
}

func TestDelayedAsyncQueue_CancelScheduled(t *testing.T) {
	logger := &mockLogger{}
	memQueue := NewMemoryAsyncQueue(10, logger)
	delayedQueue := NewDelayedAsyncQueue(memQueue, logger)
	defer delayedQueue.Close()

	callbacks := NewCallbackRegistry(logger)
	ctx := context.Background()

	// Schedule a message for far future
	msg := &message.Message{ID: "cancel-me", Priority: message.PriorityNormal}
	handle := NewAsyncHandle(msg, callbacks)
	scheduledAt := time.Now().Add(1 * time.Hour)

	err := delayedQueue.EnqueueDelayed(ctx, msg, handle, scheduledAt)
	require.NoError(t, err)
	assert.Equal(t, 1, delayedQueue.GetScheduledCount())

	// Cancel the scheduled message
	cancelled := delayedQueue.CancelScheduledMessage("cancel-me")
	assert.True(t, cancelled)
	assert.Equal(t, 0, delayedQueue.GetScheduledCount())

	// Try to cancel non-existent message
	cancelled = delayedQueue.CancelScheduledMessage("not-exists")
	assert.False(t, cancelled)
}

func TestQueueFactory_CreateDifferentTypes(t *testing.T) {
	logger := &mockLogger{}
	factory := NewQueueFactory(logger)

	// Test memory queue creation
	config := map[string]interface{}{
		"max_size": 50,
	}

	queue, err := factory.CreateQueue("memory", config)
	require.NoError(t, err)
	assert.NotNil(t, queue)
	queue.Close()

	// Test delayed memory queue creation
	config["enable_delayed"] = true
	delayedQueue, err := factory.CreateQueue("memory", config)
	require.NoError(t, err)
	assert.NotNil(t, delayedQueue)
	delayedQueue.Close()

	// Test unknown queue type
	_, err = factory.CreateQueue("unknown", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown queue type")
}