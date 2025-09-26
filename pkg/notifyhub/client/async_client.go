// Package client provides asynchronous client implementation
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/async"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

// AsyncClient implements the Client interface with true asynchronous processing
// This replaces the pseudo-async implementation identified in the architecture analysis
type AsyncClient struct {
	syncClient *SyncClient
	executor   *async.AsyncExecutor
	logger     logger.Logger
	startTime  time.Time
}

// NewAsyncClient creates a new asynchronous client
func NewAsyncClient(syncClient *SyncClient, executor *async.AsyncExecutor, logger logger.Logger) *AsyncClient {
	client := &AsyncClient{
		syncClient: syncClient,
		executor:   executor,
		logger:     logger,
		startTime:  time.Now(),
	}

	// Start the async executor
	if err := executor.Start(); err != nil {
		logger.Error("Failed to start async executor", "error", err)
	}

	return client
}

// Send delegates to sync client for immediate processing
func (c *AsyncClient) Send(ctx context.Context, msg *message.Message) (*receipt.Receipt, error) {
	return c.syncClient.Send(ctx, msg)
}

// SendBatch delegates to sync client for immediate processing
func (c *AsyncClient) SendBatch(ctx context.Context, msgs []*message.Message) ([]*receipt.Receipt, error) {
	return c.syncClient.SendBatch(ctx, msgs)
}

// SendAsync implements true asynchronous sending with queue
func (c *AsyncClient) SendAsync(ctx context.Context, msg *message.Message, opts ...async.AsyncOption) (async.AsyncHandle, error) {
	if msg == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	// Apply async options to get configuration
	config := async.ApplyOptions(nil, opts...)
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid async config: %w", err)
	}

	c.logger.Debug("Queueing message for async processing", "message_id", msg.ID, "priority", config.Priority)

	// Set message priority from config
	msg.Priority = config.Priority

	// Get callback registry and register callbacks if provided
	callbacks := c.executor.GetCallbackRegistry()
	if config.HasCallbacks() {
		callbacks.RegisterMessageCallbacks(msg.ID, config.ToCallbacks())
	}

	// Create async handle
	handle := async.NewAsyncHandle(msg, callbacks)

	// Enqueue message
	queue := c.executor.GetQueue()
	if err := queue.Enqueue(ctx, msg, handle); err != nil {
		return nil, fmt.Errorf("failed to enqueue message: %w", err)
	}

	c.logger.Debug("Message queued for async processing", "message_id", msg.ID)
	return handle, nil
}

// SendAsyncBatch implements true asynchronous batch sending
func (c *AsyncClient) SendAsyncBatch(ctx context.Context, msgs []*message.Message, opts ...async.AsyncOption) (async.AsyncBatchHandle, error) {
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message batch cannot be empty")
	}

	// Apply async options to get configuration
	config := async.ApplyOptions(nil, opts...)
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid async config: %w", err)
	}

	c.logger.Debug("Queueing batch for async processing", "count", len(msgs), "priority", config.Priority)

	// Set priority for all messages
	for _, msg := range msgs {
		msg.Priority = config.Priority
	}

	// Get callback registry and register global callbacks if provided
	callbacks := c.executor.GetCallbackRegistry()
	if config.HasCallbacks() {
		callbacks.RegisterGlobalCallbacks(config.ToCallbacks())
	}

	// Generate batch ID and create batch handle
	batchID := fmt.Sprintf("batch_%d", time.Now().UnixNano())
	batchHandle := async.NewAsyncBatchHandle(batchID, msgs, callbacks)

	// Enqueue batch
	queue := c.executor.GetQueue()
	if err := queue.EnqueueBatch(ctx, msgs, batchHandle); err != nil {
		return nil, fmt.Errorf("failed to enqueue batch: %w", err)
	}

	c.logger.Debug("Batch queued for async processing", "batch_id", batchID, "count", len(msgs))
	return batchHandle, nil
}

// Health checks the health of both sync client and async components
func (c *AsyncClient) Health(ctx context.Context) (*HealthStatus, error) {
	// Get sync client health
	health, err := c.syncClient.Health(ctx)
	if err != nil {
		return health, err
	}

	// Add async-specific metadata
	if health.Metadata == nil {
		health.Metadata = make(map[string]interface{})
	}
	health.Metadata["client_type"] = "async"
	health.Metadata["executor_healthy"] = c.executor.IsHealthy()
	health.Metadata["executor_stats"] = c.executor.GetStats()
	health.Metadata["uptime"] = time.Since(c.startTime).Seconds()

	return health, nil
}

// Close gracefully shuts down the async client
func (c *AsyncClient) Close() error {
	c.logger.Info("Closing async client")

	// Stop async executor with timeout
	if err := c.executor.Stop(30 * time.Second); err != nil {
		c.logger.Error("Error stopping async executor", "error", err)
		// Continue with sync client shutdown even if async fails
	}

	// Close underlying sync client
	if err := c.syncClient.Close(); err != nil {
		c.logger.Error("Error closing sync client", "error", err)
		return err
	}

	c.logger.Info("Async client closed successfully")
	return nil
}
