// Package validation provides validation utilities for NotifyHub
package validation

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Validator provides validation functionality
type Validator struct {
	rules map[string][]Rule
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		rules: make(map[string][]Rule),
	}
}

// Rule represents a validation rule
type Rule interface {
	Validate(value interface{}) error
	Name() string
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors map[string]string `json:"errors,omitempty"`
}

// AddRule adds a validation rule for a field
func (v *Validator) AddRule(field string, rule Rule) {
	v.rules[field] = append(v.rules[field], rule)
}

// Validate validates a data map against all registered rules
func (v *Validator) Validate(data map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make(map[string]string),
	}

	for field, rules := range v.rules {
		value, exists := data[field]

		for _, rule := range rules {
			// Special handling for required rule when value doesn't exist
			if !exists && rule.Name() == "required" {
				result.Valid = false
				result.Errors[field] = "field is required"
				break
			}

			// Skip validation for non-required fields that don't exist
			if !exists {
				continue
			}

			if err := rule.Validate(value); err != nil {
				result.Valid = false
				result.Errors[field] = err.Error()
				break // Stop at first error for this field
			}
		}
	}

	return result
}

// ValidateField validates a single field value
func (v *Validator) ValidateField(field string, value interface{}) error {
	rules, exists := v.rules[field]
	if !exists {
		return nil // No rules for this field
	}

	for _, rule := range rules {
		if err := rule.Validate(value); err != nil {
			return err
		}
	}

	return nil
}

// HasRules checks if a field has validation rules
func (v *Validator) HasRules(field string) bool {
	_, exists := v.rules[field]
	return exists
}

// GetRules returns the rules for a field
func (v *Validator) GetRules(field string) []Rule {
	return v.rules[field]
}

// ClearRules removes all rules for a field
func (v *Validator) ClearRules(field string) {
	delete(v.rules, field)
}

// ClearAllRules removes all validation rules
func (v *Validator) ClearAllRules() {
	v.rules = make(map[string][]Rule)
}

// Built-in validation rules

// RequiredRule validates that a value is present and not empty
type RequiredRule struct{}

func (r RequiredRule) Name() string { return "required" }

func (r RequiredRule) Validate(value interface{}) error {
	if value == nil {
		return fmt.Errorf("value is required")
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("value is required")
		}
	case []interface{}:
		if len(v) == 0 {
			return fmt.Errorf("value is required")
		}
	case map[string]interface{}:
		if len(v) == 0 {
			return fmt.Errorf("value is required")
		}
	}

	return nil
}

// EmailRule validates email addresses
type EmailRule struct{}

func (r EmailRule) Name() string { return "email" }

func (r EmailRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	_, err := mail.ParseAddress(str)
	if err != nil {
		return fmt.Errorf("invalid email address")
	}

	return nil
}

// URLRule validates URLs
type URLRule struct{}

func (r URLRule) Name() string { return "url" }

func (r URLRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	_, err := url.Parse(str)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}

	return nil
}

// PhoneRule validates phone numbers (basic validation)
type PhoneRule struct{}

func (r PhoneRule) Name() string { return "phone" }

func (r PhoneRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	// Basic phone validation - starts with + followed by digits
	matched, err := regexp.MatchString(`^\+[1-9]\d{1,14}$`, str)
	if err != nil {
		return fmt.Errorf("regex error: %w", err)
	}

	if !matched {
		return fmt.Errorf("invalid phone number format")
	}

	return nil
}

// MinLengthRule validates minimum string length
type MinLengthRule struct {
	Min int
}

func (r MinLengthRule) Name() string { return "min_length" }

func (r MinLengthRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if len(str) < r.Min {
		return fmt.Errorf("value must be at least %d characters long", r.Min)
	}

	return nil
}

// MaxLengthRule validates maximum string length
type MaxLengthRule struct {
	Max int
}

func (r MaxLengthRule) Name() string { return "max_length" }

func (r MaxLengthRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if len(str) > r.Max {
		return fmt.Errorf("value must be at most %d characters long", r.Max)
	}

	return nil
}

// RangeRule validates numeric ranges
type RangeRule struct {
	Min, Max float64
}

func (r RangeRule) Name() string { return "range" }

func (r RangeRule) Validate(value interface{}) error {
	var num float64
	var err error

	switch v := value.(type) {
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case float64:
		num = v
	case float32:
		num = float64(v)
	case string:
		num, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("value must be a number")
		}
	default:
		return fmt.Errorf("value must be a number")
	}

	if num < r.Min || num > r.Max {
		return fmt.Errorf("value must be between %g and %g", r.Min, r.Max)
	}

	return nil
}

// RegexRule validates against a regular expression
type RegexRule struct {
	Pattern *regexp.Regexp
	Message string
}

func NewRegexRule(pattern, message string) (*RegexRule, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return &RegexRule{
		Pattern: re,
		Message: message,
	}, nil
}

func (r RegexRule) Name() string { return "regex" }

func (r RegexRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	if !r.Pattern.MatchString(str) {
		if r.Message != "" {
			return fmt.Errorf("%s", r.Message)
		}
		return fmt.Errorf("value does not match required pattern")
	}

	return nil
}

// InRule validates that value is in a set of allowed values
type InRule struct {
	AllowedValues []interface{}
}

func (r InRule) Name() string { return "in" }

func (r InRule) Validate(value interface{}) error {
	for _, allowed := range r.AllowedValues {
		if value == allowed {
			return nil
		}
	}

	return fmt.Errorf("value must be one of: %v", r.AllowedValues)
}

// NotInRule validates that value is not in a set of forbidden values
type NotInRule struct {
	ForbiddenValues []interface{}
}

func (r NotInRule) Name() string { return "not_in" }

func (r NotInRule) Validate(value interface{}) error {
	for _, forbidden := range r.ForbiddenValues {
		if value == forbidden {
			return fmt.Errorf("value must not be one of: %v", r.ForbiddenValues)
		}
	}

	return nil
}

// DateRule validates date formats
type DateRule struct {
	Layout string
}

func (r DateRule) Name() string { return "date" }

func (r DateRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if str == "" {
		return nil // Allow empty for non-required fields
	}

	layout := r.Layout
	if layout == "" {
		layout = time.RFC3339 // Default to RFC3339
	}

	_, err := time.Parse(layout, str)
	if err != nil {
		return fmt.Errorf("invalid date format, expected %s", layout)
	}

	return nil
}

// Convenience functions for creating validators

// CreateRequiredStringValidator creates a validator for required strings
func CreateRequiredStringValidator(minLen, maxLen int) *Validator {
	v := NewValidator()
	v.AddRule("value", RequiredRule{})
	if minLen > 0 {
		v.AddRule("value", MinLengthRule{Min: minLen})
	}
	if maxLen > 0 {
		v.AddRule("value", MaxLengthRule{Max: maxLen})
	}
	return v
}

// CreateEmailValidator creates a validator for email addresses
func CreateEmailValidator(required bool) *Validator {
	v := NewValidator()
	if required {
		v.AddRule("email", RequiredRule{})
	}
	v.AddRule("email", EmailRule{})
	return v
}

// CreateURLValidator creates a validator for URLs
func CreateURLValidator(required bool) *Validator {
	v := NewValidator()
	if required {
		v.AddRule("url", RequiredRule{})
	}
	v.AddRule("url", URLRule{})
	return v
}

// CreatePhoneValidator creates a validator for phone numbers
func CreatePhoneValidator(required bool) *Validator {
	v := NewValidator()
	if required {
		v.AddRule("phone", RequiredRule{})
	}
	v.AddRule("phone", PhoneRule{})
	return v
}

// Batch validation utilities

// ValidateStruct validates a struct using reflection (simplified)
type StructValidator struct {
	validators map[string]*Validator
}

// NewStructValidator creates a new struct validator
func NewStructValidator() *StructValidator {
	return &StructValidator{
		validators: make(map[string]*Validator),
	}
}

// AddFieldValidator adds a validator for a struct field
func (sv *StructValidator) AddFieldValidator(fieldName string, validator *Validator) {
	sv.validators[fieldName] = validator
}

// ValidateMap validates a map of values
func (sv *StructValidator) ValidateMap(data map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make(map[string]string),
	}

	for fieldName, validator := range sv.validators {
		if value, exists := data[fieldName]; exists {
			fieldResult := validator.Validate(map[string]interface{}{"value": value})
			if !fieldResult.Valid {
				result.Valid = false
				if err, exists := fieldResult.Errors["value"]; exists {
					result.Errors[fieldName] = err
				}
			}
		} else {
			// Check if field is required
			if validator.HasRules("value") {
				for _, rule := range validator.GetRules("value") {
					if rule.Name() == "required" {
						result.Valid = false
						result.Errors[fieldName] = "field is required"
						break
					}
				}
			}
		}
	}

	return result
}

// Custom validation functions

// ValidateNotifyHubConfig validates NotifyHub configuration
func ValidateNotifyHubConfig(config map[string]interface{}) *ValidationResult {
	validator := NewStructValidator()

	// Platform validation
	if platforms, exists := config["platforms"]; exists {
		if platformMap, ok := platforms.(map[string]interface{}); ok {
			result := &ValidationResult{Valid: true, Errors: make(map[string]string)}

			for platform, platformConfig := range platformMap {
				if err := validatePlatformConfig(platform, platformConfig); err != nil {
					result.Valid = false
					result.Errors[fmt.Sprintf("platforms.%s", platform)] = err.Error()
				}
			}

			if !result.Valid {
				return result
			}
		}
	}

	// Timeout validation
	timeoutValidator := NewValidator()
	timeoutValidator.AddRule("timeout", RangeRule{Min: 1, Max: 300}) // 1 second to 5 minutes
	validator.AddFieldValidator("timeout", timeoutValidator)

	// Rate limit validation
	rateLimitValidator := NewValidator()
	rateLimitValidator.AddRule("rate_limit", RangeRule{Min: 1, Max: 10000})
	validator.AddFieldValidator("rate_limit", rateLimitValidator)

	return validator.ValidateMap(config)
}

// validatePlatformConfig validates platform-specific configuration
func validatePlatformConfig(platform string, config interface{}) error {
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return fmt.Errorf("platform config must be a map")
	}

	switch platform {
	case "feishu":
		return validateFeishuConfig(configMap)
	case "email":
		return validateEmailConfig(configMap)
	case "sms":
		return validateSMSConfig(configMap)
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

// validateFeishuConfig validates Feishu platform configuration
func validateFeishuConfig(config map[string]interface{}) error {
	validator := NewValidator()
	validator.AddRule("webhook_url", RequiredRule{})
	validator.AddRule("webhook_url", URLRule{})

	// Secret is optional but if provided should be non-empty
	if secret, exists := config["secret"]; exists {
		if str, ok := secret.(string); ok && strings.TrimSpace(str) != "" {
			validator.AddRule("secret", MinLengthRule{Min: 1})
		}
	}

	result := validator.Validate(config)
	if !result.Valid {
		return fmt.Errorf("feishu config validation failed: %v", result.Errors)
	}

	return nil
}

// validateEmailConfig validates Email platform configuration
func validateEmailConfig(config map[string]interface{}) error {
	validator := NewValidator()
	validator.AddRule("smtp_host", RequiredRule{})
	validator.AddRule("smtp_port", RequiredRule{})
	validator.AddRule("smtp_port", RangeRule{Min: 1, Max: 65535})
	validator.AddRule("smtp_username", RequiredRule{})
	validator.AddRule("smtp_password", RequiredRule{})
	validator.AddRule("smtp_from", RequiredRule{})
	validator.AddRule("smtp_from", EmailRule{})

	result := validator.Validate(config)
	if !result.Valid {
		return fmt.Errorf("email config validation failed: %v", result.Errors)
	}

	return nil
}

// validateSMSConfig validates SMS platform configuration
func validateSMSConfig(config map[string]interface{}) error {
	validator := NewValidator()
	validator.AddRule("provider", RequiredRule{})
	validator.AddRule("api_key", RequiredRule{})

	result := validator.Validate(config)
	if !result.Valid {
		return fmt.Errorf("sms config validation failed: %v", result.Errors)
	}

	return nil
}

// Message validation utilities

// ValidateMessage validates a notification message
func ValidateMessage(message map[string]interface{}) *ValidationResult {
	_ = NewValidator()

	// Either title or body must be present
	hasTitle := false
	hasBody := false

	if title, exists := message["title"]; exists {
		if str, ok := title.(string); ok && strings.TrimSpace(str) != "" {
			hasTitle = true
		}
	}

	if body, exists := message["body"]; exists {
		if str, ok := body.(string); ok && strings.TrimSpace(str) != "" {
			hasBody = true
		}
	}

	result := &ValidationResult{Valid: true, Errors: make(map[string]string)}

	if !hasTitle && !hasBody {
		result.Valid = false
		result.Errors["content"] = "message must have either title or body"
	}

	// Validate targets
	if targets, exists := message["targets"]; exists {
		if targetList, ok := targets.([]interface{}); ok {
			if len(targetList) == 0 {
				result.Valid = false
				result.Errors["targets"] = "message must have at least one target"
			} else {
				for i, target := range targetList {
					if err := validateTarget(target); err != nil {
						result.Valid = false
						result.Errors[fmt.Sprintf("targets[%d]", i)] = err.Error()
					}
				}
			}
		} else {
			result.Valid = false
			result.Errors["targets"] = "targets must be an array"
		}
	} else {
		result.Valid = false
		result.Errors["targets"] = "message must have targets"
	}

	// Validate priority if present
	if priority, exists := message["priority"]; exists {
		rangeRule := RangeRule{Min: 1, Max: 4}
		if err := rangeRule.Validate(priority); err != nil {
			result.Valid = false
			result.Errors["priority"] = "priority must be between 1 and 4"
		}
	}

	return result
}

// validateTarget validates a message target
func validateTarget(target interface{}) error {
	targetMap, ok := target.(map[string]interface{})
	if !ok {
		return fmt.Errorf("target must be an object")
	}

	validator := NewValidator()
	validator.AddRule("type", RequiredRule{})
	validator.AddRule("value", RequiredRule{})
	validator.AddRule("platform", RequiredRule{})

	result := validator.Validate(targetMap)
	if !result.Valid {
		return fmt.Errorf("target validation failed: %v", result.Errors)
	}

	// Validate target type-specific format
	targetType, _ := targetMap["type"].(string)
	value, _ := targetMap["value"].(string)

	switch targetType {
	case "email":
		emailRule := EmailRule{}
		if err := emailRule.Validate(value); err != nil {
			return fmt.Errorf("invalid email target: %w", err)
		}
	case "phone":
		phoneRule := PhoneRule{}
		if err := phoneRule.Validate(value); err != nil {
			return fmt.Errorf("invalid phone target: %w", err)
		}
	case "webhook":
		urlRule := URLRule{}
		if err := urlRule.Validate(value); err != nil {
			return fmt.Errorf("invalid webhook target: %w", err)
		}
	}

	return nil
}

// Security validation

// ValidateSecurityConfig validates security configuration
func ValidateSecurityConfig(config map[string]interface{}) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: make(map[string]string)}

	// Validate encryption settings
	if encEnabled, exists := config["encryption_enabled"]; exists {
		if enabled, ok := encEnabled.(bool); ok && enabled {
			if encKey, exists := config["encryption_key"]; exists {
				if key, ok := encKey.(string); ok {
					if len(key) < 32 {
						result.Valid = false
						result.Errors["encryption_key"] = "encryption key must be at least 32 characters"
					}
				} else {
					result.Valid = false
					result.Errors["encryption_key"] = "encryption key must be a string"
				}
			} else {
				result.Valid = false
				result.Errors["encryption_key"] = "encryption key is required when encryption is enabled"
			}
		}
	}

	return result
}

// Sanitization utilities

// SanitizeString removes dangerous characters from strings
func SanitizeString(input string) string {
	// Remove control characters and normalize whitespace
	var result strings.Builder
	for _, r := range input {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			continue // Skip control characters except common whitespace
		}
		result.WriteRune(r)
	}
	return strings.TrimSpace(result.String())
}

// SanitizeMap sanitizes string values in a map
func SanitizeMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		if str, ok := value.(string); ok {
			result[key] = SanitizeString(str)
		} else {
			result[key] = value
		}
	}
	return result
}
