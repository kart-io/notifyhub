package logger

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// logger implements the Interface (like GORM's default logger)
type logger struct {
	Writer
	Config
	infoStr, warnStr, errStr, debugStr  string
	traceStr, traceWarnStr, traceErrStr string
}

// New creates a new logger instance (like GORM's New)
func New(writer Writer, config Config) Interface {
	var (
		infoStr      = "%s\n[info] "
		warnStr      = "%s\n[warn] "
		errStr       = "%s\n[error] "
		debugStr     = "%s\n[debug] "
		traceStr     = "%s\n[%.3fms] [targets:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [targets:%v] %s"
		traceErrStr  = "%s %s\n[%.3fms] [targets:%v] %s"
	)

	if config.Colorful {
		infoStr = Green + "%s\n" + Reset + Green + "[info] " + Reset
		warnStr = BlueBold + "%s\n" + Reset + Magenta + "[warn] " + Reset
		errStr = Magenta + "%s\n" + Reset + Red + "[error] " + Reset
		debugStr = White + "%s\n" + Reset + Blue + "[debug] " + Reset
		traceStr = Green + "%s\n" + Reset + Yellow + "[%.3fms] " + BlueBold + "[targets:%v]" + Reset + " %s"
		traceWarnStr = Green + "%s " + Yellow + "%s\n" + Reset + RedBold + "[%.3fms] " + Yellow + "[targets:%v]" + Magenta + " %s" + Reset
		traceErrStr = RedBold + "%s " + MagentaBold + "%s\n" + Reset + Yellow + "[%.3fms] " + BlueBold + "[targets:%v]" + Reset + " %s"
	}

	return &logger{
		Writer:       writer,
		Config:       config,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		debugStr:     debugStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
}

// LogMode creates a new logger with specified log level (like GORM's LogMode)
func (l *logger) LogMode(level LogLevel) Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		l.Printf(l.infoStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

func (l *logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		l.Printf(l.warnStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

func (l *logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		l.Printf(l.errStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

func (l *logger) Debug(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Debug {
		l.Printf(l.debugStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

// Trace logs message sending operations with duration (like GORM's SQL trace)
// Now supports flexible key-value pairs like other log methods for more consistent interface
func (l *logger) Trace(ctx context.Context, begin time.Time, fc func() (operation string, targets int64), err error, data ...interface{}) {
	if l.LogLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)

	// Prepare additional context data if provided
	var contextStr string
	if len(data) > 0 {
		contextStr = " " + fmt.Sprintf("%v", data...)
	}

	switch {
	case err != nil && l.LogLevel >= Error:
		operation, targets := fc()
		if targets == -1 {
			l.Printf(l.traceErrStr, fileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", operation+contextStr)
		} else {
			l.Printf(l.traceErrStr, fileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, targets, operation+contextStr)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= Warn:
		operation, targets := fc()
		slowLog := fmt.Sprintf("SLOW OPERATION >= %v", l.SlowThreshold)
		if targets == -1 {
			l.Printf(l.traceWarnStr, fileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", operation+contextStr)
		} else {
			l.Printf(l.traceWarnStr, fileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, targets, operation+contextStr)
		}
	case l.LogLevel >= Info:
		operation, targets := fc()
		if targets == -1 {
			l.Printf(l.traceStr, fileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", operation+contextStr)
		} else {
			l.Printf(l.traceStr, fileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, targets, operation+contextStr)
		}
	}
}

// fileWithLineNum returns the file name and line number of the caller
// It skips the appropriate number of stack frames to get the actual caller
func fileWithLineNum() string {
	// Skip levels:
	// 0: runtime.Caller
	// 1: fileWithLineNum
	// 2: logger method (Info, Warn, Error, Debug, Trace)
	// 3: actual caller we want to identify
	for skip := 3; skip < 10; skip++ {
		_, file, line, ok := runtime.Caller(skip)
		if !ok {
			break
		}

		// Skip logger package itself and any testing files
		if filepath.Base(filepath.Dir(file)) == "logger" ||
			filepath.Base(file) == "testing.go" ||
			filepath.Ext(file) == "_test.go" {
			continue
		}

		// Return relative path from project root if possible
		if idx := findProjectRoot(file); idx != -1 {
			relativePath := file[idx:]
			return relativePath + ":" + strconv.Itoa(line)
		}

		// Fallback to just filename:line
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}

	// Fallback if we can't determine the caller
	return "unknown:0"
}

// findProjectRoot tries to find the project root in the file path
// Returns the index where the relative path should start, or -1 if not found
func findProjectRoot(file string) int {
	markers := []string{
		"/notifyhub/",
		"/kart-io/",
		"/src/",
	}

	for _, marker := range markers {
		if idx := findLastIndex(file, marker); idx != -1 {
			return idx + len(marker)
		}
	}
	return -1
}

// findLastIndex finds the last occurrence of substr in str
func findLastIndex(str, substr string) int {
	idx := -1
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			idx = i
		}
	}
	return idx
}

// NewStdLogger creates a logger that outputs to standard logger (convenience function)
func NewStdLogger(level LogLevel) Interface {
	return New(stdWriter{}, Config{
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      level,
		Colorful:      true,
	})
}

// stdWriter wraps Go's standard log package
type stdWriter struct{}

func (stdWriter) Printf(msg string, data ...interface{}) {
	log.Printf(msg, data...)
}

// UpdateDefault updates the default logger implementation
func init() {
	// Override the Default logger to use the new implementation
	Default = New(consoleWriter{}, Config{
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      Warn,
		Colorful:      true,
	})
}
