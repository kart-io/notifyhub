// Package main demonstrates Feishu platform-specific features
// This shows the complete Feishu integration capabilities in the unified architecture
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/feishu"
)

func main() {
	fmt.Println("🚀 Feishu Platform Features Demo")
	fmt.Println("===============================")
	fmt.Println()

	// Part 1: Different authentication modes
	fmt.Println("🔐 Part 1: Authentication Modes")
	fmt.Println("------------------------------")

	// 1. No authentication (webhook only)
	fmt.Println("1. No Authentication Mode")
	noAuthHub, err := notifyhub.NewHub(
		feishu.WithFeishu("https://example.com/feishu/webhook"),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create no-auth hub: %v", err)
	}
	defer func() { _ = noAuthHub.Close(context.Background()) }()
	fmt.Println("✅ No-auth hub created")

	// 2. Signature authentication (HMAC-SHA256)
	fmt.Println("2. Signature Authentication Mode")
	signatureHub, err := notifyhub.NewHub(
		feishu.WithFeishu("https://example.com/feishu/webhook",
			feishu.WithFeishuSecret("your-webhook-secret"),
			feishu.WithFeishuAuthMode(feishu.AuthModeSignature),
		),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create signature hub: %v", err)
	}
	defer func() { _ = signatureHub.Close(context.Background()) }()
	fmt.Println("✅ Signature-auth hub created")

	// 3. Keywords authentication
	fmt.Println("3. Keywords Authentication Mode")
	keywordsHub, err := notifyhub.NewHub(
		feishu.WithFeishu("https://example.com/feishu/webhook",
			feishu.WithFeishuKeywords([]string{"alert", "notification"}),
		),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create keywords hub: %v", err)
	}
	defer func() { _ = keywordsHub.Close(context.Background()) }()
	fmt.Println("✅ Keywords-auth hub created")
	fmt.Println()

	ctx := context.Background()

	// Part 2: Basic Text Messages
	fmt.Println("📝 Part 2: Basic Text Messages")
	fmt.Println("----------------------------")

	// Simple text message
	simpleMsg := notifyhub.NewMessage("Simple Feishu Message").
		WithBody("This is a basic text message sent to Feishu.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	if receipt, err := noAuthHub.Send(ctx, simpleMsg); err != nil {
		fmt.Printf("❌ Simple message failed: %v\n", err)
	} else {
		fmt.Printf("✅ Simple message sent (Duration: %dms)\n",
			receipt.Results[0].Duration.Milliseconds())
	}

	// Message with title and body
	titleBodyMsg := notifyhub.NewMessage("📊 System Report").
		WithBody("System health check completed successfully.\n" +
			"• CPU Usage: 45%\n" +
			"• Memory Usage: 62%\n" +
			"• Disk Usage: 78%").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	if receipt, err := noAuthHub.Send(ctx, titleBodyMsg); err != nil {
		fmt.Printf("❌ Title+body message failed: %v\n", err)
	} else {
		fmt.Printf("✅ Title+body message sent (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	}
	fmt.Println()

	// Part 3: Mentions and Special Features
	fmt.Println("👥 Part 3: Mentions and Special Features")
	fmt.Println("--------------------------------------")

	// Message with @all mention
	mentionAllMsg := notifyhub.NewAlert("🚨 Critical Alert").
		WithBody("Database connection pool is critically low!").
		WithPlatformData(map[string]interface{}{
			"feishu_mention_all": true,
		}).
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	if receipt, err := noAuthHub.Send(ctx, mentionAllMsg); err != nil {
		fmt.Printf("❌ Mention all failed: %v\n", err)
	} else {
		fmt.Printf("✅ @all mention sent (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	}

	// Message with specific user mentions
	userMentionsMsg := notifyhub.NewMessage("👋 User Mentions").
		WithBody("Please review the latest deployment status.").
		WithPlatformData(map[string]interface{}{
			"feishu_mentions": []map[string]interface{}{
				{"user_id": "ou_123456789"},
				{"user_id": "ou_987654321"},
			},
		}).
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	if receipt, err := noAuthHub.Send(ctx, userMentionsMsg); err != nil {
		fmt.Printf("❌ User mentions failed: %v\n", err)
	} else {
		fmt.Printf("✅ User mentions sent (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	}
	fmt.Println()

	// Part 4: Rich Content (Cards and Posts)
	fmt.Println("🎨 Part 4: Rich Content")
	fmt.Println("---------------------")

	// Interactive card message
	cardMsg := notifyhub.NewMessage("Interactive Card").
		WithBody("Fallback text for non-card clients").
		WithPlatformData(map[string]interface{}{
			"feishu_card": map[string]interface{}{
				"header": map[string]interface{}{
					"title": map[string]interface{}{
						"tag":     "plain_text",
						"content": "🔔 System Notification",
					},
					"template": "blue",
				},
				"elements": []map[string]interface{}{
					{
						"tag": "div",
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": "**Status**: ✅ Healthy\n**Uptime**: 99.9%\n**Last Check**: " + time.Now().Format("2006-01-02 15:04:05"),
						},
					},
					{
						"tag": "action",
						"actions": []map[string]interface{}{
							{
								"tag": "button",
								"text": map[string]interface{}{
									"tag":     "plain_text",
									"content": "View Details",
								},
								"type": "primary",
								"url":  "https://monitor.example.com",
							},
						},
					},
				},
			},
		}).
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	if receipt, err := noAuthHub.Send(ctx, cardMsg); err != nil {
		fmt.Printf("❌ Card message failed: %v\n", err)
	} else {
		fmt.Printf("✅ Interactive card sent (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	}

	// Rich text post
	richPostMsg := notifyhub.NewMessage("Rich Text Post").
		WithBody("Fallback for rich post").
		WithPlatformData(map[string]interface{}{
			"feishu_post": map[string]interface{}{
				"zh_cn": map[string]interface{}{
					"title": "📈 Daily Report",
					"content": [][]map[string]interface{}{
						{
							{
								"tag":  "text",
								"text": "Today's metrics look great! ",
							},
							{
								"tag":  "text",
								"text": "Traffic increased by ",
							},
							{
								"tag":   "text",
								"text":  "25%",
								"style": []string{"bold"},
							},
							{
								"tag":  "text",
								"text": " compared to yesterday.",
							},
						},
						{
							{
								"tag":  "text",
								"text": "\n📊 Key Metrics:\n• Active Users: 1,250\n• Conversion Rate: 3.2%\n• Response Time: 120ms",
							},
						},
					},
				},
			},
		}).
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	if receipt, err := noAuthHub.Send(ctx, richPostMsg); err != nil {
		fmt.Printf("❌ Rich post failed: %v\n", err)
	} else {
		fmt.Printf("✅ Rich text post sent (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	}
	fmt.Println()

	// Part 5: Different Target Types
	fmt.Println("🎯 Part 5: Different Target Types")
	fmt.Println("-------------------------------")

	// Webhook target (default)
	webhookMsg := notifyhub.NewMessage("Webhook Target").
		WithBody("Message sent to webhook endpoint.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	// Group target
	groupMsg := notifyhub.NewMessage("Group Target").
		WithBody("Message sent to specific group.").
		ToTarget(notifyhub.NewTarget("group", "oc_group123456789", "feishu")).
		Build()

	// User target
	userMsg := notifyhub.NewMessage("User Target").
		WithBody("Private message to specific user.").
		ToTarget(notifyhub.NewTarget("user", "ou_user123456789", "feishu")).
		Build()

	// Channel target
	channelMsg := notifyhub.NewMessage("Channel Target").
		WithBody("Message sent to specific channel.").
		ToTarget(notifyhub.NewTarget("channel", "oc_channel123456789", "feishu")).
		Build()

	targets := map[string]*notifyhub.Message{
		"Webhook": webhookMsg,
		"Group":   groupMsg,
		"User":    userMsg,
		"Channel": channelMsg,
	}

	for targetType, msg := range targets {
		if receipt, err := noAuthHub.Send(ctx, msg); err != nil {
			fmt.Printf("❌ %s target failed: %v\n", targetType, err)
		} else {
			fmt.Printf("✅ %s target sent (Duration: %dms)\n", targetType, receipt.Results[0].Duration.Milliseconds())
		}
	}
	fmt.Println()

	// Part 6: Authentication Mode Demonstrations
	fmt.Println("🔒 Part 6: Authentication Demonstrations")
	fmt.Println("-------------------------------------")

	// Keywords authentication (message will be prefixed)
	keywordMsg := notifyhub.NewMessage("Keyword Auth Test").
		WithBody("This message uses keyword authentication.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	fmt.Println("Sending with keywords authentication...")
	if receipt, err := keywordsHub.Send(ctx, keywordMsg); err != nil {
		fmt.Printf("❌ Keywords auth failed: %v\n", err)
	} else {
		fmt.Printf("✅ Keywords auth message sent (keyword will be prepended, Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	}

	// Signature authentication (will include HMAC signature)
	signatureMsg := notifyhub.NewMessage("Signature Auth Test").
		WithBody("This message includes HMAC-SHA256 signature.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	fmt.Println("Sending with signature authentication...")
	if receipt, err := signatureHub.Send(ctx, signatureMsg); err != nil {
		fmt.Printf("❌ Signature auth failed: %v\n", err)
	} else {
		fmt.Printf("✅ Signature auth message sent (includes HMAC signature, Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	}
	fmt.Println()

	// Part 7: Advanced Configuration
	fmt.Println("⚙️  Part 7: Advanced Configuration")
	fmt.Println("--------------------------------")

	// Custom timeout and advanced settings
	advancedHub, err := notifyhub.NewHub(
		feishu.WithFeishu("https://example.com/feishu/webhook",
			feishu.WithFeishuSecret("webhook-secret"),
			feishu.WithFeishuTimeout(45*time.Second),
			feishu.WithFeishuAuthMode(feishu.AuthModeSignature),
		),
	)
	if err != nil {
		log.Printf("❌ Advanced hub creation failed: %v", err)
	} else {
		defer func() { _ = advancedHub.Close(context.Background()) }()
		fmt.Println("✅ Advanced configuration hub created")

		// Test with advanced hub
		advancedMsg := notifyhub.NewMessage("Advanced Configuration").
			WithBody("Message sent with advanced Feishu configuration.").
			ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
			Build()

		if receipt, err := advancedHub.Send(ctx, advancedMsg); err != nil {
			fmt.Printf("❌ Advanced config test failed: %v\n", err)
		} else {
			fmt.Printf("✅ Advanced config test sent (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
		}
	}
	fmt.Println()

	// Part 8: Modern Configuration Only
	fmt.Println("🔄 Part 8: Modern Configuration")
	fmt.Println("------------------------------")

	// Note: Legacy WithFeishu function has been removed in this refactor
	// All Feishu configuration now uses the modern feishu.WithFeishu approach
	fmt.Println("✅ All Feishu configuration now uses modern platform packages:")
	fmt.Println("  • Use feishu.WithFeishu() for configuration")
	fmt.Println("  • Deprecated notifyhub.WithFeishu() has been removed")
	fmt.Println("  • This ensures consistency across all platforms")
	fmt.Println()

	// Summary
	fmt.Println("📋 Feishu Platform Features Summary")
	fmt.Println("=================================")
	fmt.Println("✅ AUTHENTICATION MODES:")
	fmt.Println("  • None - Simple webhook")
	fmt.Println("  • Signature - HMAC-SHA256 verification")
	fmt.Println("  • Keywords - Custom keyword prefixing")
	fmt.Println()
	fmt.Println("✅ MESSAGE TYPES:")
	fmt.Println("  • Simple text messages")
	fmt.Println("  • Rich text posts with formatting")
	fmt.Println("  • Interactive cards with buttons")
	fmt.Println("  • Messages with mentions (@all, @user)")
	fmt.Println()
	fmt.Println("✅ TARGET TYPES:")
	fmt.Println("  • Webhook - Bot webhook endpoint")
	fmt.Println("  • Group - Specific group chat")
	fmt.Println("  • User - Private user message")
	fmt.Println("  • Channel - Specific channel")
	fmt.Println()
	fmt.Println("✅ CONFIGURATION OPTIONS:")
	fmt.Println("  • Custom timeouts")
	fmt.Println("  • Flexible authentication")
	fmt.Println("  • Rich platform-specific data")
	fmt.Println("  • Backward compatibility")
	fmt.Println()

	fmt.Println("🚀 Feishu Platform Demo Complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("• Try examples/platforms/unified-demo/ for all platforms together")
	fmt.Println("• See examples/external/ for creating custom platforms")
	fmt.Println("• Check examples/advanced/ for production patterns")
}
