package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

func TestMemoryQueue(t *testing.T) {
	t.Run("Basic Operations", func(t *testing.T) {
		q := NewMemoryQueue(10, logger.Discard)
		defer func() {
			if err := q.Close(); err != nil {
				t.Errorf("Failed to close queue: %v", err)
			}
		}()

		ctx := context.Background()

		// Test empty queue
		if !q.IsEmpty() {
			t.Error("Queue should be empty")
		}
		if q.Size() != 0 {
			t.Errorf("Expected size 0, got %d", q.Size())
		}

		// Test enqueue
		msg := &Message{
			ID:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
			Payload: "test payload",
		}
		err := q.Enqueue(ctx, msg)
		if err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}

		if q.IsEmpty() {
			t.Error("Queue should not be empty")
		}
		if q.Size() != 1 {
			t.Errorf("Expected size 1, got %d", q.Size())
		}

		// Test dequeue
		dequeued, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Failed to dequeue: %v", err)
		}
		if dequeued.ID != msg.ID {
			t.Errorf("Expected ID %s, got %s", msg.ID, dequeued.ID)
		}

		if !q.IsEmpty() {
			t.Error("Queue should be empty after dequeue")
		}
	})

	t.Run("Capacity Limit", func(t *testing.T) {
		q := NewMemoryQueue(2, logger.Discard)
		defer q.Close()

		ctx := context.Background()

		// Fill queue to capacity
		err := q.Enqueue(ctx, &Message{ID: "1", Payload: "msg1"})
		if err != nil {
			t.Fatal(err)
		}
		err = q.Enqueue(ctx, &Message{ID: "2", Payload: "msg2"})
		if err != nil {
			t.Fatal(err)
		}

		// Try to exceed capacity
		err = q.Enqueue(ctx, &Message{ID: "3", Payload: "msg3"})
		if err != ErrQueueFull {
			t.Errorf("Expected ErrQueueFull, got %v", err)
		}
	})

	t.Run("Priority Queue", func(t *testing.T) {
		q := NewMemoryQueue(10, logger.Discard)
		defer q.Close()

		ctx := context.Background()

		// Enqueue messages with different priorities
		msg1 := &Message{ID: "1", Priority: PriorityLow, Payload: "low"}
		msg2 := &Message{ID: "2", Priority: PriorityHigh, Payload: "high"}
		msg3 := &Message{ID: "3", Priority: PriorityNormal, Payload: "normal"}
		msg4 := &Message{ID: "4", Priority: PriorityUrgent, Payload: "urgent"}

		if err := q.Enqueue(ctx, msg1); err != nil {
			t.Fatal(err)
		}
		if err := q.Enqueue(ctx, msg2); err != nil {
			t.Fatal(err)
		}
		if err := q.Enqueue(ctx, msg3); err != nil {
			t.Fatal(err)
		}
		if err := q.Enqueue(ctx, msg4); err != nil {
			t.Fatal(err)
		}

		// Dequeue should return highest priority first
		dequeued, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if dequeued.Priority != PriorityUrgent {
			t.Errorf("Expected priority %d, got %d", PriorityUrgent, dequeued.Priority)
		}

		dequeued, err = q.Dequeue(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if dequeued.Priority != PriorityHigh {
			t.Errorf("Expected priority %d, got %d", PriorityHigh, dequeued.Priority)
		}
	})

	t.Run("Delayed Messages", func(t *testing.T) {
		q := NewMemoryQueue(10, logger.Discard)
		defer q.Close()

		ctx := context.Background()

		// Enqueue delayed message
		futureTime := time.Now().Add(200 * time.Millisecond)
		msg := &Message{
			ID:          "delayed",
			Payload:     "delayed message",
			ScheduledAt: &futureTime,
		}

		err := q.Enqueue(ctx, msg)
		if err != nil {
			t.Fatal(err)
		}

		// Should not be available immediately
		_, err = q.Dequeue(ctx)
		if err != ErrQueueEmpty {
			t.Error("Expected ErrQueueEmpty for delayed message")
		}

		// Wait for message to become ready
		time.Sleep(300 * time.Millisecond)

		// Now should be available
		dequeued, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if dequeued.ID != msg.ID {
			t.Errorf("Expected ID %s, got %s", msg.ID, dequeued.ID)
		}
	})

	t.Run("Concurrent Operations", func(t *testing.T) {
		q := NewMemoryQueue(100, logger.Discard)
		defer q.Close()

		ctx := context.Background()
		var wg sync.WaitGroup
		messageCount := 50

		// Concurrent enqueue
		for i := 0; i < messageCount; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				msg := &Message{
					ID:      fmt.Sprintf("msg-%d-%d", id, time.Now().UnixNano()),
					Payload: id,
				}
				q.Enqueue(ctx, msg)
			}(i)
		}

		wg.Wait()
		if q.Size() != messageCount {
			t.Errorf("Expected size %d, got %d", messageCount, q.Size())
		}

		// Concurrent dequeue
		var dequeuedCount int32
		for i := 0; i < messageCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := q.Dequeue(ctx); err == nil {
					atomic.AddInt32(&dequeuedCount, 1)
				}
			}()
		}

		wg.Wait()
		if int(dequeuedCount) != messageCount {
			t.Errorf("Expected dequeued count %d, got %d", messageCount, dequeuedCount)
		}
		if !q.IsEmpty() {
			t.Error("Queue should be empty after dequeueing all messages")
		}
	})

	t.Run("Clear Operation", func(t *testing.T) {
		q := NewMemoryQueue(10, logger.Discard)
		defer q.Close()

		ctx := context.Background()

		// Add some messages
		for i := 0; i < 5; i++ {
			q.Enqueue(ctx, &Message{ID: string(rune('a' + i)), Payload: i})
		}
		if q.Size() != 5 {
			t.Errorf("Expected size 5, got %d", q.Size())
		}

		// Clear the queue
		err := q.Clear()
		if err != nil {
			t.Fatal(err)
		}
		if !q.IsEmpty() {
			t.Error("Queue should be empty after clear")
		}
	})
}

func TestRetryQueue(t *testing.T) {
	t.Run("Retry Logic", func(t *testing.T) {
		baseQueue := NewMemoryQueue(10, logger.Discard)
		retryPolicy := RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			Multiplier:      2.0,
		}
		rq := NewRetryQueue(baseQueue, retryPolicy, logger.Discard)
		defer rq.Close()

		ctx := context.Background()

		msg := &Message{
			ID:         "retry-test",
			Payload:    "test",
			RetryCount: 0,
			MaxRetries: 3,
		}

		// First retry
		err := rq.Retry(ctx, msg)
		if err != nil {
			t.Fatal(err)
		}
		if msg.RetryCount != 1 {
			t.Errorf("Expected retry count 1, got %d", msg.RetryCount)
		}
		if msg.ScheduledAt == nil {
			t.Error("ScheduledAt should be set")
		}

		// Max retries + 1 should move to DLQ
		msg.RetryCount = 3
		err = rq.Retry(ctx, msg)
		if err != nil {
			t.Fatal(err)
		}

		// Check dead letter queue
		dlq := rq.GetDeadLetterQueue()
		if dlq.Size() != 1 {
			t.Errorf("Expected DLQ size 1, got %d", dlq.Size())
		}
	})
}

func TestWorkerPool(t *testing.T) {
	t.Run("Basic Worker Pool", func(t *testing.T) {
		q := NewMemoryQueue(100, logger.Discard)
		defer q.Close()

		var processedCount int32
		handler := func(ctx context.Context, msg *Message) error {
			atomic.AddInt32(&processedCount, 1)
			return nil
		}

		pool := NewWorkerPool(q, handler, 2, 4, logger.Discard)

		ctx := context.Background()
		err := pool.Start(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Add messages
		for i := 0; i < 10; i++ {
			q.Enqueue(ctx, &Message{ID: string(rune('a' + i)), Payload: i})
		}

		// Wait for processing
		time.Sleep(500 * time.Millisecond)

		// Stop pool
		err = pool.Stop()
		if err != nil {
			t.Fatal(err)
		}

		if int(processedCount) != 10 {
			t.Errorf("Expected processed count 10, got %d", processedCount)
		}
	})

	t.Run("Worker Scaling", func(t *testing.T) {
		q := NewMemoryQueue(100, logger.Discard)
		defer q.Close()

		handler := func(ctx context.Context, msg *Message) error {
			time.Sleep(10 * time.Millisecond) // Simulate work
			return nil
		}

		pool := NewWorkerPool(q, handler, 1, 5, logger.Discard)

		ctx := context.Background()
		err := pool.Start(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Stop()

		// Initial worker count
		if pool.GetWorkerCount() != 1 {
			t.Errorf("Expected initial worker count 1, got %d", pool.GetWorkerCount())
		}

		// Scale up
		err = pool.Scale(3)
		if err != nil {
			t.Fatal(err)
		}
		if pool.GetWorkerCount() != 3 {
			t.Errorf("Expected worker count 3 after scale up, got %d", pool.GetWorkerCount())
		}

		// Scale down
		err = pool.Scale(2)
		if err != nil {
			t.Fatal(err)
		}
		if pool.GetWorkerCount() != 2 {
			t.Errorf("Expected worker count 2 after scale down, got %d", pool.GetWorkerCount())
		}

		// Invalid scale (exceeds max)
		err = pool.Scale(10)
		if err == nil {
			t.Error("Expected error for invalid scale")
		}
	})
}

func TestQueueFactory(t *testing.T) {
	factory := NewQueueFactory(logger.Discard)

	t.Run("Create Memory Queue", func(t *testing.T) {
		config := &Config{
			Type:     "memory",
			Capacity: 100,
			Workers:  4,
		}

		q, err := factory.CreateQueue(config)
		if err != nil {
			t.Fatal(err)
		}
		if q == nil {
			t.Fatal("Queue should not be nil")
		}
		defer q.Close()

		// Should be a memory queue
		if _, ok := q.(*memoryQueue); !ok {
			t.Error("Expected memory queue type")
		}
	})

	t.Run("Create Queue with Retry", func(t *testing.T) {
		config := &Config{
			Type:     "memory",
			Capacity: 100,
			RetryPolicy: RetryPolicy{
				MaxRetries:      3,
				InitialInterval: 1 * time.Second,
			},
		}

		q, err := factory.CreateQueue(config)
		if err != nil {
			t.Fatal(err)
		}
		if q == nil {
			t.Fatal("Queue should not be nil")
		}
		defer q.Close()

		// Should be a retry queue
		if _, ok := q.(RetryQueue); !ok {
			t.Error("Expected retry queue type")
		}
	})

	t.Run("Unsupported Type", func(t *testing.T) {
		config := &Config{
			Type: "unsupported",
		}

		q, err := factory.CreateQueue(config)
		if err == nil {
			t.Error("Expected error for unsupported type")
		}
		if q != nil {
			t.Error("Queue should be nil for unsupported type")
		}
	})
}

// Benchmark tests
func BenchmarkMemoryQueueEnqueue(b *testing.B) {
	q := NewMemoryQueue(b.N, logger.Discard)
	defer q.Close()

	ctx := context.Background()
	msg := &Message{
		ID:      "bench",
		Payload: "benchmark payload",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Enqueue(ctx, msg)
	}
}

func BenchmarkMemoryQueueDequeue(b *testing.B) {
	q := NewMemoryQueue(b.N, logger.Discard)
	defer q.Close()

	ctx := context.Background()

	// Pre-fill queue
	for i := 0; i < b.N; i++ {
		q.Enqueue(ctx, &Message{ID: string(rune(i)), Payload: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Dequeue(ctx)
	}
}

func BenchmarkConcurrentOperations(b *testing.B) {
	q := NewMemoryQueue(b.N*2, logger.Discard)
	defer q.Close()

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			counter++
			msg := &Message{
				ID:      fmt.Sprintf("concurrent-%d-%d", counter, time.Now().UnixNano()),
				Payload: "concurrent",
			}
			q.Enqueue(ctx, msg)
			q.Dequeue(ctx)
		}
	})
}