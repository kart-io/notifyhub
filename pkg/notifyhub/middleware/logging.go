// Package middleware provides logging middleware for NotifyHub
package middleware

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// LoggingMiddleware provides comprehensive logging for message operations
type LoggingMiddleware struct {
	BaseMiddleware
	logger    logger.Logger
	logLevel  LogLevel
	logBodies bool
}

// LogLevel defines the logging level for the middleware
type LogLevel int

const (
	LogLevelNone LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// LoggingConfig represents configuration for logging middleware
type LoggingConfig struct {
	Logger    logger.Logger
	LogLevel  LogLevel
	LogBodies bool // Whether to log message bodies (be careful with sensitive data)
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(config LoggingConfig) *LoggingMiddleware {
	if config.Logger == nil {
		config.Logger = logger.Discard
	}

	return &LoggingMiddleware{
		BaseMiddleware: NewBaseMiddleware("logging"),
		logger:         config.Logger,
		logLevel:       config.LogLevel,
		logBodies:      config.LogBodies,
	}
}

// HandleSend implements the Middleware interface with comprehensive logging
func (lm *LoggingMiddleware) HandleSend(ctx context.Context, msg *message.Message, targets []target.Target, next SendHandler) (*receipt.Receipt, error) {
	startTime := time.Now()

	// Extract trace ID if available
	traceID, _ := GetTraceID(ctx)

	// Log request start
	lm.logRequest(msg, targets, traceID)

	// Set start time in context for other middleware
	ctx = SetStartTime(ctx)

	// Execute the next handler
	receipt, err := next(ctx, msg, targets)

	// Calculate duration
	duration := time.Since(startTime)

	// Log response
	lm.logResponse(msg, receipt, err, duration, traceID)

	return receipt, err
}

// logRequest logs the incoming request details
func (lm *LoggingMiddleware) logRequest(msg *message.Message, targets []target.Target, traceID string) {
	if lm.logLevel < LogLevelInfo {
		return
	}

	fields := []interface{}{
		"message_id", msg.ID,
		"title", msg.Title,
		"format", msg.Format,
		"priority", msg.Priority,
		"target_count", len(targets),
	}

	if traceID != "" {
		fields = append(fields, "trace_id", traceID)
	}

	if lm.logBodies && msg.Body != "" {
		fields = append(fields, "body", msg.Body)
	}

	if lm.logLevel >= LogLevelDebug {
		// Add target details for debug logging
		targetTypes := make(map[string]int)
		for _, tgt := range targets {
			targetTypes[tgt.Type]++
		}
		fields = append(fields, "target_types", targetTypes)

		if len(msg.PlatformData) > 0 {
			fields = append(fields, "platform_data_keys", getMapKeys(msg.PlatformData))
		}
	}

	lm.logger.Info("Message send started", fields...)
}

// logResponse logs the response details
func (lm *LoggingMiddleware) logResponse(msg *message.Message, receipt *receipt.Receipt, err error, duration time.Duration, traceID string) {
	fields := []interface{}{
		"message_id", msg.ID,
		"duration", duration,
	}

	if traceID != "" {
		fields = append(fields, "trace_id", traceID)
	}

	if err != nil {
		if lm.logLevel >= LogLevelError {
			fields = append(fields, "error", err.Error())
			lm.logger.Error("Message send failed", fields...)
		}
		return
	}

	if receipt != nil {
		fields = append(fields,
			"status", receipt.Status,
			"successful", receipt.Successful,
			"failed", receipt.Failed,
			"total", receipt.Total,
		)

		if lm.logLevel >= LogLevelDebug && len(receipt.Results) > 0 {
			// Add platform-specific results for debug logging
			platformResults := make(map[string]int)
			platformErrors := make(map[string][]string)

			for _, result := range receipt.Results {
				if result.Success {
					platformResults[result.Platform+"_success"]++
				} else {
					platformResults[result.Platform+"_failed"]++
					if result.Error != "" {
						platformErrors[result.Platform] = append(platformErrors[result.Platform], result.Error)
					}
				}
			}

			fields = append(fields, "platform_results", platformResults)
			if len(platformErrors) > 0 {
				fields = append(fields, "platform_errors", platformErrors)
			}
		}

		// Choose appropriate log level based on result status
		if receipt.Failed > 0 && receipt.Successful == 0 {
			if lm.logLevel >= LogLevelError {
				lm.logger.Error("Message send completely failed", fields...)
			}
		} else if receipt.Failed > 0 {
			if lm.logLevel >= LogLevelWarn {
				lm.logger.Warn("Message send partially failed", fields...)
			}
		} else {
			if lm.logLevel >= LogLevelInfo {
				lm.logger.Info("Message send successful", fields...)
			}
		}
	} else {
		if lm.logLevel >= LogLevelWarn {
			fields = append(fields, "warning", "no receipt returned")
			lm.logger.Warn("Message send completed without receipt", fields...)
		}
	}
}

// getMapKeys returns the keys from a map for logging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// AccessLogMiddleware provides Apache/Nginx style access logging
type AccessLogMiddleware struct {
	BaseMiddleware
	logger logger.Logger
}

// NewAccessLogMiddleware creates a new access log middleware
func NewAccessLogMiddleware(l logger.Logger) *AccessLogMiddleware {
	if l == nil {
		l = logger.Discard
	}

	return &AccessLogMiddleware{
		BaseMiddleware: NewBaseMiddleware("access_log"),
		logger:         l,
	}
}

// HandleSend implements the Middleware interface with access-style logging
func (alm *AccessLogMiddleware) HandleSend(ctx context.Context, msg *message.Message, targets []target.Target, next SendHandler) (*receipt.Receipt, error) {
	startTime := time.Now()
	receipt, err := next(ctx, msg, targets)
	duration := time.Since(startTime)

	// Format: [timestamp] message_id targets status duration error
	timestamp := startTime.Format("2006/01/02 15:04:05")
	status := "UNKNOWN"
	errorStr := "-"

	if err != nil {
		status = "ERROR"
		errorStr = err.Error()
	} else if receipt != nil {
		status = receipt.Status
	}

	traceID := "-"
	if tid, ok := GetTraceID(ctx); ok {
		traceID = tid
	}

	alm.logger.Info("ACCESS",
		"timestamp", timestamp,
		"message_id", msg.ID,
		"targets", len(targets),
		"status", status,
		"duration_ms", duration.Milliseconds(),
		"error", errorStr,
		"trace_id", traceID,
	)

	return receipt, err
}
