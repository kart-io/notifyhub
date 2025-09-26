package message

import (
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

func TestNew(t *testing.T) {
	msg := New()

	// Check basic fields are set
	if msg.ID == "" {
		t.Error("Expected ID to be set")
	}

	if msg.Format != FormatText {
		t.Errorf("Expected format to be %v, got %v", FormatText, msg.Format)
	}

	if msg.Priority != PriorityNormal {
		t.Errorf("Expected priority to be %v, got %v", PriorityNormal, msg.Priority)
	}

	if msg.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	// Check maps are initialized
	if msg.Metadata == nil {
		t.Error("Expected Metadata to be initialized")
	}

	if msg.Variables == nil {
		t.Error("Expected Variables to be initialized")
	}

	if msg.PlatformData == nil {
		t.Error("Expected PlatformData to be initialized")
	}
}

func TestNewWithID(t *testing.T) {
	expectedID := "test-id-123"
	msg := NewWithID(expectedID)

	if msg.ID != expectedID {
		t.Errorf("Expected ID to be %s, got %s", expectedID, msg.ID)
	}

	// Other fields should still be set correctly
	if msg.Format != FormatText {
		t.Errorf("Expected format to be %v, got %v", FormatText, msg.Format)
	}

	if msg.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		format   Format
		expected string
		valid    bool
	}{
		{FormatText, "text", true},
		{FormatMarkdown, "markdown", true},
		{FormatHTML, "html", true},
		{Format("invalid"), "invalid", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			// Test String method
			if tt.format.String() != tt.expected {
				t.Errorf("Expected String() to return %s, got %s", tt.expected, tt.format.String())
			}

			// Test IsValid method
			if tt.format.IsValid() != tt.valid {
				t.Errorf("Expected IsValid() to return %v, got %v", tt.valid, tt.format.IsValid())
			}
		})
	}
}

func TestPriority(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityNormal, "normal"},
		{PriorityHigh, "high"},
		{PriorityUrgent, "urgent"},
		{Priority(99), "normal"}, // Invalid priority defaults to normal
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.priority.String() != tt.expected {
				t.Errorf("Expected String() to return %s, got %s", tt.expected, tt.priority.String())
			}
		})
	}
}

func TestMessageMethods(t *testing.T) {
	msg := New()

	// Test AddTarget
	emailTarget := target.NewEmailTarget("test@example.com")
	msg.AddTarget(emailTarget)

	if len(msg.Targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(msg.Targets))
	}

	if msg.Targets[0].Value != "test@example.com" {
		t.Errorf("Expected target value to be test@example.com, got %s", msg.Targets[0].Value)
	}

	// Test AddTargets
	phoneTarget := target.NewPhoneTarget("+1234567890")
	groupTarget := target.NewTarget(target.TargetTypeGroup, "group-123", target.PlatformFeishu)
	msg.AddTargets(phoneTarget, groupTarget)

	if len(msg.Targets) != 3 {
		t.Errorf("Expected 3 targets, got %d", len(msg.Targets))
	}

	// Test SetMetadata
	msg.SetMetadata("key1", "value1")
	if msg.Metadata["key1"] != "value1" {
		t.Errorf("Expected metadata key1 to be value1, got %v", msg.Metadata["key1"])
	}

	// Test GetMetadata returns a copy
	metadata := msg.GetMetadata()
	metadata["key2"] = "value2"
	if _, exists := msg.Metadata["key2"]; exists {
		t.Error("Expected GetMetadata to return a copy, but original was modified")
	}

	// Test SetVariable
	msg.SetVariable("var1", "varvalue1")
	if msg.Variables["var1"] != "varvalue1" {
		t.Errorf("Expected variable var1 to be varvalue1, got %v", msg.Variables["var1"])
	}

	// Test GetVariables returns a copy
	variables := msg.GetVariables()
	variables["var2"] = "varvalue2"
	if _, exists := msg.Variables["var2"]; exists {
		t.Error("Expected GetVariables to return a copy, but original was modified")
	}

	// Test SetPlatformData
	msg.SetPlatformData("platform_key", "platform_value")
	if msg.PlatformData["platform_key"] != "platform_value" {
		t.Errorf("Expected platform data platform_key to be platform_value, got %v", msg.PlatformData["platform_key"])
	}

	// Test GetPlatformData returns a copy
	platformData := msg.GetPlatformData()
	platformData["platform_key2"] = "platform_value2"
	if _, exists := msg.PlatformData["platform_key2"]; exists {
		t.Error("Expected GetPlatformData to return a copy, but original was modified")
	}
}

func TestMessageBuilder(t *testing.T) {
	// Test NewMessage builder
	msg := NewMessage("Test Title").
		WithBody("Test body content").
		WithFormat(FormatMarkdown).
		WithPriority(PriorityHigh).
		WithMetadata("source", "unit_test").
		WithVariable("user", "John").
		ToEmail("test@example.com").
		Build()

	if msg.Title != "Test Title" {
		t.Errorf("Expected title to be 'Test Title', got %s", msg.Title)
	}

	if msg.Body != "Test body content" {
		t.Errorf("Expected body to be 'Test body content', got %s", msg.Body)
	}

	if msg.Format != FormatMarkdown {
		t.Errorf("Expected format to be %v, got %v", FormatMarkdown, msg.Format)
	}

	if msg.Priority != PriorityHigh {
		t.Errorf("Expected priority to be %v, got %v", PriorityHigh, msg.Priority)
	}

	if msg.Metadata["source"] != "unit_test" {
		t.Errorf("Expected metadata source to be 'unit_test', got %v", msg.Metadata["source"])
	}

	if msg.Variables["user"] != "John" {
		t.Errorf("Expected variable user to be 'John', got %v", msg.Variables["user"])
	}

	if len(msg.Targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(msg.Targets))
	}

	if msg.Targets[0].Value != "test@example.com" {
		t.Errorf("Expected email target, got %s", msg.Targets[0].Value)
	}
}

func TestAlertAndUrgentBuilders(t *testing.T) {
	// Test NewAlert
	alert := NewAlert("System Alert").Build()
	if alert.Priority != PriorityHigh {
		t.Errorf("Expected alert priority to be %v, got %v", PriorityHigh, alert.Priority)
	}

	if alert.Title != "System Alert" {
		t.Errorf("Expected alert title to be 'System Alert', got %s", alert.Title)
	}

	// Test NewUrgent
	urgent := NewUrgent("Critical Issue").Build()
	if urgent.Priority != PriorityUrgent {
		t.Errorf("Expected urgent priority to be %v, got %v", PriorityUrgent, urgent.Priority)
	}

	if urgent.Title != "Critical Issue" {
		t.Errorf("Expected urgent title to be 'Critical Issue', got %s", urgent.Title)
	}
}

func TestBuilderFormatMethods(t *testing.T) {
	// Test WithMarkdown
	msg := NewMessage().WithMarkdown("**Bold text**").Build()
	if msg.Format != FormatMarkdown {
		t.Errorf("Expected format to be %v, got %v", FormatMarkdown, msg.Format)
	}
	if msg.Body != "**Bold text**" {
		t.Errorf("Expected body to be '**Bold text**', got %s", msg.Body)
	}

	// Test WithHTML
	msg = NewMessage().WithHTML("<b>Bold text</b>").Build()
	if msg.Format != FormatHTML {
		t.Errorf("Expected format to be %v, got %v", FormatHTML, msg.Format)
	}
	if msg.Body != "<b>Bold text</b>" {
		t.Errorf("Expected body to be '<b>Bold text</b>', got %s", msg.Body)
	}
}

func TestBuilderScheduling(t *testing.T) {
	// Test ScheduleAt
	future := time.Now().Add(1 * time.Hour)
	msg := NewMessage("Scheduled Message").ScheduleAt(future).Build()

	if msg.ScheduledAt == nil {
		t.Error("Expected ScheduledAt to be set")
	}

	if !msg.ScheduledAt.Equal(future) {
		t.Errorf("Expected ScheduledAt to be %v, got %v", future, *msg.ScheduledAt)
	}

	// Test ScheduleIn
	duration := 30 * time.Minute
	before := time.Now()
	msg = NewMessage("Delayed Message").ScheduleIn(duration).Build()
	after := time.Now()

	if msg.ScheduledAt == nil {
		t.Error("Expected ScheduledAt to be set")
	}

	expectedMin := before.Add(duration)
	expectedMax := after.Add(duration)

	if msg.ScheduledAt.Before(expectedMin) || msg.ScheduledAt.After(expectedMax) {
		t.Errorf("Expected ScheduledAt to be around %v, got %v", expectedMin, *msg.ScheduledAt)
	}
}

func TestBuilderTargetMethods(t *testing.T) {
	builder := NewMessage("Multi-target Message")

	// Test various target methods
	msg := builder.
		ToEmail("user@example.com").
		ToPhone("+1234567890").
		ToFeishuUser("user123").
		ToFeishuGroup("group456").
		ToWebhook("https://webhook.example.com").
		Build()

	if len(msg.Targets) != 5 {
		t.Errorf("Expected 5 targets, got %d", len(msg.Targets))
	}

	// Verify target types
	expectedTypes := []string{
		target.TargetTypeEmail,
		target.TargetTypePhone,
		target.TargetTypeUser,
		target.TargetTypeGroup,
		target.TargetTypeWebhook,
	}

	for i, expectedType := range expectedTypes {
		if msg.Targets[i].Type != expectedType {
			t.Errorf("Expected target %d type to be %s, got %s", i, expectedType, msg.Targets[i].Type)
		}
	}
}

func TestBuilderPlatformSpecificMethods(t *testing.T) {
	// Test Feishu-specific methods
	card := map[string]interface{}{
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"content": "Test Card",
				"tag":     "plain_text",
			},
		},
	}

	msg := NewMessage("Feishu Test").
		WithFeishuCard(card).
		WithFeishuMentions("user1", "user2").
		WithFeishuMentionAll().
		Build()

	if msg.PlatformData["feishu_card"] == nil {
		t.Error("Expected feishu_card to be set")
	}

	if msg.PlatformData["feishu_mentions"] == nil {
		t.Error("Expected feishu_mentions to be set")
	}

	if msg.PlatformData["feishu_mention_all"] != true {
		t.Error("Expected feishu_mention_all to be true")
	}

	// Test Email-specific methods
	msg = NewMessage("Email Test").
		WithEmailCC("cc1@example.com", "cc2@example.com").
		WithEmailBCC("bcc@example.com").
		WithEmailPriority("high").
		WithEmailAttachments([]string{"file1.pdf", "file2.doc"}).
		Build()

	if msg.PlatformData["email_cc"] == nil {
		t.Error("Expected email_cc to be set")
	}

	if msg.PlatformData["email_bcc"] == nil {
		t.Error("Expected email_bcc to be set")
	}

	if msg.PlatformData["email_priority"] != "high" {
		t.Error("Expected email_priority to be 'high'")
	}

	if msg.PlatformData["email_attachments"] == nil {
		t.Error("Expected email_attachments to be set")
	}

	// Test SMS-specific methods
	msg = NewMessage("SMS Test").
		WithSMSTemplate("template123").
		WithSMSVariables(map[string]interface{}{
			"code": "123456",
			"name": "John",
		}).
		Build()

	if msg.PlatformData["sms_template"] != "template123" {
		t.Error("Expected sms_template to be 'template123'")
	}

	if msg.PlatformData["sms_variables"] == nil {
		t.Error("Expected sms_variables to be set")
	}
}

func TestMessageValidation(t *testing.T) {
	// Test empty message
	msg := &Message{}
	if msg.GetMetadata() == nil {
		t.Log("Empty message metadata is nil as expected")
	}

	// Test nil maps don't cause panics in methods
	msg.SetMetadata("key", "value")
	msg.SetVariable("var", "value")
	msg.SetPlatformData("platform", "data")

	// Verify maps were created
	if msg.Metadata == nil {
		t.Error("Expected Metadata to be created")
	}
	if msg.Variables == nil {
		t.Error("Expected Variables to be created")
	}
	if msg.PlatformData == nil {
		t.Error("Expected PlatformData to be created")
	}
}
