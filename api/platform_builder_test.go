package api

import (
	"testing"

	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/platforms/feishu"
)

func TestUnifiedPlatformBuilder_Email(t *testing.T) {
	cfg := &config.Config{}
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	builder := NewUnifiedPlatformBuilder(client, PlatformEmail)

	// Test email-specific methods
	builder.Title("Test Email").
		Body("Email body").
		To("user@example.com", "admin@example.com").
		CC("cc@example.com").
		BCC("bcc@example.com").
		Subject("Custom Subject").
		HTMLBody("<h1>HTML Content</h1>")

	// Validate platform data
	emailData, ok := builder.platformData.(*EmailPlatformData)
	if !ok {
		t.Fatal("Expected EmailPlatformData")
	}

	if len(emailData.To) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(emailData.To))
	}

	if len(emailData.CC) != 1 {
		t.Errorf("Expected 1 CC recipient, got %d", len(emailData.CC))
	}

	if len(emailData.BCC) != 1 {
		t.Errorf("Expected 1 BCC recipient, got %d", len(emailData.BCC))
	}

	if emailData.Subject != "Custom Subject" {
		t.Errorf("Expected subject 'Custom Subject', got '%s'", emailData.Subject)
	}

	if emailData.HTMLBody != "<h1>HTML Content</h1>" {
		t.Errorf("Expected HTML body to be set")
	}

	// Test message format - build message to check format
	msg := builder.Build()
	if msg.Format != core.FormatHTML {
		t.Error("Expected message format to be HTML")
	}

	// Test targets generation
	targets := emailData.GetTargets()
	if len(targets) != 4 { // 2 To + 1 CC + 1 BCC
		t.Errorf("Expected 4 targets, got %d", len(targets))
	}
}

func TestUnifiedPlatformBuilder_Feishu(t *testing.T) {
	cfg := &config.Config{}
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	builder := NewUnifiedPlatformBuilder(client, PlatformFeishu)

	// Test Feishu-specific methods
	card := &feishu.FeishuCard{
		Config: feishu.FeishuCardConfig{
			WideScreenMode: true,
		},
		Elements: []feishu.FeishuCardElement{
			{Tag: "div", Text: "Test content"},
		},
	}

	builder.Title("Test Feishu").
		Body("Feishu body").
		ToGroup("group1", "group2").
		ToUser("user1", "user2").
		AtAll().
		AtUser("mention1", "mention2").
		Card(card).
		WithWebhook("https://webhook.url", "secret")

	// Validate platform data
	feishuData, ok := builder.platformData.(*FeishuPlatformData)
	if !ok {
		t.Fatal("Expected FeishuPlatformData")
	}

	if len(feishuData.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(feishuData.Groups))
	}

	if len(feishuData.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(feishuData.Users))
	}

	if !feishuData.AtAll {
		t.Error("Expected AtAll to be true")
	}

	if len(feishuData.AtUsers) != 2 {
		t.Errorf("Expected 2 mentioned users, got %d", len(feishuData.AtUsers))
	}

	if feishuData.Card == nil {
		t.Error("Expected card to be set")
	}

	if feishuData.Webhook != "https://webhook.url" {
		t.Errorf("Expected webhook URL, got '%s'", feishuData.Webhook)
	}

	if feishuData.Secret != "secret" {
		t.Errorf("Expected secret, got '%s'", feishuData.Secret)
	}

	// Test message format - build message to check format
	msg := builder.Build()
	if msg.Format != core.FormatCard {
		t.Error("Expected message format to be Card")
	}
}

func TestUnifiedPlatformBuilder_Slack(t *testing.T) {
	cfg := &config.Config{}
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	builder := NewUnifiedPlatformBuilder(client, PlatformSlack)

	// Test Slack-specific methods
	builder.Title("Test Slack").
		Body("Slack body").
		ToChannel("general", "alerts").
		ToUser("john", "jane").
		InThread("1234567890.123456").
		Broadcast().
		LinkNames().
		WithWebhook("https://hooks.slack.com/webhook")

	// Validate platform data
	slackData, ok := builder.platformData.(*SlackPlatformData)
	if !ok {
		t.Fatal("Expected SlackPlatformData")
	}

	if len(slackData.Channels) != 2 {
		t.Errorf("Expected 2 channels, got %d", len(slackData.Channels))
	}

	if len(slackData.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(slackData.Users))
	}

	if slackData.ThreadTs != "1234567890.123456" {
		t.Errorf("Expected thread timestamp, got '%s'", slackData.ThreadTs)
	}

	if !slackData.Broadcast {
		t.Error("Expected Broadcast to be true")
	}

	if !slackData.LinkNames {
		t.Error("Expected LinkNames to be true")
	}

	if slackData.Webhook != "https://hooks.slack.com/webhook" {
		t.Errorf("Expected webhook URL, got '%s'", slackData.Webhook)
	}
}

func TestUnifiedPlatformBuilder_Validation(t *testing.T) {
	cfg := &config.Config{}
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test email validation
	emailBuilder := NewUnifiedPlatformBuilder(client, PlatformEmail)
	emailBuilder.Title("Test").Body("Test body")

	// Should fail without recipients
	_, err = emailBuilder.DryRun()
	if err == nil {
		t.Error("Expected validation error for email without recipients")
	}

	// Should pass with recipients
	emailBuilder.To("user@example.com")
	result, err := emailBuilder.DryRun()
	if err != nil {
		t.Errorf("Expected validation to pass with recipients, got error: %v", err)
	}
	if !result.Valid {
		t.Error("Expected dry run result to be valid")
	}

	// Test Feishu validation
	feishuBuilder := NewUnifiedPlatformBuilder(client, PlatformFeishu)
	feishuBuilder.Title("Test").Body("Test body")

	// Should fail without targets
	_, err = feishuBuilder.DryRun()
	if err == nil {
		t.Error("Expected validation error for Feishu without targets")
	}

	// Should pass with webhook
	feishuBuilder.WithWebhook("https://webhook.url")
	_, err = feishuBuilder.DryRun()
	if err != nil {
		t.Errorf("Expected validation to pass with webhook, got error: %v", err)
	}
}

func TestUnifiedPlatformBuilder_FluentInterface(t *testing.T) {
	cfg := &config.Config{}
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test method chaining
	builder := NewUnifiedPlatformBuilder(client, PlatformEmail).
		Title("Chained Title").
		Body("Chained Body").
		Priority(message.PriorityHigh).
		Template("test-template").
		Var("key", "value").
		To("user@example.com")

	// Build message to check the chained values
	msg := builder.Build()

	if msg.Title != "Chained Title" {
		t.Error("Expected chained title to be set")
	}

	if msg.Body != "Chained Body" {
		t.Error("Expected chained body to be set")
	}

	if msg.Priority != core.PriorityHigh {
		t.Error("Expected chained priority to be set")
	}
}

func TestUnifiedPlatformBuilder_CrossPlatformMethods(t *testing.T) {
	cfg := &config.Config{}
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test ToUser works for both Feishu and Slack
	feishuBuilder := NewUnifiedPlatformBuilder(client, PlatformFeishu)
	feishuBuilder.ToUser("user1", "user2")

	feishuData := feishuBuilder.platformData.(*FeishuPlatformData)
	if len(feishuData.Users) != 2 {
		t.Error("Expected ToUser to work with Feishu")
	}

	slackBuilder := NewUnifiedPlatformBuilder(client, PlatformSlack)
	slackBuilder.ToUser("user1", "user2")

	slackData := slackBuilder.platformData.(*SlackPlatformData)
	if len(slackData.Users) != 2 {
		t.Error("Expected ToUser to work with Slack")
	}

	// Test methods that don't apply to email platform don't panic
	emailBuilder := NewUnifiedPlatformBuilder(client, PlatformEmail)
	emailBuilder.ToChannel("test") // Should be no-op for email
	emailBuilder.AtAll()           // Should be no-op for email

	// Should not cause any issues
	emailData := emailBuilder.platformData.(*EmailPlatformData)
	if len(emailData.To) != 0 {
		t.Error("Expected no changes to email data from non-email methods")
	}
}

func BenchmarkUnifiedPlatformBuilder_EmailCreation(b *testing.B) {
	cfg := &config.Config{}
	client, _ := New(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewUnifiedPlatformBuilder(client, PlatformEmail)
		builder.Title("Test").
			Body("Test body").
			To("user@example.com").
			CC("cc@example.com").
			Subject("Test subject")
	}
}

func BenchmarkUnifiedPlatformBuilder_DryRun(b *testing.B) {
	cfg := &config.Config{}
	client, _ := New(cfg)

	builder := NewUnifiedPlatformBuilder(client, PlatformEmail)
	builder.Title("Test").Body("Test body").To("user@example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = builder.DryRun()
	}
}
