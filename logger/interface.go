package logger

import (
	"context"
	"time"
)

// LogLevel defines log levels
type LogLevel int

const (
	// Silent disables all logging
	Silent LogLevel = iota + 1
	// Error logs only errors
	Error
	// Warn logs warnings and errors
	Warn
	// Info logs info, warnings and errors
	Info
	// Debug logs all messages
	Debug
)

// String returns the string representation of log level
func (l LogLevel) String() string {
	switch l {
	case Silent:
		return "silent"
	case Error:
		return "error"
	case Warn:
		return "warn"
	case Info:
		return "info"
	case Debug:
		return "debug"
	default:
		return "unknown"
	}
}

// Interface defines the logger interface that NotifyHub uses
// This is similar to GORM's logger interface design
type Interface interface {
	// LogMode sets the log level
	LogMode(level LogLevel) Interface

	// Info logs info messages
	Info(ctx context.Context, msg string, data ...interface{})

	// Warn logs warning messages
	Warn(ctx context.Context, msg string, data ...interface{})

	// Error logs error messages
	Error(ctx context.Context, msg string, data ...interface{})

	// Debug logs debug messages
	Debug(ctx context.Context, msg string, data ...interface{})

	// Trace logs message sending operations with duration
	Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error)
}

// Writer defines the interface for log output
type Writer interface {
	Printf(string, ...interface{})
}

// Config defines logger configuration
type Config struct {
	SlowThreshold             time.Duration // Slow operation threshold
	LogLevel                  LogLevel      // Log level
	IgnoreRecordNotFoundError bool          // Ignore ErrRecordNotFound error for logger
	Colorful                  bool          // Disable color
}