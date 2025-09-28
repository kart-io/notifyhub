// Package target provides target resolution functionality for NotifyHub
package target

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// TargetResolver provides advanced target resolution capabilities
type TargetResolver struct {
	mux sync.RWMutex
	emailRegex *regexp.Regexp
	phoneRegex *regexp.Regexp
	urlRegex   *regexp.Regexp
	feishuUserRegex *regexp.Regexp
	feishuGroupRegex *regexp.Regexp
}

// NewTargetResolver creates a new target resolver with compiled regex patterns
func NewTargetResolver() *TargetResolver {
	return &TargetResolver{
		emailRegex:       regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
		phoneRegex:       regexp.MustCompile(`^\+[1-9]\d{1,14}$`),
		urlRegex:         regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`),
		feishuUserRegex:  regexp.MustCompile(`^(ou_|oc_)[a-zA-Z0-9_]{6,40}$`),
		feishuGroupRegex: regexp.MustCompile(`^og_[a-zA-Z0-9_]{6,40}$`),
	}
}

// DefaultResolver is the default target resolver instance
var DefaultResolver = NewTargetResolver()

// AutoDetectTarget automatically detects target type and platform based on value
func AutoDetectTarget(value string) Target {
	return DefaultResolver.AutoDetectTarget(value)
}

// AutoDetectTarget automatically detects target type and platform based on value
func (r *TargetResolver) AutoDetectTarget(value string) Target {
	targetType, platform := r.detectTargetTypeAndPlatform(value)
	standardizedValue := r.StandardizeTarget(targetType, value)
	return Target{
		Type:     targetType,
		Value:    standardizedValue,
		Platform: platform,
	}
}

// detectTargetTypeAndPlatform automatically detects target type and platform based on value
func (r *TargetResolver) detectTargetTypeAndPlatform(value string) (string, string) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	cleanValue := strings.TrimSpace(value)

	// Email detection using regex
	if r.emailRegex.MatchString(cleanValue) {
		return TargetTypeEmail, PlatformEmail
	}

	// Phone number detection using E.164 format
	if r.phoneRegex.MatchString(cleanValue) {
		return TargetTypePhone, PlatformSMS
	}

	// National phone number detection (common patterns)
	if r.isNationalPhoneNumber(cleanValue) {
		return TargetTypePhone, PlatformSMS
	}

	// Webhook URL detection with proper URL validation
	if r.urlRegex.MatchString(cleanValue) {
		return TargetTypeWebhook, PlatformWebhook
	}

	// Feishu user ID detection with proper pattern matching
	if r.feishuUserRegex.MatchString(cleanValue) {
		return TargetTypeUser, PlatformFeishu
	}

	// Feishu group ID detection with proper pattern matching
	if r.feishuGroupRegex.MatchString(cleanValue) {
		return TargetTypeGroup, PlatformFeishu
	}

	// Default to user on auto platform
	return TargetTypeUser, PlatformAuto
}

// isNationalPhoneNumber checks for common national phone number patterns
func (r *TargetResolver) isNationalPhoneNumber(value string) bool {
	// Remove common separators
	cleanedNumber := strings.ReplaceAll(value, "-", "")
	cleanedNumber = strings.ReplaceAll(cleanedNumber, " ", "")
	cleanedNumber = strings.ReplaceAll(cleanedNumber, "(", "")
	cleanedNumber = strings.ReplaceAll(cleanedNumber, ")", "")

	// Check for common patterns:
	// US: (123) 456-7890, 123-456-7890, 1234567890
	// China: 138-0013-8000, 13800138000
	// Generic: 10-15 digits
	nationalPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^1?[0-9]{10}$`),              // US format
		regexp.MustCompile(`^1[3-9][0-9]{9}$`),           // China mobile format
		regexp.MustCompile(`^[0-9]{10,15}$`),             // Generic international
	}

	for _, pattern := range nationalPatterns {
		if pattern.MatchString(cleanedNumber) {
			return true
		}
	}

	return false
}

// Validate validates the target configuration
func (t Target) Validate() error {
	if t.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}

	switch t.Type {
	case TargetTypeEmail:
		return t.validateEmail()
	case TargetTypePhone:
		return t.validatePhone()
	case TargetTypeWebhook:
		return t.validateWebhook()
	case TargetTypeUser, TargetTypeGroup, TargetTypeChannel:
		return t.validateID()
	default:
		return fmt.Errorf("unsupported target type: %s", t.Type)
	}
}

// IsEmail returns true if the target is an email
func (t Target) IsEmail() bool {
	return t.Type == TargetTypeEmail
}

// IsPhone returns true if the target is a phone number
func (t Target) IsPhone() bool {
	return t.Type == TargetTypePhone
}

// IsWebhook returns true if the target is a webhook
func (t Target) IsWebhook() bool {
	return t.Type == TargetTypeWebhook
}

// IsFeishu returns true if the target is for Feishu platform
func (t Target) IsFeishu() bool {
	return t.Platform == PlatformFeishu
}

// validateEmail validates email format
func (t Target) validateEmail() error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(t.Value) {
		return fmt.Errorf("invalid email format: %s", t.Value)
	}
	return nil
}

// validatePhone validates phone number format
func (t Target) validatePhone() error {
	// Basic phone number validation (starts with + followed by digits)
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(t.Value) {
		return fmt.Errorf("invalid phone number format: %s (expected format: +1234567890)", t.Value)
	}
	return nil
}

// validateWebhook validates webhook URL format
func (t Target) validateWebhook() error {
	if !strings.HasPrefix(t.Value, "http://") && !strings.HasPrefix(t.Value, "https://") {
		return fmt.Errorf("invalid webhook URL: %s (must start with http:// or https://)", t.Value)
	}
	return nil
}

// validateID validates user/group/channel ID format
func (t Target) validateID() error {
	if len(t.Value) < 2 {
		return fmt.Errorf("invalid ID format: %s (too short)", t.Value)
	}

	// Check for invalid characters that would suggest this is not an ID
	if strings.HasPrefix(t.Value, "@") || strings.HasPrefix(t.Value, "+") {
		return fmt.Errorf("invalid ID format: %s (contains invalid prefix)", t.Value)
	}

	return nil
}

// StandardizeTarget standardizes target values based on type
func (r *TargetResolver) StandardizeTarget(targetType, value string) string {
	switch targetType {
	case TargetTypeEmail:
		return r.standardizeEmail(value)
	case TargetTypePhone:
		return r.standardizePhone(value)
	case TargetTypeWebhook:
		return r.standardizeURL(value)
	default:
		return strings.TrimSpace(value)
	}
}

// standardizeEmail normalizes email addresses
func (r *TargetResolver) standardizeEmail(email string) string {
	// Convert to lowercase and trim whitespace
	email = strings.ToLower(strings.TrimSpace(email))

	// Gmail specific normalization (remove dots before @)
	if strings.HasSuffix(email, "@gmail.com") {
		parts := strings.Split(email, "@")
		if len(parts) == 2 {
			// Remove dots from local part (before @)
			localPart := strings.ReplaceAll(parts[0], ".", "")
			// Remove everything after + (gmail alias)
			if plusIndex := strings.Index(localPart, "+"); plusIndex != -1 {
				localPart = localPart[:plusIndex]
			}
			email = localPart + "@gmail.com"
		}
	}

	return email
}

// standardizePhone converts phone numbers to E.164 format
func (r *TargetResolver) standardizePhone(phone string) string {
	// If already in E.164 format, return as is
	if strings.HasPrefix(phone, "+") && r.phoneRegex.MatchString(phone) {
		return phone
	}

	// Remove common separators
	cleanedPhone := strings.ReplaceAll(phone, "-", "")
	cleanedPhone = strings.ReplaceAll(cleanedPhone, " ", "")
	cleanedPhone = strings.ReplaceAll(cleanedPhone, "(", "")
	cleanedPhone = strings.ReplaceAll(cleanedPhone, ")", "")
	cleanedPhone = strings.ReplaceAll(cleanedPhone, ".", "")

	// Add country code for common formats
	if len(cleanedPhone) == 10 && !strings.HasPrefix(cleanedPhone, "+") {
		// Assume US number if 10 digits
		return "+1" + cleanedPhone
	}
	if len(cleanedPhone) == 11 && strings.HasPrefix(cleanedPhone, "1") {
		// US number with country code but no +
		return "+" + cleanedPhone
	}
	if len(cleanedPhone) == 11 && strings.HasPrefix(cleanedPhone, "86") {
		// China number with country code but no +
		return "+" + cleanedPhone
	}

	// Return original if can't standardize
	return phone
}

// standardizeURL normalizes webhook URLs
func (r *TargetResolver) standardizeURL(rawURL string) string {
	// Parse and reconstruct URL to ensure proper format
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return rawURL // Return original if can't parse
	}

	// Ensure scheme is present
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
	}

	// Normalize scheme to lowercase
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)

	// Normalize host to lowercase
	parsedURL.Host = strings.ToLower(parsedURL.Host)

	return parsedURL.String()
}

// ResolveBatch resolves multiple target values and returns deduplicated targets
func (r *TargetResolver) ResolveBatch(values []string) ([]Target, []error) {
	if len(values) == 0 {
		return nil, nil
	}

	targetMap := make(map[string]Target) // Use map for deduplication
	var errors []error

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue // Skip empty values
		}

		target := r.AutoDetectTarget(value)

		// Validate the target
		if err := target.Validate(); err != nil {
			errors = append(errors, fmt.Errorf("invalid target '%s': %w", value, err))
			continue
		}

		// Use standardized value as key for deduplication
		key := fmt.Sprintf("%s:%s@%s", target.Type, target.Value, target.Platform)
		targetMap[key] = target
	}

	// Convert map back to slice
	targets := make([]Target, 0, len(targetMap))
	for _, target := range targetMap {
		targets = append(targets, target)
	}

	return targets, errors
}

// ResolveBatch is a convenience function using the default resolver
func ResolveBatch(values []string) ([]Target, []error) {
	return DefaultResolver.ResolveBatch(values)
}

// ValidatePlatformCompatibility checks if target is compatible with platform
func (r *TargetResolver) ValidatePlatformCompatibility(target Target, platform string) error {
	if target.Platform == PlatformAuto {
		return nil // Auto platform is always compatible
	}

	if target.Platform != platform && platform != PlatformAuto {
		return fmt.Errorf("target platform '%s' is not compatible with requested platform '%s'", target.Platform, platform)
	}

	return nil
}

// GetTargetReachabilityHint provides hints about target reachability
func (r *TargetResolver) GetTargetReachabilityHint(target Target) string {
	switch target.Type {
	case TargetTypeEmail:
		if strings.HasSuffix(target.Value, "@gmail.com") ||
		   strings.HasSuffix(target.Value, "@outlook.com") ||
		   strings.HasSuffix(target.Value, "@yahoo.com") {
			return "high_reliability" // Major email providers
		}
		return "medium_reliability"
	case TargetTypePhone:
		if strings.HasPrefix(target.Value, "+1") ||
		   strings.HasPrefix(target.Value, "+86") {
			return "high_reliability" // Major countries with good SMS infrastructure
		}
		return "medium_reliability"
	case TargetTypeWebhook:
		if strings.Contains(target.Value, "localhost") ||
		   strings.Contains(target.Value, "127.0.0.1") {
			return "test_environment"
		}
		if strings.HasPrefix(target.Value, "https://") {
			return "high_reliability"
		}
		return "medium_reliability"
	case TargetTypeUser, TargetTypeGroup:
		if target.Platform == PlatformFeishu {
			return "high_reliability" // Feishu has good delivery rates
		}
		return "medium_reliability"
	default:
		return "unknown"
	}
}
