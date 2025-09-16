package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/monitoring"
	"github.com/kart-io/notifyhub/notifiers"
	"github.com/kart-io/notifyhub/observability"
	"github.com/kart-io/notifyhub/queue"
	"github.com/kart-io/notifyhub/template"
)

// Hub is the main client for sending notifications
type Hub struct {
	config    *config.Config
	notifiers map[string]notifiers.Notifier
	queue     queue.Queue
	templates *template.Engine
	routing   *config.RoutingEngine
	metrics   *monitoring.Metrics
	logger    logger.Interface
	telemetry *observability.TelemetryProvider

	mu      sync.RWMutex
	started bool
	stopCh  chan struct{}
	workers []chan struct{}
	debug   bool // Enable debug mode for detailed logging
}

// New creates a new NotifyHub instance with configuration options
func New(opts ...config.Option) (*Hub, error) {
	cfg := config.New(opts...)

	// Default queue configuration if not provided
	queueConfig := cfg.Queue()
	if queueConfig == nil {
		queueConfig = &config.QueueConfig{
			Type:        "memory",
			BufferSize:  1000,
			Workers:     2,
			RetryPolicy: queue.DefaultRetryPolicy(),
		}
	}

	hub := &Hub{
		config:    cfg,
		notifiers: make(map[string]notifiers.Notifier),
		queue:     queue.NewSimple(queueConfig.BufferSize),
		templates: template.NewEngine(),
		metrics:   monitoring.NewMetrics(),
		logger:    cfg.Logger(),
		stopCh:    make(chan struct{}),
	}

	// Initialize telemetry
	if telemetryConfig := cfg.Telemetry(); telemetryConfig != nil {
		telemetryProvider, err := observability.NewTelemetryProvider(telemetryConfig)
		if err != nil {
			hub.logger.Error(context.Background(), "Failed to initialize telemetry: %v", err)
		} else {
			hub.telemetry = telemetryProvider
			hub.logger.Info(context.Background(), "Telemetry provider initialized: enabled=%t, tracing=%t, metrics=%t",
				telemetryConfig.Enabled, telemetryConfig.TracingEnabled, telemetryConfig.MetricsEnabled)
		}
	} else {
		// Create disabled telemetry provider
		hub.telemetry, _ = observability.NewTelemetryProvider(nil)
	}

	hub.logger.Info(context.Background(), "NotifyHub initializing with config: queue_type=%s, buffer_size=%d, workers=%d",
		queueConfig.Type, queueConfig.BufferSize, queueConfig.Workers)

	// Initialize routing engine
	if routingConfig := cfg.Routing(); routingConfig != nil {
		hub.routing = config.NewRoutingEngine(routingConfig.Rules)
	} else {
		hub.routing = config.NewRoutingEngine([]config.RoutingRule{})
	}

	// Initialize Feishu notifier
	if feishuConfig := cfg.Feishu(); feishuConfig != nil {
		hub.notifiers["feishu"] = notifiers.NewFeishuNotifier(
			feishuConfig.WebhookURL,
			feishuConfig.Secret,
			feishuConfig.Timeout,
		)
		hub.logger.Info(context.Background(), "Feishu notifier initialized with webhook: %s",
			maskWebhookURL(feishuConfig.WebhookURL))
	}

	// Initialize Email notifier
	if emailConfig := cfg.Email(); emailConfig != nil && emailConfig.Host != "" && emailConfig.From != "" {
		hub.notifiers["email"] = notifiers.NewEmailNotifier(
			emailConfig.Host,
			emailConfig.Port,
			emailConfig.Username,
			emailConfig.Password,
			emailConfig.From,
			emailConfig.UseTLS,
			emailConfig.Timeout,
		)
		hub.logger.Info(context.Background(), "Email notifier initialized with SMTP: %s:%d, from: %s",
			emailConfig.Host, emailConfig.Port, emailConfig.From)
	}

	// Initialize Mock notifier (for testing)
	if mockConfig := cfg.MockNotifier(); mockConfig != nil {
		mockNotifier := notifiers.NewMockNotifier(mockConfig.Name)
		mockNotifier.WithDelay(mockConfig.Delay)
		if mockConfig.ShouldFail {
			mockNotifier.WithFailure()
		}
		if len(mockConfig.SupportedTargets) > 0 {
			targetTypes := make([]notifiers.TargetType, len(mockConfig.SupportedTargets))
			for i, target := range mockConfig.SupportedTargets {
				targetTypes[i] = notifiers.TargetType(target)
			}
			mockNotifier.WithSupportedTargets(targetTypes...)
		}
		hub.notifiers[mockConfig.Name] = mockNotifier
		hub.logger.Info(context.Background(), "Mock notifier initialized: name=%s, shouldFail=%t, delay=%v",
			mockConfig.Name, mockConfig.ShouldFail, mockConfig.Delay)
	}

	if len(hub.notifiers) == 0 {
		hub.logger.Error(context.Background(), "No notifiers configured")
		return nil, fmt.Errorf("no notifiers configured")
	}

	hub.logger.Info(context.Background(), "NotifyHub initialized successfully with %d notifiers: %v",
		len(hub.notifiers), getNotifierNames(hub.notifiers))

	return hub, nil
}

// Start starts the NotifyHub services
func (h *Hub) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		h.logger.Warn(ctx, "NotifyHub is already started")
		return fmt.Errorf("already started")
	}

	h.logger.Info(ctx, "Starting NotifyHub services...")

	// Start queue workers
	queueConfig := h.config.Queue()
	if queueConfig != nil {
		worker := queue.NewWorker(h.queue, h, queueConfig.RetryPolicy, queueConfig.Workers)
		if err := worker.Start(ctx); err != nil {
			h.logger.Error(ctx, "Failed to start queue worker: %v", err)
			return fmt.Errorf("start queue worker: %v", err)
		}
		h.logger.Info(ctx, "Queue workers started successfully: %d workers", queueConfig.Workers)
	}

	// Start health check routine
	go h.healthCheckRoutine(ctx)

	h.started = true
	h.logger.Info(ctx, "NotifyHub started successfully")
	return nil
}

// Stop stops the NotifyHub services
func (h *Hub) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.started {
		h.logger.Debug(context.Background(), "NotifyHub is not started, skip stopping")
		return nil
	}

	h.logger.Info(context.Background(), "Stopping NotifyHub services...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop all notifiers gracefully
	for name, notifier := range h.notifiers {
		h.logger.Debug(ctx, "Shutting down notifier: %s", name)
		if err := notifier.Shutdown(ctx); err != nil {
			h.logger.Error(ctx, "Failed to shutdown notifier %s: %v", name, err)
		} else {
			h.logger.Debug(ctx, "Notifier %s shutdown successfully", name)
		}
	}

	// Stop telemetry provider
	if h.telemetry != nil {
		h.logger.Debug(ctx, "Shutting down telemetry provider")
		if err := h.telemetry.Shutdown(ctx); err != nil {
			h.logger.Error(ctx, "Failed to shutdown telemetry provider: %v", err)
		} else {
			h.logger.Debug(ctx, "Telemetry provider shutdown successfully")
		}
	}

	// Stop the hub
	close(h.stopCh)
	h.started = false

	h.logger.Info(context.Background(), "NotifyHub stopped successfully")
	return nil
}

// Send sends a message synchronously or asynchronously
func (h *Hub) Send(ctx context.Context, message *notifiers.Message, options *Options) ([]*notifiers.SendResult, error) {
	h.debugLog(ctx, "Send called: message='%s', targets=%d, async=%v",
		message.Title, len(message.Targets), options != nil && options.Async)

	if options != nil && options.Async {
		h.debugLog(ctx, "Routing to async send for message: %s", message.ID)
		_, err := h.SendAsync(ctx, message, options)
		if err != nil {
			h.debugLog(ctx, "Async send failed: %v", err)
			return nil, err
		}
		h.debugLog(ctx, "Async send successful for message: %s", message.ID)
		return []*notifiers.SendResult{{Platform: "queue", Success: true, SentAt: time.Now()}}, nil
	}

	h.debugLog(ctx, "Routing to sync send for message: %s", message.ID)
	return h.SendSync(ctx, message, options)
}

// SendSync sends a message synchronously (implements queue.MessageSender)
func (h *Hub) SendSync(ctx context.Context, message *notifiers.Message, options interface{}) ([]*notifiers.SendResult, error) {
	start := time.Now()
	h.debugLog(ctx, "SendSync starting: message='%s' (ID: %s)", message.Title, message.ID)

	// Create telemetry span
	var span trace.Span
	if h.telemetry != nil {
		ctx, span = h.telemetry.TraceMessageSend(ctx, message.ID, "sync", len(message.Targets))
		defer span.End()
	}

	// Convert options to the correct type
	var opts *Options
	if options != nil {
		if typedOpts, ok := options.(*Options); ok {
			opts = typedOpts
		}
	}

	h.logger.Debug(ctx, "Starting synchronous message send: title='%s', priority=%d, targets=%d",
		message.Title, message.Priority, len(message.Targets))

	// Process through routing
	processedMessage := h.routing.ProcessMessage(message)

	// Render templates
	renderedMessage, err := h.templates.RenderMessage(processedMessage)
	if err != nil {
		h.logger.Error(ctx, "Failed to render template: %v", err)
		return nil, fmt.Errorf("render template: %v", err)
	}

	h.logger.Debug(ctx, "Message processed through routing and template rendering")

	// Send to all applicable notifiers
	var results []*notifiers.SendResult
	var errors []string

	for notifierName, notifier := range h.notifiers {
		// Check if this notifier should handle any targets
		hasTargets := len(renderedMessage.Targets) == 0
		for _, target := range renderedMessage.Targets {
			if notifier.SupportsTarget(target) {
				hasTargets = true
				break
			}
		}

		if !hasTargets {
			h.logger.Debug(ctx, "Notifier %s skipped - no matching targets", notifierName)
			continue
		}

		h.logger.Debug(ctx, "Sending message via %s notifier", notifierName)

		// Send with timeout
		sendCtx := ctx
		if opts != nil && opts.Timeout > 0 {
			var cancel context.CancelFunc
			sendCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
			defer cancel()
		}

		notifierStart := time.Now()
		notifierResults, err := notifier.Send(sendCtx, renderedMessage)
		notifierDuration := time.Since(notifierStart)

		if err != nil {
			h.logger.Error(ctx, "Notifier %s failed: %v (took %v)", notifierName, err, notifierDuration)
			errors = append(errors, fmt.Sprintf("%s: %v", notifier.Name(), err))
			h.metrics.RecordSend(notifier.Name(), false, 0, err.Error())
		} else {
			h.logger.Info(ctx, "Notifier %s succeeded: %d results (took %v)",
				notifierName, len(notifierResults), notifierDuration)
			for _, result := range notifierResults {
				h.metrics.RecordSend(result.Platform, result.Success, result.Duration, result.Error)
			}
			results = append(results, notifierResults...)
		}
	}

	// Final trace with total duration and results
	var traceErr error
	if len(errors) > 0 {
		traceErr = fmt.Errorf("partial failures: %v", errors)
	}

	h.logger.Trace(ctx, start, func() (string, int64) {
		operation := fmt.Sprintf("Send message '%s' to %d notifiers", message.Title, len(h.notifiers))
		return operation, int64(len(results))
	}, traceErr)

	// Record telemetry metrics and set span status
	if h.telemetry != nil {
		duration := time.Since(start)
		if len(errors) == 0 {
			h.telemetry.RecordMessageSent(ctx, "sync", duration)
			h.telemetry.SetSpanSuccess(span)
		} else {
			errorType := "partial_failure"
			if len(results) == 0 {
				errorType = "complete_failure"
			}
			h.telemetry.RecordMessageFailed(ctx, "sync", duration, errorType)
			if traceErr != nil {
				h.telemetry.SetSpanError(span, traceErr)
			}
		}
	}

	if len(results) == 0 && len(errors) > 0 {
		h.logger.Error(ctx, "All notifiers failed for message '%s'", message.Title)
		return nil, fmt.Errorf("all notifiers failed: %v", errors)
	}

	h.logger.Info(ctx, "Message send completed: %d successful results, %d errors", len(results), len(errors))
	return results, nil
}

// SendBatch sends multiple messages at once with optimized performance
func (h *Hub) SendBatch(ctx context.Context, messages []*notifiers.Message, options *Options) ([]*notifiers.SendResult, error) {
	if len(messages) == 0 {
		h.logger.Debug(ctx, "SendBatch called with empty message list")
		return []*notifiers.SendResult{}, nil
	}

	start := time.Now()
	h.logger.Info(ctx, "Starting batch send: %d messages", len(messages))

	// Handle async batch sending
	if options != nil && options.Async {
		var taskIDs []string
		for i, message := range messages {
			taskID, err := h.SendAsync(ctx, message, options)
			if err != nil {
				h.logger.Error(ctx, "Failed to enqueue batch message %d: %v", i, err)
				return nil, fmt.Errorf("batch enqueue failed at message %d: %v", i, err)
			}
			taskIDs = append(taskIDs, taskID)
		}

		// Return success indicators for async mode
		var results []*notifiers.SendResult
		for i, taskID := range taskIDs {
			results = append(results, &notifiers.SendResult{
				Platform: "queue",
				Success:  true,
				SentAt:   time.Now(),
				Target: notifiers.Target{
					Type:     notifiers.TargetTypeUser,
					Value:    fmt.Sprintf("batch_%d_%s", i, taskID),
					Platform: "queue",
				},
			})
		}

		h.logger.Info(ctx, "Batch async send completed: %d messages enqueued", len(messages))
		return results, nil
	}

	// Process synchronous batch sending
	var allResults []*notifiers.SendResult
	var allErrors []string
	successCount := 0

	for i, message := range messages {
		h.logger.Debug(ctx, "Processing batch message %d/%d: title='%s'", i+1, len(messages), message.Title)

		results, err := h.SendSync(ctx, message, options)
		if err != nil {
			errMsg := fmt.Sprintf("message %d failed: %v", i, err)
			allErrors = append(allErrors, errMsg)
			h.logger.Error(ctx, "Batch message %d failed: %v", i, err)

			// Add failure result for tracking
			allResults = append(allResults, &notifiers.SendResult{
				Platform: "batch",
				Success:  false,
				Error:    err.Error(),
				SentAt:   time.Now(),
				Target: notifiers.Target{
					Type:     notifiers.TargetTypeUser,
					Value:    fmt.Sprintf("batch_%d_%s", i, message.Title),
					Platform: "batch",
				},
			})
		} else {
			successCount++
			allResults = append(allResults, results...)
			h.logger.Debug(ctx, "Batch message %d succeeded: %d results", i, len(results))
		}
	}

	// Log final batch results with performance trace
	duration := time.Since(start)
	failureCount := len(messages) - successCount

	h.logger.Trace(ctx, start, func() (string, int64) {
		operation := fmt.Sprintf("SendBatch %d messages", len(messages))
		return operation, int64(len(allResults))
	}, nil)

	if len(allErrors) > 0 {
		h.logger.Warn(ctx, "Batch send completed with partial failures: %d successes, %d failures in %v",
			successCount, failureCount, duration)

		// If all messages failed, return error
		if successCount == 0 {
			h.logger.Error(ctx, "All batch messages failed")
			return allResults, fmt.Errorf("all batch messages failed: %v", allErrors)
		}

		// If partial failures, still return results but log warnings
		h.logger.Warn(ctx, "Partial batch failures: %v", allErrors)
	} else {
		h.logger.Info(ctx, "Batch send completed successfully: %d messages, %d results in %v",
			len(messages), len(allResults), duration)
	}

	return allResults, nil
}

// SendAsync sends a message asynchronously
func (h *Hub) SendAsync(ctx context.Context, message *notifiers.Message, options *Options) (string, error) {
	// Create telemetry span
	var span trace.Span
	if h.telemetry != nil {
		ctx, span = h.telemetry.TraceMessageEnqueue(ctx, message.ID, "async")
		defer span.End()
	}

	h.logger.Debug(ctx, "Enqueueing async message: title='%s', priority=%d", message.Title, message.Priority)

	// Convert options to queue message
	queueMsg := &queue.Message{
		Message: message,
		Options: options,
	}

	taskID, err := h.queue.Enqueue(ctx, queueMsg)

	// Record telemetry
	if h.telemetry != nil {
		if err != nil {
			h.telemetry.SetSpanError(span, err)
		} else {
			h.telemetry.RecordMessageEnqueued(ctx, "async")
			h.telemetry.SetSpanSuccess(span)
		}
	}

	if err != nil {
		h.logger.Error(ctx, "Failed to enqueue message '%s': %v", message.Title, err)
		return "", err
	}

	h.logger.Info(ctx, "Message '%s' enqueued successfully with task ID: %s", message.Title, taskID)
	return taskID, nil
}

// healthCheckRoutine runs periodic health checks
func (h *Hub) healthCheckRoutine(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopCh:
			return
		case <-ticker.C:
			for name, notifier := range h.notifiers {
				err := notifier.Health(ctx)
				h.metrics.RecordHealth(name, err == nil)
			}
		}
	}
}

// GetMetrics returns current metrics
func (h *Hub) GetMetrics() map[string]interface{} {
	return h.metrics.GetSnapshot()
}

// GetHealth returns health status
func (h *Hub) GetHealth(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    h.metrics.GetUptime().String(),
	}

	// Check all notifiers
	allHealthy := true
	for name, notifier := range h.notifiers {
		err := notifier.Health(ctx)
		healthy := err == nil
		if !healthy {
			allHealthy = false
		}
		h.metrics.RecordHealth(name, healthy)
	}

	if !allHealthy || h.metrics.GetSuccessRate() < 0.8 {
		health["status"] = "unhealthy"
	}

	return health
}

// GetLogger returns the logger instance for testing and debugging
func (h *Hub) GetLogger() logger.Interface {
	return h.logger
}

// Convenience send functions
func (h *Hub) SendText(ctx context.Context, title, body string, targets ...notifiers.Target) error {
	message := NewMessage().Title(title).Body(body)
	for _, target := range targets {
		message.Target(target)
	}
	_, err := h.Send(ctx, message.Build(), nil)
	return err
}

func (h *Hub) SendAlert(ctx context.Context, title, body string, targets ...notifiers.Target) error {
	message := NewAlert(title, body)
	for _, target := range targets {
		message.Target(target)
	}
	_, err := h.Send(ctx, message.Build(), &Options{Timeout: 30 * time.Second, Retry: true})
	return err
}

func (h *Hub) SendWithTemplate(ctx context.Context, templateName string, variables map[string]interface{}, targets ...notifiers.Target) error {
	message := NewMessage().Template(templateName)
	for k, v := range variables {
		message.Variable(k, v)
	}
	for _, target := range targets {
		message.Target(target)
	}
	_, err := h.Send(ctx, message.Build(), nil)
	return err
}

// SendTemplate is a simplified alias for SendWithTemplate
func (h *Hub) SendTemplate(ctx context.Context, templateName string, variables map[string]interface{}, targets ...notifiers.Target) error {
	return h.SendWithTemplate(ctx, templateName, variables, targets...)
}

// Template provides a more fluent API for template-based messages
func (h *Hub) Template(templateName string) *TemplateBuilder {
	return &TemplateBuilder{
		hub:          h,
		templateName: templateName,
		variables:    make(map[string]interface{}),
		targets:      make([]notifiers.Target, 0),
	}
}

// ================================
// Helper functions for logging
// ================================

// maskWebhookURL masks sensitive parts of webhook URLs for logging
func maskWebhookURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	// Show first 20 chars and last 10 chars
	return url[:20] + "***" + url[len(url)-10:]
}

// NewForTesting creates a NotifyHub instance configured for testing
func NewForTesting() (*Hub, error) {
	return New(config.WithTestDefaults())
}

// NewForTestingAndStart creates and starts a NotifyHub instance for testing
func NewForTestingAndStart(ctx context.Context) (*Hub, error) {
	return NewAndStart(ctx, config.WithTestDefaults())
}

// NewAndStart creates a NotifyHub instance and starts it immediately
func NewAndStart(ctx context.Context, opts ...config.Option) (*Hub, error) {
	hub, err := New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create hub: %v", err)
	}

	if err := hub.Start(ctx); err != nil {
		return nil, fmt.Errorf("start hub: %v", err)
	}

	return hub, nil
}

// NewWithDefaults creates a NotifyHub instance with sensible defaults
func NewWithDefaults() (*Hub, error) {
	return New(config.WithDefaults())
}

// NewWithDefaultsAndStart creates and starts a NotifyHub instance with defaults
func NewWithDefaultsAndStart(ctx context.Context) (*Hub, error) {
	return NewAndStart(ctx, config.WithDefaults())
}

// NewWithDebug creates a NotifyHub instance with debug mode enabled
// Debug mode automatically logs all operations with detailed information
func NewWithDebug(opts ...config.Option) (*Hub, error) {
	// Default to test configuration with debug logging if no options provided
	debugOpts := []config.Option{
		config.WithLogger(logger.Default.LogMode(logger.Debug)),
	}

	// If no options provided, use test defaults
	if len(opts) == 0 {
		debugOpts = append(debugOpts, config.WithTestDefaults())
	} else {
		debugOpts = append(debugOpts, opts...)
	}

	hub, err := New(debugOpts...)
	if err != nil {
		return nil, err
	}

	hub.debug = true
	hub.logger.Info(context.Background(), "🐛 Debug mode enabled - all operations will be logged")

	return hub, nil
}

// NewWithDebugAndStart creates and starts a NotifyHub instance with debug mode
func NewWithDebugAndStart(ctx context.Context, opts ...config.Option) (*Hub, error) {
	hub, err := NewWithDebug(opts...)
	if err != nil {
		return nil, fmt.Errorf("create debug hub: %v", err)
	}

	if err := hub.Start(ctx); err != nil {
		return nil, fmt.Errorf("start debug hub: %v", err)
	}

	return hub, nil
}

// ==========================================
// Debug and Inspection Methods
// ==========================================

// IsDebugEnabled returns whether debug mode is enabled for this hub
func (h *Hub) IsDebugEnabled() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.debug
}

// EnableDebug enables debug mode for this hub instance
func (h *Hub) EnableDebug() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.debug = true
	h.logger.Info(context.Background(), "🐛 Debug mode enabled for Hub")
}

// DisableDebug disables debug mode for this hub instance
func (h *Hub) DisableDebug() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.debug = false
	h.logger.Info(context.Background(), "🐛 Debug mode disabled for Hub")
}

// debugLog logs debug information if debug mode is enabled
func (h *Hub) debugLog(ctx context.Context, format string, args ...interface{}) {
	if h.debug {
		h.logger.Debug(ctx, "🐛 [DEBUG] "+format, args...)
	}
}

// DebugMessage creates a message builder with debug mode enabled
func (h *Hub) DebugMessage() *MessageBuilder {
	builder := NewMessage()
	if h.debug {
		builder.Debug()
	}
	return builder
}

// MustNew creates a NotifyHub instance and panics on error (for demo/testing)
func MustNew(opts ...config.Option) *Hub {
	hub, err := New(opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to create NotifyHub: %v", err))
	}
	return hub
}

// MustNewAndStart creates and starts a NotifyHub instance and panics on error
func MustNewAndStart(ctx context.Context, opts ...config.Option) *Hub {
	hub, err := NewAndStart(ctx, opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to create and start NotifyHub: %v", err))
	}
	return hub
}

// getNotifierNames returns a slice of notifier names
func getNotifierNames(notifiers map[string]notifiers.Notifier) []string {
	names := make([]string, 0, len(notifiers))
	for name := range notifiers {
		names = append(names, name)
	}
	return names
}

// ================================
// Fluent Template Builder
// ================================

// TemplateBuilder provides a fluent API for building template-based messages
type TemplateBuilder struct {
	hub          *Hub
	templateName string
	variables    map[string]interface{}
	targets      []notifiers.Target
}

// Variables sets multiple template variables
func (tb *TemplateBuilder) Variables(variables map[string]interface{}) *TemplateBuilder {
	for k, v := range variables {
		tb.variables[k] = v
	}
	return tb
}

// Variable sets a single template variable
func (tb *TemplateBuilder) Variable(key string, value interface{}) *TemplateBuilder {
	tb.variables[key] = value
	return tb
}

// To adds targets to the message
func (tb *TemplateBuilder) To(targets ...notifiers.Target) *TemplateBuilder {
	tb.targets = append(tb.targets, targets...)
	return tb
}

// SendTo sends the template message to specified targets
func (tb *TemplateBuilder) SendTo(ctx context.Context, targets ...notifiers.Target) error {
	allTargets := append(tb.targets, targets...)
	return tb.hub.SendTemplate(ctx, tb.templateName, tb.variables, allTargets...)
}

// ================================
// Batch Operation APIs
// ================================

// BatchBuilder provides a fluent API for building batch operations
type BatchBuilder struct {
	hub      *Hub
	messages []*notifiers.Message
	options  *Options // Integrated send options for the entire batch
}

// NewBatch creates a new batch builder
func (h *Hub) NewBatch() *BatchBuilder {
	return &BatchBuilder{
		hub:      h,
		messages: make([]*notifiers.Message, 0),
	}
}

// Alert adds an alert message to the batch
func (bb *BatchBuilder) Alert(title, body string) *BatchMessageBuilder {
	return &BatchMessageBuilder{
		batch:   bb,
		builder: NewAlert(title, body),
	}
}

// Notice adds a notice message to the batch
func (bb *BatchBuilder) Notice(title, body string) *BatchMessageBuilder {
	return &BatchMessageBuilder{
		batch:   bb,
		builder: NewNotice(title, body),
	}
}

// Text adds a text message to the batch
func (bb *BatchBuilder) Text(title, body string) *BatchMessageBuilder {
	return &BatchMessageBuilder{
		batch:   bb,
		builder: NewMessage().Title(title).Body(body),
	}
}

// Message adds a custom message to the batch
func (bb *BatchBuilder) Message(builder *MessageBuilder) *BatchBuilder {
	bb.messages = append(bb.messages, builder.Build())
	return bb
}

// SendAll sends all messages in the batch
func (bb *BatchBuilder) SendAll(ctx context.Context, options ...*Options) ([]*notifiers.SendResult, error) {
	var opts *Options
	if len(options) > 0 {
		opts = options[0]
	} else if bb.options != nil {
		opts = bb.options
	}
	return bb.hub.SendBatch(ctx, bb.messages, opts)
}

// Count returns the number of messages in the batch
func (bb *BatchBuilder) Count() int {
	return len(bb.messages)
}

// ================================
// BatchBuilder Send Options Integration
// ================================

// WithOptions sets send options for the entire batch
func (bb *BatchBuilder) WithOptions(options *Options) *BatchBuilder {
	bb.options = options
	return bb
}

// Async enables asynchronous sending for the batch
func (bb *BatchBuilder) Async(async bool) *BatchBuilder {
	if bb.options == nil {
		bb.options = NewOptions()
	}
	bb.options.Async = async
	return bb
}

// AsyncSend enables asynchronous sending (convenience method)
func (bb *BatchBuilder) AsyncSend() *BatchBuilder {
	return bb.Async(true)
}

// SyncSend enables synchronous sending (convenience method)
func (bb *BatchBuilder) SyncSend() *BatchBuilder {
	return bb.Async(false)
}

// WithRetry enables retry on failure for the batch
func (bb *BatchBuilder) WithRetry(retry bool) *BatchBuilder {
	if bb.options == nil {
		bb.options = NewOptions()
	}
	bb.options.Retry = retry
	return bb
}

// EnableRetry enables retry on failure (convenience method)
func (bb *BatchBuilder) EnableRetry() *BatchBuilder {
	return bb.WithRetry(true)
}

// WithMaxRetries sets maximum retry attempts for the batch
func (bb *BatchBuilder) WithMaxRetries(maxRetries int) *BatchBuilder {
	if bb.options == nil {
		bb.options = NewOptions()
	}
	bb.options.MaxRetries = maxRetries
	return bb
}

// WithTimeout sets operation timeout for the batch
func (bb *BatchBuilder) WithTimeout(timeout time.Duration) *BatchBuilder {
	if bb.options == nil {
		bb.options = NewOptions()
	}
	bb.options.Timeout = timeout
	return bb
}

// WithBatchSize sets batch size for processing
func (bb *BatchBuilder) WithBatchSize(batchSize int) *BatchBuilder {
	if bb.options == nil {
		bb.options = NewOptions()
	}
	bb.options.BatchSize = batchSize
	return bb
}

// AsQuickBatch applies quick send options (sync, no retry, 5s timeout)
func (bb *BatchBuilder) AsQuickBatch() *BatchBuilder {
	return bb.SyncSend().
		WithRetry(false).
		WithTimeout(5 * time.Second)
}

// AsReliableBatch applies reliable send options (sync, retry enabled, 30s timeout)
func (bb *BatchBuilder) AsReliableBatch() *BatchBuilder {
	return bb.SyncSend().
		EnableRetry().
		WithMaxRetries(3).
		WithTimeout(30 * time.Second)
}

// AsBackgroundBatch applies background send options (async, retry enabled)
func (bb *BatchBuilder) AsBackgroundBatch() *BatchBuilder {
	return bb.AsyncSend().
		EnableRetry().
		WithMaxRetries(5)
}

// SendAllWithAnalysis sends all messages and returns analysis
func (bb *BatchBuilder) SendAllWithAnalysis(ctx context.Context, options ...*Options) ([]*notifiers.SendResult, *ResultAnalyzer, error) {
	results, err := bb.SendAll(ctx, options...)
	if err != nil {
		return results, nil, err
	}
	analyzer := AnalyzeResults(results)
	return results, analyzer, nil
}

// BatchMessageBuilder allows configuring individual messages in a batch
type BatchMessageBuilder struct {
	batch   *BatchBuilder
	builder *MessageBuilder
}

// To adds targets to the message and returns to batch builder
func (bmb *BatchMessageBuilder) To(targets ...notifiers.Target) *BatchBuilder {
	for _, target := range targets {
		bmb.builder.Target(target)
	}
	bmb.batch.messages = append(bmb.batch.messages, bmb.builder.Build())
	return bmb.batch
}

// Email adds email targets and returns to batch builder
func (bmb *BatchMessageBuilder) Email(emails ...string) *BatchBuilder {
	for _, email := range emails {
		bmb.builder.Email(email)
	}
	bmb.batch.messages = append(bmb.batch.messages, bmb.builder.Build())
	return bmb.batch
}

// Configure allows custom configuration of the message
func (bmb *BatchMessageBuilder) Configure(fn func(*MessageBuilder) *MessageBuilder) *BatchBuilder {
	bmb.builder = fn(bmb.builder)
	bmb.batch.messages = append(bmb.batch.messages, bmb.builder.Build())
	return bmb.batch
}

// Convenience batch methods

// SendAlertBatch sends multiple alerts with the same targets
func (h *Hub) SendAlertBatch(ctx context.Context, alerts map[string]string, targets ...notifiers.Target) ([]*notifiers.SendResult, error) {
	batch := h.NewBatch()
	for title, body := range alerts {
		batch.Alert(title, body).To(targets...)
	}
	return batch.SendAll(ctx)
}

// SendTextBatch sends multiple text messages with the same targets
func (h *Hub) SendTextBatch(ctx context.Context, messages map[string]string, targets ...notifiers.Target) ([]*notifiers.SendResult, error) {
	batch := h.NewBatch()
	for title, body := range messages {
		batch.Text(title, body).To(targets...)
	}
	return batch.SendAll(ctx)
}

// SendBatchToEmails sends the same message to multiple email recipients
func (h *Hub) SendBatchToEmails(ctx context.Context, title, body string, emails ...string) error {
	targets := make([]notifiers.Target, len(emails))
	for i, email := range emails {
		targets[i] = notifiers.Target{Type: notifiers.TargetTypeEmail, Value: email}
	}
	return h.SendText(ctx, title, body, targets...)
}

// ================================
// Enhanced Send Methods with Result Analysis
// ================================

// SendWithAnalysis sends a message and returns both results and analysis
func (h *Hub) SendWithAnalysis(ctx context.Context, message *notifiers.Message, options *Options) ([]*notifiers.SendResult, *ResultAnalyzer, error) {
	results, err := h.Send(ctx, message, options)
	if err != nil {
		return results, nil, err
	}
	analyzer := AnalyzeResults(results)
	return results, analyzer, nil
}

// SendAndAnalyze is a convenience method that sends and logs analysis
func (h *Hub) SendAndAnalyze(ctx context.Context, message *notifiers.Message, options *Options) error {
	results, analyzer, err := h.SendWithAnalysis(ctx, message, options)
	if err != nil {
		return err
	}

	// Log analysis if debug mode is enabled
	if h.debug {
		h.logger.Info(ctx, "📊 Send Analysis:\n%s", analyzer.String())
	}

	// Return error if any sends failed
	if analyzer.HasFailures() {
		return fmt.Errorf("some sends failed: %v", analyzer.AllErrors())
	}

	return nil
}

// SendBatchWithAnalysis sends batch messages and returns results with analysis
func (h *Hub) SendBatchWithAnalysis(ctx context.Context, messages []*notifiers.Message, options *Options) ([]*notifiers.SendResult, *ResultAnalyzer, error) {
	results, err := h.SendBatch(ctx, messages, options)
	if err != nil {
		return results, nil, err
	}
	analyzer := AnalyzeResults(results)
	return results, analyzer, nil
}
