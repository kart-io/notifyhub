// Package core provides the core Hub implementation
package core

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"

	// Auto-register internal platforms
	_ "github.com/kart-io/notifyhub/pkg/platforms/dingtalk"
	_ "github.com/kart-io/notifyhub/pkg/platforms/email"
	_ "github.com/kart-io/notifyhub/pkg/platforms/feishu"
	_ "github.com/kart-io/notifyhub/pkg/platforms/slack"
)

// No special initialization needed - platforms register themselves

// hubImpl implements the Hub interface using the public platform manager
type hubImpl struct {
	config  *config.Config
	manager *PublicPlatformManager
	logger  logger.Logger
}

// NewHub creates a new Hub implementation with the given configuration
func NewHub(cfg *config.Config) (Hub, error) {
	if cfg.Logger != nil {
		cfg.Logger.Debug("Creating new hub", "platformCount", len(cfg.Platforms))
	}

	// Ensure internal platforms are registered
	ensureInternalPlatformsRegistered()

	// Create public platform manager
	manager := NewPublicPlatformManager()
	if cfg.Logger != nil {
		manager.SetLogger(cfg.Logger)
	}

	// Register senders for each configured platform
	for platformName, platformConfig := range cfg.Platforms {
		if cfg.Logger != nil {
			cfg.Logger.Debug("Creating sender", "platform", platformName)
		}
		sender, err := manager.CreateSender(platformName, platformConfig, cfg.Logger)
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.Error("Failed to create sender", "platform", platformName, "error", err)
			}
			return nil, fmt.Errorf("failed to create %s sender: %w", platformName, err)
		}

		if err := manager.RegisterSender(sender); err != nil {
			if cfg.Logger != nil {
				cfg.Logger.Error("Failed to register sender", "platform", platformName, "error", err)
			}
			return nil, fmt.Errorf("failed to register %s sender: %w", platformName, err)
		}
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("Hub created successfully", "platformCount", len(cfg.Platforms))
	}
	return &hubImpl{
		config:  cfg,
		manager: manager,
		logger:  cfg.Logger,
	}, nil
}

// Send sends a message synchronously
func (h *hubImpl) Send(ctx context.Context, msg *message.Message) (*receipt.Receipt, error) {
	h.logger.Debug("Sending message", "messageID", msg.ID, "title", msg.Title, "targetCount", len(msg.Targets))

	// Send directly using unified types (no conversion needed)
	h.logger.Debug("Dispatching message to platform manager", "messageID", msg.ID)
	results, err := h.manager.SendToAll(ctx, msg, msg.Targets)
	h.logger.Debug("Received results from platform manager", "messageID", msg.ID, "resultCount", len(results), "error", err)

	// Convert results to receipt
	receipt := h.convertToReceipt(msg.ID, results)
	h.logger.Debug("Converted results to receipt", "messageID", msg.ID, "status", receipt.Status)

	return receipt, err
}

// SendAsync sends a message asynchronously
func (h *hubImpl) SendAsync(ctx context.Context, msg *message.Message) (*receipt.AsyncReceipt, error) {
	h.logger.Debug("SendAsync called", "messageID", msg.ID, "title", msg.Title)

	// For now, implement async as sync and return immediately
	// In a real implementation, this would queue the message
	_, err := h.Send(ctx, msg)
	if err != nil {
		h.logger.Error("Failed to send message async", "messageID", msg.ID, "error", err)
		return nil, err
	}

	h.logger.Info("Message queued for async sending", "messageID", msg.ID)
	return &receipt.AsyncReceipt{
		MessageID: msg.ID,
		Status:    "queued",
		QueuedAt:  time.Now(),
	}, nil
}

// Health checks the health of all platforms
func (h *hubImpl) Health(ctx context.Context) (*HealthStatus, error) {
	h.logger.Debug("Performing health check")

	healthMap := h.manager.HealthCheck(ctx)

	platforms := make(map[string]PlatformHealth)
	unhealthyPlatforms := []string{}
	for platformName, err := range healthMap {
		health := PlatformHealth{
			Available: err == nil,
			Status:    "healthy",
		}
		if err != nil {
			health.Status = err.Error()
			health.Details = map[string]string{"error": err.Error()}
			unhealthyPlatforms = append(unhealthyPlatforms, platformName)
			h.logger.Warn("Platform unhealthy", "platform", platformName, "error", err)
		}
		platforms[platformName] = health
	}

	// Determine overall health
	healthy := true
	status := "healthy"
	for _, platform := range platforms {
		if !platform.Available {
			healthy = false
			status = "unhealthy"
			break
		}
	}

	if healthy {
		h.logger.Info("Health check completed - all platforms healthy", "platformCount", len(platforms))
	} else {
		h.logger.Warn("Health check completed - some platforms unhealthy", "unhealthyPlatforms", unhealthyPlatforms)
	}

	return &HealthStatus{
		Healthy:   healthy,
		Status:    status,
		Platforms: platforms,
		Queue:     QueueHealth{Available: true}, // Simplified for now
		Timestamp: time.Now(),
	}, nil
}

// Close shuts down the hub
func (h *hubImpl) Close(ctx context.Context) error {
	h.logger.Info("Closing hub")
	err := h.manager.Close()
	if err != nil {
		h.logger.Error("Failed to close hub cleanly", "error", err)
	} else {
		h.logger.Info("Hub closed successfully")
	}
	return err
}

// Helper methods for conversion

func (h *hubImpl) convertToReceipt(messageID string, results []*LocalSendResult) *receipt.Receipt {
	platformResults := make([]receipt.PlatformResult, len(results))
	successful := 0
	failed := 0

	for i, result := range results {
		// Calculate duration from metadata if available
		var duration time.Duration
		if durationMs, ok := result.Metadata["duration"].(int64); ok {
			duration = time.Duration(durationMs) * time.Millisecond
		}

		platformResults[i] = receipt.PlatformResult{
			Platform:  result.Target.Platform,
			Target:    result.Target.Value,
			Success:   result.Success,
			MessageID: result.MessageID,
			Error:     result.Error,
			Timestamp: time.Now(), // Use current time since SendResult doesn't have timestamp
			Duration:  duration,
		}

		if result.Success {
			successful++
		} else {
			failed++
		}
	}

	status := "success"
	if failed > 0 {
		if successful == 0 {
			status = "failed"
		} else {
			status = "partial"
		}
	}

	return &receipt.Receipt{
		MessageID:  messageID,
		Status:     status,
		Results:    platformResults,
		Successful: successful,
		Failed:     failed,
		Total:      len(results),
		Timestamp:  time.Now(),
	}
}
