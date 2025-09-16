package notifiers

import (
	"context"
	"testing"
	"time"
)

func TestFeishuNotifier(t *testing.T) {
	// Create Feishu notifier
	notifier := NewFeishuNotifier(
		"https://open.feishu.cn/open-apis/bot/v2/hook/test-webhook",
		"test-secret",
		30*time.Second,
	)

	// Test notifier name
	if notifier.Name() != "feishu" {
		t.Error("Feishu notifier name should be 'feishu'")
	}

	// Test target support
	testCases := []struct {
		target   Target
		expected bool
	}{
		{Target{Type: TargetTypeGroup, Platform: "feishu"}, true},
		{Target{Type: TargetTypeUser, Platform: "feishu"}, true},
		{Target{Type: TargetTypeEmail, Platform: "feishu"}, false},
		{Target{Type: TargetTypeGroup, Platform: "email"}, false},
		{Target{Type: TargetTypeGroup}, true}, // Platform can be empty, defaults to supporting
	}

	for i, tc := range testCases {
		supported := notifier.SupportsTarget(tc.target)
		if supported != tc.expected {
			t.Errorf("Test case %d: expected %v, got %v for target %+v", i, tc.expected, supported, tc.target)
		}
	}

	// Test that notifier doesn't panic with invalid configuration
	ctx := context.Background()
	_ = notifier.Health(ctx) // Should not panic

	message := &Message{
		Title:    "Test",
		Body:     "Test message",
		Format:   FormatText,
		Priority: 3,
		Targets: []Target{
			{Type: TargetTypeGroup, Value: "group123", Platform: "feishu"},
		},
	}

	_, _ = notifier.Send(ctx, message) // Should not panic
}

func TestEmailNotifier(t *testing.T) {
	// Create Email notifier
	notifier := NewEmailNotifier(
		"localhost",
		587,
		"test@example.com",
		"password",
		"sender@example.com",
		false,
		30*time.Second,
	)

	// Test notifier name
	if notifier.Name() != "email" {
		t.Error("Email notifier name should be 'email'")
	}

	// Test target support
	testCases := []struct {
		target   Target
		expected bool
	}{
		{Target{Type: TargetTypeEmail}, true},
		{Target{Type: TargetTypeUser, Value: "user@example.com"}, true}, // Email format in value
		{Target{Type: TargetTypeGroup}, false},
		{Target{Type: TargetTypeUser, Value: "user123"}, false}, // Non-email format
	}

	for i, tc := range testCases {
		supported := notifier.SupportsTarget(tc.target)
		if supported != tc.expected {
			t.Errorf("Test case %d: expected %v, got %v for target %+v", i, tc.expected, supported, tc.target)
		}
	}

	// Test that notifier doesn't panic with invalid configuration
	ctx := context.Background()
	_ = notifier.Health(ctx) // Should not panic

	message := &Message{
		Title:    "Test Email",
		Body:     "Test email body",
		Format:   FormatHTML,
		Priority: 3,
		Targets: []Target{
			{Type: TargetTypeEmail, Value: "test@example.com"},
		},
	}

	_, _ = notifier.Send(ctx, message) // Should not panic
}


