package retry

import (
	"math/rand"
	"time"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialInterval time.Duration `json:"initial_interval"`
	Multiplier      float64       `json:"multiplier"`
	MaxJitter       time.Duration `json:"max_jitter"` // Maximum jitter to add
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:      3,
		InitialInterval: 30 * time.Second,
		Multiplier:      2.0,
		MaxJitter:       5 * time.Second, // Add up to 5 seconds jitter
	}
}

// NextRetry calculates the next retry time based on attempt count
func (p *RetryPolicy) NextRetry(attempts int) time.Time {
	if attempts >= p.MaxRetries {
		return time.Time{} // No more retries
	}

	// Calculate base interval with exponential backoff
	interval := p.InitialInterval
	for i := 0; i < attempts; i++ {
		interval = time.Duration(float64(interval) * p.Multiplier)
	}

	// Add jitter to prevent thundering herd problem
	if p.MaxJitter > 0 {
		jitter := time.Duration(rand.Int63n(int64(p.MaxJitter)))
		interval += jitter
	}

	return time.Now().Add(interval)
}

// ShouldRetry determines if a message should be retried
func (p *RetryPolicy) ShouldRetry(attempts int) bool {
	return attempts < p.MaxRetries
}

// ExponentialBackoffPolicy creates a retry policy with exponential backoff
func ExponentialBackoffPolicy(maxRetries int, initialInterval time.Duration, multiplier float64) *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:      maxRetries,
		InitialInterval: initialInterval,
		Multiplier:      multiplier,
		MaxJitter:       initialInterval / 4, // Default jitter is 25% of initial interval
	}
}

// LinearBackoffPolicy creates a retry policy with linear backoff
func LinearBackoffPolicy(maxRetries int, interval time.Duration) *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:      maxRetries,
		InitialInterval: interval,
		Multiplier:      1.0,          // Linear means no multiplication
		MaxJitter:       interval / 6, // Smaller jitter for linear backoff
	}
}

// NoRetryPolicy creates a policy that disables retries
func NoRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:      0,
		InitialInterval: 0,
		Multiplier:      1.0,
		MaxJitter:       0, // No jitter needed for no retries
	}
}

// AggressiveRetryPolicy creates a retry policy for urgent messages
func AggressiveRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:      5,
		InitialInterval: 10 * time.Second,
		Multiplier:      1.5,
		MaxJitter:       2 * time.Second, // Small jitter for urgent messages
	}
}
