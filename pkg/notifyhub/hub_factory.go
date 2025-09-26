// Package notifyhub provides hub factory functions for creating notification hubs.
// This file contains the core hub creation functionality using the functional options pattern.
package notifyhub

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

// Client is the unified interface for NotifyHub
// This is the main entry point for interacting with the notification system
type Client interface {
	// Send sends a single message
	Send(ctx context.Context, message *message.Message) (*receipt.Receipt, error)

	// SendBatch sends multiple messages
	SendBatch(ctx context.Context, messages []*message.Message) ([]*receipt.Receipt, error)

	// SendAsync sends a message asynchronously (non-blocking)
	SendAsync(ctx context.Context, message *message.Message) (<-chan *receipt.Receipt, error)

	// Health checks system health status
	Health(ctx context.Context) (*core.HealthStatus, error)

	// GetPlatformStatus gets the status of a specific platform
	GetPlatformStatus(ctx context.Context, platform string) (*PlatformStatus, error)

	// Close gracefully shuts down the client
	Close() error
}

// PlatformStatus represents the status of a platform
type PlatformStatus struct {
	Platform  string
	Status    string
	Available bool
	Details   map[string]string
}

// Option represents a configuration option for creating a client/hub
// Options are used with the functional options pattern to configure
// notification clients in a flexible and extensible way
type Option func(*config.Config) error

// HubOption is an alias for backward compatibility
// It can be used interchangeably with Option
type HubOption = Option

// New creates a new NotifyHub client (unified entry point)
// This is the recommended way to create a NotifyHub instance in v2+
//
// Example:
//
//	client, err := New(
//	    WithPlatform("email", emailConfig),
//	    WithTimeout(30*time.Second),
//	)
func New(opts ...Option) (Client, error) {
	cfg := &config.Config{
		Platforms:        make(map[string]map[string]interface{}),
		DefaultTimeout:   30 * time.Second,
		Logger:           logger.Discard, // Initialize with a silent logger by default
		ValidationErrors: make([]error, 0),
		RetryPolicy: config.RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			Multiplier:      2.0,
			MaxInterval:     30 * time.Second,
		},
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Inject hub logger into platforms that don't have one
	for _, platformConfig := range cfg.Platforms {
		if _, exists := platformConfig["logger"]; !exists {
			platformConfig["logger"] = cfg.Logger
		}
	}

	// Check for validation errors
	if len(cfg.ValidationErrors) > 0 {
		return nil, fmt.Errorf("configuration validation failed: %v", cfg.ValidationErrors)
	}

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create hub implementation
	hub, err := core.NewHub(cfg)
	if err != nil {
		return nil, err
	}

	// Wrap hub in client adapter
	return &clientAdapter{hub: hub}, nil
}

// NewFromConfig creates a client from a complete configuration
func NewFromConfig(config config.Config) (Client, error) {
	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create hub implementation
	hub, err := core.NewHub(&config)
	if err != nil {
		return nil, err
	}

	// Wrap hub in client adapter
	return &clientAdapter{hub: hub}, nil
}

// NewHub creates a new notification hub with the given configuration options.
//
// Deprecated: Use New() instead for the unified Client interface.
// This function is maintained for backward compatibility.
//
// NewHub uses the functional options pattern to provide a flexible way to
// configure notification hubs. It sets up default configurations and then
// applies the provided options.
//
// Parameters:
//   - opts: Variable number of HubOption functions to configure the hub
//
// Returns:
//   - Hub: The configured notification hub instance
//   - error: Any error that occurred during hub creation
//
// Example:
//
//	// Create a hub with platform packages
//	import "github.com/kart-io/notifyhub/pkg/platforms/email"
//
//	hub, err := NewHub(
//	    email.WithEmail("smtp.example.com", 587, "from@example.com"),
//	    WithTimeout(30*time.Second),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer hub.Close(context.Background())
//
// NewHub uses the functional options pattern to provide a flexible way to
// configure notification hubs. It sets up default configurations and then
// applies the provided options.
//
// Parameters:
//   - opts: Variable number of HubOption functions to configure the hub
//
// Returns:
//   - Hub: The configured notification hub instance
//   - error: Any error that occurred during hub creation
//
// Example:
//
//	// Create a hub with platform packages
//	import "github.com/kart-io/notifyhub/pkg/platforms/email"
//
//	hub, err := NewHub(
//	    email.WithEmail("smtp.example.com", 587, "from@example.com"),
//	    WithTimeout(30*time.Second),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer hub.Close(context.Background())
func NewHub(opts ...HubOption) (core.Hub, error) {
	cfg := &config.Config{
		Platforms:        make(map[string]map[string]interface{}),
		DefaultTimeout:   30 * time.Second,
		Logger:           logger.Discard, // Initialize with a silent logger by default
		ValidationErrors: make([]error, 0),
		RetryPolicy: config.RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			Multiplier:      2.0,
			MaxInterval:     30 * time.Second,
		},
	}

	// Apply options (HubOption now handles the error internally)
	for _, opt := range opts {
		if opt != nil {
			if err := opt(cfg); err != nil {
				// For backward compatibility, store errors instead of returning immediately
				cfg.ValidationErrors = append(cfg.ValidationErrors, err)
			}
		}
	}

	// Inject hub logger into platforms that don't have one
	for _, platformConfig := range cfg.Platforms {
		if _, exists := platformConfig["logger"]; !exists {
			platformConfig["logger"] = cfg.Logger
		}
	}

	// Check for validation errors
	if len(cfg.ValidationErrors) > 0 {
		return nil, fmt.Errorf("configuration validation failed: %v", cfg.ValidationErrors)
	}

	// Create real hub implementation
	return core.NewHub(cfg)
}

// Configuration option functions

// WithPlatform configures a platform (generic interface, no need to import platform packages)
func WithPlatform(name string, platformConfig map[string]interface{}) Option {
	return func(c *config.Config) error {
		if c.Platforms == nil {
			c.Platforms = make(map[string]map[string]interface{})
		}
		c.Platforms[name] = map[string]interface{}(platformConfig)
		return nil
	}
}

// WithYAML loads configuration from a YAML file
func WithYAML(path string) Option {
	return func(c *config.Config) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read YAML file: %w", err)
		}

		// TODO: Uncomment when yaml.v3 is available
		// var yamlConfig HubConfig
		// if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		// 	return fmt.Errorf("failed to parse YAML: %w", err)
		// }
		// mergeConfig(c, &yamlConfig)

		// For now, return an error indicating YAML support is not yet available
		_ = data
		return fmt.Errorf("YAML configuration support is not yet implemented")
	}
}

// WithJSON loads configuration from JSON data
func WithJSON(data []byte) Option {
	return func(c *config.Config) error {
		// JSON parsing logic will be implemented
		// json.Unmarshal(data, &jsonConfig)
		// mergeConfig(c, &jsonConfig)
		return nil
	}
}

// WithEnv loads configuration from environment variables
func WithEnv() Option {
	return func(c *config.Config) error {
		// Read environment variables and configure
		// e.g., NOTIFYHUB_FEISHU_WEBHOOK, NOTIFYHUB_EMAIL_HOST
		return nil
	}
}

// Convenience configuration options

// WithFeishu is deprecated - feishu package has been removed
// func WithFeishu(webhook string, secret ...string) Option {
// 	return WithPlatform("feishu", map[string]interface{}{
// 		"webhook": webhook,
// 		"secret":  firstOrEmpty(secret),
// 	})
// }

// WithEmail configures Email (no need to import email package)
func WithEmail(host string, port int, from string, opts ...map[string]interface{}) Option {
	config := map[string]interface{}{
		"host": host,
		"port": port,
		"from": from,
	}

	// Merge additional options
	if len(opts) > 0 {
		for k, v := range opts[0] {
			config[k] = v
		}
	}

	return WithPlatform("email", config)
}

// WithSMS configures SMS (no need to import sms package)
func WithSMS(provider string, apiKey string, opts ...map[string]interface{}) Option {
	config := map[string]interface{}{
		"provider": provider,
		"apiKey":   apiKey,
	}

	// Merge additional options
	if len(opts) > 0 {
		for k, v := range opts[0] {
			config[k] = v
		}
	}

	return WithPlatform("sms", config)
}

// WithSlack configures Slack (no need to import slack package)
func WithSlack(token string, opts ...map[string]interface{}) Option {
	config := map[string]interface{}{
		"token": token,
	}

	// Merge additional options
	if len(opts) > 0 {
		for k, v := range opts[0] {
			config[k] = v
		}
	}

	return WithPlatform("slack", config)
}

// WithTimeout sets the default timeout for all platform operations.
//
// This timeout applies to network operations, API calls, and other
// time-sensitive operations performed by platform adapters.
//
// Parameters:
//   - timeout: Duration to wait before timing out operations
//
// Returns:
//   - HubOption: Configuration option function
//
// Example:
//
//	hub, err := NewHub(
//	    WithTimeout(45*time.Second),
//	)
func WithTimeout(timeout time.Duration) HubOption {
	return func(cfg *config.Config) error {
		cfg.DefaultTimeout = timeout
		return nil
	}
}

// WithPlatformConfig configures a platform with generic configuration.
//
// This is a low-level function for configuring platforms when you need
// full control over the configuration map. For most use cases, use the
// platform-specific packages (e.g., email.WithEmail).
//
// Parameters:
//   - platformName: Name of the platform to configure
//   - config: Configuration map with platform-specific settings
//
// Returns:
//   - HubOption: Configuration option function
//
// Example:
//
//	hub, err := NewHub(
//	    WithPlatformConfig("custom-platform", map[string]interface{}{
//	        "api_key": "your-key",
//	        "endpoint": "https://api.custom.com",
//	    }),
//	)
func WithPlatformConfig(platformName string, platformConfig map[string]interface{}) HubOption {
	return func(cfg *config.Config) error {
		cfg.Platforms[platformName] = map[string]interface{}(platformConfig)
		return nil
	}
}

// WithRetryPolicy configures the retry behavior for failed message deliveries.
//
// The retry policy uses exponential backoff with jitter to handle temporary
// failures gracefully while avoiding overwhelming external services.
//
// Parameters:
//   - policy: RetryPolicy configuration
//
// Returns:
//   - HubOption: Configuration option function
//
// Example:
//
//	policy := config.RetryPolicy{
//	    MaxRetries:      5,
//	    InitialInterval: 2 * time.Second,
//	    Multiplier:      2.0,
//	    MaxInterval:     60 * time.Second,
//	}
//	hub, err := NewHub(
//	    WithRetryPolicy(policy),
//	)
func WithRetryPolicy(policy config.RetryPolicy) HubOption {
	return func(cfg *config.Config) error {
		cfg.RetryPolicy = policy
		return nil
	}
}

// WithTestDefaults configures the hub with safe test defaults.
//
// This option sets up minimal configuration suitable for testing
// without requiring real external service credentials.
//
// Returns:
//   - HubOption: Configuration option function
//
// Example:
//
//	hub, err := NewHub(WithTestDefaults())
func WithTestDefaults() HubOption {
	return func(cfg *config.Config) error {
		cfg.Platforms["test"] = map[string]interface{}{
			"type": "mock",
		}
		return nil
	}
}

// WithLogger sets a custom logger for the hub.
//
// This allows injecting external logging libraries like zap, slog, or logrus
// through adapter implementations that comply with our Logger interface.
//
// Parameters:
//   - logger: Logger instance implementing our Logger interface
//
// Returns:
//   - Option: Configuration option function
//
// Example:
//
//	// Using slog
//	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	hub, err := New(
//	    WithLogger(logger.NewSlogAdapter(slogLogger)),
//	)
//
//	// Using default logger with custom level
//	hub, err := New(
//	    WithLogger(logger.Default.LogMode(logger.Debug)),
//	)
func WithLogger(l logger.Logger) Option {
	return func(cfg *config.Config) error {
		cfg.Logger = l
		return nil
	}
}

// WithCustomPlatform configures a custom/external platform using the extension system.
//
// This function is used by platform packages to register themselves with the hub.
// It provides a way for external platform packages to integrate seamlessly.
//
// Parameters:
//   - platformName: Name of the platform to configure
//   - config: Configuration map with platform-specific settings
//
// Returns:
//   - HubOption: Configuration option function
//
// Example:
//
//	// Used by platform packages internally
//	return WithCustomPlatform("email", config)
func WithCustomPlatform(platformName string, config map[string]interface{}) HubOption {
	return WithPlatform(platformName, config)
}

// GetAvailablePlatforms returns information about all available platforms.
//
// This function provides information about both built-in platforms and
// registered extensions, allowing applications to discover available
// notification platforms at runtime.
//
// Returns:
//   - []PlatformInfo: List of available platform information
//
// Example:
//
//	platforms := GetAvailablePlatforms()
//	for _, platform := range platforms {
//	    fmt.Printf("Platform: %s (Built-in: %v)\n", platform.Name, platform.IsBuiltIn)
//	}
func GetAvailablePlatforms() []PlatformInfo {
	return GetAllPlatformInfo()
}

// Helper functions

func validateConfig(config *config.Config) error {
	// Validate at least one platform is configured
	if len(config.Platforms) == 0 {
		return fmt.Errorf("at least one platform must be configured")
	}

	// Validate timeout setting
	if config.DefaultTimeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	// Validate retry configuration
	if config.RetryPolicy.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	if config.RetryPolicy.InitialInterval <= 0 {
		return fmt.Errorf("initial interval must be positive")
	}

	return nil
}

// firstOrEmpty is removed as unused
// func firstOrEmpty(s []string) string {
// 	if len(s) > 0 {
// 		return s[0]
// 	}
// 	return ""
// }

// clientAdapter adapts Hub interface to Client interface
type clientAdapter struct {
	hub core.Hub
}

func (c *clientAdapter) Send(ctx context.Context, message *message.Message) (*receipt.Receipt, error) {
	return c.hub.Send(ctx, message)
}

func (c *clientAdapter) SendBatch(ctx context.Context, messages []*message.Message) ([]*receipt.Receipt, error) {
	// Hub interface doesn't have SendBatch, so we implement it using Send
	receipts := make([]*receipt.Receipt, 0, len(messages))
	for _, msg := range messages {
		result, err := c.hub.Send(ctx, msg)
		if err != nil {
			// Continue sending other messages even if one fails
			// Store error information in receipt
			receipts = append(receipts, &receipt.Receipt{
				MessageID: msg.ID,
				Status:    "failed",
				Error:     err,
			})
		} else {
			receipts = append(receipts, result)
		}
	}
	return receipts, nil
}

func (c *clientAdapter) SendAsync(ctx context.Context, message *message.Message) (<-chan *receipt.Receipt, error) {
	// Hub.SendAsync returns *receipt.AsyncReceipt, we need to adapt it
	asyncReceipt, err := c.hub.SendAsync(ctx, message)
	if err != nil {
		return nil, err
	}

	// Create a channel to return
	ch := make(chan *receipt.Receipt, 1)

	// Start a goroutine to convert AsyncReceipt to Receipt
	go func() {
		defer close(ch)
		// Convert AsyncReceipt to Receipt
		// Since AsyncReceipt doesn't have a Wait method, we create a receipt immediately
		result := &receipt.Receipt{
			MessageID: asyncReceipt.MessageID,
			Status:    asyncReceipt.Status,
			Timestamp: asyncReceipt.QueuedAt,
		}
		ch <- result
	}()

	return ch, nil
}

func (c *clientAdapter) Health(ctx context.Context) (*core.HealthStatus, error) {
	return c.hub.Health(ctx)
}

func (c *clientAdapter) GetPlatformStatus(ctx context.Context, platform string) (*PlatformStatus, error) {
	// Get health status first
	health, err := c.hub.Health(ctx)
	if err != nil {
		return nil, err
	}

	// Find platform status
	if platformHealth, exists := health.Platforms[platform]; exists {
		return &PlatformStatus{
			Platform:  platform,
			Status:    platformHealth.Status,
			Available: platformHealth.Available,
			Details:   platformHealth.Details,
		}, nil
	}

	return nil, fmt.Errorf("platform %s not found", platform)
}

func (c *clientAdapter) Close() error {
	return c.hub.Close(context.Background())
}
