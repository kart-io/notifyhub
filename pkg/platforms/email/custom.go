// Package email provides custom email configuration and features for NotifyHub
package email

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/target"
	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

// CustomEmailConfig represents a custom email configuration
type CustomEmailConfig struct {
	// Basic SMTP settings
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	From        string `json:"from"`
	FromName    string `json:"from_name,omitempty"`

	// Security settings
	UseTLS         bool   `json:"use_tls"`
	UseStartTLS    bool   `json:"use_starttls"`
	SkipCertVerify bool   `json:"skip_cert_verify,omitempty"`
	AuthMethod     string `json:"auth_method,omitempty"` // plain, login, cram-md5

	// Custom settings
	Templates       map[string]*EmailTemplate `json:"templates,omitempty"`
	DefaultTemplate string                    `json:"default_template,omitempty"`

	// Advanced features
	EnableTracking bool              `json:"enable_tracking,omitempty"`
	TrackingDomain string            `json:"tracking_domain,omitempty"`
	CustomHeaders  map[string]string `json:"custom_headers,omitempty"`
	ReplyToAddress string            `json:"reply_to_address,omitempty"`
	BounceAddress  string            `json:"bounce_address,omitempty"`
	UnsubscribeURL string            `json:"unsubscribe_url,omitempty"`

	// Rate limiting
	RateLimit       int           `json:"rate_limit,omitempty"`  // emails per minute
	BurstLimit      int           `json:"burst_limit,omitempty"` // burst capacity
	RateLimitWindow time.Duration `json:"rate_limit_window,omitempty"`

	// Retry settings
	MaxRetries    int           `json:"max_retries,omitempty"`
	RetryInterval time.Duration `json:"retry_interval,omitempty"`
	Timeout       time.Duration `json:"timeout,omitempty"`

	// Validation rules
	AllowedDomains     []string `json:"allowed_domains,omitempty"`     // Only allow sending to these domains
	BlockedDomains     []string `json:"blocked_domains,omitempty"`     // Block sending to these domains
	RequireSSL         bool     `json:"require_ssl,omitempty"`         // Require SSL for all connections
	ValidateRecipients bool     `json:"validate_recipients,omitempty"` // Validate recipient email format

	// Metadata
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// CustomEmailSender provides enhanced email sending with custom configurations
type CustomEmailSender struct {
	config      *CustomEmailConfig
	smtpSender  *SMTPSender
	templateMgr *TemplateManager
	rateLimiter *RateLimiter
	logger      logger.Logger
	monitor     *EmailMonitor
}

// NewCustomEmailSender creates a new custom email sender
func NewCustomEmailSender(config *CustomEmailConfig, logger logger.Logger) (*CustomEmailSender, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid custom email config: %w", err)
	}

	// Convert to internal config
	internalConfig := config.ToInternalConfig()

	// Create SMTP sender
	smtpSender, err := NewSMTPSender(internalConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMTP sender: %w", err)
	}

	// Create template manager
	templateMgr := NewTemplateManager("", logger)
	templateMgr.CreateBasicTemplates()

	// Add custom templates
	if config.Templates != nil {
		for _, template := range config.Templates {
			if err := templateMgr.AddTemplate(template); err != nil {
				logger.Warn("添加自定义模板失败", "template", template.Name, "error", err)
			}
		}
	}

	// Create rate limiter if configured
	var rateLimiter *RateLimiter
	if config.RateLimit > 0 {
		rateLimiter = NewRateLimiter(config.RateLimit, config.BurstLimit, config.RateLimitWindow)
	}

	// Create monitor
	monitor := NewEmailMonitor(logger)

	return &CustomEmailSender{
		config:      config,
		smtpSender:  smtpSender,
		templateMgr: templateMgr,
		rateLimiter: rateLimiter,
		logger:      logger,
		monitor:     monitor,
	}, nil
}

// validateRecipients validates all recipients and returns valid ones
func (ces *CustomEmailSender) validateRecipients(recipients []string, result *CustomEmailResult) []string {
	validRecipients := make([]string, 0, len(recipients))

	for i, recipient := range recipients {
		if err := ces.validateRecipient(recipient); err != nil {
			result.Results[i] = &CustomRecipientResult{
				Recipient: recipient,
				Success:   false,
				Error:     err.Error(),
			}
			result.Failed++
			continue
		}
		validRecipients = append(validRecipients, recipient)
	}

	return validRecipients
}

// prepareTemplateData prepares template data from options
func (ces *CustomEmailSender) prepareTemplateData(options *CustomEmailOptions, start time.Time) *TemplateData {
	return &TemplateData{
		Title:     options.Subject,
		Body:      options.Body,
		Priority:  options.Priority,
		Variables: options.Variables,
		Timestamp: start.Format("2006-01-02 15:04:05"),
		Sender:    ces.config.From,
		Custom:    options.CustomData,
	}
}

// renderMessage renders message using template or creates basic message
func (ces *CustomEmailSender) renderMessage(options *CustomEmailOptions, templateData *TemplateData) (*message.Message, error) {
	if options.Template != "" {
		msg, err := ces.templateMgr.RenderTemplate(options.Template, templateData)
		if err != nil {
			return nil, fmt.Errorf("模板渲染失败: %w", err)
		}
		return msg, nil
	}

	// Create basic message
	msg := message.New()
	msg.Title = options.Subject
	msg.Body = options.Body
	msg.Format = message.FormatText
	return msg, nil
}

// addTrackingHeaders adds tracking headers if enabled
func (ces *CustomEmailSender) addTrackingHeaders(emailData map[string]interface{}) {
	if !ces.config.EnableTracking {
		return
	}

	trackingHeaders := make(map[string]string)

	if ces.config.TrackingDomain != "" {
		trackingHeaders["List-Unsubscribe"] = fmt.Sprintf("<%s/unsubscribe>", ces.config.TrackingDomain)
	}
	if ces.config.UnsubscribeURL != "" {
		trackingHeaders["List-Unsubscribe"] = fmt.Sprintf("<%s>", ces.config.UnsubscribeURL)
	}

	if len(trackingHeaders) == 0 {
		return
	}

	if emailData["headers"] == nil {
		emailData["headers"] = trackingHeaders
	} else {
		headers := emailData["headers"].(map[string]string)
		for k, v := range trackingHeaders {
			headers[k] = v
		}
	}
}

// prepareMessageWithHeaders prepares message with custom headers
func (ces *CustomEmailSender) prepareMessageWithHeaders(msg *message.Message) {
	if msg.PlatformData == nil {
		msg.PlatformData = make(map[string]interface{})
	}

	emailData := make(map[string]interface{})
	if ces.config.CustomHeaders != nil {
		emailData["headers"] = ces.config.CustomHeaders
	}

	ces.addTrackingHeaders(emailData)
	msg.PlatformData["email"] = emailData
}

// sendToRecipient sends email to a single recipient
func (ces *CustomEmailSender) sendToRecipient(ctx context.Context, msg *message.Message, recipient string, templateData *TemplateData) (*CustomRecipientResult, error) {
	recipientResult := &CustomRecipientResult{
		Recipient: recipient,
	}

	// Create target
	targets := []target.Target{{Type: "email", Value: recipient}}

	// Update template data for this recipient
	templateData.Recipient = recipient

	// Send email
	sendStart := time.Now()
	err := ces.smtpSender.SendMessage(ctx, msg, targets)
	duration := time.Since(sendStart)

	recipientResult.Duration = duration

	if err != nil {
		recipientResult.Success = false
		recipientResult.Error = err.Error()
		ces.monitor.RecordFailure(ces.config.Name, recipient, err, duration)
		ces.logger.Error("自定义邮件发送失败", "recipient", recipient, "error", err)
		return recipientResult, err
	}

	recipientResult.Success = true
	recipientResult.MessageID = fmt.Sprintf("custom_%d_%s", time.Now().UnixNano(), generateShortID())
	ces.monitor.RecordSending(ces.config.Name, recipient, duration)
	ces.logger.Info("自定义邮件发送成功", "recipient", recipient, "message_id", recipientResult.MessageID)

	return recipientResult, nil
}

// SendCustomEmail sends a custom email with template and advanced features
func (ces *CustomEmailSender) SendCustomEmail(ctx context.Context, options *CustomEmailOptions) (*CustomEmailResult, error) {
	ces.logger.Info("发送自定义邮件", "template", options.Template, "recipients", len(options.Recipients))

	// Rate limiting
	if ces.rateLimiter != nil {
		if !ces.rateLimiter.Allow() {
			return nil, NewEmailError(ErrorTypeRateLimit, "发送频率限制", nil)
		}
	}

	start := time.Now()
	result := &CustomEmailResult{
		RequestID: options.RequestID,
		Timestamp: start,
		Results:   make([]*CustomRecipientResult, len(options.Recipients)),
	}

	// Validate recipients
	validRecipients := ces.validateRecipients(options.Recipients, result)

	// Prepare template data
	templateData := ces.prepareTemplateData(options, start)

	// Render message
	msg, err := ces.renderMessage(options, templateData)
	if err != nil {
		return nil, err
	}

	// Prepare message with headers
	ces.prepareMessageWithHeaders(msg)

	// Send to each valid recipient
	successCount := 0
	for _, recipient := range validRecipients {
		recipientResult, err := ces.sendToRecipient(ctx, msg, recipient, templateData)

		if err == nil {
			successCount++
		}

		// Find the correct index in the original recipients list
		for j, originalRecipient := range options.Recipients {
			if originalRecipient == recipient {
				result.Results[j] = recipientResult
				break
			}
		}
	}

	result.Successful = successCount
	result.Total = len(options.Recipients)
	result.Duration = time.Since(start)

	ces.logger.Info("自定义邮件发送完成",
		"total", result.Total,
		"successful", result.Successful,
		"failed", result.Failed,
		"duration", result.Duration)

	return result, nil
}

// validateRecipient validates a recipient email address
func (ces *CustomEmailSender) validateRecipient(recipient string) error {
	// Basic email format validation
	if !isValidEmail(recipient) {
		return fmt.Errorf("无效的邮箱地址: %s", recipient)
	}

	// Domain validation
	if len(ces.config.AllowedDomains) > 0 {
		domain := extractDomain(recipient)
		allowed := false
		for _, allowedDomain := range ces.config.AllowedDomains {
			if strings.EqualFold(domain, allowedDomain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("域名不在允许列表中: %s", domain)
		}
	}

	// Blocked domains check
	if len(ces.config.BlockedDomains) > 0 {
		domain := extractDomain(recipient)
		for _, blockedDomain := range ces.config.BlockedDomains {
			if strings.EqualFold(domain, blockedDomain) {
				return fmt.Errorf("域名在黑名单中: %s", domain)
			}
		}
	}

	return nil
}

// extractDomain extracts domain from email address
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return strings.ToLower(parts[1])
	}
	return ""
}

// TestConnection tests the custom email configuration
func (ces *CustomEmailSender) TestConnection(ctx context.Context) error {
	ces.logger.Info("测试自定义邮件配置连接", "config", ces.config.Name)
	return ces.smtpSender.TestConnection(ctx)
}

// GetCapabilities returns the capabilities of the custom email sender
func (ces *CustomEmailSender) GetCapabilities() *CustomEmailCapabilities {
	return &CustomEmailCapabilities{
		SupportsTemplates:     true,
		SupportsTracking:      ces.config.EnableTracking,
		SupportsRateLimiting:  ces.rateLimiter != nil,
		SupportsCustomHeaders: len(ces.config.CustomHeaders) > 0,
		MaxRecipientsPerEmail: 100, // Configurable limit
		SupportedFormats:      []string{"text", "html", "markdown"},
		Templates:             ces.templateMgr.ListTemplates(),
	}
}

// GetMetrics returns email sending metrics
func (ces *CustomEmailSender) GetMetrics() EmailMetricsSnapshot {
	return ces.monitor.GetMetrics()
}

// Close closes the custom email sender
func (ces *CustomEmailSender) Close() error {
	ces.logger.Info("关闭自定义邮件发送器", "config", ces.config.Name)
	return ces.smtpSender.Close()
}

// CustomEmailOptions represents options for sending custom emails
type CustomEmailOptions struct {
	RequestID  string                 `json:"request_id,omitempty"`
	Template   string                 `json:"template,omitempty"`
	Subject    string                 `json:"subject"`
	Body       string                 `json:"body,omitempty"`
	Recipients []string               `json:"recipients"`
	Priority   string                 `json:"priority,omitempty"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
	CustomData map[string]interface{} `json:"custom_data,omitempty"`
	ScheduleAt *time.Time             `json:"schedule_at,omitempty"`
}

// CustomEmailResult represents the result of sending custom emails
type CustomEmailResult struct {
	RequestID  string                   `json:"request_id,omitempty"`
	Timestamp  time.Time                `json:"timestamp"`
	Duration   time.Duration            `json:"duration"`
	Total      int                      `json:"total"`
	Successful int                      `json:"successful"`
	Failed     int                      `json:"failed"`
	Results    []*CustomRecipientResult `json:"results"`
}

// CustomRecipientResult represents the result for a specific recipient
type CustomRecipientResult struct {
	Recipient string        `json:"recipient"`
	Success   bool          `json:"success"`
	MessageID string        `json:"message_id,omitempty"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
}

// CustomEmailCapabilities represents the capabilities of a custom email sender
type CustomEmailCapabilities struct {
	SupportsTemplates     bool                      `json:"supports_templates"`
	SupportsTracking      bool                      `json:"supports_tracking"`
	SupportsRateLimiting  bool                      `json:"supports_rate_limiting"`
	SupportsCustomHeaders bool                      `json:"supports_custom_headers"`
	MaxRecipientsPerEmail int                       `json:"max_recipients_per_email"`
	SupportedFormats      []string                  `json:"supported_formats"`
	Templates             map[string]*EmailTemplate `json:"templates"`
}

// Validate validates the custom email configuration
func (config *CustomEmailConfig) Validate() error {
	if config.Name == "" {
		return fmt.Errorf("配置名称不能为空")
	}

	if config.Host == "" {
		return fmt.Errorf("SMTP主机不能为空")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("SMTP端口必须在1-65535之间")
	}

	if config.From == "" {
		return fmt.Errorf("发件人地址不能为空")
	}

	if !isValidEmail(config.From) {
		return fmt.Errorf("发件人地址格式无效: %s", config.From)
	}

	// Validate unsubscribe URL if provided
	if config.UnsubscribeURL != "" {
		if _, err := url.Parse(config.UnsubscribeURL); err != nil {
			return fmt.Errorf("取消订阅URL格式无效: %w", err)
		}
	}

	// Validate rate limiting settings
	if config.RateLimit < 0 {
		return fmt.Errorf("发送频率限制不能为负数")
	}

	if config.BurstLimit < 0 {
		return fmt.Errorf("突发限制不能为负数")
	}

	return nil
}

// ToInternalConfig converts custom config to internal config
func (config *CustomEmailConfig) ToInternalConfig() *Config {
	internalConfig := NewConfig()

	internalConfig.SMTPHost = config.Host
	internalConfig.SMTPPort = config.Port
	internalConfig.Username = config.Username
	internalConfig.Password = config.Password
	internalConfig.From = config.From
	internalConfig.FromName = config.FromName
	internalConfig.UseTLS = config.UseTLS
	internalConfig.UseStartTLS = config.UseStartTLS
	internalConfig.SkipCertVerify = config.SkipCertVerify
	internalConfig.AuthMethod = config.AuthMethod
	internalConfig.ReplyTo = config.ReplyToAddress
	internalConfig.ReturnPath = config.BounceAddress

	if config.Timeout > 0 {
		internalConfig.Timeout = &config.Timeout
	}

	if config.MaxRetries > 0 {
		internalConfig.MaxRetries = &config.MaxRetries
	}

	if config.CustomHeaders != nil {
		internalConfig.Headers = make(map[string]string)
		for k, v := range config.CustomHeaders {
			internalConfig.Headers[k] = v
		}
	}

	return internalConfig
}
