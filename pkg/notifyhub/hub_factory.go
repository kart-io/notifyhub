// Package notifyhub provides hub factory functions for creating notification hubs.
// This file contains the core hub creation functionality using the functional options pattern.
package notifyhub

import (
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
)

// HubOption represents a configuration option for creating a hub.
// HubOptions are used with the functional options pattern to configure
// notification hubs in a flexible and extensible way.
//
// Example:
//
//	hub, err := NewHub(
//	    WithTimeout(30*time.Second),
//	)
type HubOption func(*HubConfig)

// NewHub creates a new notification hub with the given configuration options.
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
//	import "github.com/kart-io/notifyhub/pkg/platforms/feishu"
//	import "github.com/kart-io/notifyhub/pkg/platforms/email"
//
//	hub, err := NewHub(
//	    feishu.WithFeishu("webhook-url", feishu.WithFeishuSecret("secret")),
//	    email.WithEmail("smtp.example.com", 587, "from@example.com"),
//	    WithTimeout(30*time.Second),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer hub.Close(context.Background())
func NewHub(opts ...HubOption) (Hub, error) {
	cfg := &HubConfig{
		Platforms:        make(map[string]PlatformConfig),
		DefaultTimeout:   30 * time.Second,
		ValidationErrors: make([]error, 0),
		RetryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			Multiplier:      2.0,
			MaxInterval:     30 * time.Second,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Check for validation errors
	if len(cfg.ValidationErrors) > 0 {
		return nil, fmt.Errorf("configuration validation failed: %v", cfg.ValidationErrors)
	}

	// Create real hub implementation
	return core.NewHub(cfg)
}

// Configuration option functions

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
	return func(cfg *HubConfig) {
		cfg.DefaultTimeout = timeout
	}
}

// WithPlatformConfig configures a platform with generic configuration.
//
// This is a low-level function for configuring platforms when you need
// full control over the configuration map. For most use cases, use the
// platform-specific packages (e.g., feishu.WithFeishu, email.WithEmail).
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
func WithPlatformConfig(platformName string, config map[string]interface{}) HubOption {
	return func(cfg *HubConfig) {
		cfg.Platforms[platformName] = PlatformConfig(config)
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
//	policy := RetryPolicy{
//	    MaxRetries:      5,
//	    InitialInterval: 2 * time.Second,
//	    Multiplier:      2.0,
//	    MaxInterval:     60 * time.Second,
//	}
//	hub, err := NewHub(
//	    WithRetryPolicy(policy),
//	)
func WithRetryPolicy(policy RetryPolicy) HubOption {
	return func(cfg *HubConfig) {
		cfg.RetryPolicy = policy
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
	return func(cfg *HubConfig) {
		cfg.Platforms["test"] = PlatformConfig{
			"type": "mock",
		}
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
//	return WithCustomPlatform("feishu", config)
func WithCustomPlatform(platformName string, config map[string]interface{}) HubOption {
	return WithPlatformConfig(platformName, config)
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
