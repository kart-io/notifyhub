// Package adapters provides adapters for internal platform implementations
package adapters

import (
	"context"
	"fmt"

	"github.com/kart-io/notifyhub/internal/platform"
	"github.com/kart-io/notifyhub/internal/platform/feishu"
	platformPkg "github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// FeishuAdapter wraps the internal Feishu sender to implement the public ExternalSender interface
type FeishuAdapter struct {
	internal *feishu.FeishuSender
}

// NewFeishuSender creates a new Feishu sender using the internal implementation
func NewFeishuSender(config map[string]interface{}) (platformPkg.ExternalSender, error) {
	// Create internal sender
	internalSender, err := feishu.NewFeishuSender(config)
	if err != nil {
		return nil, err
	}

	// Cast to internal type (we know it's a FeishuSender)
	feishuSender, ok := internalSender.(*feishu.FeishuSender)
	if !ok {
		return nil, fmt.Errorf("expected FeishuSender, got %T", internalSender)
	}

	return &FeishuAdapter{internal: feishuSender}, nil
}

// Name returns the platform name
func (f *FeishuAdapter) Name() string {
	return f.internal.Name()
}

// Send sends a message using the internal Feishu sender
func (f *FeishuAdapter) Send(ctx context.Context, msg *platformPkg.Message, targets []platformPkg.Target) ([]*platformPkg.SendResult, error) {
	// Convert platform types to internal types
	internalMsg := convertToInternalMessage(msg)
	internalTargets := convertToInternalTargets(targets)

	// Call internal sender
	internalResults, err := f.internal.Send(ctx, internalMsg, internalTargets)
	if err != nil {
		return nil, err
	}

	// Convert internal results to platform results
	return convertFromInternalResults(internalResults), nil
}

// ValidateTarget validates a target using the internal sender
func (f *FeishuAdapter) ValidateTarget(target platformPkg.Target) error {
	internalTarget := convertToInternalTarget(target)
	return f.internal.ValidateTarget(internalTarget)
}

// GetCapabilities returns the capabilities using the internal sender
func (f *FeishuAdapter) GetCapabilities() platformPkg.Capabilities {
	internalCaps := f.internal.GetCapabilities()
	return convertFromInternalCapabilities(internalCaps)
}

// IsHealthy checks health using the internal sender
func (f *FeishuAdapter) IsHealthy(ctx context.Context) error {
	return f.internal.IsHealthy(ctx)
}

// Close closes the internal sender
func (f *FeishuAdapter) Close() error {
	return f.internal.Close()
}

// Conversion functions between public and internal types

func convertToInternalMessage(msg *platformPkg.Message) *platform.InternalMessage {
	return &platform.InternalMessage{
		ID:           msg.ID,
		Title:        msg.Title,
		Body:         msg.Body,
		Format:       msg.Format,
		Priority:     msg.Priority,
		Metadata:     msg.Metadata,
		Variables:    msg.Variables,
		PlatformData: msg.PlatformData,
	}
}

func convertToInternalTargets(targets []platformPkg.Target) []platform.InternalTarget {
	internalTargets := make([]platform.InternalTarget, len(targets))
	for i, t := range targets {
		internalTargets[i] = convertToInternalTarget(t)
	}
	return internalTargets
}

func convertToInternalTarget(target platformPkg.Target) platform.InternalTarget {
	return platform.InternalTarget{
		Type:     target.Type,
		Value:    target.Value,
		Platform: target.Platform,
	}
}

func convertFromInternalResults(internalResults []*platform.SendResult) []*platformPkg.SendResult {
	results := make([]*platformPkg.SendResult, len(internalResults))
	for i, r := range internalResults {
		results[i] = &platformPkg.SendResult{
			Target: platformPkg.Target{
				Type:     r.Target.Type,
				Value:    r.Target.Value,
				Platform: r.Target.Platform,
			},
			Success:   r.Success,
			MessageID: r.MessageID,
			Error:     r.Error,
			Response:  r.Response,
			Metadata: map[string]interface{}{
				"duration": r.Duration.Milliseconds(),
			},
		}
	}
	return results
}

func convertFromInternalCapabilities(internalCaps platform.PlatformCapabilities) platformPkg.Capabilities {
	return platformPkg.Capabilities{
		Name:                 internalCaps.Name,
		SupportedTargetTypes: internalCaps.SupportedTargetTypes,
		SupportedFormats:     internalCaps.SupportedFormats,
		MaxMessageSize:       internalCaps.MaxMessageSize,
		SupportsScheduling:   internalCaps.SupportsScheduling,
		SupportsAttachments:  internalCaps.SupportsAttachments,
		SupportsMentions:     internalCaps.SupportsMentions,
		SupportsRichContent:  internalCaps.SupportsRichContent,
		RequiredSettings:     internalCaps.RequiredSettings,
	}
}
