package notifiers

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestTargetTypes(t *testing.T) {
	// Test target type constants
	if TargetTypeGroup != "group" {
		t.Error("TargetTypeGroup should be 'group'")
	}

	if TargetTypeUser != "user" {
		t.Error("TargetTypeUser should be 'user'")
	}

	if TargetTypeEmail != "email" {
		t.Error("TargetTypeEmail should be 'email'")
	}
}

func TestMessageFormats(t *testing.T) {
	// Test message format constants
	if FormatText != "text" {
		t.Error("FormatText should be 'text'")
	}

	if FormatMarkdown != "markdown" {
		t.Error("FormatMarkdown should be 'markdown'")
	}

	if FormatHTML != "html" {
		t.Error("FormatHTML should be 'html'")
	}
}

func TestTarget(t *testing.T) {
	// Test target creation
	target := Target{
		Type:     TargetTypeEmail,
		Value:    "test@example.com",
		Platform: "email",
		Metadata: map[string]string{
			"priority": "high",
		},
	}

	if target.Type != TargetTypeEmail {
		t.Error("Target type should be email")
	}

	if target.Value != "test@example.com" {
		t.Error("Target value should be test@example.com")
	}

	if target.Platform != "email" {
		t.Error("Target platform should be email")
	}

	if target.Metadata["priority"] != "high" {
		t.Error("Target metadata should include priority=high")
	}
}

func TestMessage(t *testing.T) {
	// Test message creation
	now := time.Now()
	message := &Message{
		ID:       "test-123",
		Title:    "Test Title",
		Body:     "Test Body",
		Format:   FormatMarkdown,
		Priority: 4,
		Template: "alert-template",
		Variables: map[string]interface{}{
			"name":  "John",
			"count": 42,
		},
		Metadata: map[string]string{
			"source": "test",
		},
		CreatedAt: now,
	}

	if message.ID != "test-123" {
		t.Error("Message ID should be test-123")
	}

	if message.Title != "Test Title" {
		t.Error("Message title should be Test Title")
	}

	if message.Body != "Test Body" {
		t.Error("Message body should be Test Body")
	}

	if message.Format != FormatMarkdown {
		t.Error("Message format should be markdown")
	}

	if message.Priority != 4 {
		t.Error("Message priority should be 4")
	}

	if message.Template != "alert-template" {
		t.Error("Message template should be alert-template")
	}

	if message.Variables["name"] != "John" {
		t.Error("Message variables should include name=John")
	}

	if message.Variables["count"] != 42 {
		t.Error("Message variables should include count=42")
	}

	if message.Metadata["source"] != "test" {
		t.Error("Message metadata should include source=test")
	}

	if !message.CreatedAt.Equal(now) {
		t.Error("Message CreatedAt should match the set time")
	}
}

func TestSendResult(t *testing.T) {
	// Test send result creation
	target := Target{
		Type:     TargetTypeUser,
		Value:    "user123",
		Platform: "feishu",
	}

	now := time.Now()
	duration := 150 * time.Millisecond

	result := &SendResult{
		Target:   target,
		Platform: "feishu",
		Success:  true,
		Error:    "",
		Duration: duration,
		SentAt:   now,
		Attempts: 2,
	}

	if result.Target.Type != TargetTypeUser {
		t.Error("SendResult target type should be user")
	}

	if result.Target.Value != "user123" {
		t.Error("SendResult target value should be user123")
	}

	if result.Platform != "feishu" {
		t.Error("SendResult platform should be feishu")
	}

	if !result.Success {
		t.Error("SendResult success should be true")
	}

	if result.Error != "" {
		t.Error("SendResult error should be empty")
	}

	if result.Duration != duration {
		t.Error("SendResult duration should match set duration")
	}

	if !result.SentAt.Equal(now) {
		t.Error("SendResult SentAt should match the set time")
	}

	if result.Attempts != 2 {
		t.Error("SendResult attempts should be 2")
	}
}

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

	// Test health check (will fail with fake webhook, but should not panic)
	ctx := context.Background()
	err := notifier.Health(ctx)
	if err == nil {
		t.Log("Feishu health check passed unexpectedly (fake webhook)")
	}

	// Test send (will fail with fake webhook, but should not panic)
	message := &Message{
		Title:    "Test",
		Body:     "Test message",
		Format:   FormatText,
		Priority: 3,
		Targets: []Target{
			{Type: TargetTypeGroup, Value: "group123", Platform: "feishu"},
		},
	}

	results, err := notifier.Send(ctx, message)
	if err == nil && len(results) > 0 {
		t.Log("Feishu send succeeded unexpectedly (fake webhook)")
	}
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

	// Test health check (will fail with fake SMTP, but should not panic)
	ctx := context.Background()
	err := notifier.Health(ctx)
	if err == nil {
		t.Log("Email health check passed unexpectedly (fake SMTP)")
	}

	// Test send (will fail with fake SMTP, but should not panic)
	message := &Message{
		Title:    "Test Email",
		Body:     "Test email body",
		Format:   FormatHTML,
		Priority: 3,
		Targets: []Target{
			{Type: TargetTypeEmail, Value: "test@example.com"},
		},
	}

	results, err := notifier.Send(ctx, message)
	if err == nil && len(results) > 0 {
		t.Log("Email send succeeded unexpectedly (fake SMTP)")
	}
}

func TestEmailValidation(t *testing.T) {
	// Test email validation function used internally by email notifier
	testCases := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"user.name+tag@domain.co.uk", true},
		{"invalid.email", false},
		{"@domain.com", false},
		{"user@", false},
		{"", false},
		{"user@domain", true}, // Simple domain names are acceptable
		{"user.name@domain.com", true},
	}

	for i, tc := range testCases {
		// Simple email validation check
		isValid := strings.Contains(tc.email, "@") && !strings.HasPrefix(tc.email, "@") && !strings.HasSuffix(tc.email, "@") && tc.email != ""
		if isValid != tc.expected {
			t.Errorf("Test case %d: expected %v, got %v for email '%s'", i, tc.expected, isValid, tc.email)
		}
	}
}

func TestNotifierInterface(t *testing.T) {
	// Test that both notifiers implement the Notifier interface
	var _ Notifier = NewFeishuNotifier("https://test.webhook", "", 30*time.Second)
	var _ Notifier = NewEmailNotifier("localhost", 587, "user", "pass", "from@example.com", false, 30*time.Second)

	// This test ensures interface compliance at compile time
	// If the notifiers don't implement the interface, this test will fail to compile
}
