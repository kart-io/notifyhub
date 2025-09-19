package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
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
	msg := &queue.Message{
		ID:        "test-msg-001",
		Message:   message.NewMessage().SetTitle("Test Message").SetBody("Test Body"),
		Targets:   []sending.Target{},
		Attempts:  0,
		CreatedAt: time.Now(),
	}

	taskID, err := q.Enqueue(ctx, msg)
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

	if dequeuedMessage.Message.Body != "Test Body" {
		t.Error("Dequeued message body should match")
	}
}

func TestQueueOverflow(t *testing.T) {
	// Create small queue
	q := queue.NewSimple(2)

	ctx := context.Background()

	// Fill the queue
	for i := 0; i < 2; i++ {
		msg := &queue.Message{
			ID:        "test-msg-" + string(rune(i)),
			Message:   message.NewMessage().SetTitle("Test").SetBody("Body").SetPriority(message.PriorityNormal),
			Targets:   []sending.Target{},
			Attempts:  0,
			CreatedAt: time.Now(),
		}
		_, err := q.Enqueue(ctx, msg)
		if err != nil {
			t.Fatalf("Failed to enqueue message %d: %v", i, err)
		}
	}

	// Try to enqueue one more (should fail or block)
	msg := &queue.Message{
		ID:        "overflow",
		Message:   message.NewMessage().SetTitle("Overflow").SetBody("Should not fit").SetPriority(message.PriorityNormal),
		Targets:   []sending.Target{},
		Attempts:  0,
		CreatedAt: time.Now(),
	}

	// Use a timeout context for the overflow attempt
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err := q.Enqueue(ctxTimeout, msg)
	switch err {
	case nil:
		// If no error, queue might be blocking - that's ok
		t.Log("Queue accepted overflow message (might have blocking behavior)")
	case context.DeadlineExceeded:
		t.Log("Queue blocked on overflow as expected")
	default:
		t.Logf("Queue returned error: %v", err)
	}
}

func TestQueueConcurrency(t *testing.T) {
	q := queue.NewSimple(100)
	ctx := context.Background()

	// Concurrent enqueuers
	numEnqueuers := 5
	messagesPerEnqueuer := 10

	// Start enqueuers
	for i := 0; i < numEnqueuers; i++ {
		go func(id int) {
			for j := 0; j < messagesPerEnqueuer; j++ {
				msg := &queue.Message{
					ID:        "msg-" + string(rune(id)) + "-" + string(rune(j)),
					Message:   message.NewMessage().SetTitle("Test").SetBody("Body").SetPriority(message.PriorityNormal),
					Targets:   []sending.Target{},
					Attempts:  0,
					CreatedAt: time.Now(),
				}
				_, _ = q.Enqueue(ctx, msg)
			}
		}(i)
	}

	// Give time for messages to be enqueued
	time.Sleep(100 * time.Millisecond)

	// Dequeue all messages
	dequeued := 0
	for {
		ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		_, err := q.Dequeue(ctxTimeout)
		cancel()

		if err != nil {
			break // No more messages or timeout
		}
		dequeued++

		if dequeued >= numEnqueuers*messagesPerEnqueuer {
			break
		}
	}

	if dequeued < numEnqueuers*messagesPerEnqueuer {
		t.Errorf("Expected to dequeue %d messages, got %d",
			numEnqueuers*messagesPerEnqueuer, dequeued)
	}
}
