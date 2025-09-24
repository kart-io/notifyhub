// Package validation provides pre-defined validation rules for NotifyHub
package validation

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Advanced validation rules

// IPAddressRule validates IP addresses (IPv4 or IPv6)
type IPAddressRule struct {
	AllowIPv4 bool
	AllowIPv6 bool
}

func NewIPAddressRule(allowIPv4, allowIPv6 bool) *IPAddressRule {
	return &IPAddressRule{
		AllowIPv4: allowIPv4,
		AllowIPv6: allowIPv6,
	}
}

func (r IPAddressRule) Name() string { return "ip_address" }

func (r IPAddressRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	ip := net.ParseIP(str)
	if ip == nil {
		return fmt.Errorf("invalid IP address")
	}

	if ip.To4() != nil {
		// IPv4
		if !r.AllowIPv4 {
			return fmt.Errorf("IPv4 addresses not allowed")
		}
	} else {
		// IPv6
		if !r.AllowIPv6 {
			return fmt.Errorf("IPv6 addresses not allowed")
		}
	}

	return nil
}

// PortRule validates port numbers
type PortRule struct{}

func (r PortRule) Name() string { return "port" }

func (r PortRule) Validate(value interface{}) error {
	var port int
	var err error

	switch v := value.(type) {
	case int:
		port = v
	case int64:
		port = int(v)
	case float64:
		port = int(v)
	case string:
		port, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("port must be a number")
		}
	default:
		return fmt.Errorf("port must be a number")
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

// DomainRule validates domain names
type DomainRule struct{}

func (r DomainRule) Name() string { return "domain" }

func (r DomainRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	// Basic domain validation using regex
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !domainRegex.MatchString(str) {
		return fmt.Errorf("invalid domain name")
	}

	if len(str) > 253 {
		return fmt.Errorf("domain name too long")
	}

	return nil
}

// HostnameRule validates hostnames (domain or IP)
type HostnameRule struct{}

func (r HostnameRule) Name() string { return "hostname" }

func (r HostnameRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	// Try IP address first
	if ip := net.ParseIP(str); ip != nil {
		return nil // Valid IP address
	}

	// Try domain name
	domainRule := DomainRule{}
	return domainRule.Validate(str)
}

// JSONRule validates JSON format
type JSONRule struct{}

func (r JSONRule) Name() string { return "json" }

func (r JSONRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	var js interface{}
	if err := json.Unmarshal([]byte(str), &js); err != nil {
		return fmt.Errorf("invalid JSON format")
	}

	return nil
}

// Base64Rule validates base64 encoding
type Base64Rule struct{}

func (r Base64Rule) Name() string { return "base64" }

func (r Base64Rule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	_, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding")
	}

	return nil
}

// HexRule validates hexadecimal encoding
type HexRule struct{}

func (r HexRule) Name() string { return "hex" }

func (r HexRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	_, err := hex.DecodeString(str)
	if err != nil {
		return fmt.Errorf("invalid hexadecimal encoding")
	}

	return nil
}

// UUIDRule validates UUID format
type UUIDRule struct{}

func (r UUIDRule) Name() string { return "uuid" }

func (r UUIDRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(strings.ToLower(str)) {
		return fmt.Errorf("invalid UUID format")
	}

	return nil
}

// NotifyHub-specific validation rules

// PlatformRule validates platform names
type PlatformRule struct{}

func (r PlatformRule) Name() string { return "platform" }

func (r PlatformRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	validPlatforms := []string{"feishu", "email", "sms", "webhook"}
	for _, platform := range validPlatforms {
		if str == platform {
			return nil
		}
	}

	return fmt.Errorf("invalid platform, must be one of: %s", strings.Join(validPlatforms, ", "))
}

// TargetTypeRule validates target types
type TargetTypeRule struct{}

func (r TargetTypeRule) Name() string { return "target_type" }

func (r TargetTypeRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	validTypes := []string{"email", "phone", "user", "group", "webhook"}
	for _, targetType := range validTypes {
		if str == targetType {
			return nil
		}
	}

	return fmt.Errorf("invalid target type, must be one of: %s", strings.Join(validTypes, ", "))
}

// MessageFormatRule validates message formats
type MessageFormatRule struct{}

func (r MessageFormatRule) Name() string { return "message_format" }

func (r MessageFormatRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	validFormats := []string{"text", "markdown", "html", "json"}
	for _, format := range validFormats {
		if str == format {
			return nil
		}
	}

	return fmt.Errorf("invalid message format, must be one of: %s", strings.Join(validFormats, ", "))
}

// PriorityRule validates message priorities
type PriorityRule struct{}

func (r PriorityRule) Name() string { return "priority" }

func (r PriorityRule) Validate(value interface{}) error {
	var priority int
	var err error

	switch v := value.(type) {
	case int:
		priority = v
	case int64:
		priority = int(v)
	case float64:
		priority = int(v)
	case string:
		priority, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("priority must be a number")
		}
	default:
		return fmt.Errorf("priority must be a number")
	}

	if priority < 1 || priority > 4 {
		return fmt.Errorf("priority must be between 1 (low) and 4 (urgent)")
	}

	return nil
}

// WebhookURLRule validates webhook URLs with specific requirements
type WebhookURLRule struct{}

func (r WebhookURLRule) Name() string { return "webhook_url" }

func (r WebhookURLRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	// First validate as URL
	urlRule := URLRule{}
	if err := urlRule.Validate(str); err != nil {
		return err
	}

	// Additional webhook-specific validation
	if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
		return fmt.Errorf("webhook URL must use HTTP or HTTPS protocol")
	}

	// Recommend HTTPS for production
	if strings.HasPrefix(str, "http://") {
		// This is a warning, not an error - HTTP is allowed but not recommended
		// In production, you might want to make this an error
		_ = str // acknowledge the check was intentional
	}

	return nil
}

// FeishuUserIDRule validates Feishu user IDs
type FeishuUserIDRule struct{}

func (r FeishuUserIDRule) Name() string { return "feishu_user_id" }

func (r FeishuUserIDRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	// Feishu user IDs typically start with "ou_" or "oc_"
	if !strings.HasPrefix(str, "ou_") && !strings.HasPrefix(str, "oc_") {
		return fmt.Errorf("invalid Feishu user ID format")
	}

	if len(str) < 10 {
		return fmt.Errorf("feishu user ID too short")
	}

	return nil
}

// TimeoutRule validates timeout durations
type TimeoutRule struct {
	MinSeconds int
	MaxSeconds int
}

func (r TimeoutRule) Name() string { return "timeout" }

func (r TimeoutRule) Validate(value interface{}) error {
	var seconds int
	var err error

	switch v := value.(type) {
	case int:
		seconds = v
	case int64:
		seconds = int(v)
	case float64:
		seconds = int(v)
	case string:
		seconds, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("timeout must be a number")
		}
	case time.Duration:
		seconds = int(v.Seconds())
	default:
		return fmt.Errorf("timeout must be a number or duration")
	}

	minSec := r.MinSeconds
	if minSec == 0 {
		minSec = 1 // Default minimum 1 second
	}

	maxSec := r.MaxSeconds
	if maxSec == 0 {
		maxSec = 300 // Default maximum 5 minutes
	}

	if seconds < minSec || seconds > maxSec {
		return fmt.Errorf("timeout must be between %d and %d seconds", minSec, maxSec)
	}

	return nil
}

// Composite validation rules

// WebhookConfigRule validates complete webhook configuration
type WebhookConfigRule struct{}

func (r WebhookConfigRule) Name() string { return "webhook_config" }

func (r WebhookConfigRule) Validate(value interface{}) error {
	configMap, ok := value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("webhook config must be a map")
	}

	validator := NewValidator()
	validator.AddRule("webhook_url", RequiredRule{})
	validator.AddRule("webhook_url", WebhookURLRule{})

	// Optional timeout
	if _, exists := configMap["timeout"]; exists {
		validator.AddRule("timeout", TimeoutRule{MinSeconds: 1, MaxSeconds: 60})
	}

	result := validator.Validate(configMap)
	if !result.Valid {
		return fmt.Errorf("webhook config validation failed: %v", result.Errors)
	}

	return nil
}

// EmailConfigRule validates complete email configuration
type EmailConfigRule struct{}

func (r EmailConfigRule) Name() string { return "email_config" }

func (r EmailConfigRule) Validate(value interface{}) error {
	configMap, ok := value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("email config must be a map")
	}

	validator := NewValidator()
	validator.AddRule("smtp_host", RequiredRule{})
	validator.AddRule("smtp_host", HostnameRule{})
	validator.AddRule("smtp_port", RequiredRule{})
	validator.AddRule("smtp_port", PortRule{})
	validator.AddRule("smtp_username", RequiredRule{})
	validator.AddRule("smtp_password", RequiredRule{})
	validator.AddRule("smtp_from", RequiredRule{})
	validator.AddRule("smtp_from", EmailRule{})

	result := validator.Validate(configMap)
	if !result.Valid {
		return fmt.Errorf("email config validation failed: %v", result.Errors)
	}

	return nil
}

// Rule builder functions

// BuildStringRule builds a string validation rule with common constraints
func BuildStringRule(required bool, minLen, maxLen int, pattern string) (*Validator, error) {
	validator := NewValidator()

	if required {
		validator.AddRule("value", RequiredRule{})
	}

	if minLen > 0 {
		validator.AddRule("value", MinLengthRule{Min: minLen})
	}

	if maxLen > 0 {
		validator.AddRule("value", MaxLengthRule{Max: maxLen})
	}

	if pattern != "" {
		regexRule, err := NewRegexRule(pattern, "value does not match required pattern")
		if err != nil {
			return nil, err
		}
		validator.AddRule("value", *regexRule)
	}

	return validator, nil
}

// BuildNumberRule builds a number validation rule with range constraints
func BuildNumberRule(required bool, min, max float64) *Validator {
	validator := NewValidator()

	if required {
		validator.AddRule("value", RequiredRule{})
	}

	validator.AddRule("value", RangeRule{Min: min, Max: max})

	return validator
}

// BuildChoiceRule builds a choice validation rule
func BuildChoiceRule(required bool, choices []interface{}) *Validator {
	validator := NewValidator()

	if required {
		validator.AddRule("value", RequiredRule{})
	}

	validator.AddRule("value", InRule{AllowedValues: choices})

	return validator
}
