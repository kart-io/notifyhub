package platform

import (
	"context"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// MockSender is a mock implementation of the ExternalSender interface for testing
type MockSender struct {
	name         string
	sendError    error
	sendResults  []*platform.SendResult
	validateErr  error
	capabilities platform.Capabilities
	healthErr    error
	closeErr     error
}

// NewMockSender creates a new mock sender
func NewMockSender(name string) *MockSender {
	return &MockSender{
		name: name,
		capabilities: platform.Capabilities{
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
func (m *MockSender) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	if m.sendError != nil {
		return nil, m.sendError
	}

	if m.sendResults != nil {
		return m.sendResults, nil
	}

	// Default: create successful results
	results := make([]*platform.SendResult, len(targets))
	for i, t := range targets {
		results[i] = &platform.SendResult{
			Target:    t,
			Success:   true,
			MessageID: "mock_" + msg.ID + "_" + t.Value,
			Response:  "success",
			Metadata:  make(map[string]interface{}),
		}
	}
	return results, nil
}

// ValidateTarget mocks target validation
func (m *MockSender) ValidateTarget(t target.Target) error {
	return m.validateErr
}

// GetCapabilities returns mock capabilities
func (m *MockSender) GetCapabilities() platform.Capabilities {
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
func (m *MockSender) SetSendResults(results []*platform.SendResult) {
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
	senders map[string]platform.ExternalSender
	errors  map[string]error
}

// NewMockSenderFactory creates a new mock sender factory
func NewMockSenderFactory() *MockSenderFactory {
	return &MockSenderFactory{
		senders: make(map[string]platform.ExternalSender),
		errors:  make(map[string]error),
	}
}

// CreateSender creates a mock sender or returns an error
func (f *MockSenderFactory) CreateSender(platformName string, config map[string]interface{}, logger logger.Logger) (platform.ExternalSender, error) {
	if err, exists := f.errors[platformName]; exists {
		return nil, err
	}
	if sender, exists := f.senders[platformName]; exists {
		return sender, nil
	}
	return NewMockSender(platformName), nil
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
func (f *MockSenderFactory) ValidateConfig(platformName string, config map[string]interface{}) error {
	if err, exists := f.errors[platformName]; exists {
		return err
	}
	return nil
}

// AddSender adds a mock sender for a platform
func (f *MockSenderFactory) AddSender(platformName string, sender platform.ExternalSender) {
	f.senders[platformName] = sender
}

// SetError sets an error for a platform
func (f *MockSenderFactory) SetError(platformName string, err error) {
	f.errors[platformName] = err
}
