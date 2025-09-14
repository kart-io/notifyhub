package client

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers"
)

func TestHubCreation(t *testing.T) {
	// Test Hub creation fails without notifiers
	_, err := New()
	if err == nil {
		t.Error("Hub creation should fail without notifiers")
	}

	// Test Hub creation fails even with test defaults (no notifiers)
	_, err = New(config.WithTestDefaults())
	if err == nil {
		t.Error("Hub creation should fail without notifiers, even with test defaults")
	}

	// Test successful Hub creation with notifiers
	hub, err := New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create Hub with notifiers: %v", err)
	}

	if hub == nil {
		t.Fatal("Hub should not be nil")
	}
}

func TestHubWithNotifiers(t *testing.T) {
	// Create Hub with mock notifiers
	hub, err := New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create Hub with Feishu: %v", err)
	}

	if hub == nil {
		t.Fatal("Hub should not be nil")
	}

	// Test Hub start/stop lifecycle
	ctx := context.Background()
	err = hub.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start Hub: %v", err)
	}

	err = hub.Stop()
	if err != nil {
		t.Fatalf("Failed to stop Hub: %v", err)
	}
}

func TestHubSendSync(t *testing.T) {
	// Create Hub with test configuration
	hub, err := New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create Hub: %v", err)
	}

	ctx := context.Background()
	message := NewMessage().
		Title("Test Message").
		Body("Test Body").
		Priority(3).
		Build()

	// Test synchronous send (will fail due to invalid webhook, but should not panic)
	results, err := hub.SendSync(ctx, message, nil)

	// Since we're using a fake webhook, we expect an error, but the function should execute
	if results == nil && err == nil {
		t.Error("Either results or error should be non-nil")
	}
}

func TestHubSendBatch(t *testing.T) {
	// Create Hub with test configuration
	hub, err := New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create Hub: %v", err)
	}

	ctx := context.Background()
	messages := []*notifiers.Message{
		NewMessage().Title("Test 1").Body("Body 1").Build(),
		NewMessage().Title("Test 2").Body("Body 2").Build(),
		NewMessage().Title("Test 3").Body("Body 3").Build(),
	}

	// Test batch send with empty messages
	results, err := hub.SendBatch(ctx, []*notifiers.Message{}, nil)
	if err != nil {
		t.Errorf("Empty batch should not return error: %v", err)
	}
	if len(results) != 0 {
		t.Error("Empty batch should return empty results")
	}

	// Test batch send with messages (will fail due to invalid webhook)
	results, err = hub.SendBatch(ctx, messages, nil)

	// We expect some response even with failed sends
	if results == nil {
		t.Error("Batch send should return results even with failures")
	}
}

func TestHubHealthAndMetrics(t *testing.T) {
	// Create Hub with test configuration
	hub, err := New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create Hub: %v", err)
	}

	ctx := context.Background()

	// Test health check
	health := hub.GetHealth(ctx)
	if health == nil {
		t.Error("Health check should return non-nil result")
	}

	if health["status"] == nil {
		t.Error("Health check should include status")
	}

	// Test metrics
	metrics := hub.GetMetrics()
	if metrics == nil {
		t.Error("Metrics should return non-nil result")
	}
}

func TestMessageBuilder(t *testing.T) {
	// Test basic message building
	message := NewMessage().
		Title("Test Title").
		Body("Test Body").
		Priority(5).
		Format(notifiers.FormatMarkdown).
		Build()

	if message.Title != "Test Title" {
		t.Error("Message title should be 'Test Title'")
	}

	if message.Body != "Test Body" {
		t.Error("Message body should be 'Test Body'")
	}

	if message.Priority != 5 {
		t.Error("Message priority should be 5")
	}

	if message.Format != notifiers.FormatMarkdown {
		t.Error("Message format should be markdown")
	}

	// Test alert message
	alert := NewAlert("Alert", "Emergency").Build()
	if alert.Priority != 4 {
		t.Error("Alert should have priority 4")
	}

	// Test notice message
	notice := NewNotice("Notice", "Information").Build()
	if notice.Priority != 3 {
		t.Error("Notice should have priority 3")
	}

	// Test report message
	report := NewReport("Report", "Status").Build()
	if report.Priority != 2 {
		t.Error("Report should have priority 2")
	}
}

func TestOptions(t *testing.T) {
	// Test options creation
	opts := NewOptions()
	if opts == nil {
		t.Error("Options should not be nil")
	}

	// Test async options
	asyncOpts := NewAsyncOptions()
	if !asyncOpts.Async {
		t.Error("Async options should have Async=true")
	}

	if !asyncOpts.Retry {
		t.Error("Async options should have Retry=true")
	}

	// Test sync options
	syncOpts := NewSyncOptions()
	if syncOpts.Async {
		t.Error("Sync options should have Async=false")
	}

	if syncOpts.Timeout != 30*time.Second {
		t.Error("Sync options should have 30s timeout")
	}

	// Test retry options
	retryOpts := NewRetryOptions(5)
	if !retryOpts.Retry {
		t.Error("Retry options should have Retry=true")
	}

	if retryOpts.MaxRetries != 5 {
		t.Error("Retry options should have MaxRetries=5")
	}
}

func TestConvenienceMethods(t *testing.T) {
	// Create Hub with test configuration
	hub, err := New(
		config.WithTestDefaults(),
		config.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create Hub: %v", err)
	}

	ctx := context.Background()

	// Test SendText convenience method
	err = hub.SendText(ctx, "Test Title", "Test Body")
	// We expect an error due to invalid webhook, but method should not panic
	if err == nil {
		t.Log("SendText executed without error (unexpected with fake webhook)")
	}

	// Test SendAlert convenience method
	err = hub.SendAlert(ctx, "Alert Title", "Alert Body")
	if err == nil {
		t.Log("SendAlert executed without error (unexpected with fake webhook)")
	}

	// Test SendWithTemplate convenience method
	variables := map[string]interface{}{
		"name": "Test User",
		"count": 42,
	}
	err = hub.SendWithTemplate(ctx, "test-template", variables)
	if err == nil {
		t.Log("SendWithTemplate executed without error (unexpected with fake webhook)")
	}
}