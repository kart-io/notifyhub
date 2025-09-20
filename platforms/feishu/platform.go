package feishu

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/platforms"
)

// FeishuPlatform implements the Platform interface for Feishu
type FeishuPlatform struct {
	*platforms.BasePlatform
}

// NewFeishuPlatform creates a new Feishu platform implementation
func NewFeishuPlatform() *FeishuPlatform {
	capabilities := platforms.NewBaseCapabilities().
		WithFormats(platforms.FormatText, platforms.FormatMarkdown, platforms.FormatCard).
		WithFeatures(
			platforms.FeatureText,
			platforms.FeatureMarkdown,
			platforms.FeatureCard,
			platforms.FeatureMentions,
			platforms.FeatureBatch,
			platforms.FeatureRetry,
		).
		WithTargetTypes("group", "webhook", "user").
		WithMaxMessageSize(1024 * 100). // 100KB
		WithMaxTitleLength(256).
		WithMaxBodyLength(30000).
		WithRateLimits(platforms.RateLimitInfo{
			RequestsPerSecond: 20,
			BurstSize:         50,
			Window:            time.Second,
			Enforced:          true,
		})

	return &FeishuPlatform{
		BasePlatform: platforms.NewBasePlatform(
			"feishu",
			"Feishu (Lark) messaging platform for team collaboration",
			capabilities,
		),
	}
}

// ValidateConfig validates Feishu configuration
func (p *FeishuPlatform) ValidateConfig(config interface{}) error {
	switch cfg := config.(type) {
	case map[string]interface{}:
		if webhook, ok := cfg["webhook_url"].(string); !ok || webhook == "" {
			return fmt.Errorf("webhook_url is required")
		}
		return nil
	default:
		// Accept any config type for now
		return nil
	}
}

// CreateTransport creates a Feishu transport instance
func (p *FeishuPlatform) CreateTransport(config interface{}) (platforms.Transport, error) {
	var feishuConfig map[string]interface{}

	switch cfg := config.(type) {
	case map[string]interface{}:
		feishuConfig = cfg
	default:
		// If it's already a specific config type, try to use it as-is
		feishuConfig = map[string]interface{}{
			"config": cfg,
		}
	}

	return &FeishuTransportAdapter{config: feishuConfig}, nil
}

// DefaultConfig returns default configuration for Feishu
func (p *FeishuPlatform) DefaultConfig() interface{} {
	return map[string]interface{}{
		"webhook_url": "",
		"secret":      "",
		"timeout":     30 * time.Second,
	}
}

// FeishuTransportAdapter adapts the feishu transport to the Platform Transport interface
type FeishuTransportAdapter struct {
	config interface{}
}

// Send sends a message through Feishu
func (t *FeishuTransportAdapter) Send(ctx context.Context, msg interface{}, target interface{}) (interface{}, error) {
	// This would be implemented by the actual transport layer
	// For now, return a placeholder
	return nil, fmt.Errorf("feishu transport not fully implemented")
}

// Name returns the transport name
func (t *FeishuTransportAdapter) Name() string {
	return "feishu"
}

// Shutdown shuts down the transport
func (t *FeishuTransportAdapter) Shutdown() error {
	return nil
}

// Health checks the health of the transport
func (t *FeishuTransportAdapter) Health(ctx context.Context) error {
	// Implement health check logic
	// Could ping the webhook URL or check connection
	return nil
}
