// Package email provides Email platform integration for NotifyHub
package email

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/kart/notifyhub/pkg/config"
	"github.com/kart/notifyhub/pkg/message"
	"github.com/kart/notifyhub/pkg/platform"
	"github.com/kart/notifyhub/pkg/target"
	"github.com/kart/notifyhub/pkg/utils/logger"
)

// EmailPlatform implements the Platform interface for email notifications
type EmailPlatform struct {
	config     *config.EmailConfig
	logger     logger.Logger
	smtpSender *SMTPSender
}

// NewEmailPlatform creates a new Email platform with strong-typed configuration
func NewEmailPlatform(emailConfig *config.EmailConfig, logger logger.Logger) (platform.Platform, error) {
	if emailConfig == nil {
		return nil, fmt.Errorf("email configuration cannot be nil")
	}

	// Validate required fields
	if emailConfig.Host == "" {
		return nil, fmt.Errorf("host is required for Email platform")
	}
	if emailConfig.Port == 0 {
		return nil, fmt.Errorf("port is required for Email platform")
	}
	if emailConfig.From == "" {
		return nil, fmt.Errorf("from address is required for Email platform")
	}

	// Convert NotifyHub email config to internal email config
	internalConfig := convertToInternalConfig(emailConfig)

	// Create SMTP sender
	smtpSender, err := NewSMTPSender(internalConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMTP sender: %w", err)
	}

	return &EmailPlatform{
		config:     emailConfig,
		logger:     logger,
		smtpSender: smtpSender,
	}, nil
}

// Name returns the platform name
func (e *EmailPlatform) Name() string {
	return "email"
}

// Send sends an email message
func (e *EmailPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	// Pre-validate message
	if msg == nil {
		err := fmt.Errorf("message cannot be nil")
		e.logger.Error("邮件发送失败", "error", err)
		return nil, err
	}

	e.logger.Info("开始发送邮件", "message_title", msg.Title, "targets_count", len(targets), "smtp_host", e.config.Host)

	// Create error analyzer for enhanced error handling
	errorAnalyzer := NewErrorAnalyzer("email")

	if msg.Title == "" {
		e.logger.Warn("邮件标题为空，使用默认标题")
		msg.Title = "NotifyHub邮件通知"
	}

	results := make([]*platform.SendResult, len(targets))
	successCount := 0
	failureCount := 0

	for i, tgt := range targets {
		result := &platform.SendResult{
			Target:  tgt,
			Success: false,
		}

		e.logger.Debug("处理收件人", "target", tgt.Value, "target_type", tgt.Type)

		// Validate target
		if err := e.ValidateTarget(tgt); err != nil {
			e.logger.Error("收件人验证失败", "target", tgt.Value, "error", err)

			// Enhance error with analysis
			enhancedErr := errorAnalyzer.AnalyzeError(err)
			result.Error = enhancedErr
			result.Response = FormatErrorForUser(enhancedErr)
			results[i] = result
			failureCount++
			continue
		}

		e.logger.Info("发送邮件到", "target", tgt.Value, "smtp_host", e.config.Host, "smtp_port", e.config.Port)

		// Track sending time
		startTime := time.Now()

		// Use real SMTP sender
		if err := e.smtpSender.SendMessage(ctx, msg, []target.Target{tgt}); err != nil {
			duration := time.Since(startTime)
			e.logger.Error("SMTP邮件发送失败",
				"target", tgt.Value,
				"error", err,
				"duration", duration,
				"smtp_host", e.config.Host)

			// Enhance error with detailed analysis
			enhancedErr := errorAnalyzer.AnalyzeError(err)
			result.Error = enhancedErr
			result.Success = false
			result.Response = FormatErrorForUser(enhancedErr)

			// Log suggestions for debugging
			if len(enhancedErr.Suggestions) > 0 {
				e.logger.Info("错误解决建议", "suggestions", enhancedErr.Suggestions[:min(3, len(enhancedErr.Suggestions))])
			}

			failureCount++
		} else {
			duration := time.Since(startTime)
			result.Success = true
			result.MessageID = fmt.Sprintf("smtp_%d_%s", time.Now().UnixNano(), generateShortID())
			result.Response = fmt.Sprintf("邮件发送成功 (耗时: %v)", duration)

			e.logger.Info("邮件发送成功",
				"target", tgt.Value,
				"message_id", result.MessageID,
				"duration", duration,
				"message_size", len(msg.Body))
			successCount++
		}

		results[i] = result
	}

	// Log final summary
	e.logger.Info("邮件发送完成",
		"total", len(targets),
		"success", successCount,
		"failed", failureCount,
		"success_rate", fmt.Sprintf("%.1f%%", float64(successCount)/float64(len(targets))*100))

	// If all failed, return an error
	if failureCount == len(targets) {
		e.logger.Error("所有邮件发送失败")
		return results, fmt.Errorf("all email sending failed (%d/%d)", failureCount, len(targets))
	}

	return results, nil
}

// ValidateTarget validates a target for Email
func (e *EmailPlatform) ValidateTarget(tgt target.Target) error {
	if tgt.Type != "email" {
		return fmt.Errorf("email platform supports email targets only, got %s", tgt.Type)
	}

	if tgt.Value == "" {
		return fmt.Errorf("email target value cannot be empty")
	}

	// Basic email validation
	if !isValidEmail(tgt.Value) {
		return fmt.Errorf("invalid email address: %s", tgt.Value)
	}

	return nil
}

// GetCapabilities returns Email platform capabilities
func (e *EmailPlatform) GetCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Name:                 "email",
		SupportedTargetTypes: []string{"email"},
		SupportedFormats:     []string{"text", "html"},
		MaxMessageSize:       10 * 1024 * 1024, // 10MB
		SupportsScheduling:   false,
		SupportsAttachments:  true,
		RequiredSettings:     []string{"host", "port", "from"},
	}
}

// IsHealthy checks if SMTP server is accessible
func (e *EmailPlatform) IsHealthy(ctx context.Context) error {
	if e.config.Host == "" || e.config.Port == 0 {
		return fmt.Errorf("email configuration is incomplete")
	}

	// Perform SMTP health check
	e.logger.Debug("Performing SMTP health check", "host", e.config.Host, "port", e.config.Port)

	// Use SMTPSender's connection test capability
	if e.smtpSender != nil {
		// Test SMTP connection
		testErr := e.smtpSender.TestConnection(ctx)
		if testErr != nil {
			e.logger.Warn("SMTP health check failed", "error", testErr)
			return fmt.Errorf("SMTP server unhealthy: %w", testErr)
		}
		e.logger.Debug("SMTP health check passed")
		return nil
	}

	// Fallback: basic configuration validation
	e.logger.Debug("SMTP sender not initialized, validating configuration only")
	return nil
}

// Close cleans up resources
func (e *EmailPlatform) Close() error {
	e.logger.Info("Closing Email platform")
	if e.smtpSender != nil {
		return e.smtpSender.Close()
	}
	return nil
}

// NewPlatform is the factory function for creating Email platforms
// This function will be called by the platform registry
func NewPlatform(cfg interface{}, log logger.Logger) (platform.Platform, error) {
	emailConfig, ok := cfg.(*config.EmailConfig)
	if !ok {
		return nil, fmt.Errorf("invalid email configuration type")
	}

	return NewEmailPlatform(emailConfig, log)
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	// Very basic email validation - in production, use a proper email validation library
	if len(email) < 3 {
		return false
	}
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex >= 0 {
				return false // Multiple @ symbols
			}
			atIndex = i
		}
	}
	return atIndex > 0 && atIndex < len(email)-1
}

// convertToInternalConfig converts NotifyHub EmailConfig to internal Config
func convertToInternalConfig(nhConfig *config.EmailConfig) *Config {
	internalConfig := NewConfig()

	// Map NotifyHub config to internal config
	internalConfig.SMTPHost = nhConfig.Host
	internalConfig.SMTPPort = nhConfig.Port
	internalConfig.Username = nhConfig.Username
	internalConfig.Password = nhConfig.Password
	internalConfig.From = nhConfig.From
	internalConfig.UseTLS = nhConfig.UseTLS

	// Set appropriate TLS settings based on email provider
	host := strings.ToLower(nhConfig.Host)

	// Gmail settings
	if strings.Contains(host, "gmail.com") {
		internalConfig.UseStartTLS = true
		internalConfig.UseTLS = false // Gmail uses STARTTLS, not direct TLS
		internalConfig.AuthMethod = "plain"
	}

	// 163.com and other NetEase email settings
	if strings.Contains(host, "163.com") || strings.Contains(host, "126.com") || strings.Contains(host, "yeah.net") {
		internalConfig.UseStartTLS = true
		internalConfig.UseTLS = false // NetEase uses STARTTLS
		internalConfig.AuthMethod = "plain"
		// 163 commonly uses port 25 for STARTTLS
		if nhConfig.Port == 0 {
			internalConfig.SMTPPort = 25
		}
	}

	// QQ Mail settings (including Tencent Enterprise Mail)
	if strings.Contains(host, "qq.com") || strings.Contains(host, "exmail.qq.com") {
		internalConfig.UseStartTLS = true
		internalConfig.UseTLS = false
		internalConfig.AuthMethod = "plain"
		if nhConfig.Port == 0 {
			internalConfig.SMTPPort = 587
		}
	}

	// Sina Mail settings
	if strings.Contains(host, "sina.com") || strings.Contains(host, "sina.cn") {
		internalConfig.UseStartTLS = true
		internalConfig.UseTLS = false
		internalConfig.AuthMethod = "plain"
		if nhConfig.Port == 0 {
			internalConfig.SMTPPort = 25
		}
	}

	// Sohu Mail settings
	if strings.Contains(host, "sohu.com") {
		internalConfig.UseStartTLS = true
		internalConfig.UseTLS = false
		internalConfig.AuthMethod = "plain"
		if nhConfig.Port == 0 {
			internalConfig.SMTPPort = 25
		}
	}

	// Yahoo Mail settings (including Yahoo Japan)
	if strings.Contains(host, "yahoo.com") || strings.Contains(host, "yahoo.co.jp") {
		internalConfig.UseStartTLS = true
		internalConfig.UseTLS = false
		internalConfig.AuthMethod = "plain"
		if nhConfig.Port == 0 {
			internalConfig.SMTPPort = 587
		}
	}

	// Zoho Mail settings
	if strings.Contains(host, "zoho.com") {
		internalConfig.UseStartTLS = true
		internalConfig.UseTLS = false
		internalConfig.AuthMethod = "plain"
		if nhConfig.Port == 0 {
			internalConfig.SMTPPort = 587
		}
	}

	// Alibaba Mail settings
	if strings.Contains(host, "mxhichina.com") {
		internalConfig.UseStartTLS = true
		internalConfig.UseTLS = false
		internalConfig.AuthMethod = "plain"
		if nhConfig.Port == 0 {
			internalConfig.SMTPPort = 587
		}
	}

	// ProtonMail Bridge settings
	if strings.Contains(host, "127.0.0.1") && nhConfig.Port == 1025 {
		internalConfig.UseStartTLS = true
		internalConfig.UseTLS = false
		internalConfig.AuthMethod = "plain"
	}

	// Outlook/Hotmail settings
	if strings.Contains(host, "outlook") || strings.Contains(host, "hotmail") {
		internalConfig.UseStartTLS = true
		internalConfig.UseTLS = false
		internalConfig.AuthMethod = "plain"
		if nhConfig.Port == 0 {
			internalConfig.SMTPPort = 587
		}
	}

	// Map timeout if available
	if nhConfig.Timeout > 0 {
		timeout := nhConfig.Timeout
		internalConfig.Timeout = &timeout
	}

	return internalConfig
}

// Helper functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateShortID generates a short random ID for message tracking
func generateShortID() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return fmt.Sprintf("%x", time.Now().UnixNano()%0xFFFFFFFF)
	}
	return hex.EncodeToString(bytes)
}
