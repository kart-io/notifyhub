package message

import (
	"fmt"
	"strings"
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
	msg, err := NewMessage("Test Title").
		WithBody("Test body content").
		WithFormat(FormatMarkdown).
		WithPriority(PriorityHigh).
		WithMetadata("source", "unit_test").
		WithVariable("user", "John").
		ToEmail("test@example.com").
		Build()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

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
	alert, err := NewAlert("System Alert").
		WithBody("Alert body content").
		ToEmail("admin@example.com").
		Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if alert.Priority != PriorityHigh {
		t.Errorf("Expected alert priority to be %v, got %v", PriorityHigh, alert.Priority)
	}

	if alert.Title != "System Alert" {
		t.Errorf("Expected alert title to be 'System Alert', got %s", alert.Title)
	}

	// Test NewUrgent
	urgent, err := NewUrgent("Critical Issue").
		WithBody("Urgent issue description").
		ToEmail("emergency@example.com").
		Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if urgent.Priority != PriorityUrgent {
		t.Errorf("Expected urgent priority to be %v, got %v", PriorityUrgent, urgent.Priority)
	}

	if urgent.Title != "Critical Issue" {
		t.Errorf("Expected urgent title to be 'Critical Issue', got %s", urgent.Title)
	}
}

func TestBuilderFormatMethods(t *testing.T) {
	// Test WithMarkdown
	msg, err := NewMessage("Test Title").
		WithMarkdown("**Bold text**").
		ToEmail("test@example.com").
		Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if msg.Format != FormatMarkdown {
		t.Errorf("Expected format to be %v, got %v", FormatMarkdown, msg.Format)
	}
	if msg.Body != "**Bold text**" {
		t.Errorf("Expected body to be '**Bold text**', got %s", msg.Body)
	}

	// Test WithHTML
	msg, err = NewMessage("HTML Test").
		WithHTML("<b>Bold text</b>").
		ToEmail("test@example.com").
		Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
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
	msg, err := NewMessage("Scheduled Message").
		WithBody("This message is scheduled").
		ToEmail("scheduler@example.com").
		ScheduleAt(future).
		Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if msg.ScheduledAt == nil {
		t.Error("Expected ScheduledAt to be set")
	}

	if !msg.ScheduledAt.Equal(future) {
		t.Errorf("Expected ScheduledAt to be %v, got %v", future, *msg.ScheduledAt)
	}

	// Test ScheduleIn
	duration := 30 * time.Minute
	before := time.Now()
	msg, err = NewMessage("Delayed Message").
		WithBody("This message is delayed").
		ToEmail("delayed@example.com").
		ScheduleIn(duration).
		Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
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
	builder := NewMessage("Multi-target Message").WithBody("Testing multiple targets")

	// Test various target methods
	msg, err := builder.
		ToEmail("user@example.com").
		ToPhone("+1234567890").
		ToFeishuUser("user123").
		ToFeishuGroup("group456").
		ToWebhook("https://webhook.example.com").
		Build()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

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

	msg, err := NewMessage("Feishu Test").
		WithBody("Testing Feishu features").
		ToEmail("feishu@example.com").
		WithFeishuCard(card).
		WithFeishuMentions("user1", "user2").
		WithFeishuMentionAll().
		Build()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

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
	msg, err = NewMessage("Email Test").
		WithBody("Testing email features").
		ToEmail("primary@example.com").
		WithEmailCC("cc1@example.com", "cc2@example.com").
		WithEmailBCC("bcc@example.com").
		WithEmailPriority("high").
		WithEmailAttachments([]string{"file1.pdf", "file2.doc"}).
		Build()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

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
	msg, err = NewMessage("SMS Test").
		WithBody("Testing SMS features").
		ToPhone("+1234567890").
		WithSMSTemplate("template123").
		WithSMSVariables(map[string]interface{}{
			"code": "123456",
			"name": "John",
		}).
		Build()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

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

// TestBoundaryConditions tests various boundary conditions and edge cases
func TestBoundaryConditions(t *testing.T) {
	t.Run("Empty message validation", func(t *testing.T) {
		// Test completely empty message
		_, err := NewMessage("").Build()
		if err == nil {
			t.Error("Expected error for message with empty title")
		}

		// Test message with only title
		_, err = NewMessage("Title").Build()
		if err == nil {
			t.Error("Expected error for message without body and targets")
		}

		// Test message with title and body but no targets
		_, err = NewMessage("Title").WithBody("Body").Build()
		if err == nil {
			t.Error("Expected error for message without targets")
		}
	})

	t.Run("Content length limits", func(t *testing.T) {
		// Test title length limit (MaxTitleLength = 200)
		longTitle := strings.Repeat("A", 201)
		_, err := NewMessage(longTitle).
			WithBody("Test body").
			ToEmail("test@example.com").
			Build()
		if err == nil {
			t.Error("Expected error for title exceeding 200 characters")
		}

		// Test exactly at limit
		exactTitle := strings.Repeat("A", 200)
		_, err = NewMessage(exactTitle).
			WithBody("Test body").
			ToEmail("test@example.com").
			Build()
		if err != nil {
			t.Errorf("Expected no error for title at exactly 200 characters, got: %v", err)
		}

		// Test body length limit (MaxBodyLength = 4096)
		longBody := strings.Repeat("B", 4097)
		_, err = NewMessage("Test Title").
			WithBody(longBody).
			ToEmail("test@example.com").
			Build()
		if err == nil {
			t.Error("Expected error for body exceeding 4096 characters")
		}

		// Test exactly at limit
		exactBody := strings.Repeat("B", 4096)
		_, err = NewMessage("Test Title").
			WithBody(exactBody).
			ToEmail("test@example.com").
			Build()
		if err != nil {
			t.Errorf("Expected no error for body at exactly 4096 characters, got: %v", err)
		}
	})

	t.Run("Invalid format values", func(t *testing.T) {
		// Test invalid format
		_, err := NewMessage("Test Title").
			WithBody("Test body").
			WithFormat(Format("invalid_format")).
			ToEmail("test@example.com").
			Build()
		if err == nil {
			t.Error("Expected error for invalid format")
		}

		// Test Format.IsValid() method with invalid values
		invalidFormats := []Format{
			Format("xml"),
			Format("json"),
			Format("plain"),
			Format(""),
			Format("TEXT"), // case sensitive
		}
		for _, format := range invalidFormats {
			if format.IsValid() {
				t.Errorf("Expected format '%s' to be invalid", format)
			}
		}

		// Test valid formats
		validFormats := []Format{FormatText, FormatMarkdown, FormatHTML}
		for _, format := range validFormats {
			if !format.IsValid() {
				t.Errorf("Expected format '%s' to be valid", format)
			}
		}
	})

	t.Run("Invalid priority values", func(t *testing.T) {
		// Test priority out of range - below minimum
		_, err := NewMessage("Test Title").
			WithBody("Test body").
			WithPriority(Priority(-1)).
			ToEmail("test@example.com").
			Build()
		if err == nil {
			t.Error("Expected error for priority below minimum (0)")
		}

		// Test priority out of range - above maximum
		_, err = NewMessage("Test Title").
			WithBody("Test body").
			WithPriority(Priority(4)).
			ToEmail("test@example.com").
			Build()
		if err == nil {
			t.Error("Expected error for priority above maximum (3)")
		}

		// Test boundary values
		_, err = NewMessage("Test Title").
			WithBody("Test body").
			WithPriority(PriorityLow). // 0
			ToEmail("test@example.com").
			Build()
		if err != nil {
			t.Errorf("Expected no error for minimum priority, got: %v", err)
		}

		_, err = NewMessage("Test Title").
			WithBody("Test body").
			WithPriority(PriorityUrgent). // 3
			ToEmail("test@example.com").
			Build()
		if err != nil {
			t.Errorf("Expected no error for maximum priority, got: %v", err)
		}
	})

	t.Run("Target count limits", func(t *testing.T) {
		// Test no targets
		_, err := NewMessage("Test Title").
			WithBody("Test body").
			Build()
		if err == nil {
			t.Error("Expected error for message with no targets")
		}

		// Test maximum targets (MaxTargetCount = 100)
		builder := NewMessage("Test Title").WithBody("Test body")
		for i := 0; i < 100; i++ {
			builder = builder.ToEmail(fmt.Sprintf("test%d@example.com", i))
		}
		msg, err := builder.Build()
		if err != nil {
			t.Errorf("Expected no error for 100 targets, got: %v", err)
		}
		if len(msg.Targets) != 100 {
			t.Errorf("Expected 100 targets, got %d", len(msg.Targets))
		}

		// Test exceeding maximum targets
		builder = NewMessage("Test Title").WithBody("Test body")
		for i := 0; i < 101; i++ {
			builder = builder.ToEmail(fmt.Sprintf("test%d@example.com", i))
		}
		_, err = builder.Build()
		if err == nil {
			t.Error("Expected error for exceeding maximum target count (101)")
		}
	})

	t.Run("Null character validation", func(t *testing.T) {
		// Test null character in title
		_, err := NewMessage("Test\x00Title").
			WithBody("Test body").
			ToEmail("test@example.com").
			Build()
		if err == nil {
			t.Error("Expected error for null character in title")
		}

		// Test null character in body
		_, err = NewMessage("Test Title").
			WithBody("Test\x00body").
			ToEmail("test@example.com").
			Build()
		if err == nil {
			t.Error("Expected error for null character in body")
		}
	})
}

// TestBuilderChainingAdvanced tests comprehensive builder pattern scenarios
func TestBuilderChainingAdvanced(t *testing.T) {
	t.Run("Complex chain validation", func(t *testing.T) {
		// Test complex chaining with all methods
		msg, err := NewMessage("Complex Test").
			WithBody("This is a complex test message").
			WithFormat(FormatMarkdown).
			WithPriority(PriorityHigh).
			WithMetadata("source", "test_suite").
			WithMetadata("environment", "development").
			WithVariable("user_name", "John Doe").
			WithVariable("action", "login").
			ToEmail("primary@example.com").
			ToPhone("+1234567890").
			ToFeishuUser("user123").
			WithFeishuMentions("user1", "user2").
			WithEmailCC("cc@example.com").
			ScheduleIn(1 * time.Hour).
			Build()

		if err != nil {
			t.Fatalf("Expected no error for complex chain, got: %v", err)
		}

		// Verify all properties were set correctly
		if msg.Title != "Complex Test" {
			t.Errorf("Expected title 'Complex Test', got %s", msg.Title)
		}
		if msg.Format != FormatMarkdown {
			t.Errorf("Expected format markdown, got %s", msg.Format)
		}
		if msg.Priority != PriorityHigh {
			t.Errorf("Expected priority high, got %s", msg.Priority.String())
		}
		if len(msg.Targets) != 3 {
			t.Errorf("Expected 3 targets, got %d", len(msg.Targets))
		}
		if len(msg.Metadata) != 2 {
			t.Errorf("Expected 2 metadata entries, got %d", len(msg.Metadata))
		}
		if len(msg.Variables) != 2 {
			t.Errorf("Expected 2 variables, got %d", len(msg.Variables))
		}
		if msg.ScheduledAt == nil {
			t.Error("Expected ScheduledAt to be set")
		}
	})

	t.Run("Error accumulation in chain", func(t *testing.T) {
		// Test that errors accumulate during chaining
		builder := NewMessage(""). // Empty title - error
			WithBody("").               // Empty body - error
			WithFormat(Format("invalid")). // Invalid format - error
			WithPriority(Priority(-1))     // Invalid priority - error

		// Check that errors accumulated
		if !builder.HasErrors() {
			t.Error("Expected builder to have errors")
		}

		errors := builder.GetErrors()
		if len(errors) < 3 {
			t.Errorf("Expected at least 3 errors, got %d", len(errors))
		}

		// Try to build - should fail
		_, err := builder.Build()
		if err == nil {
			t.Error("Expected Build() to fail with accumulated errors")
		}
	})

	t.Run("Error clearing and recovery", func(t *testing.T) {
		// Start with errors
		builder := NewMessage(""). // Empty title - error
			WithBody("")              // Empty body - error

		if !builder.HasErrors() {
			t.Error("Expected builder to have errors")
		}

		// Clear errors and fix
		builder.ClearErrors().
			WithTitle("Fixed Title").
			WithBody("Fixed Body").
			ToEmail("test@example.com")

		// Should now build successfully
		msg, err := builder.Build()
		if err != nil {
			t.Errorf("Expected no error after clearing and fixing, got: %v", err)
		}
		if msg.Title != "Fixed Title" {
			t.Errorf("Expected title 'Fixed Title', got %s", msg.Title)
		}
	})

	t.Run("Build vs BuildUnsafe comparison", func(t *testing.T) {
		// Create builder with validation errors
		builder := NewMessage(""). // Empty title
			WithBody("")              // Empty body

		// Build() should fail
		_, err := builder.Build()
		if err == nil {
			t.Error("Expected Build() to fail with validation errors")
		}

		// BuildUnsafe() should succeed but return invalid message
		msg := builder.BuildUnsafe()
		if msg == nil {
			t.Error("Expected BuildUnsafe() to return a message")
		}
		if msg.Title != "" {
			t.Errorf("Expected empty title in unsafe build, got %s", msg.Title)
		}
		if msg.ID == "" {
			t.Error("Expected BuildUnsafe() to generate an ID")
		}
	})

	t.Run("Validation method testing", func(t *testing.T) {
		// Valid builder
		validBuilder := NewMessage("Valid Title").
			WithBody("Valid body").
			ToEmail("valid@example.com")

		err := validBuilder.Validate()
		if err != nil {
			t.Errorf("Expected valid builder to pass validation, got: %v", err)
		}

		// Invalid builder
		invalidBuilder := NewMessage(""). // Empty title
			WithBody("Valid body").
			ToEmail("valid@example.com")

		err = invalidBuilder.Validate()
		if err == nil {
			t.Error("Expected invalid builder to fail validation")
		}
	})
}

// TestEdgeCasesAdvanced tests various edge cases and error scenarios
func TestEdgeCasesAdvanced(t *testing.T) {
	t.Run("Nil pointer handling", func(t *testing.T) {
		// Test nil message methods don't panic
		var msg *Message

		// These should not panic even with nil message
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Unexpected panic with nil message: %v", r)
			}
		}()

		// Test that calling methods on nil message handles gracefully
		if msg != nil {
			msg.SetMetadata("key", "value")
			msg.SetVariable("var", "value")
			msg.SetPlatformData("platform", "data")
		}
	})

	t.Run("Invalid target scenarios", func(t *testing.T) {
		// Test invalid email addresses
		invalidEmails := []string{
			"invalid",
			"@example.com",
			"user@",
			"user@.com",
			"user@example",
		}

		for _, email := range invalidEmails {
			_, err := NewMessage("Test").
				WithBody("Test body").
				ToEmail(email).
				Build()
			if err == nil {
				t.Errorf("Expected error for invalid email: %s", email)
			}
		}

		// Test invalid phone numbers
		invalidPhones := []string{
			"abc",
			"123", // Too short
			"phone number",
			"@#$%",
		}

		for _, phone := range invalidPhones {
			_, err := NewMessage("Test").
				WithBody("Test body").
				ToPhone(phone).
				Build()
			if err == nil {
				t.Errorf("Expected error for invalid phone: %s", phone)
			}
		}

		// Test invalid webhook URLs
		invalidURLs := []string{
			"not-a-url",
			"ftp://example.com",
			"http://",
			"https://",
			"://example.com",
		}

		for _, url := range invalidURLs {
			_, err := NewMessage("Test").
				WithBody("Test body").
				ToWebhook(url).
				Build()
			if err == nil {
				t.Errorf("Expected error for invalid webhook URL: %s", url)
			}
		}
	})

	t.Run("Metadata validation", func(t *testing.T) {
		// Test nil variables map
		builder := NewMessage("Test").
			WithBody("Test body").
			ToEmail("test@example.com")

		// Test nil variables
		builder.WithVariables(nil)
		if !builder.HasErrors() {
			t.Error("Expected error for nil variables map")
		}

		// Test nil platform data
		builder.ClearErrors().WithPlatformData(nil)
		if !builder.HasErrors() {
			t.Error("Expected error for nil platform data map")
		}

		// Test empty key validation
		builder.ClearErrors().WithMetadata("", "value")
		if !builder.HasErrors() {
			t.Error("Expected error for empty metadata key")
		}

		builder.ClearErrors().WithVariable("", "value")
		if !builder.HasErrors() {
			t.Error("Expected error for empty variable key")
		}

		// Test key length limits (MaxKeywordLength = 50)
		longKey := strings.Repeat("k", 51)
		builder.ClearErrors().WithMetadata(longKey, "value")
		if !builder.HasErrors() {
			t.Error("Expected error for metadata key exceeding 50 characters")
		}

		builder.ClearErrors().WithVariable(longKey, "value")
		if !builder.HasErrors() {
			t.Error("Expected error for variable key exceeding 50 characters")
		}
	})

	t.Run("Scheduling validation", func(t *testing.T) {
		// Test scheduling in the past
		pastTime := time.Now().Add(-1 * time.Hour)
		_, err := NewMessage("Test").
			WithBody("Test body").
			ToEmail("test@example.com").
			ScheduleAt(pastTime).
			Build()
		if err == nil {
			t.Error("Expected error for scheduling in the past")
		}

		// Test scheduling too far in the future (> 1 year)
		futureTime := time.Now().AddDate(1, 1, 0) // More than 1 year
		_, err = NewMessage("Test").
			WithBody("Test body").
			ToEmail("test@example.com").
			ScheduleAt(futureTime).
			Build()
		if err == nil {
			t.Error("Expected error for scheduling more than 1 year in future")
		}

		// Test negative duration
		_, err = NewMessage("Test").
			WithBody("Test body").
			ToEmail("test@example.com").
			ScheduleIn(-1 * time.Hour).
			Build()
		if err == nil {
			t.Error("Expected error for negative schedule duration")
		}

		// Test duration too long (> 1 year)
		_, err = NewMessage("Test").
			WithBody("Test body").
			ToEmail("test@example.com").
			ScheduleIn(366 * 24 * time.Hour). // More than 1 year
			Build()
		if err == nil {
			t.Error("Expected error for schedule duration more than 1 year")
		}
	})
}

// BenchmarkMessageCreation benchmarks message creation performance
func BenchmarkMessageCreation(b *testing.B) {
	b.Run("Direct construction", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			msg := New()
			msg.Title = "Benchmark Title"
			msg.Body = "Benchmark body content for testing performance"
			msg.Format = FormatText
			msg.Priority = PriorityNormal
			msg.AddTarget(target.NewEmailTarget("benchmark@example.com"))
			_ = msg
		}
	})

	b.Run("Builder pattern", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			msg, err := NewMessage("Benchmark Title").
				WithBody("Benchmark body content for testing performance").
				WithFormat(FormatText).
				WithPriority(PriorityNormal).
				ToEmail("benchmark@example.com").
				Build()
			if err != nil {
				b.Fatalf("Unexpected error in benchmark: %v", err)
			}
			_ = msg
		}
	})

	b.Run("Complex builder chain", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			msg, err := NewMessage("Complex Benchmark").
				WithBody("Complex benchmark body with more content for testing").
				WithFormat(FormatMarkdown).
				WithPriority(PriorityHigh).
				WithMetadata("source", "benchmark").
				WithMetadata("iteration", i).
				WithVariable("user", "benchmark_user").
				WithVariable("timestamp", time.Now()).
				ToEmail("primary@example.com").
				ToPhone("+1234567890").
				ToFeishuUser("user123").
				WithFeishuMentions("user1", "user2").
				WithEmailCC("cc@example.com").
				Build()
			if err != nil {
				b.Fatalf("Unexpected error in complex benchmark: %v", err)
			}
			_ = msg
		}
	})

	b.Run("Multiple targets", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			builder := NewMessage("Multi-target Benchmark").
				WithBody("Testing multiple targets performance")

			// Add 10 targets
			for j := 0; j < 10; j++ {
				builder = builder.ToEmail(fmt.Sprintf("user%d@example.com", j))
			}

			msg, err := builder.Build()
			if err != nil {
				b.Fatalf("Unexpected error in multi-target benchmark: %v", err)
			}
			_ = msg
		}
	})

	b.Run("Large content", func(b *testing.B) {
		b.ReportAllocs()
		largebody := strings.Repeat("This is a large body content for testing performance with substantial text. ", 50) // ~3850 chars
		for i := 0; i < b.N; i++ {
			msg, err := NewMessage("Large Content Benchmark").
				WithBody(largebody).
				ToEmail("large@example.com").
				Build()
			if err != nil {
				b.Fatalf("Unexpected error in large content benchmark: %v", err)
			}
			_ = msg
		}
	})

	b.Run("BuildUnsafe vs Build", func(b *testing.B) {
		b.Run("BuildUnsafe", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				msg := NewMessage("Unsafe Build").
					WithBody("Testing unsafe build performance").
					ToEmail("unsafe@example.com").
					BuildUnsafe()
				_ = msg
			}
		})

		b.Run("Build", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				msg, err := NewMessage("Safe Build").
					WithBody("Testing safe build performance").
					ToEmail("safe@example.com").
					Build()
				if err != nil {
					b.Fatalf("Unexpected error in safe build benchmark: %v", err)
				}
				_ = msg
			}
		})
	})
}

// BenchmarkMessageValidation benchmarks validation performance
func BenchmarkMessageValidation(b *testing.B) {
	b.Run("Valid message validation", func(b *testing.B) {
		b.ReportAllocs()
		builder := NewMessage("Valid Message").
			WithBody("Valid body content").
			ToEmail("valid@example.com")

		for i := 0; i < b.N; i++ {
			err := builder.Validate()
			if err != nil {
				b.Fatalf("Unexpected validation error: %v", err)
			}
		}
	})

	b.Run("Invalid message validation", func(b *testing.B) {
		b.ReportAllocs()
		builder := NewMessage(""). // Invalid: empty title
			WithBody("Valid body content").
			ToEmail("valid@example.com")

		for i := 0; i < b.N; i++ {
			err := builder.Validate()
			if err == nil {
				b.Fatal("Expected validation error for invalid message")
			}
		}
	})

	b.Run("Complex validation", func(b *testing.B) {
		b.ReportAllocs()
		builder := NewMessage("Complex Message").
			WithBody("Complex body with validation requirements").
			WithFormat(FormatMarkdown).
			WithPriority(PriorityHigh).
			WithMetadata("key1", "value1").
			WithVariable("var1", "value1")

		// Add multiple targets for validation
		for i := 0; i < 5; i++ {
			builder = builder.ToEmail(fmt.Sprintf("user%d@example.com", i))
		}

		for i := 0; i < b.N; i++ {
			err := builder.Validate()
			if err != nil {
				b.Fatalf("Unexpected validation error: %v", err)
			}
		}
	})
}

// BenchmarkMessageMethods benchmarks individual message methods
func BenchmarkMessageMethods(b *testing.B) {
	b.Run("GetMetadata", func(b *testing.B) {
		b.ReportAllocs()
		msg := New()
		msg.SetMetadata("key1", "value1")
		msg.SetMetadata("key2", "value2")
		msg.SetMetadata("key3", "value3")

		for i := 0; i < b.N; i++ {
			metadata := msg.GetMetadata()
			_ = metadata
		}
	})

	b.Run("GetVariables", func(b *testing.B) {
		b.ReportAllocs()
		msg := New()
		msg.SetVariable("var1", "value1")
		msg.SetVariable("var2", "value2")
		msg.SetVariable("var3", "value3")

		for i := 0; i < b.N; i++ {
			variables := msg.GetVariables()
			_ = variables
		}
	})

	b.Run("GetPlatformData", func(b *testing.B) {
		b.ReportAllocs()
		msg := New()
		msg.SetPlatformData("platform1", "data1")
		msg.SetPlatformData("platform2", "data2")
		msg.SetPlatformData("platform3", "data3")

		for i := 0; i < b.N; i++ {
			platformData := msg.GetPlatformData()
			_ = platformData
		}
	})

	b.Run("AddTarget", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			msg := New()
			target := target.NewEmailTarget(fmt.Sprintf("user%d@example.com", i))
			msg.AddTarget(target)
		}
	})

	b.Run("AddTargets", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			msg := New()
			targets := []target.Target{
				target.NewEmailTarget("user1@example.com"),
				target.NewEmailTarget("user2@example.com"),
				target.NewEmailTarget("user3@example.com"),
			}
			msg.AddTargets(targets...)
		}
	})
}

// TestConcurrentUsage tests builder pattern under concurrent usage
func TestConcurrentUsage(t *testing.T) {
	t.Run("Concurrent message creation", func(t *testing.T) {
		// Test that multiple goroutines can create messages concurrently
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()

				msg, err := NewMessage(fmt.Sprintf("Concurrent Message %d", id)).
					WithBody(fmt.Sprintf("Body for message %d", id)).
					WithPriority(PriorityNormal).
					ToEmail(fmt.Sprintf("user%d@example.com", id)).
					Build()

				if err != nil {
					t.Errorf("Goroutine %d failed: %v", id, err)
					return
				}

				if msg.Title != fmt.Sprintf("Concurrent Message %d", id) {
					t.Errorf("Goroutine %d: expected title 'Concurrent Message %d', got %s", id, id, msg.Title)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("Concurrent message modification", func(t *testing.T) {
		// Test that modifying different message instances concurrently is safe
		messages := make([]*Message, 10)
		for i := range messages {
			messages[i] = New()
		}

		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()

				msg := messages[id]
				msg.SetMetadata("goroutine_id", id)
				msg.SetVariable("iteration", id*10)
				msg.SetPlatformData("test_data", fmt.Sprintf("data-%d", id))
				msg.AddTarget(target.NewEmailTarget(fmt.Sprintf("concurrent%d@example.com", id)))
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify each message was modified correctly
		for i, msg := range messages {
			if msg.Metadata["goroutine_id"] != i {
				t.Errorf("Message %d: expected goroutine_id %d, got %v", i, i, msg.Metadata["goroutine_id"])
			}
			if msg.Variables["iteration"] != i*10 {
				t.Errorf("Message %d: expected iteration %d, got %v", i, i*10, msg.Variables["iteration"])
			}
			if len(msg.Targets) != 1 {
				t.Errorf("Message %d: expected 1 target, got %d", i, len(msg.Targets))
			}
		}
	})
}
