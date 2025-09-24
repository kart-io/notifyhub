// Package main demonstrates Email platform-specific features
// This shows the complete Email SMTP integration capabilities in the unified architecture
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/email"
)

const (
	// Gmail SMTP configuration
	// To use Gmail SMTP, you need:
	// 1. Enable 2-factor authentication in your Google account
	// 2. Generate an app-specific password: https://myaccount.google.com/apppasswords
	gmailEmail    = "" // Your Gmail address (e.g., "user@gmail.com")
	gmailPassword = "" // Your Gmail app password (not your regular password)

	// 163 Mail SMTP configuration
	// To use 163 Mail SMTP, you need:
	// 1. Enable SMTP service in 163 Mail settings
	// 2. Use the authorization code (not your login password)
	gmailEmail163    = "" // Your 163 email address (e.g., "user@163.com")
	gmailPassword163 = "" // Your 163 SMTP authorization code

	// Recipient email address
	sendEmail = "" // Email address to send test emails to
)

// demo1BasicSMTPConfig demonstrates basic SMTP configuration
func demo1BasicSMTPConfig() {
	log := logger.New().LogMode(logger.Info)

	log.Info("🔧 Demo 1: Basic SMTP Configuration")
	log.Info("=====================================")

	// 💡 Make sure MailHog is running: brew install mailhog && mailhog
	client, err := notifyhub.New(
		email.WithEmail("smtp.gmail.com", 465, gmailEmail,
			email.WithEmailAuth(gmailEmail, gmailPassword),
			email.WithEmailSSL(true),
		),
	)
	if err != nil {
		log.Error("❌ Failed to create basic SMTP client", "error", err)
		return
	}
	defer func() { _ = client.Close() }()

	log.Info("✅ Basic SMTP client created successfully")
	log.Info("   - Host: smtp.gmail.com")
	log.Info("   - Port: 465")
	log.Info("   - From: ", "email", gmailEmail)
	log.Info("   - Auth: Yes (basic)")
	log.Info("   - TLS: Yes")
}

// demo2AuthenticatedSMTP demonstrates authenticated SMTP with TLS
func demo2AuthenticatedSMTP() {
	log := logger.New().LogMode(logger.Info)

	log.Info("🔐 Demo 2: Authenticated SMTP with TLS")
	log.Info("=========================================")

	// 💡 Make sure MailHog is running: brew install mailhog && mailhog
	// Note: MailHog doesn't require real auth, but we can still test the configuration path.
	client, err := notifyhub.New(
		email.WithEmail("smtp.gmail.com", 465, gmailEmail,
			email.WithEmailAuth(gmailEmail, gmailPassword),
			email.WithEmailSSL(true),
			email.WithEmailTimeout(10*time.Second),
		),
	)
	if err != nil {
		log.Error("❌ Failed to create authenticated SMTP client", "error", err)
		return
	}
	defer func() { _ = client.Close() }()

	log.Info("✅ Authenticated SMTP client created successfully")
	log.Info("   - Host: smtp.gmail.com")
	log.Info("   - Port: 465")
	log.Info("   - TLS: Yes")
	log.Info("   - Auth: Yes (basic)")
	log.Info("   - SSL: Yes")
	log.Info("   - Timeout: 10s")
}

// demo3SSLConfiguration demonstrates SSL configuration
func demo3SSLConfiguration() {
	fmt.Println("🔒 Demo 3: SSL SMTP Configuration")
	fmt.Println("===================================")

	// 💡 Make sure MailHog is running: brew install mailhog && mailhog
	// Note: This demo is for configuration illustration; MailHog doesn't use SSL on port 1025.
	client, err := notifyhub.New(
		email.WithEmail("smtp.gmail.com", 465, gmailEmail,
			email.WithEmailAuth(gmailEmail, gmailPassword),
			email.WithEmailSSL(false), // MailHog doesn't use SSL
			email.WithEmailTLS(false),
		),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create SSL SMTP client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()

	fmt.Println("✅ SSL SMTP client created successfully (for local testing)")
	fmt.Println("   - Host: smtp.gmail.com")
	fmt.Println("   - Port: 465")
	fmt.Println("   - TLS: Yes")
	fmt.Println("   - Auth: Yes (basic)")
	fmt.Println("   - SSL: Yes")
	fmt.Println("   - SSL: Disabled (for MailHog)")
	fmt.Println()
}

// demo4SimpleTextEmail demonstrates sending a simple text email
func demo4SimpleTextEmail() {
	fmt.Println("📝 Demo 4: Simple Text Email")
	fmt.Println("===============================")

	ctx := context.Background()
	email.UseGoMail()

	// Create client for local testing with MailHog
	// 💡 Make sure MailHog is running: brew install mailhog && mailhog
	client, err := notifyhub.New(
		email.WithEmail("smtp.gmail.com", 587, gmailEmail,
			email.WithEmailAuth(gmailEmail, gmailPassword),
			email.WithEmailSSL(false),
			email.WithEmailTLS(true),
		),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()

	// Create message
	msg := notifyhub.NewMessage("System Notification").
		WithBody("This is a simple text email notification.\n\n" +
			"System Status: Online\n" +
			"Last Check: " + time.Now().Format("2006-01-02 15:04:05")).
		ToTarget(notifyhub.NewTarget("email", sendEmail, "email")).
		Build()

	// Send
	fmt.Println("Sending simple text email...")
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		fmt.Printf("❌ Send failed: %v\n", err)
	} else if receipt.Results[0].Success {
		fmt.Printf("✅ Email sent successfully (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	} else {
		fmt.Printf("❌ Send failed: %s\n", receipt.Results[0].Error)
	}
	fmt.Println()
}

// demo5HTMLEmail demonstrates sending HTML email
func demo5HTMLEmail() {
	fmt.Println("🎨 Demo 5: HTML Email")
	fmt.Println("======================")

	ctx := context.Background()

	// Create client for local testing with MailHog
	// 💡 Make sure MailHog is running: brew install mailhog && mailhog
	client, err := notifyhub.New(
		email.WithEmail("smtp.gmail.com", 587, gmailEmail,
			email.WithEmailAuth(gmailEmail, gmailPassword),
			email.WithEmailSSL(false),
			email.WithEmailTLS(true),
		),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()

	// Create HTML message
	msg := notifyhub.NewMessage("📊 Daily Report").
		WithBody(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; }
        .header { background-color: #4CAF50; color: white; padding: 20px; }
        .content { padding: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>📊 Daily System Report</h1>
    </div>
    <div class="content">
        <p>Server Uptime: 99.9%</p>
        <p>Active Users: 1,247</p>
    </div>
</body>
</html>
		`).
		WithFormat("html").
		ToTarget(notifyhub.NewTarget("email", sendEmail, "email")).
		Build()

	// Send
	fmt.Println("Sending HTML email...")
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		fmt.Printf("❌ Send failed: %v\n", err)
	} else if receipt.Results[0].Success {
		fmt.Printf("✅ HTML email sent successfully (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	} else {
		fmt.Printf("❌ Send failed: %s\n", receipt.Results[0].Error)
	}
	fmt.Println()
}

// demo6EmailWithPriority demonstrates email with priority
func demo6EmailWithPriority() {
	fmt.Println("⚡ Demo 6: Email with Priority")
	fmt.Println("===============================")

	ctx := context.Background()

	// Create client for local testing with MailHog
	// 💡 Make sure MailHog is running: brew install mailhog && mailhog
	client, err := notifyhub.New(
		email.WithEmail("smtp.gmail.com", 465, gmailEmail,
			email.WithEmailAuth(gmailEmail, gmailPassword),
			email.WithEmailSSL(true),
			email.WithEmailTLS(false),
		),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()

	// Create alert message with high priority
	msg := notifyhub.NewAlert("🚨 Critical System Alert").
		WithBody("Database connection pool is critically low. Immediate action required!").
		WithPlatformData(map[string]interface{}{
			"email_priority": "high",
		}).
		ToTarget(notifyhub.NewTarget("email", sendEmail, "email")).
		Build()

	// Send
	fmt.Println("Sending high priority alert...")
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		fmt.Printf("❌ Send failed: %v\n", err)
	} else if receipt.Results[0].Success {
		fmt.Printf("✅ Priority email sent successfully (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	} else {
		fmt.Printf("❌ Send failed: %s\n", receipt.Results[0].Error)
	}
	fmt.Println()
}

// demo7EmailWithCC demonstrates email with CC recipients
func demo7EmailWithCC() {
	fmt.Println("👥 Demo 7: Email with CC Recipients")
	fmt.Println("=====================================")

	ctx := context.Background()

	// Create client for local testing with MailHog
	// 💡 Make sure MailHog is running: brew install mailhog && mailhog
	client, err := notifyhub.New(
		email.WithEmail("smtp.gmail.com", 465, gmailEmail,
			email.WithEmailAuth(gmailEmail, gmailPassword),
			email.WithEmailSSL(true),
			email.WithEmailTLS(false),
		),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()

	// Create message with CC
	msg := notifyhub.NewMessage("Monthly Security Review").
		WithBody("Please review the monthly security report and provide feedback by EOD.").
		WithPlatformData(map[string]interface{}{
			"email_cc": []string{"security@company.com", "manager@company.com"},
		}).
		ToTarget(notifyhub.NewTarget("email", sendEmail, "email")).
		Build()

	// Send
	fmt.Println("Sending email with CC recipients...")
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		fmt.Printf("❌ Send failed: %v\n", err)
	} else if receipt.Results[0].Success {
		fmt.Printf("✅ Email with CC sent successfully (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	} else {
		fmt.Printf("❌ Send failed: %s\n", receipt.Results[0].Error)
	}
	fmt.Println()
}

// demo8TemplateEmail demonstrates email with template variables
func demo8TemplateEmail() {
	fmt.Println("📋 Demo 8: Template Email with Variables")
	fmt.Println("===========================================")

	ctx := context.Background()

	// Create client for local testing with MailHog
	// 💡 Make sure MailHog is running: brew install mailhog && mailhog
	client, err := notifyhub.New(
		email.WithEmail("smtp.gmail.com", 465, gmailEmail,
			email.WithEmailAuth(gmailEmail, gmailPassword),
			email.WithEmailSSL(true),
			email.WithEmailTLS(false),
		),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()

	// Create template message
	msg := notifyhub.NewMessage("Welcome {{user_name}}!").
		WithBody("Hello {{user_name}},\n\n"+
			"Welcome to our platform!\n"+
			"Username: {{username}}\n"+
			"Email: {{email}}\n"+
			"Registration Date: {{reg_date}}\n\n"+
			"Best regards,\n"+
			"The {{company}} Team").
		WithVariable("user_name", "Alice Smith").
		WithVariable("username", "alice.smith").
		WithVariable("email", "alice@example.com").
		WithVariable("reg_date", time.Now().Format("2006-01-02")).
		WithVariable("company", "NotifyHub").
		ToTarget(notifyhub.NewTarget("email", sendEmail, "email")).
		Build()

	// Send
	fmt.Println("Sending template email...")
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		fmt.Printf("❌ Send failed: %v\n", err)
	} else if receipt.Results[0].Success {
		fmt.Printf("✅ Template email sent successfully (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	} else {
		fmt.Printf("❌ Send failed: %s\n", receipt.Results[0].Error)
	}
	fmt.Println()
}

// demo9MultipleRecipients demonstrates sending to multiple recipients
func demo9MultipleRecipients() {
	fmt.Println("📬 Demo 9: Multiple Recipients")
	fmt.Println("=================================")

	ctx := context.Background()

	// Create client for local testing with MailHog
	// 💡 Make sure MailHog is running: brew install mailhog && mailhog
	client, err := notifyhub.New(
		email.WithEmail("smtp.gmail.com", 465, gmailEmail,
			email.WithEmailAuth(gmailEmail, gmailPassword),
			email.WithEmailSSL(true),
			email.WithEmailTLS(false),
		),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()

	// Create message for multiple recipients
	msg := notifyhub.NewMessage("Team Meeting Reminder").
		WithBody("Don't forget about our weekly team meeting tomorrow at 10:00 AM.").
		ToTarget(notifyhub.NewTarget("email", "john@example.com", "email")).
		ToTarget(notifyhub.NewTarget("email", "jane@example.com", "email")).
		ToTarget(notifyhub.NewTarget("email", "bob@example.com", "email")).
		Build()

	// Send
	fmt.Println("Sending to multiple recipients...")
	receipt, err := client.Send(ctx, msg)
	if err != nil {
		fmt.Printf("❌ Send failed: %v\n", err)
	} else {
		successful := 0
		for _, r := range receipt.Results {
			if r.Success {
				successful++
			}
		}
		fmt.Printf("✅ Sent to %d/%d recipients successfully\n", successful, len(receipt.Results))
		for _, r := range receipt.Results {
			status := "✅"
			if !r.Success {
				status = "❌"
			}
			fmt.Printf("   %s %s\n", status, r.Target)
		}
	}
	fmt.Println()
}

// demo10DifferentMessageTypes demonstrates different message types
func demo10DifferentMessageTypes() {
	fmt.Println("📊 Demo 10: Different Message Types")
	fmt.Println("=====================================")

	ctx := context.Background()

	// Create client for local testing with MailHog
	// 💡 Make sure MailHog is running: brew install mailhog && mailhog
	client, err := notifyhub.New(
		email.WithEmail("localhost", 1025, "sender@example.com"),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()

	// Test different message types
	types := map[string]*notifyhub.Message{
		"Regular": notifyhub.NewMessage("📰 Weekly Newsletter").
			WithBody("This week's highlights...").
			ToTarget(notifyhub.NewTarget("email", "subscriber@example.com", "email")).
			Build(),

		"Alert": notifyhub.NewAlert("⚠️ System Alert").
			WithBody("Server memory at 90%").
			ToTarget(notifyhub.NewTarget("email", "ops@example.com", "email")).
			Build(),

		"Urgent": notifyhub.NewUrgent("🚨 CRITICAL").
			WithBody("Service down!").
			ToTarget(notifyhub.NewTarget("email", "oncall@example.com", "email")).
			Build(),
	}

	for typeName, msg := range types {
		fmt.Printf("Sending %s message (Priority: %d)...\n", typeName, msg.Priority)
		receipt, err := client.Send(ctx, msg)
		if err != nil {
			fmt.Printf("  ❌ Failed: %v\n", err)
		} else if receipt.Results[0].Success {
			fmt.Printf("  ✅ Sent successfully\n")
		} else {
			fmt.Printf("  ❌ Failed: %s\n", receipt.Results[0].Error)
		}
	}
	fmt.Println()
}

func demo11WithGmailSMTP() {
	fmt.Println("🔧 Demo 11: With Gmail SMTP")
	fmt.Println("=============================")

	ctx := context.Background()

	client, err := notifyhub.New(
		email.WithGmailSMTP(gmailEmail, gmailPassword),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()

	fmt.Println("✅ With Gmail SMTP client created successfully")
	fmt.Println()
	// Create message
	msg := notifyhub.NewMessage("System Notification").
		WithBody("This is a simple text email notification.\n\n" +
			"System Status: Online\n" +
			"Last Check: " + time.Now().Format("2006-01-02 15:04:05")).
		ToTarget(notifyhub.NewTarget("email", sendEmail, "email")).
		Build()

	// Send
	fmt.Println("Sending simple text email...")
	receipt, err := client.Send(ctx, msg)
	fmt.Println(receipt)
	if err != nil {
		fmt.Printf("❌ Send failed: %v\n", err)
	} else if receipt.Results[0].Success {
		fmt.Printf("✅ Template email sent successfully (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	} else {
		fmt.Printf("❌ Send failed: %s\n", receipt.Results[0].Error)
	}
	fmt.Println()
}

func demo12With163SMTP() {
	fmt.Println("🔧 Demo 12: With 163 SMTP")
	fmt.Println("=============================")

	ctx := context.Background()

	client, err := notifyhub.New(
		email.With163SMTP(gmailEmail163, gmailPassword163),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()

	fmt.Println("✅ With 163 SMTP client created successfully")
	fmt.Println()

	// Create message
	msg := notifyhub.NewMessage("System Notification").
		WithBody("This is a simple text email notification.\n\n" +
			"System Status: Online\n" +
			"Last Check: " + time.Now().Format("2006-01-02 15:04:05")).
		// ToTarget(notifyhub.NewTarget("email", sendEmail, "email")).
		ToEmail(sendEmail).
		Build()

	// Send
	fmt.Println("Sending simple text email...")
	receipt, err := client.Send(ctx, msg)
	fmt.Println(receipt)
	if err != nil {
		fmt.Printf("❌ Send failed: %v\n", err)
	} else if receipt.Results[0].Success {
		fmt.Printf("✅ Template email sent successfully (Duration: %dms)\n", receipt.Results[0].Duration.Milliseconds())
	} else {
		fmt.Printf("❌ Send failed: %s\n", receipt.Results[0].Error)
	}
	fmt.Println()
}

func main() {
	demo1BasicSMTPConfig()
	demo2AuthenticatedSMTP()
	demo3SSLConfiguration()
	demo4SimpleTextEmail()
	demo5HTMLEmail()
	demo6EmailWithPriority()
	demo7EmailWithCC()
	demo8TemplateEmail()
	demo9MultipleRecipients()
	demo10DifferentMessageTypes()
	demo11WithGmailSMTP()
	demo12With163SMTP()
}
