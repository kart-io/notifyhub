package internal

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateID(t *testing.T) {
	// Test basic ID generation and format
	id := GenerateID()
	assert.NotEmpty(t, id)
	assert.Regexp(t, `^\d+-\d+$`, id)
}

func TestGenerateIDConcurrency(t *testing.T) {
	// Test concurrent ID generation for uniqueness
	const numGoroutines = 100
	const idsPerGoroutine = 10

	idChan := make(chan string, numGoroutines*idsPerGoroutine)
	var wg sync.WaitGroup

	// Generate IDs concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				idChan <- GenerateID()
			}
		}()
	}

	wg.Wait()
	close(idChan)

	// Collect all IDs and verify uniqueness
	idSet := make(map[string]bool)
	for id := range idChan {
		assert.NotEmpty(t, id)
		assert.False(t, idSet[id], "Duplicate ID generated: %s", id)
		idSet[id] = true
	}

	assert.Len(t, idSet, numGoroutines*idsPerGoroutine)
}

// Token Bucket Tests

func TestNewTokenBucket(t *testing.T) {
	capacity := 10
	refillRate := 5
	interval := time.Second

	bucket := NewTokenBucket(capacity, refillRate, interval)

	assert.NotNil(t, bucket)
	assert.Equal(t, capacity, bucket.capacity)
	assert.Equal(t, refillRate, bucket.refillRate)
	assert.Equal(t, interval, bucket.interval)
	assert.Equal(t, capacity, bucket.tokens) // Should start full
	assert.False(t, bucket.lastRefill.IsZero())
}

func TestTokenBucketTryConsume(t *testing.T) {
	bucket := NewTokenBucket(3, 1, time.Second)

	// Should be able to consume all initial tokens
	assert.True(t, bucket.TryConsume())
	assert.True(t, bucket.TryConsume())
	assert.True(t, bucket.TryConsume())

	// Should fail when bucket is empty
	assert.False(t, bucket.TryConsume())
	assert.False(t, bucket.TryConsume())
}

func TestTokenBucketTryConsumeN(t *testing.T) {
	bucket := NewTokenBucket(10, 5, time.Second)

	// Should be able to consume multiple tokens
	assert.True(t, bucket.TryConsumeN(3))
	assert.Equal(t, 7, bucket.AvailableTokens())

	assert.True(t, bucket.TryConsumeN(7))
	assert.Equal(t, 0, bucket.AvailableTokens())

	// Should fail when requesting more than available
	assert.False(t, bucket.TryConsumeN(1))
	assert.False(t, bucket.TryConsumeN(5))
}

func TestTokenBucketRefill(t *testing.T) {
	bucket := NewTokenBucket(5, 2, 100*time.Millisecond)

	// Consume all tokens
	assert.True(t, bucket.TryConsumeN(5))
	assert.Equal(t, 0, bucket.AvailableTokens())

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should have refilled some tokens
	available := bucket.AvailableTokens()
	assert.True(t, available >= 2, "Expected at least 2 tokens, got %d", available)
	assert.True(t, available <= 5, "Expected at most 5 tokens, got %d", available)
}

func TestTokenBucketRefillCap(t *testing.T) {
	bucket := NewTokenBucket(3, 10, 100*time.Millisecond)

	// Don't consume any tokens
	assert.Equal(t, 3, bucket.AvailableTokens())

	// Wait for multiple refill intervals
	time.Sleep(300 * time.Millisecond)

	// Should not exceed capacity
	available := bucket.AvailableTokens()
	assert.Equal(t, 3, available, "Bucket should not exceed capacity")
}

func TestTokenBucketConsumeWithTimeout(t *testing.T) {
	bucket := NewTokenBucket(1, 1, 100*time.Millisecond)

	// Consume the initial token
	assert.True(t, bucket.TryConsume())

	// Should succeed with timeout (will wait for refill)
	ctx := context.Background()
	err := bucket.ConsumeWithTimeout(ctx, 200*time.Millisecond)
	assert.NoError(t, err)

	// Should timeout when no refill can happen in time
	err = bucket.ConsumeWithTimeout(ctx, 50*time.Millisecond)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestTokenBucketConsumeWithContextCancellation(t *testing.T) {
	bucket := NewTokenBucket(1, 1, time.Hour) // Very slow refill

	// Consume the initial token
	assert.True(t, bucket.TryConsume())

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Should return context error
	err := bucket.ConsumeWithTimeout(ctx, time.Second)
	assert.Equal(t, context.Canceled, err)
}

// Rate Limiter Config Tests

func TestNewRateLimiter(t *testing.T) {
	// Test with nil config
	limiter := NewRateLimiter(nil)
	assert.IsType(t, &NoOpRateLimiter{}, limiter)

	// Test with disabled config
	config := &RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		Enabled:           false,
	}
	limiter = NewRateLimiter(config)
	assert.IsType(t, &NoOpRateLimiter{}, limiter)

	// Test with zero rate
	config = &RateLimitConfig{
		RequestsPerSecond: 0,
		BurstSize:         5,
		Enabled:           true,
	}
	limiter = NewRateLimiter(config)
	assert.IsType(t, &NoOpRateLimiter{}, limiter)

	// Test with valid config
	config = &RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		Enabled:           true,
	}
	limiter = NewRateLimiter(config)
	assert.IsType(t, &TokenBucket{}, limiter)

	// Test with zero burst size (should default to rate)
	config = &RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         0,
		Enabled:           true,
	}
	limiter = NewRateLimiter(config)
	bucket := limiter.(*TokenBucket)
	assert.Equal(t, 10, bucket.capacity)
}

func TestNoOpRateLimiter(t *testing.T) {
	limiter := &NoOpRateLimiter{}

	// Should always allow
	assert.True(t, limiter.TryConsume())
	assert.True(t, limiter.TryConsume())
	assert.True(t, limiter.TryConsume())

	// Should always return 1 token available
	assert.Equal(t, 1, limiter.AvailableTokens())

	// Should never block
	ctx := context.Background()
	err := limiter.ConsumeWithTimeout(ctx, time.Millisecond)
	assert.NoError(t, err)
}

func TestEnsureRateLimiter(t *testing.T) {
	bucket := NewTokenBucket(2, 1, time.Second)
	ctx := context.Background()

	// Test successful function execution
	executed := false
	err := EnsureRateLimiter(bucket, ctx, 100*time.Millisecond, func() error {
		executed = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, executed)

	// Test function returning error
	err = EnsureRateLimiter(bucket, ctx, 100*time.Millisecond, func() error {
		return fmt.Errorf("test error")
	})
	assert.Error(t, err)
	assert.Equal(t, "test error", err.Error())

	// Test with exhausted bucket - create new bucket with 0 tokens
	emptyBucket := NewTokenBucket(0, 1, time.Hour) // Very slow refill
	executed = false
	err = EnsureRateLimiter(emptyBucket, ctx, 50*time.Millisecond, func() error {
		executed = true
		return nil
	})
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.False(t, executed)
}





