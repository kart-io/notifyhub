// Package dingtalk demonstrates how to implement an external platform for NotifyHub
package dingtalk

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// DingtalkSender implements the external platform interface for DingTalk
type DingtalkSender struct {
	name       string
	webhookURL string
	secret     string
}

// NewDingtalkSender creates a new DingTalk sender
func NewDingtalkSender(config map[string]interface{}) (platform.ExternalSender, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required for DingTalk platform")
	}

	secret, _ := config["secret"].(string)

	return &DingtalkSender{
		name:       "dingtalk",
		webhookURL: webhookURL,
		secret:     secret,
	}, nil
}

// Name returns the platform name
func (d *DingtalkSender) Name() string {
	return d.name
}

// Send sends a message to DingTalk
func (d *DingtalkSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
	fmt.Printf("🔄 DingTalk Send: %s - %s\n", msg.Title, msg.Body)

	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()

		// Simulate sending to DingTalk
		result := &platform.SendResult{
			Target:    target,
			Success:   true,
			MessageID: fmt.Sprintf("dingtalk_%s_%d", msg.ID, i),
			Response:  "DingTalk message sent successfully",
			Metadata: map[string]interface{}{
				"webhook_url": d.webhookURL,
				"duration":    time.Since(startTime).Milliseconds(),
			},
		}

		// Simulate potential errors for specific targets
		if target.Value == "error_group" {
			result.Success = false
			result.Error = "DingTalk group not found"
		}

		results[i] = result

		fmt.Printf("  ✅ DingTalk -> %s (%v)\n", target.Value, result.Success)
	}

	return results, nil
}

// ValidateTarget validates a DingTalk target
func (d *DingtalkSender) ValidateTarget(target platform.Target) error {
	if target.Type != "group" && target.Type != "user" {
		return fmt.Errorf("dingtalk platform only supports 'group' and 'user' targets, got: %s", target.Type)
	}

	if target.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}

	return nil
}

// GetCapabilities returns DingTalk platform capabilities
func (d *DingtalkSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "dingtalk",
		SupportedTargetTypes: []string{"group", "user"},
		SupportedFormats:     []string{"text", "markdown"},
		MaxMessageSize:       20 * 1024, // 20KB limit
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		SupportsMentions:     true,
		SupportsRichContent:  true,
		RequiredSettings:     []string{"webhook_url"},
	}
}

// IsHealthy checks if DingTalk sender is healthy
func (d *DingtalkSender) IsHealthy(ctx context.Context) error {
	// Simulate health check
	if d.webhookURL == "" {
		return fmt.Errorf("dingtalk webhook URL not configured")
	}

	// In a real implementation, you might ping the DingTalk API
	fmt.Println("🔍 DingTalk health check: OK")
	return nil
}

// Close cleans up DingTalk sender resources
func (d *DingtalkSender) Close() error {
	fmt.Println("🔄 DingTalk sender closed")
	return nil
}

// RegisterDingtalkPlatform registers DingTalk as an external platform
// This function should be called during application initialization
func RegisterDingtalkPlatform() {
	fmt.Println("📝 Registering DingTalk platform...")
	platform.RegisterPlatform("dingtalk", NewDingtalkSender)
	fmt.Println("✅ DingTalk platform registered successfully")
}