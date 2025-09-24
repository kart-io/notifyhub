// Package main demonstrates different message types and builder patterns
// This shows the flexible message creation system in NotifyHub
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/feishu"
)

func main() {
	log := logger.New().LogMode(logger.Info)

	log.Info("📝 Message Types and Builder Patterns")
	log.Info("===================================")

	// Create hub for testing
	hub, err := notifyhub.New(
		feishu.WithFeishu(os.Getenv("FEISHU_WEBHOOK_URL")),
	)
	if err != nil {
		log.Error("Failed to create hub", "error", err)
		os.Exit(1)
	}
	defer func() { _ = hub.Close() }()

	ctx := context.Background()

	// Part 1: Basic Message Types
	log.Info("📋 Part 1: Basic Message Types")
	log.Info("-----------------------------")

	// 1. Normal Message
	log.Info("1. Normal Message (Default Priority)")
	normalMsg := notifyhub.NewMessage("System Update").
		WithBody("Regular system maintenance completed successfully.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	log.Info("Message details", "priority", normalMsg.Priority, "title", normalMsg.Title)

	// 2. Alert Message (High Priority)
	log.Info("2. Alert Message (High Priority)")
	alertMsg := notifyhub.NewAlert("Database Warning").
		WithBody("Database connection pool is running low.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	log.Info("Alert message details", "priority", alertMsg.Priority, "title", alertMsg.Title)

	// 3. Urgent Message (Highest Priority)
	fmt.Println("3. Urgent Message (Highest Priority)")
	urgentMsg := notifyhub.NewUrgent("CRITICAL SYSTEM FAILURE").
		WithBody("Payment system is completely down - immediate action required!").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	fmt.Printf("   Priority: %d, Title: %s\n", urgentMsg.Priority, urgentMsg.Title)
	fmt.Println()

	// Part 2: Message Builder Patterns
	fmt.Println("🔧 Part 2: Message Builder Patterns")
	fmt.Println("---------------------------------")

	// 1. Simple message
	fmt.Println("1. Simple Message")
	simple := notifyhub.NewMessage("Hello").
		WithBody("This is a simple message.").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	fmt.Printf("   ✅ Created: %s\n", simple.Title)

	// 2. Message with metadata
	fmt.Println("2. Message with Metadata")
	withMetadata := notifyhub.NewMessage("Deployment Complete").
		WithBody("Version 2.1.0 deployed successfully.").
		WithMetadata("version", "2.1.0").
		WithMetadata("environment", "production").
		WithMetadata("deploy_time", time.Now().Format(time.RFC3339)).
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	fmt.Printf("   ✅ Created with %d metadata fields\n", len(withMetadata.Metadata))

	// 3. Message with variables
	fmt.Println("3. Message with Variables")
	withVariables := notifyhub.NewMessage("User Activity").
		WithBody("User {{user_name}} performed {{action}} at {{timestamp}}.").
		WithVariable("user_name", "Alice").
		WithVariable("action", "login").
		WithVariable("timestamp", time.Now().Format("15:04:05")).
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	fmt.Printf("   ✅ Created with %d variables\n", len(withVariables.Variables))

	// 4. Message with platform-specific data
	fmt.Println("4. Message with Platform-Specific Data")
	platformSpecific := notifyhub.NewMessage("Rich Feishu Message").
		WithBody("This message uses Feishu-specific features.").
		WithPlatformData(map[string]interface{}{
			"feishu_mention_all": true,
			"feishu_mentions": []map[string]interface{}{
				{"user_id": "ou_123456789", "name": "John Doe"},
			},
		}).
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	fmt.Printf("   ✅ Created with platform-specific data\n")

	// 5. Complex message with everything
	fmt.Println("5. Complex Message (All Features)")
	complex := notifyhub.NewAlert("Production Incident").
		WithBody("Incident #{{incident_id}} in {{service}} requires immediate attention.").
		WithFormat("markdown").
		WithVariable("incident_id", "INC-2024-001").
		WithVariable("service", "payment-gateway").
		WithMetadata("severity", "high").
		WithMetadata("assignee", "oncall-team").
		WithMetadata("created_at", time.Now().Unix()).
		WithPlatformData(map[string]interface{}{
			"feishu_mention_all":  true,
			"urgent_notification": true,
		}).
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	fmt.Printf("   ✅ Created complex message with all features\n")
	fmt.Println()

	// Part 3: Different Message Formats
	fmt.Println("📄 Part 3: Message Formats")
	fmt.Println("------------------------")

	// Text format (default)
	textMsg := notifyhub.NewMessage("Text Message").
		WithBody("This is plain text content.").
		WithFormat("text").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	fmt.Printf("1. Text Format: %s\n", textMsg.Format)

	// Markdown format
	markdownMsg := notifyhub.NewMessage("Markdown Message").
		WithBody("**Bold text** and *italic text* with [links](https://example.com).").
		WithFormat("markdown").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	fmt.Printf("2. Markdown Format: %s\n", markdownMsg.Format)

	// HTML format (for email)
	htmlMsg := notifyhub.NewMessage("HTML Message").
		WithBody("<h1>HTML Content</h1><p>Rich <strong>formatting</strong> support.</p>").
		WithFormat("html").
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		Build()
	fmt.Printf("3. HTML Format: %s\n", htmlMsg.Format)
	fmt.Println()

	// Part 4: Multiple Targets
	fmt.Println("🎯 Part 4: Multiple Targets")
	fmt.Println("-------------------------")

	multiTarget := notifyhub.NewMessage("Multi-Target Message").
		WithBody("This message goes to multiple targets.").
		// Multiple targets of the same platform
		ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).
		ToTarget(notifyhub.NewTarget("group", "team-alerts", "feishu")).
		ToTarget(notifyhub.NewTarget("user", "ou_user123", "feishu")).
		Build()

	fmt.Printf("✅ Created message with %d targets\n", len(multiTarget.Targets))
	fmt.Println()

	// Part 5: Send Examples
	fmt.Println("📤 Part 5: Sending Different Message Types")
	fmt.Println("----------------------------------------")

	messages := map[string]*notifyhub.Message{
		"Normal":            normalMsg,
		"Alert":             alertMsg,
		"Urgent":            urgentMsg,
		"With Metadata":     withMetadata,
		"With Variables":    withVariables,
		"Platform-Specific": platformSpecific,
		"Complex":           complex,
		"Multi-Target":      multiTarget,
	}

	for messageType, msg := range messages {
		_, err := hub.Send(ctx, msg)
		if err != nil {
			fmt.Printf("❌ %s failed: %v\n", messageType, err)
		} else {
			fmt.Printf("✅ %s sent successfully (Priority: %d)\n", messageType, msg.Priority)
		}
	}
	fmt.Println()

	// Part 6: Builder Pattern Benefits
	fmt.Println("🏗️  Builder Pattern Benefits")
	fmt.Println("---------------------------")
	fmt.Println("✅ FLUENT API:")
	fmt.Println("  • Chain methods for readable code")
	fmt.Println("  • Optional parameters as needed")
	fmt.Println("  • Build() finalizes the message")
	fmt.Println()
	fmt.Println("✅ TYPE SAFETY:")
	fmt.Println("  • Compile-time checks for required fields")
	fmt.Println("  • Clear method signatures")
	fmt.Println("  • IDE autocomplete support")
	fmt.Println()
	fmt.Println("✅ FLEXIBILITY:")
	fmt.Println("  • Start with simple, add complexity as needed")
	fmt.Println("  • Conditional building based on runtime conditions")
	fmt.Println("  • Reusable message templates")
	fmt.Println()

	// Part 7: Best Practices
	fmt.Println("💡 Best Practices")
	fmt.Println("----------------")
	fmt.Println("1. Use appropriate priority levels:")
	fmt.Println("   • Normal: Regular updates, logs")
	fmt.Println("   • Alert: Issues requiring attention")
	fmt.Println("   • Urgent: Critical problems")
	fmt.Println()
	fmt.Println("2. Include relevant metadata:")
	fmt.Println("   • Environment, version, timestamp")
	fmt.Println("   • Service name, incident ID")
	fmt.Println("   • User information for audit trails")
	fmt.Println()
	fmt.Println("3. Use variables for dynamic content:")
	fmt.Println("   • Template reusability")
	fmt.Println("   • Internationalization support")
	fmt.Println("   • Safe content injection")
	fmt.Println()

	fmt.Println("📝 Message Types Demo Complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("• Try examples/basic/error-handling/ for robust error handling")
	fmt.Println("• See examples/platforms/ for platform-specific features")
	fmt.Println("• Check examples/advanced/ for production patterns")
}
