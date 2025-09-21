package notifyhub

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/core/errors"
	"github.com/kart-io/notifyhub/core/hub"
	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/platforms"
	"github.com/kart-io/notifyhub/platforms/registry"
)

// Client is the unified client for NotifyHub, providing a simplified API entry point.
//
// The Client wraps the core Hub functionality and provides a streamlined interface
// for sending notifications across multiple platforms. It handles configuration,
// target resolution, and provides builder patterns for different message types.
//
// Example usage:
//
//	client, err := notifyhub.New(
//		WithFeishu("webhook-url", "secret"),
//		WithEmail("smtp.example.com", 587, "user", "pass", "from@example.com"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Shutdown(context.Background())
//
//	err = client.Send(ctx).
//		Title("Alert").
//		Body("System alert message").
//		ToEmail("admin@example.com").
//		ToFeishu("webhook-id").
//		Execute()
type Client struct {
	hub      *hub.Hub
	logger   logger.Interface
	config   *Config
	resolver *TargetResolver
}

// Config represents the simplified configuration structure for the NotifyHub client.
//
// This configuration is designed to be easy to use while providing full control
// over platform settings, queue management, retry behavior, and rate limiting.
// All fields are optional with sensible defaults.
type Config struct {
	// Platforms contains the configuration for each notification platform.
	// Each platform can be individually enabled/disabled and configured.
	Platforms []PlatformConfig `json:"platforms"`

	// Queue configures the internal message queue system.
	// Defaults to in-memory queue with 1000 capacity and 4 workers.
	Queue *QueueConfig `json:"queue,omitempty"`

	// Retry configures the retry behavior for failed messages.
	// Defaults to 3 attempts with exponential backoff.
	Retry *RetryConfig `json:"retry,omitempty"`

	// RateLimit configures per-platform rate limiting.
	// Helps prevent overwhelming external APIs.
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty"`

	// Logger provides the logging interface for the client.
	// Defaults to the standard logger if not provided.
	Logger logger.Interface `json:"-"`
}

// PlatformConfig 平台配置抽象
type PlatformConfig struct {
	Type     PlatformType           `json:"type"`
	Name     string                 `json:"name"`
	Settings map[string]interface{} `json:"settings"`
	Enabled  bool                   `json:"enabled"`
}

// PlatformType 平台类型
type PlatformType string

const (
	PlatformFeishu PlatformType = "feishu"
	PlatformEmail  PlatformType = "email"
	PlatformSMS    PlatformType = "sms"
	PlatformSlack  PlatformType = "slack"
)

// QueueConfig 队列配置
type QueueConfig struct {
	Type        string `json:"type"` // "memory", "redis"
	Capacity    int    `json:"capacity"`
	Concurrency int    `json:"concurrency"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	Backoff     time.Duration `json:"backoff"`
	Jitter      bool          `json:"jitter"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Rate   int           `json:"rate"`  // 每秒请求数
	Burst  int           `json:"burst"` // 突发容量
	Window time.Duration `json:"window"`
}

// Option 配置选项
type Option func(*Config)

// New creates a new NotifyHub client with the provided configuration options.
//
// This is the main entry point for creating a NotifyHub client. It accepts
// functional options to configure platforms, queue settings, retry behavior,
// and other client parameters.
//
// The client is created with sensible defaults:
//   - In-memory queue with 1000 capacity and 4 concurrent workers
//   - 3 retry attempts with exponential backoff
//   - Standard logger
//
// Example:
//
//	client, err := notifyhub.New(
//		WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/xxx", "secret"),
//		WithEmail("smtp.gmail.com", 587, "user@gmail.com", "password", "noreply@company.com"),
//		WithRetry(5, time.Second*2),
//		WithMemoryQueue(2000, 8),
//	)
//	if err != nil {
//		return fmt.Errorf("failed to create NotifyHub client: %w", err)
//	}
//
// The returned client must be properly shut down when no longer needed:
//
//	defer client.Shutdown(context.Background())
func New(options ...Option) (*Client, error) {
	cfg := &Config{
		Platforms: make([]PlatformConfig, 0),
		Queue: &QueueConfig{
			Type:        "memory",
			Capacity:    1000,
			Concurrency: 4,
		},
		Retry: &RetryConfig{
			MaxAttempts: 3,
			Backoff:     time.Second,
			Jitter:      true,
		},
		Logger: logger.Default,
	}

	// Apply configuration options
	for _, option := range options {
		option(cfg)
	}

	// Create Hub options
	hubOpts := &hub.Options{
		Logger: cfg.Logger,
	}

	// Create core Hub
	h := hub.NewHub(hubOpts)

	// Register platform transports using the new platform system
	if err := registerPlatformTransports(h, cfg.Platforms); err != nil {
		return nil, err
	}

	// Create target resolver
	resolver := &TargetResolver{}

	client := &Client{
		hub:      h,
		logger:   cfg.Logger,
		config:   cfg,
		resolver: resolver,
	}

	return client, nil
}

// Send creates a new message sending builder for composing and sending notifications.
//
// This method returns a SendBuilder that provides a fluent interface for constructing
// messages with titles, bodies, formats, and target recipients across multiple platforms.
// The builder pattern allows for easy composition of complex messages.
//
// Example usage:
//
//	err := client.Send(ctx).
//		Title("System Alert").
//		Body("Database connection failed").
//		Format(message.FormatMarkdown).
//		Priority(message.PriorityHigh).
//		ToEmail("admin@example.com").
//		ToFeishu("general").
//		Execute()
//	if err != nil {
//		log.Printf("Failed to send notification: %v", err)
//	}
//
// The context passed to this method will be used for the entire sending operation,
// including timeout handling and cancellation.
func (c *Client) Send(ctx context.Context) *SendBuilder {
	return &SendBuilder{
		client:  c,
		ctx:     ctx,
		message: core.NewMessage(),
		targets: make([]core.Target, 0),
	}
}

// Alert creates a pre-configured message builder optimized for alert notifications.
//
// This method returns an AlertBuilder that is pre-configured with high priority
// and alert-specific defaults. It's designed for urgent notifications that require
// immediate attention, such as system failures, security incidents, or critical errors.
//
// The AlertBuilder extends SendBuilder with alert-specific functionality and
// automatically sets the message priority to High.
//
// Example usage:
//
//	err := client.Alert(ctx).
//		Title("CRITICAL: Database Down").
//		Body("Primary database cluster is unreachable").
//		Severity("critical").
//		ToOnCall().
//		Execute()
//	if err != nil {
//		log.Printf("Failed to send alert: %v", err)
//	}
//
// Alerts are typically sent to on-call engineers, incident response teams,
// or escalation channels configured in your notification setup.
func (c *Client) Alert(ctx context.Context) *AlertBuilder {
	return &AlertBuilder{
		SendBuilder: c.Send(ctx),
		priority:    int(core.PriorityHigh),
	}
}

// Notification creates a pre-configured message builder for general notifications.
//
// This method returns a NotificationBuilder that is pre-configured with normal priority
// and designed for routine, informational messages. It's suitable for status updates,
// reports, reminders, and other non-urgent communications.
//
// The NotificationBuilder extends SendBuilder with notification-specific functionality
// and automatically sets the message priority to Normal.
//
// Example usage:
//
//	err := client.Notification(ctx).
//		Title("Daily Report").
//		Body("System processed 1,234 requests today").
//		ToChannel("general").
//		Schedule(time.Now().Add(time.Hour)).
//		Execute()
//	if err != nil {
//		log.Printf("Failed to send notification: %v", err)
//	}
//
// Notifications are typically sent to general channels, team inboxes,
// or subscriber lists for informational purposes.
func (c *Client) Notification(ctx context.Context) *NotificationBuilder {
	return &NotificationBuilder{
		SendBuilder: c.Send(ctx),
		priority:    int(core.PriorityNormal),
	}
}

// Configure 动态配置变更
func (c *Client) Configure(options ...Option) error {
	for _, option := range options {
		option(c.config)
	}
	// TODO: 重新配置Hub
	return nil
}

// Health performs a comprehensive health check of the NotifyHub client and all its components.
//
// This method checks the status of all registered transport connections, queue systems,
// and internal components. It returns a HealthStatus that indicates whether the system
// is operating normally and provides detailed information about each component.
//
// The health check includes:
//   - Transport connectivity (can reach external APIs)
//   - Queue system status (memory usage, worker health)
//   - Rate limiter status
//   - Overall system health
//
// Example usage:
//
//	status := client.Health()
//	if !status.Healthy {
//		log.Printf("NotifyHub unhealthy: %+v", status.Details)
//		// Take remedial action
//	}
//
// This method is safe to call frequently and can be used for monitoring
// and health check endpoints in web services.
func (c *Client) Health() HealthStatus {
	hubStatus := c.hub.Health(context.Background())

	// Add platform health information
	platformHealth := make(map[string]interface{})
	for _, info := range registry.GlobalRegistry.ListWithInfo() {
		platformHealth[info.Name] = map[string]interface{}{
			"description": info.Description,
			"capabilities": map[string]interface{}{
				"formats":     info.Capabilities.SupportedFormats(),
				"features":    info.Capabilities.Features(),
				"rate_limits": info.Capabilities.RateLimits(),
			},
		}
	}

	hubStatus.Details["platforms"] = platformHealth

	return HealthStatus{
		Healthy: hubStatus.Healthy,
		Details: hubStatus.Details,
	}
}

// Shutdown gracefully shuts down the NotifyHub client and all its components.
//
// This method ensures that all pending messages are processed, connections are
// properly closed, and resources are cleaned up. It should be called when the
// application is terminating or when the client is no longer needed.
//
// The shutdown process includes:
//   - Stopping acceptance of new messages
//   - Waiting for queued messages to be sent (respecting context timeout)
//   - Closing transport connections
//   - Cleaning up internal resources
//
// Example usage:
//
//	// Give the client 30 seconds to finish pending operations
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	if err := client.Shutdown(ctx); err != nil {
//		log.Printf("Error during shutdown: %v", err)
//	}
//
// If the context expires before shutdown completes, some messages may be lost.
// For critical applications, consider using a longer timeout or monitoring
// the queue depth before initiating shutdown.
func (c *Client) Shutdown(ctx context.Context) error {
	return c.hub.Shutdown(ctx)
}

// HealthStatus 健康状态
type HealthStatus struct {
	Healthy bool                   `json:"healthy"`
	Details map[string]interface{} `json:"details"`
}

// GetPlatformCapabilities returns the capabilities of a specific platform
func (c *Client) GetPlatformCapabilities(platformName string) (platforms.Capabilities, error) {
	platform, err := registry.GlobalRegistry.Get(platformName)
	if err != nil {
		return nil, errors.WrapWithPlatform(errors.CodeInvalidPlatform, errors.CategoryConfig,
			"platform "+platformName+" not found", platformName, err)
	}
	return platform.Capabilities(), nil
}

// ListAvailablePlatforms returns information about all available platforms
func (c *Client) ListAvailablePlatforms() []registry.PlatformInfo {
	return registry.GlobalRegistry.ListWithInfo()
}

// TargetResolver resolves targets to appropriate platforms
type TargetResolver struct {
	// Target resolution logic
}

// registerPlatformTransports registers platform transports with the hub
func registerPlatformTransports(h *hub.Hub, platformConfigs []PlatformConfig) error {
	// Get all registered platforms from the registry
	registeredPlatforms := registry.GlobalRegistry.GetAll()

	for _, cfg := range platformConfigs {
		if !cfg.Enabled {
			continue
		}

		// Find the platform implementation
		platform, exists := registeredPlatforms[cfg.Name]
		if !exists {
			return errors.New(errors.CodeInvalidPlatform, errors.CategoryConfig,
				"platform "+cfg.Name+" not registered")
		}

		// Validate configuration
		if err := platform.ValidateConfig(cfg.Settings); err != nil {
			return errors.WrapWithPlatform(errors.CodeInvalidConfig, errors.CategoryConfig,
				"invalid config for platform "+cfg.Name, cfg.Name, err)
		}

		// Create transport
		transport, err := platform.CreateTransport(cfg.Settings)
		if err != nil {
			return errors.WrapWithPlatform(errors.CodeProcessingFailed, errors.CategoryPlatform,
				"failed to create transport for "+cfg.Name, cfg.Name, err)
		}

		// Register transport with hub
		h.RegisterTransport(&transportAdapter{
			transport: transport,
			platform:  platform,
		})
	}

	return nil
}

// transportAdapter adapts platform.Transport to hub.Transport
type transportAdapter struct {
	transport platforms.Transport
	platform  platforms.Platform
}

func (t *transportAdapter) Send(ctx context.Context, msg *core.Message, target core.Target) (*core.Result, error) {
	// Check platform capabilities before sending
	caps := t.platform.Capabilities()

	// Convert message format to platform format
	platformFormat := platforms.Format(msg.Format)

	// Validate message format
	if !caps.SupportsFormat(platformFormat) {
		return nil, errors.NewWithPlatform(errors.CodeInvalidFormat, errors.CategoryValidation,
			"platform "+t.platform.Name()+" does not support format "+string(msg.Format), t.platform.Name())
	}

	// Check message size limits
	if len(msg.Title) > caps.MaxTitleLength() {
		return nil, errors.NewWithPlatform(errors.CodeInvalidFormat, errors.CategoryValidation,
			"title exceeds maximum length", t.platform.Name())
	}
	if len(msg.Body) > caps.MaxBodyLength() {
		return nil, errors.NewWithPlatform(errors.CodeInvalidFormat, errors.CategoryValidation,
			"body exceeds maximum length", t.platform.Name())
	}

	// Call transport and convert result
	result, err := t.transport.Send(ctx, msg, target)
	if err != nil {
		return nil, err
	}

	// Result is already *core.Result from transport
	return result, nil
}

func (t *transportAdapter) Name() string {
	return t.transport.Name()
}

func (t *transportAdapter) Shutdown() error {
	return t.transport.Shutdown()
}

func (t *transportAdapter) Health(ctx context.Context) error {
	return t.transport.Health(ctx)
}
