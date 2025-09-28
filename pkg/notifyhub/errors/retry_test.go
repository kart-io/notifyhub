package errors

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// mockLogger is a simple mock logger for testing
type mockLogger struct {
	logs []string
}

func (m *mockLogger) LogMode(level logger.LogLevel) logger.Logger {
	return m
}

func (m *mockLogger) Debug(msg string, args ...any) {
	m.logs = append(m.logs, "DEBUG: "+msg)
}

func (m *mockLogger) Info(msg string, args ...any) {
	m.logs = append(m.logs, "INFO: "+msg)
}

func (m *mockLogger) Warn(msg string, args ...any) {
	m.logs = append(m.logs, "WARN: "+msg)
}

func (m *mockLogger) Error(msg string, args ...any) {
	m.logs = append(m.logs, "ERROR: "+msg)
}

func TestExponentialBackoffStrategy_ShouldRetry(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()

	tests := []struct {
		name     string
		err      error
		attempt  int
		expected bool
	}{
		{
			name:     "retryable error within limit",
			err:      New(ErrPlatformUnavailable, "platform down"),
			attempt:  2,
			expected: true,
		},
		{
			name:     "retryable error at limit",
			err:      New(ErrPlatformUnavailable, "platform down"),
			attempt:  5,
			expected: false,
		},
		{
			name:     "non-retryable error",
			err:      New(ErrInvalidConfig, "invalid config"),
			attempt:  1,
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			attempt:  1,
			expected: false,
		},
		{
			name:     "rate limit error",
			err:      New(ErrPlatformRateLimit, "rate limited"),
			attempt:  2,
			expected: true,
		},
		{
			name:     "network timeout error",
			err:      New(ErrNetworkTimeout, "timeout"),
			attempt:  1,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strategy.ShouldRetry(tt.err, tt.attempt); got != tt.expected {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExponentialBackoffStrategy_GetDelay(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()
	strategy.Jitter = 0 // Disable jitter for predictable testing

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "first attempt",
			attempt:  0,
			expected: 1 * time.Second,
		},
		{
			name:     "second attempt",
			attempt:  1,
			expected: 2 * time.Second,
		},
		{
			name:     "third attempt",
			attempt:  2,
			expected: 4 * time.Second,
		},
		{
			name:     "fourth attempt",
			attempt:  3,
			expected: 8 * time.Second,
		},
		{
			name:     "large attempt (should cap at max)",
			attempt:  10,
			expected: 30 * time.Second, // Should be capped at MaxDelay
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.GetDelay(tt.attempt)
			if got != tt.expected {
				t.Errorf("GetDelay(%d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}

func TestExponentialBackoffStrategy_GetDelayWithJitter(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()
	strategy.Jitter = 0.1

	baseDelay := strategy.GetDelay(1) // Should be around 2 seconds

	// Test that jitter produces different values
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = strategy.GetDelay(1)
	}

	// Check that we get some variation (not all delays are the same)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("Expected jitter to produce different delay values")
	}

	// Check that delays are within reasonable bounds
	minExpected := time.Duration(float64(baseDelay) * 0.8) // Should be at least 80% of base
	maxExpected := time.Duration(float64(baseDelay) * 1.2) // Should be at most 120% of base

	for i, delay := range delays {
		if delay < minExpected || delay > maxExpected {
			t.Errorf("Delay %d (%v) outside expected range [%v, %v]", i, delay, minExpected, maxExpected)
		}
	}
}

func TestLinearBackoffStrategy_ShouldRetry(t *testing.T) {
	strategy := NewLinearBackoffStrategy()

	// Test retryable error
	err := New(ErrPlatformUnavailable, "platform down")
	if !strategy.ShouldRetry(err, 1) {
		t.Error("Expected retryable error to be retryable")
	}

	// Test non-retryable error
	err = New(ErrInvalidConfig, "invalid config")
	if strategy.ShouldRetry(err, 1) {
		t.Error("Expected non-retryable error to not be retryable")
	}

	// Test attempt limit
	err = New(ErrPlatformUnavailable, "platform down")
	if strategy.ShouldRetry(err, 3) {
		t.Error("Expected retry to be blocked when attempt limit reached")
	}
}

func TestLinearBackoffStrategy_GetDelay(t *testing.T) {
	strategy := NewLinearBackoffStrategy()

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "first attempt",
			attempt:  0,
			expected: 2 * time.Second, // BaseDelay
		},
		{
			name:     "second attempt",
			attempt:  1,
			expected: 3 * time.Second, // BaseDelay + 1*DelayIncrement
		},
		{
			name:     "third attempt",
			attempt:  2,
			expected: 4 * time.Second, // BaseDelay + 2*DelayIncrement
		},
		{
			name:     "large attempt (should cap at max)",
			attempt:  20,
			expected: 20 * time.Second, // Should be capped at MaxDelay
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.GetDelay(tt.attempt)
			if got != tt.expected {
				t.Errorf("GetDelay(%d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}

func TestFixedDelayStrategy(t *testing.T) {
	strategy := NewFixedDelayStrategy(5*time.Second, 3)

	// Test delay is always the same
	for i := 0; i < 5; i++ {
		delay := strategy.GetDelay(i)
		if delay != 5*time.Second {
			t.Errorf("Expected fixed delay of 5s, got %v for attempt %d", delay, i)
		}
	}

	// Test max attempts
	if strategy.GetMaxAttempts() != 3 {
		t.Errorf("Expected max attempts 3, got %d", strategy.GetMaxAttempts())
	}

	// Test retryable logic
	err := New(ErrPlatformUnavailable, "platform down")
	if !strategy.ShouldRetry(err, 1) {
		t.Error("Expected retryable error to be retryable")
	}

	if strategy.ShouldRetry(err, 3) {
		t.Error("Expected retry to be blocked when attempt limit reached")
	}
}

func TestRetryExecutor_Execute(t *testing.T) {
	mockLog := &mockLogger{}
	strategy := NewExponentialBackoffStrategy()
	strategy.BaseDelay = 10 * time.Millisecond // Speed up tests
	strategy.MaxAttempts = 3

	executor := NewRetryExecutor(strategy, mockLog)

	t.Run("successful operation on first attempt", func(t *testing.T) {
		mockLog.logs = nil
		attempts := 0

		err := executor.Execute(context.Background(), func() error {
			attempts++
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("successful operation after retries", func(t *testing.T) {
		mockLog.logs = nil
		attempts := 0

		err := executor.Execute(context.Background(), func() error {
			attempts++
			if attempts < 3 {
				return New(ErrPlatformUnavailable, "platform down")
			}
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}

		// Check that warnings were logged
		warnCount := 0
		for _, log := range mockLog.logs {
			if strings.Contains(log, "WARN") && strings.Contains(log, "retrying") {
				warnCount++
			}
		}
		if warnCount != 2 { // Should warn on first 2 failures
			t.Errorf("Expected 2 retry warnings, got %d", warnCount)
		}
	})

	t.Run("non-retryable error", func(t *testing.T) {
		mockLog.logs = nil
		attempts := 0

		err := executor.Execute(context.Background(), func() error {
			attempts++
			return New(ErrInvalidConfig, "invalid config")
		})

		if err == nil {
			t.Error("Expected error, got nil")
		}

		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}

		// Should have debug log about not retrying
		debugFound := false
		for _, log := range mockLog.logs {
			if strings.Contains(log, "DEBUG") && strings.Contains(log, "not retryable") {
				debugFound = true
				break
			}
		}
		if !debugFound {
			t.Error("Expected debug log about non-retryable error")
		}
	})

	t.Run("exhausted retry attempts", func(t *testing.T) {
		mockLog.logs = nil
		attempts := 0

		err := executor.Execute(context.Background(), func() error {
			attempts++
			return New(ErrPlatformUnavailable, "platform down")
		})

		if err == nil {
			t.Error("Expected error, got nil")
		}

		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}

		// Should have error log about exhausted attempts
		errorFound := false
		for _, log := range mockLog.logs {
			if strings.Contains(log, "ERROR") && strings.Contains(log, "failed after all retry attempts") {
				errorFound = true
				break
			}
		}
		if !errorFound {
			t.Error("Expected error log about exhausted retry attempts")
		}
	})

	t.Run("context cancellation during operation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0

		// Cancel context after first attempt
		go func() {
			time.Sleep(5 * time.Millisecond)
			cancel()
		}()

		err := executor.Execute(ctx, func() error {
			attempts++
			return New(ErrPlatformUnavailable, "platform down")
		})

		if err == nil {
			t.Error("Expected error due to context cancellation")
		}

		// Should get context cancellation error
		if notifyErr, ok := err.(*NotifyError); !ok || notifyErr.Code != ErrSystemTimeout {
			t.Errorf("Expected ErrSystemTimeout, got %v", err)
		}
	})

	t.Run("context cancellation during delay", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		strategy := NewExponentialBackoffStrategy()
		strategy.BaseDelay = 200 * time.Millisecond // Longer than context timeout
		strategy.MaxAttempts = 3

		executor := NewRetryExecutor(strategy, mockLog)
		attempts := 0

		err := executor.Execute(ctx, func() error {
			attempts++
			return New(ErrPlatformUnavailable, "platform down")
		})

		if err == nil {
			t.Error("Expected error due to context timeout")
		}

		// Should get context cancellation error
		if notifyErr, ok := err.(*NotifyError); !ok || notifyErr.Code != ErrSystemTimeout {
			t.Errorf("Expected ErrSystemTimeout, got %v", err)
		}
	})
}

func TestRetryExecutor_ExecuteWithCallback(t *testing.T) {
	mockLog := &mockLogger{}
	strategy := NewExponentialBackoffStrategy()
	strategy.BaseDelay = 10 * time.Millisecond
	strategy.MaxAttempts = 3

	executor := NewRetryExecutor(strategy, mockLog)

	retryCallbacks := 0
	var lastAttempt int
	var lastError error
	var lastDelay time.Duration

	onRetry := func(attempt int, err error, delay time.Duration) {
		retryCallbacks++
		lastAttempt = attempt
		lastError = err
		lastDelay = delay
	}

	t.Run("callback called on retries", func(t *testing.T) {
		attempts := 0

		err := executor.ExecuteWithCallback(context.Background(), func() error {
			attempts++
			if attempts < 3 {
				return New(ErrPlatformUnavailable, "platform down")
			}
			return nil
		}, onRetry)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if retryCallbacks != 2 {
			t.Errorf("Expected 2 retry callbacks, got %d", retryCallbacks)
		}

		if lastAttempt != 1 { // Last callback should be for attempt 1 (second attempt)
			t.Errorf("Expected last attempt to be 1, got %d", lastAttempt)
		}

		if lastError == nil {
			t.Error("Expected last error to be set")
		}

		if lastDelay <= 0 {
			t.Errorf("Expected positive delay, got %v", lastDelay)
		}
	})

	t.Run("callback with nil function", func(t *testing.T) {
		attempts := 0

		err := executor.ExecuteWithCallback(context.Background(), func() error {
			attempts++
			if attempts < 2 {
				return New(ErrPlatformUnavailable, "platform down")
			}
			return nil
		}, nil) // nil callback

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if attempts != 2 {
			t.Errorf("Expected 2 attempts, got %d", attempts)
		}
	})
}

func TestRetryConfig(t *testing.T) {
	t.Run("DefaultRetryConfig", func(t *testing.T) {
		config := DefaultRetryConfig()

		if config.Strategy != "exponential" {
			t.Errorf("Expected strategy 'exponential', got %v", config.Strategy)
		}

		if config.BaseDelay != 1*time.Second {
			t.Errorf("Expected base delay 1s, got %v", config.BaseDelay)
		}

		if config.MaxAttempts != 5 {
			t.Errorf("Expected max attempts 5, got %d", config.MaxAttempts)
		}

		if config.Multiplier != 2.0 {
			t.Errorf("Expected multiplier 2.0, got %v", config.Multiplier)
		}
	})

	t.Run("CreateRetryStrategy exponential", func(t *testing.T) {
		config := RetryConfig{
			Strategy:    "exponential",
			BaseDelay:   2 * time.Second,
			MaxDelay:    60 * time.Second,
			MaxAttempts: 10,
			Multiplier:  3.0,
			Jitter:      0.2,
		}

		strategy := CreateRetryStrategy(config)
		expStrategy, ok := strategy.(*ExponentialBackoffStrategy)
		if !ok {
			t.Fatal("Expected ExponentialBackoffStrategy")
		}

		if expStrategy.BaseDelay != 2*time.Second {
			t.Errorf("Expected base delay 2s, got %v", expStrategy.BaseDelay)
		}

		if expStrategy.MaxAttempts != 10 {
			t.Errorf("Expected max attempts 10, got %d", expStrategy.MaxAttempts)
		}

		if expStrategy.Multiplier != 3.0 {
			t.Errorf("Expected multiplier 3.0, got %v", expStrategy.Multiplier)
		}
	})

	t.Run("CreateRetryStrategy linear", func(t *testing.T) {
		config := RetryConfig{
			Strategy:    "linear",
			BaseDelay:   3 * time.Second,
			MaxDelay:    30 * time.Second,
			MaxAttempts: 5,
		}

		strategy := CreateRetryStrategy(config)
		linStrategy, ok := strategy.(*LinearBackoffStrategy)
		if !ok {
			t.Fatal("Expected LinearBackoffStrategy")
		}

		if linStrategy.BaseDelay != 3*time.Second {
			t.Errorf("Expected base delay 3s, got %v", linStrategy.BaseDelay)
		}

		if linStrategy.MaxAttempts != 5 {
			t.Errorf("Expected max attempts 5, got %d", linStrategy.MaxAttempts)
		}
	})

	t.Run("CreateRetryStrategy fixed", func(t *testing.T) {
		config := RetryConfig{
			Strategy:    "fixed",
			BaseDelay:   5 * time.Second,
			MaxAttempts: 3,
		}

		strategy := CreateRetryStrategy(config)
		fixedStrategy, ok := strategy.(*FixedDelayStrategy)
		if !ok {
			t.Fatal("Expected FixedDelayStrategy")
		}

		if fixedStrategy.Delay != 5*time.Second {
			t.Errorf("Expected delay 5s, got %v", fixedStrategy.Delay)
		}

		if fixedStrategy.MaxAttempts != 3 {
			t.Errorf("Expected max attempts 3, got %d", fixedStrategy.MaxAttempts)
		}
	})

	t.Run("CreateRetryStrategy unknown defaults to exponential", func(t *testing.T) {
		config := RetryConfig{
			Strategy:    "unknown",
			BaseDelay:   1 * time.Second,
			MaxAttempts: 3,
		}

		strategy := CreateRetryStrategy(config)
		_, ok := strategy.(*ExponentialBackoffStrategy)
		if !ok {
			t.Fatal("Expected unknown strategy to default to ExponentialBackoffStrategy")
		}
	})
}

func TestRetryableErrorCodes(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()

	// Test all retryable codes
	retryableCodes := []Code{
		// Platform errors
		ErrPlatformUnavailable, ErrPlatformRateLimit, ErrPlatformTimeout,
		ErrPlatformInternal, ErrPlatformMaintenance,
		// Network errors
		ErrNetworkTimeout, ErrNetworkConnection, ErrNetworkDNS,
		// Queue errors
		ErrQueueFull, ErrQueueTimeout, ErrQueueConnection, ErrQueueWorkerFailed,
		// System errors
		ErrSystemUnavailable, ErrResourceExhausted, ErrSystemTimeout, ErrSystemOverload,
		// Message errors
		ErrMessageSendFailed, ErrMessageTimeout,
		// Template errors
		ErrTemplateCacheError,
	}

	for _, code := range retryableCodes {
		t.Run(string(code), func(t *testing.T) {
			err := New(code, "test error")
			if !strategy.ShouldRetry(err, 1) {
				t.Errorf("Error code %v should be retryable", code)
			}
		})
	}

	// Test non-retryable codes
	nonRetryableCodes := []Code{
		// Configuration errors
		ErrInvalidConfig, ErrMissingConfig, ErrConfigValidation, ErrUnsupportedConfig,
		// Platform errors
		ErrPlatformNotFound, ErrPlatformAuth,
		// Message errors
		ErrInvalidMessage, ErrMessageTooLarge, ErrInvalidTarget, ErrMessageEncoding,
		// Template errors
		ErrTemplateNotFound, ErrTemplateInvalid, ErrTemplateRender,
		ErrTemplateEngine, ErrTemplateVariables,
		// Network errors
		ErrNetworkSSL, ErrNetworkProtocol,
		// Validation errors
		ErrValidationFailed, ErrInvalidFormat, ErrMissingRequired,
		ErrValueOutOfRange, ErrInvalidType,
		// System errors
		ErrInternalError, ErrPermissionDenied,
	}

	for _, code := range nonRetryableCodes {
		t.Run(string(code), func(t *testing.T) {
			err := New(code, "test error")
			if strategy.ShouldRetry(err, 1) {
				t.Errorf("Error code %v should not be retryable", code)
			}
		})
	}
}

func BenchmarkExponentialBackoffStrategy_GetDelay(b *testing.B) {
	strategy := NewExponentialBackoffStrategy()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strategy.GetDelay(i % 10)
	}
}

func BenchmarkExponentialBackoffStrategy_ShouldRetry(b *testing.B) {
	strategy := NewExponentialBackoffStrategy()
	err := New(ErrPlatformUnavailable, "test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strategy.ShouldRetry(err, i%5)
	}
}

func BenchmarkRetryExecutor_Execute(b *testing.B) {
	mockLog := &mockLogger{}
	strategy := NewFixedDelayStrategy(1*time.Millisecond, 3)
	executor := NewRetryExecutor(strategy, mockLog)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.Execute(context.Background(), func() error {
			return nil // Always succeed for benchmark
		})
	}
}