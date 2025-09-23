// Package main demonstrates the new elegant API for NotifyHub
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
	fmt.Println("=== NotifyHub Elegant API Demo ===")

	// Initialize hub with configuration
	hub, err := notifyhub.NewHub(
		notifyhub.WithFeishuFromMap(map[string]interface{}{
			"webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx",
			"secret":      "xxxxxxxx",
		}),
		notifyhub.WithEmailFromMap(map[string]interface{}{
			"smtp_host":     "smtp.gmail.com",
			"smtp_port":     587,
			"smtp_username": "your@gmail.com",
			"smtp_password": "your-password",
			"smtp_from":     "your@gmail.com",
			"smtp_tls":      true,
		}),
		notifyhub.WithSMSFromMap(map[string]interface{}{
			"provider": "twilio",
			"api_key":  "your-twilio-api-key",
			"from":     "+1234567890",
		}),
		notifyhub.WithTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create hub: %v", err)
	}
	defer func() {
		if err := hub.Close(context.Background()); err != nil {
			log.Printf("Failed to close hub: %v", err)
		}
	}()

	ctx := context.Background()

	// Example 1: Simple text message to multiple platforms
	fmt.Println("\n--- Example 1: Simple text message ---")
	receipt, err := hub.Send(ctx, notifyhub.NewMessage("Task Completed").
		WithText("Your data processing task has been completed successfully.").
		ToEmail("user@example.com").
		ToFeishu("oc_xxxxxxxx"). // Feishu user ID
		ToPhone("+1234567890").
		Build(),
	)
	if err != nil {
		fmt.Printf("Send failed: %v\n", err)
	} else {
		fmt.Printf("Send result: %s, Successful: %d, Failed: %d\n",
			receipt.Status, receipt.Successful, receipt.Failed)
	}

	// Example 2: Rich Feishu card message
	fmt.Println("\n--- Example 2: Feishu card message ---")
	cardContent := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**New Task Assigned**\nPlease check your task list.",
					"tag":     "lark_md",
				},
			},
		},
	}

	receipt, err = hub.Send(ctx, notifyhub.NewAlert("New Task Assigned").
		WithFeishuCard(cardContent).
		WithFeishuMentions("oc_xxxxxxxx").
		ToFeishuGroup("oc_xxxxxxxx"). // Feishu group ID
		Build(),
	)
	if err != nil {
		fmt.Printf("Send failed: %v\n", err)
	} else {
		fmt.Printf("Send result: %s\n", receipt.Status)
	}

	// Example 3: HTML email with CC/BCC
	fmt.Println("\n--- Example 3: HTML email with CC/BCC ---")
	htmlContent := `
	<h2 style="color: #2c3e50;">System Alert</h2>
	<p>Dear team,</p>
	<p>The following system alert has been triggered:</p>
	<div style="background-color: #f8f9fa; padding: 15px; border-left: 4px solid #007bff;">
		<strong>High CPU Usage Detected</strong><br>
		Server: web-server-01<br>
		CPU Usage: 95%<br>
		Time: ` + time.Now().Format("2006-01-02 15:04:05") + `
	</div>
	<p>Please investigate immediately.</p>
	<p>Best regards,<br>Monitoring System</p>
	`

	receipt, err = hub.Send(ctx, notifyhub.NewUrgent("System Alert: High CPU Usage").
		WithHTML(htmlContent).
		WithEmailCC("manager@example.com", "devops@example.com").
		WithEmailPriority("high").
		ToEmail("admin@example.com").
		Build(),
	)
	if err != nil {
		fmt.Printf("Send failed: %v\n", err)
	} else {
		fmt.Printf("Send result: %s\n", receipt.Status)
	}

	// Example 4: SMS with template variables
	fmt.Println("\n--- Example 4: SMS with template ---")
	receipt, err = hub.Send(ctx, notifyhub.NewMessage("Order Confirmation").
		WithSMSTemplate("order_confirmation").
		WithSMSVariables(map[string]interface{}{
			"order_id":    "ORD-12345",
			"total":       "$99.99",
			"customer":    "John Doe",
			"delivery_at": "2024-01-15 14:30",
		}).
		ToPhone("+1234567890").
		Build(),
	)
	if err != nil {
		fmt.Printf("Send failed: %v\n", err)
	} else {
		fmt.Printf("Send result: %s\n", receipt.Status)
	}

	// Example 5: Scheduled message
	fmt.Println("\n--- Example 5: Scheduled message ---")
	scheduleTime := time.Now().Add(5 * time.Minute)
	receipt, err = hub.Send(ctx, notifyhub.NewMessage("Reminder").
		WithText("Don't forget about your meeting at 3 PM.").
		ScheduleAt(scheduleTime).
		ToEmail("user@example.com").
		Build(),
	)
	if err != nil {
		fmt.Printf("Send failed: %v\n", err)
	} else {
		fmt.Printf("Scheduled message: %s, will be sent at %s\n",
			receipt.MessageID, scheduleTime.Format("15:04:05"))
	}

	// Example 6: Multi-target message with metadata
	fmt.Println("\n--- Example 6: Multi-target with metadata ---")
	message := notifyhub.NewMessage("Deployment Complete").
		WithMarkdown("## Deployment Status\n\n✅ **SUCCESS**: Version v2.1.0 deployed to production\n\n- **Deployment ID**: dep-789\n- **Duration**: 3m 45s\n- **Affected Services**: api, web, worker").
		WithMetadata("deployment_id", "dep-789").
		WithMetadata("version", "v2.1.0").
		WithVariable("service_count", 3).
		ToFeishuGroup("oc_xxxxxxxx").
		ToEmail("devops@example.com").
		Build()

	receipt, err = hub.Send(ctx, message)
	if err != nil {
		fmt.Printf("Send failed: %v\n", err)
	} else {
		fmt.Printf("Send result: %s, Message ID: %s\n", receipt.Status, receipt.MessageID)

		// Show detailed results
		for _, result := range receipt.Results {
			fmt.Printf("  Platform: %s, Target: %s, Success: %v\n",
				result.Platform, result.Target, result.Success)
		}
	}

	// Example 7: Async sending
	fmt.Println("\n--- Example 7: Async sending ---")
	asyncReceipt, err := hub.SendAsync(ctx, notifyhub.NewMessage("Background Task").
		WithText("This message is sent asynchronously.").
		ToEmail("user@example.com").
		Build(),
	)
	if err != nil {
		fmt.Printf("Async send failed: %v\n", err)
	} else {
		fmt.Printf("Async send queued: %s, Status: %s\n",
			asyncReceipt.MessageID, asyncReceipt.Status)
	}

	// Example 8: Health check
	fmt.Println("\n--- Example 8: Health check ---")
	health, err := hub.Health(ctx)
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
	} else {
		fmt.Printf("Overall health: %s\n", health.Status)
		for platform, status := range health.Platforms {
			fmt.Printf("  %s: %s\n", platform, status.Status)
		}
	}

	fmt.Println("\n=== Demo completed ===")
}
