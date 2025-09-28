// Package errors provides comprehensive tests for error handling functionality
package errors

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

// TestErrorCodeClassification tests comprehensive error code classification
func TestErrorCodeClassification(t *testing.T) {
	tests := []struct {
		name           string
		code           Code
		expectedCat    string
		expectedSev    string
		expectedRetry  bool
		description    string
	}{
		// Configuration Error Classification
		{
			name: "config load failure should be retryable",
			code: ErrConfigLoadFailed,
			expectedCat: ConfigurationCategory,
			expectedSev: "ERROR",
			expectedRetry: true,
			description: "Configuration load failures might be temporary",
		},
		{
			name: "invalid config should not be retryable",
			code: ErrInvalidConfig,
			expectedCat: ConfigurationCategory,
			expectedSev: "ERROR",
			expectedRetry: false,
			description: "Invalid configuration requires manual intervention",
		},

		// Platform Error Classification
		{
			name: "platform maintenance should be retryable",
			code: ErrPlatformMaintenance,
			expectedCat: PlatformCategory,
			expectedSev: "WARN",
			expectedRetry: true,
			description: "Platform maintenance is temporary",
		},
		{
			name: "platform auth should not be retryable",
			code: ErrPlatformAuth,
			expectedCat: PlatformCategory,
			expectedSev: "ERROR",
			expectedRetry: false,
			description: "Authentication failures require credential update",
		},

		// Queue Error Classification
		{
			name: "queue worker failure should be retryable",
			code: ErrQueueWorkerFailed,
			expectedCat: QueueCategory,
			expectedSev: "ERROR",
			expectedRetry: true,
			description: "Worker failures might be temporary",
		},
		{
			name: "queue serialization should not be retryable",
			code: ErrQueueSerialization,
			expectedCat: QueueCategory,
			expectedSev: "ERROR",
			expectedRetry: false,
			description: "Serialization errors indicate data format issues",
		},

		// System Error Classification
		{
			name: "system overload should be retryable with high severity",
			code: ErrSystemOverload,
			expectedCat: SystemCategory,
			expectedSev: "CRITICAL",
			expectedRetry: true,
			description: "System overload is temporary but critical",
		},
		{
			name: "permission denied should not be retryable",
			code: ErrPermissionDenied,
			expectedCat: SystemCategory,
			expectedSev: "ERROR",
			expectedRetry: false,
			description: "Permission issues require authorization fix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := GetErrorInfo(tt.code)

			if info.Category != tt.expectedCat {
				t.Errorf("Expected category %v, got %v", tt.expectedCat, info.Category)
			}

			if info.Severity != tt.expectedSev {
				t.Errorf("Expected severity %v, got %v", tt.expectedSev, info.Severity)
			}

			if info.Retryable != tt.expectedRetry {
				t.Errorf("Expected retryable %v, got %v", tt.expectedRetry, info.Retryable)
			}

			if info.Description == "" {
				t.Error("Expected non-empty description")
			}

			// Test helper functions
			if GetCategory(tt.code) != tt.expectedCat {
				t.Errorf("GetCategory mismatch for %v", tt.code)
			}

			if GetSeverity(tt.code) != tt.expectedSev {
				t.Errorf("GetSeverity mismatch for %v", tt.code)
			}

			if IsRetryable(tt.code) != tt.expectedRetry {
				t.Errorf("IsRetryable mismatch for %v", tt.code)
			}
		})
	}
}

// TestErrorCreationEdgeCases tests edge cases in error creation
func TestErrorCreationEdgeCases(t *testing.T) {
	t.Run("empty error message", func(t *testing.T) {
		err := New(ErrInvalidMessage, "")
		if err.Message != "" {
			t.Errorf("Expected empty message to be preserved, got %v", err.Message)
		}
	})

	t.Run("very long error message", func(t *testing.T) {
		longMessage := make([]byte, 10000)
		for i := range longMessage {
			longMessage[i] = 'a'
		}

		err := New(ErrInvalidMessage, string(longMessage))
		if err.Message != string(longMessage) {
			t.Error("Expected long message to be preserved")
		}
	})

	t.Run("unicode error message", func(t *testing.T) {
		unicodeMessage := "错误消息 🚨 エラーメッセージ"
		err := New(ErrInvalidMessage, unicodeMessage)
		if err.Message != unicodeMessage {
			t.Error("Expected unicode message to be preserved")
		}
	})

	t.Run("nil context handling", func(t *testing.T) {
		err := &NotifyError{
			Code:    ErrInvalidMessage,
			Message: "test",
			Context: nil,
		}

		err.WithContext("key", "value")
		if err.Context == nil {
			t.Error("Expected context to be initialized")
		}

		if err.Context["key"] != "value" {
			t.Error("Expected context value to be set")
		}
	})

	t.Run("context with complex values", func(t *testing.T) {
		err := New(ErrInvalidMessage, "test")

		complexValue := map[string]interface{}{
			"nested": map[string]int{"count": 42},
			"list":   []string{"a", "b", "c"},
		}

		err.WithContext("complex", complexValue)
		if !reflect.DeepEqual(err.Context["complex"], complexValue) {
			t.Error("Expected complex context value to be preserved")
		}
	})
}

// TestErrorBuilderAdvanced tests advanced error builder functionality
func TestErrorBuilderAdvanced(t *testing.T) {
	t.Run("chained builder operations", func(t *testing.T) {
		cause := errors.New("original error")

		err := NewErrorBuilder(ErrPlatformInternal, "platform error").
			WithDetails("Internal server error occurred").
			WithContext("platform", "feishu").
			WithContext("endpoint", "/webhook").
			WithContext("retry_count", 3).
			WithCause(cause).
			WithStackTrace([]string{"main", "handler", "process"}).
			Build()

		// Verify all fields are set correctly
		if err.Code != ErrPlatformInternal {
			t.Errorf("Expected code %v, got %v", ErrPlatformInternal, err.Code)
		}

		if err.Details != "Internal server error occurred" {
			t.Error("Details not set correctly")
		}

		if err.Context["platform"] != "feishu" {
			t.Error("Platform context not set correctly")
		}

		if err.Context["retry_count"] != 3 {
			t.Error("Retry count context not set correctly")
		}

		if err.Cause != cause {
			t.Error("Cause not set correctly")
		}

		if len(err.StackTrace) != 3 {
			t.Error("Stack trace not set correctly")
		}

		if err.Timestamp.IsZero() {
			t.Error("Timestamp should be set automatically")
		}
	})

	t.Run("builder reuse", func(t *testing.T) {
		builder := NewErrorBuilder(ErrNetworkTimeout, "base message")

		err1 := builder.WithContext("attempt", 1).Build()
		err2 := builder.WithContext("attempt", 2).Build()

		// Both errors should have attempt context, but err2 should override
		if err1.Context["attempt"] != 2 {
			t.Error("Expected builder state to persist across builds")
		}

		if err2.Context["attempt"] != 2 {
			t.Error("Expected second build to use updated context")
		}
	})

	t.Run("nil values handling", func(t *testing.T) {
		err := NewErrorBuilder(ErrInvalidMessage, "test").
			WithCause(nil).
			WithStackTrace(nil).
			Build()

		if err.Cause != nil {
			t.Error("Expected nil cause to be preserved")
		}

		if err.StackTrace != nil {
			t.Error("Expected nil stack trace to be preserved")
		}
	})
}

// TestErrorAggregationScenarios tests complex error aggregation scenarios
func TestErrorAggregationScenarios(t *testing.T) {
	t.Run("multi-platform error aggregation", func(t *testing.T) {
		aggregator := NewErrorAggregator()

		// Add errors from different platforms
		feishuErr := NewPlatformError(ErrPlatformTimeout, "feishu", "timeout sending to feishu")
		emailErr := NewPlatformError(ErrPlatformAuth, "email", "email authentication failed")
		webhookErr := NewPlatformError(ErrPlatformUnavailable, "webhook", "webhook service down")
		slackErr := NewPlatformError(ErrPlatformRateLimit, "slack", "rate limit exceeded")

		aggregator.Add(feishuErr)
		aggregator.Add(emailErr)
		aggregator.Add(webhookErr)
		aggregator.Add(slackErr)

		aggregatedErr := aggregator.ToError()
		if aggregatedErr == nil {
			t.Fatal("Expected aggregated error")
		}

		notifyErr, ok := aggregatedErr.(*NotifyError)
		if !ok {
			t.Fatal("Expected NotifyError type")
		}

		// Check aggregated context
		errorCount := notifyErr.Context["error_count"]
		if errorCount != 4 {
			t.Errorf("Expected error count 4, got %v", errorCount)
		}

		platforms := notifyErr.Context["affected_platforms"]
		if platforms == nil {
			t.Error("Expected affected_platforms in context")
		}

		platformMap, ok := platforms.(map[string]int)
		if !ok {
			t.Error("Expected affected_platforms to be map[string]int")
		}

		expectedPlatforms := map[string]int{
			"feishu":  1,
			"email":   1,
			"webhook": 1,
			"slack":   1,
		}

		if !reflect.DeepEqual(platformMap, expectedPlatforms) {
			t.Errorf("Expected platforms %v, got %v", expectedPlatforms, platformMap)
		}
	})

	t.Run("error code distribution analysis", func(t *testing.T) {
		aggregator := NewErrorAggregator()

		// Add multiple errors of same type
		for i := 0; i < 5; i++ {
			aggregator.Add(New(ErrPlatformTimeout, "timeout"))
		}

		for i := 0; i < 3; i++ {
			aggregator.Add(New(ErrNetworkConnection, "connection failed"))
		}

		for i := 0; i < 2; i++ {
			aggregator.Add(New(ErrQueueFull, "queue full"))
		}

		aggregatedErr := aggregator.ToError()
		notifyErr := aggregatedErr.(*NotifyError)

		errorCodes := notifyErr.Context["error_codes"]
		if errorCodes == nil {
			t.Error("Expected error_codes in context")
		}

		codeMap, ok := errorCodes.(map[Code]int64)
		if !ok {
			t.Error("Expected error_codes to be map[Code]int64")
		}

		if codeMap[ErrPlatformTimeout] != 5 {
			t.Errorf("Expected 5 platform timeout errors, got %v", codeMap[ErrPlatformTimeout])
		}

		if codeMap[ErrNetworkConnection] != 3 {
			t.Errorf("Expected 3 network connection errors, got %v", codeMap[ErrNetworkConnection])
		}

		if codeMap[ErrQueueFull] != 2 {
			t.Errorf("Expected 2 queue full errors, got %v", codeMap[ErrQueueFull])
		}
	})

	t.Run("large scale aggregation performance", func(t *testing.T) {
		aggregator := NewErrorAggregator()

		// Add many errors to test performance
		start := time.Now()
		for i := 0; i < 1000; i++ {
			platform := fmt.Sprintf("platform-%d", i%10)
			code := []Code{ErrPlatformTimeout, ErrNetworkConnection, ErrQueueFull}[i%3]
			err := NewPlatformError(code, platform, "error message")
			aggregator.Add(err)
		}

		aggregatedErr := aggregator.ToError()
		duration := time.Since(start)

		if aggregatedErr == nil {
			t.Error("Expected aggregated error for large scale test")
		}

		if duration > 100*time.Millisecond {
			t.Errorf("Large scale aggregation took too long: %v", duration)
		}

		// Verify aggregation correctness
		if aggregator.Count() != 1000 {
			t.Errorf("Expected 1000 errors, got %d", aggregator.Count())
		}
	})
}

// TestRetryStrategyAlgorithms tests detailed retry strategy algorithms
func TestRetryStrategyAlgorithms(t *testing.T) {
	t.Run("exponential backoff precision", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()
		strategy.BaseDelay = 100 * time.Millisecond
		strategy.Multiplier = 2.0
		strategy.MaxDelay = 10 * time.Second
		strategy.Jitter = 0 // Disable jitter for precision testing

		expectedDelays := []time.Duration{
			100 * time.Millisecond,  // attempt 0
			200 * time.Millisecond,  // attempt 1: 100 * 2^1
			400 * time.Millisecond,  // attempt 2: 100 * 2^2
			800 * time.Millisecond,  // attempt 3: 100 * 2^3
			1600 * time.Millisecond, // attempt 4: 100 * 2^4
			3200 * time.Millisecond, // attempt 5: 100 * 2^5
			6400 * time.Millisecond, // attempt 6: 100 * 2^6
			10 * time.Second,        // attempt 7: capped at MaxDelay
			10 * time.Second,        // attempt 8: capped at MaxDelay
		}

		for i, expected := range expectedDelays {
			actual := strategy.GetDelay(i)
			if actual != expected {
				t.Errorf("Attempt %d: expected delay %v, got %v", i, expected, actual)
			}
		}
	})

	t.Run("jitter algorithm effectiveness", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()
		strategy.BaseDelay = 1 * time.Second
		strategy.Jitter = 0.5

		jitterTypes := []JitterType{UniformJitter, FullJitter, ExponentialJitter, DecorrelatedJitter}

		for _, jitterType := range jitterTypes {
			strategy.JitterType = jitterType
			strategy.lastDelayUsed = 0 // Reset for decorrelated jitter

			delays := make([]time.Duration, 50) // Reduce sample size
			for i := 0; i < 50; i++ {
				delays[i] = strategy.GetDelay(1) // Use attempt 1 for more predictable results
			}

			// Calculate variance to ensure jitter is working
			mean := calculateMean(delays)
			variance := calculateVariance(delays, mean)

			if variance == 0 {
				t.Errorf("Jitter type %v produced no variance", jitterType)
			}

			// Ensure delays are within reasonable bounds
			for i, delay := range delays {
				if delay < 0 {
					t.Errorf("Jitter type %v produced negative delay at index %d: %v", jitterType, i, delay)
				}

				// For most jitter types, delay should be within reasonable bounds
				// DecorrelatedJitter can produce larger delays, so we skip this check for it
				if jitterType != FullJitter && jitterType != DecorrelatedJitter && delay > 10*time.Second {
					t.Errorf("Jitter type %v produced unreasonably large delay: %v", jitterType, delay)
				}
			}
		}
	})

	t.Run("linear backoff progression", func(t *testing.T) {
		strategy := NewLinearBackoffStrategy()
		strategy.BaseDelay = 1 * time.Second
		strategy.DelayIncrement = 500 * time.Millisecond
		strategy.MaxDelay = 5 * time.Second

		expectedDelays := []time.Duration{
			1 * time.Second,                    // 1s + 0*0.5s = 1s
			1500 * time.Millisecond,           // 1s + 1*0.5s = 1.5s
			2 * time.Second,                   // 1s + 2*0.5s = 2s
			2500 * time.Millisecond,           // 1s + 3*0.5s = 2.5s
			3 * time.Second,                   // 1s + 4*0.5s = 3s
			3500 * time.Millisecond,           // 1s + 5*0.5s = 3.5s
			4 * time.Second,                   // 1s + 6*0.5s = 4s
			4500 * time.Millisecond,           // 1s + 7*0.5s = 4.5s
			5 * time.Second,                   // 1s + 8*0.5s = 5s (capped)
			5 * time.Second,                   // capped at MaxDelay
		}

		for i, expected := range expectedDelays {
			actual := strategy.GetDelay(i)
			if actual != expected {
				t.Errorf("Attempt %d: expected delay %v, got %v", i, expected, actual)
			}
		}
	})

	t.Run("fixed delay consistency", func(t *testing.T) {
		delay := 3 * time.Second
		maxAttempts := 5
		strategy := NewFixedDelayStrategy(delay, maxAttempts)

		// Test that delay is always the same
		for i := 0; i < 20; i++ {
			actual := strategy.GetDelay(i)
			if actual != delay {
				t.Errorf("Attempt %d: expected fixed delay %v, got %v", i, delay, actual)
			}
		}

		if strategy.GetMaxAttempts() != maxAttempts {
			t.Errorf("Expected max attempts %d, got %d", maxAttempts, strategy.GetMaxAttempts())
		}
	})
}

// TestCircuitBreakerAdvanced tests advanced circuit breaker scenarios
func TestCircuitBreakerAdvanced(t *testing.T) {
	t.Run("circuit breaker state transitions", func(t *testing.T) {
		threshold := int64(3)
		timeout := 500 * time.Millisecond
		cb := NewCircuitBreaker(threshold, timeout)

		// Initially closed
		if cb.GetState() != CircuitClosed {
			t.Error("Expected initial state to be closed")
		}

		// Record failures to approach threshold
		for i := int64(0); i < threshold-1; i++ {
			cb.RecordFailure()
			if cb.GetState() != CircuitClosed {
				t.Errorf("Expected state to remain closed after %d failures", i+1)
			}
		}

		// One more failure should open the circuit
		cb.RecordFailure()
		if cb.GetState() != CircuitOpen {
			t.Error("Expected state to be open after threshold failures")
		}

		// Should not allow operations when open
		if cb.ShouldAllow() {
			t.Error("Expected circuit to block operations when open")
		}

		// Wait for timeout
		time.Sleep(timeout + 50*time.Millisecond)

		// Should transition to half-open
		if cb.GetState() != CircuitHalfOpen {
			t.Error("Expected state to be half-open after timeout")
		}

		// Should allow operations when half-open
		if !cb.ShouldAllow() {
			t.Error("Expected circuit to allow operations when half-open")
		}

		// Success should close the circuit
		cb.RecordSuccess()
		if cb.GetState() != CircuitClosed {
			t.Error("Expected state to be closed after success in half-open")
		}

		// Failure in half-open should open again
		cb.RecordFailure()
		cb.RecordFailure()
		cb.RecordFailure()
		time.Sleep(timeout + 50*time.Millisecond)

		if cb.GetState() != CircuitHalfOpen {
			t.Error("Expected state to be half-open after second timeout")
		}

		cb.RecordFailure() // Failure in half-open
		if cb.GetState() != CircuitOpen {
			t.Error("Expected state to be open after failure in half-open")
		}
	})

	t.Run("circuit breaker concurrent access", func(t *testing.T) {
		cb := NewCircuitBreaker(5, 100*time.Millisecond)

		var wg sync.WaitGroup
		numGoroutines := 50

		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < 10; j++ {
					_ = cb.ShouldAllow()

					if id%2 == 0 {
						cb.RecordSuccess()
					} else {
						cb.RecordFailure()
					}

					_ = cb.GetState()
				}
			}(i)
		}

		wg.Wait()

		// Should not crash and should be in a valid state
		state := cb.GetState()
		if state != CircuitClosed && state != CircuitOpen && state != CircuitHalfOpen {
			t.Errorf("Invalid circuit breaker state after concurrent access: %v", state)
		}
	})

	t.Run("circuit breaker reset", func(t *testing.T) {
		cb := NewCircuitBreaker(2, 1*time.Second)

		// Trip the circuit
		cb.RecordFailure()
		cb.RecordFailure()

		if cb.GetState() != CircuitOpen {
			t.Error("Expected circuit to be open before reset")
		}

		// Reset should return to closed state
		cb.Reset()

		if cb.GetState() != CircuitClosed {
			t.Error("Expected circuit to be closed after reset")
		}

		if !cb.ShouldAllow() {
			t.Error("Expected circuit to allow operations after reset")
		}
	})
}

// TestRetryabilityJudgment tests comprehensive retryability judgment
func TestRetryabilityJudgment(t *testing.T) {
	t.Run("platform-specific retry configuration", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()

		// Configure different retry settings for different platforms
		feishuConfig := &PlatformRetryConfig{
			Platform:    "feishu",
			MaxAttempts: 5,
			BaseDelay:   1 * time.Second,
			MaxDelay:    30 * time.Second,
			Multiplier:  2.0,
			Jitter:      0.1,
			Enabled:     true,
		}

		emailConfig := &PlatformRetryConfig{
			Platform:    "email",
			MaxAttempts: 3,
			BaseDelay:   2 * time.Second,
			MaxDelay:    20 * time.Second,
			Multiplier:  1.5,
			Jitter:      0.2,
			Enabled:     true,
		}

		slackConfig := &PlatformRetryConfig{
			Platform:    "slack",
			MaxAttempts: 0,
			Enabled:     false, // Disable retries for slack
		}

		strategy.UpdatePlatformConfig("feishu", feishuConfig)
		strategy.UpdatePlatformConfig("email", emailConfig)
		strategy.UpdatePlatformConfig("slack", slackConfig)

		// Test feishu retries
		feishuErr := &NotifyError{
			Code:     ErrPlatformTimeout,
			Message:  "timeout",
			Platform: "feishu",
		}

		for i := 0; i < 4; i++ {
			if !strategy.ShouldRetry(feishuErr, i) {
				t.Errorf("Expected feishu retry for attempt %d", i)
			}
		}

		if strategy.ShouldRetry(feishuErr, 5) {
			t.Error("Expected feishu retry to be blocked at attempt 5")
		}

		// Test email retries
		emailErr := &NotifyError{
			Code:     ErrPlatformTimeout,
			Message:  "timeout",
			Platform: "email",
		}

		for i := 0; i < 2; i++ {
			if !strategy.ShouldRetry(emailErr, i) {
				t.Errorf("Expected email retry for attempt %d", i)
			}
		}

		if strategy.ShouldRetry(emailErr, 3) {
			t.Error("Expected email retry to be blocked at attempt 3")
		}

		// Test slack retries (disabled)
		slackErr := &NotifyError{
			Code:     ErrPlatformTimeout,
			Message:  "timeout",
			Platform: "slack",
		}

		if strategy.ShouldRetry(slackErr, 0) {
			t.Error("Expected slack retry to be disabled")
		}
	})

	t.Run("conditional retry policies", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()

		// Add conditional policy for rate limit errors
		rateLimitPolicy := ConditionalRetryPolicy{
			Name: "RateLimitExtendedRetry",
			Condition: func(err error, attempt int) bool {
				if notifyErr, ok := err.(*NotifyError); ok {
					return notifyErr.Code == ErrPlatformRateLimit
				}
				return false
			},
			Action: func(err error, attempt int) RetryAction {
				return RetryAction{
					ShouldRetry: attempt < 10, // More retries for rate limits
					Delay:       time.Duration(attempt+1) * 5 * time.Second,
					MaxAttempts: 10,
				}
			},
		}

		// Add conditional policy for specific platforms
		platformPolicy := ConditionalRetryPolicy{
			Name: "CriticalPlatformPolicy",
			Condition: func(err error, attempt int) bool {
				if notifyErr, ok := err.(*NotifyError); ok {
					return notifyErr.Context["platform"] == "critical" && attempt < 15
				}
				return false
			},
			Action: func(err error, attempt int) RetryAction {
				return RetryAction{
					ShouldRetry: true,
					Delay:       1 * time.Second,
					MaxAttempts: 15,
				}
			},
		}

		strategy.ConditionalPolicies = []ConditionalRetryPolicy{rateLimitPolicy, platformPolicy}

		// Test rate limit policy
		rateLimitErr := &NotifyError{
			Code:     ErrPlatformRateLimit,
			Message:  "rate limited",
			Platform: "feishu",
		}

		// Should allow extended retries for rate limits
		for i := 0; i < 9; i++ {
			if !strategy.ShouldRetry(rateLimitErr, i) {
				t.Errorf("Expected rate limit retry for attempt %d", i)
			}
		}

		if strategy.ShouldRetry(rateLimitErr, 10) {
			t.Error("Expected rate limit retry to be blocked at attempt 10")
		}

		// Test critical platform policy
		criticalErr := &NotifyError{
			Code:     ErrPlatformTimeout,
			Message:  "timeout",
			Platform: "regular",
			Context:  map[string]interface{}{"platform": "critical"},
		}

		// Should allow extended retries for critical platform
		for i := 0; i < 14; i++ {
			if !strategy.ShouldRetry(criticalErr, i) {
				t.Errorf("Expected critical platform retry for attempt %d", i)
			}
		}

		if strategy.ShouldRetry(criticalErr, 15) {
			t.Error("Expected critical platform retry to be blocked at attempt 15")
		}
	})

	t.Run("custom retryable error implementation", func(t *testing.T) {
		// Create a custom error that implements RetryableError interface
		customErr := &CustomRetryableError{
			BaseError:      errors.New("custom error"),
			IsRetryableVal: true,
			RetryDelays:    []time.Duration{1*time.Second, 2*time.Second, 3*time.Second},
			MaxAttempts:    3,
		}

		// Test custom retry logic
		for i := 0; i < 3; i++ {
			if !customErr.ShouldRetry(i) {
				t.Errorf("Expected custom error to be retryable for attempt %d", i)
			}

			expectedDelay := customErr.RetryDelays[i]
			actualDelay := customErr.GetRetryDelay(i)
			if actualDelay != expectedDelay {
				t.Errorf("Expected custom delay %v for attempt %d, got %v", expectedDelay, i, actualDelay)
			}
		}

		if customErr.ShouldRetry(3) {
			t.Error("Expected custom error to not be retryable beyond max attempts")
		}
	})
}

// TestPerformanceMonitoringAdvanced tests advanced performance monitoring
func TestPerformanceMonitoringAdvanced(t *testing.T) {
	t.Run("latency percentile calculation", func(t *testing.T) {
		monitor := NewPerformanceMonitor()

		// Record operations with known latencies for percentile testing
		latencies := []time.Duration{
			10 * time.Millisecond,
			20 * time.Millisecond,
			30 * time.Millisecond,
			40 * time.Millisecond,
			50 * time.Millisecond,
			60 * time.Millisecond,
			70 * time.Millisecond,
			80 * time.Millisecond,
			90 * time.Millisecond,
			100 * time.Millisecond,
		}

		for _, latency := range latencies {
			monitor.RecordOperation("test", latency, true)
		}

		metrics := monitor.PlatformMetrics["test"]
		if metrics == nil {
			t.Fatal("Expected test platform metrics")
		}

		// Verify percentile calculations
		if metrics.MinLatency != 10*time.Millisecond {
			t.Errorf("Expected min latency 10ms, got %v", metrics.MinLatency)
		}

		if metrics.MaxLatency != 100*time.Millisecond {
			t.Errorf("Expected max latency 100ms, got %v", metrics.MaxLatency)
		}

		if metrics.P50Latency != 50*time.Millisecond {
			t.Errorf("Expected P50 latency 50ms, got %v", metrics.P50Latency)
		}

		if metrics.P95Latency != 95*time.Millisecond {
			t.Errorf("Expected P95 latency 95ms, got %v", metrics.P95Latency)
		}

		if metrics.P99Latency != 99*time.Millisecond {
			t.Errorf("Expected P99 latency 99ms, got %v", metrics.P99Latency)
		}

		expectedAverage := 55 * time.Millisecond // (10+20+...+100)/10
		if metrics.AverageLatency != expectedAverage {
			t.Errorf("Expected average latency %v, got %v", expectedAverage, metrics.AverageLatency)
		}
	})

	t.Run("optimization rule effectiveness", func(t *testing.T) {
		monitor := NewPerformanceMonitor()

		// Simulate a problematic platform with high failure rate
		for i := 0; i < 20; i++ {
			success := i < 8 // 40% success rate
			monitor.RecordOperation("problematic", 100*time.Millisecond, success)
		}

		// Simulate a high-latency platform
		for i := 0; i < 10; i++ {
			monitor.RecordOperation("slow", 15*time.Second, true)
		}

		// Test optimization for problematic platform
		currentConfig := &PlatformRetryConfig{
			Platform:    "problematic",
			MaxAttempts: 3,
			BaseDelay:   1 * time.Second,
			MaxDelay:    10 * time.Second,
			Multiplier:  2.0,
			Enabled:     true,
		}

		optimizedConfig := monitor.OptimizeStrategy("problematic", currentConfig)

		// Should increase max attempts for low success rate
		if optimizedConfig.MaxAttempts <= currentConfig.MaxAttempts {
			t.Error("Expected optimization to increase max attempts for problematic platform")
		}

		// Should increase base delay
		if optimizedConfig.BaseDelay <= currentConfig.BaseDelay {
			t.Error("Expected optimization to increase base delay for problematic platform")
		}

		// Test optimization for slow platform
		slowConfig := &PlatformRetryConfig{
			Platform:    "slow",
			MaxAttempts: 5,
			BaseDelay:   1 * time.Second,
			MaxDelay:    10 * time.Second,
			Multiplier:  2.0,
			Enabled:     true,
		}

		optimizedSlowConfig := monitor.OptimizeStrategy("slow", slowConfig)

		// Should increase delays for high-latency platform
		if optimizedSlowConfig.BaseDelay <= slowConfig.BaseDelay {
			t.Error("Expected optimization to increase base delay for slow platform")
		}

		if optimizedSlowConfig.MaxDelay <= slowConfig.MaxDelay {
			t.Error("Expected optimization to increase max delay for slow platform")
		}
	})

	t.Run("recommendation generation", func(t *testing.T) {
		monitor := NewPerformanceMonitor()

		// Create scenarios that should trigger recommendations
		// Low success rate platform
		for i := 0; i < 15; i++ {
			success := i < 5 // 33% success rate
			monitor.RecordOperation("unreliable", 100*time.Millisecond, success)
		}

		// High latency platform
		for i := 0; i < 10; i++ {
			monitor.RecordOperation("slow", 35*time.Second, true)
		}

		// Good platform for comparison
		for i := 0; i < 10; i++ {
			monitor.RecordOperation("good", 100*time.Millisecond, true)
		}

		recommendations := monitor.GetRecommendations()

		if len(recommendations) == 0 {
			t.Error("Expected recommendations for problematic platforms")
		}

		// Should have recommendation for low success rate
		foundSuccessRateRec := false
		for _, rec := range recommendations {
			if containsString(rec, "unreliable") && containsString(rec, "success rate") {
				foundSuccessRateRec = true
				break
			}
		}

		if !foundSuccessRateRec {
			t.Error("Expected recommendation for unreliable platform success rate")
		}

		// Should have recommendation for high latency
		foundLatencyRec := false
		for _, rec := range recommendations {
			if containsString(rec, "slow") && containsString(rec, "latency") {
				foundLatencyRec = true
				break
			}
		}

		if !foundLatencyRec {
			t.Error("Expected recommendation for slow platform latency")
		}

		// Good platform should not have recommendations
		foundGoodRec := false
		for _, rec := range recommendations {
			if containsString(rec, "good") {
				foundGoodRec = true
				break
			}
		}

		if foundGoodRec {
			t.Error("Did not expect recommendation for good platform")
		}
	})
}

// TestRetryExecutorStressScenarios tests retry executor under stress
func TestRetryExecutorStressScenarios(t *testing.T) {
	t.Run("high concurrency retry execution", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()
		strategy.BaseDelay = 1 * time.Millisecond // Speed up for testing
		strategy.MaxAttempts = 3

		logger := &mockLogger{}
		executor := NewRetryExecutor(strategy, logger)

		var wg sync.WaitGroup
		numGoroutines := 100
		operationsPerGoroutine := 10

		successCount := int64(0)
		errorCount := int64(0)
		var mu sync.Mutex

		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					attempt := 0
					err := executor.Execute(context.Background(), func() error {
						attempt++
						// Simulate some operations succeeding after retries
						if attempt >= 2 && id%3 == 0 {
							return nil
						}
						return New(ErrPlatformTimeout, "timeout")
					})

					mu.Lock()
					if err == nil {
						successCount++
					} else {
						errorCount++
					}
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		totalOperations := int64(numGoroutines * operationsPerGoroutine)
		if successCount+errorCount != totalOperations {
			t.Errorf("Expected %d total operations, got %d", totalOperations, successCount+errorCount)
		}

		// Should have some successes and some failures
		if successCount == 0 {
			t.Error("Expected some successful operations")
		}

		if errorCount == 0 {
			t.Error("Expected some failed operations")
		}
	})

	t.Run("context cancellation handling", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()
		strategy.BaseDelay = 100 * time.Millisecond
		strategy.MaxAttempts = 10

		logger := &mockLogger{}
		executor := NewRetryExecutor(strategy, logger)

		// Test various cancellation scenarios
		scenarios := []struct {
			name           string
			cancelAfter    time.Duration
			operationDelay time.Duration
		}{
			{"cancel during operation", 50 * time.Millisecond, 200 * time.Millisecond},
			{"cancel during delay", 150 * time.Millisecond, 10 * time.Millisecond},
			{"cancel after multiple retries", 300 * time.Millisecond, 10 * time.Millisecond},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())

				// Cancel context after specified duration
				go func() {
					time.Sleep(scenario.cancelAfter)
					cancel()
				}()

				startTime := time.Now()
				err := executor.Execute(ctx, func() error {
					time.Sleep(scenario.operationDelay)
					return New(ErrPlatformTimeout, "timeout")
				})
				duration := time.Since(startTime)

				if err == nil {
					t.Error("Expected error due to context cancellation")
				}

				// Should complete within reasonable time after cancellation
				expectedMaxDuration := scenario.cancelAfter + 100*time.Millisecond
				if duration > expectedMaxDuration {
					t.Errorf("Operation took too long after cancellation: %v > %v", duration, expectedMaxDuration)
				}

				// Should be a system timeout error
				if notifyErr, ok := err.(*NotifyError); !ok || notifyErr.Code != ErrSystemTimeout {
					t.Errorf("Expected ErrSystemTimeout, got %v", err)
				}
			})
		}
	})
}

// Helper types and functions

// CustomRetryableError implements RetryableError interface for testing
type CustomRetryableError struct {
	BaseError      error
	IsRetryableVal bool
	RetryDelays    []time.Duration
	MaxAttempts    int
}

func (e *CustomRetryableError) Error() string {
	return e.BaseError.Error()
}

func (e *CustomRetryableError) IsRetryable() bool {
	return e.IsRetryableVal
}

func (e *CustomRetryableError) ShouldRetry(attempt int) bool {
	return e.IsRetryableVal && attempt < e.MaxAttempts
}

func (e *CustomRetryableError) GetRetryDelay(attempt int) time.Duration {
	if attempt >= 0 && attempt < len(e.RetryDelays) {
		return e.RetryDelays[attempt]
	}
	return 1 * time.Second
}

func (e *CustomRetryableError) GetRetryContext() map[string]interface{} {
	return map[string]interface{}{
		"custom_error": true,
		"max_attempts": e.MaxAttempts,
	}
}

// Helper functions
func calculateMean(delays []time.Duration) time.Duration {
	if len(delays) == 0 {
		return 0
	}

	total := int64(0)
	for _, delay := range delays {
		total += int64(delay)
	}

	return time.Duration(total / int64(len(delays)))
}

func calculateVariance(delays []time.Duration, mean time.Duration) float64 {
	if len(delays) == 0 {
		return 0
	}

	sumSquaredDiffs := 0.0
	meanFloat := float64(mean)

	for _, delay := range delays {
		diff := float64(delay) - meanFloat
		sumSquaredDiffs += diff * diff
	}

	return sumSquaredDiffs / float64(len(delays))
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkErrorHandlingPerformance benchmarks error handling performance
func BenchmarkErrorHandlingPerformance(b *testing.B) {
	b.Run("error creation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := New(ErrPlatformTimeout, "timeout error").
				WithContext("platform", "feishu").
				WithContext("attempt", i).
				WithDetails("operation timeout")
			_ = err
		}
	})

	b.Run("error aggregation", func(b *testing.B) {
		aggregator := NewErrorAggregator()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := New(ErrPlatformTimeout, fmt.Sprintf("error %d", i))
			aggregator.Add(err)
		}

		_ = aggregator.ToError()
	})

	b.Run("retry strategy decision", func(b *testing.B) {
		strategy := NewExponentialBackoffStrategy()
		err := New(ErrPlatformTimeout, "timeout")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = strategy.ShouldRetry(err, i%5)
		}
	})

	b.Run("circuit breaker operations", func(b *testing.B) {
		cb := NewCircuitBreaker(10, 1*time.Second)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i%10 == 0 {
				cb.RecordSuccess()
			} else {
				cb.RecordFailure()
			}
			_ = cb.ShouldAllow()
			_ = cb.GetState()
		}
	})

	b.Run("performance monitoring", func(b *testing.B) {
		monitor := NewPerformanceMonitor()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			platform := fmt.Sprintf("platform-%d", i%5)
			latency := time.Duration(i%1000) * time.Microsecond
			success := i%3 != 0

			monitor.RecordOperation(platform, latency, success)
		}
	})
}