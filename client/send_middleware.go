package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/internal"
	"github.com/kart-io/notifyhub/notifiers"
)

// SendMiddlewareFunc defines the signature for send middleware functions
type SendMiddlewareFunc func(ctx context.Context, message *notifiers.Message, options *Options, next SendFunc) ([]*notifiers.SendResult, error)

// SendFunc defines the signature for the send function
type SendFunc func(ctx context.Context, message *notifiers.Message, options *Options) ([]*notifiers.SendResult, error)

// SendMiddleware represents a send middleware component
type SendMiddleware interface {
	Name() string
	Execute(ctx context.Context, message *notifiers.Message, options *Options, next SendFunc) ([]*notifiers.SendResult, error)
}

// SendMiddlewareChain manages a chain of send middleware
type SendMiddlewareChain struct {
	middlewares []SendMiddleware
	mu          sync.RWMutex
}

// NewSendMiddlewareChain creates a new send middleware chain
func NewSendMiddlewareChain() *SendMiddlewareChain {
	return &SendMiddlewareChain{
		middlewares: make([]SendMiddleware, 0),
	}
}

// Use adds a middleware to the chain
func (smc *SendMiddlewareChain) Use(middleware SendMiddleware) {
	smc.mu.Lock()
	defer smc.mu.Unlock()
	smc.middlewares = append(smc.middlewares, middleware)
}

// UseFunc adds a middleware function to the chain
func (smc *SendMiddlewareChain) UseFunc(name string, fn SendMiddlewareFunc) {
	smc.Use(&FuncSendMiddleware{name: name, fn: fn})
}

// Execute executes the send middleware chain
func (smc *SendMiddlewareChain) Execute(ctx context.Context, message *notifiers.Message, options *Options, finalHandler SendFunc) ([]*notifiers.SendResult, error) {
	smc.mu.RLock()
	middlewares := make([]SendMiddleware, len(smc.middlewares))
	copy(middlewares, smc.middlewares)
	smc.mu.RUnlock()

	if len(middlewares) == 0 {
		return finalHandler(ctx, message, options)
	}

	// Build the chain from the end
	handler := finalHandler
	for i := len(middlewares) - 1; i >= 0; i-- {
		currentMiddleware := middlewares[i]
		currentHandler := handler
		handler = func(ctx context.Context, message *notifiers.Message, options *Options) ([]*notifiers.SendResult, error) {
			return currentMiddleware.Execute(ctx, message, options, currentHandler)
		}
	}

	return handler(ctx, message, options)
}

// Count returns the number of middleware in the chain
func (smc *SendMiddlewareChain) Count() int {
	smc.mu.RLock()
	defer smc.mu.RUnlock()
	return len(smc.middlewares)
}

// Names returns the names of all middleware in the chain
func (smc *SendMiddlewareChain) Names() []string {
	smc.mu.RLock()
	defer smc.mu.RUnlock()

	names := make([]string, len(smc.middlewares))
	for i, middleware := range smc.middlewares {
		names[i] = middleware.Name()
	}
	return names
}

// FuncSendMiddleware wraps a function as middleware
type FuncSendMiddleware struct {
	name string
	fn   SendMiddlewareFunc
}

func (fsm *FuncSendMiddleware) Name() string {
	return fsm.name
}

func (fsm *FuncSendMiddleware) Execute(ctx context.Context, message *notifiers.Message, options *Options, next SendFunc) ([]*notifiers.SendResult, error) {
	return fsm.fn(ctx, message, options, next)
}

// Built-in send middleware implementations

// SendLoggingMiddleware logs message send attempts
type SendLoggingMiddleware struct {
	hub *Hub
}

func NewSendLoggingMiddleware(hub *Hub) *SendLoggingMiddleware {
	return &SendLoggingMiddleware{hub: hub}
}

func (slm *SendLoggingMiddleware) Name() string {
	return "send_logging"
}

func (slm *SendLoggingMiddleware) Execute(ctx context.Context, message *notifiers.Message, options *Options, next SendFunc) ([]*notifiers.SendResult, error) {
	start := time.Now()

	slm.hub.logger.Info(ctx, "📤 Sending message: title='%s', targets=%d", message.Title, len(message.Targets))

	results, err := next(ctx, message, options)

	duration := time.Since(start)
	if err != nil {
		slm.hub.logger.Error(ctx, "❌ Message send failed after %v: %v", duration, err)
	} else {
		successCount := 0
		for _, result := range results {
			if result.Success {
				successCount++
			}
		}
		slm.hub.logger.Info(ctx, "✅ Message sent in %v: %d/%d successful", duration, successCount, len(results))
	}

	return results, err
}

// SendMetricsMiddleware collects metrics for message sends
type SendMetricsMiddleware struct {
	hub *Hub
}

func NewSendMetricsMiddleware(hub *Hub) *SendMetricsMiddleware {
	return &SendMetricsMiddleware{hub: hub}
}

func (smm *SendMetricsMiddleware) Name() string {
	return "send_metrics"
}

func (smm *SendMetricsMiddleware) Execute(ctx context.Context, message *notifiers.Message, options *Options, next SendFunc) ([]*notifiers.SendResult, error) {
	start := time.Now()

	results, err := next(ctx, message, options)

	duration := time.Since(start)

	// Record metrics
	if smm.hub.metrics != nil {
		if err != nil {
			smm.hub.metrics.RecordSend("", false, duration, err.Error())
		} else {
			for _, result := range results {
				smm.hub.metrics.RecordSend(result.Platform, result.Success, result.Duration, result.Error)
			}
		}
	}

	return results, err
}

// SendRateLimitMiddleware implements rate limiting for sends
type SendRateLimitMiddleware struct {
	limiter  map[string]*internal.TokenBucket
	mu       sync.RWMutex
	capacity int
	refill   time.Duration
}

func NewSendRateLimitMiddleware(requestsPerPeriod int, period time.Duration) *SendRateLimitMiddleware {
	return &SendRateLimitMiddleware{
		limiter:  make(map[string]*internal.TokenBucket),
		capacity: requestsPerPeriod,
		refill:   period,
	}
}

func (srlm *SendRateLimitMiddleware) Name() string {
	return "send_rate_limit"
}

func (srlm *SendRateLimitMiddleware) Execute(ctx context.Context, message *notifiers.Message, options *Options, next SendFunc) ([]*notifiers.SendResult, error) {
	// Group by platform for rate limiting
	platformTargets := make(map[string][]notifiers.Target)
	for _, target := range message.Targets {
		platform := target.Platform
		if platform == "" {
			switch target.Type {
			case notifiers.TargetTypeEmail:
				platform = "email"
			default:
				platform = "feishu"
			}
		}
		platformTargets[platform] = append(platformTargets[platform], target)
	}

	// Check rate limits for each platform
	for platform := range platformTargets {
		bucket := srlm.getBucket(platform)
		if !bucket.TryConsume() {
			return nil, NewRateLimitError(platform, fmt.Sprintf("Rate limit exceeded for platform %s", platform))
		}
	}

	return next(ctx, message, options)
}

func (srlm *SendRateLimitMiddleware) getBucket(platform string) *internal.TokenBucket {
	srlm.mu.RLock()
	bucket, exists := srlm.limiter[platform]
	srlm.mu.RUnlock()

	if !exists {
		srlm.mu.Lock()
		bucket, exists = srlm.limiter[platform]
		if !exists {
			bucket = internal.NewTokenBucket(srlm.capacity, srlm.capacity, srlm.refill)
			srlm.limiter[platform] = bucket
		}
		srlm.mu.Unlock()
	}

	return bucket
}

// SendValidationMiddleware validates messages before sending
type SendValidationMiddleware struct{}

func NewSendValidationMiddleware() *SendValidationMiddleware {
	return &SendValidationMiddleware{}
}

func (svm *SendValidationMiddleware) Name() string {
	return "send_validation"
}

func (svm *SendValidationMiddleware) Execute(ctx context.Context, message *notifiers.Message, options *Options, next SendFunc) ([]*notifiers.SendResult, error) {
	// Validate message
	if message.Title == "" {
		return nil, NewValidationError("EMPTY_TITLE", "Message title cannot be empty", "", "Provide a non-empty title")
	}

	if message.Body == "" {
		return nil, NewValidationError("EMPTY_BODY", "Message body cannot be empty", "", "Provide a non-empty body")
	}

	if len(message.Targets) == 0 {
		return nil, NewValidationError("NO_TARGETS", "Message must have at least one target", "", "Add at least one target")
	}

	// Validate targets
	for i, target := range message.Targets {
		if target.Value == "" {
			return nil, NewValidationError("EMPTY_TARGET_VALUE",
				fmt.Sprintf("Target %d has empty value", i),
				fmt.Sprintf("Target type: %s", target.Type),
				"Provide a valid target value")
		}

		if target.Type == notifiers.TargetTypeEmail && !strings.Contains(target.Value, "@") {
			return nil, NewValidationError("INVALID_EMAIL",
				fmt.Sprintf("Invalid email address: %s", target.Value),
				"Email must contain @ symbol",
				"Provide a valid email address")
		}
	}

	return next(ctx, message, options)
}

// SendRetryMiddleware implements retry logic with exponential backoff
type SendRetryMiddleware struct {
	maxRetries int
	baseDelay  time.Duration
}

func NewSendRetryMiddleware(maxRetries int, baseDelay time.Duration) *SendRetryMiddleware {
	return &SendRetryMiddleware{
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

func (srm *SendRetryMiddleware) Name() string {
	return "send_retry"
}

func (srm *SendRetryMiddleware) Execute(ctx context.Context, message *notifiers.Message, options *Options, next SendFunc) ([]*notifiers.SendResult, error) {
	var lastErr error

	for attempt := 0; attempt <= srm.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := time.Duration(1<<uint(attempt-1)) * srm.baseDelay
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		results, err := next(ctx, message, options)
		if err == nil {
			return results, nil
		}

		lastErr = err

		// Don't retry certain types of errors
		if IsValidationError(err) || IsConfigError(err) {
			break
		}

		// Check if error is explicitly non-retryable
		if !IsRetryableError(err) {
			break
		}
	}

	return nil, lastErr
}

// Add middleware support methods to Hub
func (h *Hub) UseSendMiddleware(middleware SendMiddleware) {
	if h.middlewareChain == nil {
		h.middlewareChain = NewSendMiddlewareChain()
	}
	h.middlewareChain.Use(middleware)
}

func (h *Hub) UseSendMiddlewareFunc(name string, fn SendMiddlewareFunc) {
	if h.middlewareChain == nil {
		h.middlewareChain = NewSendMiddlewareChain()
	}
	h.middlewareChain.UseFunc(name, fn)
}

// GetMiddlewareChain returns the current middleware chain
func (h *Hub) GetMiddlewareChain() *SendMiddlewareChain {
	return h.middlewareChain
}

// SetupDefaultMiddleware sets up default middleware chain
func (h *Hub) SetupDefaultMiddleware() {
	h.UseSendMiddleware(NewSendValidationMiddleware())
	h.UseSendMiddleware(NewSendLoggingMiddleware(h))
	h.UseSendMiddleware(NewSendMetricsMiddleware(h))
}

// SetupProductionMiddleware sets up production-ready middleware
func (h *Hub) SetupProductionMiddleware() {
	h.UseSendMiddleware(NewSendValidationMiddleware())
	h.UseSendMiddleware(NewSendRateLimitMiddleware(100, time.Minute)) // 100 requests per minute
	h.UseSendMiddleware(NewSendRetryMiddleware(3, time.Second))
	h.UseSendMiddleware(NewSendLoggingMiddleware(h))
	h.UseSendMiddleware(NewSendMetricsMiddleware(h))
}

// SetupDevelopmentMiddleware sets up development-friendly middleware
func (h *Hub) SetupDevelopmentMiddleware() {
	h.UseSendMiddleware(NewSendValidationMiddleware())
	h.UseSendMiddleware(NewSendLoggingMiddleware(h))
	h.UseSendMiddleware(NewSendMetricsMiddleware(h))
}

// Update the Send method to use middleware chain
func (h *Hub) sendWithMiddleware(ctx context.Context, message *notifiers.Message, options *Options) ([]*notifiers.SendResult, error) {
	if h.middlewareChain != nil {
		return h.middlewareChain.Execute(ctx, message, options, h.sendDirect)
	}
	return h.sendDirect(ctx, message, options)
}

// sendDirect is the original Send method logic (should be renamed from existing Send method)
func (h *Hub) sendDirect(ctx context.Context, message *notifiers.Message, options *Options) ([]*notifiers.SendResult, error) {
	// This would contain the original Send method implementation
	// For now, we'll call the existing Send method to avoid breaking changes
	// In a real implementation, you'd rename the existing Send to sendDirect
	// and replace Send with sendWithMiddleware
	return h.Send(ctx, message, options)
}