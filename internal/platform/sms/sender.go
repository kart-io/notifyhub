// Package sms implements the SMS Sender for SMS notifications through various providers
// This integrates all SMS-specific logic including provider integration, message formatting, and phone number validation
package sms

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/internal/platform"
)

// SMSSender implements the platform.Sender interface for SMS notifications
type SMSSender struct {
	name     string
	config   SMSConfig
	provider SMSProvider
	client   *http.Client
}

// SMSConfig holds SMS configuration
type SMSConfig struct {
	Provider  string        `json:"provider"` // "twilio", "aliyun", "tencent", "aws_sns"
	APIKey    string        `json:"api_key"`
	APISecret string        `json:"api_secret,omitempty"`
	Region    string        `json:"region,omitempty"`
	From      string        `json:"from"` // Sender phone number or ID
	Timeout   time.Duration `json:"timeout"`
	Template  string        `json:"template,omitempty"`  // SMS template ID
	SignName  string        `json:"sign_name,omitempty"` // SMS signature name
}

// SMSProvider interface for different SMS service providers
type SMSProvider interface {
	SendSMS(ctx context.Context, to, message string, config SMSConfig) (*SMSResponse, error)
	ValidatePhoneNumber(phone string) error
	GetProviderName() string
}

// SMSResponse represents SMS API response
type SMSResponse struct {
	MessageID   string `json:"message_id"`
	Status      string `json:"status"`
	StatusCode  int    `json:"status_code"`
	Description string `json:"description,omitempty"`
	Cost        string `json:"cost,omitempty"`
}

// NewSMSSender creates a new SMS sender
func NewSMSSender(config map[string]interface{}) (platform.Sender, error) {
	smsConfig, err := parseSMSConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid sms configuration: %w", err)
	}

	// Create provider based on configuration
	provider, err := createSMSProvider(smsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMS provider: %w", err)
	}

	return &SMSSender{
		name:     "sms",
		config:   smsConfig,
		provider: provider,
		client: &http.Client{
			Timeout: smsConfig.Timeout,
		},
	}, nil
}

// Name returns the platform name
func (s *SMSSender) Name() string {
	return s.name
}

// Send sends an SMS message to the specified targets
func (s *SMSSender) Send(ctx context.Context, msg *platform.InternalMessage, targets []platform.InternalTarget) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		startTime := time.Now()
		result := platform.NewSendResult(target, false)

		// Validate target
		if err := s.ValidateTarget(target); err != nil {
			result.Error = err.Error()
			result.SetDuration(time.Since(startTime))
			results[i] = result
			continue
		}

		// Build SMS content
		smsContent, err := s.buildSMSContent(msg, target)
		if err != nil {
			result.Error = fmt.Sprintf("failed to build sms content: %v", err)
			result.SetDuration(time.Since(startTime))
			results[i] = result
			continue
		}

		// Send SMS
		response, err := s.provider.SendSMS(ctx, target.Value, smsContent, s.config)
		if err != nil {
			result.Error = fmt.Sprintf("failed to send sms: %v", err)
		} else if response.StatusCode >= 400 {
			result.Error = fmt.Sprintf("sms provider error: %s (code: %d)", response.Description, response.StatusCode)
		} else {
			result.Success = true
			result.MessageID = response.MessageID
			result.StatusCode = response.StatusCode
			result.Response = response.Description
		}

		result.SetDuration(time.Since(startTime))
		results[i] = result
	}

	return results, nil
}

// ValidateTarget validates an SMS target
func (s *SMSSender) ValidateTarget(target platform.InternalTarget) error {
	if target.Type != "phone" {
		return fmt.Errorf("sms sender only supports phone targets, got %s", target.Type)
	}

	if target.Value == "" {
		return fmt.Errorf("phone number cannot be empty")
	}

	// Use provider-specific validation
	return s.provider.ValidatePhoneNumber(target.Value)
}

// GetCapabilities returns the SMS platform capabilities
func (s *SMSSender) GetCapabilities() platform.PlatformCapabilities {
	return platform.PlatformCapabilities{
		Name:                 "sms",
		SupportedTargetTypes: []string{"phone"},
		SupportedFormats:     []string{"text"},
		MaxMessageSize:       160, // Standard SMS length
		SupportsScheduling:   false,
		SupportsAttachments:  false,
		SupportsMentions:     false,
		SupportsRichContent:  false,
		RequiredSettings:     []string{"provider", "api_key", "from"},
		RateLimits: &platform.RateLimitInfo{
			RequestsPerSecond: 10,
			RequestsPerMinute: 100,
			BurstSize:         20,
		},
	}
}

// IsHealthy checks if the SMS sender is healthy
func (s *SMSSender) IsHealthy(ctx context.Context) error {
	// For SMS, we can check if the provider is configured correctly
	// without actually sending a test message
	if s.provider == nil {
		return fmt.Errorf("sms provider not initialized")
	}

	if s.config.APIKey == "" {
		return fmt.Errorf("sms api key not configured")
	}

	// Test phone number validation with a known valid format
	testPhone := "+1234567890"
	if err := s.provider.ValidatePhoneNumber(testPhone); err != nil {
		return fmt.Errorf("sms provider validation failed: %w", err)
	}

	return nil
}

// Close cleans up the SMS sender resources
func (s *SMSSender) Close() error {
	// SMS sender doesn't hold persistent connections
	return nil
}

// buildSMSContent builds the SMS content from the internal message
func (s *SMSSender) buildSMSContent(msg *platform.InternalMessage, target platform.InternalTarget) (string, error) {
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
	if s.config.SignName != "" {
		content = fmt.Sprintf("【%s】%s", s.config.SignName, content)
	}

	// Ensure content doesn't exceed SMS limits
	if len(content) > 160 {
		// Truncate to fit SMS limit
		content = content[:157] + "..."
	}

	return content, nil
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

// parseSMSConfig parses SMS configuration from map
func parseSMSConfig(config map[string]interface{}) (SMSConfig, error) {
	smsConfig := SMSConfig{
		Timeout: 30 * time.Second, // Default timeout
	}

	// Required fields
	if provider, ok := config["provider"].(string); ok {
		smsConfig.Provider = provider
	} else {
		return smsConfig, fmt.Errorf("provider is required")
	}

	if apiKey, ok := config["api_key"].(string); ok {
		smsConfig.APIKey = apiKey
	} else {
		return smsConfig, fmt.Errorf("api_key is required")
	}

	if from, ok := config["from"].(string); ok {
		smsConfig.From = from
	} else {
		return smsConfig, fmt.Errorf("from is required")
	}

	// Optional fields
	if apiSecret, ok := config["api_secret"].(string); ok {
		smsConfig.APISecret = apiSecret
	}

	if region, ok := config["region"].(string); ok {
		smsConfig.Region = region
	}

	if template, ok := config["template"].(string); ok {
		smsConfig.Template = template
	}

	if signName, ok := config["sign_name"].(string); ok {
		smsConfig.SignName = signName
	}

	if timeout, ok := config["timeout"].(time.Duration); ok {
		smsConfig.Timeout = timeout
	}

	return smsConfig, nil
}

// createSMSProvider creates the appropriate SMS provider based on configuration
func createSMSProvider(config SMSConfig) (SMSProvider, error) {
	switch strings.ToLower(config.Provider) {
	case "twilio":
		return newTwilioProvider(), nil
	case "aliyun", "alibaba":
		return newAliyunProvider(), nil
	case "tencent":
		return newTencentProvider(), nil
	case "aws_sns", "aws":
		return newAWSSNSProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported SMS provider: %s", config.Provider)
	}
}

// Basic SMS provider implementations

// TwilioProvider implements Twilio SMS service
type TwilioProvider struct{}

func newTwilioProvider() *TwilioProvider {
	return &TwilioProvider{}
}

func (p *TwilioProvider) GetProviderName() string {
	return "twilio"
}

func (p *TwilioProvider) ValidatePhoneNumber(phone string) error {
	// Twilio accepts E.164 format: +[country code][number]
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid phone number format for Twilio: %s (expected E.164 format like +1234567890)", phone)
	}
	return nil
}

func (p *TwilioProvider) SendSMS(ctx context.Context, to, message string, config SMSConfig) (*SMSResponse, error) {
	// TODO: Implement actual Twilio API call
	// This is a placeholder implementation
	return &SMSResponse{
		MessageID:   fmt.Sprintf("twilio_%d", time.Now().UnixNano()),
		Status:      "sent",
		StatusCode:  200,
		Description: "Message sent successfully",
	}, nil
}

// AliyunProvider implements Alibaba Cloud SMS service
type AliyunProvider struct{}

func newAliyunProvider() *AliyunProvider {
	return &AliyunProvider{}
}

func (p *AliyunProvider) GetProviderName() string {
	return "aliyun"
}

func (p *AliyunProvider) ValidatePhoneNumber(phone string) error {
	// Aliyun supports various formats, but prefer international format
	phoneRegex := regexp.MustCompile(`^(\+86)?1[3-9]\d{9}$`)
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid phone number format for Aliyun: %s", phone)
	}
	return nil
}

func (p *AliyunProvider) SendSMS(ctx context.Context, to, message string, config SMSConfig) (*SMSResponse, error) {
	// TODO: Implement actual Aliyun SMS API call
	return &SMSResponse{
		MessageID:   fmt.Sprintf("aliyun_%d", time.Now().UnixNano()),
		Status:      "sent",
		StatusCode:  200,
		Description: "Message sent successfully",
	}, nil
}

// TencentProvider implements Tencent Cloud SMS service
type TencentProvider struct{}

func newTencentProvider() *TencentProvider {
	return &TencentProvider{}
}

func (p *TencentProvider) GetProviderName() string {
	return "tencent"
}

func (p *TencentProvider) ValidatePhoneNumber(phone string) error {
	// Tencent SMS supports international format
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid phone number format for Tencent: %s", phone)
	}
	return nil
}

func (p *TencentProvider) SendSMS(ctx context.Context, to, message string, config SMSConfig) (*SMSResponse, error) {
	// TODO: Implement actual Tencent SMS API call
	return &SMSResponse{
		MessageID:   fmt.Sprintf("tencent_%d", time.Now().UnixNano()),
		Status:      "sent",
		StatusCode:  200,
		Description: "Message sent successfully",
	}, nil
}

// AWSSNSProvider implements AWS SNS SMS service
type AWSSNSProvider struct{}

func newAWSSNSProvider() *AWSSNSProvider {
	return &AWSSNSProvider{}
}

func (p *AWSSNSProvider) GetProviderName() string {
	return "aws_sns"
}

func (p *AWSSNSProvider) ValidatePhoneNumber(phone string) error {
	// AWS SNS requires E.164 format
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid phone number format for AWS SNS: %s (expected E.164 format)", phone)
	}
	return nil
}

func (p *AWSSNSProvider) SendSMS(ctx context.Context, to, message string, config SMSConfig) (*SMSResponse, error) {
	// TODO: Implement actual AWS SNS API call
	return &SMSResponse{
		MessageID:   fmt.Sprintf("aws_sns_%d", time.Now().UnixNano()),
		Status:      "sent",
		StatusCode:  200,
		Description: "Message sent successfully",
	}, nil
}
