// Package ratelimit provides token bucket rate limiter implementation
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	mu       sync.Mutex
	rate     Rate
	burst    int
	tokens   float64
	lastTick time.Time
	clock    Clock
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(rate Rate, burst int) *TokenBucket {
	return NewTokenBucketWithClock(rate, burst, SystemClock{})
}

// NewTokenBucketWithClock creates a new token bucket with custom clock
func NewTokenBucketWithClock(rate Rate, burst int, clock Clock) *TokenBucket {
	if rate < 0 {
		rate = 0
	}
	if burst <= 0 {
		burst = 1
	}

	tb := &TokenBucket{
		rate:     rate,
		burst:    burst,
		tokens:   float64(burst),
		lastTick: clock.Now(),
		clock:    clock,
	}

	return tb
}

// Allow returns true if a request is allowed under the rate limit
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN returns true if n requests are allowed under the rate limit
func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := tb.clock.Now()
	tb.advance(now)

	if float64(n) <= tb.tokens {
		tb.tokens -= float64(n)
		return true
	}

	return false
}

// Wait blocks until a request is allowed or context is cancelled
func (tb *TokenBucket) Wait(ctx context.Context) error {
	return tb.WaitN(ctx, 1)
}

// WaitN blocks until n requests are allowed or context is cancelled
func (tb *TokenBucket) WaitN(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}

	reservation := tb.ReserveN(n)
	if !reservation.OK() {
		return ErrLimitExceeded
	}

	delay := reservation.Delay()
	if delay == 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		reservation.Cancel()
		return ctx.Err()
	}
}

// Reserve reserves a token and returns a reservation
func (tb *TokenBucket) Reserve() Reservation {
	return tb.ReserveN(1)
}

// ReserveN reserves n tokens and returns a reservation
func (tb *TokenBucket) ReserveN(n int) Reservation {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := tb.clock.Now()
	tb.advance(now)

	// Check if burst size is exceeded
	if n > tb.burst {
		return Reservation{
			ok:      false,
			tokens:  n,
			limiter: tb,
		}
	}

	// Calculate when we'll have enough tokens
	var waitTime time.Duration
	if float64(n) > tb.tokens {
		deficit := float64(n) - tb.tokens
		waitTime = time.Duration(deficit/float64(tb.rate)*float64(time.Second) + 0.5)
	}

	// Reserve the tokens
	tb.tokens -= float64(n)

	return Reservation{
		ok:        true,
		delay:     waitTime,
		tokens:    n,
		timeToAct: now.Add(waitTime),
		limiter:   tb,
	}
}

// Limit returns the current rate limit
func (tb *TokenBucket) Limit() Rate {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.rate
}

// Burst returns the current burst size
func (tb *TokenBucket) Burst() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.burst
}

// SetLimit changes the rate limit
func (tb *TokenBucket) SetLimit(newRate Rate) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := tb.clock.Now()
	tb.advance(now)
	tb.rate = newRate
}

// SetBurst changes the burst size
func (tb *TokenBucket) SetBurst(newBurst int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := tb.clock.Now()
	tb.advance(now)

	tb.burst = newBurst
	if tb.tokens > float64(newBurst) {
		tb.tokens = float64(newBurst)
	}
}

// Tokens returns the current number of available tokens
func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := tb.clock.Now()
	tb.advance(now)
	return tb.tokens
}

// advance updates the token bucket based on elapsed time
func (tb *TokenBucket) advance(now time.Time) {
	if now.Before(tb.lastTick) {
		// Clock went backwards, reset lastTick
		tb.lastTick = now
		return
	}

	elapsed := now.Sub(tb.lastTick).Seconds()
	tb.lastTick = now

	if tb.rate == Inf {
		tb.tokens = float64(tb.burst)
		return
	}

	// Add tokens based on elapsed time
	tb.tokens += elapsed * float64(tb.rate)
	if tb.tokens > float64(tb.burst) {
		tb.tokens = float64(tb.burst)
	}
}

// cancelReservation cancels a reservation and returns tokens
func (tb *TokenBucket) cancelReservation(r *Reservation) {
	tb.cancelReservationAt(r, tb.clock.Now())
}

// cancelReservationAt cancels a reservation at a specific time
func (tb *TokenBucket) cancelReservationAt(r *Reservation, now time.Time) {
	if !r.ok {
		return
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.advance(now)

	// Only return tokens if the reservation hasn't been used yet
	if now.Before(r.timeToAct) {
		tb.tokens += float64(r.tokens)
		if tb.tokens > float64(tb.burst) {
			tb.tokens = float64(tb.burst)
		}
	}
}

// TokenBucketStats provides statistics about token bucket usage
type TokenBucketStats struct {
	Rate           Rate      `json:"rate"`
	Burst          int       `json:"burst"`
	Tokens         float64   `json:"tokens"`
	LastTick       time.Time `json:"last_tick"`
	RequestsServed uint64    `json:"requests_served"`
	RequestsDenied uint64    `json:"requests_denied"`
}

// StatsTokenBucket wraps a TokenBucket with statistics collection
type StatsTokenBucket struct {
	*TokenBucket
	mu             sync.RWMutex
	requestsServed uint64
	requestsDenied uint64
}

// NewStatsTokenBucket creates a new token bucket with statistics
func NewStatsTokenBucket(rate Rate, burst int) *StatsTokenBucket {
	return &StatsTokenBucket{
		TokenBucket: NewTokenBucket(rate, burst),
	}
}

// Allow wraps TokenBucket.Allow with statistics
func (stb *StatsTokenBucket) Allow() bool {
	allowed := stb.TokenBucket.Allow()
	stb.recordRequest(allowed)
	return allowed
}

// AllowN wraps TokenBucket.AllowN with statistics
func (stb *StatsTokenBucket) AllowN(n int) bool {
	allowed := stb.TokenBucket.AllowN(n)
	stb.recordRequest(allowed)
	return allowed
}

// recordRequest records whether a request was allowed or denied
func (stb *StatsTokenBucket) recordRequest(allowed bool) {
	stb.mu.Lock()
	defer stb.mu.Unlock()

	if allowed {
		stb.requestsServed++
	} else {
		stb.requestsDenied++
	}
}

// Stats returns current statistics
func (stb *StatsTokenBucket) Stats() TokenBucketStats {
	stb.mu.RLock()
	defer stb.mu.RUnlock()

	stb.TokenBucket.mu.Lock()
	defer stb.TokenBucket.mu.Unlock()

	now := stb.clock.Now()
	stb.advance(now)

	return TokenBucketStats{
		Rate:           stb.rate,
		Burst:          stb.burst,
		Tokens:         stb.tokens,
		LastTick:       stb.lastTick,
		RequestsServed: stb.requestsServed,
		RequestsDenied: stb.requestsDenied,
	}
}

// Reset resets the statistics counters
func (stb *StatsTokenBucket) Reset() {
	stb.mu.Lock()
	defer stb.mu.Unlock()

	stb.requestsServed = 0
	stb.requestsDenied = 0
}

// BurstLimiter provides burst detection and limiting
type BurstLimiter struct {
	tb           *TokenBucket
	maxBurstSize int
	burstWindow  time.Duration
	requests     []time.Time
	mu           sync.Mutex
	clock        Clock
}

// NewBurstLimiter creates a new burst limiter
func NewBurstLimiter(rate Rate, burst int, maxBurstSize int, burstWindow time.Duration) *BurstLimiter {
	return &BurstLimiter{
		tb:           NewTokenBucket(rate, burst),
		maxBurstSize: maxBurstSize,
		burstWindow:  burstWindow,
		requests:     make([]time.Time, 0),
		clock:        SystemClock{},
	}
}

// Allow checks both rate limit and burst limit
func (bl *BurstLimiter) Allow() bool {
	if !bl.tb.Allow() {
		return false
	}

	bl.mu.Lock()
	defer bl.mu.Unlock()

	now := bl.clock.Now()
	cutoff := now.Add(-bl.burstWindow)

	// Remove old requests outside the window
	var validRequests []time.Time
	for _, req := range bl.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	bl.requests = validRequests

	// Check if adding this request would exceed burst limit
	if len(bl.requests) >= bl.maxBurstSize {
		return false
	}

	// Add current request
	bl.requests = append(bl.requests, now)
	return true
}

// AllowN is not supported for burst limiter
func (bl *BurstLimiter) AllowN(n int) bool {
	if n == 1 {
		return bl.Allow()
	}
	return false // Burst limiter doesn't support n > 1
}

// Other Limiter interface methods delegate to token bucket
func (bl *BurstLimiter) Wait(ctx context.Context) error {
	return bl.tb.Wait(ctx)
}

func (bl *BurstLimiter) WaitN(ctx context.Context, n int) error {
	return bl.tb.WaitN(ctx, n)
}

func (bl *BurstLimiter) Reserve() Reservation {
	return bl.tb.Reserve()
}

func (bl *BurstLimiter) ReserveN(n int) Reservation {
	return bl.tb.ReserveN(n)
}

func (bl *BurstLimiter) Limit() Rate {
	return bl.tb.Limit()
}

func (bl *BurstLimiter) Burst() int {
	return bl.tb.Burst()
}
