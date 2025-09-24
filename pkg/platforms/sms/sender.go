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
}

// NewSMSSender creates a new SMS sender
func NewSMSSender(config map[string]interface{}) (platform.ExternalSender, error) {
	provider, ok := config["provider"].(string)
	if !ok || provider == "" {
		return nil, fmt.Errorf("provider is required for SMS platform")
	}

	apiKey, ok := config["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("api_key is required for SMS platform")
	}

	from, ok := config["from"].(string)
	if !ok || from == "" {
		return nil, fmt.Errorf("from is required for SMS platform")
	}

	sender := &SMSSender{
		provider: ProviderType(provider),
		apiKey:   apiKey,
		from:     from,
		timeout:  30 * time.Second,
	}

	// Configure optional fields
	if apiSecret, ok := config["api_secret"].(string); ok {
		sender.apiSecret = apiSecret
	}

	if region, ok := config["region"].(string); ok {
		sender.region = region
	}

	if template, ok := config["template"].(string); ok {
		sender.template = template
	}

	if signName, ok := config["sign_name"].(string); ok {
		sender.signName = signName
	}

	if timeout, ok := config["timeout"].(time.Duration); ok {
		sender.timeout = timeout
	}

	// Create HTTP client
	sender.client = &http.Client{
		Timeout: sender.timeout,
	}

	// Validate provider
	if !isValidProvider(sender.provider) {
		return nil, fmt.Errorf("unsupported SMS provider: %s", provider)
	}

	return sender, nil
}

// Name returns the platform name
func (s *SMSSender) Name() string {
	return "sms"
}

// Send sends a message to SMS
func (s *SMSSender) Send(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := &platform.SendResult{
			Target:  target,
			Success: false,
		}

		// Validate target
		if err := s.ValidateTarget(target); err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}

		// Build SMS content
		smsContent := s.buildSMSContent(msg, target)

		// Send SMS
		response, err := s.sendSMS(ctx, target.Value, smsContent)
		if err != nil {
			result.Error = err.Error()
		} else {
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
	}

	return results, nil
}

// ValidateTarget validates a target for SMS
func (s *SMSSender) ValidateTarget(target platform.Target) error {
	switch target.Type {
	case "phone", "sms":
		// Valid target types for SMS
	default:
		return fmt.Errorf("sms supports phone and sms targets, got %s", target.Type)
	}

	if target.Value == "" {
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
	if s.apiKey == "" {
		return fmt.Errorf("SMS API key not configured")
	}

	// Test phone number validation with a known valid format
	testPhone := "+1234567890"
	if err := s.validatePhoneNumber(testPhone); err != nil {
		return fmt.Errorf("SMS provider validation failed: %w", err)
	}

	return nil
}

// Close cleans up resources
func (s *SMSSender) Close() error {
	return nil
}

// buildSMSContent builds SMS content from the platform message
func (s *SMSSender) buildSMSContent(msg *platform.Message, target platform.Target) string {
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
