// Package main demonstrates SMS platform-specific features
// This shows the complete SMS multi-provider integration capabilities in the unified architecture
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/sms"
)

func main() {
	fmt.Println("📱 SMS Platform Features Demo")
	fmt.Println("============================")
	fmt.Println()

	// Part 1: Different SMS Providers
	fmt.Println("🌐 Part 1: SMS Provider Configurations")
	fmt.Println("------------------------------------")

	// 1. Twilio SMS Provider
	fmt.Println("1. Twilio SMS Configuration")
	// Get Twilio credentials from environment variables
	twilioAPIKey := os.Getenv("TWILIO_API_KEY")
	twilioFromNumber := os.Getenv("TWILIO_FROM_NUMBER")

	if twilioAPIKey == "" || twilioFromNumber == "" {
		log.Fatal("TWILIO_API_KEY and TWILIO_FROM_NUMBER environment variables must be set")
	}
	twilioHub, err := notifyhub.New(
		sms.WithSMSTwilio(twilioAPIKey, twilioFromNumber),
	)
	if err != nil {
		log.Fatalf("Failed to create Twilio hub: %v", err)
	}
	defer func() { _ = twilioHub.Close() }()
	fmt.Println("✅ Twilio SMS hub created")

	// 2. Aliyun SMS Provider
	fmt.Println("2. Aliyun SMS Configuration")
	aliyunHub, err := notifyhub.New(
		sms.WithSMSAliyun("demo-aliyun-api-key", "+8612345678901",
			sms.WithSMSAPISecret("demo-secret"),
			sms.WithSMSSignName("阿里云"),
			sms.WithSMSTemplate("SMS_123456789"),
		),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create Aliyun SMS hub: %v", err)
	}
	defer func() { _ = aliyunHub.Close() }()
	fmt.Println("✅ Aliyun SMS hub created")

	// 3. Tencent SMS Provider
	fmt.Println("3. Tencent SMS Configuration")
	tencentHub, err := notifyhub.New(
		sms.WithSMSTencent("demo-tencent-api-key", "+8687654321098",
			sms.WithSMSAPISecret("demo-secret"),
			sms.WithSMSRegion("ap-beijing"),
		),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create Tencent SMS hub: %v", err)
	}
	defer func() { _ = tencentHub.Close() }()
	fmt.Println("✅ Tencent SMS hub created")

	// 4. AWS SNS SMS Provider
	fmt.Println("4. AWS SNS SMS Configuration")
	awsHub, err := notifyhub.New(
		sms.WithSMSAWS("demo-aws-access-key", "+1987654321",
			sms.WithSMSAPISecret("demo-secret-key"),
			sms.WithSMSRegion("us-east-1"),
		),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create AWS SNS hub: %v", err)
	}
	defer func() { _ = awsHub.Close() }()
	fmt.Println("✅ AWS SNS hub created")
	fmt.Println()

	ctx := context.Background()

	// Part 2: Basic SMS Messages
	fmt.Println("📝 Part 2: Basic SMS Messages")
	fmt.Println("---------------------------")

	// Simple SMS
	simpleMsg := notifyhub.NewMessage("System Alert").
		WithBody("Database backup completed successfully at " + time.Now().Format("15:04")).
		ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
		Build()

	if receipt, err := twilioHub.Send(ctx, simpleMsg); err != nil {
		fmt.Printf("❌ Simple SMS failed: %v\n", err)
	} else {
		fmt.Printf("✅ Simple SMS sent (Duration: %dms)\n",
			receipt.Results[0].Duration.Milliseconds())
	}

	// SMS with different priority levels
	alertMsg := notifyhub.NewAlert("Critical Issue").
		WithBody("Payment system is experiencing delays. Investigating...").
		ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
		Build()

	urgentMsg := notifyhub.NewUrgent("URGENT: Service Down").
		WithBody("Payment service is completely down. Immediate action required!").
		ToTarget(notifyhub.NewTarget("phone", "+1987654321", "sms")).
		Build()

	// Send different priority messages
	priorityMessages := map[string]*notifyhub.Message{
		"Alert":  alertMsg,
		"Urgent": urgentMsg,
	}

	for priority, msg := range priorityMessages {
		if _, err := twilioHub.Send(ctx, msg); err != nil {
			fmt.Printf("❌ %s SMS failed: %v\n", priority, err)
		} else {
			fmt.Printf("✅ %s SMS sent (Priority: %d)\n", priority, msg.Priority)
		}
	}
	fmt.Println()

	// Part 3: Template-Based SMS
	fmt.Println("📋 Part 3: Template-Based SMS")
	fmt.Println("---------------------------")

	// SMS with template variables
	templateMsg := notifyhub.NewMessage("Account Alert").
		WithPlatformData(map[string]interface{}{
			"sms_template": "ALERT: Login from {{location}} at {{time}}. If not you, contact support.",
			"sms_variables": map[string]interface{}{
				"location": "New York",
				"time":     time.Now().Format("15:04"),
			},
		}).
		ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
		Build()

	if _, err := twilioHub.Send(ctx, templateMsg); err != nil {
		fmt.Printf("❌ Template SMS failed: %v\n", err)
	} else {
		fmt.Printf("✅ Template SMS sent\n")
	}

	// Verification code SMS
	verificationMsg := notifyhub.NewMessage("Verification Code").
		WithPlatformData(map[string]interface{}{
			"sms_template": "Your verification code is {{code}}. Valid for {{duration}} minutes.",
			"sms_variables": map[string]interface{}{
				"code":     "123456",
				"duration": "5",
			},
		}).
		ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
		Build()

	if _, err := twilioHub.Send(ctx, verificationMsg); err != nil {
		fmt.Printf("❌ Verification SMS failed: %v\n", err)
	} else {
		fmt.Printf("✅ Verification code SMS sent\n")
	}
	fmt.Println()

	// Part 4: Provider-Specific Features
	fmt.Println("🔧 Part 4: Provider-Specific Features")
	fmt.Println("-----------------------------------")

	// Aliyun with signature and template
	aliyunMsg := notifyhub.NewMessage("阿里云短信").
		WithPlatformData(map[string]interface{}{
			"sms_template": "SMS_123456789",
			"sms_variables": map[string]interface{}{
				"code": "567890",
				"time": "5",
			},
		}).
		ToTarget(notifyhub.NewTarget("phone", "+8613812345678", "sms")).
		Build()

	fmt.Println("Sending Aliyun SMS with template...")
	if _, err := aliyunHub.Send(ctx, aliyunMsg); err != nil {
		fmt.Printf("❌ Aliyun SMS failed: %v\n", err)
	} else {
		fmt.Printf("✅ Aliyun SMS sent with signature\n")
	}

	// AWS SNS with region-specific settings
	awsMsg := notifyhub.NewMessage("AWS SNS Alert").
		WithBody("Your AWS resource usage is at 80% of quota.").
		ToTarget(notifyhub.NewTarget("phone", "+1555123456", "sms")).
		Build()

	fmt.Println("Sending AWS SNS SMS...")
	if _, err := awsHub.Send(ctx, awsMsg); err != nil {
		fmt.Printf("❌ AWS SNS SMS failed: %v\n", err)
	} else {
		fmt.Printf("✅ AWS SNS SMS sent\n")
	}
	fmt.Println()

	// Part 5: Multiple Recipients
	fmt.Println("👥 Part 5: Multiple Recipients")
	fmt.Println("----------------------------")

	// Broadcast SMS to multiple numbers
	broadcastMsg := notifyhub.NewAlert("System Maintenance").
		WithBody("Scheduled maintenance will begin at 2:00 AM EST. Expected duration: 2 hours.").
		// Multiple phone numbers
		ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
		ToTarget(notifyhub.NewTarget("phone", "+1987654321", "sms")).
		ToTarget(notifyhub.NewTarget("phone", "+1555123456", "sms")).
		Build()

	receipt, err := twilioHub.Send(ctx, broadcastMsg)
	if err != nil {
		fmt.Printf("❌ Broadcast SMS failed: %v\n", err)
	} else {
		fmt.Printf("✅ Broadcast SMS sent to %d recipients\n", len(receipt.Results))
		for _, result := range receipt.Results {
			status := "✅"
			if !result.Success {
				status = "❌"
			}
			fmt.Printf("   %s %s\n", status, result.Target)
		}
	}
	fmt.Println()

	// Part 6: SMS Use Cases
	fmt.Println("💼 Part 6: SMS Use Cases")
	fmt.Println("----------------------")

	useCases := map[string]*notifyhub.Message{
		"Authentication": notifyhub.NewMessage("2FA Code").
			WithBody("Your login code is 789012. Do not share this code.").
			ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
			Build(),

		"Delivery Update": notifyhub.NewMessage("Package Update").
			WithBody("Your package #PKG123 is out for delivery. Expected arrival: 2-4 PM.").
			ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
			Build(),

		"Payment Alert": notifyhub.NewAlert("Transaction Alert").
			WithBody("$250.00 charged to card ending in 1234 at Amazon.com. Not you? Call us.").
			ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
			Build(),

		"Appointment Reminder": notifyhub.NewMessage("Appointment Reminder").
			WithBody("Reminder: Doctor appointment tomorrow at 3:00 PM. Reply CONFIRM or CANCEL.").
			ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
			Build(),

		"Emergency Alert": notifyhub.NewUrgent("EMERGENCY").
			WithBody("URGENT: Server room temperature critical (85°C). Immediate attention required!").
			ToTarget(notifyhub.NewTarget("phone", "+1987654321", "sms")).
			Build(),
	}

	for useCase, msg := range useCases {
		if _, err := twilioHub.Send(ctx, msg); err != nil {
			fmt.Printf("❌ %s SMS failed: %v\n", useCase, err)
		} else {
			fmt.Printf("✅ %s SMS sent (Priority: %d)\n", useCase, msg.Priority)
		}
	}
	fmt.Println()

	// Part 7: Provider Comparison
	fmt.Println("⚖️  Part 7: Provider Comparison")
	fmt.Println("-----------------------------")
	fmt.Println("Provider Feature Comparison:")
	fmt.Println()

	fmt.Println("📱 TWILIO:")
	fmt.Println("   • Global coverage, reliable delivery")
	fmt.Println("   • E.164 phone number format required")
	fmt.Println("   • Rich API with delivery receipts")
	fmt.Println("   • Higher cost but premium features")
	fmt.Println()

	fmt.Println("☁️  ALIYUN (阿里云):")
	fmt.Println("   • Strong in China and Asia Pacific")
	fmt.Println("   • Template-based SMS required")
	fmt.Println("   • Signature name mandatory")
	fmt.Println("   • Cost-effective for Chinese market")
	fmt.Println()

	fmt.Println("🐧 TENCENT (腾讯云):")
	fmt.Println("   • Competitive in Chinese market")
	fmt.Println("   • Good domestic delivery rates")
	fmt.Println("   • Template and signature support")
	fmt.Println("   • Integrated with WeChat ecosystem")
	fmt.Println()

	fmt.Println("🚀 AWS SNS:")
	fmt.Println("   • Part of AWS ecosystem")
	fmt.Println("   • Pay-as-you-go pricing")
	fmt.Println("   • Global infrastructure")
	fmt.Println("   • Integrates with other AWS services")
	fmt.Println()

	// Part 8: Advanced Configuration
	fmt.Println("⚙️  Part 8: Advanced Configuration")
	fmt.Println("--------------------------------")

	// Custom configuration with all options
	advancedHub, err := notifyhub.New(
		sms.WithSMS("twilio", "advanced-api-key", "+1555000000",
			sms.WithSMSAPISecret("api-secret"),
			sms.WithSMSTimeout(45*time.Second),
			sms.WithSMSRegion("us-east-1"),
		),
	)
	if err != nil {
		log.Printf("❌ Advanced SMS hub creation failed: %v", err)
	} else {
		defer func() { _ = advancedHub.Close() }()
		fmt.Println("✅ Advanced SMS configuration created")

		advancedMsg := notifyhub.NewMessage("Advanced Config Test").
			WithBody("SMS sent with advanced configuration settings.").
			ToTarget(notifyhub.NewTarget("phone", "+1234567890", "sms")).
			Build()

		if _, err := advancedHub.Send(ctx, advancedMsg); err != nil {
			fmt.Printf("❌ Advanced config test failed: %v\n", err)
		} else {
			fmt.Printf("✅ Advanced config SMS sent\n")
		}
	}
	fmt.Println()

	// Part 9: Error Handling and Best Practices
	fmt.Println("🛡️  Part 9: Best Practices")
	fmt.Println("------------------------")
	fmt.Println("SMS Best Practices:")
	fmt.Println()
	fmt.Println("📏 MESSAGE LENGTH:")
	fmt.Println("   • Keep under 160 characters for single SMS")
	fmt.Println("   • Longer messages split into multiple parts")
	fmt.Println("   • Consider costs for multi-part messages")
	fmt.Println()
	fmt.Println("📞 PHONE NUMBER FORMAT:")
	fmt.Println("   • Use E.164 format (+country code + number)")
	fmt.Println("   • Validate numbers before sending")
	fmt.Println("   • Handle international vs domestic formats")
	fmt.Println()
	fmt.Println("🔒 SECURITY:")
	fmt.Println("   • Never include sensitive data in SMS")
	fmt.Println("   • Use secure API key storage")
	fmt.Println("   • Implement rate limiting")
	fmt.Println("   • Log sending attempts for audit")
	fmt.Println()
	fmt.Println("⏱️  TIMING:")
	fmt.Println("   • Respect time zones for recipients")
	fmt.Println("   • Avoid sending during night hours")
	fmt.Println("   • Consider business vs personal numbers")
	fmt.Println()
	fmt.Println("💰 COST OPTIMIZATION:")
	fmt.Println("   • Monitor usage and costs")
	fmt.Println("   • Use templates to reduce message length")
	fmt.Println("   • Implement opt-out mechanisms")
	fmt.Println("   • Choose providers based on target regions")
	fmt.Println()

	// Summary
	fmt.Println("📱 SMS Platform Features Summary")
	fmt.Println("===============================")
	fmt.Println("✅ MULTI-PROVIDER SUPPORT:")
	fmt.Println("  • Twilio (global)")
	fmt.Println("  • Aliyun (China/APAC)")
	fmt.Println("  • Tencent (China)")
	fmt.Println("  • AWS SNS (global)")
	fmt.Println()
	fmt.Println("✅ MESSAGE FEATURES:")
	fmt.Println("  • Plain text SMS")
	fmt.Println("  • Template-based messages")
	fmt.Println("  • Variable substitution")
	fmt.Println("  • Multiple recipients")
	fmt.Println()
	fmt.Println("✅ PROVIDER FEATURES:")
	fmt.Println("  • Region-specific optimization")
	fmt.Println("  • Template and signature support")
	fmt.Println("  • Custom timeouts and retries")
	fmt.Println("  • E.164 phone number validation")
	fmt.Println()

	fmt.Println("📱 SMS Platform Demo Complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("• Try examples/platforms/unified-demo/ for all platforms together")
	fmt.Println("• See examples/external/ for creating custom SMS providers")
	fmt.Println("• Check examples/advanced/ for production patterns")
}
