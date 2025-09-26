// Package errors provides retry strategies and mechanisms for NotifyHub
package errors

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// RetryStrategy defines the interface for retry strategies
type RetryStrategy interface {
	// ShouldRetry determines if an operation should be retried based on the error and attempt count
	ShouldRetry(err error, attempt int) bool
	// GetDelay calculates the delay before the next retry attempt
	GetDelay(attempt int) time.Duration
	// GetMaxAttempts returns the maximum number of retry attempts
	GetMaxAttempts() int
}

// ExponentialBackoffStrategy implements exponential backoff with jitter
type ExponentialBackoffStrategy struct {
	BaseDelay      time.Duration // Base delay for first retry
	MaxDelay       time.Duration // Maximum delay between retries
	Multiplier     float64       // Exponential multiplier (typically 2.0)
	Jitter         float64       // Jitter factor (0.0 to 1.0)
	MaxAttempts    int           // Maximum retry attempts
	RetryableCodes map[Code]bool // Map of retryable error codes
}

// NewExponentialBackoffStrategy creates a new exponential backoff strategy
func NewExponentialBackoffStrategy() *ExponentialBackoffStrategy {
	retryableCodes := make(map[Code]bool)

	// Platform errors that are retryable
	retryableCodes[ErrPlatformUnavailable] = true
	retryableCodes[ErrPlatformRateLimit] = true
	retryableCodes[ErrPlatformTimeout] = true
	retryableCodes[ErrPlatformInternal] = true
	retryableCodes[ErrPlatformMaintenance] = true

	// Network errors that are retryable
	retryableCodes[ErrNetworkTimeout] = true
	retryableCodes[ErrNetworkConnection] = true
	retryableCodes[ErrNetworkDNS] = true

	// Queue errors that are retryable
	retryableCodes[ErrQueueFull] = true
	retryableCodes[ErrQueueTimeout] = true
	retryableCodes[ErrQueueConnection] = true
	retryableCodes[ErrQueueWorkerFailed] = true

	// System errors that are retryable
	retryableCodes[ErrSystemUnavailable] = true
	retryableCodes[ErrResourceExhausted] = true
	retryableCodes[ErrSystemTimeout] = true
	retryableCodes[ErrSystemOverload] = true

	// Message errors that are retryable
	retryableCodes[ErrMessageSendFailed] = true
	retryableCodes[ErrMessageTimeout] = true

	// Template errors that are retryable
	retryableCodes[ErrTemplateCacheError] = true

	return &ExponentialBackoffStrategy{
		BaseDelay:      1 * time.Second,
		MaxDelay:       30 * time.Second,
		Multiplier:     2.0,
		Jitter:         0.1,
		MaxAttempts:    5,
		RetryableCodes: retryableCodes,
	}
}

// ShouldRetry determines if an operation should be retried
func (s *ExponentialBackoffStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= s.MaxAttempts {
		return false
	}

	// Check if it's a NotifyError with a retryable code
	if notifyErr, ok := err.(*NotifyError); ok {
		return s.RetryableCodes[notifyErr.Code]
	}

	// For non-NotifyError, don't retry by default
	return false
}

// GetDelay calculates the delay with exponential backoff and jitter
func (s *ExponentialBackoffStrategy) GetDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return s.BaseDelay
	}

	// Calculate exponential delay
	delay := float64(s.BaseDelay) * math.Pow(s.Multiplier, float64(attempt))

	// Apply maximum delay limit
	if delay > float64(s.MaxDelay) {
		delay = float64(s.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	if s.Jitter > 0 {
		jitterAmount := delay * s.Jitter
		jitter := rand.Float64()*2*jitterAmount - jitterAmount // Random value between -jitterAmount and +jitterAmount
		delay += jitter
	}

	// Ensure delay is not negative
	if delay < 0 {
		delay = float64(s.BaseDelay)
	}

	return time.Duration(delay)
}

// GetMaxAttempts returns the maximum number of retry attempts
func (s *ExponentialBackoffStrategy) GetMaxAttempts() int {
	return s.MaxAttempts
}

// LinearBackoffStrategy implements linear backoff
type LinearBackoffStrategy struct {
	BaseDelay      time.Duration
	DelayIncrement time.Duration
	MaxDelay       time.Duration
	MaxAttempts    int
	RetryableCodes map[Code]bool
}

// NewLinearBackoffStrategy creates a new linear backoff strategy
func NewLinearBackoffStrategy() *LinearBackoffStrategy {
	strategy := NewExponentialBackoffStrategy()
	return &LinearBackoffStrategy{
		BaseDelay:      2 * time.Second,
		DelayIncrement: 1 * time.Second,
		MaxDelay:       20 * time.Second,
		MaxAttempts:    3,
		RetryableCodes: strategy.RetryableCodes,
	}
}

// ShouldRetry determines if an operation should be retried
func (s *LinearBackoffStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= s.MaxAttempts {
		return false
	}

	if notifyErr, ok := err.(*NotifyError); ok {
		return s.RetryableCodes[notifyErr.Code]
	}

	return false
}

// GetDelay calculates the delay with linear backoff
func (s *LinearBackoffStrategy) GetDelay(attempt int) time.Duration {
	delay := s.BaseDelay + time.Duration(attempt)*s.DelayIncrement
	if delay > s.MaxDelay {
		delay = s.MaxDelay
	}
	return delay
}

// GetMaxAttempts returns the maximum number of retry attempts
func (s *LinearBackoffStrategy) GetMaxAttempts() int {
	return s.MaxAttempts
}

// FixedDelayStrategy implements fixed delay between retries
type FixedDelayStrategy struct {
	Delay          time.Duration
	MaxAttempts    int
	RetryableCodes map[Code]bool
}

// NewFixedDelayStrategy creates a new fixed delay strategy
func NewFixedDelayStrategy(delay time.Duration, maxAttempts int) *FixedDelayStrategy {
	strategy := NewExponentialBackoffStrategy()
	return &FixedDelayStrategy{
		Delay:          delay,
		MaxAttempts:    maxAttempts,
		RetryableCodes: strategy.RetryableCodes,
	}
}

// ShouldRetry determines if an operation should be retried
func (s *FixedDelayStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= s.MaxAttempts {
		return false
	}

	if notifyErr, ok := err.(*NotifyError); ok {
		return s.RetryableCodes[notifyErr.Code]
	}

	return false
}

// GetDelay returns the fixed delay
func (s *FixedDelayStrategy) GetDelay(attempt int) time.Duration {
	return s.Delay
}

// GetMaxAttempts returns the maximum number of retry attempts
func (s *FixedDelayStrategy) GetMaxAttempts() int {
	return s.MaxAttempts
}

// RetryExecutor executes operations with retry logic
type RetryExecutor struct {
	strategy RetryStrategy
	logger   logger.Logger
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(strategy RetryStrategy, logger logger.Logger) *RetryExecutor {
	return &RetryExecutor{
		strategy: strategy,
		logger:   logger,
	}
}

// Execute executes a function with retry logic
func (r *RetryExecutor) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < r.strategy.GetMaxAttempts(); attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return NewSystemError(ErrSystemTimeout, "retry_executor", "context cancelled during retry")
		default:
		}

		// Execute the operation
		if err := operation(); err != nil {
			lastErr = err

			// Check if we should retry
			if !r.strategy.ShouldRetry(err, attempt) {
				r.logger.Debug("Operation failed, not retryable",
					"attempt", attempt+1,
					"error", err.Error())
				return err
			}

			// Calculate delay for next attempt
			delay := r.strategy.GetDelay(attempt)

			r.logger.Warn("Operation failed, retrying",
				"attempt", attempt+1,
				"max_attempts", r.strategy.GetMaxAttempts(),
				"delay", delay,
				"error", err.Error())

			// Wait for delay or context cancellation
			select {
			case <-ctx.Done():
				return NewSystemError(ErrSystemTimeout, "retry_executor", "context cancelled during retry delay")
			case <-time.After(delay):
				// Continue to next attempt
			}
		} else {
			// Operation succeeded
			if attempt > 0 {
				r.logger.Info("Operation succeeded after retries", "attempts", attempt+1)
			}
			return nil
		}
	}

	// All retry attempts exhausted
	r.logger.Error("Operation failed after all retry attempts",
		"attempts", r.strategy.GetMaxAttempts(),
		"last_error", lastErr.Error())

	return Wrap(lastErr, ErrSystemTimeout, "operation failed after all retry attempts")
}

// ExecuteWithCallback executes a function with retry logic and callbacks
func (r *RetryExecutor) ExecuteWithCallback(
	ctx context.Context,
	operation func() error,
	onRetry func(attempt int, err error, delay time.Duration),
) error {
	var lastErr error

	for attempt := 0; attempt < r.strategy.GetMaxAttempts(); attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return NewSystemError(ErrSystemTimeout, "retry_executor", "context cancelled during retry")
		default:
		}

		// Execute the operation
		if err := operation(); err != nil {
			lastErr = err

			// Check if we should retry
			if !r.strategy.ShouldRetry(err, attempt) {
				return err
			}

			// Calculate delay for next attempt
			delay := r.strategy.GetDelay(attempt)

			// Call retry callback if provided
			if onRetry != nil {
				onRetry(attempt, err, delay)
			}

			// Wait for delay or context cancellation
			select {
			case <-ctx.Done():
				return NewSystemError(ErrSystemTimeout, "retry_executor", "context cancelled during retry delay")
			case <-time.After(delay):
				// Continue to next attempt
			}
		} else {
			// Operation succeeded
			return nil
		}
	}

	// All retry attempts exhausted
	return Wrap(lastErr, ErrSystemTimeout, "operation failed after all retry attempts")
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	Strategy    string        `json:"strategy" yaml:"strategy"`         // "exponential", "linear", "fixed"
	BaseDelay   time.Duration `json:"base_delay" yaml:"base_delay"`     // Base delay
	MaxDelay    time.Duration `json:"max_delay" yaml:"max_delay"`       // Maximum delay
	MaxAttempts int           `json:"max_attempts" yaml:"max_attempts"` // Maximum attempts
	Multiplier  float64       `json:"multiplier" yaml:"multiplier"`     // Exponential multiplier
	Jitter      float64       `json:"jitter" yaml:"jitter"`             // Jitter factor
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Strategy:    "exponential",
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		MaxAttempts: 5,
		Multiplier:  2.0,
		Jitter:      0.1,
	}
}

// CreateRetryStrategy creates a retry strategy from configuration
func CreateRetryStrategy(config RetryConfig) RetryStrategy {
	switch config.Strategy {
	case "linear":
		strategy := NewLinearBackoffStrategy()
		strategy.BaseDelay = config.BaseDelay
		strategy.MaxDelay = config.MaxDelay
		strategy.MaxAttempts = config.MaxAttempts
		return strategy

	case "fixed":
		return NewFixedDelayStrategy(config.BaseDelay, config.MaxAttempts)

	default: // "exponential" or unknown defaults to exponential
		strategy := NewExponentialBackoffStrategy()
		strategy.BaseDelay = config.BaseDelay
		strategy.MaxDelay = config.MaxDelay
		strategy.MaxAttempts = config.MaxAttempts
		strategy.Multiplier = config.Multiplier
		strategy.Jitter = config.Jitter
		return strategy
	}
}
