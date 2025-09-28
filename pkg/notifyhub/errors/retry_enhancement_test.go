// Package errors provides comprehensive tests for enhanced retry strategies
package errors

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// TestExponentialBackoffStrategyEnhanced tests the enhanced exponential backoff strategy
func TestExponentialBackoffStrategyEnhanced(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()

	// Test platform-specific configuration
	platformConfig := &PlatformRetryConfig{
		Platform:    "feishu",
		MaxAttempts: 3,
		BaseDelay:   2 * time.Second,
		MaxDelay:    20 * time.Second,
		Multiplier:  1.5,
		Jitter:      0.2,
		Enabled:     true,
	}

	err := strategy.UpdatePlatformConfig("feishu", platformConfig)
	if err != nil {
		t.Fatalf("Failed to update platform config: %v", err)
	}

	// Test platform config retrieval
	retrievedConfig := strategy.GetPlatformConfig("feishu")
	if retrievedConfig == nil {
		t.Fatal("Failed to retrieve platform config")
	}

	if retrievedConfig.Platform != "feishu" {
		t.Errorf("Expected platform 'feishu', got '%s'", retrievedConfig.Platform)
	}

	// Test ShouldRetry with platform-specific config
	feishuErr := &NotifyError{
		Code:     ErrPlatformTimeout,
		Message:  "timeout",
		Platform: "feishu",
	}

	// Should retry within platform limits
	if !strategy.ShouldRetry(feishuErr, 1) {
		t.Error("Expected retry for feishu platform within limits")
	}

	// Should not retry beyond platform limits
	if strategy.ShouldRetry(feishuErr, 5) {
		t.Error("Expected no retry for feishu platform beyond limits")
	}
}

// TestRetryStatistics tests the enhanced retry statistics functionality
func TestRetryStatistics(t *testing.T) {
	stats := NewRetryStatistics()

	// Record some attempts with proper modeling of initial attempt + retries
	stats.RecordAttempt("feishu", 0, nil, false) // Initial attempt (failed) - not counted in FailedRetries
	stats.RecordAttempt("feishu", 100*time.Millisecond, nil, false) // First retry (failed)
	stats.RecordAttempt("feishu", 200*time.Millisecond, nil, true) // Second retry (success)
	stats.RecordAttempt("email", 50*time.Millisecond, nil, true) // First retry (success)

	// Test success rate calculation
	successRate := stats.GetSuccessRate()

	// Debug: print actual values
	t.Logf("TotalAttempts: %d, FailedRetries: %d, SuccessfulRetries: %d",
		stats.TotalAttempts, stats.FailedRetries, stats.SuccessfulRetries)

	// Success rate = (TotalAttempts - FailedRetries) / TotalAttempts
	// TotalAttempts = 4, FailedRetries = 1 (only retry with delay > 0 that failed)
	// So success rate = (4 - 1) / 4 = 0.75
	expectedRate := 0.75
	if successRate != expectedRate {
		t.Errorf("Expected success rate %.2f, got %.2f", expectedRate, successRate)
	}

	// Test retry rate calculation
	retryRate := stats.GetRetryRate()
	// TotalRetries = 3 (all attempts with delay > 0), TotalAttempts = 4
	expectedRetryRate := 3.0 / 4.0 // 3 retries out of 4 total attempts
	if retryRate != expectedRetryRate {
		t.Errorf("Expected retry rate %.2f, got %.2f", expectedRetryRate, retryRate)
	}

	// Test platform statistics
	if len(stats.PlatformStats) != 2 {
		t.Errorf("Expected 2 platforms in stats, got %d", len(stats.PlatformStats))
	}

	feishuStats := stats.PlatformStats["feishu"]
	if feishuStats == nil {
		t.Fatal("Expected feishu platform stats")
	}

	if feishuStats.Attempts != 3 {
		t.Errorf("Expected 3 feishu attempts, got %d", feishuStats.Attempts)
	}

	if feishuStats.Successes != 1 {
		t.Errorf("Expected 1 feishu success, got %d", feishuStats.Successes)
	}

	if feishuStats.Failures != 2 {
		t.Errorf("Expected 2 feishu failures, got %d", feishuStats.Failures)
	}
}

// TestCircuitBreaker tests the circuit breaker functionality
func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	// Initially should be closed
	if cb.GetState() != CircuitClosed {
		t.Error("Expected circuit breaker to be closed initially")
	}

	if !cb.ShouldAllow() {
		t.Error("Expected circuit breaker to allow operations when closed")
	}

	// Record failures to trip the circuit breaker
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Should now be open
	if cb.GetState() != CircuitOpen {
		t.Error("Expected circuit breaker to be open after threshold failures")
	}

	if cb.ShouldAllow() {
		t.Error("Expected circuit breaker to block operations when open")
	}

	// Test recovery after timeout
	time.Sleep(1100 * time.Millisecond) // Wait for timeout + margin

	if cb.GetState() != CircuitHalfOpen {
		t.Error("Expected circuit breaker to be half-open after timeout")
	}

	// Record success to close the circuit
	cb.RecordSuccess()

	if cb.GetState() != CircuitClosed {
		t.Error("Expected circuit breaker to be closed after success")
	}
}

// TestPerformanceMonitor tests the performance monitoring functionality
func TestPerformanceMonitor(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Record some operations
	monitor.RecordOperation("feishu", 100*time.Millisecond, true)
	monitor.RecordOperation("feishu", 200*time.Millisecond, true)
	monitor.RecordOperation("feishu", 150*time.Millisecond, false)
	monitor.RecordOperation("email", 50*time.Millisecond, true)

	// Test platform metrics
	feishuMetrics := monitor.PlatformMetrics["feishu"]
	if feishuMetrics == nil {
		t.Fatal("Expected feishu platform metrics")
	}

	if feishuMetrics.TotalOperations != 3 {
		t.Errorf("Expected 3 feishu operations, got %d", feishuMetrics.TotalOperations)
	}

	if feishuMetrics.SuccessfulOps != 2 {
		t.Errorf("Expected 2 feishu successes, got %d", feishuMetrics.SuccessfulOps)
	}

	if feishuMetrics.FailedOps != 1 {
		t.Errorf("Expected 1 feishu failure, got %d", feishuMetrics.FailedOps)
	}

	expectedSuccessRate := 2.0 / 3.0
	if feishuMetrics.SuccessRate != expectedSuccessRate {
		t.Errorf("Expected feishu success rate %.2f, got %.2f", expectedSuccessRate, feishuMetrics.SuccessRate)
	}

	// Test latency statistics
	if feishuMetrics.MinLatency != 100*time.Millisecond {
		t.Errorf("Expected min latency 100ms, got %v", feishuMetrics.MinLatency)
	}

	if feishuMetrics.MaxLatency != 200*time.Millisecond {
		t.Errorf("Expected max latency 200ms, got %v", feishuMetrics.MaxLatency)
	}
}

// TestPerformanceOptimization tests the performance-based optimization
func TestPerformanceOptimization(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Simulate poor performance platform
	for i := 0; i < 15; i++ {
		monitor.RecordOperation("problematic", 100*time.Millisecond, false)
	}

	// Test optimization
	currentConfig := &PlatformRetryConfig{
		Platform:    "problematic",
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    10 * time.Second,
		Multiplier:  2.0,
		Enabled:     true,
	}

	optimizedConfig := monitor.OptimizeStrategy("problematic", currentConfig)

	// Should increase retry attempts for problematic platform
	if optimizedConfig.MaxAttempts <= currentConfig.MaxAttempts {
		t.Error("Expected optimization to increase max attempts for problematic platform")
	}

	// Test recommendations
	recommendations := monitor.GetRecommendations()
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
		t.Error("Expected at least one recommendation for problematic platform")
	}
}

// TestJitterTypes tests different jitter algorithms
func TestJitterTypes(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()

	jitterTypes := []JitterType{UniformJitter, FullJitter, ExponentialJitter, DecorrelatedJitter}

	for _, jitterType := range jitterTypes {
		strategy.JitterType = jitterType
		strategy.Jitter = 0.5

		delays := make([]time.Duration, 10)
		for i := 0; i < 10; i++ {
			delays[i] = strategy.GetDelay(1)
		}

		// Check that delays are different (jitter working)
		allSame := true
		firstDelay := delays[0]
		for _, delay := range delays[1:] {
			if delay != firstDelay {
				allSame = false
				break
			}
		}

		if allSame {
			t.Errorf("Expected different delays with jitter type %v, but all delays were the same", jitterType)
		}
	}
}

// TestConditionalRetryPolicies tests conditional retry policies
func TestConditionalRetryPolicies(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()

	// Add a conditional policy for rate limit errors
	rateLimitPolicy := ConditionalRetryPolicy{
		Name: "RateLimitPolicy",
		Condition: func(err error, attempt int) bool {
			if notifyErr, ok := err.(*NotifyError); ok {
				return notifyErr.Code == ErrPlatformRateLimit
			}
			return false
		},
		Action: func(err error, attempt int) RetryAction {
			return RetryAction{
				ShouldRetry: attempt < 10, // Allow more retries for rate limits
				Delay:       time.Duration(attempt) * 5 * time.Second,
				MaxAttempts: 10,
			}
		},
	}

	strategy.ConditionalPolicies = append(strategy.ConditionalPolicies, rateLimitPolicy)

	rateLimitErr := &NotifyError{
		Code:     ErrPlatformRateLimit,
		Message:  "rate limit exceeded",
		Platform: "feishu",
	}

	// Should trigger the conditional policy
	shouldRetry := strategy.ShouldRetry(rateLimitErr, 8)
	if !shouldRetry {
		t.Error("Expected conditional policy to allow retry for rate limit error")
	}
}

// TestConcurrentRetryOperations tests thread safety of retry operations
func TestConcurrentRetryOperations(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()
	stats := strategy.GetStatistics()

	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 10

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				platform := fmt.Sprintf("platform-%d", goroutineID%5)
				success := j%2 == 0

				stats.RecordAttempt(platform, time.Duration(j)*time.Millisecond, nil, success)
				_ = strategy.GetDelay(j)
				_ = strategy.ShouldRetry(nil, j)
			}
		}(i)
	}

	wg.Wait()

	// Verify that all operations were recorded
	expectedTotal := int64(numGoroutines * operationsPerGoroutine)
	if stats.TotalAttempts != expectedTotal {
		t.Errorf("Expected %d total attempts, got %d", expectedTotal, stats.TotalAttempts)
	}

	// Verify platform stats were collected
	if len(stats.PlatformStats) != 5 { // We used 5 different platforms
		t.Errorf("Expected 5 platforms in stats, got %d", len(stats.PlatformStats))
	}
}

// TestRetryExecutorWithAdvancedFeatures tests the retry executor with enhanced features
func TestRetryExecutorWithAdvancedFeatures(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()
	logger := logger.Discard
	executor := NewRetryExecutor(strategy, logger)

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return &NotifyError{
				Code:     ErrPlatformTimeout,
				Message:  "timeout",
				Platform: "test",
			}
		}
		return nil
	}

	ctx := context.Background()
	err := executor.Execute(ctx, operation)

	if err != nil {
		t.Errorf("Expected operation to succeed after retries, got error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 operation calls, got %d", callCount)
	}

	// Test with callback
	retryCallbackCount := 0
	callCount = 0

	err = executor.ExecuteWithCallback(ctx, operation, func(attempt int, err error, delay time.Duration) {
		retryCallbackCount++
	})

	if err != nil {
		t.Errorf("Expected operation to succeed with callback, got error: %v", err)
	}

	if retryCallbackCount != 2 { // 2 retries before success
		t.Errorf("Expected 2 retry callbacks, got %d", retryCallbackCount)
	}
}

// BenchmarkRetryStrategy benchmarks the enhanced retry strategy
func BenchmarkRetryStrategy(b *testing.B) {
	strategy := NewExponentialBackoffStrategy()

	err := &NotifyError{
		Code:     ErrPlatformTimeout,
		Message:  "timeout",
		Platform: "feishu",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = strategy.ShouldRetry(err, i%5)
		_ = strategy.GetDelay(i % 5)
		strategy.RecordAttempt("feishu", time.Duration(i)*time.Microsecond, err, i%2 == 0)
	}
}

// BenchmarkPerformanceMonitorEnhanced benchmarks the performance monitor
func BenchmarkPerformanceMonitorEnhanced(b *testing.B) {
	monitor := NewPerformanceMonitor()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		platform := fmt.Sprintf("platform-%d", i%10)
		latency := time.Duration(i%1000) * time.Microsecond
		success := i%2 == 0

		monitor.RecordOperation(platform, latency, success)
	}
}