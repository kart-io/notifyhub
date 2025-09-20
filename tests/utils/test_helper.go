package utils

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/api"
	apiconfig "github.com/kart-io/notifyhub/api/config"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/tests/mocks"
)

// TestHelper 测试辅助结构体
type TestHelper struct {
	t *testing.T
}

// NewTestHelper 创建测试辅助工具
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t: t}
}

// AssertNil 断言值为nil
func (h *TestHelper) AssertNil(value interface{}, msgAndArgs ...interface{}) {
	h.t.Helper()
	if value != nil {
		h.t.Errorf("Expected nil, but got: %v. %v", value, msgAndArgs)
	}
}

// AssertNotNil 断言值不为nil
func (h *TestHelper) AssertNotNil(value interface{}, msgAndArgs ...interface{}) {
	h.t.Helper()
	if value == nil {
		h.t.Errorf("Expected non-nil value. %v", msgAndArgs)
	}
}

// AssertEqual 断言相等
func (h *TestHelper) AssertEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	h.t.Helper()
	if expected != actual {
		h.t.Errorf("Expected %v, but got %v. %v", expected, actual, msgAndArgs)
	}
}

// AssertNotEqual 断言不相等
func (h *TestHelper) AssertNotEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	h.t.Helper()
	if expected == actual {
		h.t.Errorf("Expected values to be different, but both were %v. %v", expected, msgAndArgs)
	}
}

// AssertTrue 断言为真
func (h *TestHelper) AssertTrue(value bool, msgAndArgs ...interface{}) {
	h.t.Helper()
	if !value {
		h.t.Errorf("Expected true, but got false. %v", msgAndArgs)
	}
}

// AssertFalse 断言为假
func (h *TestHelper) AssertFalse(value bool, msgAndArgs ...interface{}) {
	h.t.Helper()
	if value {
		h.t.Errorf("Expected false, but got true. %v", msgAndArgs)
	}
}

// AssertError 断言有错误
func (h *TestHelper) AssertError(err error, msgAndArgs ...interface{}) {
	h.t.Helper()
	if err == nil {
		h.t.Errorf("Expected an error, but got nil. %v", msgAndArgs)
	}
}

// AssertNoError 断言没有错误
func (h *TestHelper) AssertNoError(err error, msgAndArgs ...interface{}) {
	h.t.Helper()
	if err != nil {
		h.t.Errorf("Expected no error, but got: %v. %v", err, msgAndArgs)
	}
}

// AssertContains 断言包含子串
func (h *TestHelper) AssertContains(str, substr string, msgAndArgs ...interface{}) {
	h.t.Helper()
	if !contains(str, substr) {
		h.t.Errorf("Expected '%s' to contain '%s'. %v", str, substr, msgAndArgs)
	}
}

// AssertNotContains 断言不包含子串
func (h *TestHelper) AssertNotContains(str, substr string, msgAndArgs ...interface{}) {
	h.t.Helper()
	if contains(str, substr) {
		h.t.Errorf("Expected '%s' not to contain '%s'. %v", str, substr, msgAndArgs)
	}
}

// AssertPanic 断言会产生panic
func (h *TestHelper) AssertPanic(fn func(), msgAndArgs ...interface{}) {
	h.t.Helper()
	defer func() {
		if r := recover(); r == nil {
			h.t.Errorf("Expected panic, but function executed normally. %v", msgAndArgs)
		}
	}()
	fn()
}

// AssertNoPanic 断言不会产生panic
func (h *TestHelper) AssertNoPanic(fn func(), msgAndArgs ...interface{}) {
	h.t.Helper()
	defer func() {
		if r := recover(); r != nil {
			h.t.Errorf("Expected no panic, but got: %v. %v", r, msgAndArgs)
		}
	}()
	fn()
}

// AssertEventually 断言最终条件满足
func (h *TestHelper) AssertEventually(condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	h.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), waitFor)
	defer cancel()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.t.Errorf("Condition not met within %v. %v", waitFor, msgAndArgs)
			return
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// AssertNever 断言条件永不满足
func (h *TestHelper) AssertNever(condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	h.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), waitFor)
	defer cancel()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Success - condition never met
		case <-ticker.C:
			if condition() {
				h.t.Errorf("Condition should never be met, but was met. %v", msgAndArgs)
				return
			}
		}
	}
}

// CreateTestMessageWithParams 创建测试消息 (原来的CreateTestMessage重命名)
func CreateTestMessageWithParams(title, body string, priority int) *message.Message {
	msg := message.NewMessage()
	msg.SetTitle(title)
	msg.SetBody(body)
	msg.SetPriority(priority)
	return msg
}

// CreateTestTarget 创建测试目标
func CreateTestTarget(targetType sending.TargetType, value, platform string) sending.Target {
	return sending.NewTarget(targetType, value, platform)
}

// CreateTestConfig 创建测试配置
func CreateTestConfig() *apiconfig.Config {
	cfg := apiconfig.NewConfig()
	cfg.Queue = &apiconfig.QueueConfig{
		Type:    "memory",
		Size:    100,
		Workers: 2,
	}
	cfg.Debug = false
	return cfg
}

// WaitForCondition 等待条件满足
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// contains 检查字符串是否包含子串
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 || indexOfString(str, substr) >= 0)
}

// indexOfString 查找子串位置
func indexOfString(str, substr string) int {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// ================================
// Global assertion functions for backward compatibility
// ================================

// AssertTrue global assertion function
func AssertTrue(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	if !value {
		t.Errorf("Expected true, but got false. %v", msgAndArgs)
	}
}

// AssertFalse global assertion function
func AssertFalse(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	if value {
		t.Errorf("Expected false, but got true. %v", msgAndArgs)
	}
}

// AssertEqual global assertion function
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, but got %v. %v", expected, actual, msgAndArgs)
	}
}

// AssertNoError global assertion function
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected no error, but got: %v. %v", err, msgAndArgs)
	}
}

// AssertError global assertion function
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected an error, but got nil. %v", msgAndArgs)
	}
}

// ================================
// Test creation helper functions
// ================================

// NewMockLogger creates a mock logger
func NewMockLogger() *mocks.MockLogger {
	return mocks.NewMockLogger()
}

// CreateTestHub creates a test client (V2 API)
func CreateTestHub(t *testing.T) *api.Client {
	config := CreateTestConfig()
	// Note: V2 API doesn't use separate Options, logger is configured in config
	client, err := api.New(config)
	AssertNoError(t, err)
	return client
}

// CreateTestMessage creates a test message (no parameters version)
func CreateTestMessage() *message.Message {
	return CreateTestMessageWithDetails("Test Title", "Test Body", 3)
}

// CreateTestMessageWithDetails creates a test message with specific details
func CreateTestMessageWithDetails(title, body string, priority int) *message.Message {
	msg := message.NewMessage()
	msg.SetTitle(title)
	msg.SetBody(body)
	msg.SetPriority(priority)
	return msg
}

// CreateTestTargets creates test targets
func CreateTestTargets() []sending.Target {
	return []sending.Target{
		sending.NewTarget(sending.TargetTypeEmail, "test@example.com", "email"),
		sending.NewTarget(sending.TargetTypeUser, "user123", "feishu"),
	}
}
