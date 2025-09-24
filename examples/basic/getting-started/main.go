// Package main demonstrates getting started with NotifyHub
// This is your first NotifyHub application showing basic concepts
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/feishu"
)

func main() {
	fmt.Println("🚀 Getting Started with NotifyHub")
	fmt.Println("================================")
	fmt.Println()

	// Step 1: Create a Hub with Feishu platform
	fmt.Println("Step 1: Creating NotifyHub instance...")

	hub, err := notifyhub.NewHub(
		// Configure Feishu platform using the new unified platform package
		feishu.WithFeishuSignature(
			"https://open.feishu.cn/open-apis/bot/v2/hook/688dc0bf-c74b-41d1-a6b9-8cb660477488", // Replace with your webhook URL
			"gQURr67BPOsTZlI7jBn0Jh", // Required: add webhook secret
		),
		// Configure Feishu platform using the new unified platform package
		feishu.WithFeishuKeywordAuth(
			"https://open.feishu.cn/open-apis/bot/v2/hook/b6bd1f02-01a7-4adc-9cd0-f043414dd5f1", // Replace with your webhook URL
			[]string{"alert", "notification"}, // Required: add webhook secret
		),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Close(context.Background()) }()

	fmt.Println("✅ Hub created successfully!")
	fmt.Println()

	// Step 2: Create a simple message
	fmt.Println("Step 2: Creating a message...")

	message := notifyhub.NewMessage("Hello NotifyHub!").
		WithBody("This is my first notification using the unified platform system.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()

	fmt.Printf("✅ Message created: %s\n", message.Title)
	fmt.Println()

	// Step 3: Send the message
	fmt.Println("Step 3: Sending message...")

	ctx := context.Background()
	receipt, err := hub.Send(ctx, message)
	if err != nil {
		log.Printf("❌ Send failed: %v", err)
	} else {
		fmt.Printf("✅ Message sent successfully!\n")
		fmt.Printf("   📊 Results: %d total, %d successful, %d failed\n",
			receipt.Total, receipt.Successful, receipt.Failed)

		// Show detailed results
		for _, result := range receipt.Results {
			status := "✅"
			if !result.Success {
				status = "❌"
			}
			fmt.Printf("   %s Platform: %s, Target: %s\n",
				status, result.Platform, result.Target)
		}
	}
	fmt.Println()

	// Step 4: Try different message types
	fmt.Println("Step 4: Trying different message types...")

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
		fmt.Printf("❌ Alert send failed: %v\n", err)
	} else {
		fmt.Printf("✅ Alert sent (Priority: %d)\n", alertMsg.Priority)
	}

	// Send urgent
	if _, err := hub.Send(ctx, urgentMsg); err != nil {
		fmt.Printf("❌ Urgent send failed: %v\n", err)
	} else {
		fmt.Printf("✅ Urgent message sent (Priority: %d)\n", urgentMsg.Priority)
	}
	fmt.Println()

	// Step 5: Understanding the new architecture
	fmt.Println("🏗️  Understanding the New Architecture")
	fmt.Println("-----------------------------------")
	fmt.Println("✅ UNIFIED PLATFORM PACKAGES:")
	fmt.Println("  • Each platform lives in its own package")
	fmt.Println("  • Auto-registration when package is imported")
	fmt.Println("  • Consistent API across all platforms")
	fmt.Println()
	fmt.Println("✅ EXTERNAL EXTENSIBILITY:")
	fmt.Println("  • External developers can create platform packages")
	fmt.Println("  • Same API quality as built-in platforms")
	fmt.Println("  • No core library modifications needed")
	fmt.Println()
	fmt.Println("✅ BACKWARD COMPATIBILITY:")
	fmt.Println("  • Old notifyhub.WithFeishu() still works")
	fmt.Println("  • Gradual migration path available")
	fmt.Println("  • Deprecated functions clearly marked")
	fmt.Println()

	fmt.Println("🎉 Getting Started Complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("• Try examples/basic/multi-platform/ for multiple platforms")
	fmt.Println("• See examples/platforms/ for platform-specific features")
	fmt.Println("• Check examples/external/ for building custom platforms")
}
