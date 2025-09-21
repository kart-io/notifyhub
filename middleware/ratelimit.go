package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/core/hub"
	"github.com/kart-io/notifyhub/logger"
)

// RateLimitMiddleware implements rate limiting for message sending
type RateLimitMiddleware struct {
	limiters map[string]*tokenBucket
	mutex    sync.RWMutex
	logger   logger.Interface
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(logger logger.Interface) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiters: make(map[string]*tokenBucket),
		logger:   logger,
	}
}

// SetLimit sets rate limit for a platform
func (m *RateLimitMiddleware) SetLimit(platform string, rate float64, capacity int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.limiters[platform] = newTokenBucket(rate, capacity)
	if m.logger != nil {
		m.logger.Info(context.Background(), "rate limit set", "platform", platform, "rate", rate, "capacity", capacity)
	}
}

// Process processes the message through rate limiting
func (m *RateLimitMiddleware) Process(ctx context.Context, msg *core.Message, targets []core.Target, next hub.ProcessFunc) (*core.SendingResults, error) {
	// Group targets by platform for rate limiting
	platformGroups := m.groupTargetsByPlatform(targets)

	var allowedTargets []core.Target
	results := core.NewSendingResults()

	for platform, platformTargets := range platformGroups {
		limiter := m.getLimiter(platform)
		if limiter == nil {
			// No rate limit for this platform, allow all
			allowedTargets = append(allowedTargets, platformTargets...)
			continue
		}

		// Check rate limit for each target
		for _, target := range platformTargets {
			if limiter.allow() {
				allowedTargets = append(allowedTargets, target)
			} else {
				// Rate limited, create failed result
				result := core.NewResult(msg.ID, target)
				result.Error = fmt.Errorf("rate limited")
				results.AddResult(result)

				if m.logger != nil {
					m.logger.Warn(ctx, "rate limited", "platform", platform, "target", target.String())
				}
			}
		}
	}

	// Process allowed targets
	if len(allowedTargets) > 0 {
		nextResults, err := next(ctx, msg, allowedTargets)
		if err != nil {
			return results, err
		}

		// Merge results
		for _, result := range nextResults.Results {
			results.AddResult(result)
		}
	}

	return results, nil
}

// getLimiter gets the rate limiter for a platform
func (m *RateLimitMiddleware) getLimiter(platform string) *tokenBucket {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.limiters[platform]
}

// groupTargetsByPlatform groups targets by platform
func (m *RateLimitMiddleware) groupTargetsByPlatform(targets []core.Target) map[string][]core.Target {
	groups := make(map[string][]core.Target)
	for _, target := range targets {
		groups[target.Platform] = append(groups[target.Platform], target)
	}
	return groups
}

// tokenBucket implements a simple token bucket for rate limiting
type tokenBucket struct {
	capacity   int
	tokens     int
	refillRate float64
	lastRefill time.Time
	mutex      sync.Mutex
}

// newTokenBucket creates a new token bucket
func newTokenBucket(refillRate float64, capacity int) *tokenBucket {
	return &tokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// allow checks if a token is available
func (tb *tokenBucket) allow() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := int(elapsed * tb.refillRate)

	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}

	// Check if token is available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}
