// Package main demonstrates how to register and use external platforms with NotifyHub
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub/examples/external_platform/slack"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
	fmt.Println("🚀 External Platform Extension Demo")
	fmt.Println("==================================")

	// Step 1: Register external platform BEFORE creating the hub
	fmt.Println("\n📋 Step 1: Registering External Platforms")
	fmt.Println("------------------------------------------")

	// Register Slack platform extension
	err := notifyhub.RegisterExtension(&notifyhub.PlatformExtension{
		Name:    "slack",
		Creator: slack.NewSlackSender,
		DefaultOpts: func() map[string]interface{} {
			return map[string]interface{}{
				"timeout": 30 * time.Second,
			}
		},
		Validator: func(config map[string]interface{}) error {
			if _, ok := config["webhook_url"].(string); !ok || config["webhook_url"].(string) == "" {
				return fmt.Errorf("webhook_url is required")
			}
			return nil
		},
	})
	if err != nil {
		log.Fatalf("Failed to register Slack platform: %v", err)
	}
	fmt.Println("✅ Slack platform registered successfully")

	// Step 2: Create hub with both internal and external platforms
	fmt.Println("\n🏗️ Step 2: Creating NotifyHub with Mixed Platforms")
	fmt.Println("--------------------------------------------------")

	hub, err := notifyhub.NewHub(
		// Internal platform (Feishu)
		notifyhub.WithFeishu("https://feishu.example.com/webhook", "feishu-secret"),

		// External platform (Slack) - using the new extension system
		notifyhub.WithSlack("https://example.com/slack/webhook/placeholder"),

		// Alternative: configure via generic platform config
		// notifyhub.WithPlatformConfig("slack", map[string]interface{}{
		//     "webhook_url": "https://example.com/slack/webhook/placeholder",
		//     "timeout":     30 * time.Second,
		// }),

		// Other settings
		notifyhub.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create hub: %v", err)
	}
	defer func() {
		if err := hub.Close(context.Background()); err != nil {
			log.Printf("Failed to close hub: %v", err)
		}
	}()

	ctx := context.Background()

	// Step 3: Send messages to mixed platforms
	fmt.Println("\n📤 Step 3: Sending Messages to Mixed Platforms")
	fmt.Println("----------------------------------------------")

	// Create a message that will be sent to multiple platforms
	message := notifyhub.NewMessage("Multi-Platform Alert").
		WithBody("🚨 System monitoring alert: CPU usage exceeded 90%").
		WithMetadata("alert_level", "high").
		WithMetadata("timestamp", time.Now().Format(time.RFC3339)).
		// Internal platform targets
		ToFeishuGroup("feishu_ops_group").
		// External platform targets
		ToTarget(notifyhub.NewTarget("channel", "#alerts", "slack")).
		ToTarget(notifyhub.NewTarget("user", "@admin", "slack")).
		Build()

	// Send the message
	receipt, err := hub.Send(ctx, message)
	if err != nil {
		log.Printf("Send failed: %v", err)
	} else {
		fmt.Printf("✅ Multi-platform send completed - Total: %d, Success: %d, Failed: %d\n",
			receipt.Total, receipt.Successful, receipt.Failed)

		// Show detailed results for each platform
		fmt.Println("\n📊 Detailed Results:")
		for _, result := range receipt.Results {
			status := "✅"
			if !result.Success {
				status = "❌"
			}
			fmt.Printf("  %s %s -> %s (%dms)\n",
				status, result.Target, result.Platform, result.Duration.Milliseconds())
			if result.Error != "" {
				fmt.Printf("      Error: %s\n", result.Error)
			}
		}
	}

	// Step 4: Test platform-specific features
	fmt.Println("\n🎯 Step 4: Platform-Specific Features")
	fmt.Println("------------------------------------")

	// Slack-specific message with rich blocks
	slackMessage := notifyhub.NewAlert("Slack Rich Message").
		WithBody("Deployment notification with rich formatting").
		WithPlatformData(map[string]interface{}{
			"slack_blocks": []map[string]interface{}{
				{
					"type": "header",
					"text": map[string]interface{}{
						"type":  "plain_text",
						"text":  "🚀 Deployment Complete",
						"emoji": true,
					},
				},
				{
					"type": "section",
					"text": map[string]interface{}{
						"type": "mrkdwn",
						"text": "*Version:* v2.1.0\n*Environment:* Production\n*Status:* :large_green_circle: Success",
					},
				},
				{
					"type": "actions",
					"elements": []map[string]interface{}{
						{
							"type": "button",
							"text": map[string]interface{}{
								"type":  "plain_text",
								"text":  "View Logs",
								"emoji": true,
							},
							"url": "https://logs.example.com",
						},
					},
				},
			},
		}).
		ToTarget(notifyhub.NewTarget("channel", "#deployments", "slack")).
		Build()

	receipt, err = hub.Send(ctx, slackMessage)
	if err != nil {
		log.Printf("Slack send failed: %v", err)
	} else {
		fmt.Printf("✅ Slack rich message sent - ID: %s\n", receipt.MessageID)
	}

	// Step 5: Health check for all platforms
	fmt.Println("\n🔍 Step 5: Multi-Platform Health Check")
	fmt.Println("-------------------------------------")

	health, err := hub.Health(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Printf("🏥 Overall system health: %s\n", health.Status)
		fmt.Printf("📅 Last check: %s\n", health.Timestamp.Format("15:04:05"))

		fmt.Println("\n📋 Platform Health Details:")
		for platform, status := range health.Platforms {
			healthIcon := "🟢"
			if !status.Available {
				healthIcon = "🔴"
			}
			fmt.Printf("  %s %s: %s\n", healthIcon, platform, status.Status)
		}
	}

	fmt.Println("\n✨ External Platform Demo Completed!")
	fmt.Println("===================================")
}

// Helper functions for external platform targets
// These would typically be added to the main notifyhub package

// ToSlackChannel adds a Slack channel target
func ToSlackChannel(mb *notifyhub.MessageBuilder, channel string) *notifyhub.MessageBuilder {
	return mb.ToTarget(notifyhub.NewTarget("channel", channel, "slack"))
}

// ToSlackUser adds a Slack user target
func ToSlackUser(mb *notifyhub.MessageBuilder, user string) *notifyhub.MessageBuilder {
	return mb.ToTarget(notifyhub.NewTarget("user", user, "slack"))
}
