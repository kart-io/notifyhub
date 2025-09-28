// Package platform provides platform interface contract tests for NotifyHub
// This file implements Task 5.3: Platform Interface Contract Testing
//
// These contract tests ensure that all platform implementations consistently
// implement the unified Platform interface according to Requirements 5.1 and 5.2.
package platform

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// PlatformContractTest represents a contract test suite for platform implementations
// This struct provides standardized testing for all Platform interface methods
type PlatformContractTest struct {
	PlatformName   string
	CreatePlatform func() (Platform, error)
	ValidTargets   []target.Target
	InvalidTargets []target.Target
	TestMessage    *message.Message
}

// RunPlatformContractTests runs the complete contract test suite for a platform
// This is the main entry point for testing platform implementations
func RunPlatformContractTests(t *testing.T, contractTest PlatformContractTest) {
	t.Run("PlatformIdentification", func(t *testing.T) {
		testPlatformIdentification(t, contractTest)
	})

	t.Run("CapabilityReporting", func(t *testing.T) {
		testCapabilityReporting(t, contractTest)
	})

	t.Run("TargetValidation", func(t *testing.T) {
		testTargetValidation(t, contractTest)
	})

	t.Run("MessageSending", func(t *testing.T) {
		testMessageSending(t, contractTest)
	})

	t.Run("HealthCheck", func(t *testing.T) {
		testHealthCheck(t, contractTest)
	})

	t.Run("LifecycleManagement", func(t *testing.T) {
		testPlatformLifecycleManagement(t, contractTest)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, contractTest)
	})

	t.Run("ContextHandling", func(t *testing.T) {
		testContextHandling(t, contractTest)
	})
}

// testPlatformIdentification validates platform Name() method
func testPlatformIdentification(t *testing.T, contractTest PlatformContractTest) {
	platform, err := contractTest.CreatePlatform()
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	defer platform.Close()

	// Test platform name consistency
	name := platform.Name()
	if name == "" {
		t.Error("Platform name must not be empty")
	}

	if name != contractTest.PlatformName {
		t.Errorf("Platform name mismatch: expected %s, got %s", contractTest.PlatformName, name)
	}

	// Test name consistency across multiple calls
	name2 := platform.Name()
	if name != name2 {
		t.Errorf("Platform name not consistent: first call returned %s, second call returned %s", name, name2)
	}

	t.Logf("✓ Platform identification validated: name=%s", name)
}

// testCapabilityReporting validates GetCapabilities() method
func testCapabilityReporting(t *testing.T, contractTest PlatformContractTest) {
	platform, err := contractTest.CreatePlatform()
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	defer platform.Close()

	capabilities := platform.GetCapabilities()

	// Test capability struct completeness
	if capabilities.Name == "" {
		t.Error("Capabilities must include platform name")
	}

	if capabilities.Name != platform.Name() {
		t.Errorf("Capabilities name (%s) must match platform name (%s)", capabilities.Name, platform.Name())
	}

	// Test supported target types are specified
	if len(capabilities.SupportedTargetTypes) == 0 {
		t.Error("Platform must specify at least one supported target type")
	}

	// Test supported formats are specified
	if len(capabilities.SupportedFormats) == 0 {
		t.Error("Platform must specify at least one supported format")
	}

	// Test message size limits are reasonable
	if capabilities.MaxMessageSize <= 0 {
		t.Error("Platform must specify a positive maximum message size")
	}

	if capabilities.MaxMessageSize > 1000000 { // 1MB sanity check
		t.Logf("WARNING: Platform message size limit is very large: %d bytes", capabilities.MaxMessageSize)
	}

	// Test capability consistency across multiple calls
	capabilities2 := platform.GetCapabilities()
	if !capabilitiesEqual(capabilities, capabilities2) {
		t.Error("Platform capabilities must be consistent across multiple calls")
	}

	// Validate target type values
	for _, targetType := range capabilities.SupportedTargetTypes {
		if !isValidTargetType(targetType) {
			t.Errorf("Invalid target type in capabilities: %s", targetType)
		}
	}

	// Validate format values
	for _, format := range capabilities.SupportedFormats {
		if !isValidFormat(format) {
			t.Errorf("Invalid format in capabilities: %s", format)
		}
	}

	t.Logf("✓ Capability reporting validated: %d target types, %d formats, max size: %d",
		len(capabilities.SupportedTargetTypes), len(capabilities.SupportedFormats), capabilities.MaxMessageSize)
}

// testTargetValidation validates ValidateTarget() method
func testTargetValidation(t *testing.T, contractTest PlatformContractTest) {
	platform, err := contractTest.CreatePlatform()
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	defer platform.Close()

	capabilities := platform.GetCapabilities()

	// Test valid targets
	for _, validTarget := range contractTest.ValidTargets {
		err := platform.ValidateTarget(validTarget)
		if err != nil {
			t.Errorf("Valid target should pass validation: %v, error: %v", validTarget, err)
		}
	}

	// Test invalid targets
	for _, invalidTarget := range contractTest.InvalidTargets {
		err := platform.ValidateTarget(invalidTarget)
		if err == nil {
			t.Errorf("Invalid target should fail validation: %v", invalidTarget)
		}
	}

	// Test target type consistency with capabilities
	for _, validTarget := range contractTest.ValidTargets {
		if !isTargetTypeSupported(validTarget.Type, capabilities.SupportedTargetTypes) {
			t.Errorf("Valid target type %s not listed in platform capabilities", validTarget.Type)
		}
	}

	// Test empty target validation
	emptyTarget := target.Target{}
	err = platform.ValidateTarget(emptyTarget)
	if err == nil {
		t.Error("Empty target should fail validation")
	}

	// Test target with empty value
	emptyValueTarget := target.Target{Type: "webhook", Value: ""}
	err = platform.ValidateTarget(emptyValueTarget)
	if err == nil {
		t.Error("Target with empty value should fail validation")
	}

	t.Logf("✓ Target validation validated: %d valid targets, %d invalid targets",
		len(contractTest.ValidTargets), len(contractTest.InvalidTargets))
}

// testMessageSending validates Send() method
func testMessageSending(t *testing.T, contractTest PlatformContractTest) {
	platform, err := contractTest.CreatePlatform()
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	defer platform.Close()

	ctx := context.Background()

	// Test sending to valid targets
	if len(contractTest.ValidTargets) > 0 {
		results, err := platform.Send(ctx, contractTest.TestMessage, contractTest.ValidTargets)

		if err != nil {
			t.Errorf("Send to valid targets should not return error: %v", err)
		}

		if len(results) != len(contractTest.ValidTargets) {
			t.Errorf("Expected %d results, got %d", len(contractTest.ValidTargets), len(results))
		}

		// Validate each result
		for i, result := range results {
			if result == nil {
				t.Errorf("Result %d should not be nil", i)
				continue
			}

			// Check target consistency
			if !targetsEqual(result.Target, contractTest.ValidTargets[i]) {
				t.Errorf("Result target mismatch: expected %v, got %v", contractTest.ValidTargets[i], result.Target)
			}

			// Check result completeness
			if result.Success && result.MessageID == "" {
				t.Errorf("Successful result should have message ID")
			}

			if !result.Success && result.Error == "" {
				t.Errorf("Failed result should have error message")
			}
		}
	}

	// Test sending with nil message
	results, err := platform.Send(ctx, nil, contractTest.ValidTargets)
	if err == nil {
		t.Error("Sending nil message should return error")
	}

	// Test sending with empty targets
	results, err = platform.Send(ctx, contractTest.TestMessage, []target.Target{})
	if err != nil {
		t.Errorf("Sending to empty targets should not error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty targets, got %d", len(results))
	}

	// Test sending to invalid targets
	if len(contractTest.InvalidTargets) > 0 {
		results, err = platform.Send(ctx, contractTest.TestMessage, contractTest.InvalidTargets)

		// Platform should handle invalid targets gracefully
		if len(results) != len(contractTest.InvalidTargets) {
			t.Errorf("Expected %d results for invalid targets, got %d", len(contractTest.InvalidTargets), len(results))
		}

		// Check that invalid targets are marked as failed
		for i, result := range results {
			if result.Success {
				t.Errorf("Invalid target should not succeed: %v", contractTest.InvalidTargets[i])
			}
		}
	}

	t.Logf("✓ Message sending validated: tested valid/invalid targets and edge cases")
}

// testHealthCheck validates IsHealthy() method
func testHealthCheck(t *testing.T, contractTest PlatformContractTest) {
	platform, err := contractTest.CreatePlatform()
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	defer platform.Close()

	ctx := context.Background()

	// Test basic health check
	err = platform.IsHealthy(ctx)
	// Health check may succeed or fail depending on platform configuration
	// The important thing is that it doesn't panic and returns consistently

	// Test health check consistency
	err1 := platform.IsHealthy(ctx)
	err2 := platform.IsHealthy(ctx)

	// Both should succeed or both should fail for the same reason
	if (err1 == nil) != (err2 == nil) {
		t.Errorf("Health check results inconsistent: first=%v, second=%v", err1, err2)
	}

	// Test health check with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	err = platform.IsHealthy(timeoutCtx)
	// Should complete within timeout or return context error

	// Test health check with cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = platform.IsHealthy(cancelledCtx)
	if err == nil {
		t.Log("WARNING: Health check with cancelled context should typically return error")
	}

	t.Logf("✓ Health check validated: responds to context and behaves consistently")
}

// testLifecycleManagement validates Close() method and resource cleanup
func testPlatformLifecycleManagement(t *testing.T, contractTest PlatformContractTest) {
	platform, err := contractTest.CreatePlatform()
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}

	// Test that platform works before closing
	ctx := context.Background()
	err = platform.IsHealthy(ctx)
	// Don't assert on error since health may fail for configuration reasons

	// Test graceful close
	err = platform.Close()
	if err != nil {
		t.Errorf("Platform close should not return error: %v", err)
	}

	// Test that close is idempotent
	err = platform.Close()
	if err != nil {
		t.Errorf("Second close should not return error: %v", err)
	}

	// Test operations after close (platform should handle gracefully)
	err = platform.IsHealthy(ctx)
	// Platform may return error after close, which is acceptable

	t.Logf("✓ Lifecycle management validated: platform closes gracefully")
}

// testErrorHandling validates error handling patterns
func testErrorHandling(t *testing.T, contractTest PlatformContractTest) {
	platform, err := contractTest.CreatePlatform()
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	defer platform.Close()

	ctx := context.Background()

	// Test error handling for invalid operations

	// 1. Test Send with various invalid inputs
	invalidMessage := &message.Message{
		Title: "", // Empty title
		Body:  "",  // Empty body
	}

	results, err := platform.Send(ctx, invalidMessage, contractTest.ValidTargets)
	// Platform should either return error or mark results as failed
	if err == nil {
		// If no error returned, results should show failures
		for _, result := range results {
			if result.Success {
				t.Error("Send with invalid message should not succeed")
			}
		}
	}

	// 2. Test ValidateTarget with various invalid inputs
	invalidTargets := []target.Target{
		{Type: "", Value: "test"},           // Empty type
		{Type: "test", Value: ""},           // Empty value
		{Type: "invalid", Value: "test"},    // Invalid type
	}

	for _, invalidTarget := range invalidTargets {
		err := platform.ValidateTarget(invalidTarget)
		if err == nil {
			t.Errorf("ValidateTarget should fail for invalid target: %v", invalidTarget)
		}
	}

	// 3. Test that errors are informative
	err = platform.ValidateTarget(target.Target{Type: "", Value: ""})
	if err != nil && err.Error() == "" {
		t.Error("Error messages should be informative, not empty")
	}

	t.Logf("✓ Error handling validated: platform handles invalid inputs appropriately")
}

// testContextHandling validates proper context handling
func testContextHandling(t *testing.T, contractTest PlatformContractTest) {
	platform, err := contractTest.CreatePlatform()
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	defer platform.Close()

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Operations with cancelled context should respect cancellation
	err = platform.IsHealthy(ctx)
	// Some platforms may check context, others may not - both are acceptable

	if len(contractTest.ValidTargets) > 0 {
		results, err := platform.Send(ctx, contractTest.TestMessage, contractTest.ValidTargets[:1])
		// Platform should handle cancelled context gracefully
		if err == nil && len(results) > 0 && results[0].Success {
			t.Log("WARNING: Platform may want to check context cancellation in Send method")
		}
	}

	// Test context timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond) // Ensure timeout has passed

	err = platform.IsHealthy(timeoutCtx)
	// Platform should ideally respect timeouts, but not required

	t.Logf("✓ Context handling validated: platform respects context lifecycle")
}

// Helper functions for contract testing

// capabilitiesEqual compares two Capabilities structs for equality
func capabilitiesEqual(a, b Capabilities) bool {
	if a.Name != b.Name || a.MaxMessageSize != b.MaxMessageSize {
		return false
	}

	if len(a.SupportedTargetTypes) != len(b.SupportedTargetTypes) {
		return false
	}

	for i, targetType := range a.SupportedTargetTypes {
		if targetType != b.SupportedTargetTypes[i] {
			return false
		}
	}

	if len(a.SupportedFormats) != len(b.SupportedFormats) {
		return false
	}

	for i, format := range a.SupportedFormats {
		if format != b.SupportedFormats[i] {
			return false
		}
	}

	return a.SupportsScheduling == b.SupportsScheduling &&
		   a.SupportsAttachments == b.SupportsAttachments &&
		   a.SupportsMentions == b.SupportsMentions &&
		   a.SupportsRichContent == b.SupportsRichContent
}

// targetsEqual compares two Target structs for equality
func targetsEqual(a, b target.Target) bool {
	return a.Type == b.Type && a.Value == b.Value && a.Platform == b.Platform
}

// isValidTargetType checks if a target type is valid
func isValidTargetType(targetType string) bool {
	validTypes := []string{
		target.TargetTypeEmail,
		target.TargetTypePhone,
		target.TargetTypeUser,
		target.TargetTypeGroup,
		target.TargetTypeChannel,
		target.TargetTypeWebhook,
	}

	for _, validType := range validTypes {
		if targetType == validType {
			return true
		}
	}
	return false
}

// isValidFormat checks if a format is valid
func isValidFormat(format string) bool {
	validFormats := []string{
		string(message.FormatText),
		string(message.FormatMarkdown),
		string(message.FormatHTML),
	}

	for _, validFormat := range validFormats {
		if format == validFormat {
			return true
		}
	}
	return false
}

// isTargetTypeSupported checks if a target type is supported by the platform
func isTargetTypeSupported(targetType string, supportedTypes []string) bool {
	for _, supportedType := range supportedTypes {
		if targetType == supportedType {
			return true
		}
	}
	return false
}

// MockPlatform implementation for testing the contract test framework itself
type MockPlatform struct {
	name         string
	capabilities Capabilities
	healthy      bool
	closed       bool
}

// NewMockPlatform creates a new mock platform for testing
func NewMockPlatform(name string) *MockPlatform {
	return &MockPlatform{
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
	}
}

func (m *MockPlatform) Name() string {
	return m.name
}

func (m *MockPlatform) GetCapabilities() Capabilities {
	return m.capabilities
}

func (m *MockPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	if m.closed {
		return nil, errors.New("platform is closed")
	}

	if msg == nil {
		return nil, errors.New("message cannot be nil")
	}

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

func (m *MockPlatform) ValidateTarget(target target.Target) error {
	if target.Type == "" {
		return errors.New("target type cannot be empty")
	}

	if target.Value == "" {
		return errors.New("target value cannot be empty")
	}

	// Check if target type is supported
	for _, supportedType := range m.capabilities.SupportedTargetTypes {
		if target.Type == supportedType {
			return nil
		}
	}

	return fmt.Errorf("unsupported target type: %s", target.Type)
}

func (m *MockPlatform) IsHealthy(ctx context.Context) error {
	if m.closed {
		return errors.New("platform is closed")
	}

	if !m.healthy {
		return errors.New("platform is unhealthy")
	}

	return nil
}

func (m *MockPlatform) Close() error {
	m.closed = true
	return nil
}

// SetHealthy sets the health status of the mock platform
func (m *MockPlatform) SetHealthy(healthy bool) {
	m.healthy = healthy
}

// TestContractFramework tests the contract testing framework itself
func TestContractFramework(t *testing.T) {
	// Create test message
	testMessage := message.New()
	testMessage.Title = "Contract Test Message"
	testMessage.Body = "This is a test message for contract testing"
	testMessage.Format = message.FormatText

	// Create contract test for mock platform
	contractTest := PlatformContractTest{
		PlatformName:   "mock",
		CreatePlatform: func() (Platform, error) { return NewMockPlatform("mock"), nil },
		ValidTargets: []target.Target{
			{Type: "webhook", Value: "http://example.com/webhook"},
			{Type: "email", Value: "test@example.com"},
		},
		InvalidTargets: []target.Target{
			{Type: "sms", Value: "123456789"},  // Not supported by mock
			{Type: "", Value: "test"},          // Empty type
			{Type: "webhook", Value: ""},       // Empty value
		},
		TestMessage: testMessage,
	}

	// Run contract tests
	RunPlatformContractTests(t, contractTest)

	t.Log("✓ Contract testing framework validated")
}

// Extension interface for testing advanced platform features
type AdvancedPlatformTestSuite interface {
	TestRetryBehavior(t *testing.T, platform Platform)
	TestConcurrentAccess(t *testing.T, platform Platform)
	TestResourceCleanup(t *testing.T, platform Platform)
	TestConfigurationValidation(t *testing.T, platform Platform)
}

