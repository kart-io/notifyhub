// Package middleware provides retry middleware for NotifyHub
package middleware

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/errors"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// RetryMiddleware provides intelligent retry logic with exponential backoff
type RetryMiddleware struct {
	BaseMiddleware
	strategy RetryStrategy
	logger   logger.Logger
}

// RetryStrategy defines the retry behavior
type RetryStrategy interface {
	ShouldRetry(ctx context.Context, err error, attempt int) bool
	GetDelay(attempt int) time.Duration
	GetMaxAttempts() int
}

// ExponentialBackoffStrategy implements exponential backoff with jitter
type ExponentialBackoffStrategy struct {
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	Multiplier     float64
	Jitter         float64
	MaxAttempts    int
	RetryableCodes map[errors.Code]bool
}

// NewExponentialBackoffStrategy creates a new exponential backoff strategy
func NewExponentialBackoffStrategy(config ExponentialBackoffConfig) *ExponentialBackoffStrategy {
	strategy := &ExponentialBackoffStrategy{
		BaseDelay:      config.BaseDelay,
		MaxDelay:       config.MaxDelay,
		Multiplier:     config.Multiplier,
		Jitter:         config.Jitter,
		MaxAttempts:    config.MaxAttempts,
		RetryableCodes: make(map[errors.Code]bool),
	}

	// Default retryable error codes
	if len(config.RetryableCodes) > 0 {
		strategy.RetryableCodes = config.RetryableCodes
	} else {
		// Set default retryable codes
		strategy.RetryableCodes = map[errors.Code]bool{
			errors.ErrPlatformTimeout:     true,
			errors.ErrPlatformUnavailable: true,
			errors.ErrPlatformRateLimit:   true,
			errors.ErrNetworkTimeout:      true,
			errors.ErrNetworkConnection:   true,
			errors.ErrQueueTimeout:        true,
			errors.ErrSystemTimeout:       true,
			errors.ErrResourceExhausted:   true,
		}
	}

	return strategy
}

// ExponentialBackoffConfig represents configuration for exponential backoff
type ExponentialBackoffConfig struct {
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	Multiplier     float64
	Jitter         float64
	MaxAttempts    int
	RetryableCodes map[errors.Code]bool
}

// ShouldRetry determines if an operation should be retried
func (e *ExponentialBackoffStrategy) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	// Check max attempts
	if attempt >= e.MaxAttempts {
		return false
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return false
	default:
	}

	// Check if error is retryable
	if notifyErr, ok := err.(*errors.NotifyError); ok {
		return e.RetryableCodes[notifyErr.Code]
	}

	return false
}

// GetDelay calculates delay with exponential backoff and jitter
func (e *ExponentialBackoffStrategy) GetDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return e.BaseDelay
	}

	delay := float64(e.BaseDelay) * math.Pow(e.Multiplier, float64(attempt))

	// Apply maximum delay limit
	if delay > float64(e.MaxDelay) {
		delay = float64(e.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	if e.Jitter > 0 {
		jitterAmount := delay * e.Jitter
		jitter := rand.Float64()*2*jitterAmount - jitterAmount
		delay += jitter
	}

	// Ensure delay is not negative
	if delay < 0 {
		delay = float64(e.BaseDelay)
	}

	return time.Duration(delay)
}

// GetMaxAttempts returns the maximum number of retry attempts
func (e *ExponentialBackoffStrategy) GetMaxAttempts() int {
	return e.MaxAttempts
}

// NewRetryMiddleware creates a new retry middleware
func NewRetryMiddleware(strategy RetryStrategy, l logger.Logger) *RetryMiddleware {
	if l == nil {
		l = logger.Discard
	}

	return &RetryMiddleware{
		BaseMiddleware: NewBaseMiddleware("retry"),
		strategy:       strategy,
		logger:         l,
	}
}

// HandleSend implements retry logic with exponential backoff
func (rm *RetryMiddleware) HandleSend(ctx context.Context, msg *message.Message, targets []target.Target, next SendHandler) (*receipt.Receipt, error) {
	var lastErr error
	var lastReceipt *receipt.Receipt

	for attempt := 0; attempt < rm.strategy.GetMaxAttempts(); attempt++ {
		// Check context cancellation before each attempt
		select {
		case <-ctx.Done():
			return lastReceipt, errors.NewSystemError(errors.ErrSystemTimeout, "retry_middleware", "context cancelled during retry")
		default:
		}

		// Execute the operation
		receipt, err := next(ctx, msg, targets)

		// If successful, return immediately
		if err == nil {
			if attempt > 0 {
				rm.logger.Info("Message send successful after retries",
					"message_id", msg.ID,
					"attempts", attempt+1)
			}
			return receipt, nil
		}

		// Store the error for potential return
		lastErr = err
		lastReceipt = receipt

		// Check if we should retry
		if !rm.strategy.ShouldRetry(ctx, err, attempt) {
			rm.logger.Debug("Error not retryable or max attempts reached",
				"message_id", msg.ID,
				"attempt", attempt+1,
				"error", err.Error())
			break
		}

		// Calculate and apply delay before next retry
		delay := rm.strategy.GetDelay(attempt)

		rm.logger.Warn("Message send failed, retrying",
			"message_id", msg.ID,
			"attempt", attempt+1,
			"max_attempts", rm.strategy.GetMaxAttempts(),
			"delay", delay,
			"error", err.Error())

		// Wait for delay or context cancellation
		select {
		case <-ctx.Done():
			return lastReceipt, errors.NewSystemError(errors.ErrSystemTimeout, "retry_middleware", "context cancelled during retry delay")
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retry attempts exhausted
	rm.logger.Error("Message send failed after all retry attempts",
		"message_id", msg.ID,
		"attempts", rm.strategy.GetMaxAttempts(),
		"last_error", lastErr.Error())

	return lastReceipt, errors.Wrap(lastErr, errors.ErrSystemTimeout, "message send failed after all retry attempts")
}

// FixedDelayStrategy implements fixed delay between retries
type FixedDelayStrategy struct {
	Delay          time.Duration
	MaxAttempts    int
	RetryableCodes map[errors.Code]bool
}

// NewFixedDelayStrategy creates a fixed delay strategy
func NewFixedDelayStrategy(delay time.Duration, maxAttempts int) *FixedDelayStrategy {
	return &FixedDelayStrategy{
		Delay:       delay,
		MaxAttempts: maxAttempts,
		RetryableCodes: map[errors.Code]bool{
			errors.ErrPlatformTimeout:     true,
			errors.ErrPlatformUnavailable: true,
			errors.ErrNetworkTimeout:      true,
			errors.ErrSystemTimeout:       true,
		},
	}
}

// ShouldRetry determines if an operation should be retried
func (f *FixedDelayStrategy) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	if attempt >= f.MaxAttempts {
		return false
	}

	select {
	case <-ctx.Done():
		return false
	default:
	}

	if notifyErr, ok := err.(*errors.NotifyError); ok {
		return f.RetryableCodes[notifyErr.Code]
	}

	return false
}

// GetDelay returns the fixed delay
func (f *FixedDelayStrategy) GetDelay(attempt int) time.Duration {
	return f.Delay
}

// GetMaxAttempts returns the maximum number of retry attempts
func (f *FixedDelayStrategy) GetMaxAttempts() int {
	return f.MaxAttempts
}

// LinearBackoffStrategy implements linear backoff
type LinearBackoffStrategy struct {
	BaseDelay      time.Duration
	DelayIncrement time.Duration
	MaxDelay       time.Duration
	MaxAttempts    int
	RetryableCodes map[errors.Code]bool
}

// NewLinearBackoffStrategy creates a linear backoff strategy
func NewLinearBackoffStrategy(config LinearBackoffConfig) *LinearBackoffStrategy {
	strategy := &LinearBackoffStrategy{
		BaseDelay:      config.BaseDelay,
		DelayIncrement: config.DelayIncrement,
		MaxDelay:       config.MaxDelay,
		MaxAttempts:    config.MaxAttempts,
		RetryableCodes: make(map[errors.Code]bool),
	}

	if len(config.RetryableCodes) > 0 {
		strategy.RetryableCodes = config.RetryableCodes
	} else {
		strategy.RetryableCodes = map[errors.Code]bool{
			errors.ErrPlatformTimeout: true,
			errors.ErrNetworkTimeout:  true,
			errors.ErrSystemTimeout:   true,
		}
	}

	return strategy
}

// LinearBackoffConfig represents configuration for linear backoff
type LinearBackoffConfig struct {
	BaseDelay      time.Duration
	DelayIncrement time.Duration
	MaxDelay       time.Duration
	MaxAttempts    int
	RetryableCodes map[errors.Code]bool
}

// ShouldRetry determines if an operation should be retried
func (l *LinearBackoffStrategy) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	if attempt >= l.MaxAttempts {
		return false
	}

	select {
	case <-ctx.Done():
		return false
	default:
	}

	if notifyErr, ok := err.(*errors.NotifyError); ok {
		return l.RetryableCodes[notifyErr.Code]
	}

	return false
}

// GetDelay calculates delay with linear backoff
func (l *LinearBackoffStrategy) GetDelay(attempt int) time.Duration {
	delay := l.BaseDelay + time.Duration(attempt)*l.DelayIncrement
	if delay > l.MaxDelay {
		delay = l.MaxDelay
	}
	return delay
}

// GetMaxAttempts returns the maximum number of retry attempts
func (l *LinearBackoffStrategy) GetMaxAttempts() int {
	return l.MaxAttempts
}
