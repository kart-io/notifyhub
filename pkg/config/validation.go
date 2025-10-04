// Package config provides configuration validation functionality for NotifyHub
package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/kart/notifyhub/pkg/config/platforms"
)

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
	Summary  ValidationSummary   `json:"summary"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalErrors   int `json:"total_errors"`
	TotalWarnings int `json:"total_warnings"`
	PlatformCount int `json:"platform_count"`
}

// Validator provides configuration validation functionality
type Validator struct {
	strict bool
}

// NewValidator creates a new configuration validator
func NewValidator(strict bool) *Validator {
	return &Validator{strict: strict}
}

// Validate validates a configuration
func (v *Validator) Validate(cfg *Config) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Validate core configuration
	v.validateCore(cfg, result)

	// Validate platform configurations
	v.validatePlatforms(cfg, result)

	// Validate async configuration
	v.validateAsync(cfg, result)

	// Validate logger configuration
	v.validateLogger(cfg, result)

	// Set overall validity
	result.Valid = len(result.Errors) == 0

	// Generate summary
	result.Summary = ValidationSummary{
		TotalErrors:   len(result.Errors),
		TotalWarnings: len(result.Warnings),
		PlatformCount: v.countConfiguredPlatforms(cfg),
	}

	return result
}

// validateCore validates core configuration settings
func (v *Validator) validateCore(cfg *Config, result *ValidationResult) {
	// Validate timeout
	if cfg.Timeout <= 0 {
		v.addError(result, "timeout", "INVALID_TIMEOUT", "timeout must be greater than 0")
	} else if cfg.Timeout > 5*time.Minute {
		v.addWarning(result, "timeout", "LONG_TIMEOUT", "timeout is unusually long, consider reducing it")
	}

	// Validate max retries
	if cfg.MaxRetries < 0 {
		v.addError(result, "max_retries", "INVALID_RETRIES", "max_retries cannot be negative")
	} else if cfg.MaxRetries > 10 {
		v.addWarning(result, "max_retries", "HIGH_RETRIES", "max_retries is unusually high, consider reducing it")
	}
}

// validatePlatforms validates platform configurations
func (v *Validator) validatePlatforms(cfg *Config, result *ValidationResult) {
	platformCount := 0

	// Validate Feishu configuration
	if cfg.Feishu != nil {
		v.validateFeishu(cfg.Feishu, result)
		platformCount++
	}

	// Validate Email configuration
	if cfg.Email != nil {
		v.validateEmail(cfg.Email, result)
		platformCount++
	}

	// Validate Webhook configuration
	if cfg.Webhook != nil {
		v.validateWebhook(cfg.Webhook, result)
		platformCount++
	}

	// Warn if no platforms configured
	if platformCount == 0 {
		v.addWarning(result, "platforms", "NO_PLATFORMS", "no platforms configured, notifications cannot be sent")
	}
}

// validateFeishu validates Feishu platform configuration
func (v *Validator) validateFeishu(cfg *platforms.FeishuConfig, result *ValidationResult) {
	// Validate webhook URL
	if cfg.WebhookURL == "" {
		v.addError(result, "feishu.webhook_url", "MISSING_WEBHOOK_URL", "Feishu webhook URL is required")
		return
	}

	if err := v.validateURL(cfg.WebhookURL); err != nil {
		v.addError(result, "feishu.webhook_url", "INVALID_WEBHOOK_URL", fmt.Sprintf("invalid Feishu webhook URL: %v", err))
	}

	// Validate Feishu-specific URL format
	if !strings.Contains(cfg.WebhookURL, "feishu.cn") && !strings.Contains(cfg.WebhookURL, "larksuite.com") {
		v.addWarning(result, "feishu.webhook_url", "SUSPICIOUS_URL", "webhook URL does not appear to be a valid Feishu/Lark webhook")
	}

	// Validate timeout
	if cfg.Timeout < 0 {
		v.addError(result, "feishu.timeout", "INVALID_TIMEOUT", "Feishu timeout cannot be negative")
	}

	// Validate retries
	if cfg.MaxRetries < 0 {
		v.addError(result, "feishu.max_retries", "INVALID_RETRIES", "Feishu max_retries cannot be negative")
	}

	// Validate keywords
	if len(cfg.Keywords) > 0 {
		for i, keyword := range cfg.Keywords {
			if strings.TrimSpace(keyword) == "" {
				v.addError(result, fmt.Sprintf("feishu.keywords[%d]", i), "EMPTY_KEYWORD", "keyword cannot be empty")
			}
		}
	}

	// Warn about security
	if cfg.Secret == "" {
		v.addWarning(result, "feishu.secret", "NO_SECRET", "Feishu secret not configured, webhook security is disabled")
	}
}

// validateEmail validates Email platform configuration
func (v *Validator) validateEmail(cfg *platforms.EmailConfig, result *ValidationResult) {
	// Validate SMTP host
	if cfg.Host == "" {
		v.addError(result, "email.host", "MISSING_HOST", "email host is required")
	}

	// Validate SMTP port
	if cfg.Port <= 0 || cfg.Port > 65535 {
		v.addError(result, "email.port", "INVALID_PORT", "email port must be between 1 and 65535")
	}

	// Validate from address
	if cfg.From == "" {
		v.addError(result, "email.from", "MISSING_FROM", "email from address is required")
	} else if err := v.validateEmailAddress(cfg.From); err != nil {
		v.addError(result, "email.from", "INVALID_FROM", fmt.Sprintf("invalid from email address: %v", err))
	}

	// Validate authentication if provided
	if cfg.Username != "" || cfg.Password != "" {
		if cfg.Username == "" {
			v.addError(result, "email.username", "MISSING_USERNAME", "email username is required when password is provided")
		}
		if cfg.Password == "" {
			v.addError(result, "email.password", "MISSING_PASSWORD", "email password is required when username is provided")
		}
	}

	// Validate timeout
	if cfg.Timeout < 0 {
		v.addError(result, "email.timeout", "INVALID_TIMEOUT", "email timeout cannot be negative")
	}

	// Security warnings
	if !cfg.UseTLS && cfg.Port != 25 {
		v.addWarning(result, "email.use_tls", "NO_TLS", "TLS is disabled, consider enabling for security")
	}
}

// validateWebhook validates Webhook platform configuration
func (v *Validator) validateWebhook(cfg *platforms.WebhookConfig, result *ValidationResult) {
	// Validate webhook URL
	if cfg.URL == "" {
		v.addError(result, "webhook.url", "MISSING_URL", "webhook URL is required")
		return
	}

	if err := v.validateURL(cfg.URL); err != nil {
		v.addError(result, "webhook.url", "INVALID_URL", fmt.Sprintf("invalid webhook URL: %v", err))
	}

	// Validate method
	validMethods := []string{"GET", "POST", "PUT", "PATCH"}
	methodValid := false
	for _, method := range validMethods {
		if cfg.Method == method {
			methodValid = true
			break
		}
	}
	if !methodValid {
		v.addError(result, "webhook.method", "INVALID_METHOD", fmt.Sprintf("invalid HTTP method: %s", cfg.Method))
	}

	// Validate timeout
	if cfg.Timeout < 0 {
		v.addError(result, "webhook.timeout", "INVALID_TIMEOUT", "webhook timeout cannot be negative")
	}

	// Validate retries
	if cfg.MaxRetries < 0 {
		v.addError(result, "webhook.max_retries", "INVALID_RETRIES", "webhook max_retries cannot be negative")
	}

	// Validate headers
	for key, value := range cfg.Headers {
		if strings.TrimSpace(key) == "" {
			v.addError(result, "webhook.headers", "EMPTY_HEADER_KEY", "header key cannot be empty")
		}
		if strings.TrimSpace(value) == "" {
			v.addWarning(result, "webhook.headers", "EMPTY_HEADER_VALUE", fmt.Sprintf("header '%s' has empty value", key))
		}
	}

	// Security warning
	if strings.HasPrefix(cfg.URL, "http://") {
		v.addWarning(result, "webhook.url", "INSECURE_URL", "webhook URL uses HTTP, consider using HTTPS for security")
	}
}

// validateAsync validates async configuration
func (v *Validator) validateAsync(cfg *Config, result *ValidationResult) {
	// Validate worker count
	if cfg.Async.Workers <= 0 {
		v.addError(result, "async.workers", "INVALID_WORKERS", "async workers must be greater than 0")
	} else if cfg.Async.Workers > 100 {
		v.addWarning(result, "async.workers", "HIGH_WORKERS", "async worker count is unusually high")
	}
}

// validateLogger validates logger configuration
func (v *Validator) validateLogger(cfg *Config, result *ValidationResult) {
	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLevels {
		if cfg.Logger.Level == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		v.addError(result, "logger.level", "INVALID_LEVEL", fmt.Sprintf("invalid log level: %s", cfg.Logger.Level))
	}

	// Validate log format
	validFormats := []string{"json", "text", "console"}
	formatValid := false
	for _, format := range validFormats {
		if cfg.Logger.Format == format {
			formatValid = true
			break
		}
	}
	if !formatValid {
		v.addError(result, "logger.format", "INVALID_FORMAT", fmt.Sprintf("invalid log format: %s", cfg.Logger.Format))
	}
}

// Helper methods

// validateURL validates a URL
func (v *Validator) validateURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	if u.Scheme == "" {
		return fmt.Errorf("URL scheme is required")
	}

	if u.Host == "" {
		return fmt.Errorf("URL host is required")
	}

	return nil
}

// validateEmailAddress validates an email address
func (v *Validator) validateEmailAddress(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// addError adds a validation error
func (v *Validator) addError(result *ValidationResult, field, code, message string) {
	result.Errors = append(result.Errors, ValidationError{
		Field:   field,
		Code:    code,
		Message: message,
	})
}

// addWarning adds a validation warning
func (v *Validator) addWarning(result *ValidationResult, field, code, message string) {
	// Skip warnings in strict mode
	if v.strict {
		return
	}

	result.Warnings = append(result.Warnings, ValidationWarning{
		Field:   field,
		Code:    code,
		Message: message,
	})
}

// countConfiguredPlatforms counts configured platforms
func (v *Validator) countConfiguredPlatforms(cfg *Config) int {
	count := 0
	if cfg.Feishu != nil {
		count++
	}
	if cfg.Email != nil {
		count++
	}
	if cfg.Webhook != nil {
		count++
	}
	return count
}

// Convenience functions

// ValidateQuick performs quick validation with default settings
func ValidateQuick(cfg *Config) *ValidationResult {
	validator := NewValidator(false)
	return validator.Validate(cfg)
}

// ValidateStrict performs strict validation
func ValidateStrict(cfg *Config) *ValidationResult {
	validator := NewValidator(true)
	return validator.Validate(cfg)
}

// IsValid checks if configuration is valid
func IsValid(cfg *Config) bool {
	result := ValidateQuick(cfg)
	return result.Valid
}
