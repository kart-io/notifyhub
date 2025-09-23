package notifyhub

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

// TestIntegrationFullWorkflow tests the complete end-to-end workflow
func TestIntegrationFullWorkflow(t *testing.T) {
	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create hub with test configuration
	hub, err := notifyhub.NewHub(
		notifyhub.WithTestDefaults(),
		notifyhub.WithFeishuFromMap(map[string]interface{}{
			"webhook_url": "https://test.example.com/webhook",
			"secret":      "test-secret",
		}),
		notifyhub.WithEmailFromMap(map[string]interface{}{
			"smtp_host":     "smtp.test.example.com",
			"smtp_port":     587,
			"smtp_username": "test@example.com",
			"smtp_password": "test-password",
			"smtp_from":     "test@example.com",
			"smtp_tls":      true,
		}),
		notifyhub.WithSMSFromMap(map[string]interface{}{
			"provider": "test",
			"api_key":  "test-key",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create hub: %v", err)
	}
	defer hub.Close(context.Background())

	ctx := context.Background()

	// Test 1: Simple text message to multiple platforms
	t.Run("Simple text message", func(t *testing.T) {
		message := notifyhub.NewMessage("Integration Test").
			WithText("This is a test message from integration test").
			ToEmail("test@example.com").
			ToPhone("+1234567890").
			Build()

		receipt, err := hub.Send(ctx, message)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		if receipt.MessageID != message.ID {
			t.Errorf("Expected MessageID %s, got %s", message.ID, receipt.MessageID)
		}

		if receipt.Total == 0 {
			t.Error("Expected at least one result")
		}

		// Check that we have results for each target
		expectedTargets := len(message.Targets)
		if len(receipt.Results) != expectedTargets {
			t.Errorf("Expected %d results, got %d", expectedTargets, len(receipt.Results))
		}
	})

	// Test 2: Rich content message
	t.Run("Rich content message", func(t *testing.T) {
		htmlContent := `
		<h1>Test Alert</h1>
		<p>This is a <strong>test alert</strong> with HTML content.</p>
		<ul>
			<li>Item 1</li>
			<li>Item 2</li>
		</ul>
		`

		message := notifyhub.NewAlert("Test Alert").
			WithHTML(htmlContent).
			WithEmailCC("cc@example.com").
			WithEmailPriority("high").
			ToEmail("recipient@example.com").
			Build()

		receipt, err := hub.Send(ctx, message)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		if receipt.Status == "" {
			t.Error("Expected receipt status to be set")
		}
	})

	// Test 3: Feishu card message
	t.Run("Feishu card message", func(t *testing.T) {
		cardContent := map[string]interface{}{
			"config": map[string]interface{}{
				"wide_screen_mode": true,
			},
			"elements": []map[string]interface{}{
				{
					"tag": "div",
					"text": map[string]interface{}{
						"content": "**Integration Test**\nThis is a test card message.",
						"tag":     "lark_md",
					},
				},
			},
		}

		message := notifyhub.NewMessage("Card Test").
			WithFeishuCard(cardContent).
			WithFeishuMentions("oc_test123").
			ToFeishuGroup("oc_group123").
			Build()

		receipt, err := hub.Send(ctx, message)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		if len(receipt.Results) == 0 {
			t.Error("Expected at least one result")
		}
	})

	// Test 4: SMS with template
	t.Run("SMS with template", func(t *testing.T) {
		message := notifyhub.NewMessage("Order Confirmation").
			WithSMSTemplate("order_confirmation").
			WithSMSVariables(map[string]interface{}{
				"order_id":    "TEST-12345",
				"total":       "$99.99",
				"customer":    "Test User",
				"delivery_at": time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04"),
			}).
			ToPhone("+1234567890").
			Build()

		receipt, err := hub.Send(ctx, message)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		if receipt.MessageID == "" {
			t.Error("Expected MessageID to be set")
		}
	})

	// Test 5: Scheduled message
	t.Run("Scheduled message", func(t *testing.T) {
		scheduleTime := time.Now().Add(1 * time.Minute)
		message := notifyhub.NewMessage("Scheduled Test").
			WithText("This message was scheduled").
			ScheduleAt(scheduleTime).
			ToEmail("test@example.com").
			Build()

		_, err := hub.Send(ctx, message)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		if message.ScheduledAt == nil {
			t.Error("Expected ScheduledAt to be set")
		}

		if !message.ScheduledAt.Equal(scheduleTime) {
			t.Errorf("Expected ScheduledAt %v, got %v", scheduleTime, *message.ScheduledAt)
		}
	})

	// Test 6: Async sending
	t.Run("Async sending", func(t *testing.T) {
		message := notifyhub.NewMessage("Async Test").
			WithText("This message is sent asynchronously").
			ToEmail("async@example.com").
			Build()

		asyncReceipt, err := hub.SendAsync(ctx, message)
		if err != nil {
			t.Fatalf("SendAsync failed: %v", err)
		}

		if asyncReceipt.MessageID != message.ID {
			t.Errorf("Expected MessageID %s, got %s", message.ID, asyncReceipt.MessageID)
		}

		if asyncReceipt.Status == "" {
			t.Error("Expected Status to be set")
		}

		if asyncReceipt.QueuedAt.IsZero() {
			t.Error("Expected QueuedAt to be set")
		}
	})

	// Test 7: Health check
	t.Run("Health check", func(t *testing.T) {
		health, err := hub.Health(ctx)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		if health.Status == "" {
			t.Error("Expected health status to be set")
		}

		if health.Platforms == nil {
			t.Error("Expected platforms health to be set")
		}

		if health.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set")
		}

		// Check that we have health info for configured platforms
		expectedPlatforms := []string{"email", "feishu", "sms"}
		for _, platform := range expectedPlatforms {
			if _, exists := health.Platforms[platform]; !exists {
				t.Errorf("Expected health info for platform %s", platform)
			}
		}
	})

	// Test 8: Multi-target complex message
	t.Run("Multi-target complex message", func(t *testing.T) {
		message := notifyhub.NewUrgent("Complex Multi-Target Test").
			WithMarkdown("## Urgent Notification\n\n- **Status**: Critical\n- **Action**: Required\n- **Deadline**: ASAP").
			WithMetadata("priority", "critical").
			WithMetadata("department", "engineering").
			WithVariable("incident_id", "INC-12345").
			WithVariable("affected_users", 1500).
			ToEmail("admin@example.com").
			ToEmail("manager@example.com").
			ToFeishu("oc_admin123").
			ToFeishuGroup("oc_engineering").
			ToPhone("+1234567890").
			Build()

		receipt, err := hub.Send(ctx, message)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		expectedTargets := 5 // 2 emails + 1 feishu user + 1 feishu group + 1 phone
		if len(receipt.Results) != expectedTargets {
			t.Errorf("Expected %d results, got %d", expectedTargets, len(receipt.Results))
		}

		// Verify that each platform received the message
		platformsSeen := make(map[string]bool)
		for _, result := range receipt.Results {
			platformsSeen[result.Platform] = true
		}

		expectedPlatforms := []string{"email", "feishu", "sms"}
		for _, platform := range expectedPlatforms {
			if !platformsSeen[platform] {
				t.Errorf("Expected platform %s to receive message", platform)
			}
		}
	})
}

// TestIntegrationErrorHandling tests error handling scenarios
func TestIntegrationErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	hub, err := notifyhub.NewHub(notifyhub.WithTestDefaults())
	if err != nil {
		t.Fatalf("Failed to create hub: %v", err)
	}
	defer hub.Close(context.Background())

	ctx := context.Background()

	t.Run("Invalid message validation", func(t *testing.T) {
		// Test message without title or body
		message := &notifyhub.Message{
			ID:           "test-id",
			Title:        "",
			Body:         "",
			Targets:      []notifyhub.Target{notifyhub.NewEmailTarget("test@example.com")},
			Metadata:     make(map[string]interface{}),
			Variables:    make(map[string]interface{}),
			PlatformData: make(map[string]interface{}),
		}

		_, err := hub.Send(ctx, message)
		if err == nil {
			t.Error("Expected Send to fail for invalid message")
		}

		if err == nil || !notifyhub.IsValidationError(err) {
			t.Errorf("Expected validation error, got: %v", err)
		}
	})

	t.Run("Message without targets", func(t *testing.T) {
		message := &notifyhub.Message{
			ID:           "test-id",
			Title:        "Test",
			Body:         "Test body",
			Targets:      []notifyhub.Target{},
			Metadata:     make(map[string]interface{}),
			Variables:    make(map[string]interface{}),
			PlatformData: make(map[string]interface{}),
		}

		_, err := hub.Send(ctx, message)
		if err == nil {
			t.Error("Expected Send to fail for message without targets")
		}
	})

	t.Run("Operations after close", func(t *testing.T) {
		tempHub, err := notifyhub.NewHub(notifyhub.WithTestDefaults())
		if err != nil {
			t.Fatalf("Failed to create temp hub: %v", err)
		}

		// Close the hub
		err = tempHub.Close(ctx)
		if err != nil {
			t.Fatalf("Failed to close hub: %v", err)
		}

		message := notifyhub.NewMessage("Test").
			WithText("Test").
			ToEmail("test@example.com").
			Build()

		// Try to send after close
		_, err = tempHub.Send(ctx, message)
		if err == nil {
			t.Error("Expected Send to fail after close")
		}

		// Try async send after close
		_, err = tempHub.SendAsync(ctx, message)
		if err == nil {
			t.Error("Expected SendAsync to fail after close")
		}

		// Try health check after close
		_, err = tempHub.Health(ctx)
		if err == nil {
			t.Error("Expected Health to fail after close")
		}
	})
}

// TestIntegrationConfiguration tests different configuration scenarios
func TestIntegrationConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("Default configuration", func(t *testing.T) {
		hub, err := notifyhub.NewHub()
		if err != nil {
			t.Fatalf("Failed to create hub with defaults: %v", err)
		}
		defer hub.Close(context.Background())

		// Should be able to check health even without platforms
		health, err := hub.Health(context.Background())
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}
		if health == nil {
			t.Error("Expected health status")
		}
	})

	t.Run("Test configuration", func(t *testing.T) {
		hub, err := notifyhub.NewHub(notifyhub.TestConfig()...)
		if err != nil {
			t.Fatalf("Failed to create hub with test config: %v", err)
		}
		defer hub.Close(context.Background())

		// Test config should work
		message := notifyhub.NewMessage("Test").
			WithText("Test message").
			ToEmail("test@example.com").
			Build()

		_, err = hub.Send(context.Background(), message)
		if err != nil {
			t.Errorf("Send failed with test config: %v", err)
		}
	})

	t.Run("Quick start configuration", func(t *testing.T) {
		hub, err := notifyhub.QuickStart().
			WithFeishuWebhook("https://test.example.com/webhook", "secret").
			WithEmailSMTP("smtp.test.com", 587, "user", "pass", "from@test.com").
			Build()
		if err != nil {
			t.Fatalf("Failed to create hub with quick start: %v", err)
		}
		defer hub.Close(context.Background())

		health, err := hub.Health(context.Background())
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}

		// Should have both email and feishu platforms
		if len(health.Platforms) == 0 {
			t.Error("Expected platforms to be configured")
		}
	})
}

// TestIntegrationConcurrency tests concurrent operations
func TestIntegrationConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	hub, err := notifyhub.NewHub(notifyhub.WithTestDefaults())
	if err != nil {
		t.Fatalf("Failed to create hub: %v", err)
	}
	defer hub.Close(context.Background())

	ctx := context.Background()

	// Send multiple messages concurrently
	const numMessages = 10
	results := make(chan error, numMessages)

	for i := 0; i < numMessages; i++ {
		go func(i int) {
			message := notifyhub.NewMessage("Concurrent Test").
				WithText("Concurrent message").
				WithMetadata("message_num", i).
				ToEmail("test@example.com").
				Build()

			_, err := hub.Send(ctx, message)
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numMessages; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent send %d failed: %v", i, err)
		}
	}

	// Test concurrent health checks
	const numHealthChecks = 5
	healthResults := make(chan error, numHealthChecks)

	for i := 0; i < numHealthChecks; i++ {
		go func() {
			_, err := hub.Health(ctx)
			healthResults <- err
		}()
	}

	// Collect health check results
	for i := 0; i < numHealthChecks; i++ {
		err := <-healthResults
		if err != nil {
			t.Errorf("Concurrent health check %d failed: %v", i, err)
		}
	}
}
