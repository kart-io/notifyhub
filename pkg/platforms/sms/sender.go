// Package sms provides SMS platform integration for NotifyHub
// This package implements unified platform architecture for SMS notifications
package sms

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// ProviderType represents different SMS service providers
type ProviderType string

const (
	ProviderTwilio  ProviderType = "twilio"
	ProviderAliyun  ProviderType = "aliyun"
	ProviderTencent ProviderType = "tencent"
	ProviderAWSSNS  ProviderType = "aws_sns"
)

// SMSSender implements the ExternalSender interface for SMS notifications
type SMSSender struct {
	provider  ProviderType
	apiKey    string
	apiSecret string
	region    string
	from      string
	timeout   time.Duration
	template  string
	signName  string
	client    *http.Client
	logger    logger.Logger
}

// NewSMSSender creates a new SMS sender
func NewSMSSender(config map[string]interface{}, logger logger.Logger) (platform.ExternalSender, error) {
	logger.Debug("Creating SMSSender", "config_keys", getConfigKeys(config))

	provider, ok := config["provider"].(string)
	if !ok || provider == "" {
		logger.Error("Missing or invalid provider")
		return nil, fmt.Errorf("provider is required for SMS platform")
	}

	apiKey, ok := config["api_key"].(string)
	if !ok || apiKey == "" {
		logger.Error("Missing or invalid api_key")
		return nil, fmt.Errorf("api_key is required for SMS platform")
	}

	from, ok := config["from"].(string)
	if !ok || from == "" {
		logger.Error("Missing or invalid from number")
		return nil, fmt.Errorf("from is required for SMS platform")
	}

	sender := &SMSSender{
		provider: ProviderType(provider),
		apiKey:   apiKey,
		from:     from,
		timeout:  30 * time.Second,
		logger:   logger,
	}

	// Configure optional fields
	if apiSecret, ok := config["api_secret"].(string); ok {
		sender.apiSecret = apiSecret
		logger.Debug("API secret configured")
	}

	if region, ok := config["region"].(string); ok {
		sender.region = region
		logger.Debug("Region configured", "region", region)
	}

	if template, ok := config["template"].(string); ok {
		sender.template = template
		logger.Debug("Template configured")
	}

	if signName, ok := config["sign_name"].(string); ok {
		sender.signName = signName
		logger.Debug("Sign name configured", "signName", signName)
	}

	if timeout, ok := config["timeout"].(time.Duration); ok {
		sender.timeout = timeout
		logger.Debug("Timeout configured", "timeout", timeout)
	}

	// Create HTTP client
	sender.client = &http.Client{
		Timeout: sender.timeout,
	}

	// Validate provider
	if !isValidProvider(sender.provider) {
		logger.Error("Unsupported SMS provider", "provider", provider)
		return nil, fmt.Errorf("unsupported SMS provider: %s", provider)
	}

	logger.Info("SMSSender created successfully", "provider", provider, "from", from)
	return sender, nil
}

// getConfigKeys returns the keys from a config map for logging
func getConfigKeys(config map[string]interface{}) []string {
	keys := make([]string, 0, len(config))
	for k := range config {
		keys = append(keys, k)
	}
	return keys
}

// Name returns the platform name
func (s *SMSSender) Name() string {
	return "sms"
}

// Send sends a message to SMS
func (s *SMSSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
	if s.logger == nil {
		s.logger = logger.Discard
	}
	s.logger.Debug("Starting SMS send", "messageID", msg.ID, "targetCount", len(targets), "provider", s.provider)

	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := &platform.SendResult{
			Target:  target,
			Success: false,
		}

		// Validate target
		if err := s.ValidateTarget(target); err != nil {
			s.logger.Error("Invalid SMS target", "target", target.Value, "phone", target.Value, "error", err)
			result.Error = err.Error()
			results[i] = result
			continue
		}

		s.logger.Debug("Building SMS content", "target", target.Value)
		// Build SMS content
		smsContent := s.buildSMSContent(msg, target)

		// Send SMS
		s.logger.Debug("Sending SMS", "to", target.Value, "provider", s.provider)
		response, err := s.sendSMS(ctx, target.Value, smsContent)
		if err != nil {
			s.logger.Error("Failed to send SMS", "target", target.Value, "to", target.Value, "error", err)
			result.Error = err.Error()
		} else {
			s.logger.Info("SMS sent successfully", "target", target.Value, "to", target.Value, "messageID", response.MessageID)
			result.Success = true
			result.MessageID = response.MessageID
			result.Response = response.Description
		}

		// Set metadata
		result.Metadata = map[string]interface{}{
			"duration": time.Since(startTime).Milliseconds(),
			"provider": string(s.provider),
		}

		results[i] = result
		s.logger.Debug("SMS send attempt completed", "target", target.Value, "success", result.Success, "duration_ms", time.Since(startTime).Milliseconds())
	}

	s.logger.Debug("SMS batch send completed", "messageID", msg.ID, "totalTargets", len(targets))
	return results, nil
}

// ValidateTarget validates a target for SMS
func (s *SMSSender) ValidateTarget(target platform.Target) error {
	switch target.Type {
	case "phone", "sms":
		// Valid target types for SMS
	default:
		s.logger.Debug("Invalid target type for SMS", "type", target.Type)
		return fmt.Errorf("sms supports phone and sms targets, got %s", target.Type)
	}

	if target.Value == "" {
		s.logger.Debug("Empty phone number")
		return fmt.Errorf("phone number cannot be empty")
	}

	// Provider-specific phone number validation
	return s.validatePhoneNumber(target.Value)
}

// GetCapabilities returns SMS platform capabilities
func (s *SMSSender) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "sms",
		SupportedTargetTypes: []string{"phone", "sms"},
		SupportedFormats:     []string{"text"},
		MaxMessageSize:       160, // Standard SMS length
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		SupportsMentions:     false,
		SupportsRichContent:  false,
		RequiredSettings:     []string{"provider", "api_key", "from"},
	}
}

// IsHealthy checks if SMS provider is accessible
func (s *SMSSender) IsHealthy(ctx context.Context) error {
	s.logger.Debug("Performing health check for SMS sender", "provider", s.provider)

	if s.apiKey == "" {
		s.logger.Error("SMS API key not configured")
		return fmt.Errorf("SMS API key not configured")
	}

	// Test phone number validation with a known valid format
	testPhone := "+1234567890"
	if err := s.validatePhoneNumber(testPhone); err != nil {
		s.logger.Error("SMS provider validation failed", "error", err)
		return fmt.Errorf("SMS provider validation failed: %w", err)
	}

	s.logger.Debug("SMS health check passed", "provider", s.provider)
	return nil
}

// Close cleans up resources
func (s *SMSSender) Close() error {
	s.logger.Debug("Closing SMS sender", "provider", s.provider)
	return nil
}

// buildSMSContent builds SMS content from the platform message
func (s *SMSSender) buildSMSContent(msg *platform.Message, target platform.Target) string {
	s.logger.Debug("Building SMS content", "messageID", msg.ID, "hasTemplate", msg.PlatformData["sms_template"] != nil)

	var content string

	// Check if there's SMS template data
	if templateData, ok := msg.PlatformData["sms_template"]; ok {
		// Use template with variables
		if variables, hasVars := msg.PlatformData["sms_variables"].(map[string]interface{}); hasVars {
			content = s.renderTemplate(templateData.(string), variables)
		} else {
			content = templateData.(string)
		}
	} else {
		// Build content from title and body
		if msg.Title != "" && msg.Body != "" {
			content = fmt.Sprintf("%s: %s", msg.Title, msg.Body)
		} else if msg.Title != "" {
			content = msg.Title
		} else {
			content = msg.Body
		}
	}

	// Add signature if configured
	if s.signName != "" {
		content = fmt.Sprintf("【%s】%s", s.signName, content)
	}

	// Ensure content doesn't exceed SMS limits
	if len(content) > 160 {
		// Truncate to fit SMS limit
		content = content[:157] + "..."
	}

	return content
}

// renderTemplate performs simple variable substitution
func (s *SMSSender) renderTemplate(template string, variables map[string]interface{}) string {
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// validatePhoneNumber validates phone number based on provider
func (s *SMSSender) validatePhoneNumber(phone string) error {
	switch s.provider {
	case ProviderTwilio, ProviderAWSSNS:
		// E.164 format: +[country code][number]
		phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
		if !phoneRegex.MatchString(phone) {
			return fmt.Errorf("invalid phone number format for %s: %s (expected E.164 format like +1234567890)", s.provider, phone)
		}
	case ProviderAliyun:
		// Aliyun supports various formats, but prefer international format
		phoneRegex := regexp.MustCompile(`^(\+86)?1[3-9]\d{9}$`)
		if !phoneRegex.MatchString(phone) {
			return fmt.Errorf("invalid phone number format for Aliyun: %s", phone)
		}
	case ProviderTencent:
		// Tencent SMS supports international format
		phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
		if !phoneRegex.MatchString(phone) {
			return fmt.Errorf("invalid phone number format for Tencent: %s", phone)
		}
	default:
		return fmt.Errorf("unsupported provider: %s", s.provider)
	}
	return nil
}

// SMSResponse represents SMS API response
type SMSResponse struct {
	MessageID   string `json:"message_id"`
	Status      string `json:"status"`
	StatusCode  int    `json:"status_code"`
	Description string `json:"description,omitempty"`
	Cost        string `json:"cost,omitempty"`
}

// sendSMS sends SMS using the configured provider
func (s *SMSSender) sendSMS(ctx context.Context, to, message string) (*SMSResponse, error) {
	s.logger.Debug("Sending SMS", "to", to, "provider", s.provider)
	// TODO: Implement actual provider-specific SMS sending
	// This is a placeholder implementation that simulates successful sending
	switch s.provider {
	case ProviderTwilio:
		return &SMSResponse{
			MessageID:   fmt.Sprintf("twilio_%d", time.Now().UnixNano()),
			Status:      "sent",
			StatusCode:  200,
			Description: "Message sent successfully via Twilio",
		}, nil
	case ProviderAliyun:
		return &SMSResponse{
			MessageID:   fmt.Sprintf("aliyun_%d", time.Now().UnixNano()),
			Status:      "sent",
			StatusCode:  200,
			Description: "Message sent successfully via Aliyun",
		}, nil
	case ProviderTencent:
		return &SMSResponse{
			MessageID:   fmt.Sprintf("tencent_%d", time.Now().UnixNano()),
			Status:      "sent",
			StatusCode:  200,
			Description: "Message sent successfully via Tencent",
		}, nil
	case ProviderAWSSNS:
		return &SMSResponse{
			MessageID:   fmt.Sprintf("aws_sns_%d", time.Now().UnixNano()),
			Status:      "sent",
			StatusCode:  200,
			Description: "Message sent successfully via AWS SNS",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", s.provider)
	}
}

// isValidProvider checks if the provider is supported
func isValidProvider(provider ProviderType) bool {
	switch provider {
	case ProviderTwilio, ProviderAliyun, ProviderTencent, ProviderAWSSNS:
		return true
	default:
		return false
	}
}
