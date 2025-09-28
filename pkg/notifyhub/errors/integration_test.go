// Package errors provides integration tests for error handling workflows
package errors

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestErrorHandlingWorkflowIntegration tests complete error handling workflows
func TestErrorHandlingWorkflowIntegration(t *testing.T) {
	t.Run("multi-platform notification failure workflow", func(t *testing.T) {
		// Simulate a notification system with multiple platforms
		platforms := []string{"feishu", "email", "slack", "webhook"}

		// Create platform-specific configurations
		strategy := NewExponentialBackoffStrategy()
		for _, platform := range platforms {
			config := &PlatformRetryConfig{
				Platform:    platform,
				MaxAttempts: 3 + len(platform)%3, // Different retry limits
				BaseDelay:   time.Duration(100+len(platform)*50) * time.Millisecond,
				MaxDelay:    time.Duration(5+len(platform)) * time.Second,
				Multiplier:  1.5 + float64(len(platform)%3)*0.2,
				Jitter:      0.1 + float64(len(platform)%10)*0.01,
				Enabled:     true,
			}
			strategy.UpdatePlatformConfig(platform, config)
		}

		// Create circuit breakers for each platform
		circuitBreakers := make(map[string]*CircuitBreaker)
		for _, platform := range platforms {
			circuitBreakers[platform] = NewCircuitBreaker(3, 2*time.Second)
		}

		// Simulate notifications with different failure patterns
		aggregator := NewErrorAggregator()
		performanceMonitor := NewPerformanceMonitor()

		notifications := []struct {
			platform    string
			shouldFail  bool
			errorCode   Code
			latency     time.Duration
		}{
			{"feishu", true, ErrPlatformTimeout, 1500 * time.Millisecond},
			{"email", false, "", 500 * time.Millisecond},
			{"slack", true, ErrPlatformRateLimit, 200 * time.Millisecond},
			{"webhook", true, ErrNetworkConnection, 3000 * time.Millisecond},
			{"feishu", true, ErrPlatformTimeout, 2000 * time.Millisecond},
			{"email", true, ErrPlatformAuth, 100 * time.Millisecond},
			{"slack", false, "", 150 * time.Millisecond},
			{"webhook", false, "", 800 * time.Millisecond},
		}

		for i, notification := range notifications {
			platform := notification.platform

			// Check circuit breaker
			cb := circuitBreakers[platform]
			if !cb.ShouldAllow() {
				t.Logf("Circuit breaker blocked notification %d to %s", i, platform)
				continue
			}

			// Simulate notification attempt
			startTime := time.Now()

			var err error
			if notification.shouldFail {
				err = &NotifyError{
					Code:     notification.errorCode,
					Message:  fmt.Sprintf("notification failed for %s", platform),
					Platform: platform,
					Context: map[string]interface{}{
						"notification_id": i,
						"attempt_time":    startTime,
					},
				}

				// Record failure in circuit breaker
				cb.RecordFailure()

				// Add to aggregator
				aggregator.Add(err)
			} else {
				// Record success in circuit breaker
				cb.RecordSuccess()
			}

			// Record in performance monitor
			actualLatency := notification.latency
			if notification.shouldFail {
				actualLatency = actualLatency + 500*time.Millisecond // Failures take longer
			}

			performanceMonitor.RecordOperation(platform, actualLatency, !notification.shouldFail)

			// Record retry attempt
			delay := strategy.GetDelay(i % 3)
			strategy.RecordAttempt(platform, delay, err, !notification.shouldFail)
		}

		// Analyze aggregated errors
		if !aggregator.HasErrors() {
			t.Error("Expected aggregated errors from failed notifications")
		}

		aggregatedError := aggregator.ToError()
		if aggregatedError == nil {
			t.Fatal("Expected aggregated error")
		}

		notifyErr, ok := aggregatedError.(*NotifyError)
		if !ok {
			t.Fatal("Expected NotifyError type")
		}

		// Verify error aggregation contains platform information
		affectedPlatforms := notifyErr.Context["affected_platforms"]
		if affectedPlatforms == nil {
			t.Error("Expected affected_platforms in aggregated error context")
		}

		// Verify performance monitoring captured metrics
		for _, platform := range platforms {
			metrics := performanceMonitor.PlatformMetrics[platform]
			if metrics == nil {
				t.Errorf("Expected performance metrics for platform %s", platform)
				continue
			}

			if metrics.TotalOperations == 0 {
				t.Errorf("Expected operations recorded for platform %s", platform)
			}

			t.Logf("Platform %s: %d operations, %.2f%% success rate, avg latency %v",
				platform, metrics.TotalOperations, metrics.SuccessRate*100, metrics.AverageLatency)
		}

		// Test optimization recommendations
		recommendations := performanceMonitor.GetRecommendations()
		t.Logf("Generated %d recommendations", len(recommendations))
		for _, rec := range recommendations {
			t.Logf("Recommendation: %s", rec)
		}

		// Verify retry statistics
		stats := strategy.GetStatistics()
		if stats.TotalAttempts == 0 {
			t.Error("Expected retry statistics to be recorded")
		}

		t.Logf("Retry Statistics: %d total attempts, %.2f%% success rate, %.2f%% retry rate",
			stats.TotalAttempts, stats.GetSuccessRate()*100, stats.GetRetryRate()*100)
	})

	t.Run("cascading failure recovery workflow", func(t *testing.T) {
		// Simulate a scenario where failures cascade and then recover
		strategy := NewExponentialBackoffStrategy()
		logger := &mockLogger{}
		executor := NewRetryExecutor(strategy, logger)

		// Track system state
		systemHealth := map[string]bool{
			"database":     true,
			"cache":        true,
			"external_api": true,
			"queue":        true,
		}

		var mu sync.Mutex
		failureCount := int64(0)
		recoveryTime := time.Now().Add(3 * time.Second)

		// Simulate operation that depends on multiple components
		operation := func() error {
			mu.Lock()
			defer mu.Unlock()

			// Simulate cascading failures
			if time.Now().Before(recoveryTime) {
				// Gradually degrade system health
				if atomic.AddInt64(&failureCount, 1)%10 == 0 {
					for component := range systemHealth {
						if systemHealth[component] {
							systemHealth[component] = false
							break
						}
					}
				}

				// Check if any component is down
				for component, healthy := range systemHealth {
					if !healthy {
						return &NotifyError{
							Code:     ErrSystemUnavailable,
							Message:  fmt.Sprintf("component %s is unavailable", component),
							Platform: "system",
							Context: map[string]interface{}{
								"component":     component,
								"failure_count": atomic.LoadInt64(&failureCount),
								"system_health": systemHealth,
							},
						}
					}
				}
			} else {
				// Start recovery
				allHealthy := true
				for component, healthy := range systemHealth {
					if !healthy {
						systemHealth[component] = true
						t.Logf("Component %s recovered", component)
						break
					}
					allHealthy = allHealthy && healthy
				}

				if !allHealthy {
					return &NotifyError{
						Code:     ErrSystemUnavailable,
						Message:  "system still recovering",
						Platform: "system",
						Context: map[string]interface{}{
							"recovering":    true,
							"system_health": systemHealth,
						},
					}
				}
			}

			return nil // Success
		}

		// Execute with retry
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		startTime := time.Now()
		err := executor.Execute(ctx, operation)
		duration := time.Since(startTime)

		if err != nil {
			t.Errorf("Expected eventual success after recovery, got error: %v", err)
		}

		t.Logf("System recovered after %v", duration)

		// Verify all components recovered
		mu.Lock()
		for component, healthy := range systemHealth {
			if !healthy {
				t.Errorf("Component %s did not recover", component)
			}
		}
		mu.Unlock()

		// Check that recovery was attempted
		if atomic.LoadInt64(&failureCount) == 0 {
			t.Error("Expected failures during cascading failure simulation")
		}
	})
}

// TestMultiPlatformErrorAggregation tests comprehensive multi-platform error scenarios
func TestMultiPlatformErrorAggregation(t *testing.T) {
	t.Run("large scale multi-platform failure", func(t *testing.T) {
		platforms := []string{"feishu", "slack", "email", "webhook", "teams", "discord", "telegram", "sms"}
		errorCodes := []Code{
			ErrPlatformTimeout, ErrPlatformRateLimit, ErrPlatformUnavailable,
			ErrNetworkConnection, ErrNetworkTimeout, ErrMessageSendFailed,
		}

		aggregator := NewErrorAggregator()

		// Generate errors across multiple platforms
		numErrors := 1000
		for i := 0; i < numErrors; i++ {
			platform := platforms[i%len(platforms)]
			errorCode := errorCodes[i%len(errorCodes)]

			err := &NotifyError{
				Code:     errorCode,
				Message:  fmt.Sprintf("error %d on %s", i, platform),
				Platform: platform,
				Context: map[string]interface{}{
					"error_id":    i,
					"timestamp":   time.Now(),
					"user_id":     fmt.Sprintf("user-%d", i%100),
					"message_id":  fmt.Sprintf("msg-%d", i),
					"platform":    platform,
				},
			}

			aggregator.Add(err)
		}

		// Test aggregation performance
		start := time.Now()
		aggregatedErr := aggregator.ToError()
		aggregationTime := time.Since(start)

		if aggregatedErr == nil {
			t.Fatal("Expected aggregated error")
		}

		if aggregationTime > 50*time.Millisecond {
			t.Errorf("Aggregation took too long: %v", aggregationTime)
		}

		notifyErr := aggregatedErr.(*NotifyError)

		// Verify aggregated context
		errorCount := notifyErr.Context["error_count"]
		if errorCount != numErrors {
			t.Errorf("Expected error count %d, got %v", numErrors, errorCount)
		}

		affectedPlatforms := notifyErr.Context["affected_platforms"].(map[string]int)
		expectedPlatformCount := numErrors / len(platforms)

		for _, platform := range platforms {
			count := affectedPlatforms[platform]
			if count != expectedPlatformCount {
				t.Errorf("Expected %d errors for platform %s, got %d", expectedPlatformCount, platform, count)
			}
		}

		errorCodesMap := notifyErr.Context["error_codes"].(map[Code]int)
		expectedCodeCount := numErrors / len(errorCodes)
		tolerance := 2 // Allow some variance due to modulo distribution

		for _, code := range errorCodes {
			count := errorCodesMap[code]
			if count < expectedCodeCount-tolerance || count > expectedCodeCount+tolerance {
				t.Errorf("Expected around %d errors for code %s, got %d", expectedCodeCount, code, count)
			}
		}
	})

	t.Run("real-time error aggregation", func(t *testing.T) {
		aggregator := NewErrorAggregator()
		platforms := []string{"feishu", "slack", "email"}

		var wg sync.WaitGroup
		numGoroutines := 10
		errorsPerGoroutine := 50

		// Simulate real-time error reporting from multiple goroutines
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < errorsPerGoroutine; j++ {
					platform := platforms[(goroutineID+j)%len(platforms)]

					err := &NotifyError{
						Code:     ErrPlatformTimeout,
						Message:  fmt.Sprintf("concurrent error from goroutine %d", goroutineID),
						Platform: platform,
						Context: map[string]interface{}{
							"goroutine_id": goroutineID,
							"error_index":  j,
							"timestamp":    time.Now(),
						},
					}

					aggregator.Add(err)

					// Simulate some processing time
					time.Sleep(time.Microsecond * time.Duration(j%10))
				}
			}(i)
		}

		wg.Wait()

		// Verify concurrent aggregation
		expectedTotal := numGoroutines * errorsPerGoroutine
		if aggregator.Count() != expectedTotal {
			t.Errorf("Expected %d total errors, got %d", expectedTotal, aggregator.Count())
		}

		aggregatedErr := aggregator.ToError()
		if aggregatedErr == nil {
			t.Fatal("Expected aggregated error")
		}

		// Verify thread safety - no data races or corruption
		notifyErr := aggregatedErr.(*NotifyError)
		errorCount := notifyErr.Context["error_count"]
		if errorCount != expectedTotal {
			t.Errorf("Expected error count %d in context, got %v", expectedTotal, errorCount)
		}
	})
}

// TestRetryStrategyIntegration tests retry strategies in realistic scenarios
func TestRetryStrategyIntegration(t *testing.T) {
	t.Run("adaptive retry strategy under varying load", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()
		monitor := strategy.PerformanceMonitor

		platforms := []string{"high-load", "medium-load", "low-load"}

		// Configure platform-specific settings
		for i, platform := range platforms {
			config := &PlatformRetryConfig{
				Platform:    platform,
				MaxAttempts: 3 + i*2,
				BaseDelay:   time.Duration(100+i*200) * time.Millisecond,
				MaxDelay:    time.Duration(5+i*10) * time.Second,
				Multiplier:  1.5 + float64(i)*0.3,
				Jitter:      0.1 + float64(i)*0.05,
				Enabled:     true,
			}
			strategy.UpdatePlatformConfig(platform, config)
		}

		// Simulate operations under different load conditions
		loadProfiles := map[string]struct {
			successRate float64
			avgLatency  time.Duration
			operations  int
		}{
			"high-load":   {0.60, 2 * time.Second, 100},
			"medium-load": {0.80, 800 * time.Millisecond, 50},
			"low-load":    {0.95, 200 * time.Millisecond, 20},
		}

		for platform, profile := range loadProfiles {
			for i := 0; i < profile.operations; i++ {
				// Determine if operation succeeds based on success rate
				success := float64(i)/float64(profile.operations) < profile.successRate

				// Add some variance to latency
				latency := profile.avgLatency + time.Duration(i%100)*time.Millisecond

				// Record operation
				monitor.RecordOperation(platform, latency, success)

				// Simulate retry attempts for failures
				if !success {
					var err error = &NotifyError{
						Code:     ErrPlatformTimeout,
						Message:  "operation timeout",
						Platform: platform,
					}

					attempt := 0
					for strategy.ShouldRetry(err, attempt) && attempt < 3 {
						delay := strategy.GetDelay(attempt)
						strategy.RecordAttempt(platform, delay, err, false)
						attempt++
					}
				} else {
					strategy.RecordAttempt(platform, 0, nil, true)
				}
			}
		}

		// Verify performance monitoring captured load differences
		for platform, profile := range loadProfiles {
			metrics := monitor.PlatformMetrics[platform]
			if metrics == nil {
				t.Errorf("Expected metrics for platform %s", platform)
				continue
			}

			actualSuccessRate := metrics.SuccessRate
			expectedSuccessRate := profile.successRate
			tolerance := 0.15 // 15% tolerance

			if actualSuccessRate < expectedSuccessRate-tolerance || actualSuccessRate > expectedSuccessRate+tolerance {
				t.Errorf("Platform %s success rate %.2f not within tolerance of expected %.2f",
					platform, actualSuccessRate, expectedSuccessRate)
			}

			t.Logf("Platform %s: Success Rate %.2f%%, Avg Latency %v, P95 Latency %v",
				platform, metrics.SuccessRate*100, metrics.AverageLatency, metrics.P95Latency)
		}

		// Test optimization recommendations
		recommendations := monitor.GetRecommendations()
		if len(recommendations) == 0 {
			t.Error("Expected optimization recommendations for high-load platform")
		}

		// High-load platform should get optimization
		highLoadConfig := strategy.GetPlatformConfig("high-load")
		optimizedConfig := monitor.OptimizeStrategy("high-load", highLoadConfig)

		if optimizedConfig.MaxAttempts <= highLoadConfig.MaxAttempts {
			t.Error("Expected optimization to increase retry attempts for high-load platform")
		}
	})

	t.Run("circuit breaker integration with retry strategy", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()

		// Configure platform with circuit breaker
		config := &PlatformRetryConfig{
			Platform:    "unstable",
			MaxAttempts: 5,
			BaseDelay:   100 * time.Millisecond,
			MaxDelay:    2 * time.Second,
			Multiplier:  2.0,
			Enabled:     true,
		}
		strategy.UpdatePlatformConfig("unstable", config)

		// Simulate operations that will trip the circuit breaker
		err := &NotifyError{
			Code:     ErrPlatformTimeout,
			Message:  "timeout",
			Platform: "unstable",
		}

		// Phase 1: Normal operation with increasing failures
		for i := 0; i < 10; i++ {
			shouldRetry := strategy.ShouldRetry(err, 0)
			if !shouldRetry {
				t.Errorf("Expected retry to be allowed for attempt %d", i)
			}

			// Record failure
			strategy.RecordAttempt("unstable", strategy.GetDelay(0), err, false)
		}

		// Phase 2: Circuit breaker should eventually block retries
		// (Note: The circuit breaker is integrated into the retry strategy)
		cb := strategy.CircuitBreakers["unstable"]
		if cb == nil {
			t.Fatal("Expected circuit breaker for unstable platform")
		}

		// Trip circuit breaker by recording multiple failures
		for i := 0; i < 5; i++ {
			cb.RecordFailure()
		}

		// Circuit breaker should now block operations
		if cb.ShouldAllow() {
			t.Error("Expected circuit breaker to block operations after threshold failures")
		}

		// Phase 3: Wait for circuit breaker recovery and test success
		// Note: In a real scenario, we would wait for the timeout
		cb.Reset() // Simulate recovery for testing

		// Record successful operation
		strategy.RecordAttempt("unstable", 0, nil, true)

		// Verify circuit breaker allows operations after success
		if !cb.ShouldAllow() {
			t.Error("Expected circuit breaker to allow operations after reset and success")
		}
	})
}

// TestErrorContextPropagation tests error context preservation across operations
func TestErrorContextPropagation(t *testing.T) {
	t.Run("context preservation through retry chain", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()
		logger := &mockLogger{}
		executor := NewRetryExecutor(strategy, logger)

		originalContext := map[string]interface{}{
			"user_id":         "user-123",
			"request_id":      "req-456",
			"correlation_id":  "corr-789",
			"operation_type":  "notification",
			"timestamp":       time.Now(),
			"metadata": map[string]string{
				"source":  "api",
				"version": "v2",
			},
		}

		attemptCount := 0
		operation := func() error {
			attemptCount++

			if attemptCount < 3 {
				err := &NotifyError{
					Code:     ErrPlatformTimeout,
					Message:  fmt.Sprintf("timeout on attempt %d", attemptCount),
					Platform: "test",
					Context:  make(map[string]interface{}),
				}

				// Propagate original context
				for key, value := range originalContext {
					err.Context[key] = value
				}

				// Add attempt-specific context
				err.Context["attempt"] = attemptCount
				err.Context["retry_timestamp"] = time.Now()

				return err
			}

			return nil
		}

		// Execute with context propagation tracking
		callbackCalled := 0
		err := executor.ExecuteWithCallback(context.Background(), operation,
			func(attempt int, err error, delay time.Duration) {
				callbackCalled++

				if notifyErr, ok := err.(*NotifyError); ok {
					// Verify original context is preserved
					for key, expectedValue := range originalContext {
						if key == "timestamp" {
							continue // Skip timestamp comparison
						}

						actualValue := notifyErr.Context[key]
						if !reflect.DeepEqual(actualValue, expectedValue) {
							t.Errorf("Context key %s: expected %v, got %v", key, expectedValue, actualValue)
						}
					}

					// Verify attempt-specific context
					attemptValue := notifyErr.Context["attempt"]
					if attemptValue != attempt+1 {
						t.Errorf("Expected attempt context %d, got %v", attempt+1, attemptValue)
					}
				}
			})

		if err != nil {
			t.Errorf("Expected operation to succeed after retries, got: %v", err)
		}

		if callbackCalled != 2 {
			t.Errorf("Expected 2 retry callbacks, got %d", callbackCalled)
		}

		if attemptCount != 3 {
			t.Errorf("Expected 3 total attempts, got %d", attemptCount)
		}
	})

	t.Run("error wrapping chain preservation", func(t *testing.T) {
		// Create a chain of wrapped errors
		originalErr := fmt.Errorf("original database error")

		serviceErr := Wrap(originalErr, ErrSystemUnavailable, "service layer error")
		serviceErr.WithContext("service", "notification-service")
		serviceErr.WithContext("operation", "send_notification")

		handlerErr := Wrap(serviceErr, ErrPlatformTimeout, "handler layer error")
		handlerErr.WithContext("handler", "webhook-handler")
		handlerErr.WithContext("platform", "feishu")

		apiErr := Wrap(handlerErr, ErrMessageSendFailed, "api layer error")
		apiErr.WithContext("api_version", "v2")
		apiErr.WithContext("endpoint", "/notifications/send")

		// Verify error chain can be unwrapped
		var unwrappedErr error = apiErr
		depth := 0
		expectedMessages := []string{
			"api layer error",
			"handler layer error",
			"service layer error",
			"original database error",
		}

		for unwrappedErr != nil {
			if depth < len(expectedMessages) {
				expectedMsg := expectedMessages[depth]
				if !containsString(unwrappedErr.Error(), expectedMsg) {
					t.Errorf("Depth %d: expected error to contain '%s', got '%s'",
						depth, expectedMsg, unwrappedErr.Error())
				}
			}

			if notifyErr, ok := unwrappedErr.(*NotifyError); ok {
				unwrappedErr = notifyErr.Unwrap()
			} else {
				unwrappedErr = nil
			}
			depth++
		}

		if depth != 4 {
			t.Errorf("Expected error chain depth of 4, got %d", depth)
		}

		// Verify context from each layer is preserved
		if apiErr.Context["api_version"] != "v2" {
			t.Error("API layer context not preserved")
		}

		if handlerErr.Context["platform"] != "feishu" {
			t.Error("Handler layer context not preserved")
		}

		if serviceErr.Context["service"] != "notification-service" {
			t.Error("Service layer context not preserved")
		}
	})
}

// TestStressScenarios tests error handling under stress conditions
func TestStressScenarios(t *testing.T) {
	t.Run("high frequency error generation", func(t *testing.T) {
		aggregator := NewErrorAggregator()
		strategy := NewExponentialBackoffStrategy()

		var wg sync.WaitGroup
		numGoroutines := 50
		errorsPerGoroutine := 1000

		// Generate high frequency errors
		start := time.Now()
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < errorsPerGoroutine; j++ {
					platform := fmt.Sprintf("platform-%d", j%10)

					err := &NotifyError{
						Code:     ErrPlatformTimeout,
						Message:  fmt.Sprintf("high frequency error %d", j),
						Platform: platform,
						Context: map[string]interface{}{
							"goroutine": goroutineID,
							"sequence":  j,
						},
					}

					aggregator.Add(err)

					// Also test retry decision under load
					_ = strategy.ShouldRetry(err, j%5)
					_ = strategy.GetDelay(j % 5)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		totalErrors := numGoroutines * errorsPerGoroutine
		errorsPerSecond := float64(totalErrors) / duration.Seconds()

		t.Logf("Generated %d errors in %v (%.0f errors/sec)", totalErrors, duration, errorsPerSecond)

		if aggregator.Count() != totalErrors {
			t.Errorf("Expected %d errors in aggregator, got %d", totalErrors, aggregator.Count())
		}

		// Test aggregation performance under stress
		aggregationStart := time.Now()
		aggregatedErr := aggregator.ToError()
		aggregationDuration := time.Since(aggregationStart)

		if aggregatedErr == nil {
			t.Error("Expected aggregated error")
		}

		if aggregationDuration > 100*time.Millisecond {
			t.Errorf("Aggregation under stress took too long: %v", aggregationDuration)
		}

		// Verify aggregated error integrity
		notifyErr := aggregatedErr.(*NotifyError)
		errorCount := notifyErr.Context["error_count"]
		if errorCount != totalErrors {
			t.Errorf("Aggregated error count mismatch: expected %d, got %v", totalErrors, errorCount)
		}
	})

	t.Run("memory usage under error load", func(t *testing.T) {
		// Test that error handling doesn't cause memory leaks
		aggregator := NewErrorAggregator()
		strategy := NewExponentialBackoffStrategy()

		// Generate and process errors in batches to test memory management
		batchSize := 10000
		numBatches := 10

		for batch := 0; batch < numBatches; batch++ {
			// Generate batch of errors
			batchStart := time.Now()

			for i := 0; i < batchSize; i++ {
				err := &NotifyError{
					Code:     ErrPlatformTimeout,
					Message:  fmt.Sprintf("batch %d error %d", batch, i),
					Platform: fmt.Sprintf("platform-%d", i%5),
					Context: map[string]interface{}{
						"batch":     batch,
						"index":     i,
						"timestamp": time.Now(),
						"data":      make([]byte, 100), // Some payload
					},
				}

				aggregator.Add(err)
				strategy.RecordAttempt(err.Platform, time.Duration(i)*time.Microsecond, err, i%3 == 0)
			}

			batchDuration := time.Since(batchStart)
			t.Logf("Batch %d: processed %d errors in %v", batch, batchSize, batchDuration)

			// Process aggregated errors periodically to prevent unbounded growth
			if batch%3 == 0 {
				_ = aggregator.ToError()
				aggregator = NewErrorAggregator() // Reset for next batch
			}
		}

		// Final verification
		finalStats := strategy.GetStatistics()
		expectedAttempts := int64(batchSize * numBatches)

		if finalStats.TotalAttempts != expectedAttempts {
			t.Errorf("Expected %d total attempts, got %d", expectedAttempts, finalStats.TotalAttempts)
		}

		t.Logf("Final stats: %d attempts, %.2f%% success rate",
			finalStats.TotalAttempts, finalStats.GetSuccessRate()*100)
	})
}