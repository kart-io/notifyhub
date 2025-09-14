package internal

import (
	"context"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	capacity   int           // Maximum number of tokens
	tokens     int           // Current number of tokens
	refillRate int           // Tokens added per interval
	interval   time.Duration // Refill interval
	lastRefill time.Time     // Last refill time
	mu         sync.Mutex    // Protects token operations
}

// NewTokenBucket creates a new token bucket rate limiter
// capacity: maximum number of tokens in the bucket
// refillRate: number of tokens to add per interval
// interval: how often to add tokens
func NewTokenBucket(capacity, refillRate int, interval time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity, // Start with full bucket
		refillRate: refillRate,
		interval:   interval,
		lastRefill: time.Now(),
	}
}

// TryConsume attempts to consume a token from the bucket
// Returns true if successful, false if not enough tokens
func (tb *TokenBucket) TryConsume() bool {
	return tb.TryConsumeN(1)
}

// TryConsumeN attempts to consume n tokens from the bucket
func (tb *TokenBucket) TryConsumeN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}
	return false
}

// ConsumeWithTimeout attempts to consume a token with timeout
func (tb *TokenBucket) ConsumeWithTimeout(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if tb.TryConsume() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return context.DeadlineExceeded
		}

		// Wait a bit before trying again
		time.Sleep(10 * time.Millisecond)
	}
}

// AvailableTokens returns the current number of available tokens
func (tb *TokenBucket) AvailableTokens() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	return tb.tokens
}

// refill adds tokens based on elapsed time (must be called with lock held)
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	if elapsed >= tb.interval {
		intervals := int(elapsed / tb.interval)
		tokensToAdd := intervals * tb.refillRate

		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}

		tb.lastRefill = tb.lastRefill.Add(time.Duration(intervals) * tb.interval)
	}
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// TryConsume attempts to consume a token, returns true if successful
	TryConsume() bool
	// ConsumeWithTimeout attempts to consume with timeout
	ConsumeWithTimeout(ctx context.Context, timeout time.Duration) error
	// AvailableTokens returns current available tokens
	AvailableTokens() int
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// RequestsPerSecond is the maximum requests per second
	RequestsPerSecond int `json:"requests_per_second" yaml:"requests_per_second"`
	// BurstSize is the maximum burst capacity
	BurstSize int `json:"burst_size" yaml:"burst_size"`
	// Enabled controls whether rate limiting is active
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// NewRateLimiter creates a rate limiter from config
func NewRateLimiter(config *RateLimitConfig) RateLimiter {
	if config == nil || !config.Enabled || config.RequestsPerSecond <= 0 {
		return &NoOpRateLimiter{}
	}

	capacity := config.BurstSize
	if capacity <= 0 {
		capacity = config.RequestsPerSecond // Default burst = rate
	}

	return NewTokenBucket(capacity, config.RequestsPerSecond, time.Second)
}

// NoOpRateLimiter is a rate limiter that never limits
type NoOpRateLimiter struct{}

func (n *NoOpRateLimiter) TryConsume() bool { return true }
func (n *NoOpRateLimiter) ConsumeWithTimeout(ctx context.Context, timeout time.Duration) error {
	return nil
}
func (n *NoOpRateLimiter) AvailableTokens() int { return 1 }

// EnsureRateLimiter wraps a function call with rate limiting
func EnsureRateLimiter(limiter RateLimiter, ctx context.Context, timeout time.Duration, fn func() error) error {
	if err := limiter.ConsumeWithTimeout(ctx, timeout); err != nil {
		return err
	}
	return fn()
}