// Package sms provides SMS platform implementation for NotifyHub
// This demonstrates how to implement an external SMS platform with multiple providers
package sms

import (
	"context"
	"fmt"
	"strings"

	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/platform"
	"github.com/kart-io/notifyhub/pkg/target"
)

// Provider represents different SMS service providers
type Provider string

const (
	ProviderAliyun  Provider = "aliyun"  // 阿里云短信
	ProviderTencent Provider = "tencent" // 腾讯云短信
	ProviderTwilio  Provider = "twilio"  // Twilio
	ProviderNexmo   Provider = "nexmo"   // Vonage (Nexmo)
	ProviderMock    Provider = "mock"    // 模拟测试
)

// Config holds SMS platform configuration
type Config struct {
	Provider    Provider          `json:"provider"`    // 短信服务提供商
	Credentials map[string]string `json:"credentials"` // 认证信息
	Templates   map[string]string `json:"templates"`   // 短信模板
	RateLimit   RateLimitConfig   `json:"rate_limit"`  // 限流配置
	Timeout     int               `json:"timeout"`     // 超时时间（秒）
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled    bool `json:"enabled"`      // 是否启用限流
	MaxPerHour int  `json:"max_per_hour"` // 每小时最大发送数
	MaxPerDay  int  `json:"max_per_day"`  // 每天最大发送数
}

// Platform implements the Platform interface for SMS
type Platform struct {
	config   Config
	provider SMSProvider
	limiter  *RateLimiter
}

// SMSProvider defines the interface for different SMS service providers
type SMSProvider interface {
	Name() string
	Send(ctx context.Context, phone, content string, templateID string) (*SMSResult, error)
	ValidateCredentials() error
	GetStatus() ProviderStatus
	Close() error
}

// SMSResult represents the result of SMS sending
type SMSResult struct {
	MessageID string            `json:"message_id"`
	Status    string            `json:"status"`
	Cost      float64           `json:"cost"`     // 费用
	Parts     int               `json:"parts"`    // 短信条数
	Metadata  map[string]string `json:"metadata"` // 额外信息
}

// ProviderStatus represents provider status
type ProviderStatus struct {
	Available bool              `json:"available"`
	Quota     QuotaInfo         `json:"quota"`
	Metadata  map[string]string `json:"metadata"`
}

// QuotaInfo represents quota information
type QuotaInfo struct {
	Remaining int `json:"remaining"` // 剩余条数
	Total     int `json:"total"`     // 总配额
	Reset     int `json:"reset"`     // 重置时间（Unix时间戳）
}

// New creates a new SMS platform instance
func New(config interface{}) (platform.Platform, error) {
	cfg, ok := config.(Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type for SMS platform")
	}

	// 验证配置
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid SMS config: %w", err)
	}

	// 创建短信提供商
	provider, err := createProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMS provider: %w", err)
	}

	// 创建限流器
	var limiter *RateLimiter
	if cfg.RateLimit.Enabled {
		limiter = NewRateLimiter(cfg.RateLimit)
	}

	return &Platform{
		config:   cfg,
		provider: provider,
		limiter:  limiter,
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "sms"
}

// GetCapabilities returns platform capabilities
func (p *Platform) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "sms",
		SupportedTargetTypes: []string{"phone", "mobile"},
		SupportedFormats:     []string{"text"},
		MaxMessageSize:       70, // 单条短信70个字符（中文）
		SupportsScheduling:   true,
		SupportsAttachments:  false,
		RequiredSettings:     []string{"provider", "credentials"},
	}
}

// ValidateTarget validates if the target is supported
func (p *Platform) ValidateTarget(target target.Target) error {
	if target.Type != "phone" && target.Type != "mobile" {
		return fmt.Errorf("unsupported target type: %s", target.Type)
	}

	// 验证手机号格式
	phone := target.Value
	if !isValidPhoneNumber(phone) {
		return fmt.Errorf("invalid phone number format: %s", phone)
	}

	return nil
}

// Send sends SMS to targets
func (p *Platform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	results := make([]*platform.SendResult, len(targets))

	for i, target := range targets {
		result := &platform.SendResult{
			Target: target,
		}

		// 验证目标
		if err := p.ValidateTarget(target); err != nil {
			result.Error = err
			results[i] = result
			continue
		}

		// 检查限流
		if p.limiter != nil {
			if !p.limiter.Allow(target.Value) {
				result.Error = fmt.Errorf("rate limit exceeded for %s", target.Value)
				results[i] = result
				continue
			}
		}

		// 准备短信内容
		content, templateID, err := p.prepareContent(msg, target)
		if err != nil {
			result.Error = fmt.Errorf("failed to prepare content: %w", err)
			results[i] = result
			continue
		}

		// 发送短信
		smsResult, err := p.provider.Send(ctx, target.Value, content, templateID)
		if err != nil {
			result.Error = err
			result.Response = err.Error()
		} else {
			result.Success = true
			result.MessageID = smsResult.MessageID
			result.Response = fmt.Sprintf("Status: %s, Parts: %d, Cost: %.4f",
				smsResult.Status, smsResult.Parts, smsResult.Cost)
		}

		results[i] = result
	}

	return results, nil
}

// prepareContent prepares SMS content from message
func (p *Platform) prepareContent(msg *message.Message, target target.Target) (string, string, error) {
	var content string
	var templateID string

	// 检查是否使用模板
	if templateName, exists := msg.Metadata["template"]; exists {
		if template, ok := p.config.Templates[templateName.(string)]; ok {
			templateID = templateName.(string)
			content = p.replaceVariables(template, msg.Variables)
		} else {
			return "", "", fmt.Errorf("template not found: %s", templateName)
		}
	} else {
		// 使用消息内容
		content = msg.Body
		if msg.Title != "" {
			content = fmt.Sprintf("%s: %s", msg.Title, msg.Body)
		}
	}

	// 检查内容长度
	if len([]rune(content)) > 70 {
		return "", "", fmt.Errorf("SMS content too long: %d characters (max 70)", len([]rune(content)))
	}

	return content, templateID, nil
}

// replaceVariables replaces variables in template
func (p *Platform) replaceVariables(template string, variables map[string]interface{}) string {
	content := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		content = strings.ReplaceAll(content, placeholder, fmt.Sprintf("%v", value))
	}
	return content
}

// IsHealthy checks if the platform is healthy
func (p *Platform) IsHealthy(ctx context.Context) error {
	// 检查提供商状态
	status := p.provider.GetStatus()
	if !status.Available {
		return fmt.Errorf("SMS provider %s is not available", p.provider.Name())
	}

	// 检查配额（跳过按用量计费的提供商）
	if status.Quota.Remaining <= 0 && status.Quota.Total > 0 {
		return fmt.Errorf("SMS quota exhausted: %d/%d remaining",
			status.Quota.Remaining, status.Quota.Total)
	}

	// 验证认证信息
	if err := p.provider.ValidateCredentials(); err != nil {
		return fmt.Errorf("SMS credentials validation failed: %w", err)
	}

	return nil
}

// Close cleans up platform resources
func (p *Platform) Close() error {
	if p.provider != nil {
		return p.provider.Close()
	}
	return nil
}

// validateConfig validates SMS configuration
func validateConfig(cfg Config) error {
	if cfg.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	if cfg.Credentials == nil || len(cfg.Credentials) == 0 {
		return fmt.Errorf("credentials are required")
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 // 默认30秒
	}

	return nil
}

// createProvider creates appropriate SMS provider
func createProvider(cfg Config) (SMSProvider, error) {
	switch cfg.Provider {
	case ProviderAliyun:
		return NewAliyunProvider(cfg.Credentials)
	case ProviderTencent:
		return NewTencentProvider(cfg.Credentials)
	case ProviderTwilio:
		return NewTwilioProvider(cfg.Credentials)
	case ProviderNexmo:
		return NewNexmoProvider(cfg.Credentials)
	case ProviderMock:
		return NewMockProvider(cfg.Credentials)
	default:
		return nil, fmt.Errorf("unsupported SMS provider: %s", cfg.Provider)
	}
}

// isValidPhoneNumber validates phone number format
func isValidPhoneNumber(phone string) bool {
	// 去除空格和连字符进行验证
	cleaned := strings.ReplaceAll(strings.ReplaceAll(phone, " ", ""), "-", "")

	// 简单的手机号验证，支持国际格式
	if len(cleaned) < 10 || len(cleaned) > 15 {
		return false
	}

	// 检查第一个字符是否为 + 或数字
	if len(cleaned) > 0 && cleaned[0] != '+' && (cleaned[0] < '0' || cleaned[0] > '9') {
		return false
	}

	// 检查其余字符是否为数字
	start := 0
	if len(cleaned) > 0 && cleaned[0] == '+' {
		start = 1
	}

	for i := start; i < len(cleaned); i++ {
		if cleaned[i] < '0' || cleaned[i] > '9' {
			return false
		}
	}

	return true
}

// CreateTarget creates an SMS target with phone number
func CreateTarget(phone string) target.Target {
	return target.Target{
		Type:     "phone",
		Value:    phone,
		Platform: "sms",
	}
}

// CreateMobileTarget creates a mobile target with phone number
func CreateMobileTarget(phone string) target.Target {
	return target.Target{
		Type:     "mobile",
		Value:    phone,
		Platform: "sms",
	}
}

// CreateTargetWithTemplate creates an SMS target with template
func CreateTargetWithTemplate(phone, templateName string) target.Target {
	return target.Target{
		Type:     "phone",
		Value:    phone,
		Platform: "sms",
	}
}
