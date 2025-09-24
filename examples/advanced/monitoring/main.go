// Package main demonstrates monitoring and observability patterns
// This shows enterprise monitoring with health checks, metrics, and alerting
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/feishu"
)

func main() {
	fmt.Println("📊 Monitoring and Observability Demo")
	fmt.Println("====================================")
	fmt.Println()

	// Create a hub with monitoring enabled (conceptual)
	hub, err := notifyhub.New(
		feishu.WithFeishu(os.Getenv("FEISHU_WEBHOOK_URL")),
		// Add monitoring middleware
		// notifyhub.WithMiddleware(metrics.NewPrometheusMiddleware()),
	)
	if err != nil {
		log.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Close() }()

	// Demonstrate monitoring patterns
	performHealthChecks(hub)
	collectPerformanceMetrics(hub)
	monitorErrorRates(hub)
	collectPlatformMetrics(hub)
	alertingDemo(hub)
	createMonitoringDashboard(hub)
	demonstrateLogAggregation(hub)

	fmt.Println("\n✅ Monitoring and Observability Demo Complete!")
}

// Monitoring examples

func performHealthChecks(hub notifyhub.Client) {
	fmt.Println("\n--- Health Checks ---")
	health, err := hub.Health(context.Background())
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		return
	}
	fmt.Printf("Overall status: %s\n", health.Status)
	for name, status := range health.Platforms {
		fmt.Printf("  - %s: %s\n", name, status.Status)
	}
}

func collectPerformanceMetrics(hub notifyhub.Client) {
	fmt.Println("\n--- Performance Metrics ---")
	msg := notifyhub.NewMessage("Performance Test").WithBody("Measuring send duration.").ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).Build()
	start := time.Now()
	receipt, _ := hub.Send(context.Background(), msg)
	duration := time.Since(start)
	fmt.Printf("Message send duration: %dms\n", duration.Milliseconds())
	if receipt != nil && len(receipt.Results) > 0 {
		fmt.Printf("Platform-reported duration: %dms\n", receipt.Results[0].Duration.Milliseconds())
	}
}

func monitorErrorRates(hub notifyhub.Client) {
	fmt.Println("\n--- Error Rate Monitoring ---")
	// Simulate sending some messages that will fail
	for i := 0; i < 3; i++ {
		msg := notifyhub.NewMessage("Error Rate Test").WithBody("This message will fail.").ToTarget(notifyhub.NewTarget("invalid-type", "", "feishu")).Build()
		_, _ = hub.Send(context.Background(), msg)
	}
	fmt.Println("Simulated 3 failed messages to track error rate.")
}

func collectPlatformMetrics(hub notifyhub.Client) {
	fmt.Println("\n--- Platform-Specific Metrics ---")
	status, err := hub.GetPlatformStatus(context.Background(), "feishu")
	if err != nil {
		fmt.Printf("Failed to get platform status: %v\n", err)
	} else {
		fmt.Printf("Feishu platform status: %s, Available: %v\n", status.Status, status.Available)
	}
}

func alertingDemo(hub notifyhub.Client) {
	fmt.Println("\n--- Alerting on Failures ---")
	msg := notifyhub.NewMessage("Alerting Test").WithBody("This is a critical alert.").ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).Build()
	receipt, err := hub.Send(context.Background(), msg)
	if err != nil || (receipt != nil && receipt.Failed > 0) {
		fmt.Println("ALERT: Message delivery failed! Triggering alert to on-call engineer.")
		// Here you would integrate with PagerDuty, Opsgenie, etc.
	}
}

func createMonitoringDashboard(hub notifyhub.Client) {
	fmt.Println("\n--- Monitoring Dashboard (Conceptual) ---")
	fmt.Println("  - [Gauge] notifyhub_up: 1")
	fmt.Println("  - [Counter] notifyhub_messages_sent_total{platform='feishu'}: 10")
	fmt.Println("  - [Counter] notifyhub_messages_failed_total{platform='feishu'}: 2")
	fmt.Println("  - [Histogram] notifyhub_message_duration_seconds{platform='feishu'}: ...")
}

func demonstrateLogAggregation(hub notifyhub.Client) {
	fmt.Println("\n--- Log Aggregation ---")
	fmt.Println("Logs are being sent to a central logging system (e.g., ELK, Splunk, Datadog).")
	fmt.Println(`  Example log entry: {"level":"info", "time":"...", "msg":"Message sent", "message_id":"...", "platform":"feishu"}`)
}
