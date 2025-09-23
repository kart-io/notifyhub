package platform

import (
	"context"
	"fmt"
	"testing"
)

func TestNewManager(t *testing.T) {
	factory := NewMockSenderFactory()
	resolver := NewDefaultTargetResolver()
	converter := NewDefaultMessageConverter()
	validator := NewDefaultValidator()

	manager := NewManager(factory, resolver, converter, validator)

	if manager == nil {
		t.Fatal("Expected NewManager to return a non-nil manager")
	}
}

func TestManagerRegisterSender(t *testing.T) {
	manager := NewManager(
		NewMockSenderFactory(),
		NewDefaultTargetResolver(),
		NewDefaultMessageConverter(),
		NewDefaultValidator(),
	)

	sender := NewMockSender("test-platform")

	err := manager.RegisterSender(sender)
	if err != nil {
		t.Fatalf("Expected RegisterSender to succeed, got error: %v", err)
	}

	// Test registering duplicate sender
	err = manager.RegisterSender(sender)
	if err == nil {
		t.Error("Expected RegisterSender to fail for duplicate sender")
	}
}

func TestManagerRegisterSenderEmptyName(t *testing.T) {
	manager := NewManager(
		NewMockSenderFactory(),
		NewDefaultTargetResolver(),
		NewDefaultMessageConverter(),
		NewDefaultValidator(),
	)

	sender := NewMockSender("")

	err := manager.RegisterSender(sender)
	if err == nil {
		t.Error("Expected RegisterSender to fail for empty name")
	}
}

func TestManagerGetSender(t *testing.T) {
	manager := NewManager(
		NewMockSenderFactory(),
		NewDefaultTargetResolver(),
		NewDefaultMessageConverter(),
		NewDefaultValidator(),
	)

	sender := NewMockSender("test-platform")
	if err := manager.RegisterSender(sender); err != nil {
		t.Fatalf("Failed to register sender: %v", err)
	}

	// Test getting existing sender
	retrieved, exists := manager.GetSender("test-platform")
	if !exists {
		t.Error("Expected GetSender to find registered sender")
	}
	if retrieved != sender {
		t.Error("Expected GetSender to return the same sender instance")
	}

	// Test getting non-existent sender
	_, exists = manager.GetSender("non-existent")
	if exists {
		t.Error("Expected GetSender to return false for non-existent sender")
	}
}

func TestManagerListSenders(t *testing.T) {
	manager := NewManager(
		NewMockSenderFactory(),
		NewDefaultTargetResolver(),
		NewDefaultMessageConverter(),
		NewDefaultValidator(),
	)

	// Initially should be empty
	senders := manager.ListSenders()
	if len(senders) != 0 {
		t.Errorf("Expected empty sender list, got %d senders", len(senders))
	}

	// Add senders
	sender1 := NewMockSender("platform1")
	sender2 := NewMockSender("platform2")
	if err := manager.RegisterSender(sender1); err != nil {
		t.Fatalf("Failed to register sender1: %v", err)
	}
	if err := manager.RegisterSender(sender2); err != nil {
		t.Fatalf("Failed to register sender2: %v", err)
	}

	senders = manager.ListSenders()
	if len(senders) != 2 {
		t.Errorf("Expected 2 senders, got %d", len(senders))
	}
}

func TestManagerSendToAll(t *testing.T) {
	manager := NewManager(
		NewMockSenderFactory(),
		NewDefaultTargetResolver(),
		NewDefaultMessageConverter(),
		NewDefaultValidator(),
	)

	// Register mock senders
	sender1 := NewMockSender("platform1")
	sender2 := NewMockSender("platform2")
	if err := manager.RegisterSender(sender1); err != nil {
		t.Fatalf("Failed to register sender1: %v", err)
	}
	if err := manager.RegisterSender(sender2); err != nil {
		t.Fatalf("Failed to register sender2: %v", err)
	}

	msg := NewInternalMessage("test-id", "Test", "Test message")
	targets := []InternalTarget{
		NewInternalTarget("test", "target1", "platform1"),
		NewInternalTarget("test", "target2", "platform2"),
	}

	ctx := context.Background()
	results, err := manager.SendToAll(ctx, msg, targets)

	if err != nil {
		t.Fatalf("Expected SendToAll to succeed, got error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	for _, result := range results {
		if !result.Success {
			t.Errorf("Expected result to be successful, got: %+v", result)
		}
	}
}

func TestManagerSendToAllWithUnknownPlatform(t *testing.T) {
	manager := NewManager(
		NewMockSenderFactory(),
		NewDefaultTargetResolver(),
		NewDefaultMessageConverter(),
		NewDefaultValidator(),
	)

	msg := NewInternalMessage("test-id", "Test", "Test message")
	targets := []InternalTarget{
		NewInternalTarget("test", "target1", "unknown-platform"),
	}

	ctx := context.Background()
	results, err := manager.SendToAll(ctx, msg, targets)

	// Should succeed but with failed results
	if err != nil {
		t.Fatalf("Expected SendToAll to succeed even with unknown platform, got error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Success {
		t.Error("Expected result to be failed for unknown platform")
	}
	if result.Error == "" {
		t.Error("Expected error message for unknown platform")
	}
}

func TestManagerSendToAllWithSenderError(t *testing.T) {
	manager := NewManager(
		NewMockSenderFactory(),
		NewDefaultTargetResolver(),
		NewDefaultMessageConverter(),
		NewDefaultValidator(),
	)

	sender := NewMockSender("platform1")
	expectedError := fmt.Errorf("sender error")
	sender.SetSendError(expectedError)
	if err := manager.RegisterSender(sender); err != nil {
		t.Fatalf("Failed to register sender: %v", err)
	}

	msg := NewInternalMessage("test-id", "Test", "Test message")
	targets := []InternalTarget{
		NewInternalTarget("test", "target1", "platform1"),
	}

	ctx := context.Background()
	results, err := manager.SendToAll(ctx, msg, targets)

	if err == nil {
		t.Error("Expected SendToAll to return error when sender fails")
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Success {
		t.Error("Expected result to be failed when sender returns error")
	}
}

func TestManagerHealthCheck(t *testing.T) {
	manager := NewManager(
		NewMockSenderFactory(),
		NewDefaultTargetResolver(),
		NewDefaultMessageConverter(),
		NewDefaultValidator(),
	)

	// Register healthy and unhealthy senders
	healthySender := NewMockSender("healthy")
	unhealthySender := NewMockSender("unhealthy")
	unhealthySender.SetHealthError(fmt.Errorf("health check failed"))

	if err := manager.RegisterSender(healthySender); err != nil {
		t.Fatalf("Failed to register healthy sender: %v", err)
	}
	if err := manager.RegisterSender(unhealthySender); err != nil {
		t.Fatalf("Failed to register unhealthy sender: %v", err)
	}

	ctx := context.Background()
	health := manager.HealthCheck(ctx)

	if len(health) != 2 {
		t.Errorf("Expected 2 health results, got %d", len(health))
	}

	if health["healthy"] != nil {
		t.Errorf("Expected healthy sender to return nil error, got: %v", health["healthy"])
	}

	if health["unhealthy"] == nil {
		t.Error("Expected unhealthy sender to return error")
	}
}

func TestManagerClose(t *testing.T) {
	manager := NewManager(
		NewMockSenderFactory(),
		NewDefaultTargetResolver(),
		NewDefaultMessageConverter(),
		NewDefaultValidator(),
	)

	sender1 := NewMockSender("platform1")
	sender2 := NewMockSender("platform2")
	sender2.SetCloseError(fmt.Errorf("close error"))

	if err := manager.RegisterSender(sender1); err != nil {
		t.Fatalf("Failed to register sender1: %v", err)
	}
	if err := manager.RegisterSender(sender2); err != nil {
		t.Fatalf("Failed to register sender2: %v", err)
	}

	err := manager.Close()
	if err == nil {
		t.Error("Expected Close to return error when one sender fails to close")
	}

	// After close, sender list should be empty
	senders := manager.ListSenders()
	if len(senders) != 0 {
		t.Errorf("Expected empty sender list after close, got %d senders", len(senders))
	}
}

func TestDefaultSenderFactory(t *testing.T) {
	factory := NewDefaultSenderFactory()

	platforms := factory.GetSupportedPlatforms()
	expectedPlatforms := []string{"email", "feishu", "sms"}

	if len(platforms) != len(expectedPlatforms) {
		t.Errorf("Expected %d supported platforms, got %d", len(expectedPlatforms), len(platforms))
	}

	for _, platform := range expectedPlatforms {
		found := false
		for _, p := range platforms {
			if p == platform {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected platform %s to be supported", platform)
		}
	}
}

func TestDefaultSenderFactoryValidateConfig(t *testing.T) {
	factory := NewDefaultSenderFactory()

	tests := []struct {
		platform string
		config   map[string]interface{}
		wantErr  bool
	}{
		{
			platform: "email",
			config: map[string]interface{}{
				"smtp_host":     "smtp.example.com",
				"smtp_port":     587,
				"smtp_username": "user",
				"smtp_password": "pass",
				"smtp_from":     "from@example.com",
			},
			wantErr: false,
		},
		{
			platform: "email",
			config:   map[string]interface{}{},
			wantErr:  true,
		},
		{
			platform: "feishu",
			config: map[string]interface{}{
				"webhook_url": "https://example.com/webhook",
			},
			wantErr: false,
		},
		{
			platform: "feishu",
			config:   map[string]interface{}{},
			wantErr:  true,
		},
		{
			platform: "sms",
			config: map[string]interface{}{
				"provider": "twilio",
				"api_key":  "key123",
			},
			wantErr: false,
		},
		{
			platform: "sms",
			config:   map[string]interface{}{},
			wantErr:  true,
		},
		{
			platform: "unknown",
			config:   map[string]interface{}{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%v", tt.platform, tt.wantErr), func(t *testing.T) {
			err := factory.ValidateConfig(tt.platform, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultMessageConverter(t *testing.T) {
	converter := NewDefaultMessageConverter()

	// Test Convert method
	input := "test input"
	output, err := converter.Convert(input)
	if err != nil {
		t.Errorf("Convert() error = %v", err)
	}
	if output != input {
		t.Errorf("Convert() = %v, want %v", output, input)
	}

	// Test ToInternal method
	msg, err := converter.ToInternal("test", "platform")
	if err != nil {
		t.Errorf("ToInternal() error = %v", err)
	}
	if msg == nil {
		t.Error("ToInternal() returned nil message")
		return
	}
	if msg.Format != "text" {
		t.Errorf("ToInternal() format = %s, want text", msg.Format)
	}
}

func TestDefaultValidator(t *testing.T) {
	validator := NewDefaultValidator()

	// Test ValidateMessage
	validMsg := NewInternalMessage("id", "title", "body")
	err := validator.ValidateMessage(validMsg)
	if err != nil {
		t.Errorf("ValidateMessage() error = %v for valid message", err)
	}

	invalidMsg := NewInternalMessage("id", "", "")
	err = validator.ValidateMessage(invalidMsg)
	if err == nil {
		t.Error("ValidateMessage() should return error for message without title or body")
	}

	// Test ValidateTarget
	validTarget := NewInternalTarget("email", "test@example.com", "email")
	err = validator.ValidateTarget(validTarget)
	if err != nil {
		t.Errorf("ValidateTarget() error = %v for valid target", err)
	}

	invalidTarget := NewInternalTarget("", "", "")
	err = validator.ValidateTarget(invalidTarget)
	if err == nil {
		t.Error("ValidateTarget() should return error for invalid target")
	}
}

func TestDefaultTargetResolver(t *testing.T) {
	resolver := NewDefaultTargetResolver()

	// Test Resolve method
	input := "test input"
	output, err := resolver.Resolve(input)
	if err != nil {
		t.Errorf("Resolve() error = %v", err)
	}
	if output != input {
		t.Errorf("Resolve() = %v, want %v", output, input)
	}

	// Test ResolveTargets method
	targetsInput := []interface{}{
		map[string]interface{}{
			"type":     "email",
			"value":    "test@example.com",
			"platform": "email",
		},
		map[string]interface{}{
			"type":  "phone",
			"value": "+1234567890",
		},
	}

	result := resolver.ResolveTargets(targetsInput)
	if len(result) == 0 {
		t.Error("ResolveTargets() returned empty result")
	}

	if targets, exists := result["email"]; exists {
		if len(targets) != 1 {
			t.Errorf("Expected 1 email target, got %d", len(targets))
		}
		if targets[0].Value != "test@example.com" {
			t.Errorf("Expected email target value test@example.com, got %s", targets[0].Value)
		}
	} else {
		t.Error("Expected email platform in result")
	}

	if targets, exists := result["sms"]; exists {
		if len(targets) != 1 {
			t.Errorf("Expected 1 sms target, got %d", len(targets))
		}
		if targets[0].Value != "+1234567890" {
			t.Errorf("Expected sms target value +1234567890, got %s", targets[0].Value)
		}
	} else {
		t.Error("Expected sms platform in result")
	}
}
