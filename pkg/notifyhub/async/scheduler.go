// Package async provides message scheduling functionality for delayed queue processing
package async

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// MessageScheduler manages scheduled message processing
type MessageScheduler struct {
	delayedQueue *DelayedAsyncQueue
	logger       logger.Logger
	tickInterval time.Duration
}

// run executes the message scheduler
func (s *MessageScheduler) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	s.logger.Info("Message scheduler started", "tick_interval", s.tickInterval)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Message scheduler stopping")
			return

		case <-ticker.C:
			s.processScheduledMessages()
		}
	}
}

// processScheduledMessages moves ready messages from delayed heap to underlying queue
func (s *MessageScheduler) processScheduledMessages() {
	now := time.Now()
	processedCount := 0

	s.delayedQueue.mutex.Lock()
	defer s.delayedQueue.mutex.Unlock()

	// Process all ready messages
	for s.delayedQueue.heap.Len() > 0 {
		// Peek at the next item
		next := (*s.delayedQueue.heap)[0]

		if next.ScheduledAt.After(now) {
			// Next message isn't ready yet
			break
		}

		// Remove from heap
		item := heap.Pop(s.delayedQueue.heap).(*DelayedQueueItem)

		// Check if handle was cancelled
		if handleImpl, ok := item.Handle.(*HandleImpl); ok && handleImpl.IsCancelled() {
			s.logger.Debug("Skipping cancelled scheduled message", "message_id", item.Message.ID)
			continue
		}

		// Enqueue to underlying queue
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var err error

		if item.BatchInfo != nil {
			// This is part of a batch - we need to collect all batch items first
			// For simplicity, we'll enqueue individual items and let batch tracking handle coordination
			err = s.delayedQueue.underlying.Enqueue(ctx, item.Message, item.Handle)
		} else {
			err = s.delayedQueue.underlying.Enqueue(ctx, item.Message, item.Handle)
		}

		cancel()

		if err != nil {
			s.logger.Error("Failed to enqueue scheduled message",
				"message_id", item.Message.ID,
				"error", err)

			// Update handle with error
			if handleImpl, ok := item.Handle.(*HandleImpl); ok {
				handleImpl.UpdateStatus(StatusFailedOp, 1.0, nil, err)
			}
		} else {
			processedCount++
			s.logger.Debug("Scheduled message enqueued",
				"message_id", item.Message.ID,
				"scheduled_at", item.ScheduledAt,
				"delay_was", now.Sub(item.ScheduledAt))
		}
	}

	if processedCount > 0 {
		s.logger.Debug("Processed scheduled messages", "count", processedCount)
	}
}

// CancelScheduledMessage cancels a scheduled message by message ID
func (q *DelayedAsyncQueue) CancelScheduledMessage(messageID string) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Linear search through heap to find the message
	for i := 0; i < q.heap.Len(); i++ {
		if (*q.heap)[i].Message.ID == messageID {
			// Remove from heap
			item := heap.Remove(q.heap, i).(*DelayedQueueItem)

			// Cancel the handle
			_ = item.Handle.Cancel()

			q.logger.Debug("Cancelled scheduled message", "message_id", messageID)
			return true
		}
	}

	return false
}

// GetScheduledMessages returns information about all scheduled messages
func (q *DelayedAsyncQueue) GetScheduledMessages() []ScheduledMessageInfo {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	messages := make([]ScheduledMessageInfo, 0, q.heap.Len())

	for _, item := range *q.heap {
		messages = append(messages, ScheduledMessageInfo{
			MessageID:   item.Message.ID,
			ScheduledAt: item.ScheduledAt,
			Priority:    item.Priority,
			QueuedAt:    item.QueuedAt,
		})
	}

	return messages
}

// ScheduledMessageInfo represents information about a scheduled message
type ScheduledMessageInfo struct {
	MessageID   string    `json:"message_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Priority    int       `json:"priority"`
	QueuedAt    time.Time `json:"queued_at"`
}