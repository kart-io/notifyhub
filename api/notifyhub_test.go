package api

import (
	"context"
	"testing"

	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/notifiers"
)

func TestNotifyHubCreation(t *testing.T) {
	cfg := config.New(config.WithTestDefaults())
	hub, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub: %v", err)
	}

	if hub == nil {
		t.Fatal("NotifyHub should not be nil")
	}
}

func TestMessageBuilder(t *testing.T) {
	cfg := config.New(config.WithTestDefaults())
	_, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub: %v", err)
	}

	// Test message builder
	builder := NewMessage()
	if builder == nil {
		t.Fatal("MessageBuilder should not be nil")
	}

	// Build a message
	msg := builder.
		Title("Test Title").
		Body("Test Body").
		Priority(message.PriorityHigh).
		ToEmail("test@example.com").
		Build()

	if msg.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", msg.Title)
	}

	if msg.Body != "Test Body" {
		t.Errorf("Expected body 'Test Body', got '%s'", msg.Body)
	}

	if msg.Priority != message.PriorityHigh {
		t.Errorf("Expected priority %d, got %d", message.PriorityHigh, msg.Priority)
	}

	if len(msg.Targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(msg.Targets))
	}

	if msg.Targets[0].Type != notifiers.TargetTypeEmail {
		t.Errorf("Expected email target, got %s", msg.Targets[0].Type)
	}
}

func TestAlertBuilder(t *testing.T) {
	cfg := config.New(config.WithTestDefaults())
	_, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub: %v", err)
	}

	// Test alert builder using NewMessage with alert priority
	builder := NewMessage().
		Title("Alert Title").
		Body("Alert Body").
		Priority(message.PriorityHigh) // Use high priority for alerts
	if builder == nil {
		t.Fatal("Alert builder should not be nil")
	}

	msg := builder.ToEmail("alert@example.com").Build()

	if msg.Title != "Alert Title" {
		t.Errorf("Expected title 'Alert Title', got '%s'", msg.Title)
	}

	if msg.Priority != message.PriorityHigh {
		t.Errorf("Expected high priority for alert, got %d", msg.Priority)
	}

	// Check alert metadata
	if msg.Metadata["type"] != "alert" {
		t.Errorf("Expected alert type metadata, got %s", msg.Metadata["type"])
	}
}

func TestSendWithMockTransport(t *testing.T) {
	cfg := config.New(config.WithTestDefaults())
	hub, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub: %v", err)
	}

	// Create a message
	msg := message.NewMessage()
	msg.SetTitle("Test Message").SetBody("Test Body")

	// Create targets (mock targets since we don't have real transports)
	targets := []sending.Target{
		sending.NewTarget(sending.TargetTypeEmail, "test@example.com", "email"),
	}

	// This will fail because we don't have email transport configured, but it tests the flow
	ctx := context.Background()
	results, err := hub.Send(ctx, msg, targets)

	// We expect this to fail with transport not found error
	if err != nil {
		t.Logf("Expected failure due to missing transport: %v", err)
	}

	if results == nil {
		t.Fatal("Results should not be nil even on failure")
	}

	if len(results.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results.Results))
	}

	// Should have failed result
	if results.Results[0].IsSuccess() {
		t.Error("Expected result to fail due to missing transport")
	}
}

func TestShutdown(t *testing.T) {
	cfg := config.New(config.WithTestDefaults())
	hub, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()
	err = hub.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown should not fail: %v", err)
	}

	if !hub.IsShutdown() {
		t.Error("Hub should be marked as shutdown")
	}
}
