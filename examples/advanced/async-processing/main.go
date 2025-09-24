// Package main demonstrates async message processing with queue support
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub"
)

func main() {
	// Create logger
	log := logger.New().LogMode(logger.Debug)

	// Create base hub with notification platforms
	//nolint:staticcheck // NewHub is needed to get Hub interface for AsyncHub
	baseHub, err := notifyhub.NewHub(
		notifyhub.WithLogger(log),
		notifyhub.WithFeishu(
			os.Getenv("FEISHU_WEBHOOK_URL"),
			os.Getenv("FEISHU_SECRET"),
		),
		notifyhub.WithEmail(
			os.Getenv("SMTP_HOST"),
			587,
			os.Getenv("SMTP_FROM"),
		),
	)
	if err != nil {
		log.Error("Failed to create hub", "error", err)
		os.Exit(1)
	}

	// Create async hub with memory queue
	asyncHub, err := notifyhub.NewAsyncHubWithOptions(
		baseHub,
		log,
		notifyhub.WithMemoryQueue(1000, 4), // 1000 capacity, 4 workers
		notifyhub.WithQueueRetry(3, 1*time.Second),
		notifyhub.WithDeadLetterQueue(true),
	)
	if err != nil {
		log.Error("Failed to create async hub", "error", err)
		os.Exit(1)
	}

	// Start processing queued messages
	ctx := context.Background()
	if err := asyncHub.ProcessQueuedMessages(ctx); err != nil {
		log.Error("Failed to start processing", "error", err)
		os.Exit(1)
	}

	// Example 1: Send immediate high-priority alert
	alert := notifyhub.NewAlert("System Alert").
		WithBody("CPU usage exceeds 90%").
		WithPriority(notifyhub.PriorityUrgent).
		ToEmail("ops@example.com").
		Build()

	receipt, err := asyncHub.SendQueued(ctx, alert)
	if err != nil {
		log.Error("Failed to queue alert", "error", err)
	} else {
		log.Info("Alert queued",
			"messageID", receipt.MessageID,
			"status", receipt.Status,
			"queuedAt", receipt.QueuedAt)
	}

	// Example 2: Batch queue multiple notifications
	notifications := []struct {
		title    string
		body     string
		priority notifyhub.Priority
		email    string
	}{
		{"Daily Report", "Sales report ready", notifyhub.PriorityNormal, "sales@example.com"},
		{"Weekly Summary", "Team metrics available", notifyhub.PriorityNormal, "team@example.com"},
		{"Monthly Review", "Performance dashboard updated", notifyhub.PriorityLow, "hr@example.com"},
	}

	for _, n := range notifications {
		msg := notifyhub.NewMessage(n.title).
			WithBody(n.body).
			WithPriority(n.priority).
			ToEmail(n.email).
			Build()

		if _, err := asyncHub.SendQueued(ctx, msg); err != nil {
			log.Error("Failed to queue message",
				"title", n.title,
				"error", err)
		}
	}

	// Example 3: Schedule a delayed message
	scheduledTime := time.Now().Add(5 * time.Minute)
	reminder := notifyhub.NewMessage("Meeting Reminder").
		WithBody("Team standup in 10 minutes").
		ScheduleAt(scheduledTime).
		ToCustomTarget("channel", "team-channel", "feishu").
		Build()

	if _, err := asyncHub.SendQueued(ctx, reminder); err != nil {
		log.Error("Failed to queue scheduled message", "error", err)
	} else {
		log.Info("Scheduled message queued", "scheduledAt", scheduledTime)
	}

	// Monitor queue statistics
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := asyncHub.GetQueueStats()
			log.Info("Queue statistics",
				"queueSize", stats["queue_size"],
				"isEmpty", stats["is_empty"],
				"processing", stats["processing"])

			if workers, ok := stats["workers"].(map[string]interface{}); ok {
				log.Debug("Worker pool stats",
					"workerCount", workers["worker_count"],
					"processedCount", workers["processed_count"],
					"errorCount", workers["error_count"])
			}
		}
	}()

	// Example 4: Process with custom retry policy
	criticalMsg := notifyhub.NewMessage("Critical Update").
		WithBody("Database backup failed").
		WithPriority(notifyhub.PriorityUrgent).
		WithMetadata("max_retries", 5).
		WithMetadata("alert_level", "critical").
		ToEmail("dba@example.com").
		ToCustomTarget("user", "@oncall", "feishu").
		Build()

	if _, err := asyncHub.SendQueued(ctx, criticalMsg); err != nil {
		log.Error("Failed to queue critical message", "error", err)
	}

	// Example 5: Using Redis queue for persistence (if configured)
	// This would require Redis to be available and configured
	if os.Getenv("REDIS_ADDR") != "" {
		redisHub, err := notifyhub.NewAsyncHubWithOptions(
			baseHub,
			log,
			notifyhub.WithRedisQueue(os.Getenv("REDIS_ADDR"), 10000, 8),
			notifyhub.WithQueueRetry(5, 2*time.Second),
			notifyhub.WithDeadLetterQueue(true),
		)
		if err == nil {
			// Use Redis-backed queue for persistent messages
			persistentMsg := notifyhub.NewMessage("Persistent Alert").
				WithBody("This message survives restarts").
				ToEmail("admin@example.com").
				Build()

			if _, err := redisHub.SendQueued(ctx, persistentMsg); err != nil {
				log.Error("Failed to queue to Redis", "error", err)
			}
			defer func() {
				if err := redisHub.Close(ctx); err != nil {
					log.Error("Failed to close Redis hub", "error", err)
				}
			}()
		}
	}

	// Set up graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Async processing started. Press Ctrl+C to stop...")

	<-sigCh
	log.Info("Shutting down...")

	// Stop processing
	if err := asyncHub.StopProcessing(); err != nil {
		log.Error("Error stopping processing", "error", err)
	}

	// Close the hub
	if err := asyncHub.Close(ctx); err != nil {
		log.Error("Error closing async hub", "error", err)
	}

	log.Info("Shutdown complete")
}
