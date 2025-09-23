// Package ratelimit provides rate limiting utilities for NotifyHub
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Limiter defines the interface for rate limiting
type Limiter interface {
	// Allow returns true if the request is allowed under the rate limit
	Allow() bool
	// AllowN returns true if n requests are allowed under the rate limit
	AllowN(n int) bool
	// Wait blocks until the rate limit allows the request or context is cancelled
	Wait(ctx context.Context) error
	// WaitN blocks until n requests are allowed under the rate limit or context is cancelled
	WaitN(ctx context.Context, n int) error
	// Reserve reserves a token and returns the delay before the request can proceed
	Reserve() Reservation
	// ReserveN reserves n tokens and returns the delay before the requests can proceed
	ReserveN(n int) Reservation
	// Limit returns the current rate limit
	Limit() Rate
	// Burst returns the current burst size
	Burst() int
}

// Rate represents requests per second
type Rate float64

// Constants for common rates
const (
	Inf = Rate(1 << 62) // Infinite rate
)

// Per returns the rate for requests per duration
func Per(requests int, duration time.Duration) Rate {
	return Rate(float64(requests) / duration.Seconds())
}

// Reservation holds information about a rate limit reservation
type Reservation struct {
	ok        bool
	delay     time.Duration
	tokens    int
	timeToAct time.Time
	limiter   *TokenBucket
}

// OK returns whether the reservation is valid
func (r *Reservation) OK() bool {
	return r.ok
}

// Delay returns the duration to wait before the reservation can be used
func (r *Reservation) Delay() time.Duration {
	if !r.ok {
		return time.Duration(-1)
	}
	delay := time.Until(r.timeToAct)
	if delay < 0 {
		return 0
	}
	return delay
}

// DelayFrom returns the duration to wait from the given time
func (r *Reservation) DelayFrom(now time.Time) time.Duration {
	if !r.ok {
		return time.Duration(-1)
	}
	delay := r.timeToAct.Sub(now)
	if delay < 0 {
		return 0
	}
	return delay
}

// Cancel cancels the reservation and returns tokens to the bucket
func (r *Reservation) Cancel() {
	if !r.ok || r.limiter == nil {
		return
	}
	r.limiter.cancelReservation(r)
}

// CancelAt cancels the reservation at the given time
func (r *Reservation) CancelAt(now time.Time) {
	if !r.ok || r.limiter == nil {
		return
	}
	r.limiter.cancelReservationAt(r, now)
}

// NewLimiter creates a new rate limiter with the given rate and burst size
func NewLimiter(rate Rate, burst int) Limiter {
	return NewTokenBucket(rate, burst)
}

// NewLimiterWithClock creates a new rate limiter with a custom clock
func NewLimiterWithClock(rate Rate, burst int, clock Clock) Limiter {
	return NewTokenBucketWithClock(rate, burst, clock)
}

// Clock interface allows for testing with mock time
type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

// SystemClock implements Clock using system time
type SystemClock struct{}

// Now returns current system time
func (SystemClock) Now() time.Time {
	return time.Now()
}

// Sleep sleeps for the given duration
func (SystemClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// MockClock implements Clock for testing
type MockClock struct {
	mu   sync.Mutex
	now  time.Time
	wait []waiter
}

type waiter struct {
	until time.Time
	ch    chan struct{}
}

// NewMockClock creates a new mock clock starting at the given time
func NewMockClock(start time.Time) *MockClock {
	return &MockClock{now: start}
}

// Now returns the current mock time
func (c *MockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Sleep does not actually sleep but advances the mock time
func (c *MockClock) Sleep(d time.Duration) {
	c.Advance(d)
}

// Advance advances the mock clock by the given duration
func (c *MockClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)

	// Wake up any waiters whose time has come
	var remaining []waiter
	for _, w := range c.wait {
		if c.now.After(w.until) || c.now.Equal(w.until) {
			close(w.ch)
		} else {
			remaining = append(remaining, w)
		}
	}
	c.wait = remaining
}

// Set sets the mock clock to the given time
func (c *MockClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = t

	// Wake up any waiters whose time has come
	var remaining []waiter
	for _, w := range c.wait {
		if c.now.After(w.until) || c.now.Equal(w.until) {
			close(w.ch)
		} else {
			remaining = append(remaining, w)
		}
	}
	c.wait = remaining
}

// WaitUntil blocks until the mock clock reaches the given time
func (c *MockClock) WaitUntil(t time.Time) <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan struct{})
	if c.now.After(t) || c.now.Equal(t) {
		close(ch)
		return ch
	}

	c.wait = append(c.wait, waiter{until: t, ch: ch})
	return ch
}

// Sometimes is a utility for probabilistic rate limiting
type Sometimes struct {
	limiter Limiter
}

// NewSometimes creates a new Sometimes limiter that allows approximately
// 1 out of every n requests through
func NewSometimes(n int) *Sometimes {
	rate := Rate(1.0 / float64(n))
	return &Sometimes{
		limiter: NewLimiter(rate, 1),
	}
}

// Allow returns true approximately 1/n of the time
func (s *Sometimes) Allow() bool {
	return s.limiter.Allow()
}

// Multi provides rate limiting for multiple keys
type Multi struct {
	mu       sync.RWMutex
	limiters map[string]Limiter
	rate     Rate
	burst    int
	clock    Clock
}

// NewMulti creates a new multi-key rate limiter
func NewMulti(rate Rate, burst int) *Multi {
	return &Multi{
		limiters: make(map[string]Limiter),
		rate:     rate,
		burst:    burst,
		clock:    SystemClock{},
	}
}

// NewMultiWithClock creates a new multi-key rate limiter with custom clock
func NewMultiWithClock(rate Rate, burst int, clock Clock) *Multi {
	return &Multi{
		limiters: make(map[string]Limiter),
		rate:     rate,
		burst:    burst,
		clock:    clock,
	}
}

// Allow returns true if the request is allowed for the given key
func (m *Multi) Allow(key string) bool {
	return m.getLimiter(key).Allow()
}

// AllowN returns true if n requests are allowed for the given key
func (m *Multi) AllowN(key string, n int) bool {
	return m.getLimiter(key).AllowN(n)
}

// Wait blocks until the rate limit allows the request for the given key
func (m *Multi) Wait(ctx context.Context, key string) error {
	return m.getLimiter(key).Wait(ctx)
}

// WaitN blocks until n requests are allowed for the given key
func (m *Multi) WaitN(ctx context.Context, key string, n int) error {
	return m.getLimiter(key).WaitN(ctx, n)
}

// Reserve reserves a token for the given key
func (m *Multi) Reserve(key string) Reservation {
	return m.getLimiter(key).Reserve()
}

// ReserveN reserves n tokens for the given key
func (m *Multi) ReserveN(key string, n int) Reservation {
	return m.getLimiter(key).ReserveN(n)
}

// getLimiter gets or creates a limiter for the given key
func (m *Multi) getLimiter(key string) Limiter {
	m.mu.RLock()
	limiter, exists := m.limiters[key]
	m.mu.RUnlock()

	if exists {
		return limiter
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	limiter, exists = m.limiters[key]
	if exists {
		return limiter
	}

	limiter = NewLimiterWithClock(m.rate, m.burst, m.clock)
	m.limiters[key] = limiter
	return limiter
}

// RemoveKey removes the limiter for the given key
func (m *Multi) RemoveKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.limiters, key)
}

// Keys returns all currently tracked keys
func (m *Multi) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.limiters))
	for key := range m.limiters {
		keys = append(keys, key)
	}
	return keys
}

// Clear removes all limiters
func (m *Multi) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limiters = make(map[string]Limiter)
}

// Error types
var (
	ErrLimitExceeded = fmt.Errorf("rate limit exceeded")
	ErrInvalidRate   = fmt.Errorf("invalid rate")
	ErrInvalidBurst  = fmt.Errorf("invalid burst size")
)

// Helper functions

// Every creates a rate for one request per duration
func Every(interval time.Duration) Rate {
	if interval <= 0 {
		return Inf
	}
	return Rate(time.Second) / Rate(interval)
}

// PerSecond creates a rate for n requests per second
func PerSecond(n float64) Rate {
	return Rate(n)
}

// PerMinute creates a rate for n requests per minute
func PerMinute(n float64) Rate {
	return Rate(n / 60.0)
}

// PerHour creates a rate for n requests per hour
func PerHour(n float64) Rate {
	return Rate(n / 3600.0)
}
