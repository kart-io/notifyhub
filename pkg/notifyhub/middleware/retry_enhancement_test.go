// Package middleware provides comprehensive tests for enhanced retry middleware
package middleware

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/errors"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// mockSendHandler creates a mock send handler for testing
func mockSendHandler(failCount int, err error) SendHandler {
	attempts := 0
	return func(ctx context.Context, msg *message.Message, targets []target.Target) (*receipt.Receipt, error) {
		attempts++
		if attempts <= failCount {
			return nil, err
		}
		return &receipt.Receipt{
			MessageID:  msg.ID,
			Status:     "success",
			Successful: 1,
			Total:      1,
		}, nil
	}
}

// TestRetryMiddlewareBasicFunctionality tests basic retry middleware functionality
func TestRetryMiddlewareBasicFunctionality(t *testing.T) {
	strategy := NewExponentialBackoffStrategy(ExponentialBackoffConfig{
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
		Jitter:      0.1,
		MaxAttempts: 3,
	})

	middleware := NewRetryMiddleware(strategy, logger.Discard)

	msg := &message.Message{
		ID:    "test-message-1",
		Title: "Test Message",
		Body:  "Test body",
	}

	targets := []target.Target{
		{Type: "webhook", Value: "test", Platform: "feishu"},
	}

	// Test successful operation (no retries needed)
	handler := mockSendHandler(0, nil)
	receipt, err := middleware.HandleSend(context.Background(), msg, targets, handler)

	if err != nil {
		t.Errorf("Expected no error for successful operation, got: %v", err)
	}

	if receipt == nil {
		t.Error("Expected receipt for successful operation")
	}
}

// TestRetryMiddlewareWithRetries tests retry middleware with actual retries
func TestRetryMiddlewareWithRetries(t *testing.T) {
	strategy := NewExponentialBackoffStrategy(ExponentialBackoffConfig{
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      0.0, // No jitter for predictable testing
		MaxAttempts: 3,
	})

	middleware := NewRetryMiddleware(strategy, logger.Discard)

	msg := &message.Message{
		ID:    "test-message-2",
		Title: "Test Message",
		Body:  "Test body",
	}

	targets := []target.Target{
		{Type: "webhook", Value: "test", Platform: "feishu"},
	}

	// Test with 2 failures, then success
	retryableErr := &errors.NotifyError{
		Code:     errors.ErrPlatformTimeout,
		Message:  "timeout error",
		Platform: "feishu",
	}

	handler := mockSendHandler(2, retryableErr)
	start := time.Now()
	receipt, err := middleware.HandleSend(context.Background(), msg, targets, handler)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected operation to succeed after retries, got: %v", err)
	}

	if receipt == nil {
		t.Error("Expected receipt after successful retry")
	}

	// Should have taken at least 2 retry delays (50ms + 100ms = 150ms)
	expectedMinDuration := 150 * time.Millisecond
	if duration < expectedMinDuration {
		t.Errorf("Expected operation to take at least %v, took %v", expectedMinDuration, duration)
	}

	// Check statistics
	stats := middleware.GetStatistics()
	if stats.TotalAttempts != 3 {
		t.Errorf("Expected 3 total attempts, got %d", stats.TotalAttempts)
	}

	if stats.TotalRetries != 2 {
		t.Errorf("Expected 2 retries, got %d", stats.TotalRetries)
	}

	if stats.SuccessfulRetries != 2 {
		t.Errorf("Expected 2 successful retries, got %d", stats.SuccessfulRetries)
	}
}

// TestRetryMiddlewareNonRetryableError tests middleware with non-retryable errors
func TestRetryMiddlewareNonRetryableError(t *testing.T) {
	strategy := NewExponentialBackoffStrategy(ExponentialBackoffConfig{
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
		MaxAttempts: 3,
	})

	middleware := NewRetryMiddleware(strategy, logger.Discard)

	msg := &message.Message{
		ID:    "test-message-3",
		Title: "Test Message",
		Body:  "Test body",
	}

	targets := []target.Target{
		{Type: "webhook", Value: "test", Platform: "feishu"},
	}

	// Non-retryable error
	nonRetryableErr := &errors.NotifyError{
		Code:     errors.ErrInvalidConfig,
		Message:  "invalid configuration",
		Platform: "feishu",
	}

	handler := mockSendHandler(10, nonRetryableErr) // Always fails
	receipt, err := middleware.HandleSend(context.Background(), msg, targets, handler)

	if err == nil {
		t.Error("Expected error for non-retryable operation")
	}

	if receipt != nil {
		t.Error("Expected no receipt for failed operation")
	}

	// Should only attempt once (no retries)
	stats := middleware.GetStatistics()
	if stats.TotalAttempts != 1 {
		t.Errorf("Expected 1 total attempt, got %d", stats.TotalAttempts)
	}
}

// TestRetryMiddlewareCircuitBreaker tests circuit breaker functionality
func TestRetryMiddlewareCircuitBreaker(t *testing.T) {
	strategy := NewExponentialBackoffStrategy(ExponentialBackoffConfig{
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
		Multiplier:  2.0,
		MaxAttempts: 3,
	})

	middleware := NewRetryMiddleware(strategy, logger.Discard)

	msg := &message.Message{
		ID:    "test-message-4",
		Title: "Test Message",
		Body:  "Test body",
	}

	targets := []target.Target{
		{Type: "webhook", Value: "test", Platform: "problematic"},
	}

	retryableErr := &errors.NotifyError{
		Code:     errors.ErrPlatformTimeout,
		Message:  "timeout error",
		Platform: "problematic",
	}

	// Trigger circuit breaker by failing multiple times
	handler := mockSendHandler(10, retryableErr) // Always fails

	// Make several failed attempts to trip the circuit breaker
	for i := 0; i < 6; i++ {
		_, _ = middleware.HandleSend(context.Background(), msg, targets, handler)
	}

	// Check circuit breaker status
	cbStatus := middleware.GetCircuitBreakerStatus("problematic")
	if cbStatus != errors.CircuitOpen {
		t.Errorf("Expected circuit breaker to be open, got status: %v", cbStatus)
	}

	// Next attempt should be blocked by circuit breaker
	receipt, err := middleware.HandleSend(context.Background(), msg, targets, handler)
	if err == nil {
		t.Error("Expected circuit breaker to block the operation")
	}

	if receipt != nil {
		t.Error("Expected no receipt when circuit breaker is open")
	}
}

// TestRetryMiddlewarePerformanceMonitoring tests performance monitoring features
func TestRetryMiddlewarePerformanceMonitoring(t *testing.T) {
	strategy := NewExponentialBackoffStrategy(ExponentialBackoffConfig{
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
		Multiplier:  2.0,
		MaxAttempts: 3,
	})

	middleware := NewRetryMiddleware(strategy, logger.Discard)

	msg := &message.Message{
		ID:    "test-message-5",
		Title: "Test Message",
		Body:  "Test body",
	}

	// Test different platforms
	platforms := []string{"feishu", "email", "webhook"}

	for _, platform := range platforms {
		targets := []target.Target{
			{Type: "webhook", Value: "test", Platform: platform},
		}

		// Successful operation
		handler := mockSendHandler(0, nil)
		_, err := middleware.HandleSend(context.Background(), msg, targets, handler)
		if err != nil {
			t.Errorf("Unexpected error for platform %s: %v", platform, err)
		}

		// Failed operation
		retryableErr := &errors.NotifyError{
			Code:     errors.ErrPlatformTimeout,
			Message:  "timeout error",
			Platform: platform,
		}
		handler = mockSendHandler(10, retryableErr)
		_, _ = middleware.HandleSend(context.Background(), msg, targets, handler)
	}

	// Check performance metrics
	performanceMonitor := middleware.GetPerformanceMetrics()
	if len(performanceMonitor.PlatformMetrics) != 3 {
		t.Errorf("Expected metrics for 3 platforms, got %d", len(performanceMonitor.PlatformMetrics))
	}

	// Check that each platform has recorded operations
	for _, platform := range platforms {
		metrics := performanceMonitor.PlatformMetrics[platform]
		if metrics == nil {
			t.Errorf("Expected metrics for platform %s", platform)
			continue
		}

		if metrics.TotalOperations != 2 {
			t.Errorf("Expected 2 operations for platform %s, got %d", platform, metrics.TotalOperations)
		}
	}
}

// TestRetryMiddlewareOptimization tests performance-based optimization
func TestRetryMiddlewareOptimization(t *testing.T) {
	strategy := NewExponentialBackoffStrategy(ExponentialBackoffConfig{
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
		Multiplier:  2.0,
		MaxAttempts: 3,
	})

	middleware := NewRetryMiddleware(strategy, logger.Discard)

	msg := &message.Message{
		ID:    "test-message-6",
		Title: "Test Message",
		Body:  "Test body",
	}

	targets := []target.Target{
		{Type: "webhook", Value: "test", Platform: "problematic"},
	}

	retryableErr := &errors.NotifyError{
		Code:     errors.ErrPlatformTimeout,
		Message:  "timeout error",
		Platform: "problematic",
	}

	// Create many failed operations to trigger optimization
	handler := mockSendHandler(10, retryableErr)
	for i := 0; i < 15; i++ {
		_, _ = middleware.HandleSend(context.Background(), msg, targets, handler)
	}

	// Test optimization
	currentConfig := &errors.PlatformRetryConfig{
		Platform:    "problematic",
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    10 * time.Second,
		Multiplier:  2.0,
		Enabled:     true,
	}

	optimizedConfig := middleware.OptimizePlatformConfig("problematic", currentConfig)

	// Should suggest increasing retry attempts for problematic platform
	if optimizedConfig.MaxAttempts <= currentConfig.MaxAttempts {
		t.Error("Expected optimization to increase max attempts for problematic platform")
	}

	// Test recommendations
	recommendations := middleware.GetRecommendations()
	if len(recommendations) == 0 {
		t.Error("Expected recommendations for problematic platform")
	}

	foundRecommendation := false
	for _, rec := range recommendations {
		if fmt.Sprintf("%v", rec) != "" {
			foundRecommendation = true
			break
		}
	}

	if !foundRecommendation {
		t.Error("Expected at least one recommendation")
	}
}

// TestRetryMiddlewareContextCancellation tests context cancellation behavior
func TestRetryMiddlewareContextCancellation(t *testing.T) {
	strategy := NewExponentialBackoffStrategy(ExponentialBackoffConfig{
		BaseDelay:   1 * time.Second, // Long delay to allow cancellation
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
		MaxAttempts: 5,
	})

	middleware := NewRetryMiddleware(strategy, logger.Discard)

	msg := &message.Message{
		ID:    "test-message-7",
		Title: "Test Message",
		Body:  "Test body",
	}

	targets := []target.Target{
		{Type: "webhook", Value: "test", Platform: "feishu"},
	}

	retryableErr := &errors.NotifyError{
		Code:     errors.ErrPlatformTimeout,
		Message:  "timeout error",
		Platform: "feishu",
	}

	handler := mockSendHandler(10, retryableErr) // Always fails

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	receipt, err := middleware.HandleSend(ctx, msg, targets, handler)
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	if receipt != nil {
		t.Error("Expected no receipt when context is cancelled")
	}

	// Should complete quickly due to context cancellation
	if duration > 500*time.Millisecond {
		t.Errorf("Expected operation to be cancelled quickly, took %v", duration)
	}
}

// TestRetryMiddlewareReset tests the reset functionality
func TestRetryMiddlewareReset(t *testing.T) {
	strategy := NewExponentialBackoffStrategy(ExponentialBackoffConfig{
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
		Multiplier:  2.0,
		MaxAttempts: 3,
	})

	middleware := NewRetryMiddleware(strategy, logger.Discard)

	msg := &message.Message{
		ID:    "test-message-8",
		Title: "Test Message",
		Body:  "Test body",
	}

	targets := []target.Target{
		{Type: "webhook", Value: "test", Platform: "feishu"},
	}

	// Generate some statistics
	handler := mockSendHandler(0, nil)
	_, _ = middleware.HandleSend(context.Background(), msg, targets, handler)

	// Verify statistics exist
	stats := middleware.GetStatistics()
	if stats.TotalAttempts == 0 {
		t.Error("Expected some statistics before reset")
	}

	// Reset middleware
	middleware.Reset()

	// Verify statistics are cleared
	stats = middleware.GetStatistics()
	if stats.TotalAttempts != 0 {
		t.Error("Expected statistics to be cleared after reset")
	}

	performanceMonitor := middleware.GetPerformanceMetrics()
	if len(performanceMonitor.PlatformMetrics) != 0 {
		t.Error("Expected performance metrics to be cleared after reset")
	}
}

// BenchmarkRetryMiddleware benchmarks the retry middleware performance
func BenchmarkRetryMiddleware(b *testing.B) {
	strategy := NewExponentialBackoffStrategy(ExponentialBackoffConfig{
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
		MaxAttempts: 3,
	})

	middleware := NewRetryMiddleware(strategy, logger.Discard)

	msg := &message.Message{
		ID:    "benchmark-message",
		Title: "Benchmark Message",
		Body:  "Benchmark body",
	}

	targets := []target.Target{
		{Type: "webhook", Value: "test", Platform: "feishu"},
	}

	// Successful handler (no retries)
	handler := mockSendHandler(0, nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = middleware.HandleSend(context.Background(), msg, targets, handler)
	}
}

// BenchmarkRetryMiddlewareWithRetries benchmarks middleware with retries
func BenchmarkRetryMiddlewareWithRetries(b *testing.B) {
	strategy := NewExponentialBackoffStrategy(ExponentialBackoffConfig{
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    5 * time.Millisecond,
		Multiplier:  2.0,
		MaxAttempts: 3,
	})

	middleware := NewRetryMiddleware(strategy, logger.Discard)

	msg := &message.Message{
		ID:    "benchmark-message-retry",
		Title: "Benchmark Message",
		Body:  "Benchmark body",
	}

	targets := []target.Target{
		{Type: "webhook", Value: "test", Platform: "feishu"},
	}

	retryableErr := &errors.NotifyError{
		Code:     errors.ErrPlatformTimeout,
		Message:  "timeout error",
		Platform: "feishu",
	}

	// Handler that fails twice, then succeeds
	handler := mockSendHandler(2, retryableErr)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = middleware.HandleSend(context.Background(), msg, targets, handler)
	}
}