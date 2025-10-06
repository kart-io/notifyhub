// Package notifyhub provides the unified client factory for NotifyHub
package notifyhub

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kart/notifyhub/pkg/async"
	"github.com/kart/notifyhub/pkg/config"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/platform"
	"github.com/kart/notifyhub/pkg/platforms/email"
	"github.com/kart/notifyhub/pkg/platforms/feishu"
	"github.com/kart/notifyhub/pkg/platforms/slack"
	"github.com/kart/notifyhub/pkg/platforms/webhook"
	receiptpkg "github.com/kart/notifyhub/pkg/receipt"
	"github.com/kart/notifyhub/pkg/target"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

// clientImpl implements the Client interface
type clientImpl struct {
	config           *config.Config
	platformRegistry platform.Registry
	asyncQueue       *async.MemoryQueue
	logger           logger.Logger

	// Metrics
	startTime    time.Time
	activeTasks  atomic.Int64
	totalSent    atomic.Int64
	totalSuccess atomic.Int64
	totalFailed  atomic.Int64
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
	if err := registerPlatformFactories(registry, cfg, logger); err != nil {
		return nil, fmt.Errorf("failed to register platform factories: %w", err)
	}

	// Set platform configurations
	if err := setPlatformConfigurations(registry, cfg); err != nil {
		return nil, fmt.Errorf("failed to set platform configurations: %w", err)
	}

	// Get async configuration with defaults
	asyncConfig := cfg.GetAsyncDefaults()

	// Create async queue if pool mode is enabled
	var asyncQueue *async.MemoryQueue
	if cfg.IsPoolModeEnabled() {
		queueConfig := async.QueueConfig{
			Workers:    asyncConfig.Workers,
			BufferSize: asyncConfig.BufferSize,
			Timeout:    asyncConfig.Timeout,
		}
		asyncQueue = async.NewMemoryQueue(queueConfig)

		// Start the queue
		ctx := context.Background()
		if err := asyncQueue.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start async queue: %w", err)
		}

		logger.Info("Goroutine pool enabled", "workers", asyncConfig.Workers, "buffer_size", asyncConfig.BufferSize)
	} else {
		logger.Info("Using direct goroutine mode (pool disabled)")
	}

	client := &clientImpl{
		config:           cfg,
		platformRegistry: registry,
		asyncQueue:       asyncQueue,
		logger:           logger,
		startTime:        time.Now(),
	}
	// Initialize atomic counters
	client.activeTasks.Store(0)
	client.totalSent.Store(0)
	client.totalSuccess.Store(0)
	client.totalFailed.Store(0)

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
func registerPlatformFactories(registry platform.Registry, cfg *config.Config, logger logger.Logger) error {
	// Register Feishu factory if configured
	if cfg.Feishu != nil {
		factory := func(config interface{}) (platform.Platform, error) {
			return feishu.NewPlatform(config, logger)
		}

		if err := registry.RegisterFactory("feishu", factory); err != nil {
			return fmt.Errorf("failed to register feishu factory: %w", err)
		}
	}

	// Register Email factory if configured
	if cfg.Email != nil {
		factory := func(config interface{}) (platform.Platform, error) {
			return email.NewPlatform(config, logger)
		}

		if err := registry.RegisterFactory("email", factory); err != nil {
			return fmt.Errorf("failed to register email factory: %w", err)
		}
	}

	// Register Webhook factory if configured
	if cfg.Webhook != nil {
		factory := func(config interface{}) (platform.Platform, error) {
			return webhook.NewPlatform(config, logger)
		}

		if err := registry.RegisterFactory("webhook", factory); err != nil {
			return fmt.Errorf("failed to register webhook factory: %w", err)
		}
	}

	// Register Slack factory if configured
	if cfg.Slack != nil {
		factory := func(config interface{}) (platform.Platform, error) {
			return slack.NewPlatform(config, logger)
		}

		if err := registry.RegisterFactory("slack", factory); err != nil {
			return fmt.Errorf("failed to register slack factory: %w", err)
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

	// Set Slack configuration
	if cfg.Slack != nil {
		if err := registry.SetConfig("slack", cfg.Slack); err != nil {
			return fmt.Errorf("failed to set slack configuration: %w", err)
		}
	}

	return nil
}

// Client interface implementation

// Send sends a message synchronously
func (c *clientImpl) Send(ctx context.Context, msg *message.Message) (*receiptpkg.Receipt, error) {
	c.logger.Debug("NotifyHub.Send() called", "message_id", msg.ID, "targets_count", len(msg.Targets))

	// Track active task
	c.activeTasks.Add(1)
	defer c.activeTasks.Add(-1)

	// Track total messages sent
	c.totalSent.Add(1)

	// Create receipt
	receipt := receiptpkg.New(msg.ID)

	// Send to all platforms configured in message targets
	for i, tgt := range msg.Targets {
		c.logger.Debug("处理目标 %d: Type=%s, Value=%s, Platform=%s", i+1, tgt.Type, tgt.Value, tgt.Platform)

		platformName := tgt.Platform
		if platformName == "" {
			// Auto-detect platform based on target type
			platformName = c.determinePlatformByTargetType(&tgt)
			if platformName == "" {
				c.logger.Warn("无法确定目标 %d 的平台类型，跳过", i+1)
				receipt.AddResult(receiptpkg.PlatformResult{
					Platform:  "unknown",
					Target:    tgt.Value,
					Success:   false,
					Error:     "unable to determine platform for target type: " + tgt.Type,
					Timestamp: receipt.Timestamp,
				})
				continue
			}
			c.logger.Debug("自动检测到平台类型", "target_type", tgt.Type, "platform", platformName)
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
			c.totalFailed.Add(1) // Track failed send
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
			if result.Success {
				c.totalSuccess.Add(1) // Track successful send
			} else {
				c.totalFailed.Add(1) // Track failed send
			}
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

// SendAsync sends a message asynchronously using the goroutine pool
func (c *clientImpl) SendAsync(ctx context.Context, msg *message.Message, opts ...async.Option) (async.Handle, error) {
	c.logger.Debug("NotifyHub.SendAsync() called", "message_id", msg.ID, "targets_count", len(msg.Targets))

	// Check if async queue is enabled
	if c.asyncQueue != nil && c.config.IsPoolModeEnabled() {
		// Use goroutine pool via async queue
		processor := func(ctx context.Context, message *message.Message, targets []target.Target) async.Result {
			// Call the synchronous Send method
			receipt, err := c.Send(ctx, message)
			return async.Result{
				Receipt: receipt,
				Error:   err,
			}
		}

		handle, err := c.asyncQueue.EnqueueWithProcessor(ctx, msg, msg.Targets, processor, opts...)
		if err != nil {
			c.logger.Error("Failed to enqueue message for async processing", "message_id", msg.ID, "error", err)
			return nil, err
		}

		c.logger.Debug("Message enqueued for async processing", "message_id", msg.ID)
		return handle, nil
	} else {
		// Fallback to direct goroutine (legacy mode)
		c.logger.Debug("Using legacy async mode (direct goroutine)", "message_id", msg.ID)

		var handle async.Handle = async.NewMemoryHandle(msg.ID)

		// Process the message in a goroutine
		go func(parentCtx context.Context, message *message.Message, asyncHandle async.Handle) {
			// Create a new context with timeout for async operation
			asyncCtx := context.Background()
			if c.config.Async.Timeout > 0 {
				var cancel context.CancelFunc
				asyncCtx, cancel = context.WithTimeout(asyncCtx, c.config.Async.Timeout)
				defer cancel()
			}

			// Call the synchronous Send method
			receipt, err := c.Send(asyncCtx, message)

			// Create result
			result := async.Result{
				Receipt: receipt,
				Error:   err,
			}

			// Send result to handle and trigger callbacks
			if memHandle, ok := asyncHandle.(*async.MemoryHandle); ok {
				memHandle.SetResultWithCallback(result, message)
			}
			c.logger.Debug("Async result sent successfully", "message_id", message.ID)
		}(ctx, msg, handle)

		return handle, nil
	}
}

// SendAsyncBatch sends multiple messages asynchronously using the goroutine pool
func (c *clientImpl) SendAsyncBatch(ctx context.Context, msgs []*message.Message, opts ...async.Option) (async.BatchHandle, error) {
	c.logger.Debug("NotifyHub.SendAsyncBatch() called", "message_count", len(msgs))

	if len(msgs) == 0 {
		return nil, fmt.Errorf("no messages provided for batch processing")
	}

	// Check if async queue is enabled
	if c.asyncQueue != nil && c.config.IsPoolModeEnabled() {
		// Use goroutine pool via async queue
		c.logger.Debug("Using goroutine pool for batch processing", "message_count", len(msgs))

		handles := make([]async.Handle, len(msgs))

		// Enqueue each message to the goroutine pool
		for i, currentMsg := range msgs {
			// Capture message for closure
			msg := currentMsg
			msgIndex := i

			processor := func(ctx context.Context, message *message.Message, targets []target.Target) async.Result {
				// Call the synchronous Send method
				receipt, err := c.Send(ctx, message)
				return async.Result{
					Receipt: receipt,
					Error:   err,
				}
			}

			handle, err := c.asyncQueue.EnqueueWithProcessor(ctx, msg, msg.Targets, processor, opts...)
			if err != nil {
				c.logger.Error("Failed to enqueue batch message", "message_id", msg.ID, "index", msgIndex, "error", err)
				return nil, fmt.Errorf("failed to enqueue message %d: %w", msgIndex, err)
			}
			handles[msgIndex] = handle
		}

		// Create batch handle
		batchHandle := async.NewBatchHandle(handles)
		c.logger.Debug("Batch messages enqueued for pool processing", "batch_id", batchHandle.BatchID())

		return batchHandle, nil
	} else {
		// Fallback to direct goroutines (legacy mode)
		c.logger.Debug("Using legacy batch async mode (direct goroutines)", "message_count", len(msgs))

		// Create individual handles for each message
		handles := make([]async.Handle, len(msgs))
		for i, msg := range msgs {
			handle := async.NewMemoryHandle(msg.ID)
			handles[i] = handle
		}

		// Create batch handle
		var batchHandle async.BatchHandle = async.NewBatchHandle(handles)

		// Process all messages in parallel using goroutines
		go func(parentCtx context.Context, messages []*message.Message, asyncHandles []async.Handle, batchAsyncHandle async.BatchHandle) {
			for idx, msgItem := range messages {
				go func(i int, msg *message.Message) {
					// Create a new context with timeout for async operation
					asyncCtx := context.Background()
					if c.config.Async.Timeout > 0 {
						var cancel context.CancelFunc
						asyncCtx, cancel = context.WithTimeout(asyncCtx, c.config.Async.Timeout)
						defer cancel()
					}

					// Call the synchronous Send method
					receipt, err := c.Send(asyncCtx, msg)

					// Create result
					result := async.Result{
						Receipt: receipt,
						Error:   err,
					}

					// Send result to individual handle
					if memHandle, ok := asyncHandles[i].(*async.MemoryHandle); ok {
						memHandle.SetResultWithCallback(result, msg)
					}

					// Also send result to batch handle
					if batchMemHandle, ok := batchAsyncHandle.(*async.MemoryBatchHandle); ok {
						batchMemHandle.AddResult(result)
					}
					c.logger.Debug("Batch result sent successfully", "message_id", msg.ID, "batch_id", batchAsyncHandle.BatchID())
				}(idx, msgItem)
			}
		}(ctx, msgs, handles, batchHandle)

		return batchHandle, nil
	}
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

	// Calculate metrics
	uptime := time.Since(c.startTime).Seconds()
	queueDepth := int64(0)
	if c.asyncQueue != nil {
		stats := c.asyncQueue.GetStats()
		queueDepth = stats.Pending
	}

	return &HealthStatus{
		Status:      status,
		Platforms:   platforms,
		Uptime:      uptime,
		ActiveTasks: c.activeTasks.Load(),
		QueueDepth:  queueDepth,
		TotalSent:   c.totalSent.Load(),
		SuccessRate: c.calculateSuccessRate(),
	}, nil
}

// calculateSuccessRate calculates the success rate percentage
func (c *clientImpl) calculateSuccessRate() float64 {
	total := c.totalSent.Load()
	if total == 0 {
		return 0.0
	}
	success := c.totalSuccess.Load()
	return (float64(success) / float64(total)) * 100.0
}

// Close closes the client and releases resources
func (c *clientImpl) Close() error {
	var lastErr error

	// Stop async queue
	if c.asyncQueue != nil {
		ctx := context.Background()
		if err := c.asyncQueue.Stop(ctx); err != nil {
			c.logger.Error("Failed to stop async queue", "error", err)
			lastErr = err
		}
	}

	// Close platform registry
	if err := c.platformRegistry.Close(); err != nil {
		c.logger.Error("Failed to close platform registry", "error", err)
		lastErr = err
	}

	c.logger.Info("NotifyHub client closed")
	return lastErr
}

// determinePlatformByTargetType determines the platform based on target type
func (c *clientImpl) determinePlatformByTargetType(tgt *target.Target) string {
	// Platform detection rules based on target type and value
	switch tgt.Type {
	case "email":
		// Email type always uses email platform
		return "email"

	case "phone":
		// Phone type uses SMS platform (if available)
		// Check if SMS is configured (via external platform or custom)
		if c.config.Slack != nil { // Placeholder - should check for SMS config
			return "sms"
		}
		// Fallback: some services can send to phone via other platforms
		c.logger.Debug("SMS platform not configured, checking alternatives")
		return ""

	case "webhook":
		// Webhook type uses webhook platform
		return "webhook"

	case "feishu":
		// Feishu specific type
		return "feishu"

	case "slack":
		// Slack specific type
		return "slack"

	case "dingtalk":
		// DingTalk type uses external platform
		// Check if DingTalk external platform is configured
		return ""

	case "user", "group":
		// User/group types need to be resolved first
		// Default to email if available
		if c.config.HasEmail() {
			return "email"
		}
		// Try other platforms in order of preference
		if c.config.HasFeishu() {
			return "feishu"
		}
		if c.config.HasSlack() {
			return "slack"
		}
		if c.config.HasWebhook() {
			return "webhook"
		}

	default:
		// For unknown types, try to infer from value format
		value := tgt.Value

		// Check if value looks like an email
		if strings.Contains(value, "@") && strings.Contains(value, ".") {
			if c.config.HasEmail() {
				c.logger.Debug("Value looks like email, using email platform", "value", value)
				return "email"
			}
		}

		// Check if value looks like a URL
		if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			if c.config.HasWebhook() {
				c.logger.Debug("Value looks like URL, using webhook platform", "value", value)
				return "webhook"
			}
		}

		// Check if value looks like a phone number
		if len(value) >= 10 && len(value) <= 15 {
			// Simple check: mostly digits with possible + or - prefix
			digits := 0
			for _, r := range value {
				if r >= '0' && r <= '9' {
					digits++
				}
			}
			if float64(digits)/float64(len(value)) > 0.8 {
				// Check if SMS platform is configured
				// Placeholder: no direct method to check SMS
				c.logger.Debug("Value looks like phone number but SMS platform not available", "value", value)
			}
		}
	}

	c.logger.Debug("无法自动检测平台类型", "target_type", tgt.Type, "value", tgt.Value)
	return ""
}
