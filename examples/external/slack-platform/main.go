// Package main demonstrates creating a Slack platform integration
// This shows how to create external platforms for popular services like Slack
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kart-io/notifyhub/examples/external/slack-platform/slack"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
	fmt.Println("💬 Slack Platform Integration Demo")
	fmt.Println("=================================")
	fmt.Println()
	fmt.Println("This demo shows how to create a Slack platform integration")
	fmt.Println("that seamlessly works with NotifyHub's unified architecture.")
	fmt.Println()

	// Step 1: Platform Registration
	fmt.Println("📋 Step 1: Platform Registration")
	fmt.Println("------------------------------")
	fmt.Println("Available platforms:")
	showAvailablePlatforms()
	fmt.Println()

	// Step 2: Create Hub with Slack Integration
	fmt.Println("💬 Step 2: Slack Integration")
	fmt.Println("---------------------------")

	// Get webhook URL from environment variable
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		log.Fatal("SLACK_WEBHOOK_URL environment variable not set")
	}

	// Create a new hub with Slack platform
	hub, err := notifyhub.New(
		slack.WithSlack(webhookURL, slack.WithSlackChannel("#random")),
	)
	if err != nil {
		log.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Close() }()

	fmt.Println("✅ Slack platform configured")
	fmt.Printf("   💬 Channel: #notifications\n")
	fmt.Printf("   🤖 Username: NotifyHub Bot\n")
	fmt.Printf("   🔔 Icon: :bell:\n")
	fmt.Println()

	ctx := context.Background()

	// Step 3: Send Basic Message
	fmt.Println("📤 Step 3: Basic Slack Message")
	fmt.Println("----------------------------")

	basicMsg := notifyhub.NewMessage("Basic Slack Test").
		WithBody("This is a basic message sent to Slack via NotifyHub.").
		WithMetadata("service", "notifyhub").
		WithMetadata("environment", "demo").
		ToTarget(notifyhub.NewTarget("webhook", "", "slack")).
		Build()

	receipt, err := hub.Send(ctx, basicMsg)
	if err != nil {
		fmt.Printf("❌ Basic message failed: %v\n", err)
	} else {
		fmt.Printf("✅ Basic message sent (Duration: %dms)\n",
			receipt.Results[0].Duration.Milliseconds())
	}
	fmt.Println()

	// Step 4: Rich Slack Attachments
	fmt.Println("🎨 Step 4: Rich Slack Attachments")
	fmt.Println("-------------------------------")

	richMsg := notifyhub.NewAlert("System Alert").
		WithBody("This message uses Slack's rich attachment formatting.").
		WithPlatformData(map[string]interface{}{
			"slack_attachments": []map[string]interface{}{
				{
					"color": "danger",
					"title": "🚨 System Alert",
					"text":  "Database connection pool exhausted",
					"fields": []map[string]interface{}{
						{
							"title": "Service",
							"value": "user-api",
							"short": true,
						},
						{
							"title": "Environment",
							"value": "production",
							"short": true,
						},
						{
							"title": "Severity",
							"value": "High",
							"short": true,
						},
						{
							"title": "Duration",
							"value": "2m 30s",
							"short": true,
						},
					},
					"footer":    "AlertManager",
					"ts":        time.Now().Unix(),
					"mrkdwn_in": []string{"text"},
				},
			},
		}).
		ToTarget(notifyhub.NewTarget("webhook", "", "slack")).
		Build()

	if _, err := hub.Send(ctx, richMsg); err != nil {
		fmt.Printf("❌ Rich message failed: %v\n", err)
	} else {
		fmt.Printf("✅ Rich attachment sent\n")
	}
	fmt.Println()

	// Step 5: Slack Blocks (Modern UI)
	fmt.Println("🧱 Step 5: Slack Blocks (Modern UI)")
	fmt.Println("---------------------------------")

	blocksMsg := notifyhub.NewMessage("Interactive Slack Block").
		WithBody("Fallback text for notifications").
		WithPlatformData(map[string]interface{}{
			"slack_blocks": []map[string]interface{}{
				{
					"type": "header",
					"text": map[string]interface{}{
						"type":  "plain_text",
						"text":  "📊 Daily Report",
						"emoji": true,
					},
				},
				{
					"type": "section",
					"text": map[string]interface{}{
						"type": "mrkdwn",
						"text": "*System Status:* All systems operational\n*Uptime:* 99.9%\n*Active Users:* 1,247",
					},
					"accessory": map[string]interface{}{
						"type": "button",
						"text": map[string]interface{}{
							"type":  "plain_text",
							"text":  "View Dashboard",
							"emoji": true,
						},
						"value": "dashboard_click",
						"url":   "https://dashboard.example.com",
					},
				},
				{
					"type": "divider",
				},
				{
					"type": "section",
					"fields": []map[string]interface{}{
						{
							"type": "mrkdwn",
							"text": "*API Requests:*\n15,420",
						},
						{
							"type": "mrkdwn",
							"text": "*Response Time:*\n145ms avg",
						},
						{
							"type": "mrkdwn",
							"text": "*Error Rate:*\n0.02%",
						},
						{
							"type": "mrkdwn",
							"text": "*Throughput:*\n2.1k req/min",
						},
					},
				},
			},
		}).
		ToTarget(notifyhub.NewTarget("webhook", "", "slack")).
		Build()

	if _, err := hub.Send(ctx, blocksMsg); err != nil {
		fmt.Printf("❌ Blocks message failed: %v\n", err)
	} else {
		fmt.Printf("✅ Slack blocks sent\n")
	}
	fmt.Println()

	// Step 6: Channel-Specific Messaging
	fmt.Println("🎯 Step 6: Channel-Specific Messaging")
	fmt.Println("------------------------------------")

	channels := []string{"#general", "#alerts", "#development"}

	for _, channel := range channels {
		channelMsg := notifyhub.NewMessage(fmt.Sprintf("Channel Test for %s", channel)).
			WithBody(fmt.Sprintf("This message is specifically for %s channel.", channel)).
			WithMetadata("channel", channel).
			ToTarget(notifyhub.NewTarget("channel", channel, "slack")).
			Build()

		if _, err := hub.Send(ctx, channelMsg); err != nil {
			fmt.Printf("❌ %s message failed: %v\n", channel, err)
		} else {
			fmt.Printf("✅ %s message sent\n", channel)
		}
	}
	fmt.Println()

	// Step 7: Priority-Based Formatting
	fmt.Println("🚨 Step 7: Priority-Based Formatting")
	fmt.Println("-----------------------------------")

	priorities := []struct {
		level   string
		message *notifyhub.Message
	}{
		{
			level: "Low Priority",
			message: notifyhub.NewMessage("Low Priority Update").
				WithBody("Routine maintenance scheduled for tonight.").
				Build(),
		},
		{
			level: "Normal Priority",
			message: notifyhub.NewMessage("System Update").
				WithBody("New features deployed successfully.").
				Build(),
		},
		{
			level: "Alert Priority",
			message: notifyhub.NewAlert("Performance Warning").
				WithBody("Response time increased above threshold.").
				Build(),
		},
		{
			level: "Urgent Priority",
			message: notifyhub.NewUrgent("Critical System Error").
				WithBody("Payment system is experiencing errors.").
				Build(),
		},
	}

	for _, p := range priorities {
		p.message.Targets = append(p.message.Targets, notifyhub.NewTarget("webhook", "", "slack"))

		if _, err := hub.Send(ctx, p.message); err != nil {
			fmt.Printf("❌ %s failed: %v\n", p.level, err)
		} else {
			fmt.Printf("✅ %s sent (auto-colored by priority)\n", p.level)
		}

		time.Sleep(500 * time.Millisecond) // Rate limit messages
	}
	fmt.Println()

	// Step 8: Slack-Specific Features
	fmt.Println("🔧 Step 8: Slack-Specific Features")
	fmt.Println("---------------------------------")
	fmt.Println("The Slack platform integration provides:")
	fmt.Println()
	fmt.Println("💬 MESSAGING FEATURES:")
	fmt.Println("   • Rich attachments with colors and fields")
	fmt.Println("   • Modern Slack blocks with interactive elements")
	fmt.Println("   • Channel-specific message targeting")
	fmt.Println("   • User mentions and thread support")
	fmt.Println()
	fmt.Println("🎨 FORMATTING OPTIONS:")
	fmt.Println("   • Markdown text formatting")
	fmt.Println("   • Custom colors based on message priority")
	fmt.Println("   • Emoji and icon customization")
	fmt.Println("   • Field layouts and dividers")
	fmt.Println()
	fmt.Println("🔧 INTEGRATION CAPABILITIES:")
	fmt.Println("   • Webhook URL configuration")
	fmt.Println("   • Custom bot username and avatars")
	fmt.Println("   • Channel routing and targeting")
	fmt.Println("   • Timeout and retry handling")
	fmt.Println()

	fmt.Println("💬 Slack Platform Integration Demo Complete!")
	fmt.Println()
	fmt.Println("This external platform demonstrates:")
	fmt.Println("• Seamless integration with NotifyHub architecture")
	fmt.Println("• Full feature parity with built-in platforms")
	fmt.Println("• Slack-specific rich content and formatting")
	fmt.Println("• Production-ready error handling and validation")
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
