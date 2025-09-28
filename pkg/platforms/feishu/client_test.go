package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// mockLogger implements logger.Logger for testing
type mockLogger struct {
	logs []string
}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf("DEBUG: %s %v", msg, keysAndValues))
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf("INFO: %s %v", msg, keysAndValues))
}

func (m *mockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf("WARN: %s %v", msg, keysAndValues))
}

func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf("ERROR: %s %v", msg, keysAndValues))
}

func (m *mockLogger) With(keysAndValues ...interface{}) logger.Logger {
	return m
}

func newMockLogger() *mockLogger {
	return &mockLogger{logs: make([]string, 0)}
}

func TestNewHTTPClient(t *testing.T) {
	tests := []struct {
		name   string
		client *http.Client
		logger logger.Logger
	}{
		{
			name:   "with valid parameters",
			client: http.DefaultClient,
			logger: newMockLogger(),
		},
		{
			name:   "with nil client",
			client: nil,
			logger: newMockLogger(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := NewHTTPClient(tt.client, tt.logger)

			assert.NotNil(t, httpClient)
			assert.Equal(t, tt.client, httpClient.client)
			assert.Equal(t, tt.logger, httpClient.logger)
			assert.Equal(t, DefaultRetryConfig(), httpClient.retryConfig)
		})
	}
}

func TestNewHTTPClientWithRetry(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries:      5,
		InitialInterval: 200 * time.Millisecond,
		MaxInterval:     10 * time.Second,
		Multiplier:      1.5,
		Jitter:          false,
	}

	httpClient := NewHTTPClientWithRetry(http.DefaultClient, newMockLogger(), retryConfig)

	assert.NotNil(t, httpClient)
	assert.Equal(t, retryConfig, httpClient.retryConfig)
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.InitialInterval)
	assert.Equal(t, 5*time.Second, config.MaxInterval)
	assert.Equal(t, 2.0, config.Multiplier)
	assert.True(t, config.Jitter)
}

func TestSendToWebhook_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		response := FeishuResponse{
			Code: 0,
			Msg:  "success",
			Data: nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create HTTP client
	mockLog := newMockLogger()
	httpClient := NewHTTPClient(http.DefaultClient, mockLog)

	// Create test message
	msg := &FeishuMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": "test message",
		},
	}

	// Test
	err := httpClient.SendToWebhook(context.Background(), server.URL, msg)

	assert.NoError(t, err)
	// Check that success log was written
	assert.True(t, containsLog(mockLog.logs, "INFO: Feishu message sent successfully"))
}

func TestSendToWebhook_HTTPError(t *testing.T) {
	// Create test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Create HTTP client with no retries for faster test
	mockLog := newMockLogger()
	retryConfig := RetryConfig{
		MaxRetries:      0, // No retries
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}
	httpClient := NewHTTPClientWithRetry(http.DefaultClient, mockLog, retryConfig)

	msg := &FeishuMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": "test message",
		},
	}

	// Test
	err := httpClient.SendToWebhook(context.Background(), server.URL, msg)

	assert.Error(t, err)
	var httpErr *HTTPError
	assert.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusInternalServerError, httpErr.StatusCode)
}

func TestSendToWebhook_FeishuAPIError(t *testing.T) {
	// Create test server that returns Feishu API error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := FeishuResponse{
			Code: 9999,
			Msg:  "invalid webhook url",
			Data: nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	mockLog := newMockLogger()
	httpClient := NewHTTPClient(http.DefaultClient, mockLog)

	msg := &FeishuMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": "test message",
		},
	}

	// Test
	err := httpClient.SendToWebhook(context.Background(), server.URL, msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feishu API error (code 9999)")
}

func TestSendToWebhook_NetworkError(t *testing.T) {
	mockLog := newMockLogger()
	httpClient := NewHTTPClient(http.DefaultClient, mockLog)

	msg := &FeishuMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": "test message",
		},
	}

	// Test with invalid URL
	err := httpClient.SendToWebhook(context.Background(), "http://invalid-url-that-does-not-exist", msg)

	assert.Error(t, err)
}

func TestSendToWebhook_InvalidJSON(t *testing.T) {
	mockLog := newMockLogger()
	httpClient := NewHTTPClient(http.DefaultClient, mockLog)

	// Create message with invalid JSON (circular reference)
	msg := &FeishuMessage{
		MsgType: "text",
		Content: make(map[string]interface{}),
	}
	msg.Content["self"] = msg // Circular reference

	// Test
	err := httpClient.SendToWebhook(context.Background(), "http://example.com", msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal Feishu message")
}

func TestSendToWebhook_ContextCancellation(t *testing.T) {
	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockLog := newMockLogger()
	httpClient := NewHTTPClient(http.DefaultClient, mockLog)

	msg := &FeishuMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": "test message",
		},
	}

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Test
	err := httpClient.SendToWebhook(ctx, server.URL, msg)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestRetryLogic(t *testing.T) {
	requestCount := 0

	// Create test server that fails first 2 requests, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := FeishuResponse{
			Code: 0,
			Msg:  "success",
			Data: nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create HTTP client with retries
	mockLog := newMockLogger()
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}
	httpClient := NewHTTPClientWithRetry(http.DefaultClient, mockLog, retryConfig)

	msg := &FeishuMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": "test message",
		},
	}

	// Test
	err := httpClient.SendToWebhook(context.Background(), server.URL, msg)

	assert.NoError(t, err)
	assert.Equal(t, 3, requestCount) // Should have made 3 requests (2 failures + 1 success)
}

func TestRetryLogic_AllFailures(t *testing.T) {
	requestCount := 0

	// Create test server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create HTTP client with limited retries
	mockLog := newMockLogger()
	retryConfig := RetryConfig{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}
	httpClient := NewHTTPClientWithRetry(http.DefaultClient, mockLog, retryConfig)

	msg := &FeishuMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": "test message",
		},
	}

	// Test
	err := httpClient.SendToWebhook(context.Background(), server.URL, msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed after 3 attempts")
	assert.Equal(t, 3, requestCount) // Should have made 3 attempts (1 initial + 2 retries)
}

func TestIsRetryableError(t *testing.T) {
	mockLog := newMockLogger()
	httpClient := NewHTTPClient(http.DefaultClient, mockLog)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "HTTP 400 error",
			err:      &HTTPError{StatusCode: 400, Message: "Bad Request"},
			expected: false,
		},
		{
			name:     "HTTP 404 error",
			err:      &HTTPError{StatusCode: 404, Message: "Not Found"},
			expected: false,
		},
		{
			name:     "HTTP 500 error",
			err:      &HTTPError{StatusCode: 500, Message: "Internal Server Error"},
			expected: true,
		},
		{
			name:     "HTTP 502 error",
			err:      &HTTPError{StatusCode: 502, Message: "Bad Gateway"},
			expected: true,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("network error"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := httpClient.isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWaitWithJitter(t *testing.T) {
	mockLog := newMockLogger()
	retryConfig := RetryConfig{
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		Jitter:          true,
	}
	httpClient := NewHTTPClientWithRetry(http.DefaultClient, mockLog, retryConfig)

	// Test with jitter enabled
	start := time.Now()
	err := httpClient.waitWithJitter(context.Background(), 50*time.Millisecond)
	duration := time.Since(start)

	assert.NoError(t, err)
	// Should wait approximately 50ms +/- 25% jitter
	assert.True(t, duration >= 35*time.Millisecond && duration <= 70*time.Millisecond)
}

func TestWaitWithJitter_ContextCancellation(t *testing.T) {
	mockLog := newMockLogger()
	httpClient := NewHTTPClient(http.DefaultClient, mockLog)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := httpClient.waitWithJitter(ctx, 100*time.Millisecond)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestIsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		client   *http.Client
		expected bool
	}{
		{
			name:     "with valid client",
			client:   http.DefaultClient,
			expected: true,
		},
		{
			name:     "with nil client",
			client:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLog := newMockLogger()
			httpClient := NewHTTPClient(tt.client, mockLog)

			err := httpClient.IsHealthy(context.Background())

			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "HTTP client is not initialized")
			}
		})
	}
}

func TestClose(t *testing.T) {
	mockLog := newMockLogger()
	httpClient := NewHTTPClient(http.DefaultClient, mockLog)

	err := httpClient.Close()
	assert.NoError(t, err)
}

func TestHTTPError(t *testing.T) {
	err := &HTTPError{
		StatusCode: 404,
		Message:    "Not Found",
	}

	assert.Equal(t, "HTTP 404: Not Found", err.Error())
}

// Helper function to check if logs contain specific message
func containsLog(logs []string, message string) bool {
	for _, log := range logs {
		if strings.Contains(log, message) {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkSendToWebhook(b *testing.B) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := FeishuResponse{
			Code: 0,
			Msg:  "success",
			Data: nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	mockLog := newMockLogger()
	httpClient := NewHTTPClient(http.DefaultClient, mockLog)

	msg := &FeishuMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": "benchmark test message",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := httpClient.SendToWebhook(context.Background(), server.URL, msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRetryLogic(b *testing.B) {
	requestCount := 0

	// Create test server that fails first request, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount%2 == 1 { // Fail every odd request
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := FeishuResponse{
			Code: 0,
			Msg:  "success",
			Data: nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	mockLog := newMockLogger()
	retryConfig := RetryConfig{
		MaxRetries:      1,
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}
	httpClient := NewHTTPClientWithRetry(http.DefaultClient, mockLog, retryConfig)

	msg := &FeishuMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": "benchmark test message",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := httpClient.SendToWebhook(context.Background(), server.URL, msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}