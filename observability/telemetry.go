package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/kart-io/notifyhub/config"
)

// TelemetryProvider provides observability features
type TelemetryProvider struct {
	config        *config.TelemetryConfig
	tracer        trace.Tracer
	meter         metric.Meter
	traceProvider *sdktrace.TracerProvider

	// Metrics
	messagesSent     metric.Int64Counter
	messagesEnqueued metric.Int64Counter
	messagesFailed   metric.Int64Counter
	sendDuration     metric.Float64Histogram
	queueSize        metric.Int64UpDownCounter
}

// NewTelemetryProvider creates a new telemetry provider
func NewTelemetryProvider(cfg *config.TelemetryConfig) (*TelemetryProvider, error) {
	if cfg == nil {
		cfg = &config.TelemetryConfig{
			ServiceName:    "notifyhub",
			ServiceVersion: "1.2.0",
			Environment:    "development",
			OTLPEndpoint:   "http://localhost:4318",
			TracingEnabled: true,
			MetricsEnabled: true,
			SampleRate:     1.0,
			Enabled:        false,
		}
	}

	tp := &TelemetryProvider{
		config: cfg,
	}

	if !cfg.Enabled {
		// Return no-op provider
		tp.tracer = otel.Tracer("notifyhub")
		tp.meter = otel.Meter("notifyhub")
		return tp, nil
	}

	// Initialize tracing
	if cfg.TracingEnabled {
		if err := tp.initTracing(); err != nil {
			return nil, fmt.Errorf("init tracing: %v", err)
		}
	}

	// Initialize metrics
	if cfg.MetricsEnabled {
		if err := tp.initMetrics(); err != nil {
			return nil, fmt.Errorf("init metrics: %v", err)
		}
	}

	return tp, nil
}

// initTracing initializes OpenTelemetry tracing
func (tp *TelemetryProvider) initTracing() error {
	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(tp.config.ServiceName),
			semconv.ServiceVersion(tp.config.ServiceVersion),
			semconv.DeploymentEnvironment(tp.config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("create resource: %v", err)
	}

	// Create OTLP HTTP exporter
	exporter, err := otlptrace.New(context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(tp.config.OTLPEndpoint),
			otlptracehttp.WithHeaders(tp.config.OTLPHeaders),
		),
	)
	if err != nil {
		return fmt.Errorf("create exporter: %v", err)
	}

	// Create trace provider
	tp.traceProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(tp.config.SampleRate)),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp.traceProvider)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Get tracer
	tp.tracer = otel.Tracer("notifyhub",
		trace.WithInstrumentationVersion("1.1.0"),
		trace.WithSchemaURL(semconv.SchemaURL),
	)

	return nil
}

// initMetrics initializes OpenTelemetry metrics
func (tp *TelemetryProvider) initMetrics() error {
	// Get meter
	tp.meter = otel.Meter("notifyhub",
		metric.WithInstrumentationVersion("1.1.0"),
		metric.WithSchemaURL(semconv.SchemaURL),
	)

	var err error

	// Create counters
	tp.messagesSent, err = tp.meter.Int64Counter(
		"notifyhub_messages_sent_total",
		metric.WithDescription("Total number of messages sent"),
	)
	if err != nil {
		return fmt.Errorf("create messages_sent counter: %v", err)
	}

	tp.messagesEnqueued, err = tp.meter.Int64Counter(
		"notifyhub_messages_enqueued_total",
		metric.WithDescription("Total number of messages enqueued"),
	)
	if err != nil {
		return fmt.Errorf("create messages_enqueued counter: %v", err)
	}

	tp.messagesFailed, err = tp.meter.Int64Counter(
		"notifyhub_messages_failed_total",
		metric.WithDescription("Total number of messages failed"),
	)
	if err != nil {
		return fmt.Errorf("create messages_failed counter: %v", err)
	}

	// Create histograms
	tp.sendDuration, err = tp.meter.Float64Histogram(
		"notifyhub_send_duration_seconds",
		metric.WithDescription("Duration of message send operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("create send_duration histogram: %v", err)
	}

	// Create up/down counters
	tp.queueSize, err = tp.meter.Int64UpDownCounter(
		"notifyhub_queue_size",
		metric.WithDescription("Current queue size"),
	)
	if err != nil {
		return fmt.Errorf("create queue_size counter: %v", err)
	}

	return nil
}

// TraceOperation creates a new span for an operation
func (tp *TelemetryProvider) TraceOperation(ctx context.Context, operationName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	if tp.tracer == nil {
		// Return no-op span
		return ctx, trace.SpanFromContext(ctx)
	}

	return tp.tracer.Start(ctx, operationName,
		trace.WithAttributes(attributes...),
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

// TraceMessageSend creates a span for message sending
func (tp *TelemetryProvider) TraceMessageSend(ctx context.Context, messageID string, platform string, targets int) (context.Context, trace.Span) {
	attributes := []attribute.KeyValue{
		attribute.String("notifyhub.message.id", messageID),
		attribute.String("notifyhub.platform", platform),
		attribute.Int("notifyhub.targets.count", targets),
		attribute.String("notifyhub.operation", "send"),
	}

	return tp.TraceOperation(ctx, "notifyhub.send", attributes...)
}

// TraceMessageEnqueue creates a span for message enqueueing
func (tp *TelemetryProvider) TraceMessageEnqueue(ctx context.Context, messageID string, queueType string) (context.Context, trace.Span) {
	attributes := []attribute.KeyValue{
		attribute.String("notifyhub.message.id", messageID),
		attribute.String("notifyhub.queue.type", queueType),
		attribute.String("notifyhub.operation", "enqueue"),
	}

	return tp.TraceOperation(ctx, "notifyhub.enqueue", attributes...)
}

// RecordMessageSent records a successful message send
func (tp *TelemetryProvider) RecordMessageSent(ctx context.Context, platform string, duration time.Duration) {
	if tp.messagesSent != nil {
		tp.messagesSent.Add(ctx, 1, metric.WithAttributes(
			attribute.String("platform", platform),
			attribute.String("status", "success"),
		))
	}

	if tp.sendDuration != nil {
		tp.sendDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
			attribute.String("platform", platform),
			attribute.String("status", "success"),
		))
	}
}

// RecordMessageFailed records a failed message send
func (tp *TelemetryProvider) RecordMessageFailed(ctx context.Context, platform string, duration time.Duration, errorType string) {
	if tp.messagesFailed != nil {
		tp.messagesFailed.Add(ctx, 1, metric.WithAttributes(
			attribute.String("platform", platform),
			attribute.String("error_type", errorType),
		))
	}

	if tp.sendDuration != nil {
		tp.sendDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
			attribute.String("platform", platform),
			attribute.String("status", "error"),
		))
	}
}

// RecordMessageEnqueued records a message enqueue
func (tp *TelemetryProvider) RecordMessageEnqueued(ctx context.Context, queueType string) {
	if tp.messagesEnqueued != nil {
		tp.messagesEnqueued.Add(ctx, 1, metric.WithAttributes(
			attribute.String("queue_type", queueType),
		))
	}
}

// UpdateQueueSize updates the current queue size
func (tp *TelemetryProvider) UpdateQueueSize(ctx context.Context, queueType string, size int64) {
	if tp.queueSize != nil {
		tp.queueSize.Add(ctx, size, metric.WithAttributes(
			attribute.String("queue_type", queueType),
		))
	}
}

// SetSpanError sets an error on the current span
func (tp *TelemetryProvider) SetSpanError(span trace.Span, err error) {
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanSuccess marks the span as successful
func (tp *TelemetryProvider) SetSpanSuccess(span trace.Span) {
	if span != nil {
		span.SetStatus(codes.Ok, "")
	}
}

// Shutdown gracefully shuts down the telemetry provider
func (tp *TelemetryProvider) Shutdown(ctx context.Context) error {
	if tp.traceProvider != nil {
		return tp.traceProvider.Shutdown(ctx)
	}
	return nil
}

// GetTracer returns the tracer instance
func (tp *TelemetryProvider) GetTracer() trace.Tracer {
	return tp.tracer
}

// GetMeter returns the meter instance
func (tp *TelemetryProvider) GetMeter() metric.Meter {
	return tp.meter
}
