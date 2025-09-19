package middleware

import (
	"context"

	"github.com/kart-io/notifyhub/config/routing"
	"github.com/kart-io/notifyhub/core/hub"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/logger"
)

// RoutingMiddleware applies routing rules to determine message targets
type RoutingMiddleware struct {
	engine *routing.Engine
	logger logger.Interface
}

// NewRoutingMiddleware creates a new routing middleware
func NewRoutingMiddleware(rules []routing.Rule, logger logger.Interface) *RoutingMiddleware {
	return &RoutingMiddleware{
		engine: routing.NewEngine(rules),
		logger: logger,
	}
}

// Process processes the message through routing rules
func (m *RoutingMiddleware) Process(ctx context.Context, msg *message.Message, targets []sending.Target, next hub.ProcessFunc) (*sending.SendingResults, error) {
	// Convert message to map for routing engine
	msgMap := m.convertToMap(msg, targets)

	// Apply routing rules
	processedMap, err := m.engine.ProcessMessage(msgMap)
	if err != nil {
		return nil, err
	}

	// Extract targets from processed map
	newTargets := m.extractTargets(processedMap)

	// Update message from routing results
	m.updateMessageFromMap(msg, processedMap)

	// Log routing results
	if m.logger != nil {
		m.logger.Info(ctx, "routing processed", "original_targets", len(targets), "new_targets", len(newTargets))
	}

	// Continue to next middleware
	return next(ctx, msg, newTargets)
}

// convertToMap converts message to map for routing engine
func (m *RoutingMiddleware) convertToMap(msg *message.Message, targets []sending.Target) map[string]interface{} {
	// Convert targets to interfaces
	targetMaps := make([]interface{}, len(targets))
	for i, target := range targets {
		targetMaps[i] = map[string]interface{}{
			"type":     target.Type,
			"value":    target.Value,
			"platform": target.Platform,
			"metadata": target.Metadata,
		}
	}

	return map[string]interface{}{
		"id":        msg.ID,
		"title":     msg.Title,
		"body":      msg.Body,
		"format":    msg.Format,
		"priority":  msg.Priority,
		"template":  msg.Template,
		"variables": msg.Variables,
		"metadata":  msg.Metadata,
		"targets":   targetMaps,
	}
}

// extractTargets extracts targets from processed map
func (m *RoutingMiddleware) extractTargets(processedMap map[string]interface{}) []sending.Target {
	var targets []sending.Target

	if targetsInterface, ok := processedMap["targets"]; ok {
		if targetSlice, ok := targetsInterface.([]interface{}); ok {
			for _, targetInterface := range targetSlice {
				if targetMap, ok := targetInterface.(map[string]interface{}); ok {
					target := sending.Target{}
					if typ, ok := targetMap["type"].(string); ok {
						target.Type = sending.TargetType(typ)
					}
					if value, ok := targetMap["value"].(string); ok {
						target.Value = value
					}
					if platform, ok := targetMap["platform"].(string); ok {
						target.Platform = platform
					}
					if metadata, ok := targetMap["metadata"].(map[string]string); ok {
						target.Metadata = metadata
					}
					targets = append(targets, target)
				}
			}
		}
	}

	return targets
}

// updateMessageFromMap updates message based on routing results
func (m *RoutingMiddleware) updateMessageFromMap(msg *message.Message, processedMap map[string]interface{}) {
	// Update priority if changed by routing
	if priority, ok := processedMap["priority"].(int); ok && priority != msg.Priority {
		msg.SetPriority(priority)
	}

	// Update metadata if changed by routing
	if metadata, ok := processedMap["metadata"].(map[string]string); ok {
		for key, value := range metadata {
			if existingValue, exists := msg.Metadata[key]; !exists || existingValue != value {
				msg.AddMetadata(key, value)
			}
		}
	}
}
