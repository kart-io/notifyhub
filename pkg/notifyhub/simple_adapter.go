// Package notifyhub provides a simple working adapter showing the modular structure
package notifyhub

import (
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Demonstrate the modular structure working

// ExampleModularUsage shows how the new modular structure works
func ExampleModularUsage() {
	// 1. Create a message using the message module
	msg := message.NewMessage("Test Modular Structure").
		WithBody("This demonstrates the new modular architecture").
		WithPriority(message.PriorityNormal).
		Build()

	// 2. Create targets using the target module
	emailTarget := target.NewEmailTarget("test@example.com")
	phoneTarget := target.NewPhoneTarget("+1234567890")

	// Add targets to message
	msg.AddTarget(emailTarget)
	msg.AddTarget(phoneTarget)

	// 3. Auto-detect targets
	autoTarget := target.AutoDetectTarget("user@example.com")
	msg.AddTarget(autoTarget)

	// 4. Create a receipt (simulated)
	result := &receipt.Receipt{
		MessageID:  msg.ID,
		Status:     "success",
		Successful: len(msg.Targets),
		Failed:     0,
		Timestamp:  time.Now(),
		Results: []receipt.PlatformResult{
			{
				Platform:  "email",
				Target:    emailTarget.Value,
				Success:   true,
				Timestamp: time.Now(),
			},
		},
	}

	// Log the results
	println("📦 Modular Structure Demo")
	println("Message ID:", msg.ID)
	println("Message Title:", msg.Title)
	println("Target Count:", len(msg.Targets))
	println("Receipt Status:", result.Status)
}

// ShowModularStructure displays the new module organization
func ShowModularStructure() {
	println("\n🏗️ NotifyHub Modular Structure:")
	println("├── core/")
	println("│   ├── hub.go              # Hub interface and creation")
	println("│   ├── lifecycle.go        # Startup/shutdown management")
	println("│   └── health.go           # Health monitoring")
	println("├── message/")
	println("│   ├── message.go          # Message structure")
	println("│   ├── builder.go          # Fluent message builder")
	println("│   └── priority.go         # Priority definitions")
	println("├── target/")
	println("│   ├── target.go           # Target abstraction")
	println("│   ├── factory.go          # Target creation")
	println("│   └── resolver.go         # Auto-detection logic")
	println("├── config/")
	println("│   └── config.go           # Configuration structures")
	println("├── errors/")
	println("│   └── types.go            # Error definitions")
	println("├── receipt/")
	println("│   └── receipt.go          # Result tracking")
	println("└── adapter.go              # Backwards compatibility")
	println("\n✅ Benefits:")
	println("• Clear separation of concerns")
	println("• Easier testing and maintenance")
	println("• Better code organization")
	println("• Modular development possible")
}
