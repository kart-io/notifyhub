// Package middleware provides rate limiting middleware for NotifyHub
package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/errors"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// RateLimitMiddleware provides rate limiting for message operations
type RateLimitMiddleware struct {
	BaseMiddleware
	limiter RateLimiter
	logger  logger.Logger
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// Allow checks if a request should be allowed
	Allow(ctx context.Context, key string) (bool, error)

	// GetLimit returns the current limit for a key
	GetLimit(key string) (limit int, window time.Duration)

	// GetUsage returns current usage for a key
	GetUsage(key string) (used int, remaining int, resetTime time.Time)
}

// TokenBucketLimiter implements token bucket algorithm
type TokenBucketLimiter struct {
	buckets      map[string]*TokenBucket
	maxTokens    int
	refillRate   time.Duration
	keyExtractor KeyExtractor
	mutex        sync.RWMutex
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mutex      sync.Mutex
}

// KeyExtractor defines how to extract rate limiting keys from requests
type KeyExtractor func(ctx context.Context, msg *message.Message, targets []target.Target) string

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(maxTokens int, refillRate time.Duration, keyExtractor KeyExtractor) *TokenBucketLimiter {
	if keyExtractor == nil {
		keyExtractor = DefaultKeyExtractor
	}

	return &TokenBucketLimiter{
		buckets:      make(map[string]*TokenBucket),
		maxTokens:    maxTokens,
		refillRate:   refillRate,
		keyExtractor: keyExtractor,
	}
}

// Allow checks if a request should be allowed
func (tbl *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	tbl.mutex.Lock()
	bucket, exists := tbl.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			tokens:     tbl.maxTokens,
			maxTokens:  tbl.maxTokens,
			refillRate: tbl.refillRate,
			lastRefill: time.Now(),
		}
		tbl.buckets[key] = bucket
	}
	tbl.mutex.Unlock()

	return bucket.takeToken(), nil
}

// takeToken attempts to take a token from the bucket
func (tb *TokenBucket) takeToken() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	if elapsed >= tb.refillRate {
		tokensToAdd := int(elapsed / tb.refillRate)
		tb.tokens = min(tb.maxTokens, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}

	// Try to take a token
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// GetLimit returns the current limit for a key
func (tbl *TokenBucketLimiter) GetLimit(key string) (limit int, window time.Duration) {
	return tbl.maxTokens, tbl.refillRate
}

// GetUsage returns current usage for a key
func (tbl *TokenBucketLimiter) GetUsage(key string) (used int, remaining int, resetTime time.Time) {
	tbl.mutex.RLock()
	bucket, exists := tbl.buckets[key]
	tbl.mutex.RUnlock()

	if !exists {
		return 0, tbl.maxTokens, time.Now().Add(tbl.refillRate)
	}

	bucket.mutex.Lock()
	remaining = bucket.tokens
	used = tbl.maxTokens - remaining
	resetTime = bucket.lastRefill.Add(tbl.refillRate)
	bucket.mutex.Unlock()

	return used, remaining, resetTime
}

// SlidingWindowLimiter implements sliding window algorithm
type SlidingWindowLimiter struct {
	windows      map[string]*SlidingWindow
	maxRequests  int
	windowSize   time.Duration
	keyExtractor KeyExtractor
	mutex        sync.RWMutex
}

// SlidingWindow represents a sliding window for rate limiting
type SlidingWindow struct {
	requests    []time.Time
	windowSize  time.Duration
	maxRequests int
	mutex       sync.Mutex
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(maxRequests int, windowSize time.Duration, keyExtractor KeyExtractor) *SlidingWindowLimiter {
	if keyExtractor == nil {
		keyExtractor = DefaultKeyExtractor
	}

	return &SlidingWindowLimiter{
		windows:      make(map[string]*SlidingWindow),
		maxRequests:  maxRequests,
		windowSize:   windowSize,
		keyExtractor: keyExtractor,
	}
}

// Allow checks if a request should be allowed
func (swl *SlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	swl.mutex.Lock()
	window, exists := swl.windows[key]
	if !exists {
		window = &SlidingWindow{
			requests:    make([]time.Time, 0, swl.maxRequests*2),
			windowSize:  swl.windowSize,
			maxRequests: swl.maxRequests,
		}
		swl.windows[key] = window
	}
	swl.mutex.Unlock()

	return window.allowRequest(), nil
}

// allowRequest checks if a request can be allowed in the sliding window
func (sw *SlidingWindow) allowRequest() bool {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.windowSize)

	// Remove requests outside the current window
	validRequests := make([]time.Time, 0, len(sw.requests))
	for _, reqTime := range sw.requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	sw.requests = validRequests

	// Check if we can allow this request
	if len(sw.requests) < sw.maxRequests {
		sw.requests = append(sw.requests, now)
		return true
	}

	return false
}

// GetLimit returns the current limit for a key
func (swl *SlidingWindowLimiter) GetLimit(key string) (limit int, window time.Duration) {
	return swl.maxRequests, swl.windowSize
}

// GetUsage returns current usage for a key
func (swl *SlidingWindowLimiter) GetUsage(key string) (used int, remaining int, resetTime time.Time) {
	swl.mutex.RLock()
	window, exists := swl.windows[key]
	swl.mutex.RUnlock()

	if !exists {
		return 0, swl.maxRequests, time.Now().Add(swl.windowSize)
	}

	window.mutex.Lock()
	now := time.Now()
	windowStart := now.Add(-swl.windowSize)

	// Count valid requests
	validCount := 0
	for _, reqTime := range window.requests {
		if reqTime.After(windowStart) {
			validCount++
		}
	}

	used = validCount
	remaining = swl.maxRequests - used
	resetTime = now.Add(swl.windowSize)
	window.mutex.Unlock()

	return used, remaining, resetTime
}

// Default key extractors

// DefaultKeyExtractor uses a global key for rate limiting
func DefaultKeyExtractor(ctx context.Context, msg *message.Message, targets []target.Target) string {
	return "global"
}

// MessageIDKeyExtractor uses message ID for per-message rate limiting
func MessageIDKeyExtractor(ctx context.Context, msg *message.Message, targets []target.Target) string {
	return msg.ID
}

// FormatKeyExtractor uses message format for per-format rate limiting
func FormatKeyExtractor(ctx context.Context, msg *message.Message, targets []target.Target) string {
	return string(msg.Format)
}

// TargetTypeKeyExtractor uses target type for per-platform rate limiting
func TargetTypeKeyExtractor(ctx context.Context, msg *message.Message, targets []target.Target) string {
	if len(targets) > 0 {
		return targets[0].Type // Use first target type
	}
	return "unknown"
}

// CompositeKeyExtractor combines multiple key components
func CompositeKeyExtractor(extractors ...KeyExtractor) KeyExtractor {
	return func(ctx context.Context, msg *message.Message, targets []target.Target) string {
		var key string
		for i, extractor := range extractors {
			if i > 0 {
				key += ":"
			}
			key += extractor(ctx, msg, targets)
		}
		return key
	}
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(limiter RateLimiter, l logger.Logger) *RateLimitMiddleware {
	if l == nil {
		l = logger.Discard
	}

	return &RateLimitMiddleware{
		BaseMiddleware: NewBaseMiddleware("ratelimit"),
		limiter:        limiter,
		logger:         l,
	}
}

// HandleSend implements rate limiting for message operations
func (rlm *RateLimitMiddleware) HandleSend(ctx context.Context, msg *message.Message, targets []target.Target, next SendHandler) (*receipt.Receipt, error) {
	// Extract rate limiting key
	key := ""
	if tbl, ok := rlm.limiter.(*TokenBucketLimiter); ok {
		key = tbl.keyExtractor(ctx, msg, targets)
	} else if swl, ok := rlm.limiter.(*SlidingWindowLimiter); ok {
		key = swl.keyExtractor(ctx, msg, targets)
	} else {
		key = "global" // fallback
	}

	// Check rate limit
	allowed, err := rlm.limiter.Allow(ctx, key)
	if err != nil {
		rlm.logger.Error("Rate limiter error",
			"message_id", msg.ID,
			"key", key,
			"error", err.Error())
		return nil, errors.NewSystemError(errors.ErrResourceExhausted, "rate_limiter", err.Error())
	}

	if !allowed {
		used, remaining, resetTime := rlm.limiter.GetUsage(key)
		limit, window := rlm.limiter.GetLimit(key)

		rlm.logger.Warn("Rate limit exceeded",
			"message_id", msg.ID,
			"key", key,
			"limit", limit,
			"window", window,
			"used", used,
			"remaining", remaining,
			"reset_time", resetTime.Format(time.RFC3339))

		// Create a rate limit error with detailed information
		return nil, errors.NewPlatformError(
			errors.ErrPlatformRateLimit,
			"rate_limiter",
			fmt.Sprintf("rate limit exceeded: %d/%d requests used, resets at %s",
				used, limit, resetTime.Format(time.RFC3339)))
	}

	// Log successful rate limit check
	used, remaining, resetTime := rlm.limiter.GetUsage(key)
	rlm.logger.Debug("Rate limit check passed",
		"message_id", msg.ID,
		"key", key,
		"used", used,
		"remaining", remaining,
		"reset_time", resetTime.Format(time.RFC3339))

	// Proceed with the request
	return next(ctx, msg, targets)
}

// GetLimiterStats returns current rate limiter statistics
func (rlm *RateLimitMiddleware) GetLimiterStats(key string) map[string]interface{} {
	used, remaining, resetTime := rlm.limiter.GetUsage(key)
	limit, window := rlm.limiter.GetLimit(key)

	return map[string]interface{}{
		"key":        key,
		"limit":      limit,
		"window":     window.String(),
		"used":       used,
		"remaining":  remaining,
		"reset_time": resetTime.Format(time.RFC3339),
	}
}

// Utility function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
