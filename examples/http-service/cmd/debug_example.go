package main

import (
	"context"
	"log"
	"time"

	"github.com/kart-io/notifyhub/client"
	"github.com/kart-io/notifyhub/notifiers"
)

// Debug Example - Demonstrates the new debug features
//
// This example shows how to use the debug mode for development
// and troubleshooting NotifyHub applications.
//
// Usage:
//   go run cmd/debug_example.go
//
func main() {
	log.Println("🐛 NotifyHub Debug Mode Example")

	ctx := context.Background()

	// ==========================================
	// 1. Create Hub with Debug Mode
	// ==========================================

	log.Println("\n🔧 Creating Hub with debug mode enabled...")

	// Method 1: Create hub with debug mode from start
	hub, err := client.NewWithDebugAndStart(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to create debug hub: %v", err)
	}
	defer hub.Stop()

	log.Println("✅ Debug hub created and started")

	// ==========================================
	// 2. Debug Message Building
	// ==========================================

	log.Println("\n📝 Building message with debug output...")

	// Create a message with debug output
	message := client.NewMessage().
		Title("Debug Test Message").
		Body("This message demonstrates debug capabilities").
		Email("debug@example.com").
		User("dev-team", "slack").
		High().
		Metadata("environment", "development").
		Debug() // This will output detailed information about the message

	log.Println("\n🔍 Message inspection:")
	log.Printf("   %s", message.Inspect())

	// ==========================================
	// 3. Debug Hub Methods
	// ==========================================

	log.Println("\n🔧 Hub debug methods:")
	log.Printf("   Debug enabled: %v", hub.IsDebugEnabled())

	// Create a debug message builder from hub
	debugMessage := hub.DebugMessage().
		Title("Hub Debug Message").
		Body("Created using hub.DebugMessage()").
		Email("ops@example.com")

	// ==========================================
	// 4. Send Messages with Debug Logging
	// ==========================================

	log.Println("\n📤 Sending messages with debug logging...")

	// Send the first message (will show debug output)
	results1, err := hub.Send(ctx, message.Build(), nil)
	if err != nil {
		log.Printf("❌ Failed to send message: %v", err)
	} else {
		log.Printf("✅ Message sent successfully: %d results", len(results1))
	}

	// Wait a moment for processing
	time.Sleep(100 * time.Millisecond)

	// Send the second message
	results2, err := hub.Send(ctx, debugMessage.Build(), nil)
	if err != nil {
		log.Printf("❌ Failed to send debug message: %v", err)
	} else {
		log.Printf("✅ Debug message sent successfully: %d results", len(results2))
	}

	// ==========================================
	// 5. Runtime Debug Control
	// ==========================================

	log.Println("\n🎛️ Runtime debug control:")

	// Disable debug mode temporarily
	hub.DisableDebug()
	log.Printf("   Debug enabled: %v", hub.IsDebugEnabled())

	// Send a message without debug output
	silentMessage := client.NewMessage().
		Title("Silent Message").
		Body("This won't show debug output").
		Email("silent@example.com").
		Build()

	results3, err := hub.Send(ctx, silentMessage, nil)
	if err != nil {
		log.Printf("❌ Failed to send silent message: %v", err)
	} else {
		log.Printf("✅ Silent message sent (no debug output): %d results", len(results3))
	}

	// Re-enable debug mode
	hub.EnableDebug()
	log.Printf("   Debug enabled: %v", hub.IsDebugEnabled())

	// ==========================================
	// 6. Advanced Debug Features
	// ==========================================

	log.Println("\n🚀 Advanced debug features:")

	// Build a complex message with conditional logic
	complexMessage := client.NewAlert("System Alert", "Service degradation detected").
		Email("sre@company.com").
		If(true, func(b *client.MessageBuilder) *client.MessageBuilder {
			return b.Urgent().Email("oncall@company.com").Metadata("severity", "high")
		}).
		Unless(false, func(b *client.MessageBuilder) *client.MessageBuilder {
			return b.Metadata("auto-escalate", "true")
		}).
		Debug() // Show the final message state

	// Send the complex message
	results4, err := hub.Send(ctx, complexMessage.Build(), nil)
	if err != nil {
		log.Printf("❌ Failed to send complex message: %v", err)
	} else {
		log.Printf("✅ Complex message sent successfully: %d results", len(results4))
	}

	// ==========================================
	// 7. Batch Send with Debug
	// ==========================================

	log.Println("\n📦 Batch send with debug logging...")

	// Create multiple messages
	messages := []*client.MessageBuilder{
		client.NewMessage().Title("Batch 1").Body("First batch message").Email("batch1@example.com"),
		client.NewMessage().Title("Batch 2").Body("Second batch message").Email("batch2@example.com"),
		client.NewMessage().Title("Batch 3").Body("Third batch message").Email("batch3@example.com"),
	}

	// Build all messages
	var builtMessages []*notifiers.Message
	for _, builder := range messages {
		builtMessages = append(builtMessages, builder.Build())
	}

	// Send batch with debug logging
	batchResults, err := hub.SendBatch(ctx, builtMessages, nil)
	if err != nil {
		log.Printf("❌ Failed to send batch: %v", err)
	} else {
		log.Printf("✅ Batch sent successfully: %d results", len(batchResults))
	}

	log.Println("\n🎉 Debug example complete!")
	log.Println("\n💡 Key Debug Features Demonstrated:")
	log.Println("   • client.NewWithDebug() - Create hub with debug mode")
	log.Println("   • message.Debug() - Enable debug output for message building")
	log.Println("   • hub.IsDebugEnabled() - Check debug status")
	log.Println("   • hub.EnableDebug()/DisableDebug() - Runtime debug control")
	log.Println("   • hub.DebugMessage() - Create debug-enabled message builder")
	log.Println("   • message.Inspect() - Get human-readable message summary")
	log.Println("   • Detailed logging of all send operations")
}