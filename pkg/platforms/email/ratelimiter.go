// Package email provides rate limiting functionality for email sending
package email

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu           sync.Mutex
	tokens       int           // Current number of tokens
	capacity     int           // Maximum number of tokens (burst limit)
	refillRate   int           // Tokens added per refill interval
	refillPeriod time.Duration // How often to refill tokens
	lastRefill   time.Time     // Last time tokens were refilled
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, burst int, window time.Duration) *RateLimiter {
	if window == 0 {
		window = time.Minute // Default to 1 minute window
	}

	if burst == 0 {
		burst = rate // Default burst equals rate
	}

	return &RateLimiter{
		tokens:       burst,
		capacity:     burst,
		refillRate:   rate,
		refillPeriod: window,
		lastRefill:   time.Now(),
	}
}

// Allow checks if an operation is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.refill(now)

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// AllowN checks if N operations are allowed under the rate limit
func (rl *RateLimiter) AllowN(n int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.refill(now)

	if rl.tokens >= n {
		rl.tokens -= n
		return true
	}

	return false
}

// refill adds tokens based on elapsed time
func (rl *RateLimiter) refill(now time.Time) {
	elapsed := now.Sub(rl.lastRefill)
	if elapsed < rl.refillPeriod {
		return
	}

	// Calculate how many refill periods have passed
	periods := int(elapsed / rl.refillPeriod)
	tokensToAdd := periods * rl.refillRate

	// Add tokens up to capacity
	rl.tokens += tokensToAdd
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}

	// Update last refill time
	rl.lastRefill = rl.lastRefill.Add(time.Duration(periods) * rl.refillPeriod)
}

// Tokens returns the current number of available tokens
func (rl *RateLimiter) Tokens() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.refill(now)
	return rl.tokens
}

// Wait blocks until an operation is allowed
func (rl *RateLimiter) Wait() {
	for !rl.Allow() {
		time.Sleep(time.Millisecond * 100)
	}
}

// TimeToRefill returns the time until the next refill
func (rl *RateLimiter) TimeToRefill() time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	nextRefill := rl.lastRefill.Add(rl.refillPeriod)
	return time.Until(nextRefill)
}

// Stats returns rate limiter statistics
func (rl *RateLimiter) Stats() RateLimiterStats {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.refill(now)

	// Calculate time to refill without calling TimeToRefill() to avoid deadlock
	nextRefill := rl.lastRefill.Add(rl.refillPeriod)
	timeToRefill := time.Until(nextRefill)

	return RateLimiterStats{
		CurrentTokens: rl.tokens,
		Capacity:      rl.capacity,
		RefillRate:    rl.refillRate,
		RefillPeriod:  rl.refillPeriod,
		TimeToRefill:  timeToRefill,
		LastRefill:    rl.lastRefill,
	}
}

// RateLimiterStats represents rate limiter statistics
type RateLimiterStats struct {
	CurrentTokens int           `json:"current_tokens"`
	Capacity      int           `json:"capacity"`
	RefillRate    int           `json:"refill_rate"`
	RefillPeriod  time.Duration `json:"refill_period"`
	TimeToRefill  time.Duration `json:"time_to_refill"`
	LastRefill    time.Time     `json:"last_refill"`
}
