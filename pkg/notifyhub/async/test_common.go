package async

import "github.com/kart-io/notifyhub/pkg/logger"

// testLogger provides a common mock logger for all tests in the async package
type testLogger struct{}

func (m *testLogger) Debug(msg string, args ...interface{}) {}
func (m *testLogger) Info(msg string, args ...interface{})  {}
func (m *testLogger) Warn(msg string, args ...interface{})  {}
func (m *testLogger) Error(msg string, args ...interface{}) {}
func (m *testLogger) LogMode(level logger.LogLevel) logger.Logger { return m }

// getTestLogger returns a shared mock logger instance for tests
func getTestLogger() logger.Logger {
	return &testLogger{}
}