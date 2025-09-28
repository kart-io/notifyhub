// Package platform provides test utilities for platform contract testing
// This file implements utilities for Task 5.3: Platform Interface Contract Testing
package platform

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct {
	rand *rand.Rand
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateTestMessage creates a test message with random but valid data
func (g *TestDataGenerator) GenerateTestMessage() *message.Message {
	msg := message.New()
	msg.Title = g.randomTitle()
	msg.Body = g.randomBody()
	msg.Format = g.randomFormat()
	msg.Priority = g.randomPriority()
	return msg
}

// GenerateTestMessages creates multiple test messages
func (g *TestDataGenerator) GenerateTestMessages(count int) []*message.Message {
	messages := make([]*message.Message, count)
	for i := 0; i < count; i++ {
		messages[i] = g.GenerateTestMessage()
	}
	return messages
}

// GenerateValidTargets creates valid targets for a given platform
func (g *TestDataGenerator) GenerateValidTargets(platformCapabilities Capabilities, count int) []target.Target {
	targets := make([]target.Target, count)
	supportedTypes := platformCapabilities.SupportedTargetTypes

	for i := 0; i < count; i++ {
		targetType := supportedTypes[g.rand.Intn(len(supportedTypes))]
		targets[i] = target.Target{
			Type:     targetType,
			Value:    g.randomTargetValue(targetType),
			Platform: platformCapabilities.Name,
		}
	}
	return targets
}

// GenerateInvalidTargets creates invalid targets for testing error handling
func (g *TestDataGenerator) GenerateInvalidTargets(platformCapabilities Capabilities, count int) []target.Target {
	targets := make([]target.Target, count)
	invalidTypes := g.getUnsupportedTargetTypes(platformCapabilities.SupportedTargetTypes)

	for i := 0; i < count; i++ {
		switch i % 4 {
		case 0:
			// Unsupported target type
			if len(invalidTypes) > 0 {
				invalidType := invalidTypes[g.rand.Intn(len(invalidTypes))]
				targets[i] = target.Target{
					Type:     invalidType,
					Value:    g.randomTargetValue(invalidType),
					Platform: platformCapabilities.Name,
				}
			} else {
				targets[i] = target.Target{
					Type:     "invalid",
					Value:    "test-value",
					Platform: platformCapabilities.Name,
				}
			}
		case 1:
			// Empty value
			validType := platformCapabilities.SupportedTargetTypes[0]
			targets[i] = target.Target{
				Type:     validType,
				Value:    "",
				Platform: platformCapabilities.Name,
			}
		case 2:
			// Empty type
			targets[i] = target.Target{
				Type:     "",
				Value:    "test-value",
				Platform: platformCapabilities.Name,
			}
		case 3:
			// Both empty
			targets[i] = target.Target{
				Type:     "",
				Value:    "",
				Platform: platformCapabilities.Name,
			}
		}
	}
	return targets
}

// randomTitle generates a random message title
func (g *TestDataGenerator) randomTitle() string {
	titles := []string{
		"Test Message",
		"Important Update",
		"System Alert",
		"Notification",
		"Daily Report",
		"Status Update",
		"Action Required",
		"Information",
	}
	return titles[g.rand.Intn(len(titles))]
}

// randomBody generates a random message body
func (g *TestDataGenerator) randomBody() string {
	bodies := []string{
		"This is a test message for platform validation.",
		"Please review the attached information and take appropriate action.",
		"System status has been updated. All services are operational.",
		"Your attention is required for the following items.",
		"Daily report is ready for review.",
		"Notification message with important details.",
		"Test content for validation purposes.",
		"Message body with various content types and formatting.",
	}
	return bodies[g.rand.Intn(len(bodies))]
}

// randomFormat generates a random message format
func (g *TestDataGenerator) randomFormat() message.Format {
	formats := []message.Format{
		message.FormatText,
		message.FormatMarkdown,
		message.FormatHTML,
	}
	return formats[g.rand.Intn(len(formats))]
}

// randomPriority generates a random message priority
func (g *TestDataGenerator) randomPriority() message.Priority {
	priorities := []message.Priority{
		message.PriorityLow,
		message.PriorityNormal,
		message.PriorityHigh,
		message.PriorityUrgent,
	}
	return priorities[g.rand.Intn(len(priorities))]
}

// randomTargetValue generates a random target value based on type
func (g *TestDataGenerator) randomTargetValue(targetType string) string {
	switch targetType {
	case target.TargetTypeEmail:
		return fmt.Sprintf("test%d@example.com", g.rand.Intn(1000))
	case target.TargetTypePhone:
		return fmt.Sprintf("+1555%07d", g.rand.Intn(10000000))
	case target.TargetTypeUser:
		return fmt.Sprintf("user_%d", g.rand.Intn(10000))
	case target.TargetTypeGroup:
		return fmt.Sprintf("group_%d", g.rand.Intn(1000))
	case target.TargetTypeChannel:
		return fmt.Sprintf("channel_%d", g.rand.Intn(1000))
	case target.TargetTypeWebhook:
		return fmt.Sprintf("https://example.com/webhook/%d", g.rand.Intn(1000))
	default:
		return fmt.Sprintf("test_value_%d", g.rand.Intn(1000))
	}
}

// getUnsupportedTargetTypes returns target types not supported by the platform
func (g *TestDataGenerator) getUnsupportedTargetTypes(supportedTypes []string) []string {
	allTypes := []string{
		target.TargetTypeEmail,
		target.TargetTypePhone,
		target.TargetTypeUser,
		target.TargetTypeGroup,
		target.TargetTypeChannel,
		target.TargetTypeWebhook,
	}

	var unsupported []string
	for _, t := range allTypes {
		supported := false
		for _, s := range supportedTypes {
			if t == s {
				supported = true
				break
			}
		}
		if !supported {
			unsupported = append(unsupported, t)
		}
	}
	return unsupported
}

// ConfigurableMockPlatform is an advanced mock platform for testing various scenarios
type ConfigurableMockPlatform struct {
	name               string
	capabilities       Capabilities
	healthy            bool
	closed             bool
	sendError          error
	sendDelay          time.Duration
	healthCheckError   error
	closeError         error
	sendResults        []*SendResult
	sendCallCount      int
	healthCallCount    int
	validationRules    func(target.Target) error
}

// NewConfigurableMockPlatform creates a new configurable mock platform
func NewConfigurableMockPlatform(name string) *ConfigurableMockPlatform {
	return &ConfigurableMockPlatform{
		name:    name,
		healthy: true,
		closed:  false,
		capabilities: Capabilities{
			Name:                 name,
			SupportedTargetTypes: []string{"webhook", "email"},
			SupportedFormats:     []string{"text", "markdown"},
			MaxMessageSize:       4096,
			SupportsScheduling:   false,
			SupportsAttachments:  false,
			SupportsMentions:     false,
			SupportsRichContent:  true,
		},
		validationRules: func(t target.Target) error {
			if t.Type == "" {
				return errors.New("target type cannot be empty")
			}
			if t.Value == "" {
				return errors.New("target value cannot be empty")
			}
			return nil
		},
	}
}

// Platform interface implementation

func (m *ConfigurableMockPlatform) Name() string {
	return m.name
}

func (m *ConfigurableMockPlatform) GetCapabilities() Capabilities {
	return m.capabilities
}

func (m *ConfigurableMockPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	m.sendCallCount++

	if m.closed {
		return nil, errors.New("platform is closed")
	}

	if m.sendError != nil {
		return nil, m.sendError
	}

	if msg == nil {
		return nil, errors.New("message cannot be nil")
	}

	// Simulate processing delay
	if m.sendDelay > 0 {
		select {
		case <-time.After(m.sendDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Return pre-configured results if available
	if len(m.sendResults) > 0 {
		results := make([]*SendResult, len(targets))
		for i := range targets {
			if i < len(m.sendResults) {
				results[i] = m.sendResults[i]
			} else {
				results[i] = &SendResult{
					Target:  targets[i],
					Success: true,
					MessageID: fmt.Sprintf("mock-%s-%d", msg.ID, i),
				}
			}
		}
		return results, nil
	}

	// Generate results based on validation
	results := make([]*SendResult, len(targets))
	for i, t := range targets {
		err := m.ValidateTarget(t)
		success := err == nil

		result := &SendResult{
			Target:  t,
			Success: success,
		}

		if success {
			result.MessageID = fmt.Sprintf("mock-%s-%d", msg.ID, i)
		} else {
			result.Error = err.Error()
		}

		results[i] = result
	}

	return results, nil
}

func (m *ConfigurableMockPlatform) ValidateTarget(target target.Target) error {
	if m.validationRules != nil {
		if err := m.validationRules(target); err != nil {
			return err
		}
	}

	// Check if target type is supported
	for _, supportedType := range m.capabilities.SupportedTargetTypes {
		if target.Type == supportedType {
			return nil
		}
	}

	return fmt.Errorf("unsupported target type: %s", target.Type)
}

func (m *ConfigurableMockPlatform) IsHealthy(ctx context.Context) error {
	m.healthCallCount++

	if m.closed {
		return errors.New("platform is closed")
	}

	if m.healthCheckError != nil {
		return m.healthCheckError
	}

	if !m.healthy {
		return errors.New("platform is unhealthy")
	}

	return nil
}

func (m *ConfigurableMockPlatform) Close() error {
	m.closed = true
	return m.closeError
}

// Configuration methods

// SetCapabilities updates the platform capabilities
func (m *ConfigurableMockPlatform) SetCapabilities(caps Capabilities) {
	caps.Name = m.name // Ensure name consistency
	m.capabilities = caps
}

// SetHealthy sets the health status
func (m *ConfigurableMockPlatform) SetHealthy(healthy bool) {
	m.healthy = healthy
}

// SetSendError configures the Send method to return an error
func (m *ConfigurableMockPlatform) SetSendError(err error) {
	m.sendError = err
}

// SetSendDelay configures a delay for Send method
func (m *ConfigurableMockPlatform) SetSendDelay(delay time.Duration) {
	m.sendDelay = delay
}

// SetHealthCheckError configures the IsHealthy method to return an error
func (m *ConfigurableMockPlatform) SetHealthCheckError(err error) {
	m.healthCheckError = err
}

// SetCloseError configures the Close method to return an error
func (m *ConfigurableMockPlatform) SetCloseError(err error) {
	m.closeError = err
}

// SetSendResults configures pre-defined send results
func (m *ConfigurableMockPlatform) SetSendResults(results []*SendResult) {
	m.sendResults = results
}

// SetValidationRules configures custom validation rules
func (m *ConfigurableMockPlatform) SetValidationRules(rules func(target.Target) error) {
	m.validationRules = rules
}

// GetCallCounts returns method call counts for testing
func (m *ConfigurableMockPlatform) GetCallCounts() (sendCalls, healthCalls int) {
	return m.sendCallCount, m.healthCallCount
}

// ResetCallCounts resets method call counters
func (m *ConfigurableMockPlatform) ResetCallCounts() {
	m.sendCallCount = 0
	m.healthCallCount = 0
}

// PerformanceTestPlatform is a mock platform designed for performance testing
type PerformanceTestPlatform struct {
	name               string
	capabilities       Capabilities
	processingTime     time.Duration
	memoryUsage        int64
	operationCounts    map[string]int64
}

// NewPerformanceTestPlatform creates a platform for performance testing
func NewPerformanceTestPlatform(name string, processingTime time.Duration) *PerformanceTestPlatform {
	return &PerformanceTestPlatform{
		name:           name,
		processingTime: processingTime,
		capabilities: Capabilities{
			Name:                 name,
			SupportedTargetTypes: []string{"webhook"},
			SupportedFormats:     []string{"text"},
			MaxMessageSize:       1024,
		},
		operationCounts: make(map[string]int64),
	}
}

func (p *PerformanceTestPlatform) Name() string {
	p.operationCounts["Name"]++
	return p.name
}

func (p *PerformanceTestPlatform) GetCapabilities() Capabilities {
	p.operationCounts["GetCapabilities"]++
	return p.capabilities
}

func (p *PerformanceTestPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	p.operationCounts["Send"]++

	// Simulate processing time
	if p.processingTime > 0 {
		select {
		case <-time.After(p.processingTime):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	results := make([]*SendResult, len(targets))
	for i, target := range targets {
		results[i] = &SendResult{
			Target:    target,
			Success:   true,
			MessageID: fmt.Sprintf("perf-%d", i),
		}
	}

	return results, nil
}

func (p *PerformanceTestPlatform) ValidateTarget(target target.Target) error {
	p.operationCounts["ValidateTarget"]++

	if target.Type != "webhook" {
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}
	if target.Value == "" {
		return errors.New("target value cannot be empty")
	}
	return nil
}

func (p *PerformanceTestPlatform) IsHealthy(ctx context.Context) error {
	p.operationCounts["IsHealthy"]++
	return nil
}

func (p *PerformanceTestPlatform) Close() error {
	p.operationCounts["Close"]++
	return nil
}

// GetOperationCounts returns the count of each operation performed
func (p *PerformanceTestPlatform) GetOperationCounts() map[string]int64 {
	counts := make(map[string]int64)
	for op, count := range p.operationCounts {
		counts[op] = count
	}
	return counts
}

// ResetOperationCounts resets all operation counters
func (p *PerformanceTestPlatform) ResetOperationCounts() {
	p.operationCounts = make(map[string]int64)
}

// PlatformTestSuite provides a comprehensive test suite for platform testing
type PlatformTestSuite struct {
	generator *TestDataGenerator
	logger    logger.Logger
}

// NewPlatformTestSuite creates a new test suite
func NewPlatformTestSuite() *PlatformTestSuite {
	return &PlatformTestSuite{
		generator: NewTestDataGenerator(),
		logger:    &noopTestLogger{},
	}
}

// CreateStandardContractTest creates test data for contract testing
// This method can be used to generate test components but does not return PlatformContractTest
// to avoid circular dependencies. Use this data to construct your own contract tests.
func (s *PlatformTestSuite) CreateTestData(capabilities Capabilities) (
	validTargets []target.Target,
	invalidTargets []target.Target,
	testMessage *message.Message,
) {
	validTargets = s.generator.GenerateValidTargets(capabilities, 3)
	invalidTargets = s.generator.GenerateInvalidTargets(capabilities, 4)
	testMessage = s.generator.GenerateTestMessage()
	return
}

// noopTestLogger is a no-operation logger for testing
type noopTestLogger struct{}

func (l *noopTestLogger) LogMode(level logger.LogLevel) logger.Logger { return l }
func (l *noopTestLogger) Debug(msg string, args ...any)               {}
func (l *noopTestLogger) Info(msg string, args ...any)                {}
func (l *noopTestLogger) Warn(msg string, args ...any)                {}
func (l *noopTestLogger) Error(msg string, args ...any)               {}

