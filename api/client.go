package api

import (
	"context"
	"fmt"

	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/core/hub"
	"github.com/kart-io/notifyhub/core/message"
)

// Client is the main entry point for the unified NotifyHub API
type Client struct {
	hub    *hub.Hub
	config *config.Config
}

// New creates a new NotifyHub client with unified API
func New(cfg *config.Config) (*Client, error) {
	// Simplified validation - use main config validation in production
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// TODO: Configure hub with transports from config
	coreHub := hub.NewHub(nil)

	return &Client{
		hub:    coreHub,
		config: cfg,
	}, nil
}

// Send creates a new message sending builder
func (c *Client) Send() *BaseBuilder {
	return NewBaseBuilder(c)
}

// Alert creates a high-priority alert message
func (c *Client) Alert() *BaseBuilder {
	return c.Send().Priority(message.PriorityHigh)
}

// Notification creates a normal priority notification
func (c *Client) Notification() *BaseBuilder {
	return c.Send().Priority(message.PriorityNormal)
}

// Email returns an email-specific builder with enhanced type safety
func (c *Client) Email() *UnifiedPlatformBuilder {
	return NewUnifiedPlatformBuilder(c, PlatformEmail)
}

// Feishu returns a Feishu-specific builder with enhanced type safety
func (c *Client) Feishu() *UnifiedPlatformBuilder {
	return NewUnifiedPlatformBuilder(c, PlatformFeishu)
}

// Slack returns a Slack-specific builder with enhanced type safety
func (c *Client) Slack() *UnifiedPlatformBuilder {
	return NewUnifiedPlatformBuilder(c, PlatformSlack)
}

// Platform returns a unified platform builder (new unified approach)
func (c *Client) Platform(platform PlatformType) *UnifiedPlatformBuilder {
	return NewUnifiedPlatformBuilder(c, platform)
}

// Shutdown gracefully shuts down the client
func (c *Client) Shutdown(ctx context.Context) error {
	return c.hub.Shutdown(ctx)
}

// Health returns the current health status
// func (c *Client) Health() *hub.HealthStatus {
// 	return c.hub.Health()
// }

// TODO: Add Metrics() method
// TODO: Add RegisterTransport() method

