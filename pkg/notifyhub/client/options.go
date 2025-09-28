// Package client provides configuration options for NotifyHub clients
package client

import (
	"fmt"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
)

// ClientOption represents a functional configuration option for client creation
type ClientOption func(*ClientConfig) error

// ClientConfig represents client-specific configuration
type ClientConfig struct {
	// Core configuration options
	PlatformOptions []config.Option
	
	// Client-specific settings
	Logger logger.Logger
	Async  bool
	Workers int
	
	// Registry settings
	InstanceRegistry bool // Use instance-level registry vs global
}

// WithPlatformOptions adds platform configuration options
func WithPlatformOptions(opts ...config.Option) ClientOption {
	return func(c *ClientConfig) error {
		c.PlatformOptions = append(c.PlatformOptions, opts...)
		return nil
	}
}

// WithLogger sets a custom logger for the client
func WithLogger(logger logger.Logger) ClientOption {
	return func(c *ClientConfig) error {
		c.Logger = logger
		return nil
	}
}

// WithAsync enables asynchronous processing with specified worker count
func WithAsync(workers int) ClientOption {
	return func(c *ClientConfig) error {
		if workers <= 0 {
			return fmt.Errorf("worker count must be positive, got: %d", workers)
		}
		c.Async = true
		c.Workers = workers
		return nil
	}
}

// WithSync disables asynchronous processing (sync-only mode)
func WithSync() ClientOption {
	return func(c *ClientConfig) error {
		c.Async = false
		c.Workers = 0
		return nil
	}
}

// WithInstanceRegistry enables instance-level platform registry (recommended)
func WithInstanceRegistry() ClientOption {
	return func(c *ClientConfig) error {
		c.InstanceRegistry = true
		return nil
	}
}

// WithDefaults applies sensible default client configuration
func WithDefaults() ClientOption {
	return func(c *ClientConfig) error {
		c.Logger = logger.New()
		c.Async = false
		c.Workers = 4 // Default worker count when async is enabled
		c.InstanceRegistry = true // Always use instance registry to avoid global state
		return nil
	}
}

// WithTestDefaults applies test-friendly client configuration
func WithTestDefaults() ClientOption {
	return func(c *ClientConfig) error {
		c.Logger = logger.New() // Use regular logger for tests
		c.Async = true
		c.Workers = 2 // Fewer workers for tests
		c.InstanceRegistry = true
		return nil
	}
}

// applyClientDefaults applies default values to client configuration
func applyClientDefaults(cfg *ClientConfig) {
	if cfg.Logger == nil {
		cfg.Logger = logger.New()
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	cfg.InstanceRegistry = true // Always use instance registry
}

// buildClientConfig creates a client configuration from options
func buildClientConfig(opts ...ClientOption) (*ClientConfig, error) {
	cfg := &ClientConfig{}
	
	// Apply options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}
	
	// Apply defaults for unset values
	applyClientDefaults(cfg)
	
	return cfg, nil
}