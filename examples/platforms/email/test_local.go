//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/email"
)

// This example uses MailHog for local testing
// Install MailHog: brew install mailhog
// Run MailHog: mailhog
// View emails: http://localhost:8025

func main() {
	fmt.Println("📧 Email Platform - Local Testing with MailHog")
	fmt.Println("===============================================")
	fmt.Println()
	fmt.Println("Prerequisites:")
	fmt.Println("  1. Install MailHog: brew install mailhog")
	fmt.Println("  2. Run MailHog: mailhog")
	fmt.Println("  3. Open UI: http://localhost:8025")
	fmt.Println()

	ctx := context.Background()

	// Create hub with MailHog configuration (no TLS, no auth)
	hub, err := notifyhub.NewHub(
		email.WithEmail("localhost", 1025, "test@example.com",
			email.WithEmailTLS(false), // MailHog doesn't use TLS
		),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Close(ctx) }()

	fmt.Println("✅ Hub created with MailHog configuration")
	fmt.Println()

	// Test 1: Simple text email
	fmt.Println("📝 Test 1: Sending simple text email...")
	simpleMsg := notifyhub.NewMessage("Test Email - Simple Text").
		WithBody("This is a test email sent to MailHog.\n\nNo actual email will be sent.").
		ToTarget(notifyhub.NewTarget("email", "recipient@example.com", "email")).
		Build()

	receipt, err := hub.Send(ctx, simpleMsg)
	if err != nil {
		log.Printf("❌ Failed: %v", err)
	} else if receipt.Results[0].Success {
		fmt.Printf("✅ Email sent successfully (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
		fmt.Printf("   Check MailHog UI: http://localhost:8025\n")
	} else {
		fmt.Printf("❌ Send failed: %s\n", receipt.Results[0].Error)
	}
	fmt.Println()

	// Test 2: HTML email
	fmt.Println("🎨 Test 2: Sending HTML email...")
	htmlMsg := notifyhub.NewMessage("Test Email - HTML Content").
		WithBody(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .button { background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to NotifyHub!</h1>
        </div>
        <div class="content">
            <p>This is a test HTML email sent via MailHog.</p>
            <p>Features tested:</p>
            <ul>
                <li>✅ HTML rendering</li>
                <li>✅ CSS styling</li>
                <li>✅ Email delivery</li>
            </ul>
            <p>
                <a href="http://localhost:8025" class="button">View in MailHog</a>
            </p>
        </div>
    </div>
</body>
</html>
		`).
		WithFormat("html").
		ToTarget(notifyhub.NewTarget("email", "test@example.com", "email")).
		Build()

	receipt, err = hub.Send(ctx, htmlMsg)
	if err != nil {
		log.Printf("❌ Failed: %v", err)
	} else if receipt.Results[0].Success {
		fmt.Printf("✅ HTML email sent successfully (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
		fmt.Printf("   Check MailHog UI: http://localhost:8025\n")
	} else {
		fmt.Printf("❌ Send failed: %s\n", receipt.Results[0].Error)
	}
	fmt.Println()

	// Test 3: Multiple recipients
	fmt.Println("👥 Test 3: Sending to multiple recipients...")
	multiMsg := notifyhub.NewMessage("Team Notification").
		WithBody("This notification was sent to multiple recipients.").
		ToTarget(notifyhub.NewTarget("email", "alice@example.com", "email")).
		ToTarget(notifyhub.NewTarget("email", "bob@example.com", "email")).
		ToTarget(notifyhub.NewTarget("email", "charlie@example.com", "email")).
		Build()

	receipt, err = hub.Send(ctx, multiMsg)
	if err != nil {
		log.Printf("❌ Failed: %v", err)
	} else {
		successful := 0
		for _, r := range receipt.Results {
			if r.Success {
				successful++
			}
		}
		fmt.Printf("✅ Sent to %d/%d recipients\n", successful, len(receipt.Results))
		fmt.Printf("   Check MailHog UI: http://localhost:8025\n")
	}
	fmt.Println()

	fmt.Println("📧 Local Email Testing Complete!")
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Println("  • All emails are captured by MailHog")
	fmt.Println("  • No actual emails are sent")
	fmt.Println("  • Perfect for development and testing")
	fmt.Println("  • View all emails at: http://localhost:8025")
}
