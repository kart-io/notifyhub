package logger

import (
	"context"
	"fmt"
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
// This follows GORM's logger interface design pattern exactly
type Interface interface {
	// LogMode sets the log level and returns a new logger instance
	LogMode(level LogLevel) Interface

	// Info logs info messages
	Info(ctx context.Context, msg string, data ...interface{})

	// Warn logs warning messages
	Warn(ctx context.Context, msg string, data ...interface{})

	// Error logs error messages
	Error(ctx context.Context, msg string, data ...interface{})

	// Debug logs debug messages
	Debug(ctx context.Context, msg string, data ...interface{})

	// Trace logs message sending operations with duration (like GORM's SQL trace)
	// Now supports flexible key-value pairs like other log methods for more consistent interface
	Trace(ctx context.Context, begin time.Time, fc func() (operation string, targets int64), err error, data ...interface{})
}

// Writer defines the interface for log output (like GORM's Writer)
type Writer interface {
	Printf(string, ...interface{})
}

// Config defines logger configuration (like GORM's Config)
type Config struct {
	SlowThreshold time.Duration // Slow operation threshold (like GORM's slow SQL)
	LogLevel      LogLevel      // Log level
	Colorful      bool          // Enable color output
}

// Colors for console output
const (
	Reset       = "\033[0m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	White       = "\033[37m"
	BlueBold    = "\033[34;1m"
	MagentaBold = "\033[35;1m"
	RedBold     = "\033[31;1m"
	YellowBold  = "\033[33;1m"
)

// Predefined loggers (like GORM's Default and Discard)
var (
	// Discard discards all log messages
	Discard Interface = New(discardWriter{}, Config{LogLevel: Silent})

	// Default provides a default logger with standard output
	Default Interface = New(consoleWriter{}, Config{
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      Warn,
		Colorful:      true,
	})
)

// discardWriter discards all output
type discardWriter struct{}

func (discardWriter) Printf(string, ...interface{}) {}

// consoleWriter writes to console
type consoleWriter struct{}

func (consoleWriter) Printf(msg string, data ...interface{}) {
	fmt.Printf(msg, data...)
}
