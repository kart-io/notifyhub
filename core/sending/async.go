package sending

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/logger"
)

// AsyncSender handles asynchronous message sending
// This implements the proposal's asynchronous sending pattern
type AsyncSender struct {
	queue      Queue
	transports map[string]Transport
	workers    int
	logger     logger.Interface
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// Queue interface for message queuing
type Queue interface {
	Enqueue(ctx context.Context, item QueueItem) error
	Dequeue(ctx context.Context) (QueueItem, error)
	Size() int
	Close() error
}

// QueueItem represents an item in the queue
type QueueItem struct {
	Message   *message.Message
	Targets   []Target
	Priority  int
	Timestamp time.Time
	Retries   int
}

// NewAsyncSender creates a new asynchronous sender
func NewAsyncSender(queue Queue, logger logger.Interface, workers int) *AsyncSender {
	if workers <= 0 {
		workers = 4
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &AsyncSender{
		queue:      queue,
		transports: make(map[string]Transport),
		workers:    workers,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// RegisterTransport registers a transport for a platform
func (a *AsyncSender) RegisterTransport(transport Transport) error {
	name := transport.Name()
	if _, exists := a.transports[name]; exists {
		return fmt.Errorf("transport %s already registered", name)
	}
	a.transports[name] = transport
	return nil
}

// Start starts the async sender workers
func (a *AsyncSender) Start() error {
	a.logger.Info(context.Background(), "Starting async sender", "workers", a.workers)

	for i := 0; i < a.workers; i++ {
		a.wg.Add(1)
		go a.worker(i)
	}

	return nil
}

// Stop stops the async sender gracefully
func (a *AsyncSender) Stop(timeout time.Duration) error {
	a.logger.Info(context.Background(), "Stopping async sender")

	// Cancel context to signal workers to stop
	a.cancel()

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Info(context.Background(), "Async sender stopped gracefully")
	case <-time.After(timeout):
		return fmt.Errorf("async sender stop timeout")
	}

	// Close the queue
	if err := a.queue.Close(); err != nil {
		return fmt.Errorf("failed to close queue: %w", err)
	}

	return nil
}

// SendAsync queues a message for asynchronous sending
func (a *AsyncSender) SendAsync(ctx context.Context, msg *message.Message, targets []Target) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("message is nil")
	}
	if len(targets) == 0 {
		return "", fmt.Errorf("no targets provided")
	}

	// Create queue item
	item := QueueItem{
		Message:   msg,
		Targets:   targets,
		Priority:  int(msg.Priority),
		Timestamp: time.Now(),
		Retries:   0,
	}

	// Enqueue the item
	if err := a.queue.Enqueue(ctx, item); err != nil {
		return "", fmt.Errorf("failed to enqueue message: %w", err)
	}

	a.logger.Info(ctx, "Message queued", "message_id", msg.ID, "target_count", len(targets), "queue_size", a.queue.Size())

	return msg.ID, nil
}

// worker processes queued messages
func (a *AsyncSender) worker(id int) {
	defer a.wg.Done()

	a.logger.Debug(context.Background(), "Worker started", "worker_id", id)

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Debug(context.Background(), "Worker stopping", "worker_id", id)
			return
		default:
			a.processQueueItem(id)
		}
	}
}

// processQueueItem processes a single item from the queue
func (a *AsyncSender) processQueueItem(workerID int) {
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	// Dequeue an item
	item, err := a.queue.Dequeue(ctx)
	if err != nil {
		// Queue might be empty or closed
		time.Sleep(100 * time.Millisecond)
		return
	}

	a.logger.Debug(ctx, "Processing queue item", "worker_id", workerID, "message_id", item.Message.ID, "targets", len(item.Targets))

	// Process each target
	for _, target := range item.Targets {
		a.sendToTarget(ctx, item.Message, target)
	}
}

// sendToTarget sends a message to a single target
func (a *AsyncSender) sendToTarget(ctx context.Context, msg *message.Message, target Target) {
	// Get transport for target platform
	transport, exists := a.transports[target.GetPlatform()]
	if !exists {
		a.logger.Error(ctx, "Transport not found", "platform", target.GetPlatform(), "target", target.GetValue())
		return
	}

	// Send through transport
	_, err := transport.Send(ctx, msg, target)
	if err != nil {
		a.logger.Error(ctx, "Async send failed", "platform", target.Platform, "target", target.Value, "message", msg.ID, "error", err)
	} else {
		a.logger.Info(ctx, "Async message sent", "platform", target.Platform, "target", target.Value, "message", msg.ID)
	}
}

// GetQueueSize returns the current queue size
func (a *AsyncSender) GetQueueSize() int {
	return a.queue.Size()
}
