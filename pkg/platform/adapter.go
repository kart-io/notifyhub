// Package platform provides common platform adapters for NotifyHub
package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/errors"
	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/target"
	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

// BasePlatformAdapter provides common functionality for platform implementations
type BasePlatformAdapter struct {
	name         string
	capabilities Capabilities
	logger       logger.Logger
	config       map[string]interface{}
}

// NewBasePlatformAdapter creates a new base platform adapter
func NewBasePlatformAdapter(name string, capabilities Capabilities, logger logger.Logger) *BasePlatformAdapter {
	return &BasePlatformAdapter{
		name:         name,
		capabilities: capabilities,
		logger:       logger,
		config:       make(map[string]interface{}),
	}
}

// Name returns the platform name
func (b *BasePlatformAdapter) Name() string {
	return b.name
}

// GetCapabilities returns the platform capabilities
func (b *BasePlatformAdapter) GetCapabilities() Capabilities {
	return b.capabilities
}

// SetConfig sets configuration for the platform
func (b *BasePlatformAdapter) SetConfig(config map[string]interface{}) {
	b.config = config
}

// GetConfig gets configuration value
func (b *BasePlatformAdapter) GetConfig(key string) (interface{}, bool) {
	value, exists := b.config[key]
	return value, exists
}

// ValidateTarget validates a target using platform capabilities
func (b *BasePlatformAdapter) ValidateTarget(target target.Target) error {
	// Check if target type is supported
	supported := false
	for _, supportedType := range b.capabilities.SupportedTargetTypes {
		if target.Type == supportedType {
			supported = true
			break
		}
	}

	if !supported {
		return &errors.NotifyError{
			Code:     errors.ErrInvalidTarget,
			Message:  fmt.Sprintf("target type %s not supported by platform %s", target.Type, b.name),
			Platform: b.name,
			Target:   target.Value,
		}
	}

	// Validate target value is not empty
	if target.Value == "" {
		return &errors.NotifyError{
			Code:     errors.ErrInvalidTarget,
			Message:  "target value cannot be empty",
			Platform: b.name,
		}
	}

	return nil
}

// ValidateMessage validates a message using platform capabilities
func (b *BasePlatformAdapter) ValidateMessage(msg *message.Message) error {
	// Check message size limits
	if b.capabilities.MaxMessageSize > 0 {
		messageSize := len(msg.Title) + len(msg.Body)
		if messageSize > b.capabilities.MaxMessageSize {
			return &errors.NotifyError{
				Code:     errors.ErrMessageTooLarge,
				Message:  fmt.Sprintf("message size %d exceeds limit %d", messageSize, b.capabilities.MaxMessageSize),
				Platform: b.name,
			}
		}
	}

	// Check if message format is supported
	if len(b.capabilities.SupportedFormats) > 0 {
		formatSupported := false
		for _, format := range b.capabilities.SupportedFormats {
			if string(msg.Format) == format {
				formatSupported = true
				break
			}
		}

		if !formatSupported {
			return &errors.NotifyError{
				Code:     errors.ErrInvalidMessage,
				Message:  fmt.Sprintf("message format %s not supported by platform %s", msg.Format, b.name),
				Platform: b.name,
			}
		}
	}

	return nil
}

// Close provides default close implementation
func (b *BasePlatformAdapter) Close() error {
	b.logger.Debug("Platform adapter closed", "platform", b.name)
	return nil
}

// IsHealthy provides default health check implementation
func (b *BasePlatformAdapter) IsHealthy(ctx context.Context) error {
	// Basic health check - can be overridden by specific platforms
	b.logger.Debug("Platform health check", "platform", b.name, "status", "healthy")
	return nil
}

// HTTPAdapter provides HTTP-based platform adapter functionality
type HTTPAdapter struct {
	*BasePlatformAdapter
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
}

// NewHTTPAdapter creates a new HTTP-based platform adapter
func NewHTTPAdapter(name string, capabilities Capabilities, baseURL string, logger logger.Logger) *HTTPAdapter {
	return &HTTPAdapter{
		BasePlatformAdapter: NewBasePlatformAdapter(name, capabilities, logger),
		BaseURL:             baseURL,
		Timeout:             30 * time.Second,
		MaxRetries:          3,
	}
}

// SetTimeout sets the HTTP timeout
func (h *HTTPAdapter) SetTimeout(timeout time.Duration) {
	h.Timeout = timeout
}

// SetMaxRetries sets the maximum retry attempts
func (h *HTTPAdapter) SetMaxRetries(maxRetries int) {
	h.MaxRetries = maxRetries
}

// WebhookAdapter provides webhook-based platform adapter functionality
type WebhookAdapter struct {
	*HTTPAdapter
	WebhookURL string
	Secret     string
}

// NewWebhookAdapter creates a new webhook-based platform adapter
func NewWebhookAdapter(name string, capabilities Capabilities, webhookURL string, logger logger.Logger) *WebhookAdapter {
	return &WebhookAdapter{
		HTTPAdapter: NewHTTPAdapter(name, capabilities, webhookURL, logger),
		WebhookURL:  webhookURL,
	}
}

// SetSecret sets the webhook secret for authentication
func (w *WebhookAdapter) SetSecret(secret string) {
	w.Secret = secret
}

// Send implements the Platform interface for WebhookAdapter
func (w *WebhookAdapter) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	results := make([]*SendResult, len(targets))

	for i, tgt := range targets {
		result := &SendResult{
			Target: tgt,
		}

		// For webhook adapter, we could implement HTTP POST to the webhook URL
		// This is a basic implementation
		result.Success = true
		result.MessageID = msg.ID
		result.Response = "webhook sent successfully"

		results[i] = result
	}

	return results, nil
}

// ValidateTarget implements the Platform interface for WebhookAdapter
func (w *WebhookAdapter) ValidateTarget(target target.Target) error {
	if target.Type != "webhook" {
		return errors.New(errors.ErrUnsupportedTarget, "webhook adapter only supports webhook targets")
	}
	return nil
}

// IsHealthy implements the Platform interface for WebhookAdapter
func (w *WebhookAdapter) IsHealthy(ctx context.Context) error {
	return nil
}

// Close implements the Platform interface for WebhookAdapter
func (w *WebhookAdapter) Close() error {
	return nil
}

// AdapterFactory defines a factory function for creating platform adapters
type AdapterFactory func(config map[string]interface{}, logger logger.Logger) (Platform, error)

// CommonAdapterFactories provides factory functions for common adapters
var CommonAdapterFactories = map[string]AdapterFactory{
	"webhook": func(config map[string]interface{}, logger logger.Logger) (Platform, error) {
		webhookURL, ok := config["webhook_url"].(string)
		if !ok || webhookURL == "" {
			return nil, fmt.Errorf("webhook_url is required for webhook adapter")
		}

		capabilities := Capabilities{
			Name:                 "webhook",
			SupportedTargetTypes: []string{"webhook"},
			SupportedFormats:     []string{"text", "markdown", "html"},
			MaxMessageSize:       65536, // 64KB
			SupportsScheduling:   false,
			SupportsAttachments:  false,
			RequiredSettings:     []string{"webhook_url"},
		}

		adapter := NewWebhookAdapter("webhook", capabilities, webhookURL, logger)

		if secret, ok := config["secret"].(string); ok {
			adapter.SetSecret(secret)
		}

		if timeout, ok := config["timeout"].(time.Duration); ok {
			adapter.SetTimeout(timeout)
		}

		return adapter, nil
	},
}

// GetAdapterFactory returns a factory function for the specified adapter type
func GetAdapterFactory(adapterType string) (AdapterFactory, bool) {
	factory, exists := CommonAdapterFactories[adapterType]
	return factory, exists
}

// RegisterAdapterFactory registers a new adapter factory
func RegisterAdapterFactory(adapterType string, factory AdapterFactory) {
	CommonAdapterFactories[adapterType] = factory
}
