// Package ratelimit provides sliding window rate limiter implementation
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// SlidingWindowLimiter implements a sliding window rate limiter
type SlidingWindowLimiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	requests  []time.Time
	clock     Clock
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return NewSlidingWindowLimiterWithClock(limit, window, SystemClock{})
}

// NewSlidingWindowLimiterWithClock creates a new sliding window limiter with custom clock
func NewSlidingWindowLimiterWithClock(limit int, window time.Duration, clock Clock) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		limit:    limit,
		window:   window,
		requests: make([]time.Time, 0, limit),
		clock:    clock,
	}
}

// Allow returns true if a request is allowed under the rate limit
func (swl *SlidingWindowLimiter) Allow() bool {
	return swl.AllowN(1)
}

// AllowN returns true if n requests are allowed under the rate limit
func (swl *SlidingWindowLimiter) AllowN(n int) bool {
	if n <= 0 {
		return true
	}

	swl.mu.Lock()
	defer swl.mu.Unlock()

	now := swl.clock.Now()
	swl.cleanExpiredRequests(now)

	if len(swl.requests)+n > swl.limit {
		return false
	}

	// Add n requests
	for i := 0; i < n; i++ {
		swl.requests = append(swl.requests, now)
	}

	return true
}

// Wait blocks until a request is allowed or context is cancelled
func (swl *SlidingWindowLimiter) Wait(ctx context.Context) error {
	return swl.WaitN(ctx, 1)
}

// WaitN blocks until n requests are allowed or context is cancelled
func (swl *SlidingWindowLimiter) WaitN(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}

	for {
		if swl.AllowN(n) {
			return nil
		}

		// Calculate wait time until the oldest request expires
		swl.mu.Lock()
		waitTime := swl.calculateWaitTime(n)
		swl.mu.Unlock()

		if waitTime <= 0 {
			continue // Try again immediately
		}

		timer := time.NewTimer(waitTime)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Continue the loop to try again
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Reserve reserves tokens and returns a reservation (simplified implementation)
func (swl *SlidingWindowLimiter) Reserve() Reservation {
	return swl.ReserveN(1)
}

// ReserveN reserves n tokens and returns a reservation (simplified implementation)
func (swl *SlidingWindowLimiter) ReserveN(n int) Reservation {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	now := swl.clock.Now()
	swl.cleanExpiredRequests(now)

	if len(swl.requests)+n > swl.limit {
		waitTime := swl.calculateWaitTime(n)
		return Reservation{
			ok:        waitTime < time.Hour, // Don't wait more than an hour
			delay:     waitTime,
			tokens:    n,
			timeToAct: now.Add(waitTime),
		}
	}

	// Add n requests
	for i := 0; i < n; i++ {
		swl.requests = append(swl.requests, now)
	}

	return Reservation{
		ok:        true,
		delay:     0,
		tokens:    n,
		timeToAct: now,
	}
}

// Limit returns the rate limit (converted to Rate)
func (swl *SlidingWindowLimiter) Limit() Rate {
	swl.mu.Lock()
	defer swl.mu.Unlock()
	return Rate(float64(swl.limit) / swl.window.Seconds())
}

// Burst returns the burst size (same as limit for sliding window)
func (swl *SlidingWindowLimiter) Burst() int {
	swl.mu.Lock()
	defer swl.mu.Unlock()
	return swl.limit
}

// SetLimit changes the rate limit
func (swl *SlidingWindowLimiter) SetLimit(newLimit int) {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	swl.limit = newLimit

	// If new limit is smaller, trim the requests slice
	if len(swl.requests) > newLimit {
		swl.requests = swl.requests[len(swl.requests)-newLimit:]
	}
}

// SetWindow changes the time window
func (swl *SlidingWindowLimiter) SetWindow(newWindow time.Duration) {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	swl.window = newWindow
	now := swl.clock.Now()
	swl.cleanExpiredRequests(now)
}

// cleanExpiredRequests removes requests outside the time window
func (swl *SlidingWindowLimiter) cleanExpiredRequests(now time.Time) {
	cutoff := now.Add(-swl.window)

	// Find the first request within the window
	validIndex := 0
	for i, req := range swl.requests {
		if req.After(cutoff) {
			validIndex = i
			break
		}
		if i == len(swl.requests)-1 {
			validIndex = len(swl.requests) // All requests are expired
		}
	}

	// Keep only the valid requests
	if validIndex > 0 {
		copy(swl.requests, swl.requests[validIndex:])
		swl.requests = swl.requests[:len(swl.requests)-validIndex]
	}
}

// calculateWaitTime calculates how long to wait for n requests to be allowed
func (swl *SlidingWindowLimiter) calculateWaitTime(n int) time.Duration {
	if len(swl.requests)+n <= swl.limit {
		return 0
	}

	// Find when enough requests will expire to allow n new requests
	needed := len(swl.requests) + n - swl.limit
	if needed > len(swl.requests) {
		needed = len(swl.requests)
	}

	if needed <= 0 {
		return 0
	}

	// The wait time is until the needed-th oldest request expires
	oldestToExpire := swl.requests[needed-1]
	now := swl.clock.Now()
	waitTime := oldestToExpire.Add(swl.window).Sub(now)

	if waitTime <= 0 {
		return 0
	}

	return waitTime
}

// RequestCount returns the current number of requests in the window
func (swl *SlidingWindowLimiter) RequestCount() int {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	now := swl.clock.Now()
	swl.cleanExpiredRequests(now)
	return len(swl.requests)
}

// WindowInfo returns information about the sliding window
type WindowInfo struct {
	Limit           int           `json:"limit"`
	Window          time.Duration `json:"window"`
	CurrentRequests int           `json:"current_requests"`
	Available       int           `json:"available"`
	NextReset       time.Time     `json:"next_reset,omitempty"`
}

// Info returns current window information
func (swl *SlidingWindowLimiter) Info() WindowInfo {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	now := swl.clock.Now()
	swl.cleanExpiredRequests(now)

	info := WindowInfo{
		Limit:           swl.limit,
		Window:          swl.window,
		CurrentRequests: len(swl.requests),
		Available:       swl.limit - len(swl.requests),
	}

	// Find the next reset time (when the oldest request expires)
	if len(swl.requests) > 0 {
		oldest := swl.requests[0]
		info.NextReset = oldest.Add(swl.window)
	}

	return info
}

// MultiSlidingWindow provides sliding window rate limiting for multiple keys
type MultiSlidingWindow struct {
	mu       sync.RWMutex
	limiters map[string]*SlidingWindowLimiter
	limit    int
	window   time.Duration
	clock    Clock
}

// NewMultiSlidingWindow creates a new multi-key sliding window limiter
func NewMultiSlidingWindow(limit int, window time.Duration) *MultiSlidingWindow {
	return &MultiSlidingWindow{
		limiters: make(map[string]*SlidingWindowLimiter),
		limit:    limit,
		window:   window,
		clock:    SystemClock{},
	}
}

// Allow returns true if a request is allowed for the given key
func (msw *MultiSlidingWindow) Allow(key string) bool {
	return msw.getLimiter(key).Allow()
}

// AllowN returns true if n requests are allowed for the given key
func (msw *MultiSlidingWindow) AllowN(key string, n int) bool {
	return msw.getLimiter(key).AllowN(n)
}

// Wait blocks until a request is allowed for the given key
func (msw *MultiSlidingWindow) Wait(ctx context.Context, key string) error {
	return msw.getLimiter(key).Wait(ctx)
}

// WaitN blocks until n requests are allowed for the given key
func (msw *MultiSlidingWindow) WaitN(ctx context.Context, key string, n int) error {
	return msw.getLimiter(key).WaitN(ctx, n)
}

// Info returns window information for the given key
func (msw *MultiSlidingWindow) Info(key string) WindowInfo {
	return msw.getLimiter(key).Info()
}

// getLimiter gets or creates a limiter for the given key
func (msw *MultiSlidingWindow) getLimiter(key string) *SlidingWindowLimiter {
	msw.mu.RLock()
	limiter, exists := msw.limiters[key]
	msw.mu.RUnlock()

	if exists {
		return limiter
	}

	msw.mu.Lock()
	defer msw.mu.Unlock()

	// Double-check after acquiring write lock
	limiter, exists = msw.limiters[key]
	if exists {
		return limiter
	}

	limiter = NewSlidingWindowLimiterWithClock(msw.limit, msw.window, msw.clock)
	msw.limiters[key] = limiter
	return limiter
}

// RemoveKey removes the limiter for the given key
func (msw *MultiSlidingWindow) RemoveKey(key string) {
	msw.mu.Lock()
	defer msw.mu.Unlock()
	delete(msw.limiters, key)
}

// Keys returns all currently tracked keys
func (msw *MultiSlidingWindow) Keys() []string {
	msw.mu.RLock()
	defer msw.mu.RUnlock()

	keys := make([]string, 0, len(msw.limiters))
	for key := range msw.limiters {
		keys = append(keys, key)
	}
	return keys
}

// Clear removes all limiters
func (msw *MultiSlidingWindow) Clear() {
	msw.mu.Lock()
	defer msw.mu.Unlock()
	msw.limiters = make(map[string]*SlidingWindowLimiter)
}