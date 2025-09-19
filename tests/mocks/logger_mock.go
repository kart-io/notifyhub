package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/logger"
)

// MockLogger 模拟日志器
type MockLogger struct {
	mu       sync.Mutex
	messages []LogMessage
}

// LogMessage 日志消息
type LogMessage struct {
	Level     string
	Message   string
	Fields    []interface{}
	Timestamp time.Time
}

// NewMockLogger 创建新的模拟日志器
func NewMockLogger() *MockLogger {
	return &MockLogger{
		messages: make([]LogMessage, 0),
	}
}

// Debug 记录调试日志 (implements logger.Interface)
func (m *MockLogger) Debug(ctx context.Context, msg string, data ...interface{}) {
	m.log("DEBUG", msg, data...)
}

// Info 记录信息日志 (implements logger.Interface)
func (m *MockLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	m.log("INFO", msg, data...)
}

// Warn 记录警告日志 (implements logger.Interface)
func (m *MockLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	m.log("WARN", msg, data...)
}

// Error 记录错误日志 (implements logger.Interface)
func (m *MockLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	m.log("ERROR", msg, data...)
}

func (m *MockLogger) log(level, msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, LogMessage{
		Level:     level,
		Message:   msg,
		Fields:    fields,
		Timestamp: time.Now(),
	})
}

// GetMessages 获取所有日志消息
func (m *MockLogger) GetMessages() []LogMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	messages := make([]LogMessage, len(m.messages))
	copy(messages, m.messages)
	return messages
}

// GetMessagesByLevel 获取指定级别的日志
func (m *MockLogger) GetMessagesByLevel(level string) []LogMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []LogMessage
	for _, msg := range m.messages {
		if msg.Level == level {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// HasMessage 检查是否包含特定消息
func (m *MockLogger) HasMessage(message string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, msg := range m.messages {
		if msg.Message == message {
			return true
		}
	}
	return false
}

// HasError 检查是否有错误日志
func (m *MockLogger) HasError() bool {
	return len(m.GetMessagesByLevel("ERROR")) > 0
}

// Clear 清空所有日志
func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]LogMessage, 0)
}

// Count 获取日志数量
func (m *MockLogger) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

// CountByLevel 获取指定级别的日志数量
func (m *MockLogger) CountByLevel(level string) int {
	return len(m.GetMessagesByLevel(level))
}

// LogMode sets the log level (implements logger.Interface)
func (m *MockLogger) LogMode(level logger.LogLevel) logger.Interface {
	// Return a copy with the specified level (simplified for testing)
	return m
}

// Trace logs operation traces (implements logger.Interface)
func (m *MockLogger) Trace(ctx context.Context, begin time.Time, fc func() (operation string, targets int64), err error, data ...interface{}) {
	if fc != nil {
		operation, targets := fc()
		duration := time.Since(begin)
		logData := []interface{}{"duration", duration.Milliseconds(), "targets", targets, "error", err}
		logData = append(logData, data...)
		m.log("TRACE", operation, logData...)
	}
}
