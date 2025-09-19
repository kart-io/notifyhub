package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/core/hub"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
)

// MockTransport 模拟传输层实现
type MockTransport struct {
	mu        sync.Mutex
	name      string
	calls     []SendCall
	responses map[string]*sending.Result
	errors    map[string]error
	delay     time.Duration
	failRate  float32 //nolint:unused // 保留供未来使用
}

// SendCall 记录发送调用
type SendCall struct {
	Message   *message.Message
	Target    sending.Target
	Timestamp time.Time
}

// NewMockTransport 创建新的模拟传输器
func NewMockTransport(name string) *MockTransport {
	return &MockTransport{
		name:      name,
		calls:     make([]SendCall, 0),
		responses: make(map[string]*sending.Result),
		errors:    make(map[string]error),
	}
}

// Name 返回传输器名称
func (m *MockTransport) Name() string {
	return m.name
}

// Send 模拟发送消息
func (m *MockTransport) Send(ctx context.Context, msg *message.Message, target sending.Target) (*sending.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 记录调用
	m.calls = append(m.calls, SendCall{
		Message:   msg,
		Target:    target,
		Timestamp: time.Now(),
	})

	// 模拟延迟
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// 生成结果键
	key := target.Value

	// 检查是否有预设错误
	if err, ok := m.errors[key]; ok {
		result := sending.NewResult(msg.ID, target)
		result.Status = sending.StatusFailed
		result.Error = err
		return result, err
	}

	// 检查是否有预设响应
	if result, ok := m.responses[key]; ok {
		return result, nil
	}

	// 生成默认成功结果
	result := sending.NewResult(msg.ID, target)
	result.Status = sending.StatusSent
	result.Success = true
	now := time.Now()
	result.SentAt = &now
	result.EndTime = now

	return result, nil
}

// Shutdown 关闭传输器
func (m *MockTransport) Shutdown() error {
	return nil
}

// SetResponse 设置特定目标的响应
func (m *MockTransport) SetResponse(targetValue string, result *sending.Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[targetValue] = result
}

// SetError 设置特定目标的错误
func (m *MockTransport) SetError(targetValue string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[targetValue] = err
}

// SetDelay 设置发送延迟
func (m *MockTransport) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = delay
}

// GetCalls 获取所有调用记录
func (m *MockTransport) GetCalls() []SendCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]SendCall, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// Reset 重置所有记录
func (m *MockTransport) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]SendCall, 0)
	m.responses = make(map[string]*sending.Result)
	m.errors = make(map[string]error)
}

// MockMiddleware 模拟中间件
type MockMiddleware struct {
	mu          sync.Mutex
	name        string
	calls       []MiddlewareCall
	shouldError bool
	errorMsg    string
	beforeFunc  func(ctx context.Context, msg *message.Message, targets []sending.Target)
	afterFunc   func(ctx context.Context, results *sending.SendingResults)
}

// MiddlewareCall 记录中间件调用
type MiddlewareCall struct {
	Message   *message.Message
	Targets   []sending.Target
	Timestamp time.Time
}

// NewMockMiddleware 创建模拟中间件
func NewMockMiddleware(name string) *MockMiddleware {
	return &MockMiddleware{
		name:  name,
		calls: make([]MiddlewareCall, 0),
	}
}

// Process 处理中间件逻辑
func (m *MockMiddleware) Process(ctx context.Context, msg *message.Message, targets []sending.Target, next hub.ProcessFunc) (*sending.SendingResults, error) {
	m.mu.Lock()
	m.calls = append(m.calls, MiddlewareCall{
		Message:   msg,
		Targets:   targets,
		Timestamp: time.Now(),
	})

	if m.beforeFunc != nil {
		m.beforeFunc(ctx, msg, targets)
	}
	m.mu.Unlock()

	if m.shouldError {
		return nil, &MiddlewareError{Message: m.errorMsg}
	}

	// 调用下一个中间件
	results, err := next(ctx, msg, targets)

	m.mu.Lock()
	if m.afterFunc != nil && results != nil {
		m.afterFunc(ctx, results)
	}
	m.mu.Unlock()

	return results, err
}

// GetCalls 获取调用记录
func (m *MockMiddleware) GetCalls() []MiddlewareCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]MiddlewareCall, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// SetBeforeFunc 设置前置处理函数
func (m *MockMiddleware) SetBeforeFunc(fn func(ctx context.Context, msg *message.Message, targets []sending.Target)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.beforeFunc = fn
}

// SetAfterFunc 设置后置处理函数
func (m *MockMiddleware) SetAfterFunc(fn func(ctx context.Context, results *sending.SendingResults)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.afterFunc = fn
}

// SetShouldError 设置是否应该返回错误
func (m *MockMiddleware) SetShouldError(shouldError bool, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorMsg = errorMsg
}

// MiddlewareError 中间件错误
type MiddlewareError struct {
	Message string
}

func (e *MiddlewareError) Error() string {
	return e.Message
}
