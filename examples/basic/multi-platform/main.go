// Package main demonstrates using multiple platforms together
// This shows how the unified architecture makes multi-platform integration seamless
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/email"
	"github.com/kart-io/notifyhub/pkg/platforms/feishu"
	"github.com/kart-io/notifyhub/pkg/platforms/sms"
)

func main() {
	log := logger.New().LogMode(logger.Info)

	log.Info("🌐 Multi-Platform NotifyHub Demo")
	log.Info("===============================")

	// Step 1: Create hub with multiple platforms
	log.Info("Step 1: Setting up multi-platform hub...")

	hub, err := notifyhub.New(
		// Feishu for team notifications
		feishu.WithFeishu(
			"https://example.com/feishu/webhook",
			feishu.WithFeishuSecret("demo-secret"),
			feishu.WithFeishuTimeout(30*time.Second),
		),

		// Email for external notifications
		email.WithEmail(
			"smtp.example.com", 587, "notifications@company.com",
			email.WithEmailAuth("smtp-username", "smtp-password"),
			email.WithEmailTLS(true),
			email.WithEmailTimeout(45*time.Second),
		),

		// SMS for urgent alerts
		sms.WithSMSTwilio(
			"demo-twilio-api-key",
			"+1234567890", // From number
			sms.WithSMSTimeout(20*time.Second),
		),
	)
	if err != nil {
		log.Error("❌ Failed to create hub", "error", err)
		os.Exit(1)
	}
	defer func() { _ = hub.Close() }()

	log.Info("✅ Multi-platform hub created!")
	log.Info("📱 Platforms configured: Feishu, Email, SMS")

	ctx := context.Background()

	// Step 2: Send to all platforms
	log.Info("Step 2: Broadcasting message to all platforms...")

	broadcastMsg := notifyhub.NewMessage("Multi-Platform Broadcast").
		WithBody("🌐 This message is sent to multiple platforms using the unified architecture!").
		WithMetadata("broadcast_id", "demo-001").
		// Target each platform
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		ToTarget(notifyhub.NewTarget("email", "admin@company.com", "email")).
		ToTarget(notifyhub.NewTarget("phone", "+1987654321", "sms")).
		Build()

	receipt, err := hub.Send(ctx, broadcastMsg)
	if err != nil {
		log.Error("❌ Broadcast failed", "error", err)
	} else {
		log.Info("✅ Broadcast sent to platforms!", "count", len(receipt.Results))
		for _, result := range receipt.Results {
			status := "✅"
			if !result.Success {
				status = "❌"
			}
			fmt.Printf("   %s %s -> %s (%dms)\n",
				status, result.Platform, result.Target, result.Duration.Milliseconds())
		}
	}
	fmt.Println()

	// Step 3: Platform-specific messaging
	fmt.Println("Step 3: Platform-specific messaging...")

	// Feishu with rich content
	feishuMsg := notifyhub.NewAlert("Team Alert").
		WithBody("Feishu supports rich formatting and mentions.").
		WithPlatformData(map[string]interface{}{
			"feishu_mention_all": true, // @all mention
		}).
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	// Email with HTML content
	emailMsg := notifyhub.NewMessage("HTML Email Report").
		WithBody("<h1>📊 Daily Report</h1><p>Today's metrics look <strong>great</strong>!</p>").
		WithFormat("html").
		WithPlatformData(map[string]interface{}{
			"email_cc":       []string{"manager@company.com"},
			"email_priority": "high",
		}).
		ToTarget(notifyhub.NewTarget("email", "team@company.com", "email")).
		Build()

	// SMS with template variables
	smsMsg := notifyhub.NewUrgent("Account Alert").
		WithPlatformData(map[string]interface{}{
			"sms_template": "ALERT: Login from {{location}} at {{time}}",
			"sms_variables": map[string]interface{}{
				"location": "New York",
				"time":     time.Now().Format("15:04"),
			},
		}).
		ToTarget(notifyhub.NewTarget("phone", "+1987654321", "sms")).
		Build()

	// Send platform-specific messages
	messages := map[string]*notifyhub.Message{
		"Feishu": feishuMsg,
		"Email":  emailMsg,
		"SMS":    smsMsg,
	}

	for platform, msg := range messages {
		if receipt, err := hub.Send(ctx, msg); err != nil {
			fmt.Printf("❌ %s message failed: %v\n", platform, err)
		} else {
			fmt.Printf("✅ %s message sent successfully (Duration: %dms)\n", platform, receipt.Results[0].Duration.Milliseconds())
		}
	}
	fmt.Println()

	// Step 4: Conditional messaging based on priority
	fmt.Println("Step 4: Priority-based routing...")

	// Normal priority - only internal team (Feishu)
	normalMsg := notifyhub.NewMessage("Normal Update").
		WithBody("Regular status update for the team.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	// High priority - team + email
	alertMsg := notifyhub.NewAlert("Important Alert").
		WithBody("Something important happened that needs attention.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		ToTarget(notifyhub.NewTarget("email", "oncall@company.com", "email")).
		Build()

	// Urgent - all channels
	urgentMsg := notifyhub.NewUrgent("CRITICAL: System Down").
		WithBody("Critical system failure - immediate action required!").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		ToTarget(notifyhub.NewTarget("email", "emergency@company.com", "email")).
		ToTarget(notifyhub.NewTarget("phone", "+1Emergency123", "sms")).
		Build()

	priorityMessages := map[string]*notifyhub.Message{
		"Normal": normalMsg,
		"Alert":  alertMsg,
		"Urgent": urgentMsg,
	}

	for priority, msg := range priorityMessages {
		receipt, err := hub.Send(ctx, msg)
		if err != nil {
			fmt.Printf("❌ %s priority message failed: %v\n", priority, err)
		} else {
			fmt.Printf("✅ %s priority sent to %d platforms (Priority: %d)\n",
				priority, len(receipt.Results), msg.Priority)
		}
	}
	fmt.Println()

	// Step 5: Show the power of unified architecture
	fmt.Println("🏗️  Unified Architecture Benefits")
	fmt.Println("-------------------------------")
	fmt.Println("✅ CONSISTENT CONFIGURATION:")
	fmt.Println("  • All platforms use the same patterns")
	fmt.Println("  • feishu.WithFeishu(), email.WithEmail(), sms.WithSMS()")
	fmt.Println("  • Same optional parameter system")
	fmt.Println()
	fmt.Println("✅ SEAMLESS INTEGRATION:")
	fmt.Println("  • Single hub handles all platforms")
	fmt.Println("  • Unified error handling and responses")
	fmt.Println("  • Cross-platform message broadcasting")
	fmt.Println()
	fmt.Println("✅ PLATFORM INDEPENDENCE:")
	fmt.Println("  • Add/remove platforms without code changes")
	fmt.Println("  • Platform-specific features preserved")
	fmt.Println("  • External platforms work exactly the same")
	fmt.Println()

	fmt.Println("🌐 Multi-Platform Demo Complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("• Try examples/platforms/ for advanced platform features")
	fmt.Println("• See examples/external/ for creating custom platforms")
	fmt.Println("• Check examples/advanced/ for production patterns")
}
