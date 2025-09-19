package scheduler

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/queue/core"
)

// DelayedMessage represents a message scheduled for future delivery
type DelayedMessage struct {
	Message    *core.Message `json:"message"`
	ScheduleAt time.Time     `json:"schedule_at"`
	index      int           // for heap.Interface
}

// DelayedMessageHeap implements heap.Interface for DelayedMessage
type DelayedMessageHeap []*DelayedMessage

func (h DelayedMessageHeap) Len() int           { return len(h) }
func (h DelayedMessageHeap) Less(i, j int) bool { return h[i].ScheduleAt.Before(h[j].ScheduleAt) }
func (h DelayedMessageHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *DelayedMessageHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*DelayedMessage)
	item.index = n
	*h = append(*h, item)
}

func (h *DelayedMessageHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*h = old[0 : n-1]
	return item
}

// MessageScheduler manages delayed message scheduling using a min-heap
type MessageScheduler struct {
	heap   DelayedMessageHeap
	mu     sync.RWMutex
	queue  core.Queue
	stopCh chan struct{}
	ticker *time.Ticker
	ctx    context.Context
	cancel context.CancelFunc
}

// NewMessageScheduler creates a new message scheduler
func NewMessageScheduler(queue core.Queue) *MessageScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	scheduler := &MessageScheduler{
		heap:   make(DelayedMessageHeap, 0),
		queue:  queue,
		stopCh: make(chan struct{}),
		ticker: time.NewTicker(1 * time.Second), // Check every second
		ctx:    ctx,
		cancel: cancel,
	}

	heap.Init(&scheduler.heap)

	// Start the scheduling goroutine
	go scheduler.run()

	return scheduler
}

// ScheduleMessage schedules a message for future delivery
func (s *MessageScheduler) ScheduleMessage(msg *core.Message) error {
	if msg.Message.Delay <= 0 {
		// No delay, enqueue immediately
		_, err := s.queue.Enqueue(s.ctx, msg)
		return err
	}

	scheduleAt := time.Now().Add(msg.Message.Delay)
	delayedMsg := &DelayedMessage{
		Message:    msg,
		ScheduleAt: scheduleAt,
	}

	s.mu.Lock()
	heap.Push(&s.heap, delayedMsg)
	s.mu.Unlock()

	return nil
}

// run is the main scheduler loop
func (s *MessageScheduler) run() {
	defer s.ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-s.ticker.C:
			s.processScheduledMessages()
		}
	}
}

// processScheduledMessages processes messages that are ready to be sent
func (s *MessageScheduler) processScheduledMessages() {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Process all messages that are ready
	for s.heap.Len() > 0 {
		// Peek at the next message
		next := s.heap[0]

		if next.ScheduleAt.After(now) {
			// Next message is not ready yet
			break
		}

		// Remove and enqueue the message
		delayedMsg := heap.Pop(&s.heap).(*DelayedMessage)

		// Enqueue without blocking the scheduler
		go func(msg *core.Message) {
			if _, err := s.queue.Enqueue(s.ctx, msg); err != nil {
				// TODO: Log error or handle failure
				// Could implement a retry mechanism here
				_ = err // Explicitly acknowledge error handling is TODO
			}
		}(delayedMsg.Message)
	}
}

// Stop stops the message scheduler
func (s *MessageScheduler) Stop() {
	s.cancel()
	close(s.stopCh)

	// Process remaining messages immediately if needed
	s.processScheduledMessages()
}

// GetPendingCount returns the number of pending scheduled messages
func (s *MessageScheduler) GetPendingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.heap.Len()
}

// GetNextScheduleTime returns the time of the next scheduled message
func (s *MessageScheduler) GetNextScheduleTime() *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.heap.Len() == 0 {
		return nil
	}

	next := s.heap[0].ScheduleAt
	return &next
}
