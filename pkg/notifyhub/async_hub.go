// Package notifyhub provides async hub implementation with queue support
package notifyhub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/queue"
)

// AsyncHub extends the Hub interface with asynchronous message processing
type AsyncHub interface {
	Hub

	// SendQueued enqueues a message for async processing
	SendQueued(ctx context.Context, message *Message) (*AsyncReceipt, error)

	// GetQueueStats returns queue statistics
	GetQueueStats() map[string]interface{}

	// GetQueueMetrics returns detailed queue metrics
	GetQueueMetrics() *queue.QueueMetrics

	// GetWorkerMetrics returns worker pool metrics
	GetWorkerMetrics() *queue.WorkerMetrics

	// GetHealthStatus returns health status
	GetHealthStatus() *queue.HealthStatus

	// ProcessQueuedMessages starts processing queued messages
	ProcessQueuedMessages(ctx context.Context) error

	// StopProcessing stops processing queued messages
	StopProcessing() error
}

// asyncHub implements AsyncHub with queue support
type asyncHub struct {
	Hub
	queue         queue.Queue
	workerPool    queue.WorkerPool
	healthChecker *queue.HealthChecker
	processing    bool
	mutex         sync.RWMutex
	logger        logger.Logger
}

// NewAsyncHub creates a new async hub with queue support
func NewAsyncHub(baseHub Hub, queueConfig *queue.Config, log logger.Logger) (AsyncHub, error) {
	if log == nil {
		log = logger.Discard
	}

	// Create queue factory
	factory := queue.NewQueueFactory(log)

	// Create queue
	q, err := factory.CreateQueue(queueConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue: %w", err)
	}

	ah := &asyncHub{
		Hub:    baseHub,
		queue:  q,
		logger: log,
	}

	// Create message handler
	handler := ah.createMessageHandler()

	// Create worker pool
	workerCount := queueConfig.Workers
	if workerCount <= 0 {
		workerCount = 4
	}

	ah.workerPool = queue.NewWorkerPool(q, handler, 1, workerCount, log)

	// Initialize health checker if queue supports monitoring
	if observableQueue, ok := q.(queue.ObservableQueue); ok {
		ah.healthChecker = queue.NewHealthChecker(observableQueue, ah.workerPool, 30*time.Second)
	}

	log.Info("Async hub created",
		"queueType", queueConfig.Type,
		"capacity", queueConfig.Capacity,
		"workers", workerCount,
		"healthCheckEnabled", ah.healthChecker != nil)

	return ah, nil
}

// SendQueued enqueues a message for async processing
func (ah *asyncHub) SendQueued(ctx context.Context, message *Message) (*AsyncReceipt, error) {
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	// Generate message ID if not set
	if message.ID == "" {
		message.ID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	}

	// Create queue message
	queueMsg := &queue.Message{
		ID:        message.ID,
		Payload:   message,
		Priority:  queue.Priority(message.Priority),
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set scheduled time if specified
	if message.ScheduledAt != nil {
		queueMsg.ScheduledAt = message.ScheduledAt
	}

	// Enqueue the message
	if err := ah.queue.Enqueue(ctx, queueMsg); err != nil {
		ah.logger.Error("Failed to enqueue message", "messageID", message.ID, "error", err)
		return nil, fmt.Errorf("failed to enqueue message: %w", err)
	}

	// Create async receipt
	receipt := &AsyncReceipt{
		MessageID: message.ID,
		Status:    "queued",
		QueuedAt:  time.Now(),
	}

	ah.logger.Debug("Message queued for async processing",
		"messageID", message.ID,
		"priority", message.Priority)

	return receipt, nil
}

// GetQueueStats returns queue statistics
func (ah *asyncHub) GetQueueStats() map[string]interface{} {
	stats := map[string]interface{}{
		"queue_size": ah.queue.Size(),
		"is_empty":   ah.queue.IsEmpty(),
	}

	// Add detailed metrics if available
	if observableQueue, ok := ah.queue.(queue.ObservableQueue); ok {
		queueStats := observableQueue.GetStats()
		stats["detailed"] = map[string]interface{}{
			"size":              queueStats.Size,
			"enqueued_count":    queueStats.EnqueuedCount,
			"dequeued_count":    queueStats.DequeuedCount,
			"retry_count":       queueStats.RetryCount,
			"dead_letter_count": queueStats.DeadLetterCount,
			"average_wait_time": queueStats.AverageWaitTime,
			"created_at":        queueStats.CreatedAt,
			"updated_at":        queueStats.UpdatedAt,
		}
	}

	// Add worker pool stats if available
	if ah.workerPool != nil {
		stats["workers"] = ah.workerPool.GetStats()
	}

	// Add health status if available
	if ah.healthChecker != nil {
		healthStatus := ah.healthChecker.GetStatus()
		stats["health"] = map[string]interface{}{
			"status":       healthStatus.Status,
			"queue_size":   healthStatus.QueueSize,
			"workers":      healthStatus.Workers,
			"last_process": healthStatus.LastProcess,
			"uptime":       healthStatus.Uptime,
			"error_count":  len(healthStatus.Errors),
		}
	}

	// Add processing status
	ah.mutex.RLock()
	stats["processing"] = ah.processing
	ah.mutex.RUnlock()

	return stats
}

// GetQueueMetrics returns detailed queue metrics
func (ah *asyncHub) GetQueueMetrics() *queue.QueueMetrics {
	if observableQueue, ok := ah.queue.(queue.ObservableQueue); ok {
		return observableQueue.GetMetrics()
	}
	return nil
}

// GetWorkerMetrics returns worker pool metrics
func (ah *asyncHub) GetWorkerMetrics() *queue.WorkerMetrics {
	if ah.workerPool != nil {
		return ah.workerPool.GetMetrics()
	}
	return nil
}

// GetHealthStatus returns health status
func (ah *asyncHub) GetHealthStatus() *queue.HealthStatus {
	if ah.healthChecker != nil {
		return ah.healthChecker.GetStatus()
	}
	return nil
}

// ProcessQueuedMessages starts processing queued messages
func (ah *asyncHub) ProcessQueuedMessages(ctx context.Context) error {
	ah.mutex.Lock()
	if ah.processing {
		ah.mutex.Unlock()
		return fmt.Errorf("already processing messages")
	}
	ah.processing = true
	ah.mutex.Unlock()

	ah.logger.Info("Starting async message processing")

	// Start worker pool
	if err := ah.workerPool.Start(ctx); err != nil {
		ah.mutex.Lock()
		ah.processing = false
		ah.mutex.Unlock()
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Start health checker if available
	if ah.healthChecker != nil {
		ah.healthChecker.Start(ctx)
		ah.logger.Debug("Health checker started")
	}

	return nil
}

// StopProcessing stops processing queued messages
func (ah *asyncHub) StopProcessing() error {
	ah.mutex.Lock()
	if !ah.processing {
		ah.mutex.Unlock()
		return fmt.Errorf("not processing messages")
	}
	ah.processing = false
	ah.mutex.Unlock()

	ah.logger.Info("Stopping async message processing")

	// Stop health checker if running
	if ah.healthChecker != nil {
		ah.healthChecker.Stop()
		ah.logger.Debug("Health checker stopped")
	}

	// Stop worker pool
	if err := ah.workerPool.Stop(); err != nil {
		return fmt.Errorf("failed to stop worker pool: %w", err)
	}

	return nil
}

// Close closes the async hub and releases resources
func (ah *asyncHub) Close(ctx context.Context) error {
	ah.logger.Info("Closing async hub")

	// Stop processing if running
	if ah.processing {
		_ = ah.StopProcessing()
	}

	// Stop health checker if running
	if ah.healthChecker != nil {
		ah.healthChecker.Stop()
	}

	// Close queue
	if err := ah.queue.Close(); err != nil {
		ah.logger.Error("Failed to close queue", "error", err)
	}

	// Close base hub
	if err := ah.Hub.Close(ctx); err != nil {
		ah.logger.Error("Failed to close base hub", "error", err)
		return err
	}

	ah.logger.Info("Async hub closed")
	return nil
}

// createMessageHandler creates a message handler for the worker pool
func (ah *asyncHub) createMessageHandler() queue.MessageHandler {
	return func(ctx context.Context, queueMsg *queue.Message) error {
		// Extract the original message
		msg, ok := queueMsg.Payload.(*Message)
		if !ok {
			ah.logger.Error("Invalid message payload type", "messageID", queueMsg.ID)
			return fmt.Errorf("invalid message payload")
		}

		ah.logger.Debug("Processing queued message", "messageID", msg.ID)

		// Send the message through the base hub
		receipt, err := ah.Send(ctx, msg)
		if err != nil {
			ah.logger.Error("Failed to send queued message",
				"messageID", msg.ID,
				"error", err)
			return fmt.Errorf("failed to send message: %w", err)
		}

		ah.logger.Info("Queued message processed successfully",
			"messageID", msg.ID,
			"receiptStatus", receipt.Status)

		return nil
	}
}

// QueuedHubOption represents configuration options for async hub
type QueuedHubOption func(*queuedHubConfig)

// queuedHubConfig holds configuration for async hub
type queuedHubConfig struct {
	QueueType       string
	QueueCapacity   int
	Workers         int
	RetryPolicy     queue.RetryPolicy
	DeadLetterQueue bool
	RedisOptions    *queue.RedisOptions
}

// WithMemoryQueue configures async hub with memory queue
func WithMemoryQueue(capacity int, workers int) QueuedHubOption {
	return func(c *queuedHubConfig) {
		c.QueueType = "memory"
		c.QueueCapacity = capacity
		c.Workers = workers
	}
}

// WithRedisQueue configures async hub with Redis queue
func WithRedisQueue(addr string, capacity int, workers int) QueuedHubOption {
	return func(c *queuedHubConfig) {
		c.QueueType = "redis"
		c.QueueCapacity = capacity
		c.Workers = workers
		c.RedisOptions = &queue.RedisOptions{
			Addr:      addr,
			KeyPrefix: "notifyhub:async:",
		}
	}
}

// WithQueueRetry configures retry policy for queued messages
func WithQueueRetry(maxRetries int, initialInterval time.Duration) QueuedHubOption {
	return func(c *queuedHubConfig) {
		c.RetryPolicy = queue.RetryPolicy{
			MaxRetries:      maxRetries,
			InitialInterval: initialInterval,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			MaxElapsedTime:  5 * time.Minute,
		}
	}
}

// WithDeadLetterQueue enables dead letter queue for failed messages
func WithDeadLetterQueue(enabled bool) QueuedHubOption {
	return func(c *queuedHubConfig) {
		c.DeadLetterQueue = enabled
	}
}

// NewAsyncHubWithOptions creates an async hub with options
func NewAsyncHubWithOptions(baseHub Hub, logger logger.Logger, opts ...QueuedHubOption) (AsyncHub, error) {
	// Default configuration
	config := &queuedHubConfig{
		QueueType:       "memory",
		QueueCapacity:   1000,
		Workers:         4,
		DeadLetterQueue: true,
		RetryPolicy: queue.RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			MaxElapsedTime:  5 * time.Minute,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Create queue config
	queueConfig := &queue.Config{
		Type:            config.QueueType,
		Capacity:        config.QueueCapacity,
		Workers:         config.Workers,
		RetryPolicy:     config.RetryPolicy,
		DeadLetterQueue: config.DeadLetterQueue,
		RedisOptions:    config.RedisOptions,
	}

	return NewAsyncHub(baseHub, queueConfig, logger)
}
