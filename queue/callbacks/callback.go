package callbacks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/kart-io/notifyhub/core"
	queueCore "github.com/kart-io/notifyhub/queue/core"
)

// CallbackEvent represents the event type for callbacks
type CallbackEvent string

const (
	// CallbackEventSent fires when message is successfully sent
	CallbackEventSent CallbackEvent = "sent"
	// CallbackEventFailed fires when message sending fails
	CallbackEventFailed CallbackEvent = "failed"
	// CallbackEventRetry fires when message is being retried
	CallbackEventRetry CallbackEvent = "retry"
	// CallbackEventMaxRetries fires when max retries are exceeded
	CallbackEventMaxRetries CallbackEvent = "max_retries"
)

// CallbackContext contains information about the callback execution
type CallbackContext struct {
	MessageID  string               `json:"message_id"`
	Event      CallbackEvent        `json:"event"`
	Message    interface{}          `json:"message"` // Can be *message.Message
	Results    *core.SendingResults `json:"results,omitempty"`
	Error      error                `json:"error,omitempty"`
	Attempts   int                  `json:"attempts"`
	ExecutedAt time.Time            `json:"executed_at"`
	Duration   time.Duration        `json:"duration,omitempty"`
}

// Callback defines the interface for message processing callbacks
type Callback interface {
	// Execute is called when the callback event occurs
	Execute(ctx context.Context, callbackCtx *CallbackContext) error

	// Name returns the callback name for identification
	Name() string
}

// CallbackFunc is a function type that implements the Callback interface
type CallbackFunc struct {
	name string
	fn   func(ctx context.Context, callbackCtx *CallbackContext) error
}

// NewCallbackFunc creates a new callback function
func NewCallbackFunc(name string, fn func(ctx context.Context, callbackCtx *CallbackContext) error) *CallbackFunc {
	return &CallbackFunc{name: name, fn: fn}
}

// Execute implements the Callback interface
func (cf *CallbackFunc) Execute(ctx context.Context, callbackCtx *CallbackContext) error {
	return cf.fn(ctx, callbackCtx)
}

// Name implements the Callback interface
func (cf *CallbackFunc) Name() string {
	return cf.name
}

// CallbackOptions defines callback configuration for a message
type CallbackOptions struct {
	// OnSent callback for successful sends
	OnSent []Callback `json:"-"`
	// OnFailed callback for failed sends
	OnFailed []Callback `json:"-"`
	// OnRetry callback for retries
	OnRetry []Callback `json:"-"`
	// OnMaxRetries callback when max retries exceeded
	OnMaxRetries []Callback `json:"-"`
	// WebhookURL for HTTP callback (alternative to function callbacks)
	WebhookURL string `json:"webhook_url,omitempty"`
	// WebhookSecret for webhook authentication
	WebhookSecret string `json:"webhook_secret,omitempty"`
	// CallbackTimeout timeout for callback execution
	CallbackTimeout time.Duration `json:"callback_timeout"`
}

// AddCallback adds a callback for a specific event
func (co *CallbackOptions) AddCallback(event CallbackEvent, callback Callback) {
	switch event {
	case CallbackEventSent:
		co.OnSent = append(co.OnSent, callback)
	case CallbackEventFailed:
		co.OnFailed = append(co.OnFailed, callback)
	case CallbackEventRetry:
		co.OnRetry = append(co.OnRetry, callback)
	case CallbackEventMaxRetries:
		co.OnMaxRetries = append(co.OnMaxRetries, callback)
	}
}

// CallbackExecutor handles callback execution
type CallbackExecutor struct {
	client *http.Client
}

// NewCallbackExecutor creates a new callback executor
func NewCallbackExecutor() *CallbackExecutor {
	return &CallbackExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteCallbacks executes all callbacks for a given event
func (ce *CallbackExecutor) ExecuteCallbacks(ctx context.Context, event CallbackEvent, queueMsg *queueCore.Message, results *core.SendingResults, err error, duration time.Duration) {
	// Type assert callbacks to our CallbackOptions type
	callbacks, ok := queueMsg.Callbacks.(*CallbackOptions)
	if !ok || callbacks == nil {
		return
	}

	callbackCtx := &CallbackContext{
		MessageID:  queueMsg.ID,
		Event:      event,
		Message:    queueMsg.Message,
		Results:    results,
		Error:      err,
		Attempts:   queueMsg.Attempts,
		ExecutedAt: time.Now(),
		Duration:   duration,
	}

	// Get callbacks for this event
	var callbackList []Callback
	switch event {
	case CallbackEventSent:
		callbackList = callbacks.OnSent
	case CallbackEventFailed:
		callbackList = callbacks.OnFailed
	case CallbackEventRetry:
		callbackList = callbacks.OnRetry
	case CallbackEventMaxRetries:
		callbackList = callbacks.OnMaxRetries
	}

	// Execute function callbacks
	for _, callback := range callbackList {
		go ce.executeCallback(ctx, callback, callbackCtx, callbacks.CallbackTimeout)
	}

	// Execute webhook callback if configured
	if callbacks.WebhookURL != "" {
		go ce.executeWebhookCallback(ctx, callbacks, callbackCtx)
	}
}

// executeCallback executes a single callback with timeout
func (ce *CallbackExecutor) executeCallback(ctx context.Context, callback Callback, callbackCtx *CallbackContext, timeout time.Duration) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	callbackCtxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err := callback.Execute(callbackCtxWithTimeout, callbackCtx)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Callback %s failed for message %s: %v (took %v)", callback.Name(), callbackCtx.MessageID, err, duration)
	} else {
		log.Printf("Callback %s succeeded for message %s (took %v)", callback.Name(), callbackCtx.MessageID, duration)
	}
}

// executeWebhookCallback executes webhook callback via HTTP
func (ce *CallbackExecutor) executeWebhookCallback(ctx context.Context, callbacks *CallbackOptions, callbackCtx *CallbackContext) {
	timeout := callbacks.CallbackTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	webhookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare webhook payload
	payload, err := json.Marshal(callbackCtx)
	if err != nil {
		log.Printf("Failed to marshal webhook payload for message %s: %v", callbackCtx.MessageID, err)
		return
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(webhookCtx, "POST", callbacks.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Failed to create webhook request for message %s: %v", callbackCtx.MessageID, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NotifyHub-Webhook/1.0")

	// Add HMAC signature if secret is provided
	if callbacks.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(callbacks.WebhookSecret))
		mac.Write(payload)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-NotifyHub-Signature", "sha256="+signature)
	}

	// Execute webhook
	start := time.Now()
	resp, err := ce.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Webhook failed for message %s: %v (took %v)", callbackCtx.MessageID, err, duration)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Webhook succeeded for message %s: HTTP %d (took %v)", callbackCtx.MessageID, resp.StatusCode, duration)
	} else {
		log.Printf("Webhook failed for message %s: HTTP %d (took %v)", callbackCtx.MessageID, resp.StatusCode, duration)
	}
}

// ================================
// 内置回调实现
// ================================

// LoggingCallback is a simple callback that logs message events
type LoggingCallback struct {
	name   string
	logger func(format string, v ...interface{})
}

// NewLoggingCallback creates a new logging callback
func NewLoggingCallback(name string, logger func(format string, v ...interface{})) *LoggingCallback {
	if logger == nil {
		logger = log.Printf
	}
	return &LoggingCallback{name: name, logger: logger}
}

// Execute implements the Callback interface
func (lc *LoggingCallback) Execute(ctx context.Context, callbackCtx *CallbackContext) error {
	switch callbackCtx.Event {
	case CallbackEventSent:
		lc.logger("Message %s sent successfully after %d attempts", callbackCtx.MessageID, callbackCtx.Attempts)
	case CallbackEventFailed:
		lc.logger("Message %s failed: %v (attempt %d)", callbackCtx.MessageID, callbackCtx.Error, callbackCtx.Attempts)
	case CallbackEventRetry:
		lc.logger("Message %s will be retried (attempt %d)", callbackCtx.MessageID, callbackCtx.Attempts)
	case CallbackEventMaxRetries:
		lc.logger("Message %s exceeded max retries (%d attempts)", callbackCtx.MessageID, callbackCtx.Attempts)
	}
	return nil
}

// Name implements the Callback interface
func (lc *LoggingCallback) Name() string {
	return lc.name
}
