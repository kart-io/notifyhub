// Package errors provides performance and stress tests for error handling
package errors

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkErrorCreation benchmarks various error creation patterns
func BenchmarkErrorCreation(b *testing.B) {
	b.Run("simple_error_creation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = New(ErrPlatformTimeout, "timeout error")
		}
	})

	b.Run("error_with_context", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := New(ErrPlatformTimeout, "timeout error")
			err.WithContext("platform", "feishu")
			err.WithContext("attempt", i)
		}
	})

	b.Run("error_with_details", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := New(ErrPlatformTimeout, "timeout error")
			err.WithDetails(fmt.Sprintf("operation failed after %d ms", i*100))
		}
	})

	b.Run("builder_pattern", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewErrorBuilder(ErrPlatformTimeout, "timeout error").
				WithContext("platform", "feishu").
				WithContext("attempt", i).
				WithDetails("operation timeout").
				Build()
		}
	})

	b.Run("factory_functions", func(b *testing.B) {
		factories := []func(int) *NotifyError{
			func(i int) *NotifyError { return NewPlatformError(ErrPlatformTimeout, "feishu", fmt.Sprintf("error %d", i)) },
			func(i int) *NotifyError { return NewNetworkError(ErrNetworkTimeout, "api.feishu.cn", fmt.Sprintf("error %d", i)) },
			func(i int) *NotifyError { return NewSystemError(ErrSystemOverload, "dispatcher", fmt.Sprintf("error %d", i)) },
		}

		for i := 0; i < b.N; i++ {
			factory := factories[i%len(factories)]
			_ = factory(i)
		}
	})
}

// BenchmarkErrorAggregation benchmarks error aggregation performance
func BenchmarkErrorAggregation(b *testing.B) {
	b.Run("sequential_aggregation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			aggregator := NewErrorAggregator()
			for j := 0; j < 100; j++ {
				err := New(ErrPlatformTimeout, fmt.Sprintf("error %d", j))
				aggregator.Add(err)
			}
			_ = aggregator.ToError()
		}
	})

	b.Run("concurrent_aggregation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			aggregator := NewErrorAggregator()
			var wg sync.WaitGroup

			numGoroutines := 10
			errorsPerGoroutine := 10

			wg.Add(numGoroutines)
			for g := 0; g < numGoroutines; g++ {
				go func(goroutineID int) {
					defer wg.Done()
					for j := 0; j < errorsPerGoroutine; j++ {
						err := New(ErrPlatformTimeout, fmt.Sprintf("error %d-%d", goroutineID, j))
						aggregator.Add(err)
					}
				}(g)
			}

			wg.Wait()
			_ = aggregator.ToError()
		}
	})

	b.Run("large_scale_aggregation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			aggregator := NewErrorAggregator()
			for j := 0; j < 1000; j++ {
				platform := fmt.Sprintf("platform-%d", j%10)
				err := NewPlatformError(ErrPlatformTimeout, platform, fmt.Sprintf("error %d", j))
				aggregator.Add(err)
			}
			_ = aggregator.ToError()
		}
	})
}

// BenchmarkRetryStrategies benchmarks different retry strategy implementations
func BenchmarkRetryStrategies(b *testing.B) {
	err := New(ErrPlatformTimeout, "timeout")

	b.Run("exponential_backoff", func(b *testing.B) {
		strategy := NewExponentialBackoffStrategy()
		for i := 0; i < b.N; i++ {
			_ = strategy.ShouldRetry(err, i%5)
			_ = strategy.GetDelay(i % 5)
		}
	})

	b.Run("linear_backoff", func(b *testing.B) {
		strategy := NewLinearBackoffStrategy()
		for i := 0; i < b.N; i++ {
			_ = strategy.ShouldRetry(err, i%3)
			_ = strategy.GetDelay(i % 3)
		}
	})

	b.Run("fixed_delay", func(b *testing.B) {
		strategy := NewFixedDelayStrategy(1*time.Second, 3)
		for i := 0; i < b.N; i++ {
			_ = strategy.ShouldRetry(err, i%3)
			_ = strategy.GetDelay(i % 3)
		}
	})

	b.Run("platform_specific_config", func(b *testing.B) {
		strategy := NewExponentialBackoffStrategy()
		config := &PlatformRetryConfig{
			Platform:    "feishu",
			MaxAttempts: 5,
			BaseDelay:   1 * time.Second,
			MaxDelay:    30 * time.Second,
			Multiplier:  2.0,
			Enabled:     true,
		}
		strategy.UpdatePlatformConfig("feishu", config)

		platformErr := &NotifyError{
			Code:     ErrPlatformTimeout,
			Message:  "timeout",
			Platform: "feishu",
		}

		for i := 0; i < b.N; i++ {
			_ = strategy.ShouldRetry(platformErr, i%5)
			_ = strategy.GetDelay(i % 5)
		}
	})
}

// BenchmarkCircuitBreaker benchmarks circuit breaker operations
func BenchmarkCircuitBreaker(b *testing.B) {
	b.Run("circuit_breaker_operations", func(b *testing.B) {
		cb := NewCircuitBreaker(10, 1*time.Second)

		for i := 0; i < b.N; i++ {
			if i%20 == 0 {
				cb.RecordSuccess()
			} else {
				cb.RecordFailure()
			}
			_ = cb.ShouldAllow()
			_ = cb.GetState()
		}
	})

	b.Run("concurrent_circuit_breaker", func(b *testing.B) {
		cb := NewCircuitBreaker(10, 1*time.Second)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = cb.ShouldAllow()
				cb.RecordFailure()
				_ = cb.GetState()
			}
		})
	})
}

// BenchmarkPerformanceMonitor benchmarks performance monitoring operations
func BenchmarkPerformanceMonitor(b *testing.B) {
	b.Run("operation_recording", func(b *testing.B) {
		monitor := NewPerformanceMonitor()

		for i := 0; i < b.N; i++ {
			platform := fmt.Sprintf("platform-%d", i%5)
			latency := time.Duration(i%1000) * time.Microsecond
			success := i%3 != 0
			monitor.RecordOperation(platform, latency, success)
		}
	})

	b.Run("optimization_calculation", func(b *testing.B) {
		monitor := NewPerformanceMonitor()

		// Pre-populate with data
		for i := 0; i < 1000; i++ {
			monitor.RecordOperation("test", time.Duration(i)*time.Microsecond, i%5 != 0)
		}

		config := &PlatformRetryConfig{
			Platform:    "test",
			MaxAttempts: 3,
			BaseDelay:   1 * time.Second,
			MaxDelay:    10 * time.Second,
			Multiplier:  2.0,
			Enabled:     true,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = monitor.OptimizeStrategy("test", config)
		}
	})

	b.Run("recommendations_generation", func(b *testing.B) {
		monitor := NewPerformanceMonitor()

		// Pre-populate with problematic data
		for i := 0; i < 100; i++ {
			monitor.RecordOperation("problematic", 20*time.Second, i%10 == 0) // 10% success rate, high latency
			monitor.RecordOperation("good", 100*time.Millisecond, true)       // 100% success rate, low latency
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = monitor.GetRecommendations()
		}
	})
}

// BenchmarkRetryExecution benchmarks retry executor performance
func BenchmarkRetryExecution(b *testing.B) {
	logger := &mockLogger{}

	b.Run("successful_operation", func(b *testing.B) {
		strategy := NewFixedDelayStrategy(1*time.Millisecond, 3)
		executor := NewRetryExecutor(strategy, logger)

		for i := 0; i < b.N; i++ {
			_ = executor.Execute(context.Background(), func() error {
				return nil // Always succeed
			})
		}
	})

	b.Run("operation_with_retries", func(b *testing.B) {
		strategy := NewFixedDelayStrategy(1*time.Millisecond, 3)
		executor := NewRetryExecutor(strategy, logger)

		for i := 0; i < b.N; i++ {
			attempt := 0
			_ = executor.Execute(context.Background(), func() error {
				attempt++
				if attempt < 2 {
					return New(ErrPlatformTimeout, "timeout")
				}
				return nil
			})
		}
	})

	b.Run("operation_with_callback", func(b *testing.B) {
		strategy := NewFixedDelayStrategy(1*time.Millisecond, 3)
		executor := NewRetryExecutor(strategy, logger)

		for i := 0; i < b.N; i++ {
			attempt := 0
			_ = executor.ExecuteWithCallback(context.Background(), func() error {
				attempt++
				if attempt < 2 {
					return New(ErrPlatformTimeout, "timeout")
				}
				return nil
			}, func(attempt int, err error, delay time.Duration) {
				// Callback overhead
			})
		}
	})
}

// BenchmarkErrorSerialization benchmarks error serialization performance
func BenchmarkErrorSerialization(b *testing.B) {
	serializer := NewErrorSerializer()

	err := NewErrorBuilder(ErrPlatformTimeout, "complex error").
		WithDetails("Detailed error information").
		WithContext("platform", "feishu").
		WithContext("user_id", "user-123").
		WithContext("request_id", "req-456").
		WithContext("metadata", map[string]interface{}{
			"retry_count": 3,
			"latency":     "1.5s",
			"error_code":  "TIMEOUT_001",
		}).
		Build()

	b.Run("serialize_for_logging", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = serializer.SerializeForLogging(err)
		}
	})

	b.Run("serialize_for_api", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = serializer.SerializeForAPI(err)
		}
	})

	b.Run("serialize_for_debug", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = serializer.SerializeForDebug(err)
		}
	})
}

// TestMemoryUsage tests memory usage patterns under different loads
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	t.Run("error_creation_memory_usage", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Create many errors
		errors := make([]*NotifyError, 10000)
		for i := 0; i < 10000; i++ {
			errors[i] = NewErrorBuilder(ErrPlatformTimeout, fmt.Sprintf("error %d", i)).
				WithContext("platform", "feishu").
				WithContext("index", i).
				WithDetails(fmt.Sprintf("Error details for error %d", i)).
				Build()
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		bytesPerError := float64(m2.Alloc-m1.Alloc) / float64(len(errors))
		t.Logf("Memory per error: %.2f bytes", bytesPerError)

		if bytesPerError > 2000 { // Reasonable threshold
			t.Errorf("Memory usage per error too high: %.2f bytes", bytesPerError)
		}

		// Keep reference to prevent optimization
		_ = errors[len(errors)-1]
	})

	t.Run("aggregation_memory_efficiency", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		aggregator := NewErrorAggregator()

		// Add many errors
		for i := 0; i < 50000; i++ {
			err := New(ErrPlatformTimeout, fmt.Sprintf("error %d", i))
			aggregator.Add(err)

			// Periodically aggregate to test memory management
			if i%10000 == 0 {
				_ = aggregator.ToError()
			}
		}

		finalError := aggregator.ToError()

		runtime.GC()
		runtime.ReadMemStats(&m2)

		totalMemory := float64(m2.Alloc - m1.Alloc)
		t.Logf("Total memory for 50k errors and aggregation: %.2f MB", totalMemory/(1024*1024))

		// Verify aggregation worked
		if finalError == nil {
			t.Error("Expected aggregated error")
		}

		// Memory usage should be reasonable (less than 100MB for 50k errors)
		if totalMemory > 100*1024*1024 {
			t.Errorf("Memory usage too high: %.2f MB", totalMemory/(1024*1024))
		}
	})

	t.Run("retry_strategy_memory_stability", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		strategy := NewExponentialBackoffStrategy()

		// Simulate many retry operations
		for i := 0; i < 100000; i++ {
			platform := fmt.Sprintf("platform-%d", i%10)
			err := &NotifyError{
				Code:     ErrPlatformTimeout,
				Message:  "timeout",
				Platform: platform,
			}

			_ = strategy.ShouldRetry(err, i%5)
			delay := strategy.GetDelay(i % 5)
			strategy.RecordAttempt(platform, delay, err, i%3 == 0)

			// Periodically reset statistics to prevent unbounded growth
			if i%20000 == 0 {
				strategy.Reset()
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		memoryGrowth := float64(m2.Alloc - m1.Alloc)
		t.Logf("Memory growth for 100k retry operations: %.2f MB", memoryGrowth/(1024*1024))

		// Memory growth should be bounded (less than 50MB)
		if memoryGrowth > 50*1024*1024 {
			t.Errorf("Memory growth too high: %.2f MB", memoryGrowth/(1024*1024))
		}
	})
}

// TestConcurrencyStress tests error handling under high concurrency
func TestConcurrencyStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency stress test in short mode")
	}

	t.Run("concurrent_error_aggregation", func(t *testing.T) {
		aggregator := NewErrorAggregator()
		var wg sync.WaitGroup

		numGoroutines := 100
		errorsPerGoroutine := 1000

		wg.Add(numGoroutines)
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < errorsPerGoroutine; j++ {
					err := New(ErrPlatformTimeout, fmt.Sprintf("error %d-%d", goroutineID, j))
					aggregator.Add(err)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		totalErrors := numGoroutines * errorsPerGoroutine
		errorsPerSecond := float64(totalErrors) / duration.Seconds()

		t.Logf("Concurrent aggregation: %d errors in %v (%.0f errors/sec)",
			totalErrors, duration, errorsPerSecond)

		if aggregator.Count() != totalErrors {
			t.Errorf("Expected %d errors, got %d", totalErrors, aggregator.Count())
		}

		// Test final aggregation performance
		aggregationStart := time.Now()
		finalError := aggregator.ToError()
		aggregationDuration := time.Since(aggregationStart)

		if finalError == nil {
			t.Error("Expected aggregated error")
		}

		t.Logf("Final aggregation took: %v", aggregationDuration)

		if aggregationDuration > 200*time.Millisecond {
			t.Errorf("Aggregation took too long: %v", aggregationDuration)
		}
	})

	t.Run("concurrent_retry_strategy_access", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()
		var wg sync.WaitGroup

		numGoroutines := 50
		operationsPerGoroutine := 2000

		wg.Add(numGoroutines)
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					platform := fmt.Sprintf("platform-%d", j%5)
					err := &NotifyError{
						Code:     ErrPlatformTimeout,
						Message:  "timeout",
						Platform: platform,
					}

					// Test concurrent access to strategy methods
					_ = strategy.ShouldRetry(err, j%3)
					delay := strategy.GetDelay(j % 3)
					strategy.RecordAttempt(platform, delay, err, j%4 != 0)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		totalOperations := numGoroutines * operationsPerGoroutine
		operationsPerSecond := float64(totalOperations) / duration.Seconds()

		t.Logf("Concurrent retry operations: %d operations in %v (%.0f ops/sec)",
			totalOperations, duration, operationsPerSecond)

		// Verify strategy state is consistent
		stats := strategy.GetStatistics()
		if stats.TotalAttempts == 0 {
			t.Error("Expected recorded attempts")
		}

		t.Logf("Final retry stats: %d attempts, %.2f%% success rate",
			stats.TotalAttempts, stats.GetSuccessRate()*100)
	})

	t.Run("concurrent_circuit_breaker_stress", func(t *testing.T) {
		cb := NewCircuitBreaker(50, 1*time.Second)
		var wg sync.WaitGroup

		numGoroutines := 20
		operationsPerGoroutine := 5000

		wg.Add(numGoroutines)
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					// Mix of operations
					allowed := cb.ShouldAllow()
					if allowed {
						if j%10 == 0 {
							cb.RecordSuccess()
						} else {
							cb.RecordFailure()
						}
					}
					_ = cb.GetState()
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		totalOperations := numGoroutines * operationsPerGoroutine
		operationsPerSecond := float64(totalOperations) / duration.Seconds()

		t.Logf("Concurrent circuit breaker operations: %d operations in %v (%.0f ops/sec)",
			totalOperations, duration, operationsPerSecond)

		// Circuit breaker should still be in a valid state
		state := cb.GetState()
		if state != CircuitClosed && state != CircuitOpen && state != CircuitHalfOpen {
			t.Errorf("Invalid circuit breaker state after stress test: %v", state)
		}
	})
}

// TestPerformanceRegression tests for performance regressions
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	// These tests establish performance baselines
	// Adjust thresholds based on your performance requirements

	t.Run("error_creation_performance", func(t *testing.T) {
		numErrors := 100000
		start := time.Now()

		for i := 0; i < numErrors; i++ {
			_ = NewErrorBuilder(ErrPlatformTimeout, "timeout error").
				WithContext("platform", "feishu").
				WithContext("attempt", i).
				WithDetails("timeout occurred").
				Build()
		}

		duration := time.Since(start)
		errorsPerSecond := float64(numErrors) / duration.Seconds()

		t.Logf("Error creation: %d errors in %v (%.0f errors/sec)", numErrors, duration, errorsPerSecond)

		// Should be able to create at least 50k errors per second
		if errorsPerSecond < 50000 {
			t.Errorf("Error creation too slow: %.0f errors/sec", errorsPerSecond)
		}
	})

	t.Run("retry_decision_performance", func(t *testing.T) {
		strategy := NewExponentialBackoffStrategy()
		err := New(ErrPlatformTimeout, "timeout")

		numDecisions := 1000000
		start := time.Now()

		for i := 0; i < numDecisions; i++ {
			_ = strategy.ShouldRetry(err, i%5)
		}

		duration := time.Since(start)
		decisionsPerSecond := float64(numDecisions) / duration.Seconds()

		t.Logf("Retry decisions: %d decisions in %v (%.0f decisions/sec)",
			numDecisions, duration, decisionsPerSecond)

		// Should be able to make at least 500k decisions per second
		if decisionsPerSecond < 500000 {
			t.Errorf("Retry decision too slow: %.0f decisions/sec", decisionsPerSecond)
		}
	})

	t.Run("aggregation_performance", func(t *testing.T) {
		numErrors := 10000
		start := time.Now()

		aggregator := NewErrorAggregator()
		for i := 0; i < numErrors; i++ {
			err := New(ErrPlatformTimeout, fmt.Sprintf("error %d", i))
			aggregator.Add(err)
		}

		_ = aggregator.ToError()
		duration := time.Since(start)

		errorsPerSecond := float64(numErrors) / duration.Seconds()

		t.Logf("Error aggregation: %d errors in %v (%.0f errors/sec)",
			numErrors, duration, errorsPerSecond)

		// Should be able to aggregate at least 100k errors per second
		if errorsPerSecond < 100000 {
			t.Errorf("Error aggregation too slow: %.0f errors/sec", errorsPerSecond)
		}
	})
}