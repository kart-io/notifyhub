// Package async provides configuration options for asynchronous operations
package async

import (
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

// AsyncOption represents an option for asynchronous operations
type AsyncOption func(*AsyncConfig)

// AsyncConfig represents configuration for asynchronous operations
type AsyncConfig struct {
	// Callback functions
	OnResult   func(*receipt.Receipt)
	OnError    func(*message.Message, error)
	OnProgress func(completed, total int)
	OnComplete func(*BatchSummary)

	// Operation parameters
	Priority message.Priority
	Timeout  time.Duration

	// Retry configuration
	MaxRetries       int
	RetryInterval    time.Duration
	RetryMultiplier  float64
	MaxRetryInterval time.Duration

	// Custom metadata
	Metadata map[string]interface{}
}

// DefaultAsyncConfig returns the default async configuration
func DefaultAsyncConfig() *AsyncConfig {
	return &AsyncConfig{
		Priority:         message.PriorityNormal,
		Timeout:          30 * time.Second,
		MaxRetries:       3,
		RetryInterval:    1 * time.Second,
		RetryMultiplier:  2.0,
		MaxRetryInterval: 30 * time.Second,
		Metadata:         make(map[string]interface{}),
	}
}

// WithResultCallback sets a callback for successful results
func WithResultCallback(callback func(*receipt.Receipt)) AsyncOption {
	return func(config *AsyncConfig) {
		config.OnResult = callback
	}
}

// WithErrorCallback sets a callback for errors
func WithErrorCallback(callback func(*message.Message, error)) AsyncOption {
	return func(config *AsyncConfig) {
		config.OnError = callback
	}
}

// WithProgressCallback sets a callback for batch progress updates
func WithProgressCallback(callback func(completed, total int)) AsyncOption {
	return func(config *AsyncConfig) {
		config.OnProgress = callback
	}
}

// WithCompleteCallback sets a callback for batch completion
func WithCompleteCallback(callback func(*BatchSummary)) AsyncOption {
	return func(config *AsyncConfig) {
		config.OnComplete = callback
	}
}

// WithPriority sets the message priority
func WithPriority(priority message.Priority) AsyncOption {
	return func(config *AsyncConfig) {
		config.Priority = priority
	}
}

// WithTimeout sets the operation timeout
func WithTimeout(timeout time.Duration) AsyncOption {
	return func(config *AsyncConfig) {
		config.Timeout = timeout
	}
}

// WithRetryPolicy sets the retry configuration
func WithRetryPolicy(maxRetries int, interval time.Duration, multiplier float64, maxInterval time.Duration) AsyncOption {
	return func(config *AsyncConfig) {
		config.MaxRetries = maxRetries
		config.RetryInterval = interval
		config.RetryMultiplier = multiplier
		config.MaxRetryInterval = maxInterval
	}
}

// WithMetadata sets custom metadata for the operation
func WithMetadata(key string, value interface{}) AsyncOption {
	return func(config *AsyncConfig) {
		if config.Metadata == nil {
			config.Metadata = make(map[string]interface{})
		}
		config.Metadata[key] = value
	}
}

// WithHighPriority sets the message to high priority
func WithHighPriority() AsyncOption {
	return WithPriority(message.PriorityHigh)
}

// WithLowPriority sets the message to low priority
func WithLowPriority() AsyncOption {
	return WithPriority(message.PriorityLow)
}

// WithUrgentPriority sets the message to urgent priority
func WithUrgentPriority() AsyncOption {
	return WithPriority(message.PriorityUrgent)
}

// WithNoRetry disables retries for this operation
func WithNoRetry() AsyncOption {
	return WithRetryPolicy(0, 0, 0, 0)
}

// WithQuickRetry sets a quick retry policy (faster retries, fewer attempts)
func WithQuickRetry() AsyncOption {
	return WithRetryPolicy(2, 500*time.Millisecond, 1.5, 5*time.Second)
}

// WithAggressiveRetry sets an aggressive retry policy (more retries, longer intervals)
func WithAggressiveRetry() AsyncOption {
	return WithRetryPolicy(5, 2*time.Second, 2.0, 60*time.Second)
}

// WithCallbacks is a convenience function to set all callback types at once
func WithCallbacks(
	onResult func(*receipt.Receipt),
	onError func(*message.Message, error),
	onProgress func(completed, total int),
	onComplete func(*BatchSummary),
) AsyncOption {
	return func(config *AsyncConfig) {
		if onResult != nil {
			config.OnResult = onResult
		}
		if onError != nil {
			config.OnError = onError
		}
		if onProgress != nil {
			config.OnProgress = onProgress
		}
		if onComplete != nil {
			config.OnComplete = onComplete
		}
	}
}

// ApplyOptions applies multiple options to a config
func ApplyOptions(config *AsyncConfig, opts ...AsyncOption) *AsyncConfig {
	if config == nil {
		config = DefaultAsyncConfig()
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

// ToCallbacks converts AsyncConfig callback functions to Callbacks struct
func (c *AsyncConfig) ToCallbacks() *Callbacks {
	return &Callbacks{
		OnResult:   c.OnResult,
		OnError:    c.OnError,
		OnProgress: c.OnProgress,
		OnComplete: c.OnComplete,
	}
}

// HasCallbacks returns whether any callbacks are configured
func (c *AsyncConfig) HasCallbacks() bool {
	return c.OnResult != nil || c.OnError != nil || c.OnProgress != nil || c.OnComplete != nil
}

// Clone creates a deep copy of the AsyncConfig
func (c *AsyncConfig) Clone() *AsyncConfig {
	clone := &AsyncConfig{
		OnResult:         c.OnResult,
		OnError:          c.OnError,
		OnProgress:       c.OnProgress,
		OnComplete:       c.OnComplete,
		Priority:         c.Priority,
		Timeout:          c.Timeout,
		MaxRetries:       c.MaxRetries,
		RetryInterval:    c.RetryInterval,
		RetryMultiplier:  c.RetryMultiplier,
		MaxRetryInterval: c.MaxRetryInterval,
		Metadata:         make(map[string]interface{}),
	}

	// Deep copy metadata
	for k, v := range c.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// Validate validates the async configuration
func (c *AsyncConfig) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if c.RetryInterval < 0 {
		return fmt.Errorf("retry interval cannot be negative")
	}

	if c.RetryMultiplier < 1.0 && c.MaxRetries > 0 {
		return fmt.Errorf("retry multiplier must be >= 1.0 when retries are enabled")
	}

	if c.MaxRetryInterval < c.RetryInterval && c.MaxRetries > 0 {
		return fmt.Errorf("max retry interval must be >= retry interval")
	}

	return nil
}

// String returns a string representation of the configuration
func (c *AsyncConfig) String() string {
	return fmt.Sprintf("AsyncConfig{Priority:%v, Timeout:%v, MaxRetries:%d, HasCallbacks:%t}",
		c.Priority, c.Timeout, c.MaxRetries, c.HasCallbacks())
}
