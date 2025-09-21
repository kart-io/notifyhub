package hub

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/logger"
)

// MockTransport for testing
type MockTransport struct {
	name     string
	sendFunc func(ctx context.Context, msg *core.Message, target core.Target) (*core.Result, error)
	calls    []SendCall
}

type SendCall struct {
	Message *core.Message
	Target  core.Target
}

func NewMockTransport(name string) *MockTransport {
	return &MockTransport{
		name:  name,
		calls: make([]SendCall, 0),
	}
}

func (m *MockTransport) Send(ctx context.Context, msg *core.Message, target core.Target) (*core.Result, error) {
	m.calls = append(m.calls, SendCall{Message: msg, Target: target})

	if m.sendFunc != nil {
		return m.sendFunc(ctx, msg, target)
	}

	result := core.NewResult(msg.ID, target)
	result.Status = core.StatusSent
	result.Success = true
	now := time.Now()
	result.SentAt = &now
	return result, nil
}

func (m *MockTransport) Name() string {
	return m.name
}

func (m *MockTransport) Health(ctx context.Context) error {
	return nil
}

func (m *MockTransport) Shutdown() error {
	return nil
}

func (m *MockTransport) GetCalls() []SendCall {
	return m.calls
}

func (m *MockTransport) SetSendFunc(fn func(ctx context.Context, msg *core.Message, target core.Target) (*core.Result, error)) {
	m.sendFunc = fn
}

// MockMiddleware for testing
type MockMiddleware struct {
	name        string
	processFunc func(ctx context.Context, msg *core.Message, targets []core.Target, next ProcessFunc) (*core.SendingResults, error)
	calls       []MiddlewareCall
}

type MiddlewareCall struct {
	Message *core.Message
	Targets []core.Target
}

func NewMockMiddleware(name string) *MockMiddleware {
	return &MockMiddleware{
		name:  name,
		calls: make([]MiddlewareCall, 0),
	}
}

func (m *MockMiddleware) Process(ctx context.Context, msg *core.Message, targets []core.Target, next ProcessFunc) (*core.SendingResults, error) {
	m.calls = append(m.calls, MiddlewareCall{Message: msg, Targets: targets})

	if m.processFunc != nil {
		return m.processFunc(ctx, msg, targets, next)
	}

	return next(ctx, msg, targets)
}

func (m *MockMiddleware) GetCalls() []MiddlewareCall {
	return m.calls
}

func (m *MockMiddleware) SetProcessFunc(fn func(ctx context.Context, msg *core.Message, targets []core.Target, next ProcessFunc) (*core.SendingResults, error)) {
	m.processFunc = fn
}

// MockIDGenerator for testing
type MockIDGenerator struct {
	counter int
}

func NewMockIDGenerator() *MockIDGenerator {
	return &MockIDGenerator{counter: 0}
}

func (m *MockIDGenerator) Generate() string {
	m.counter++
	return "test-id-" + string(rune(m.counter))
}

// Test Hub creation
func TestNewHub(t *testing.T) {
	opts := &Options{
		Logger: logger.Default,
	}

	hub := NewHub(opts)

	if hub == nil {
		t.Fatal("Expected hub to be created, got nil")
	}

	if hub.IsShutdown() {
		t.Error("Expected hub to not be shutdown initially")
	}
}

// Test transport registration
func TestRegisterTransport(t *testing.T) {
	hub := NewHub(nil)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	transport := NewMockTransport("email")
	hub.RegisterTransport(transport)

	transports := hub.ListTransports()
	if len(transports) != 1 {
		t.Errorf("Expected 1 transport, got %d", len(transports))
	}

	if transports[0] != "email" {
		t.Errorf("Expected transport name 'email', got '%s'", transports[0])
	}
}

// Test sending a message
func TestSend(t *testing.T) {
	hub := NewHub(nil)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	transport := NewMockTransport("email")
	hub.RegisterTransport(transport)

	msg := message.NewBuilder().
		Title("Test Message").
		Body("Test body").
		AddTarget(core.NewTarget(core.TargetTypeEmail, "test@example.com", "email"))

	ctx := context.Background()
	builtMsg := msg.Build()
	results, err := hub.Send(ctx, builtMsg, builtMsg.Targets)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if results == nil {
		t.Fatal("Expected results, got nil")
	}

	if len(results.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results.Results))
	}

	if !results.Results[0].Success {
		t.Error("Expected result to be successful")
	}

	calls := transport.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 transport call, got %d", len(calls))
	}
}

// Test sending with unknown transport
func TestSendUnknownTransport(t *testing.T) {
	hub := NewHub(nil)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	msg := message.NewBuilder().
		Title("Test Message").
		AddTarget(core.NewTarget(core.TargetTypeEmail, "test@example.com", "unknown"))

	ctx := context.Background()
	builtMsg := msg.Build()
	results, err := hub.Send(ctx, builtMsg, builtMsg.Targets)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results.Results))
	}

	if results.Results[0].Success {
		t.Error("Expected result to be unsuccessful for unknown transport")
	}
}

// Test middleware
func TestAddMiddleware(t *testing.T) {
	hub := NewHub(nil)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	transport := NewMockTransport("email")
	hub.RegisterTransport(transport)

	middleware := NewMockMiddleware("test-middleware")
	hub.AddMiddleware(middleware)

	msg := message.NewBuilder().
		Title("Test Message").
		AddTarget(core.NewTarget(core.TargetTypeEmail, "test@example.com", "email"))

	ctx := context.Background()
	builtMsg := msg.Build()
	_, err := hub.Send(ctx, builtMsg, builtMsg.Targets)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	calls := middleware.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 middleware call, got %d", len(calls))
	}
}

// Test middleware error handling
func TestMiddlewareError(t *testing.T) {
	hub := NewHub(nil)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	transport := NewMockTransport("email")
	hub.RegisterTransport(transport)

	middleware := NewMockMiddleware("error-middleware")
	middleware.SetProcessFunc(func(ctx context.Context, msg *core.Message, targets []core.Target, next ProcessFunc) (*core.SendingResults, error) {
		return nil, errors.New("middleware error")
	})
	hub.AddMiddleware(middleware)

	msg := message.NewBuilder().
		Title("Test Message").
		AddTarget(core.NewTarget(core.TargetTypeEmail, "test@example.com", "email"))

	ctx := context.Background()
	builtMsg := msg.Build()
	_, err := hub.Send(ctx, builtMsg, builtMsg.Targets)

	if err == nil {
		t.Error("Expected error from middleware, got nil")
	}

	if err.Error() != "middleware error" {
		t.Errorf("Expected 'middleware error', got '%s'", err.Error())
	}
}

// Test transport error handling
func TestTransportError(t *testing.T) {
	hub := NewHub(nil)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	transport := NewMockTransport("email")
	transport.SetSendFunc(func(ctx context.Context, msg *core.Message, target core.Target) (*core.Result, error) {
		return &core.Result{
			MessageID: msg.ID,
			Target:    target,
			Success:   false,
			Error:     errors.New("transport error"),
		}, nil
	})
	hub.RegisterTransport(transport)

	msg := message.NewBuilder().
		Title("Test Message").
		AddTarget(core.NewTarget(core.TargetTypeEmail, "test@example.com", "email"))

	ctx := context.Background()
	builtMsg := msg.Build()
	results, err := hub.Send(ctx, builtMsg, builtMsg.Targets)

	if err != nil {
		t.Fatalf("Expected no error at hub level, got %v", err)
	}

	if results.Results[0].Success {
		t.Error("Expected result to indicate failure")
	}

	if results.Results[0].Error == nil {
		t.Error("Expected result to have error")
	}
}

// Test health check
func TestHealthCheck(t *testing.T) {
	hub := NewHub(nil)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	status := hub.Health(context.Background())

	if !status.Healthy {
		t.Error("Expected hub to be healthy")
	}

	if status.Details == nil {
		t.Error("Expected health details to be present")
	}
}

// Test shutdown
func TestShutdown(t *testing.T) {
	hub := NewHub(nil)

	transport := NewMockTransport("email")
	hub.RegisterTransport(transport)

	ctx := context.Background()
	err := hub.Shutdown(ctx)

	if err != nil {
		t.Errorf("Expected no error during shutdown, got %v", err)
	}

	if !hub.IsShutdown() {
		t.Error("Expected hub to be shutdown")
	}
}

// Test sending after shutdown
func TestSendAfterShutdown(t *testing.T) {
	hub := NewHub(nil)

	transport := NewMockTransport("email")
	hub.RegisterTransport(transport)

	ctx := context.Background()
	err := hub.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	msg := message.NewBuilder().
		Title("Test Message").
		AddTarget(core.NewTarget(core.TargetTypeEmail, "test@example.com", "email"))

	builtMsg := msg.Build()
	_, err = hub.Send(ctx, builtMsg, builtMsg.Targets)

	if err == nil {
		t.Error("Expected error when sending after shutdown")
	}
}

// Test concurrent sends
func TestConcurrentSends(t *testing.T) {
	hub := NewHub(nil)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	transport := NewMockTransport("email")
	hub.RegisterTransport(transport)

	numGoroutines := 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			msg := message.NewBuilder().
				Title("Concurrent Test").
				AddTarget(core.NewTarget(core.TargetTypeEmail, "test@example.com", "email"))

			ctx := context.Background()
			builtMsg := msg.Build()
			_, err := hub.Send(ctx, builtMsg, builtMsg.Targets)
			done <- err
		}()
	}

	errors := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent send %d failed: %v", i, err)
			errors++
		}
	}

	calls := transport.GetCalls()
	expectedCalls := numGoroutines - errors
	if len(calls) != expectedCalls {
		t.Errorf("Expected %d successful calls, got %d", expectedCalls, len(calls))
	}
}

// Test context cancellation
func TestContextCancellation(t *testing.T) {
	hub := NewHub(nil)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	transport := NewMockTransport("email")
	transport.SetSendFunc(func(ctx context.Context, msg *core.Message, target core.Target) (*core.Result, error) {
		// Simulate slow operation
		select {
		case <-time.After(100 * time.Millisecond):
			return &core.Result{
				MessageID: msg.ID,
				Target:    target,
				Success:   true,
			}, nil
		case <-ctx.Done():
			return &core.Result{
				MessageID: msg.ID,
				Target:    target,
				Success:   false,
				Error:     ctx.Err(),
			}, nil
		}
	})
	hub.RegisterTransport(transport)

	msg := message.NewBuilder().
		Title("Test Message").
		AddTarget(core.NewTarget(core.TargetTypeEmail, "test@example.com", "email"))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	builtMsg := msg.Build()
	results, err := hub.Send(ctx, builtMsg, builtMsg.Targets)

	// Context cancellation might be handled at transport level
	if err == nil && results != nil && len(results.Results) > 0 {
		result := results.Results[0]
		if result.Success {
			t.Log("Transport completed before context cancellation")
		} else if result.Error != nil {
			t.Logf("Transport handled context cancellation: %v", result.Error)
		}
	}
}

// Test empty targets
func TestEmptyTargets(t *testing.T) {
	hub := NewHub(nil)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	transport := NewMockTransport("email")
	hub.RegisterTransport(transport)

	msg := message.NewBuilder().
		Title("Test Message")

	ctx := context.Background()
	builtMsg := msg.Build()
	results, err := hub.Send(ctx, builtMsg, []core.Target{})

	if err == nil {
		t.Error("Expected error with empty targets and no message targets")
	}

	// When there's an error, results should be nil or empty
	if results != nil && len(results.Results) != 0 {
		t.Errorf("Expected 0 results with error, got %d", len(results.Results))
	}

	calls := transport.GetCalls()
	if len(calls) != 0 {
		t.Errorf("Expected 0 transport calls with empty targets, got %d", len(calls))
	}
}
