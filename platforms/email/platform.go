package email

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/platforms"
)

// EmailPlatform implements the Platform interface for Email
type EmailPlatform struct {
	*platforms.BasePlatform
}

// NewEmailPlatform creates a new Email platform implementation
func NewEmailPlatform() *EmailPlatform {
	capabilities := platforms.NewBaseCapabilities().
		WithFormats(platforms.FormatText, platforms.FormatHTML, platforms.FormatMarkdown).
		WithFeatures(
			platforms.FeatureText,
			platforms.FeatureHTML,
			platforms.FeatureMarkdown,
			platforms.FeatureAttachments,
			platforms.FeatureTemplate,
			platforms.FeatureBatch,
			platforms.FeatureRetry,
			platforms.FeatureScheduling,
		).
		WithTargetTypes("email", "user", "group").
		WithMaxMessageSize(1024 * 1024 * 10). // 10MB with attachments
		WithMaxTitleLength(998).              // RFC 2822 recommends 998 chars max
		WithMaxBodyLength(1024 * 1024).       // 1MB for body
		WithRateLimits(platforms.RateLimitInfo{
			RequestsPerSecond: 50,
			BurstSize:         100,
			Window:            time.Minute,
			Enforced:          false, // Depends on SMTP server
		})

	return &EmailPlatform{
		BasePlatform: platforms.NewBasePlatform(
			"email",
			"Email platform for sending notifications via SMTP",
			capabilities,
		),
	}
}

// ValidateConfig validates Email configuration
func (p *EmailPlatform) ValidateConfig(config interface{}) error {
	switch cfg := config.(type) {
	case map[string]interface{}:
		if host, ok := cfg["host"].(string); !ok || host == "" {
			return fmt.Errorf("host is required")
		}
		if port, ok := cfg["port"].(int); !ok || port == 0 {
			return fmt.Errorf("valid port is required")
		}
		if from, ok := cfg["from"].(string); !ok || from == "" {
			return fmt.Errorf("from address is required")
		}
		return nil
	default:
		// Accept any config type for now
		return nil
	}
}

// CreateTransport creates an Email transport instance
func (p *EmailPlatform) CreateTransport(config interface{}) (platforms.Transport, error) {
	var emailConfig map[string]interface{}

	switch cfg := config.(type) {
	case map[string]interface{}:
		emailConfig = cfg
	default:
		// If it's already a specific config type, try to use it as-is
		emailConfig = map[string]interface{}{
			"config": cfg,
		}
	}

	return &EmailTransportAdapter{config: emailConfig}, nil
}

// DefaultConfig returns default configuration for Email
func (p *EmailPlatform) DefaultConfig() interface{} {
	return map[string]interface{}{
		"host":     "",
		"port":     587,
		"username": "",
		"password": "",
		"from":     "",
		"use_tls":  true,
		"timeout":  30 * time.Second,
	}
}

// EmailTransportAdapter adapts the email transport to the Platform Transport interface
type EmailTransportAdapter struct {
	config interface{}
}

// Send sends a message through Email
func (t *EmailTransportAdapter) Send(ctx context.Context, msg interface{}, target interface{}) (interface{}, error) {
	// This would be implemented by the actual transport layer
	// For now, return a placeholder
	return nil, fmt.Errorf("email transport not fully implemented")
}

// Name returns the transport name
func (t *EmailTransportAdapter) Name() string {
	return "email"
}

// Shutdown shuts down the transport
func (t *EmailTransportAdapter) Shutdown() error {
	return nil
}

// Health checks the health of the transport
func (t *EmailTransportAdapter) Health(ctx context.Context) error {
	// Implement health check logic
	// Could test SMTP connection
	return nil
}
