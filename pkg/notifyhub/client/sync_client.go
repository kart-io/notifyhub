// Package client provides synchronous client implementation
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/async"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

// SyncClient implements the Client interface for synchronous operations
// This eliminates the clientAdapter layer identified in the architecture analysis
type SyncClient struct {
	dispatcher core.Dispatcher
	logger     logger.Logger
	startTime  time.Time
}

// NewSyncClient creates a new synchronous client
func NewSyncClient(dispatcher core.Dispatcher, logger logger.Logger) *SyncClient {
	return &SyncClient{
		dispatcher: dispatcher,
		logger:     logger,
		startTime:  time.Now(),
	}
}

// Send sends a single message synchronously
func (c *SyncClient) Send(ctx context.Context, msg *message.Message) (*receipt.Receipt, error) {
	if msg == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	c.logger.Debug("Sending message", "message_id", msg.ID)

	// Direct call to dispatcher - no adapter layer
	result, err := c.dispatcher.Dispatch(ctx, msg)
	if err != nil {
		c.logger.Error("Failed to send message", "message_id", msg.ID, "error", err)
		return nil, fmt.Errorf("failed to dispatch message: %w", err)
	}

	c.logger.Debug("Message sent successfully", "message_id", msg.ID, "successful", result.Successful, "failed", result.Failed)
	return result, nil
}

// SendBatch sends multiple messages synchronously
func (c *SyncClient) SendBatch(ctx context.Context, msgs []*message.Message) ([]*receipt.Receipt, error) {
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message batch cannot be empty")
	}

	c.logger.Debug("Sending message batch", "count", len(msgs))

	results := make([]*receipt.Receipt, 0, len(msgs))
	var errors []error

	for i, msg := range msgs {
		if msg == nil {
			err := fmt.Errorf("message at index %d cannot be nil", i)
			errors = append(errors, err)
			continue
		}

		result, err := c.Send(ctx, msg)
		if err != nil {
			errors = append(errors, fmt.Errorf("message %d failed: %w", i, err))
			continue
		}

		results = append(results, result)
	}

	if len(errors) > 0 {
		c.logger.Warn("Batch sending completed with errors", "successful", len(results), "failed", len(errors))
		return results, fmt.Errorf("batch sending failed: %d errors occurred", len(errors))
	}

	c.logger.Debug("Batch sent successfully", "count", len(results))
	return results, nil
}

// SendAsync creates an async handle that immediately executes (fallback for sync-only mode)
func (c *SyncClient) SendAsync(ctx context.Context, msg *message.Message, opts ...async.AsyncOption) (async.AsyncHandle, error) {
	// For sync client, we execute immediately and return a completed handle
	result, err := c.Send(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Create a callback registry for fallback compatibility
	callbacks := async.NewCallbackRegistry(c.logger)
	handle := async.NewAsyncHandle(msg, callbacks)

	// Immediately update with completed status since this is sync execution
	if handleImpl, ok := handle.(*async.HandleImpl); ok {
		handleImpl.UpdateStatus(async.StatusCompleted, 1.0, result, nil)
	}

	return handle, nil
}

// SendAsyncBatch creates an async batch handle that immediately executes (fallback for sync-only mode)
func (c *SyncClient) SendAsyncBatch(ctx context.Context, msgs []*message.Message, opts ...async.AsyncOption) (async.AsyncBatchHandle, error) {
	// For sync client, we execute immediately and return a completed handle
	_, err := c.SendBatch(ctx, msgs)

	// Create callback registry and batch handle
	callbacks := async.NewCallbackRegistry(c.logger)
	batchID := fmt.Sprintf("batch_%d", time.Now().UnixNano())
	handle := async.NewAsyncBatchHandle(batchID, msgs, callbacks)

	// For sync implementation, the batch is already completed
	// Update batch handle with completion status
	if batchImpl, ok := handle.(*async.BatchHandleImpl); ok {
		if err != nil {
			batchImpl.UpdateProgress(0, len(msgs)) // All failed
		} else {
			batchImpl.UpdateProgress(len(msgs), 0) // All succeeded
		}
	}

	return handle, err
}

// Health checks the health of the client and its platforms
func (c *SyncClient) Health(ctx context.Context) (*HealthStatus, error) {
	c.logger.Debug("Performing health check")

	// Get platform health from dispatcher
	platformHealth, err := c.dispatcher.Health(ctx)
	if err != nil {
		return &HealthStatus{
			Status:    "unhealthy",
			Platforms: map[string]string{},
			Uptime:    time.Since(c.startTime).Milliseconds(),
		}, err
	}

	status := &HealthStatus{
		Status:    "healthy",
		Platforms: platformHealth,
		Uptime:    time.Since(c.startTime).Milliseconds(),
		Metadata: map[string]interface{}{
			"client_type": "sync",
			"start_time":  c.startTime.Unix(),
		},
	}

	return status, nil
}

// Close gracefully shuts down the client
func (c *SyncClient) Close() error {
	c.logger.Info("Closing sync client")

	if err := c.dispatcher.Close(); err != nil {
		c.logger.Error("Error closing dispatcher", "error", err)
		return err
	}

	c.logger.Info("Sync client closed successfully")
	return nil
}
