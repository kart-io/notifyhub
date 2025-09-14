package scheduler

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/internal"
	"github.com/kart-io/notifyhub/queue/core"
)

// EnhancedQueue wraps a basic queue with scheduling capabilities
type EnhancedQueue struct {
	baseQueue core.Queue
	scheduler *MessageScheduler
}

// NewEnhancedQueue creates a queue with scheduling support
func NewEnhancedQueue(baseQueue core.Queue) *EnhancedQueue {
	eq := &EnhancedQueue{
		baseQueue: baseQueue,
	}
	eq.scheduler = NewMessageScheduler(eq.baseQueue)
	return eq
}

// Enqueue adds a message to the queue, supporting delayed messages
func (eq *EnhancedQueue) Enqueue(ctx context.Context, msg *core.Message) (string, error) {
	// Generate ID if not set
	if msg.ID == "" {
		msg.ID = internal.GenerateID()
	}

	// Set creation time if not set
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	// Check if message has delay
	if msg.Message != nil && msg.Message.Delay > 0 {
		// Use scheduler for delayed messages
		return msg.ID, eq.scheduler.ScheduleMessage(msg)
	}

	// Immediate processing for non-delayed messages
	return eq.baseQueue.Enqueue(ctx, msg)
}

// Dequeue retrieves a message from the queue
func (eq *EnhancedQueue) Dequeue(ctx context.Context) (*core.Message, error) {
	return eq.baseQueue.Dequeue(ctx)
}

// Ack acknowledges successful processing of a message
func (eq *EnhancedQueue) Ack(msgID string) error {
	return eq.baseQueue.Ack(msgID)
}

// Nack indicates failed processing and schedules retry
func (eq *EnhancedQueue) Nack(msgID string, nextRetry time.Time) error {
	return eq.baseQueue.Nack(msgID, nextRetry)
}

// Close closes the enhanced queue
func (eq *EnhancedQueue) Close() error {
	if eq.scheduler != nil {
		eq.scheduler.Stop()
	}
	return eq.baseQueue.Close()
}

// Size returns the current queue size
func (eq *EnhancedQueue) Size() int {
	return eq.baseQueue.Size()
}

// Health checks queue health
func (eq *EnhancedQueue) Health(ctx context.Context) error {
	return eq.baseQueue.Health(ctx)
}