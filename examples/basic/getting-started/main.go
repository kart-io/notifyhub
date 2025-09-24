// Package main demonstrates getting started with NotifyHub
// This is your first NotifyHub application showing basic concepts
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/feishu"
)

func main() {
	log := logger.New().LogMode(logger.Info)

	log.Info("🚀 Getting Started with NotifyHub")
	log.Info("================================")

	// Get Feishu webhook URL and secret from environment variables
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	secret := os.Getenv("FEISHU_SECRET")

	if webhookURL == "" {
		log.Error("FEISHU_WEBHOOK_URL environment variable not set")
		os.Exit(1)
	}

	// Create a new NotifyHub client
	hub, err := notifyhub.New(
		feishu.WithFeishu(webhookURL, feishu.WithFeishuSecret(secret)),
	)
	if err != nil {
		log.Error("Failed to create NotifyHub client", "error", err)
		os.Exit(1)
	}
	defer func() { _ = hub.Close() }()

	log.Info("✅ Hub created successfully!")

	// Step 2: Create a simple message
	log.Info("Step 2: Creating a message...")

	message := notifyhub.NewMessage("Hello NotifyHub!").
		WithBody("This is my first notification using the unified platform system.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	log.Info("✅ Message created", "title", message.Title)

	// Step 3: Send the message
	log.Info("Step 3: Sending message...")

	ctx := context.Background()
	receipt, err := hub.Send(ctx, message)
	if err != nil {
		log.Error("❌ Send failed", "error", err)
	} else {
		log.Info("✅ Message sent successfully!")
		log.Info("📊 Results", "total", receipt.Total, "successful", receipt.Successful, "failed", receipt.Failed)

		// Show detailed results
		for _, result := range receipt.Results {
			status := "✅"
			if !result.Success {
				status = "❌"
			}
			log.Info(fmt.Sprintf("%s Platform result", status), "platform", result.Platform, "target", result.Target)
		}
	}

	// Step 4: Try different message types
	log.Info("Step 4: Trying different message types...")

	// Alert message (high priority)
	alertMsg := notifyhub.NewAlert("System Alert").
		WithBody("This is a high-priority alert message.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	// Urgent message (highest priority)
	urgentMsg := notifyhub.NewUrgent("URGENT: Action Required").
		WithBody("This is the highest priority message type.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	// Send alert
	if _, err := hub.Send(ctx, alertMsg); err != nil {
		log.Error("❌ Alert send failed", "error", err)
	} else {
		log.Info("✅ Alert sent", "priority", alertMsg.Priority)
	}

	// Send urgent
	if _, err := hub.Send(ctx, urgentMsg); err != nil {
		log.Error("❌ Urgent send failed", "error", err)
	} else {
		log.Info("✅ Urgent message sent", "priority", urgentMsg.Priority)
	}

	// Step 5: Understanding the new architecture
	log.Info("🏗️  Understanding the New Architecture")
	log.Info("-----------------------------------")
	log.Info("✅ UNIFIED PLATFORM PACKAGES:")
	log.Info("  • Each platform lives in its own package")
	log.Info("  • Auto-registration when package is imported")
	log.Info("  • Consistent API across all platforms")
	log.Info("✅ EXTERNAL EXTENSIBILITY:")
	log.Info("  • External developers can create platform packages")
	log.Info("  • Same API quality as built-in platforms")
	log.Info("  • No core library modifications needed")
	log.Info("✅ BACKWARD COMPATIBILITY:")
	log.Info("  • Old notifyhub.WithFeishu() still works")
	log.Info("  • Gradual migration path available")
	log.Info("  • Deprecated functions clearly marked")

	log.Info("🎉 Getting Started Complete!")
	log.Info("Next steps:")
	log.Info("• Try examples/basic/multi-platform/ for multiple platforms")
	log.Info("• See examples/platforms/ for platform-specific features")
	log.Info("• Check examples/external/ for building custom platforms")
}
