// Package logger provides a GORM-style logging interface for NotifyHub.
// This logger is designed to be used across all platforms and components
// and supports pluggable external logging libraries like zap, logrus, slog.
package logger

import (
	"fmt"
	"log"
	"os"
)

// LogLevel represents the severity level of a log message.
type LogLevel int

const (
	// Silent suppresses all log output.
	Silent LogLevel = iota + 1
	// Error only logs error messages.
	Error
	// Warn logs warnings and errors.
	Warn
	// Info logs informational messages, warnings, and errors.
	Info
	// Debug logs all messages including debug information.
	Debug
)

// Logger is the interface that wraps the basic logging methods.
// This interface is inspired by GORM's logger design and adapted for slog-style structured logging.
type Logger interface {
	// LogMode sets the log level and returns a new logger instance.
	LogMode(level LogLevel) Logger
	// Info logs an informational message with structured key-value pairs.
	Info(msg string, args ...any)
	// Warn logs a warning message with structured key-value pairs.
	Warn(msg string, args ...any)
	// Error logs an error message with structured key-value pairs.
	Error(msg string, args ...any)
	// Debug logs a debug message with structured key-value pairs.
	Debug(msg string, args ...any)
}

// StandardLogger is the default implementation of the Logger interface, using the standard log package.
type StandardLogger struct {
	logger *log.Logger
	level  LogLevel
	prefix string
}

// NewStandardLogger creates a new logger with the given writer and configuration.
func NewStandardLogger(writer *log.Logger, level LogLevel, prefix string) Logger {
	return &StandardLogger{
		logger: writer,
		level:  level,
		prefix: prefix,
	}
}

// LogMode sets the log level and returns a new logger instance.
func (l *StandardLogger) LogMode(level LogLevel) Logger {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

// Info logs an informational message.
func (l *StandardLogger) Info(msg string, args ...any) {
	if l.level >= Info {
		l.logger.Print(l.formatLog("INFO", msg, args...))
	}
}

// Warn logs a warning message.
func (l *StandardLogger) Warn(msg string, args ...any) {
	if l.level >= Warn {
		l.logger.Print(l.formatLog("WARN", msg, args...))
	}
}

// Error logs an error message.
func (l *StandardLogger) Error(msg string, args ...any) {
	if l.level >= Error {
		l.logger.Print(l.formatLog("ERROR", msg, args...))
	}
}

// Debug logs a debug message.
func (l *StandardLogger) Debug(msg string, args ...any) {
	if l.level >= Debug {
		l.logger.Print(l.formatLog("DEBUG", msg, args...))
	}
}

func (l *StandardLogger) formatLog(level, msg string, args ...any) string {
	formattedMsg := fmt.Sprintf("%s [%s] %s", l.prefix, level, msg)
	if len(args) > 0 {
		// Simple key-value pair formatting for standard logger
		fieldsStr := ""
		for i := 0; i < len(args); i += 2 {
			key := args[i]
			var val any = "(no value)"
			if i+1 < len(args) {
				val = args[i+1]
			}
			fieldsStr += fmt.Sprintf(" %v=%v", key, val)
		}
		return formattedMsg + fieldsStr
	}
	return formattedMsg
}

// discardLogger is a logger that discards all output.
type discardLogger struct{}

// LogMode returns the discard logger itself.
func (d *discardLogger) LogMode(LogLevel) Logger { return d }

// Info does nothing.
func (d *discardLogger) Info(string, ...any) {}

// Warn does nothing.
func (d *discardLogger) Warn(string, ...any) {}

// Error does nothing.
func (d *discardLogger) Error(string, ...any) {}

// Debug does nothing.
func (d *discardLogger) Debug(string, ...any) {}

// Discard is a logger that discards all output.
var Discard Logger = &discardLogger{}

// New returns a default logger that writes to stdout.
func New() Logger {
	return NewStandardLogger(log.New(os.Stdout, "", log.LstdFlags), Warn, "[notifyhub]")
}
