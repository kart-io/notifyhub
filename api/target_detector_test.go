package api

import (
	"testing"

	"github.com/kart-io/notifyhub/core/message"
)

func TestTargetDetector_EmailDetection(t *testing.T) {
	detector := NewTargetDetector()

	tests := []struct {
		input    string
		expected message.TargetType
		platform string
	}{
		{"user@example.com", message.TargetTypeEmail, "email"},
		{"test.user@company.org", message.TargetTypeEmail, "email"},
		{"simple@domain.co.uk", message.TargetTypeEmail, "email"},
	}

	for _, test := range tests {
		target := detector.DetectTarget(test.input)
		if target.Type != test.expected {
			t.Errorf("Expected type %v for %s, got %v", test.expected, test.input, target.Type)
		}
		if target.Platform != test.platform {
			t.Errorf("Expected platform %s for %s, got %s", test.platform, test.input, target.Platform)
		}
		if target.Value != test.input {
			t.Errorf("Expected value %s for %s, got %s", test.input, test.input, target.Value)
		}
	}
}

func TestTargetDetector_UserMentionDetection(t *testing.T) {
	detector := NewTargetDetector()

	tests := []struct {
		input        string
		expectedType message.TargetType
		expectedValue string
	}{
		{"@john", message.TargetTypeUser, "john"},
		{"@jane.doe", message.TargetTypeUser, "jane.doe"},
		{"@user123", message.TargetTypeUser, "user123"},
	}

	for _, test := range tests {
		target := detector.DetectTarget(test.input)
		if target.Type != test.expectedType {
			t.Errorf("Expected type %v for %s, got %v", test.expectedType, test.input, target.Type)
		}
		if target.Value != test.expectedValue {
			t.Errorf("Expected value %s for %s, got %s", test.expectedValue, test.input, target.Value)
		}
	}
}

func TestTargetDetector_ChannelDetection(t *testing.T) {
	detector := NewTargetDetector()

	tests := []struct {
		input        string
		expectedType message.TargetType
		expectedValue string
	}{
		{"#general", message.TargetTypeChannel, "general"},
		{"#dev-team", message.TargetTypeChannel, "dev-team"},
		{"#alerts", message.TargetTypeChannel, "alerts"},
	}

	for _, test := range tests {
		target := detector.DetectTarget(test.input)
		if target.Type != test.expectedType {
			t.Errorf("Expected type %v for %s, got %v", test.expectedType, test.input, target.Type)
		}
		if target.Value != test.expectedValue {
			t.Errorf("Expected value %s for %s, got %s", test.expectedValue, test.input, target.Value)
		}
	}
}

func TestSlackTargetDetector(t *testing.T) {
	detector := NewSlackTargetDetector()

	tests := []struct {
		input        string
		expectedType message.TargetType
		expectedValue string
		platform     string
	}{
		{"#general", message.TargetTypeChannel, "general", "slack"},
		{"@john", message.TargetTypeUser, "john", "slack"},
		{"random", message.TargetTypeChannel, "random", "slack"}, // default
	}

	for _, test := range tests {
		target := detector.DetectTarget(test.input)
		if target.Type != test.expectedType {
			t.Errorf("Expected type %v for %s, got %v", test.expectedType, test.input, target.Type)
		}
		if target.Value != test.expectedValue {
			t.Errorf("Expected value %s for %s, got %s", test.expectedValue, test.input, target.Value)
		}
		if target.Platform != test.platform {
			t.Errorf("Expected platform %s for %s, got %s", test.platform, test.input, target.Platform)
		}
	}
}

func TestFeishuTargetDetector(t *testing.T) {
	detector := NewFeishuTargetDetector()

	tests := []struct {
		input        string
		expectedType message.TargetType
		expectedValue string
		platform     string
	}{
		{"#general", message.TargetTypeChannel, "general", "feishu"},
		{"@john", message.TargetTypeUser, "john", "feishu"},
		{"random", message.TargetTypeGroup, "random", "feishu"}, // default for Feishu
	}

	for _, test := range tests {
		target := detector.DetectTarget(test.input)
		if target.Type != test.expectedType {
			t.Errorf("Expected type %v for %s, got %v", test.expectedType, test.input, target.Type)
		}
		if target.Value != test.expectedValue {
			t.Errorf("Expected value %s for %s, got %s", test.expectedValue, test.input, target.Value)
		}
		if target.Platform != test.platform {
			t.Errorf("Expected platform %s for %s, got %s", test.platform, test.input, target.Platform)
		}
	}
}

func TestTargetDetector_MultipeTargets(t *testing.T) {
	detector := NewTargetDetector()

	inputs := []string{"user@example.com", "@john", "#general", "plaintext"}
	targets := detector.DetectTargets(inputs...)

	if len(targets) != 4 {
		t.Errorf("Expected 4 targets, got %d", len(targets))
	}

	// Verify email detection
	if targets[0].Type != message.TargetTypeEmail {
		t.Errorf("Expected first target to be email, got %v", targets[0].Type)
	}

	// Verify user mention detection
	if targets[1].Type != message.TargetTypeUser {
		t.Errorf("Expected second target to be user, got %v", targets[1].Type)
	}

	// Verify channel detection
	if targets[2].Type != message.TargetTypeChannel {
		t.Errorf("Expected third target to be channel, got %v", targets[2].Type)
	}

	// Verify default detection
	if targets[3].Type != message.TargetTypeUser {
		t.Errorf("Expected fourth target to be user (default), got %v", targets[3].Type)
	}
}

func TestTargetDetector_SetPlatformForTargets(t *testing.T) {
	detector := NewTargetDetector()

	// Create targets without platform
	targets := []message.Target{
		{Type: message.TargetTypeUser, Value: "john", Platform: ""},
		{Type: message.TargetTypeChannel, Value: "general", Platform: "slack"}, // already has platform
	}

	// Set platform for targets without one
	updatedTargets := detector.SetPlatformForTargets(targets, "default-platform")

	if updatedTargets[0].Platform != "default-platform" {
		t.Errorf("Expected first target platform to be 'default-platform', got '%s'", updatedTargets[0].Platform)
	}

	if updatedTargets[1].Platform != "slack" {
		t.Errorf("Expected second target platform to remain 'slack', got '%s'", updatedTargets[1].Platform)
	}
}

func TestTargetDetector_CustomStrategy(t *testing.T) {
	detector := NewTargetDetector()

	// Add custom strategy
	customStrategy := &testCustomStrategy{}
	detector.AddStrategy(customStrategy)

	target := detector.DetectTarget("CUSTOM:test")
	if target.Type != message.TargetTypeWebhook {
		t.Errorf("Expected custom strategy to detect CUSTOM prefix, got type %v", target.Type)
	}
	if target.Value != "test" {
		t.Errorf("Expected value 'test', got '%s'", target.Value)
	}
}

// testCustomStrategy for testing custom strategy functionality
type testCustomStrategy struct{}

func (s *testCustomStrategy) CanHandle(input string) bool {
	return len(input) > 7 && input[:7] == "CUSTOM:"
}

func (s *testCustomStrategy) Detect(input string) (message.TargetType, string, string) {
	return message.TargetTypeWebhook, input[7:], "custom"
}

func BenchmarkTargetDetector_DetectEmail(b *testing.B) {
	detector := NewTargetDetector()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		detector.DetectTarget("user@example.com")
	}
}

func BenchmarkTargetDetector_DetectMultiple(b *testing.B) {
	detector := NewTargetDetector()
	inputs := []string{"user@example.com", "@john", "#general", "plaintext"}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		detector.DetectTargets(inputs...)
	}
}