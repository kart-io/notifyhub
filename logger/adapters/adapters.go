// Package adapters provides logger adapters for integrating various logging libraries with NotifyHub
package adapters

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/logger"
)

// ================================
// Base Adapter
// ================================

// AdapterBase provides common functionality for logger adapters
type AdapterBase struct {
	level logger.LogLevel
}

// NewAdapterBase creates a new adapter base
func NewAdapterBase(level logger.LogLevel) *AdapterBase {
	return &AdapterBase{level: level}
}

// ShouldLog checks if the message should be logged at the given level
func (a *AdapterBase) ShouldLog(level logger.LogLevel) bool {
	return a.level >= level
}

// GetLevel returns the current log level
func (a *AdapterBase) GetLevel() logger.LogLevel {
	return a.level
}

// SetLevel sets the log level
func (a *AdapterBase) SetLevel(level logger.LogLevel) {
	a.level = level
}

// ================================
// Custom Adapter Framework
// ================================

// CustomLogger defines a minimal interface for custom logger implementations
type CustomLogger interface {
	// Log is the main logging method that custom loggers must implement
	Log(level logger.LogLevel, msg string, fields map[string]interface{})
}

// CustomAdapter adapts any custom logger that implements CustomLogger interface
type CustomAdapter struct {
	*AdapterBase
	logger CustomLogger
}

// NewCustomAdapter creates a new custom adapter
func NewCustomAdapter(customLogger CustomLogger, level logger.LogLevel) logger.Interface {
	return &CustomAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      customLogger,
	}
}

func (c *CustomAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return &CustomAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      c.logger,
	}
}

func (c *CustomAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if c.ShouldLog(logger.Info) {
		fields := c.parseFields(data...)
		c.logger.Log(logger.Info, msg, fields)
	}
}

func (c *CustomAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if c.ShouldLog(logger.Warn) {
		fields := c.parseFields(data...)
		c.logger.Log(logger.Warn, msg, fields)
	}
}

func (c *CustomAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if c.ShouldLog(logger.Error) {
		fields := c.parseFields(data...)
		c.logger.Log(logger.Error, msg, fields)
	}
}

func (c *CustomAdapter) Debug(ctx context.Context, msg string, data ...interface{}) {
	if c.ShouldLog(logger.Debug) {
		fields := c.parseFields(data...)
		c.logger.Log(logger.Debug, msg, fields)
	}
}

func (c *CustomAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if c.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	operation, affected := fc()

	fields := map[string]interface{}{
		"operation":   operation,
		"duration_ms": float64(elapsed.Nanoseconds()) / 1e6,
		"affected":    affected,
	}

	if err != nil {
		fields["error"] = err.Error()
		if c.ShouldLog(logger.Error) {
			c.logger.Log(logger.Error, "Operation failed", fields)
		}
	} else {
		if c.ShouldLog(logger.Info) {
			c.logger.Log(logger.Info, "Operation completed", fields)
		}
	}
}

// parseFields converts variadic arguments to a map
func (c *CustomAdapter) parseFields(data ...interface{}) map[string]interface{} {
	fields := make(map[string]interface{})

	for i := 0; i < len(data)-1; i += 2 {
		if key, ok := data[i].(string); ok && i+1 < len(data) {
			fields[key] = data[i+1]
		}
	}

	return fields
}

// ================================
// Function-based Custom Adapter
// ================================

// LogFunc defines a function signature for simple logging functions
type LogFunc func(level string, msg string, keyvals ...interface{})

// FuncAdapter adapts a function to NotifyHub logger interface
type FuncAdapter struct {
	*AdapterBase
	logFunc LogFunc
}

// NewFuncAdapter creates a new function adapter
func NewFuncAdapter(logFunc LogFunc, level logger.LogLevel) logger.Interface {
	return &FuncAdapter{
		AdapterBase: NewAdapterBase(level),
		logFunc:     logFunc,
	}
}

func (f *FuncAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return &FuncAdapter{
		AdapterBase: NewAdapterBase(level),
		logFunc:     f.logFunc,
	}
}

func (f *FuncAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if f.ShouldLog(logger.Info) {
		f.logFunc("info", msg, data...)
	}
}

func (f *FuncAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if f.ShouldLog(logger.Warn) {
		f.logFunc("warn", msg, data...)
	}
}

func (f *FuncAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if f.ShouldLog(logger.Error) {
		f.logFunc("error", msg, data...)
	}
}

func (f *FuncAdapter) Debug(ctx context.Context, msg string, data ...interface{}) {
	if f.ShouldLog(logger.Debug) {
		f.logFunc("debug", msg, data...)
	}
}

func (f *FuncAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if f.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	operation, affected := fc()

	if err != nil && f.ShouldLog(logger.Error) {
		f.logFunc("error", "Operation failed",
			"operation", operation,
			"duration_ms", float64(elapsed.Nanoseconds())/1e6,
			"affected", affected,
			"error", err.Error())
	} else if f.ShouldLog(logger.Info) {
		f.logFunc("info", "Operation completed",
			"operation", operation,
			"duration_ms", float64(elapsed.Nanoseconds())/1e6,
			"affected", affected)
	}
}

// ================================
// Standard log adapter
// ================================

// StdLogger interface for standard Go log package
type StdLogger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

// StdLogAdapter adapts standard log package to NotifyHub logger interface
type StdLogAdapter struct {
	*AdapterBase
	logger StdLogger
}

// NewStdLogAdapter creates a new standard log adapter
func NewStdLogAdapter(stdLogger StdLogger, level logger.LogLevel) logger.Interface {
	return &StdLogAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      stdLogger,
	}
}

func (s *StdLogAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return &StdLogAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      s.logger,
	}
}

func (s *StdLogAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if s.ShouldLog(logger.Info) {
		if len(data) > 0 {
			s.logger.Printf("[INFO] "+msg, data...)
		} else {
			s.logger.Printf("[INFO] " + msg)
		}
	}
}

func (s *StdLogAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if s.ShouldLog(logger.Warn) {
		if len(data) > 0 {
			s.logger.Printf("[WARN] "+msg, data...)
		} else {
			s.logger.Printf("[WARN] " + msg)
		}
	}
}

func (s *StdLogAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if s.ShouldLog(logger.Error) {
		if len(data) > 0 {
			s.logger.Printf("[ERROR] "+msg, data...)
		} else {
			s.logger.Printf("[ERROR] " + msg)
		}
	}
}

func (s *StdLogAdapter) Debug(ctx context.Context, msg string, data ...interface{}) {
	if s.ShouldLog(logger.Debug) {
		if len(data) > 0 {
			s.logger.Printf("[DEBUG] "+msg, data...)
		} else {
			s.logger.Printf("[DEBUG] " + msg)
		}
	}
}

func (s *StdLogAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if s.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	operation, affected := fc()

	if err != nil && s.ShouldLog(logger.Error) {
		s.logger.Printf("[ERROR] Operation failed: %s, Duration: %.3fms, Affected: %d, Error: %v",
			operation, float64(elapsed.Nanoseconds())/1e6, affected, err)
	} else if s.ShouldLog(logger.Info) {
		s.logger.Printf("[INFO] Operation: %s, Duration: %.3fms, Affected: %d",
			operation, float64(elapsed.Nanoseconds())/1e6, affected)
	}
}

// ================================
// Logrus adapter
// ================================

// LogrusLogger interface compatible with logrus
type LogrusLogger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// LogrusAdapter adapts logrus to NotifyHub logger interface
type LogrusAdapter struct {
	*AdapterBase
	logger LogrusLogger
}

// NewLogrusAdapter creates a new logrus adapter
func NewLogrusAdapter(logrusLogger LogrusLogger, level logger.LogLevel) logger.Interface {
	return &LogrusAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      logrusLogger,
	}
}

func (l *LogrusAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return &LogrusAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      l.logger,
	}
}

func (l *LogrusAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.ShouldLog(logger.Info) {
		if len(data) > 0 {
			l.logger.Infof(msg, data...)
		} else {
			l.logger.Info(msg)
		}
	}
}

func (l *LogrusAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.ShouldLog(logger.Warn) {
		if len(data) > 0 {
			l.logger.Warnf(msg, data...)
		} else {
			l.logger.Warn(msg)
		}
	}
}

func (l *LogrusAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.ShouldLog(logger.Error) {
		if len(data) > 0 {
			l.logger.Errorf(msg, data...)
		} else {
			l.logger.Error(msg)
		}
	}
}

func (l *LogrusAdapter) Debug(ctx context.Context, msg string, data ...interface{}) {
	if l.ShouldLog(logger.Debug) {
		if len(data) > 0 {
			l.logger.Debugf(msg, data...)
		} else {
			l.logger.Debug(msg)
		}
	}
}

func (l *LogrusAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	operation, affected := fc()

	if err != nil && l.ShouldLog(logger.Error) {
		l.logger.Errorf("Operation failed: %s, Duration: %.3fms, Affected: %d, Error: %v",
			operation, float64(elapsed.Nanoseconds())/1e6, affected, err)
	} else if l.ShouldLog(logger.Info) {
		l.logger.Infof("Operation: %s, Duration: %.3fms, Affected: %d",
			operation, float64(elapsed.Nanoseconds())/1e6, affected)
	}
}

// ================================
// Zap adapter
// ================================

// ZapLogger interface compatible with zap SugaredLogger
type ZapLogger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
}

// ZapAdapter adapts zap to NotifyHub logger interface
type ZapAdapter struct {
	*AdapterBase
	logger ZapLogger
}

// NewZapAdapter creates a new zap adapter
func NewZapAdapter(zapLogger ZapLogger, level logger.LogLevel) logger.Interface {
	return &ZapAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      zapLogger,
	}
}

func (z *ZapAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return &ZapAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      z.logger,
	}
}

func (z *ZapAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if z.ShouldLog(logger.Info) {
		if len(data) > 0 {
			z.logger.Infof(msg, data...)
		} else {
			z.logger.Info(msg)
		}
	}
}

func (z *ZapAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if z.ShouldLog(logger.Warn) {
		if len(data) > 0 {
			z.logger.Warnf(msg, data...)
		} else {
			z.logger.Warn(msg)
		}
	}
}

func (z *ZapAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if z.ShouldLog(logger.Error) {
		if len(data) > 0 {
			z.logger.Errorf(msg, data...)
		} else {
			z.logger.Error(msg)
		}
	}
}

func (z *ZapAdapter) Debug(ctx context.Context, msg string, data ...interface{}) {
	if z.ShouldLog(logger.Debug) {
		if len(data) > 0 {
			z.logger.Debugf(msg, data...)
		} else {
			z.logger.Debug(msg)
		}
	}
}

func (z *ZapAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if z.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	operation, affected := fc()

	if err != nil && z.ShouldLog(logger.Error) {
		z.logger.Errorf("Operation failed: %s, Duration: %.3fms, Affected: %d, Error: %v",
			operation, float64(elapsed.Nanoseconds())/1e6, affected, err)
	} else if z.ShouldLog(logger.Info) {
		z.logger.Infof("Operation: %s, Duration: %.3fms, Affected: %d",
			operation, float64(elapsed.Nanoseconds())/1e6, affected)
	}
}

// ================================
// Kart Logger adapter (github.com/kart-io/logger)
// ================================

// KartLogger interface compatible with github.com/kart-io/logger
type KartLogger interface {
	// 基础日志方法
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})

	// 格式化日志方法
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})

	// 带字段的日志方法（如果支持结构化日志）
	WithField(key string, value interface{}) interface{}
	WithFields(fields map[string]interface{}) interface{}
}

// KartLoggerAdapter adapts github.com/kart-io/logger to NotifyHub logger interface
type KartLoggerAdapter struct {
	*AdapterBase
	logger KartLogger
}

// NewKartLoggerAdapter creates a new Kart logger adapter
func NewKartLoggerAdapter(kartLogger KartLogger, level logger.LogLevel) logger.Interface {
	return &KartLoggerAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      kartLogger,
	}
}

func (k *KartLoggerAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return &KartLoggerAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      k.logger,
	}
}

func (k *KartLoggerAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if k.ShouldLog(logger.Info) {
		if len(data) > 0 {
			k.logger.Infof(msg, data...)
		} else {
			k.logger.Info(msg)
		}
	}
}

func (k *KartLoggerAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if k.ShouldLog(logger.Warn) {
		if len(data) > 0 {
			k.logger.Warnf(msg, data...)
		} else {
			k.logger.Warn(msg)
		}
	}
}

func (k *KartLoggerAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if k.ShouldLog(logger.Error) {
		if len(data) > 0 {
			k.logger.Errorf(msg, data...)
		} else {
			k.logger.Error(msg)
		}
	}
}

func (k *KartLoggerAdapter) Debug(ctx context.Context, msg string, data ...interface{}) {
	if k.ShouldLog(logger.Debug) {
		if len(data) > 0 {
			k.logger.Debugf(msg, data...)
		} else {
			k.logger.Debug(msg)
		}
	}
}

func (k *KartLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if k.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	operation, affected := fc()

	// 使用 WithFields 来记录结构化数据（如果支持）
	loggerWithFields := k.logger.WithFields(map[string]interface{}{
		"operation":   operation,
		"duration_ms": float64(elapsed.Nanoseconds()) / 1e6,
		"affected":    affected,
	})

	if err != nil && k.ShouldLog(logger.Error) {
		if withFieldLogger, ok := loggerWithFields.(KartLogger); ok {
			withFieldLogger.WithField("error", err.Error()).(KartLogger).Error("Operation failed")
		} else {
			k.logger.Errorf("Operation failed: %s, Duration: %.3fms, Affected: %d, Error: %v",
				operation, float64(elapsed.Nanoseconds())/1e6, affected, err)
		}
	} else if k.ShouldLog(logger.Info) {
		if infoLogger, ok := loggerWithFields.(KartLogger); ok {
			infoLogger.Info("Operation completed")
		} else {
			k.logger.Infof("Operation completed: %s, Duration: %.3fms, Affected: %d",
				operation, float64(elapsed.Nanoseconds())/1e6, affected)
		}
	}
}

// ================================
// 简化版 Kart Logger adapter（如果不支持 WithField 方法）
// ================================

// SimpleKartLogger interface for simpler kart logger implementations
type SimpleKartLogger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// SimpleKartLoggerAdapter adapts simple kart logger implementations
type SimpleKartLoggerAdapter struct {
	*AdapterBase
	logger SimpleKartLogger
}

// NewSimpleKartLoggerAdapter creates a new simple Kart logger adapter
func NewSimpleKartLoggerAdapter(simpleKartLogger SimpleKartLogger, level logger.LogLevel) logger.Interface {
	return &SimpleKartLoggerAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      simpleKartLogger,
	}
}

func (s *SimpleKartLoggerAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return &SimpleKartLoggerAdapter{
		AdapterBase: NewAdapterBase(level),
		logger:      s.logger,
	}
}

func (s *SimpleKartLoggerAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if s.ShouldLog(logger.Info) {
		if len(data) > 0 {
			s.logger.Infof(msg, data...)
		} else {
			s.logger.Info(msg)
		}
	}
}

func (s *SimpleKartLoggerAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if s.ShouldLog(logger.Warn) {
		if len(data) > 0 {
			s.logger.Warnf(msg, data...)
		} else {
			s.logger.Warn(msg)
		}
	}
}

func (s *SimpleKartLoggerAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if s.ShouldLog(logger.Error) {
		if len(data) > 0 {
			s.logger.Errorf(msg, data...)
		} else {
			s.logger.Error(msg)
		}
	}
}

func (s *SimpleKartLoggerAdapter) Debug(ctx context.Context, msg string, data ...interface{}) {
	if s.ShouldLog(logger.Debug) {
		if len(data) > 0 {
			s.logger.Debugf(msg, data...)
		} else {
			s.logger.Debug(msg)
		}
	}
}

func (s *SimpleKartLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if s.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	operation, affected := fc()

	if err != nil && s.ShouldLog(logger.Error) {
		s.logger.Errorf("Operation failed: %s, Duration: %.3fms, Affected: %d, Error: %v",
			operation, float64(elapsed.Nanoseconds())/1e6, affected, err)
	} else if s.ShouldLog(logger.Info) {
		s.logger.Infof("Operation completed: %s, Duration: %.3fms, Affected: %d",
			operation, float64(elapsed.Nanoseconds())/1e6, affected)
	}
}
