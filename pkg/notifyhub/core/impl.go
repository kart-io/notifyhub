// Package core provides the core Hub implementation
package core

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
	// Note: Internal platform registration is handled during NewHub creation
	// No need to import internal packages here
)

// No special initialization needed - platforms register themselves

// hubImpl implements the Hub interface using the public platform manager
type hubImpl struct {
	config  *config.HubConfig
	manager *PublicPlatformManager
}

// NewHub creates a new Hub implementation with the given configuration
func NewHub(cfg *config.HubConfig) (Hub, error) {
	// Ensure internal platforms are registered
	ensureInternalPlatformsRegistered()

	// Create public platform manager
	manager := NewPlatformManager()

	// Register senders for each configured platform
	for platformName, platformConfig := range cfg.Platforms {
		sender, err := manager.CreateSender(platformName, map[string]interface{}(platformConfig))
		if err != nil {
			return nil, fmt.Errorf("failed to create %s sender: %w", platformName, err)
		}

		if err := manager.RegisterSender(sender); err != nil {
			return nil, fmt.Errorf("failed to register %s sender: %w", platformName, err)
		}
	}

	return &hubImpl{
		config:  cfg,
		manager: manager,
	}, nil
}

// Send sends a message synchronously
func (h *hubImpl) Send(ctx context.Context, msg *message.Message) (*receipt.Receipt, error) {
	// Convert public message to platform message format
	platformMsg := h.convertToPlatformMessage(msg)

	// Convert targets to platform target format
	platformTargets := h.convertToPlatformTargets(msg.Targets)

	// Send through platform manager
	results, err := h.manager.SendToAll(ctx, platformMsg, platformTargets)

	// Convert results to receipt
	receipt := h.convertToReceipt(msg.ID, results)

	return receipt, err
}

// SendAsync sends a message asynchronously
func (h *hubImpl) SendAsync(ctx context.Context, msg *message.Message) (*receipt.AsyncReceipt, error) {
	// For now, implement async as sync and return immediately
	// In a real implementation, this would queue the message
	_, err := h.Send(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &receipt.AsyncReceipt{
		MessageID: msg.ID,
		Status:    "queued",
		QueuedAt:  time.Now(),
	}, nil
}

// Health checks the health of all platforms
func (h *hubImpl) Health(ctx context.Context) (*HealthStatus, error) {
	healthMap := h.manager.HealthCheck(ctx)

	platforms := make(map[string]PlatformHealth)
	for platformName, err := range healthMap {
		health := PlatformHealth{
			Available: err == nil,
			Status:    "healthy",
		}
		if err != nil {
			health.Status = err.Error()
			health.Details = map[string]string{"error": err.Error()}
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
	return h.manager.Close()
}

// Helper methods for conversion

func (h *hubImpl) convertToPlatformMessage(msg *message.Message) *platform.Message {
	platformMsg := &platform.Message{
		ID:           msg.ID,
		Title:        msg.Title,
		Body:         msg.Body,
		Format:       msg.Format,
		Priority:     int(msg.Priority),
		Metadata:     msg.Metadata,
		Variables:    msg.Variables,
		PlatformData: make(map[string]interface{}),
	}

	// Copy platform-specific data
	if msg.PlatformData != nil {
		for key, value := range msg.PlatformData {
			platformMsg.PlatformData[key] = value
		}
	}

	return platformMsg
}

func (h *hubImpl) convertToPlatformTargets(targets []target.Target) []platform.Target {
	platformTargets := make([]platform.Target, len(targets))
	for i, t := range targets {
		platformTargets[i] = platform.Target{
			Type:     t.Type,
			Value:    t.Value,
			Platform: t.Platform,
		}
	}
	return platformTargets
}

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
