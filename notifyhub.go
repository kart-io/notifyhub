// Package notifyhub provides a unified notification system that supports multiple platforms
// including Email and Feishu, with template rendering, queue management, monitoring, and callbacks.
//
// Basic usage:
//
//	hub, err := notifyhub.New(nil) // Load config from env
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	message := notifyhub.NewAlert("Alert Title", "Alert message").
//		Email("admin@company.com").
//		Build()
//
//	results, err := hub.Send(context.Background(), message, nil)
//
// Advanced usage with callbacks:
//
//	callbacks := &notifyhub.CallbackOptions{
//		WebhookURL: "https://example.com/webhook",
//	}
//	callbacks.AddCallback(notifyhub.CallbackEventSent, notifyhub.NewLoggingCallback("success-logger", nil))
//
//	options := notifyhub.NewAsyncOptions().WithCallbacks(callbacks)
//	taskID, err := hub.SendAsync(ctx, message, options)
package notifyhub

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/logger/adapters"
	"github.com/kart-io/notifyhub/notifiers"
	"github.com/kart-io/notifyhub/queue"
	"github.com/kart-io/notifyhub/queue/callbacks"
)

// ================================
// Core type aliases for backward compatibility
// ================================

type (
	// Hub is the main client for sending notifications
	Hub = client.Hub

	// Target represents a notification destination
	Target = notifiers.Target

	// TargetType defines the type of notification target
	TargetType = notifiers.TargetType

	// MessageFormat defines the format of message content
	MessageFormat = notifiers.MessageFormat

	// Message represents a notification message
	Message = notifiers.Message

	// SendResult represents the result of a send operation
	SendResult = notifiers.SendResult

	// Notifier interface that all notification platforms must implement
	Notifier = notifiers.Notifier

	// Options for sending messages
	Options = client.Options

	// MessageBuilder for fluent message construction
	MessageBuilder = client.MessageBuilder

	// Config holds NotifyHub configuration
	Config = config.Config

	// ConfigOption defines configuration options
	ConfigOption = config.Option

	// RoutingRule defines message routing rules
	RoutingRule = config.RoutingRule

	// CallbackOptions defines callback configuration
	CallbackOptions = queue.CallbackOptions

	// Callback interface for message lifecycle events
	Callback = queue.Callback

	// CallbackEvent represents callback event types
	CallbackEvent = queue.CallbackEvent

	// CallbackContext contains callback execution information
	CallbackContext = queue.CallbackContext

	// RetryPolicy defines retry behavior
	RetryPolicy = queue.RetryPolicy

	// Logger interface for logging
	Logger = logger.Interface

	// LogLevel defines log levels
	LogLevel = logger.LogLevel
)

// Constants for target types
const (
	TargetTypeGroup = notifiers.TargetTypeGroup
	TargetTypeUser  = notifiers.TargetTypeUser
	TargetTypeEmail = notifiers.TargetTypeEmail
)

// Constants for message formats
const (
	FormatText     = notifiers.FormatText
	FormatMarkdown = notifiers.FormatMarkdown
	FormatHTML     = notifiers.FormatHTML
)

// Constants for callback events
const (
	CallbackEventSent       = queue.CallbackEventSent
	CallbackEventFailed     = queue.CallbackEventFailed
	CallbackEventRetry      = queue.CallbackEventRetry
	CallbackEventMaxRetries = queue.CallbackEventMaxRetries
)

// Constants for log levels
const (
	LogLevelSilent = logger.Silent
	LogLevelError  = logger.Error
	LogLevelWarn   = logger.Warn
	LogLevelInfo   = logger.Info
	LogLevelDebug  = logger.Debug
)

// ================================
// Main constructor functions
// ================================

// New creates a new NotifyHub instance with configuration options
func New(opts ...ConfigOption) (*Hub, error) {
	return client.New(opts...)
}

// NewWithDefaults creates a NotifyHub with default configuration from environment
func NewWithDefaults() (*Hub, error) {
	return client.New(WithDefaults())
}

// ================================
// Message builders
// ================================

// NewMessage creates a new message builder
func NewMessage() *MessageBuilder {
	return client.NewMessage()
}

// NewAlert creates a new alert message builder
func NewAlert(title, body string) *MessageBuilder {
	return client.NewAlert(title, body)
}

// NewNotice creates a new notice message builder
func NewNotice(title, body string) *MessageBuilder {
	return client.NewNotice(title, body)
}

// NewReport creates a new report message builder
func NewReport(title, body string) *MessageBuilder {
	return client.NewReport(title, body)
}

// NewMarkdown creates a new markdown message builder
func NewMarkdown(title, body string) *MessageBuilder {
	return client.NewMarkdown(title, body)
}

// NewHTML creates a new HTML message builder
func NewHTML(title, body string) *MessageBuilder {
	return client.NewHTML(title, body)
}

// ================================
// Options builders
// ================================

// NewOptions creates new send options
func NewOptions() *Options {
	return client.NewOptions()
}

// NewAsyncOptions creates options for async sending
func NewAsyncOptions() *Options {
	return client.NewAsyncOptions()
}

// NewSyncOptions creates options for sync sending
func NewSyncOptions() *Options {
	return client.NewSyncOptions()
}

// NewRetryOptions creates options with retry enabled
func NewRetryOptions(maxRetries int) *Options {
	return client.NewRetryOptions(maxRetries)
}

// ================================
// Configuration functions
// ================================

// WithDefaults applies default configuration from environment variables
func WithDefaults() ConfigOption {
	return config.WithDefaults()
}

// WithTestDefaults applies test-friendly configuration
func WithTestDefaults() ConfigOption {
	return config.WithTestDefaults()
}

// WithFeishu configures Feishu webhook settings
func WithFeishu(webhookURL, secret string) ConfigOption {
	return config.WithFeishu(webhookURL, secret)
}

// WithFeishuFromEnv configures Feishu from environment variables
func WithFeishuFromEnv() ConfigOption {
	return config.WithFeishuFromEnv()
}

// WithEmail configures SMTP email settings
func WithEmail(host string, port int, username, password, from string) ConfigOption {
	return config.WithEmail(host, port, username, password, from)
}

// WithEmailFromEnv configures email from environment variables
func WithEmailFromEnv() ConfigOption {
	return config.WithEmailFromEnv()
}

// WithQueue configures queue settings
func WithQueue(queueType string, bufferSize, workers int) ConfigOption {
	return config.WithQueue(queueType, bufferSize, workers)
}

// WithQueueFromEnv configures queue from environment variables
func WithQueueFromEnv() ConfigOption {
	return config.WithQueueFromEnv()
}

// WithQueueRetryPolicy sets queue retry policy
func WithQueueRetryPolicy(policy *RetryPolicy) ConfigOption {
	return config.WithQueueRetryPolicy(policy)
}

// WithRouting configures routing rules
func WithRouting(rules ...RoutingRule) ConfigOption {
	return config.WithRouting(rules...)
}

// WithDefaultRouting configures default routing rules
func WithDefaultRouting() ConfigOption {
	return config.WithDefaultRouting()
}

// NewRoutingRule creates a routing rule builder
func NewRoutingRule(name string) *config.RoutingRuleBuilder {
	return config.NewRoutingRule(name)
}

// ================================
// Logger configuration functions
// ================================

// WithLogger configures a custom logger
func WithLogger(logger Logger) ConfigOption {
	return config.WithLogger(logger)
}

// WithDefaultLogger configures the default logger with specified level
func WithDefaultLogger(level LogLevel) ConfigOption {
	return config.WithDefaultLogger(level)
}

// WithSilentLogger disables logging
func WithSilentLogger() ConfigOption {
	return config.WithSilentLogger()
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger() Logger {
	return logger.Default
}

// NewStdLogAdapter creates a standard log adapter
func NewStdLogAdapter(stdLogger interface{}, level LogLevel) Logger {
	return adapters.NewStdLogAdapter(stdLogger.(adapters.StdLogger), level)
}

// NewLogrusAdapter creates a logrus adapter
func NewLogrusAdapter(logrusLogger interface{}, level LogLevel) Logger {
	return adapters.NewLogrusAdapter(logrusLogger.(adapters.LogrusLogger), level)
}

// NewZapAdapter creates a zap adapter
func NewZapAdapter(zapLogger interface{}, level LogLevel) Logger {
	return adapters.NewZapAdapter(zapLogger.(adapters.ZapLogger), level)
}

// NewFuncAdapter creates a function adapter
func NewFuncAdapter(logFunc func(level string, msg string, keyvals ...interface{}), level LogLevel) Logger {
	return adapters.NewFuncAdapter(logFunc, level)
}

// NewKartLoggerAdapter creates a Kart logger adapter for github.com/kart-io/logger
func NewKartLoggerAdapter(kartLogger interface{}, level LogLevel) Logger {
	return adapters.NewKartLoggerAdapter(kartLogger.(adapters.KartLogger), level)
}

// NewSimpleKartLoggerAdapter creates a simple Kart logger adapter
func NewSimpleKartLoggerAdapter(simpleKartLogger interface{}, level LogLevel) Logger {
	return adapters.NewSimpleKartLoggerAdapter(simpleKartLogger.(adapters.SimpleKartLogger), level)
}

// NewCustomAdapter creates a custom logger adapter
// The customLogger must implement the logger.CustomLogger interface
func NewCustomAdapter(customLogger interface{}, level LogLevel) Logger {
	return adapters.NewCustomAdapter(customLogger.(adapters.CustomLogger), level)
}

// ================================
// Callback functions
// ================================

// NewCallbackFunc creates a new callback function
func NewCallbackFunc(name string, fn func(ctx context.Context, callbackCtx *CallbackContext) error) *callbacks.CallbackFunc {
	return callbacks.NewCallbackFunc(name, fn)
}

// NewLoggingCallback creates a logging callback
func NewLoggingCallback(name string, logger func(format string, v ...interface{})) *callbacks.LoggingCallback {
	return callbacks.NewLoggingCallback(name, logger)
}

// ================================
// Retry policy functions
// ================================

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return queue.DefaultRetryPolicy()
}

// ExponentialBackoffPolicy creates exponential backoff retry policy
func ExponentialBackoffPolicy(maxRetries int, initialInterval time.Duration, multiplier float64) *RetryPolicy {
	return queue.ExponentialBackoffPolicy(maxRetries, initialInterval, multiplier)
}

// NoRetryPolicy creates a policy that disables retries
func NoRetryPolicy() *RetryPolicy {
	return queue.NoRetryPolicy()
}
