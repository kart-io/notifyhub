//go:build ignore
// +build ignore

// Test script to verify go-mail implementation without network dependency
package main

import (
	"fmt"
	"reflect"

	"github.com/kart-io/notifyhub/pkg/platforms/email"
)

func main() {
	fmt.Println("🧪 Testing Email Implementation Selection")
	fmt.Println("==========================================\n")

	// Test 1: Default should be go-mail
	fmt.Println("Test 1: Default Implementation")
	config1 := map[string]interface{}{
		"smtp_host": "smtp.example.com",
		"smtp_port": 587,
		"smtp_from": "test@example.com",
	}

	sender1, err := email.NewEmailSenderGoMail(config1)
	if err != nil {
		fmt.Printf("❌ Failed to create go-mail sender: %v\n", err)
	} else {
		fmt.Printf("✅ Created go-mail sender: %v\n", reflect.TypeOf(sender1))
		fmt.Printf("   Platform: %s\n", sender1.Name())
	}

	// Test 2: Legacy net/smtp implementation
	fmt.Println("\nTest 2: Legacy net/smtp Implementation")
	sender2, err := email.NewEmailSender(config1)
	if err != nil {
		fmt.Printf("❌ Failed to create net/smtp sender: %v\n", err)
	} else {
		fmt.Printf("✅ Created net/smtp sender: %v\n", reflect.TypeOf(sender2))
		fmt.Printf("   Platform: %s\n", sender2.Name())
	}

	// Test 3: Capabilities comparison
	fmt.Println("\nTest 3: Capabilities Comparison")
	if sender1 != nil {
		caps1 := sender1.GetCapabilities()
		fmt.Printf("go-mail capabilities:\n")
		fmt.Printf("  - Name: %s\n", caps1.Name)
		fmt.Printf("  - Formats: %v\n", caps1.SupportedFormats)
		fmt.Printf("  - Attachments: %v\n", caps1.SupportsAttachments)
	}

	if sender2 != nil {
		caps2 := sender2.GetCapabilities()
		fmt.Printf("net/smtp capabilities:\n")
		fmt.Printf("  - Name: %s\n", caps2.Name)
		fmt.Printf("  - Formats: %v\n", caps2.SupportedFormats)
		fmt.Printf("  - Attachments: %v\n", caps2.SupportsAttachments)
	}

	fmt.Println("\n✅ Implementation test completed")
}
