// Package feishu provides HTTP client functionality for Feishu platform
package feishu

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// Config structures for HTTP client settings
type TimeoutConfig struct {
	Connect, Read, Write, Total time.Duration
}

type ConnectionPoolConfig struct {
	MaxIdleConns, MaxIdleConnsPerHost, MaxConnsPerHost int
	IdleConnTimeout                                     time.Duration
}

type RetryConfig struct {
	MaxRetries                    int
	InitialInterval, MaxInterval  time.Duration
	Multiplier                    float64
	Jitter                        bool
}

// HTTPClientMetrics contains metrics for HTTP client operations
type HTTPClientMetrics struct {
	RequestCount, SuccessCount, ErrorCount, RetryCount, TotalDuration int64
}

// HTTPClient wraps the HTTP client with Feishu-specific functionality
type HTTPClient struct {
	client         *http.Client
	logger         logger.Logger
	retryConfig    RetryConfig
	timeoutConfig  TimeoutConfig
	defaultHeaders map[string]string
	metrics        *HTTPClientMetrics
	closed         int32
}

// FeishuResponse represents the response structure from Feishu API
type FeishuResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// Error types and handling
type ErrorType string
const (
	ErrorTypeNetwork, ErrorTypeClient, ErrorTypeServer, ErrorTypeRateLimit, ErrorTypeAuth = "network", "client", "server", "rate_limit", "auth"
)

func getErrorType(statusCode int) ErrorType {
	switch {
	case statusCode == 429: return ErrorTypeRateLimit
	case statusCode == 401 || statusCode == 403: return ErrorTypeAuth
	case statusCode >= 400 && statusCode < 500: return ErrorTypeClient
	case statusCode >= 500: return ErrorTypeServer
	default: return ErrorTypeNetwork
	}
}

type ClassifiedError struct {
	Type ErrorType; OriginalError error; IsRetryable bool; StatusCode *int
}

func (e *ClassifiedError) Error() string {
	if e.StatusCode != nil { return fmt.Sprintf("%s error (%d): %v", e.Type, *e.StatusCode, e.OriginalError) }
	return fmt.Sprintf("%s error: %v", e.Type, e.OriginalError)
}
func (e *ClassifiedError) Unwrap() error { return e.OriginalError }

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message) }
// Default configurations
func DefaultTimeoutConfig() TimeoutConfig { return TimeoutConfig{10 * time.Second, 30 * time.Second, 30 * time.Second, 60 * time.Second} }
func DefaultConnectionPoolConfig() ConnectionPoolConfig { return ConnectionPoolConfig{100, 10, 30, 90 * time.Second} }
func DefaultRetryConfig() RetryConfig { return RetryConfig{3, 100 * time.Millisecond, 5 * time.Second, 2.0, true} }

// createHTTPClient creates an HTTP client with custom transport settings
func createHTTPClient(tc TimeoutConfig, pc ConnectionPoolConfig) *http.Client {
	return &http.Client{
		Timeout: tc.Total,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: tc.Connect, KeepAlive: 30 * time.Second}).DialContext,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			MaxIdleConns: pc.MaxIdleConns, MaxIdleConnsPerHost: pc.MaxIdleConnsPerHost,
			MaxConnsPerHost: pc.MaxConnsPerHost, IdleConnTimeout: pc.IdleConnTimeout,
			TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: tc.Read,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// NewHTTPClient creates a new HTTP client wrapper with default configuration
func NewHTTPClient(logger logger.Logger) *HTTPClient {
	return NewHTTPClientWithConfig(logger, DefaultTimeoutConfig(), DefaultConnectionPoolConfig(), DefaultRetryConfig())
}

// NewHTTPClientWithConfig creates a new HTTP client wrapper with custom configuration
func NewHTTPClientWithConfig(logger logger.Logger, tc TimeoutConfig, pc ConnectionPoolConfig, rc RetryConfig) *HTTPClient {
	return &HTTPClient{
		client: createHTTPClient(tc, pc), logger: logger, retryConfig: rc, timeoutConfig: tc,
		defaultHeaders: make(map[string]string), metrics: &HTTPClientMetrics{}, closed: 0,
	}
}

// SetDefaultHeaders sets default headers for all requests
func (h *HTTPClient) SetDefaultHeaders(headers map[string]string) { for k, v := range headers { h.defaultHeaders[k] = v } }

// SendToWebhook sends the message to Feishu webhook with retry mechanism
func (h *HTTPClient) SendToWebhook(ctx context.Context, webhookURL string, feishuMsg *FeishuMessage, headers ...map[string]string) error {
	if atomic.LoadInt32(&h.closed) == 1 { return fmt.Errorf("HTTP client is closed") }
	startTime := time.Now(); atomic.AddInt64(&h.metrics.RequestCount, 1)
	h.logger.Debug("Sending to Feishu webhook", "webhookURL", webhookURL)

	payload, err := json.Marshal(feishuMsg)
	if err != nil {
		h.logger.Error("Failed to marshal Feishu message", "error", err); atomic.AddInt64(&h.metrics.ErrorCount, 1)
		return fmt.Errorf("failed to marshal Feishu message: %w", err)
	}

	// Merge headers
	reqHeaders := make(map[string]string)
	for k, v := range h.defaultHeaders { reqHeaders[k] = v }
	for _, headerMap := range headers {
		for k, v := range headerMap { reqHeaders[k] = v }
	}

	err = h.executeWithRetry(ctx, func() error { return h.sendRequest(ctx, webhookURL, payload, reqHeaders) })
	atomic.AddInt64(&h.metrics.TotalDuration, time.Since(startTime).Nanoseconds())
	if err != nil { atomic.AddInt64(&h.metrics.ErrorCount, 1); return err }
	atomic.AddInt64(&h.metrics.SuccessCount, 1); return nil
}

// executeWithRetry executes the given operation with exponential backoff retry
func (h *HTTPClient) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error
	interval := h.retryConfig.InitialInterval

	for attempt := 0; attempt <= h.retryConfig.MaxRetries; attempt++ {
		select {
		case <-ctx.Done(): return ctx.Err()
		default:
		}

		err := operation()
		if err == nil { return nil }

		lastErr = err
		if attempt == h.retryConfig.MaxRetries { break }

		if !h.isRetryableError(err) {
			h.logger.Debug("Error is not retryable, aborting", "error", err, "attempt", attempt+1)
			return err
		}

		atomic.AddInt64(&h.metrics.RetryCount, 1)
		h.logger.Debug("Retrying request", "attempt", attempt+1, "error", err, "interval", interval)

		if err := h.waitWithJitter(ctx, interval); err != nil { return err }

		interval = time.Duration(float64(interval) * h.retryConfig.Multiplier)
		if interval > h.retryConfig.MaxInterval { interval = h.retryConfig.MaxInterval }
	}

	h.logger.Error("All retry attempts failed", "maxRetries", h.retryConfig.MaxRetries, "lastError", lastErr)
	return fmt.Errorf("request failed after %d attempts: %w", h.retryConfig.MaxRetries+1, lastErr)
}

// sendRequest sends the actual HTTP request
func (h *HTTPClient) sendRequest(ctx context.Context, webhookURL string, payload []byte, headers map[string]string) error {
	h.logger.Debug("Sending HTTP request to Feishu", "url", webhookURL, "payloadSize", len(payload))

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(payload))
	if err != nil { return h.classifyError(fmt.Errorf("failed to create HTTP request: %w", err)) }

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NotifyHub-Feishu-Client/1.0")
	for k, v := range headers { req.Header.Set(k, v) }

	resp, err := h.client.Do(req)
	if err != nil { return h.classifyError(fmt.Errorf("failed to send HTTP request: %w", err)) }
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Failed to read response body", "error", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	h.logger.Debug("Feishu API Response", "statusCode", resp.StatusCode, "body", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Feishu API returned non-OK status", "statusCode", resp.StatusCode, "body", string(bodyBytes))
		return h.classifyError(&HTTPError{StatusCode: resp.StatusCode, Message: string(bodyBytes)})
	}

	// Check Feishu API response
	var feishuResp FeishuResponse
	if err := json.Unmarshal(bodyBytes, &feishuResp); err == nil {
		if feishuResp.Code != 0 {
			h.logger.Error("Feishu API returned error code", "code", feishuResp.Code, "msg", feishuResp.Msg)
			return fmt.Errorf("feishu API error (code %d): %s", feishuResp.Code, feishuResp.Msg)
		}
		h.logger.Info("Feishu message sent successfully", "statusCode", resp.StatusCode)
	} else {
		h.logger.Debug("Response is not in Feishu format (likely testing endpoint)", "parseError", err)
		h.logger.Info("Message sent successfully to endpoint", "statusCode", resp.StatusCode)
	}
	return nil
}

// classifyError classifies errors for better handling and reporting
func (h *HTTPClient) classifyError(err error) error {
	if err == nil { return nil }

	switch e := err.(type) {
	case *HTTPError:
		return &ClassifiedError{
			Type: getErrorType(e.StatusCode), OriginalError: err,
			IsRetryable: e.StatusCode >= 500 && e.StatusCode < 600, StatusCode: &e.StatusCode,
		}
	default:
		isRetryable := err != context.Canceled && err != context.DeadlineExceeded
		return &ClassifiedError{Type: ErrorTypeNetwork, OriginalError: err, IsRetryable: isRetryable}
	}
}

// isRetryableError determines if an error should be retried
func (h *HTTPClient) isRetryableError(err error) bool {
	if err == nil { return false }
	if classifiedErr, ok := err.(*ClassifiedError); ok { return classifiedErr.IsRetryable }
	if err == context.Canceled || err == context.DeadlineExceeded { return false }
	if httpErr, ok := err.(*HTTPError); ok { return httpErr.StatusCode >= 500 }
	return true
}

// waitWithJitter waits for the specified interval with optional jitter
func (h *HTTPClient) waitWithJitter(ctx context.Context, interval time.Duration) error {
	waitTime := interval
	if h.retryConfig.Jitter {
		jitterRange := float64(interval) * 0.25
		jitter := time.Duration((rand.Float64() - 0.5) * 2 * jitterRange)
		waitTime = interval + jitter
		if waitTime < 0 { waitTime = interval / 4 }
	}
	select {
	case <-ctx.Done(): return ctx.Err()
	case <-time.After(waitTime): return nil
	}
}

// GetMetrics returns current client metrics
func (h *HTTPClient) GetMetrics() HTTPClientMetrics {
	return HTTPClientMetrics{atomic.LoadInt64(&h.metrics.RequestCount), atomic.LoadInt64(&h.metrics.SuccessCount),
		atomic.LoadInt64(&h.metrics.ErrorCount), atomic.LoadInt64(&h.metrics.RetryCount), atomic.LoadInt64(&h.metrics.TotalDuration)}
}

// ResetMetrics resets all metrics to zero
func (h *HTTPClient) ResetMetrics() {
	atomic.StoreInt64(&h.metrics.RequestCount, 0); atomic.StoreInt64(&h.metrics.SuccessCount, 0); atomic.StoreInt64(&h.metrics.ErrorCount, 0); atomic.StoreInt64(&h.metrics.RetryCount, 0); atomic.StoreInt64(&h.metrics.TotalDuration, 0)
}

// IsHealthy performs a health check on the HTTP client
func (h *HTTPClient) IsHealthy(ctx context.Context) error {
	if atomic.LoadInt32(&h.closed) == 1 { return fmt.Errorf("HTTP client is closed") }
	if h.client == nil { return fmt.Errorf("HTTP client is not initialized") }
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second); defer cancel()
	req, err := http.NewRequestWithContext(testCtx, "HEAD", "https://httpbin.org/status/200", nil)
	if err != nil { return fmt.Errorf("failed to create health check request: %w", err) }
	resp, err := h.client.Do(req)
	if err != nil { return fmt.Errorf("health check failed: %w", err) }
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("health check returned status %d", resp.StatusCode) }
	return nil
}

// Close closes the HTTP client and cleans up resources
func (h *HTTPClient) Close() error {
	if !atomic.CompareAndSwapInt32(&h.closed, 0, 1) { return nil }
	if h.client != nil { h.client.CloseIdleConnections(); h.logger.Info("HTTP client closed successfully") }
	return nil
}