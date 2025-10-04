// Package notifyhub provides the unified client factory for NotifyHub
package notifyhub

import (
	"context"
	"fmt"

	"github.com/kart/notifyhub/pkg/async"
	"github.com/kart/notifyhub/pkg/config"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/platform"
	"github.com/kart/notifyhub/pkg/platforms/email"
	"github.com/kart/notifyhub/pkg/platforms/feishu"
	"github.com/kart/notifyhub/pkg/platforms/webhook"
	receiptpkg "github.com/kart/notifyhub/pkg/receipt"
	"github.com/kart/notifyhub/pkg/target"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

// clientImpl implements the Client interface
type clientImpl struct {
	config           *config.Config
	platformRegistry platform.Registry
	asyncProcessor   *async.WorkerPool
	logger           logger.Logger
}

// NewClient creates a new NotifyHub client with the given configuration
func NewClient(cfg *config.Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create logger instance
	logger := cfg.LoggerInstance
	if logger == nil {
		return nil, fmt.Errorf("logger instance is required")
	}

	// Create platform registry
	registry := platform.NewRegistry(logger)

	// Register platform factories
	if err := registerPlatformFactories(registry, cfg); err != nil {
		return nil, fmt.Errorf("failed to register platform factories: %w", err)
	}

	// Set platform configurations
	if err := setPlatformConfigurations(registry, cfg); err != nil {
		return nil, fmt.Errorf("failed to set platform configurations: %w", err)
	}

	// Create async processor
	workerPoolConfig := async.WorkerPoolConfig{
		MinWorkers: cfg.Async.Workers,
		MaxWorkers: cfg.Async.Workers * 2,
	}
	asyncProcessor := async.NewWorkerPool(workerPoolConfig)

	client := &clientImpl{
		config:           cfg,
		platformRegistry: registry,
		asyncProcessor:   asyncProcessor,
		logger:           logger,
	}

	logger.Info("NotifyHub client created successfully")
	return client, nil
}

// NewClientFromOptions creates a new NotifyHub client with functional options
func NewClientFromOptions(opts ...config.Option) (Client, error) {
	cfg := &config.Config{}

	// Apply options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return NewClient(cfg)
}

// registerPlatformFactories registers all available platform factories
func registerPlatformFactories(registry platform.Registry, cfg *config.Config) error {
	// Register Feishu factory if configured
	if cfg.Feishu != nil {
		factory := func(config interface{}) (platform.Platform, error) {
			return feishu.NewPlatform(config)
		}

		if err := registry.RegisterFactory("feishu", factory); err != nil {
			return fmt.Errorf("failed to register feishu factory: %w", err)
		}
	}

	// Register Email factory if configured
	if cfg.Email != nil {
		factory := func(config interface{}) (platform.Platform, error) {
			return email.NewPlatform(config)
		}

		if err := registry.RegisterFactory("email", factory); err != nil {
			return fmt.Errorf("failed to register email factory: %w", err)
		}
	}

	// Register Webhook factory if configured
	if cfg.Webhook != nil {
		factory := func(config interface{}) (platform.Platform, error) {
			return webhook.NewPlatform(config)
		}

		if err := registry.RegisterFactory("webhook", factory); err != nil {
			return fmt.Errorf("failed to register webhook factory: %w", err)
		}
	}

	return nil
}

// setPlatformConfigurations sets platform configurations in the registry
func setPlatformConfigurations(registry platform.Registry, cfg *config.Config) error {
	// Set Feishu configuration
	if cfg.Feishu != nil {
		if err := registry.SetConfig("feishu", cfg.Feishu); err != nil {
			return fmt.Errorf("failed to set feishu configuration: %w", err)
		}
	}

	// Set Email configuration
	if cfg.Email != nil {
		if err := registry.SetConfig("email", cfg.Email); err != nil {
			return fmt.Errorf("failed to set email configuration: %w", err)
		}
	}

	// Set Webhook configuration
	if cfg.Webhook != nil {
		if err := registry.SetConfig("webhook", cfg.Webhook); err != nil {
			return fmt.Errorf("failed to set webhook configuration: %w", err)
		}
	}

	return nil
}

// Client interface implementation

// Send sends a message synchronously
func (c *clientImpl) Send(ctx context.Context, msg *message.Message) (*receiptpkg.Receipt, error) {
	c.logger.Debug("NotifyHub.Send() called", "message_id", msg.ID, "targets_count", len(msg.Targets))

	// Create receipt
	receipt := receiptpkg.New(msg.ID)

	// Send to all platforms configured in message targets
	for i, tgt := range msg.Targets {
		c.logger.Debug("处理目标 %d: Type=%s, Value=%s, Platform=%s", i+1, tgt.Type, tgt.Value, tgt.Platform)

		platformName := tgt.Platform
		if platformName == "" {
			c.logger.Warn("目标 %d 的platform为空，跳过", i+1)
			// TODO: Add routing logic to determine platform based on target type
			continue
		}

		platform, err := c.platformRegistry.GetPlatform(platformName)
		if err != nil {
			c.logger.Error("Failed to get platform", "platform", platformName, "error", err)
			receipt.AddResult(receiptpkg.PlatformResult{
				Platform:  platformName,
				Target:    tgt.Value,
				Success:   false,
				Error:     err.Error(),
				Timestamp: receipt.Timestamp,
			})
			continue
		}

		c.logger.Debug("Calling platform send method", "platform", platformName, "target", tgt.Value)
		results, err := platform.Send(ctx, msg, []target.Target{tgt})
		c.logger.Debug("Platform send completed", "platform", platformName, "success", err == nil, "results_count", len(results))
		if err != nil {
			c.logger.Error("Failed to send message", "platform", platformName, "error", err)
			receipt.AddResult(receiptpkg.PlatformResult{
				Platform:  platformName,
				Target:    tgt.Value,
				Success:   false,
				Error:     err.Error(),
				Timestamp: receipt.Timestamp,
			})
			continue
		}

		// Add results to receipt
		for _, result := range results {
			receipt.AddResult(receiptpkg.PlatformResult{
				Platform:  platformName,
				Target:    result.Target.Value,
				Success:   result.Success,
				MessageID: result.MessageID,
				Error:     "",
				Timestamp: receipt.Timestamp,
			})
		}
	}

	return receipt, nil
}

// SendBatch sends multiple messages synchronously
func (c *clientImpl) SendBatch(ctx context.Context, msgs []*message.Message) ([]*receiptpkg.Receipt, error) {
	receipts := make([]*receiptpkg.Receipt, len(msgs))
	var lastErr error

	for i, msg := range msgs {
		receipt, err := c.Send(ctx, msg)
		if err != nil {
			lastErr = err
		}
		receipts[i] = receipt
	}

	return receipts, lastErr
}

// SendAsync sends a message asynchronously
func (c *clientImpl) SendAsync(ctx context.Context, msg *message.Message, opts ...async.Option) (async.Handle, error) {
	// TODO: Implement async processing with WorkerPool
	return nil, fmt.Errorf("async processing not yet implemented")
}

// SendAsyncBatch sends multiple messages asynchronously
func (c *clientImpl) SendAsyncBatch(ctx context.Context, msgs []*message.Message, opts ...async.Option) (async.BatchHandle, error) {
	// TODO: Implement async batch processing with WorkerPool
	return nil, fmt.Errorf("async batch processing not yet implemented")
}

// Health returns the health status of the client
func (c *clientImpl) Health(ctx context.Context) (*HealthStatus, error) {
	platformHealth := c.platformRegistry.Health(ctx)

	platforms := make(map[string]string)
	allHealthy := true

	for name, err := range platformHealth {
		if err != nil {
			platforms[name] = "unhealthy: " + err.Error()
			allHealthy = false
		} else {
			platforms[name] = "healthy"
		}
	}

	status := "healthy"
	if !allHealthy {
		status = "degraded"
	}

	// TODO: Add metrics for uptime, active tasks, and queue depth
	return &HealthStatus{
		Status:      status,
		Platforms:   platforms,
		Uptime:      0,
		ActiveTasks: 0,
		QueueDepth:  0,
	}, nil
}

// Close closes the client and releases resources
func (c *clientImpl) Close() error {
	var lastErr error

	// Stop async processor
	if c.asyncProcessor != nil {
		c.asyncProcessor.Stop()
	}

	// Close platform registry
	if err := c.platformRegistry.Close(); err != nil {
		c.logger.Error("Failed to close platform registry", "error", err)
		lastErr = err
	}

	c.logger.Info("NotifyHub client closed")
	return lastErr
}
