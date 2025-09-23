package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	rate := PerSecond(10) // 10 requests per second
	burst := 5
	limiter := NewTokenBucket(rate, burst)

	// Test initial burst
	for i := 0; i < burst; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed in initial burst", i+1)
		}
	}

	// Next request should be denied
	if limiter.Allow() {
		t.Error("Request after burst should be denied")
	}

	// Test rate and burst getters
	if limiter.Limit() != rate {
		t.Errorf("Expected rate %v, got %v", rate, limiter.Limit())
	}

	if limiter.Burst() != burst {
		t.Errorf("Expected burst %d, got %d", burst, limiter.Burst())
	}
}

func TestTokenBucketAllowN(t *testing.T) {
	rate := PerSecond(10)
	burst := 5
	limiter := NewTokenBucket(rate, burst)

	// Test allowing N tokens
	if !limiter.AllowN(3) {
		t.Error("Should allow 3 tokens from initial burst")
	}

	if !limiter.AllowN(2) {
		t.Error("Should allow remaining 2 tokens from initial burst")
	}

	// Should not allow more tokens
	if limiter.AllowN(1) {
		t.Error("Should not allow more tokens after burst is exhausted")
	}

	// Test requesting more than burst size
	if limiter.AllowN(burst + 1) {
		t.Error("Should not allow more tokens than burst size")
	}
}

func TestTokenBucketWithMockClock(t *testing.T) {
	clock := NewMockClock(time.Now())
	rate := PerSecond(10) // 10 requests per second = 1 request per 100ms
	burst := 1
	limiter := NewTokenBucketWithClock(rate, burst, clock)

	// Use up the burst
	if !limiter.Allow() {
		t.Error("First request should be allowed")
	}

	// Should be denied immediately
	if limiter.Allow() {
		t.Error("Second request should be denied")
	}

	// Advance time by 100ms to allow one more request
	clock.Advance(100 * time.Millisecond)

	// Should be allowed now
	if !limiter.Allow() {
		t.Error("Request should be allowed after time advance")
	}
}

func TestTokenBucketWait(t *testing.T) {
	clock := NewMockClock(time.Now())
	rate := PerSecond(10)
	burst := 1
	limiter := NewTokenBucketWithClock(rate, burst, clock)

	// Use up the burst
	if !limiter.Allow() {
		t.Error("First request should be allowed")
	}

	// Test wait with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := limiter.Wait(ctx)
	elapsed := time.Since(start)

	// Should timeout
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	if elapsed < 40*time.Millisecond {
		t.Error("Wait should have taken at least the timeout duration")
	}
}

func TestTokenBucketReservation(t *testing.T) {
	clock := NewMockClock(time.Now())
	rate := PerSecond(10)
	burst := 2 // Use burst of 2 for more predictable behavior
	limiter := NewTokenBucketWithClock(rate, burst, clock)

	// Use up one token
	reservation1 := limiter.Reserve()
	if !reservation1.OK() {
		t.Error("First reservation should be OK")
	}
	if reservation1.Delay() != 0 {
		t.Error("First reservation should have no delay")
	}

	// Second should also be immediate (burst=2)
	reservation2 := limiter.Reserve()
	if !reservation2.OK() {
		t.Error("Second reservation should be OK")
	}
	if reservation2.Delay() != 0 {
		t.Error("Second reservation should have no delay with burst=2")
	}

	// Third should have delay
	reservation3 := limiter.Reserve()
	if !reservation3.OK() {
		t.Error("Third reservation should be OK")
	}
	if reservation3.Delay() == 0 {
		t.Error("Third reservation should have delay")
	}

	// Cancel the third reservation
	reservation3.Cancel()

	// Advance time slightly to allow token refill
	clock.Advance(100 * time.Millisecond) // 0.1 second at 10 req/sec = 1 token

	// Now the next request should be allowed immediately
	if !limiter.Allow() {
		t.Error("Request should be allowed after cancellation and time advance")
	}
}

func TestSlidingWindow(t *testing.T) {
	clock := NewMockClock(time.Now())
	limit := 5
	window := time.Minute
	limiter := NewSlidingWindowLimiterWithClock(limit, window, clock)

	// Test initial requests
	for i := 0; i < limit; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Next request should be denied
	if limiter.Allow() {
		t.Error("Request after limit should be denied")
	}

	// Advance time by half the window
	clock.Advance(30 * time.Second)

	// Should still be denied
	if limiter.Allow() {
		t.Error("Request should still be denied")
	}

	// Advance time past the window
	clock.Advance(31 * time.Second)

	// Should be allowed now
	if !limiter.Allow() {
		t.Error("Request should be allowed after window expires")
	}
}

func TestSlidingWindowInfo(t *testing.T) {
	clock := NewMockClock(time.Now())
	limit := 3
	window := time.Minute
	limiter := NewSlidingWindowLimiterWithClock(limit, window, clock)

	// Make some requests
	limiter.AllowN(2)

	info := limiter.Info()
	if info.Limit != limit {
		t.Errorf("Expected limit %d, got %d", limit, info.Limit)
	}
	if info.CurrentRequests != 2 {
		t.Errorf("Expected current requests 2, got %d", info.CurrentRequests)
	}
	if info.Available != 1 {
		t.Errorf("Expected available 1, got %d", info.Available)
	}
}

func TestMultiLimiter(t *testing.T) {
	rate := PerSecond(10)
	burst := 2
	multi := NewMulti(rate, burst)

	// Test different keys
	if !multi.Allow("user1") {
		t.Error("First request for user1 should be allowed")
	}
	if !multi.Allow("user2") {
		t.Error("First request for user2 should be allowed")
	}

	// Each user should have their own limit
	if !multi.Allow("user1") {
		t.Error("Second request for user1 should be allowed")
	}

	// Third request for user1 should be denied
	if multi.Allow("user1") {
		t.Error("Third request for user1 should be denied")
	}

	// But user2 should still have capacity
	if !multi.Allow("user2") {
		t.Error("Second request for user2 should be allowed")
	}

	// Test key management
	keys := multi.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	multi.RemoveKey("user1")
	keys = multi.Keys()
	if len(keys) != 1 {
		t.Errorf("Expected 1 key after removal, got %d", len(keys))
	}

	multi.Clear()
	keys = multi.Keys()
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys after clear, got %d", len(keys))
	}
}

func TestSometimes(t *testing.T) {
	// Test that Sometimes at least works and allows some requests
	sometimes := NewSometimes(1) // Allow all requests (rate 1.0)

	// Check that we can get at least one request allowed
	allowed := false
	for i := 0; i < 5; i++ {
		if sometimes.Allow() {
			allowed = true
			break
		}
	}

	if !allowed {
		t.Error("Sometimes should allow at least one request out of 5")
	}
}

func TestRateHelpers(t *testing.T) {
	// Test rate creation helpers
	if PerSecond(60) != Rate(60) {
		t.Error("PerSecond should create correct rate")
	}

	if PerMinute(60) != Rate(1) {
		t.Error("PerMinute should create correct rate")
	}

	if PerHour(3600) != Rate(1) {
		t.Error("PerHour should create correct rate")
	}

	if Every(time.Second) != Rate(1) {
		t.Error("Every should create correct rate")
	}

	if Per(10, time.Second) != Rate(10) {
		t.Error("Per should create correct rate")
	}
}

func TestBurstLimiter(t *testing.T) {
	rate := PerSecond(100) // Higher rate to ensure token bucket allows requests
	burst := 10           // Higher burst for token bucket
	maxBurstSize := 3
	burstWindow := time.Second

	limiter := NewBurstLimiter(rate, burst, maxBurstSize, burstWindow)

	// Should allow up to maxBurstSize requests quickly
	for i := 0; i < maxBurstSize; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed in burst", i+1)
		}
	}

	// Next request should be denied due to burst limit
	if limiter.Allow() {
		t.Error("Request should be denied due to burst limit")
	}
}

func BenchmarkTokenBucketAllow(b *testing.B) {
	limiter := NewTokenBucket(PerSecond(1000), 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

func BenchmarkSlidingWindowAllow(b *testing.B) {
	limiter := NewSlidingWindowLimiter(1000, time.Minute)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

func BenchmarkMultiLimiterAllow(b *testing.B) {
	multi := NewMulti(PerSecond(1000), 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		multi.Allow("user1")
	}
}