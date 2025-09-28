// Package middleware provides retry middleware for NotifyHub
package middleware

import (
	"context"
	"math"
	"math/rand"
	"sync"
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
	strategy       RetryStrategy
	logger         logger.Logger
	statistics     *errors.RetryStatistics
	performanceMonitor *errors.PerformanceMonitor
	circuitBreakers    map[string]*errors.CircuitBreaker
	mu                 sync.RWMutex
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
		BaseMiddleware:      NewBaseMiddleware("retry"),
		strategy:           strategy,
		logger:             l,
		statistics:         errors.NewRetryStatistics(),
		performanceMonitor: errors.NewPerformanceMonitor(),
		circuitBreakers:    make(map[string]*errors.CircuitBreaker),
	}
}

// HandleSend implements retry logic with exponential backoff and performance monitoring
func (rm *RetryMiddleware) HandleSend(ctx context.Context, msg *message.Message, targets []target.Target, next SendHandler) (*receipt.Receipt, error) {
	var lastErr error
	var lastReceipt *receipt.Receipt
	startTime := time.Now()

	// Determine primary platform for metrics
	platform := rm.extractPlatformFromTargets(targets)

	// Check circuit breaker for platform
	if rm.shouldSkipDueToCircuitBreaker(platform) {
		return nil, errors.NewPlatformError(errors.ErrPlatformUnavailable, platform, "circuit breaker open")
	}

	for attempt := 0; attempt < rm.strategy.GetMaxAttempts(); attempt++ {
		// Check context cancellation before each attempt
		select {
		case <-ctx.Done():
			rm.recordAttempt(platform, time.Since(startTime), lastErr, false)
			return lastReceipt, errors.NewSystemError(errors.ErrSystemTimeout, "retry_middleware", "context cancelled during retry")
		default:
		}

		// Execute the operation
		attemptStart := time.Now()
		receipt, err := next(ctx, msg, targets)
		attemptDuration := time.Since(attemptStart)

		// If successful, return immediately
		if err == nil {
			rm.recordAttempt(platform, attemptDuration, nil, true)
			rm.recordCircuitBreakerSuccess(platform)
			if attempt > 0 {
				rm.logger.Info("Message send successful after retries",
					"message_id", msg.ID,
					"platform", platform,
					"attempts", attempt+1,
					"total_duration", time.Since(startTime))
			}
			return receipt, nil
		}

		// Store the error for potential return
		lastErr = err
		lastReceipt = receipt
		rm.recordCircuitBreakerFailure(platform)

		// Check if we should retry
		if !rm.strategy.ShouldRetry(ctx, err, attempt) {
			rm.logger.Debug("Error not retryable or max attempts reached",
				"message_id", msg.ID,
				"platform", platform,
				"attempt", attempt+1,
				"error", err.Error())
			break
		}

		// Calculate and apply delay before next retry
		delay := rm.strategy.GetDelay(attempt)

		rm.logger.Warn("Message send failed, retrying",
			"message_id", msg.ID,
			"platform", platform,
			"attempt", attempt+1,
			"max_attempts", rm.strategy.GetMaxAttempts(),
			"delay", delay,
			"error", err.Error())

		// Record failed attempt
		rm.recordAttempt(platform, attemptDuration, err, false)

		// Wait for delay or context cancellation
		select {
		case <-ctx.Done():
			return lastReceipt, errors.NewSystemError(errors.ErrSystemTimeout, "retry_middleware", "context cancelled during retry delay")
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retry attempts exhausted
	totalDuration := time.Since(startTime)
	rm.recordAttempt(platform, totalDuration, lastErr, false)

	rm.logger.Error("Message send failed after all retry attempts",
		"message_id", msg.ID,
		"platform", platform,
		"attempts", rm.strategy.GetMaxAttempts(),
		"total_duration", totalDuration,
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

// Helper methods for RetryMiddleware

// extractPlatformFromTargets extracts the primary platform from targets
func (rm *RetryMiddleware) extractPlatformFromTargets(targets []target.Target) string {
	if len(targets) == 0 {
		return "unknown"
	}

	// Return the platform of the first target
	return targets[0].Platform
}

// shouldSkipDueToCircuitBreaker checks if operation should be skipped due to circuit breaker
func (rm *RetryMiddleware) shouldSkipDueToCircuitBreaker(platform string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if cb, exists := rm.circuitBreakers[platform]; exists {
		return !cb.ShouldAllow()
	}
	return false
}

// recordAttempt records a retry attempt with performance monitoring
func (rm *RetryMiddleware) recordAttempt(platform string, duration time.Duration, err error, success bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Record in statistics
	rm.statistics.RecordAttempt(platform, duration, err, success)

	// Record in performance monitor
	rm.performanceMonitor.RecordOperation(platform, duration, success)
}

// recordCircuitBreakerSuccess records a successful operation for circuit breaker
func (rm *RetryMiddleware) recordCircuitBreakerSuccess(platform string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if cb, exists := rm.circuitBreakers[platform]; exists {
		cb.RecordSuccess()
	}
}

// recordCircuitBreakerFailure records a failed operation for circuit breaker
func (rm *RetryMiddleware) recordCircuitBreakerFailure(platform string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Create circuit breaker if it doesn't exist
	if _, exists := rm.circuitBreakers[platform]; !exists {
		// Default circuit breaker configuration: 5 failures, 30 second timeout
		rm.circuitBreakers[platform] = errors.NewCircuitBreaker(5, 30*time.Second)
	}

	rm.circuitBreakers[platform].RecordFailure()
}

// GetStatistics returns retry statistics
func (rm *RetryMiddleware) GetStatistics() *errors.RetryStatistics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.statistics
}

// GetPerformanceMetrics returns performance metrics
func (rm *RetryMiddleware) GetPerformanceMetrics() *errors.PerformanceMonitor {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.performanceMonitor
}

// GetCircuitBreakerStatus returns circuit breaker status for a platform
func (rm *RetryMiddleware) GetCircuitBreakerStatus(platform string) errors.CircuitBreakerState {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if cb, exists := rm.circuitBreakers[platform]; exists {
		return cb.GetState()
	}
	return errors.CircuitClosed
}

// OptimizePlatformConfig optimizes retry configuration for a platform based on performance
func (rm *RetryMiddleware) OptimizePlatformConfig(platform string, currentConfig *errors.PlatformRetryConfig) *errors.PlatformRetryConfig {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.performanceMonitor.OptimizeStrategy(platform, currentConfig)
}

// GetRecommendations returns performance-based recommendations
func (rm *RetryMiddleware) GetRecommendations() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.performanceMonitor.GetRecommendations()
}

// Reset resets all retry statistics and circuit breakers
func (rm *RetryMiddleware) Reset() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.statistics.Reset()
	rm.performanceMonitor.Reset()

	// Reset all circuit breakers
	for _, cb := range rm.circuitBreakers {
		cb.Reset()
	}
}
