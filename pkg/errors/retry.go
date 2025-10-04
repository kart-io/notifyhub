// Package errors provides retry logic and recovery mechanisms
package errors

import (
	"context"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/kart/notifyhub/pkg/utils/logger"
)

// RetryPolicy defines how operations should be retried
type RetryPolicy interface {
	// ShouldRetry determines if an error should be retried
	ShouldRetry(err error, attempt int) bool

	// RetryDelay calculates the delay before the next retry
	RetryDelay(attempt int) time.Duration

	// MaxAttempts returns the maximum number of retry attempts
	MaxAttempts() int
}

// ExponentialBackoffPolicy implements exponential backoff with jitter
type ExponentialBackoffPolicy struct {
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       float64
	MaxAttempts_ int
}

// NewExponentialBackoffPolicy creates a new exponential backoff policy
func NewExponentialBackoffPolicy(baseDelay, maxDelay time.Duration, maxAttempts int) *ExponentialBackoffPolicy {
	return &ExponentialBackoffPolicy{
		BaseDelay:    baseDelay,
		MaxDelay:     maxDelay,
		Multiplier:   2.0,
		Jitter:       0.1,
		MaxAttempts_: maxAttempts,
	}
}

// ShouldRetry determines if an error should be retried
func (p *ExponentialBackoffPolicy) ShouldRetry(err error, attempt int) bool {
	if attempt >= p.MaxAttempts_ {
		return false
	}

	// Check if error is retryable
	if nhErr, ok := err.(*NotifyError); ok {
		return nhErr.Retryable
	}

	// Default: retry for certain error types
	return isRetryableError(err)
}

// RetryDelay calculates the delay before the next retry
func (p *ExponentialBackoffPolicy) RetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate exponential delay
	delay := float64(p.BaseDelay) * math.Pow(p.Multiplier, float64(attempt-1))

	// Apply jitter
	if p.Jitter > 0 {
		jitter := delay * p.Jitter * (rand.Float64()*2 - 1)
		delay += jitter
	}

	// Cap at max delay
	if time.Duration(delay) > p.MaxDelay {
		delay = float64(p.MaxDelay)
	}

	return time.Duration(delay)
}

// MaxAttempts returns the maximum number of retry attempts
func (p *ExponentialBackoffPolicy) MaxAttempts() int {
	return p.MaxAttempts_
}

// LinearBackoffPolicy implements linear backoff
type LinearBackoffPolicy struct {
	BaseDelay    time.Duration
	Increment    time.Duration
	MaxDelay     time.Duration
	MaxAttempts_ int
}

// NewLinearBackoffPolicy creates a new linear backoff policy
func NewLinearBackoffPolicy(baseDelay, increment, maxDelay time.Duration, maxAttempts int) *LinearBackoffPolicy {
	return &LinearBackoffPolicy{
		BaseDelay:    baseDelay,
		Increment:    increment,
		MaxDelay:     maxDelay,
		MaxAttempts_: maxAttempts,
	}
}

// ShouldRetry determines if an error should be retried
func (p *LinearBackoffPolicy) ShouldRetry(err error, attempt int) bool {
	if attempt >= p.MaxAttempts_ {
		return false
	}

	if nhErr, ok := err.(*NotifyError); ok {
		return nhErr.IsRetryable()
	}

	return isRetryableError(err)
}

// RetryDelay calculates the delay before the next retry
func (p *LinearBackoffPolicy) RetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	delay := p.BaseDelay + time.Duration(attempt-1)*p.Increment
	if delay > p.MaxDelay {
		delay = p.MaxDelay
	}

	return delay
}

// MaxAttempts returns the maximum number of retry attempts
func (p *LinearBackoffPolicy) MaxAttempts() int {
	return p.MaxAttempts_
}

// FixedDelayPolicy implements fixed delay between retries
type FixedDelayPolicy struct {
	Delay        time.Duration
	MaxAttempts_ int
}

// NewFixedDelayPolicy creates a new fixed delay policy
func NewFixedDelayPolicy(delay time.Duration, maxAttempts int) *FixedDelayPolicy {
	return &FixedDelayPolicy{
		Delay:        delay,
		MaxAttempts_: maxAttempts,
	}
}

// ShouldRetry determines if an error should be retried
func (p *FixedDelayPolicy) ShouldRetry(err error, attempt int) bool {
	if attempt >= p.MaxAttempts_ {
		return false
	}

	if nhErr, ok := err.(*NotifyError); ok {
		return nhErr.IsRetryable()
	}

	return isRetryableError(err)
}

// RetryDelay calculates the delay before the next retry
func (p *FixedDelayPolicy) RetryDelay(attempt int) time.Duration {
	return p.Delay
}

// MaxAttempts returns the maximum number of retry attempts
func (p *FixedDelayPolicy) MaxAttempts() int {
	return p.MaxAttempts_
}

// RetryExecutor handles the execution of retryable operations
type RetryExecutor struct {
	policy RetryPolicy
	logger logger.Logger
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(policy RetryPolicy, logger logger.Logger) *RetryExecutor {
	return &RetryExecutor{
		policy: policy,
		logger: logger,
	}
}

// Execute executes an operation with retry logic
func (r *RetryExecutor) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= r.policy.MaxAttempts(); attempt++ {
		// Execute the operation
		err := operation()
		if err == nil {
			// Success
			if attempt > 1 {
				r.logger.Info("Operation succeeded after retry",
					"attempt", attempt,
					"total_attempts", r.policy.MaxAttempts())
			}
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !r.policy.ShouldRetry(err, attempt) {
			r.logger.Warn("Operation failed, not retryable",
				"error", err.Error(),
				"attempt", attempt)
			break
		}

		// Check if we've reached max attempts
		if attempt >= r.policy.MaxAttempts() {
			r.logger.Error("Operation failed after max attempts",
				"error", err.Error(),
				"attempts", attempt,
				"max_attempts", r.policy.MaxAttempts())
			break
		}

		// Calculate delay for next attempt
		delay := r.policy.RetryDelay(attempt)
		r.logger.Debug("Operation failed, retrying",
			"error", err.Error(),
			"attempt", attempt,
			"next_delay", delay,
			"max_attempts", r.policy.MaxAttempts())

		// Wait for the specified delay, but respect context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

// ExecuteWithResult executes an operation with retry logic and returns result
func (r *RetryExecutor) ExecuteWithResult(ctx context.Context, operation func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	var result interface{}

	for attempt := 1; attempt <= r.policy.MaxAttempts(); attempt++ {
		// Execute the operation
		res, err := operation()
		if err == nil {
			// Success
			if attempt > 1 {
				r.logger.Info("Operation succeeded after retry",
					"attempt", attempt,
					"total_attempts", r.policy.MaxAttempts())
			}
			return res, nil
		}

		lastErr = err

		// Check if we should retry
		if !r.policy.ShouldRetry(err, attempt) {
			r.logger.Warn("Operation failed, not retryable",
				"error", err.Error(),
				"attempt", attempt)
			break
		}

		// Check if we've reached max attempts
		if attempt >= r.policy.MaxAttempts() {
			r.logger.Error("Operation failed after max attempts",
				"error", err.Error(),
				"attempts", attempt,
				"max_attempts", r.policy.MaxAttempts())
			break
		}

		// Calculate delay for next attempt
		delay := r.policy.RetryDelay(attempt)
		r.logger.Debug("Operation failed, retrying",
			"error", err.Error(),
			"attempt", attempt,
			"next_delay", delay,
			"max_attempts", r.policy.MaxAttempts())

		// Wait for the specified delay, but respect context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return result, lastErr
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name         string
	maxFailures  int
	resetTimeout time.Duration
	state        CircuitState
	failures     int
	lastFailTime time.Time
	mutex        *sync.RWMutex
	logger       logger.Logger
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation
	CircuitOpen                         // Circuit is open, rejecting calls
	CircuitHalfOpen                     // Testing if service has recovered
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, maxFailures int, resetTimeout time.Duration, logger logger.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		name:         name,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
		failures:     0,
		mutex:        &sync.RWMutex{},
		logger:       logger,
	}
}

// Execute executes an operation through the circuit breaker
func (cb *CircuitBreaker) Execute(operation func() error) error {
	// Check if circuit should allow the call
	if !cb.allowCall() {
		return New(ErrResourceExhausted, "Circuit breaker is open").
			WithContext("circuit_name: " + cb.name)
	}

	// Execute the operation
	err := operation()

	// Update circuit breaker state based on result
	cb.recordResult(err)

	return err
}

// allowCall determines if a call should be allowed
func (cb *CircuitBreaker) allowCall() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if reset timeout has passed
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.logger.Info("Circuit breaker transitioning to half-open",
				"circuit_name", cb.name)
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult records the result of an operation and updates circuit state
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.state == CircuitHalfOpen {
			// Failure in half-open state, go back to open
			cb.state = CircuitOpen
			cb.logger.Warn("Circuit breaker opened from half-open state",
				"circuit_name", cb.name,
				"error", err.Error())
		} else if cb.failures >= cb.maxFailures {
			// Too many failures, open the circuit
			cb.state = CircuitOpen
			cb.logger.Error("Circuit breaker opened due to failures",
				"circuit_name", cb.name,
				"failures", cb.failures,
				"max_failures", cb.maxFailures)
		}
	} else {
		// Success
		switch cb.state {
		case CircuitHalfOpen:
			// Success in half-open state, close the circuit
			cb.state = CircuitClosed
			cb.failures = 0
			cb.logger.Info("Circuit breaker closed after successful test",
				"circuit_name", cb.name)
		case CircuitClosed:
			// Reset failure count on success
			cb.failures = 0
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.failures
}

// isRetryableError determines if a standard error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common retryable error patterns
	errStr := err.Error()
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"network",
		"temporary failure",
		"service unavailable",
		"too many requests",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}

// DefaultRetryPolicy returns a default retry policy suitable for most operations
func DefaultRetryPolicy() RetryPolicy {
	return NewExponentialBackoffPolicy(
		100*time.Millisecond, // Base delay
		30*time.Second,       // Max delay
		3,                    // Max attempts
	)
}

// AggressiveRetryPolicy returns a retry policy with more attempts and longer delays
func AggressiveRetryPolicy() RetryPolicy {
	return NewExponentialBackoffPolicy(
		500*time.Millisecond, // Base delay
		2*time.Minute,        // Max delay
		5,                    // Max attempts
	)
}

// ConservativeRetryPolicy returns a retry policy with fewer attempts and shorter delays
func ConservativeRetryPolicy() RetryPolicy {
	return NewFixedDelayPolicy(
		1*time.Second, // Fixed delay
		2,             // Max attempts
	)
}
