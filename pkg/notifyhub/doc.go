// Package notifyhub provides a unified notification system with support for multiple platforms.
//
// NotifyHub is a modular, extensible notification system that allows sending messages
// through various platforms using a single, unified API. It supports both internal
// platforms (like Feishu, Email, SMS) and external platform extensions.
//
// # Quick Start
//
//	import (
//	    "context"
//	    "github.com/kart-io/notifyhub/pkg/notifyhub"
//	    "github.com/kart-io/notifyhub/pkg/notifyhub/config"
//	    "github.com/kart-io/notifyhub/pkg/notifyhub/message"
//	    "github.com/kart-io/notifyhub/pkg/notifyhub/target"
//	)
//
//	func main() {
//	    // Create Hub configuration
//	    cfg := &config.HubConfig{
//	        Platforms: map[string]config.PlatformConfig{
//	            "feishu": {
//	                "webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook",
//	                "secret":      "your-secret",
//	            },
//	        },
//	    }
//
//	    // Create Hub instance
//	    hub, err := notifyhub.NewHub(cfg)
//	    if err != nil {
//	        panic(err)
//	    }
//	    defer hub.Close(context.Background())
//
//	    // Create and send message
//	    msg := &message.Message{
//	        ID:    "msg-001",
//	        Title: "Test Notification",
//	        Body:  "This is a test message",
//	        Targets: []target.Target{
//	            {Type: "webhook", Value: "feishu-bot", Platform: "feishu"},
//	        },
//	    }
//
//	    receipt, err := hub.Send(context.Background(), msg)
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    fmt.Printf("Status: %s, Success: %d, Failed: %d\n",
//	        receipt.Status, receipt.Successful, receipt.Failed)
//	}
//
// # Architecture
//
// The NotifyHub system is built with a modular, plugin-based architecture:
//
//   - Hub: Central coordinator that orchestrates message sending
//   - Platforms: Pluggable notification providers (Feishu, Email, SMS, etc.)
//   - Registry: Platform registration and discovery mechanism
//   - Adapters: Bridge between internal implementations and public APIs
//   - Messages: Unified message structure with platform-agnostic targets
//
// # Platform Extension
//
// You can extend NotifyHub with custom platforms by implementing the
// platform.ExternalSender interface:
//
//	type CustomSender struct{}
//
//	func (c *CustomSender) Name() string { return "custom" }
//
//	func (c *CustomSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
//	    // Implementation here
//	    return results, nil
//	}
//
//	func (c *CustomSender) ValidateTarget(target platform.Target) error { return nil }
//	func (c *CustomSender) GetCapabilities() platform.Capabilities { return capabilities }
//	func (c *CustomSender) IsHealthy(ctx context.Context) error { return nil }
//	func (c *CustomSender) Close() error { return nil }
//
//	func init() {
//	    platform.RegisterPlatform("custom", func(config map[string]interface{}) (platform.ExternalSender, error) {
//	        return &CustomSender{}, nil
//	    })
//	}
//
// # Error Handling
//
// NotifyHub provides detailed error information and status tracking:
//
//	receipt, err := hub.Send(ctx, msg)
//	if err != nil {
//	    // Handle send errors
//	    return err
//	}
//
//	// Check for partial failures
//	if receipt.Status == "partial" {
//	    for _, result := range receipt.Results {
//	        if !result.Success {
//	            log.Printf("Platform %s failed: %s", result.Platform, result.Error)
//	        }
//	    }
//	}
//
// # Health Monitoring
//
// Built-in health checking for all registered platforms:
//
//	health, err := hub.Health(context.Background())
//	if err != nil {
//	    return err
//	}
//
//	for platform, status := range health.Platforms {
//	    if !status.Available {
//	        log.Printf("Platform %s is unhealthy: %s", platform, status.Status)
//	    }
//	}
package notifyhub