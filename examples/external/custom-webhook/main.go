// Package main demonstrates creating a custom webhook platform
// This shows how to create a generic webhook platform as an external package
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub/examples/external/custom-webhook/webhook"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
	fmt.Println("🔗 Custom Webhook Platform Demo")
	fmt.Println("==============================")
	fmt.Println()
	fmt.Println("This demo shows how to create a generic webhook platform")
	fmt.Println("that can integrate with any webhook-based service.")
	fmt.Println()

	// Step 1: Platform Registration
	fmt.Println("📋 Step 1: Platform Registration")
	fmt.Println("------------------------------")
	fmt.Println("Available platforms:")
	showAvailablePlatforms()
	fmt.Println()

	// Step 2: Create Hub with Custom Webhook
	fmt.Println("🚀 Step 2: Custom Webhook Integration")
	fmt.Println("-----------------------------------")

	hub, err := notifyhub.NewHub(
		webhook.WithWebhook("https://httpbin.org/post",
			webhook.WithWebhookMethod("POST"),
			webhook.WithWebhookContentType("application/json"),
			webhook.WithWebhookTimeout(30*time.Second),
			webhook.WithWebhookHeaders(map[string]string{
				"Authorization": "Bearer demo-token",
				"X-API-Version": "v1",
			}),
		),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create webhook hub: %v", err)
	}
	defer func() { _ = hub.Close(context.Background()) }()

	fmt.Println("✅ Custom webhook platform configured")
	fmt.Printf("   🔗 URL: https://httpbin.org/post\n")
	fmt.Printf("   📄 Content-Type: application/json\n")
	fmt.Printf("   🔑 Authentication: Bearer token\n")
	fmt.Println()

	ctx := context.Background()

	// Step 3: Send Basic Webhook
	fmt.Println("📤 Step 3: Basic Webhook Message")
	fmt.Println("------------------------------")

	basicMsg := notifyhub.NewMessage("Custom Webhook Test").
		WithBody("This message is sent to a custom webhook endpoint.").
		WithMetadata("service", "notifyhub").
		WithMetadata("environment", "demo").
		ToTarget(notifyhub.NewTarget("webhook", "", "webhook")).
		Build()

	receipt, err := hub.Send(ctx, basicMsg)
	if err != nil {
		fmt.Printf("❌ Basic webhook failed: %v\n", err)
	} else {
		fmt.Printf("✅ Basic webhook sent (Duration: %dms)\n",
			receipt.Results[0].Duration.Milliseconds())
	}
	fmt.Println()

	// Step 4: Custom Payload Format
	fmt.Println("🎨 Step 4: Custom Payload Formatting")
	fmt.Println("----------------------------------")

	customPayloadMsg := notifyhub.NewAlert("Custom Payload Alert").
		WithBody("This webhook uses custom payload formatting.").
		WithPlatformData(map[string]interface{}{
			"webhook_payload": map[string]interface{}{
				"alert": map[string]interface{}{
					"severity": "high",
					"source":   "notifyhub-demo",
					"message":  "Custom formatted webhook message",
					"metadata": map[string]interface{}{
						"timestamp": time.Now().Unix(),
						"version":   "1.0.0",
					},
				},
				"context": map[string]interface{}{
					"user_id":    "demo-user",
					"session_id": "demo-session-123",
				},
			},
		}).
		ToTarget(notifyhub.NewTarget("webhook", "", "webhook")).
		Build()

	if _, err := hub.Send(ctx, customPayloadMsg); err != nil {
		fmt.Printf("❌ Custom payload failed: %v\n", err)
	} else {
		fmt.Printf("✅ Custom payload sent\n")
	}
	fmt.Println()

	// Step 5: Multiple Webhook Configurations
	fmt.Println("🔧 Step 5: Multiple Webhook Configurations")
	fmt.Println("----------------------------------------")

	// Slack-compatible webhook
	slackHub, err := notifyhub.NewHub(
		webhook.WithWebhook("https://hooks.slack.com/services/demo/webhook",
			webhook.WithWebhookMethod("POST"),
			webhook.WithWebhookContentType("application/json"),
			webhook.WithWebhookPayloadTemplate(map[string]interface{}{
				"text":     "{{message}}",
				"username": "NotifyHub Bot",
				"channel":  "#general",
			}),
		),
	)
	if err != nil {
		fmt.Printf("❌ Slack webhook hub failed: %v\n", err)
	} else {
		defer func() { _ = slackHub.Close(context.Background()) }()
		fmt.Println("✅ Slack-compatible webhook configured")
	}

	// Microsoft Teams webhook
	teamsHub, err := notifyhub.NewHub(
		webhook.WithWebhook("https://outlook.office.com/webhook/demo",
			webhook.WithWebhookMethod("POST"),
			webhook.WithWebhookContentType("application/json"),
			webhook.WithWebhookPayloadTemplate(map[string]interface{}{
				"@type":    "MessageCard",
				"@context": "http://schema.org/extensions",
				"summary":  "{{title}}",
				"text":     "{{message}}",
			}),
		),
	)
	if err != nil {
		fmt.Printf("❌ Teams webhook hub failed: %v\n", err)
	} else {
		defer func() { _ = teamsHub.Close(context.Background()) }()
		fmt.Println("✅ Teams-compatible webhook configured")
	}

	// Generic webhook with form data
	formHub, err := notifyhub.NewHub(
		webhook.WithWebhook("https://httpbin.org/post",
			webhook.WithWebhookMethod("POST"),
			webhook.WithWebhookContentType("application/x-www-form-urlencoded"),
		),
	)
	if err != nil {
		fmt.Printf("❌ Form webhook hub failed: %v\n", err)
	} else {
		defer func() { _ = formHub.Close(context.Background()) }()
		fmt.Println("✅ Form data webhook configured")
	}
	fmt.Println()

	// Step 6: Use Cases
	fmt.Println("💼 Step 6: Common Webhook Use Cases")
	fmt.Println("---------------------------------")

	useCases := map[string]*notifyhub.Message{
		"Monitoring Alert": notifyhub.NewAlert("Server Down").
			WithBody("Production server is not responding").
			WithPlatformData(map[string]interface{}{
				"webhook_payload": map[string]interface{}{
					"alert_type": "server_down",
					"severity":   "critical",
					"server":     "prod-web-01",
					"timestamp":  time.Now().Unix(),
				},
			}).
			Build(),

		"CI/CD Notification": notifyhub.NewMessage("Build Complete").
			WithBody("Build #123 completed successfully").
			WithPlatformData(map[string]interface{}{
				"webhook_payload": map[string]interface{}{
					"build_id":     123,
					"status":       "success",
					"branch":       "main",
					"commit":       "abc123def",
					"duration":     "2m 30s",
					"triggered_by": "john.doe",
				},
			}).
			Build(),

		"Security Event": notifyhub.NewUrgent("Security Alert").
			WithBody("Suspicious login detected").
			WithPlatformData(map[string]interface{}{
				"webhook_payload": map[string]interface{}{
					"event_type": "suspicious_login",
					"user_id":    "user_456",
					"ip_address": "192.168.1.100",
					"location":   "Unknown",
					"timestamp":  time.Now().Unix(),
					"risk_score": 85,
				},
			}).
			Build(),

		"Business Metric": notifyhub.NewMessage("Daily Report").
			WithBody("Daily metrics summary").
			WithPlatformData(map[string]interface{}{
				"webhook_payload": map[string]interface{}{
					"date":         time.Now().Format("2006-01-02"),
					"active_users": 1247,
					"revenue":      25630.50,
					"conversions":  89,
					"page_views":   15420,
					"bounce_rate":  0.32,
				},
			}).
			Build(),
	}

	for useCase, msg := range useCases {
		msg.Targets = append(msg.Targets, notifyhub.NewTarget("webhook", "", "webhook"))
		if _, err := hub.Send(ctx, msg); err != nil {
			fmt.Printf("❌ %s failed: %v\n", useCase, err)
		} else {
			fmt.Printf("✅ %s sent\n", useCase)
		}
	}
	fmt.Println()

	// Step 7: Advanced Features
	fmt.Println("🚀 Step 7: Advanced Webhook Features")
	fmt.Println("----------------------------------")
	fmt.Println("The custom webhook platform supports:")
	fmt.Println()
	fmt.Println("🔧 CONFIGURATION OPTIONS:")
	fmt.Println("   • Custom HTTP methods (POST, PUT, PATCH)")
	fmt.Println("   • Content types (JSON, form data, XML)")
	fmt.Println("   • Custom headers and authentication")
	fmt.Println("   • Configurable timeouts and retries")
	fmt.Println()
	fmt.Println("🎨 PAYLOAD FORMATTING:")
	fmt.Println("   • Custom payload templates")
	fmt.Println("   • Variable substitution")
	fmt.Println("   • Platform-specific formats")
	fmt.Println("   • Nested data structures")
	fmt.Println()
	fmt.Println("🔗 INTEGRATION TARGETS:")
	fmt.Println("   • Slack webhooks")
	fmt.Println("   • Microsoft Teams connectors")
	fmt.Println("   • Custom monitoring systems")
	fmt.Println("   • CI/CD pipelines")
	fmt.Println("   • Business intelligence tools")
	fmt.Println()

	fmt.Println("🔗 Custom Webhook Platform Demo Complete!")
	fmt.Println()
	fmt.Println("This demo shows how external platforms can:")
	fmt.Println("• Integrate with any webhook-based service")
	fmt.Println("• Provide flexible configuration options")
	fmt.Println("• Support custom payload formatting")
	fmt.Println("• Work seamlessly with the unified architecture")
}

func showAvailablePlatforms() {
	platforms := notifyhub.GetAvailablePlatforms()
	for _, platform := range platforms {
		platformType := "Built-in"
		if platform.IsExtension {
			platformType = "External"
		}
		fmt.Printf("  📦 %s (%s)\n", platform.Name, platformType)
	}
}
