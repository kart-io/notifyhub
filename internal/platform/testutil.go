package platform

import (
	"context"
	"time"
)

// MockSender is a mock implementation of the Sender interface for testing
type MockSender struct {
	name         string
	sendError    error
	sendResults  []*SendResult
	validateErr  error
	capabilities PlatformCapabilities
	healthErr    error
	closeErr     error
}

// NewMockSender creates a new mock sender
func NewMockSender(name string) *MockSender {
	return &MockSender{
		name: name,
		capabilities: PlatformCapabilities{
			Name:                 name,
			SupportedTargetTypes: []string{"test"},
			SupportedFormats:     []string{"text"},
			MaxMessageSize:       1000,
			RequiredSettings:     []string{},
		},
	}
}

// Name returns the mock sender name
func (m *MockSender) Name() string {
	return m.name
}

// Send mocks sending a message
func (m *MockSender) Send(ctx context.Context, msg *InternalMessage, targets []InternalTarget) ([]*SendResult, error) {
	if m.sendError != nil {
		return nil, m.sendError
	}

	if m.sendResults != nil {
		return m.sendResults, nil
	}

	// Default: create successful results
	results := make([]*SendResult, len(targets))
	for i, target := range targets {
		results[i] = &SendResult{
			Target:    target,
			Success:   true,
			MessageID: "mock_" + msg.ID + "_" + target.Value,
			SentAt:    time.Now(),
			Duration:  time.Millisecond * 100,
			Metadata:  make(map[string]string),
		}
	}
	return results, nil
}

// ValidateTarget mocks target validation
func (m *MockSender) ValidateTarget(target InternalTarget) error {
	return m.validateErr
}

// GetCapabilities returns mock capabilities
func (m *MockSender) GetCapabilities() PlatformCapabilities {
	return m.capabilities
}

// IsHealthy mocks health check
func (m *MockSender) IsHealthy(ctx context.Context) error {
	return m.healthErr
}

// Close mocks closing the sender
func (m *MockSender) Close() error {
	return m.closeErr
}

// SetSendError sets an error to be returned by Send
func (m *MockSender) SetSendError(err error) {
	m.sendError = err
}

// SetSendResults sets specific results to be returned by Send
func (m *MockSender) SetSendResults(results []*SendResult) {
	m.sendResults = results
}

// SetValidateError sets an error to be returned by ValidateTarget
func (m *MockSender) SetValidateError(err error) {
	m.validateErr = err
}

// SetHealthError sets an error to be returned by IsHealthy
func (m *MockSender) SetHealthError(err error) {
	m.healthErr = err
}

// SetCloseError sets an error to be returned by Close
func (m *MockSender) SetCloseError(err error) {
	m.closeErr = err
}

// MockSenderFactory is a mock implementation of SenderFactory
type MockSenderFactory struct {
	senders map[string]Sender
	errors  map[string]error
}

// NewMockSenderFactory creates a new mock sender factory
func NewMockSenderFactory() *MockSenderFactory {
	return &MockSenderFactory{
		senders: make(map[string]Sender),
		errors:  make(map[string]error),
	}
}

// CreateSender creates a mock sender or returns an error
func (f *MockSenderFactory) CreateSender(platform string, config map[string]interface{}) (Sender, error) {
	if err, exists := f.errors[platform]; exists {
		return nil, err
	}
	if sender, exists := f.senders[platform]; exists {
		return sender, nil
	}
	return NewMockSender(platform), nil
}

// GetSupportedPlatforms returns mock supported platforms
func (f *MockSenderFactory) GetSupportedPlatforms() []string {
	var platforms []string
	for platform := range f.senders {
		platforms = append(platforms, platform)
	}
	return platforms
}

// ValidateConfig always returns nil for mocking
func (f *MockSenderFactory) ValidateConfig(platform string, config map[string]interface{}) error {
	if err, exists := f.errors[platform]; exists {
		return err
	}
	return nil
}

// AddSender adds a mock sender for a platform
func (f *MockSenderFactory) AddSender(platform string, sender Sender) {
	f.senders[platform] = sender
}

// SetError sets an error for a platform
func (f *MockSenderFactory) SetError(platform string, err error) {
	f.errors[platform] = err
}
