package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/internal"
)

// SimpleQueue is a basic in-memory queue implementation
type SimpleQueue struct {
	messages map[string]*Message
	pending  chan *Message
	mu       sync.RWMutex
	closed   bool
}

// NewSimple creates a new in-memory queue
func NewSimple(bufferSize int) *SimpleQueue {
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	q := &SimpleQueue{
		messages: make(map[string]*Message),
		pending:  make(chan *Message, bufferSize),
	}

	return q
}

// Enqueue adds a message to the queue
func (q *SimpleQueue) Enqueue(ctx context.Context, msg *Message) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return "", fmt.Errorf("queue closed")
	}

	// Generate ID if not set
	if msg.ID == "" {
		msg.ID = internal.GenerateID()
	}

	// Set creation time if not set
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	// Note: Delayed messages are now handled at a higher level
	// SimpleQueue only handles immediate enqueuing

	// Immediate processing for non-delayed messages
	q.messages[msg.ID] = msg
	select {
	case q.pending <- msg:
		return msg.ID, nil
	default:
		delete(q.messages, msg.ID)
		return "", fmt.Errorf("queue full")
	}
}

// Dequeue retrieves a message from the queue
func (q *SimpleQueue) Dequeue(ctx context.Context) (*Message, error) {
	select {
	case msg := <-q.pending:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Ack acknowledges successful processing of a message
func (q *SimpleQueue) Ack(msgID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.messages, msgID)
	return nil
}

// Nack indicates failed processing and schedules retry
func (q *SimpleQueue) Nack(msgID string, nextRetry time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if msg, exists := q.messages[msgID]; exists {
		msg.NextRetry = nextRetry
		msg.Attempts++
	}
	return nil
}

// Close closes the queue
func (q *SimpleQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.closed = true
		close(q.pending)
	}
	return nil
}

// Size returns the current queue size
func (q *SimpleQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.messages)
}

// Health checks queue health
func (q *SimpleQueue) Health(ctx context.Context) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}
	return nil
}
