// Package platform provides example usage of platform interface contract tests
// This file demonstrates how to use the contract testing framework for various platform types
package platform

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// TestExamplePlatformContracts demonstrates contract testing for different platform types
func TestExamplePlatformContracts(t *testing.T) {
	t.Run("EmailPlatform", func(t *testing.T) {
		testEmailPlatformContract(t)
	})

	t.Run("WebhookPlatform", func(t *testing.T) {
		testWebhookPlatformContract(t)
	})

	t.Run("SMSPlatform", func(t *testing.T) {
		testSMSPlatformContract(t)
	})

	t.Run("AdvancedPlatform", func(t *testing.T) {
		testAdvancedPlatformContract(t)
	})
}

// testEmailPlatformContract shows how to test an email platform
func testEmailPlatformContract(t *testing.T) {
	// Create test message
	testMessage := message.New()
	testMessage.Title = "Email Contract Test"
	testMessage.Body = "Testing email platform contract compliance"
	testMessage.Format = message.FormatHTML // Email platforms often support HTML

	// Define email platform contract test
	contractTest := PlatformContractTest{
		PlatformName: "email",
		CreatePlatform: func() (Platform, error) {
			return NewExampleEmailPlatform(), nil
		},
		ValidTargets: []target.Target{
			{Type: "email", Value: "test@example.com", Platform: "email"},
			{Type: "email", Value: "user@domain.org", Platform: "email"},
			{Type: "email", Value: "admin@company.com", Platform: "auto"},
		},
		InvalidTargets: []target.Target{
			{Type: "phone", Value: "123456789", Platform: "email"},      // Wrong type
			{Type: "webhook", Value: "http://example.com", Platform: "email"}, // Wrong type
			{Type: "email", Value: "invalid-email", Platform: "email"},  // Invalid email
			{Type: "email", Value: "", Platform: "email"},               // Empty value
			{Type: "", Value: "test@example.com", Platform: "email"},    // Empty type
		},
		TestMessage: testMessage,
	}

	// Run contract tests
	RunPlatformContractTests(t, contractTest)
}

// testWebhookPlatformContract shows how to test a webhook platform
func testWebhookPlatformContract(t *testing.T) {
	// Create test message
	testMessage := message.New()
	testMessage.Title = "Webhook Contract Test"
	testMessage.Body = "Testing webhook platform contract compliance"
	testMessage.Format = message.FormatMarkdown

	// Define webhook platform contract test
	contractTest := PlatformContractTest{
		PlatformName: "webhook",
		CreatePlatform: func() (Platform, error) {
			return NewExampleWebhookPlatform(), nil
		},
		ValidTargets: []target.Target{
			{Type: "webhook", Value: "https://api.example.com/webhook", Platform: "webhook"},
			{Type: "webhook", Value: "http://localhost:8080/notify", Platform: "webhook"},
			{Type: "webhook", Value: "https://hooks.slack.com/services/xxx", Platform: "auto"},
		},
		InvalidTargets: []target.Target{
			{Type: "email", Value: "test@example.com", Platform: "webhook"}, // Wrong type
			{Type: "user", Value: "user123", Platform: "webhook"},           // Wrong type
			{Type: "webhook", Value: "invalid-url", Platform: "webhook"},    // Invalid URL
			{Type: "webhook", Value: "", Platform: "webhook"},               // Empty value
			{Type: "", Value: "http://example.com", Platform: "webhook"},    // Empty type
		},
		TestMessage: testMessage,
	}

	// Run contract tests
	RunPlatformContractTests(t, contractTest)
}

// testSMSPlatformContract shows how to test an SMS platform
func testSMSPlatformContract(t *testing.T) {
	// Create test message
	testMessage := message.New()
	testMessage.Title = "SMS Contract Test"
	testMessage.Body = "Testing SMS platform contract compliance"
	testMessage.Format = message.FormatText // SMS is typically text-only

	// Define SMS platform contract test
	contractTest := PlatformContractTest{
		PlatformName: "sms",
		CreatePlatform: func() (Platform, error) {
			return NewExampleSMSPlatform(), nil
		},
		ValidTargets: []target.Target{
			{Type: "phone", Value: "+1234567890", Platform: "sms"},
			{Type: "phone", Value: "+44123456789", Platform: "sms"},
			{Type: "phone", Value: "123-456-7890", Platform: "auto"},
		},
		InvalidTargets: []target.Target{
			{Type: "email", Value: "test@example.com", Platform: "sms"},  // Wrong type
			{Type: "webhook", Value: "http://example.com", Platform: "sms"}, // Wrong type
			{Type: "phone", Value: "invalid-phone", Platform: "sms"},     // Invalid phone
			{Type: "phone", Value: "", Platform: "sms"},                  // Empty value
			{Type: "", Value: "+1234567890", Platform: "sms"},            // Empty type
		},
		TestMessage: testMessage,
	}

	// Run contract tests
	RunPlatformContractTests(t, contractTest)
}

// testAdvancedPlatformContract shows how to test a platform with advanced features
func testAdvancedPlatformContract(t *testing.T) {
	// Create test message
	testMessage := message.New()
	testMessage.Title = "Advanced Contract Test"
	testMessage.Body = "Testing advanced platform features"
	testMessage.Format = message.FormatMarkdown
	testMessage.Priority = message.PriorityHigh

	// Define advanced platform contract test
	contractTest := PlatformContractTest{
		PlatformName: "advanced",
		CreatePlatform: func() (Platform, error) {
			return NewExampleAdvancedPlatform(), nil
		},
		ValidTargets: []target.Target{
			{Type: "user", Value: "user123", Platform: "advanced"},
			{Type: "group", Value: "group456", Platform: "advanced"},
			{Type: "channel", Value: "channel789", Platform: "advanced"},
			{Type: "email", Value: "test@example.com", Platform: "advanced"},
		},
		InvalidTargets: []target.Target{
			{Type: "webhook", Value: "http://example.com", Platform: "advanced"}, // Not supported
			{Type: "phone", Value: "+1234567890", Platform: "advanced"},          // Not supported
			{Type: "user", Value: "", Platform: "advanced"},                      // Empty value
		},
		TestMessage: testMessage,
	}

	// Run contract tests
	RunPlatformContractTests(t, contractTest)
}

// Example platform implementations for demonstration

// ExampleEmailPlatform demonstrates an email platform implementation
type ExampleEmailPlatform struct {
	name   string
	closed bool
}

func NewExampleEmailPlatform() *ExampleEmailPlatform {
	return &ExampleEmailPlatform{
		name:   "email",
		closed: false,
	}
}

func (p *ExampleEmailPlatform) Name() string {
	return p.name
}

func (p *ExampleEmailPlatform) GetCapabilities() Capabilities {
	return Capabilities{
		Name:                 "email",
		SupportedTargetTypes: []string{"email"},
		SupportedFormats:     []string{"text", "html"},
		MaxMessageSize:       10240, // 10KB
		SupportsScheduling:   true,
		SupportsAttachments:  true,
		SupportsMentions:     false,
		SupportsRichContent:  true,
		RequiredSettings:     []string{"smtp_server", "username", "password"},
	}
}

func (p *ExampleEmailPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	if p.closed {
		return nil, errors.New("platform is closed")
	}

	if msg == nil {
		return nil, errors.New("message cannot be nil")
	}

	results := make([]*SendResult, len(targets))
	for i, target := range targets {
		err := p.ValidateTarget(target)
		if err != nil {
			results[i] = &SendResult{
				Target:  target,
				Success: false,
				Error:   err.Error(),
			}
		} else {
			// Simulate successful email send
			results[i] = &SendResult{
				Target:    target,
				Success:   true,
				MessageID: "email-" + msg.ID + "-" + string(rune(i)),
				Response:  "Email sent successfully",
			}
		}
	}

	return results, nil
}

func (p *ExampleEmailPlatform) ValidateTarget(target target.Target) error {
	if target.Type != "email" {
		return errors.New("email platform only supports email targets")
	}

	if target.Value == "" {
		return errors.New("email address cannot be empty")
	}

	// Simple email validation
	if len(target.Value) < 5 || !contains(target.Value, "@") {
		return errors.New("invalid email address format")
	}

	return nil
}

func (p *ExampleEmailPlatform) IsHealthy(ctx context.Context) error {
	if p.closed {
		return errors.New("platform is closed")
	}
	// Simulate health check
	return nil
}

func (p *ExampleEmailPlatform) Close() error {
	p.closed = true
	return nil
}

// ExampleWebhookPlatform demonstrates a webhook platform implementation
type ExampleWebhookPlatform struct {
	name   string
	closed bool
}

func NewExampleWebhookPlatform() *ExampleWebhookPlatform {
	return &ExampleWebhookPlatform{
		name:   "webhook",
		closed: false,
	}
}

func (p *ExampleWebhookPlatform) Name() string {
	return p.name
}

func (p *ExampleWebhookPlatform) GetCapabilities() Capabilities {
	return Capabilities{
		Name:                 "webhook",
		SupportedTargetTypes: []string{"webhook"},
		SupportedFormats:     []string{"text", "markdown", "html"},
		MaxMessageSize:       65536, // 64KB
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		SupportsMentions:     false,
		SupportsRichContent:  true,
		RequiredSettings:     []string{"webhook_url"},
	}
}

func (p *ExampleWebhookPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	if p.closed {
		return nil, errors.New("platform is closed")
	}

	if msg == nil {
		return nil, errors.New("message cannot be nil")
	}

	results := make([]*SendResult, len(targets))
	for i, target := range targets {
		err := p.ValidateTarget(target)
		if err != nil {
			results[i] = &SendResult{
				Target:  target,
				Success: false,
				Error:   err.Error(),
			}
		} else {
			// Simulate webhook send
			results[i] = &SendResult{
				Target:    target,
				Success:   true,
				MessageID: "webhook-" + msg.ID + "-" + string(rune(i)),
				Response:  "Webhook delivered successfully",
			}
		}
	}

	return results, nil
}

func (p *ExampleWebhookPlatform) ValidateTarget(target target.Target) error {
	if target.Type != "webhook" {
		return errors.New("webhook platform only supports webhook targets")
	}

	if target.Value == "" {
		return errors.New("webhook URL cannot be empty")
	}

	// Simple URL validation
	if len(target.Value) < 7 || (!startsWith(target.Value, "http://") && !startsWith(target.Value, "https://")) {
		return errors.New("invalid webhook URL format")
	}

	return nil
}

func (p *ExampleWebhookPlatform) IsHealthy(ctx context.Context) error {
	if p.closed {
		return errors.New("platform is closed")
	}
	return nil
}

func (p *ExampleWebhookPlatform) Close() error {
	p.closed = true
	return nil
}

// ExampleSMSPlatform demonstrates an SMS platform implementation
type ExampleSMSPlatform struct {
	name   string
	closed bool
}

func NewExampleSMSPlatform() *ExampleSMSPlatform {
	return &ExampleSMSPlatform{
		name:   "sms",
		closed: false,
	}
}

func (p *ExampleSMSPlatform) Name() string {
	return p.name
}

func (p *ExampleSMSPlatform) GetCapabilities() Capabilities {
	return Capabilities{
		Name:                 "sms",
		SupportedTargetTypes: []string{"phone"},
		SupportedFormats:     []string{"text"},
		MaxMessageSize:       160, // SMS character limit
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		SupportsMentions:     false,
		SupportsRichContent:  false,
		RequiredSettings:     []string{"api_key", "sender_number"},
	}
}

func (p *ExampleSMSPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	if p.closed {
		return nil, errors.New("platform is closed")
	}

	if msg == nil {
		return nil, errors.New("message cannot be nil")
	}

	results := make([]*SendResult, len(targets))
	for i, target := range targets {
		err := p.ValidateTarget(target)
		if err != nil {
			results[i] = &SendResult{
				Target:  target,
				Success: false,
				Error:   err.Error(),
			}
		} else {
			// Simulate SMS send
			results[i] = &SendResult{
				Target:    target,
				Success:   true,
				MessageID: "sms-" + msg.ID + "-" + string(rune(i)),
				Response:  "SMS sent successfully",
			}
		}
	}

	return results, nil
}

func (p *ExampleSMSPlatform) ValidateTarget(target target.Target) error {
	if target.Type != "phone" {
		return errors.New("SMS platform only supports phone targets")
	}

	if target.Value == "" {
		return errors.New("phone number cannot be empty")
	}

	// Simple phone validation
	if len(target.Value) < 10 {
		return errors.New("phone number too short")
	}

	return nil
}

func (p *ExampleSMSPlatform) IsHealthy(ctx context.Context) error {
	if p.closed {
		return errors.New("platform is closed")
	}
	return nil
}

func (p *ExampleSMSPlatform) Close() error {
	p.closed = true
	return nil
}

// ExampleAdvancedPlatform demonstrates a platform with advanced features
type ExampleAdvancedPlatform struct {
	name   string
	closed bool
}

func NewExampleAdvancedPlatform() *ExampleAdvancedPlatform {
	return &ExampleAdvancedPlatform{
		name:   "advanced",
		closed: false,
	}
}

func (p *ExampleAdvancedPlatform) Name() string {
	return p.name
}

func (p *ExampleAdvancedPlatform) GetCapabilities() Capabilities {
	return Capabilities{
		Name:                 "advanced",
		SupportedTargetTypes: []string{"user", "group", "channel", "email"},
		SupportedFormats:     []string{"text", "markdown", "html"},
		MaxMessageSize:       32768, // 32KB
		SupportsScheduling:   true,
		SupportsAttachments:  true,
		SupportsMentions:     true,
		SupportsRichContent:  true,
		RequiredSettings:     []string{"api_token", "base_url"},
	}
}

func (p *ExampleAdvancedPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	if p.closed {
		return nil, errors.New("platform is closed")
	}

	if msg == nil {
		return nil, errors.New("message cannot be nil")
	}

	// Simulate processing time for advanced features
	select {
	case <-time.After(10 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	results := make([]*SendResult, len(targets))
	for i, target := range targets {
		err := p.ValidateTarget(target)
		if err != nil {
			results[i] = &SendResult{
				Target:  target,
				Success: false,
				Error:   err.Error(),
			}
		} else {
			// Simulate advanced platform send
			results[i] = &SendResult{
				Target:    target,
				Success:   true,
				MessageID: "advanced-" + msg.ID + "-" + string(rune(i)),
				Response:  "Message delivered via advanced platform",
				Metadata: map[string]interface{}{
					"delivery_time": time.Now(),
					"target_type":   target.Type,
					"format":        msg.Format,
				},
			}
		}
	}

	return results, nil
}

func (p *ExampleAdvancedPlatform) ValidateTarget(target target.Target) error {
	supportedTypes := []string{"user", "group", "channel", "email"}

	found := false
	for _, supportedType := range supportedTypes {
		if target.Type == supportedType {
			found = true
			break
		}
	}

	if !found {
		return errors.New("unsupported target type for advanced platform")
	}

	if target.Value == "" {
		return errors.New("target value cannot be empty")
	}

	// Type-specific validation
	switch target.Type {
	case "email":
		if !contains(target.Value, "@") {
			return errors.New("invalid email format")
		}
	case "user", "group", "channel":
		if len(target.Value) < 3 {
			return errors.New("identifier too short")
		}
	}

	return nil
}

func (p *ExampleAdvancedPlatform) IsHealthy(ctx context.Context) error {
	if p.closed {
		return errors.New("platform is closed")
	}

	// Simulate health check with timeout
	select {
	case <-time.After(5 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *ExampleAdvancedPlatform) Close() error {
	p.closed = true
	return nil
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(substr) > 0 && findSubstring(s, substr) >= 0))
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestPlatformTestUtilities validates the test utilities themselves
func TestPlatformTestUtilities(t *testing.T) {
	// Test data generator
	generator := NewTestDataGenerator()

	// Test message generation
	testMessage := generator.GenerateTestMessage()
	if testMessage.Title == "" {
		t.Error("Generated message should have title")
	}
	if testMessage.Body == "" {
		t.Error("Generated message should have body")
	}

	// Test target generation
	capabilities := Capabilities{
		Name:                 "test",
		SupportedTargetTypes: []string{"email", "webhook"},
		SupportedFormats:     []string{"text"},
		MaxMessageSize:       1024,
	}

	validTargets := generator.GenerateValidTargets(capabilities, 3)
	if len(validTargets) != 3 {
		t.Errorf("Expected 3 valid targets, got %d", len(validTargets))
	}

	invalidTargets := generator.GenerateInvalidTargets(capabilities, 2)
	if len(invalidTargets) != 2 {
		t.Errorf("Expected 2 invalid targets, got %d", len(invalidTargets))
	}

	// Test configurable mock platform
	mock := NewConfigurableMockPlatform("test-mock")
	mock.SetHealthy(false)

	err := mock.IsHealthy(context.Background())
	if err == nil {
		t.Error("Mock platform should be unhealthy")
	}

	mock.SetHealthy(true)
	err = mock.IsHealthy(context.Background())
	if err != nil {
		t.Error("Mock platform should be healthy")
	}

	t.Log("✓ Platform test utilities validated")
}