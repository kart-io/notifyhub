package notifiers

import (
	"context"
	"fmt"
	"time"
)

// MockNotifier is a mock implementation for testing purposes
type MockNotifier struct {
	name             string
	supportedTargets []TargetType
	shouldFail       bool
	delay            time.Duration
	sendResults      []*SendResult
}

// NewMockNotifier creates a new mock notifier
func NewMockNotifier(name string) *MockNotifier {
	return &MockNotifier{
		name: name,
		supportedTargets: []TargetType{
			TargetTypeEmail,
			TargetTypeUser,
			TargetTypeGroup,
			TargetTypeChannel,
		},
		shouldFail:  false,
		delay:       10 * time.Millisecond,
		sendResults: make([]*SendResult, 0),
	}
}

// WithSupportedTargets sets which target types this mock supports
func (m *MockNotifier) WithSupportedTargets(targets ...TargetType) *MockNotifier {
	m.supportedTargets = targets
	return m
}

// WithFailure makes this mock notifier always fail
func (m *MockNotifier) WithFailure() *MockNotifier {
	m.shouldFail = true
	return m
}

// WithDelay sets artificial delay for testing
func (m *MockNotifier) WithDelay(delay time.Duration) *MockNotifier {
	m.delay = delay
	return m
}

// GetName returns the notifier name
func (m *MockNotifier) GetName() string {
	return m.name
}

// Name returns the notifier name (implements Notifier interface)
func (m *MockNotifier) Name() string {
	return m.name
}

// GetSupportedTargets returns supported target types
func (m *MockNotifier) GetSupportedTargets() []TargetType {
	return m.supportedTargets
}

// SupportsTarget checks if the notifier supports the given target
func (m *MockNotifier) SupportsTarget(target Target) bool {
	for _, supportedType := range m.supportedTargets {
		if target.Type == supportedType {
			return true
		}
	}
	return false
}

// Send implements the Notifier interface
func (m *MockNotifier) Send(ctx context.Context, message *Message) ([]*SendResult, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	results := make([]*SendResult, 0, len(message.Targets))

	for _, target := range message.Targets {
		result := &SendResult{
			Target:   target,
			Platform: m.name,
			Success:  !m.shouldFail,
			Duration: m.delay,
			SentAt:   time.Now(),
			Attempts: 1,
		}

		if m.shouldFail {
			result.Error = "mock notifier configured to fail"
		}

		results = append(results, result)
		m.sendResults = append(m.sendResults, result)
	}

	if m.shouldFail {
		return results, fmt.Errorf("mock notifier send failed")
	}

	return results, nil
}

// IsHealthy implements the Notifier interface
func (m *MockNotifier) IsHealthy(ctx context.Context) error {
	if m.shouldFail {
		return fmt.Errorf("mock notifier is unhealthy")
	}
	return nil
}

// Health implements the Notifier interface
func (m *MockNotifier) Health(ctx context.Context) error {
	return m.IsHealthy(ctx)
}

// Shutdown implements the Notifier interface
func (m *MockNotifier) Shutdown(ctx context.Context) error {
	// Mock shutdown - always successful
	return nil
}

// GetLastResults returns the last send results for testing assertions
func (m *MockNotifier) GetLastResults() []*SendResult {
	return m.sendResults
}

// GetSendCount returns the number of sends performed
func (m *MockNotifier) GetSendCount() int {
	return len(m.sendResults)
}

// Reset clears the send history
func (m *MockNotifier) Reset() {
	m.sendResults = make([]*SendResult, 0)
}
