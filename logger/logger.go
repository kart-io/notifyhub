package logger

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

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

// Default logger implementation
type logger struct {
	Writer
	Config
	infoStr, warnStr, errStr, debugStr string
	traceStr, traceErrStr, traceWarnStr string
}

// NewLogger creates a new logger with default configuration
func NewLogger(writer Writer, config Config) Interface {
	var (
		infoStr      = "%s\n[info] "
		warnStr      = "%s\n[warn] "
		errStr       = "%s\n[error] "
		debugStr     = "%s\n[debug] "
		traceStr     = "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s\n[%.3fms] [rows:%v] %s"
	)

	if config.Colorful {
		infoStr = Green + "%s\n" + Reset + Green + "[info] " + Reset
		warnStr = BlueBold + "%s\n" + Reset + Magenta + "[warn] " + Reset
		errStr = Magenta + "%s\n" + Reset + Red + "[error] " + Reset
		debugStr = White + "%s\n" + Reset + Blue + "[debug] " + Reset
		traceStr = Green + "%s\n" + Reset + Yellow + "[%.3fms] " + BlueBold + "[rows:%v]" + Reset + " %s"
		traceWarnStr = Green + "%s " + Yellow + "%s\n" + Reset + RedBold + "[%.3fms] " + Yellow + "[rows:%v]" + Magenta + " %s" + Reset
		traceErrStr = RedBold + "%s " + MagentaBold + "%s\n" + Reset + Yellow + "[%.3fms] " + BlueBold + "[rows:%v]" + Reset + " %s"
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

// New creates a new logger with default writer and config
func New(writer Writer, config Config) Interface {
	return NewLogger(writer, config)
}

// Default creates a logger with default configuration
func Default() Interface {
	return New(log.New(os.Stdout, "\r\n", log.LstdFlags), Config{
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      Warn,
		Colorful:      true,
	})
}

// LogMode sets log level
func (l *logger) LogMode(level LogLevel) Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info logs info level messages
func (l *logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		l.Printf(l.infoStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

// Warn logs warning level messages
func (l *logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		l.Printf(l.warnStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

// Error logs error level messages
func (l *logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		l.Printf(l.errStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

// Debug logs debug level messages
func (l *logger) Debug(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Debug {
		l.Printf(l.debugStr+msg, append([]interface{}{fileWithLineNum()}, data...)...)
	}
}

// Trace logs operation trace with duration
func (l *logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= Error && (!errors.Is(err, ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceErrStr, fileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceErrStr, fileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW OPERATION >= %v", l.SlowThreshold)
		if rows == -1 {
			l.Printf(l.traceWarnStr, fileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceWarnStr, fileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.LogLevel == Info:
		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceStr, fileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceStr, fileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

// Common error types
var (
	ErrRecordNotFound = errors.New("record not found")
)

// fileWithLineNum returns caller file and line number
func fileWithLineNum() string {
	// This is simplified - in real implementation would use runtime.Caller
	// to get actual file and line number like GORM does
	return "notifyhub"
}

