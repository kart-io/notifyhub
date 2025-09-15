package logger

import (
	"context"
	"fmt"
	"log"
	"time"
)

// logger implements the Interface (like GORM's default logger)
type defaultLogger struct {
	Writer
	Config
	infoStr, warnStr, errStr, debugStr string
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

	return &defaultLogger{
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
func (l *defaultLogger) LogMode(level LogLevel) Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *defaultLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		l.Printf(l.infoStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

func (l *defaultLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		l.Printf(l.warnStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

func (l *defaultLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		l.Printf(l.errStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

func (l *defaultLogger) Debug(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Debug {
		l.Printf(l.debugStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

// Trace logs message sending operations with duration (like GORM's SQL trace)
func (l *defaultLogger) Trace(ctx context.Context, begin time.Time, fc func() (operation string, targets int64), err error) {
	if l.LogLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= Error:
		operation, targets := fc()
		if targets == -1 {
			l.Printf(l.traceErrStr, fileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", operation)
		} else {
			l.Printf(l.traceErrStr, fileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, targets, operation)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= Warn:
		operation, targets := fc()
		slowLog := fmt.Sprintf("SLOW OPERATION >= %v", l.SlowThreshold)
		if targets == -1 {
			l.Printf(l.traceWarnStr, fileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", operation)
		} else {
			l.Printf(l.traceWarnStr, fileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, targets, operation)
		}
	case l.LogLevel >= Info:
		operation, targets := fc()
		if targets == -1 {
			l.Printf(l.traceStr, fileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", operation)
		} else {
			l.Printf(l.traceStr, fileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, targets, operation)
		}
	}
}

// fileWithLineNum returns the file name and line number of the caller
func fileWithLineNum() string {
	return "notifyhub"
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