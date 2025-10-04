// Package webhook provides webhook platform configuration for NotifyHub
package webhook

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Config represents webhook platform configuration
type Config struct {
	// Webhook settings
	WebhookURL string `json:"webhook_url" yaml:"webhook_url"`
	Method     string `json:"method" yaml:"method"` // HTTP method (GET, POST, PUT, PATCH)

	// Authentication
	Secret       string `json:"secret,omitempty" yaml:"secret,omitempty"`                 // Webhook secret for signatures
	AuthType     string `json:"auth_type,omitempty" yaml:"auth_type,omitempty"`           // "none", "basic", "bearer", "api_key", "signature"
	Username     string `json:"username,omitempty" yaml:"username,omitempty"`             // For basic auth
	Password     string `json:"password,omitempty" yaml:"password,omitempty"`             // For basic auth
	BearerToken  string `json:"bearer_token,omitempty" yaml:"bearer_token,omitempty"`     // For bearer auth
	APIKey       string `json:"api_key,omitempty" yaml:"api_key,omitempty"`               // For API key auth
	APIKeyHeader string `json:"api_key_header,omitempty" yaml:"api_key_header,omitempty"` // Header name for API key

	// Headers
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty" yaml:"user_agent,omitempty"`
	ContentType string            `json:"content_type,omitempty" yaml:"content_type,omitempty"`

	// Connection settings
	Timeout      *time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	MaxRetries   *int           `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
	RetryDelay   *time.Duration `json:"retry_delay,omitempty" yaml:"retry_delay,omitempty"`
	KeepAlive    bool           `json:"keep_alive,omitempty" yaml:"keep_alive,omitempty"`
	MaxIdleConns int            `json:"max_idle_conns,omitempty" yaml:"max_idle_conns,omitempty"`

	// TLS settings
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty" yaml:"insecure_skip_verify,omitempty"`
	CertFile           string `json:"cert_file,omitempty" yaml:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty" yaml:"key_file,omitempty"`
	CAFile             string `json:"ca_file,omitempty" yaml:"ca_file,omitempty"`

	// Proxy settings
	ProxyURL      string `json:"proxy_url,omitempty" yaml:"proxy_url,omitempty"`
	ProxyUsername string `json:"proxy_username,omitempty" yaml:"proxy_username,omitempty"`
	ProxyPassword string `json:"proxy_password,omitempty" yaml:"proxy_password,omitempty"`

	// Payload settings
	PayloadFormat   string            `json:"payload_format,omitempty" yaml:"payload_format,omitempty"`     // "json", "form", "xml"
	PayloadTemplate string            `json:"payload_template,omitempty" yaml:"payload_template,omitempty"` // Custom payload template
	FieldMapping    map[string]string `json:"field_mapping,omitempty" yaml:"field_mapping,omitempty"`       // Field name mapping

	// Response settings
	ExpectedStatus   []int          `json:"expected_status,omitempty" yaml:"expected_status,omitempty"` // Expected HTTP status codes
	ResponseTimeout  *time.Duration `json:"response_timeout,omitempty" yaml:"response_timeout,omitempty"`
	IgnoreResponse   bool           `json:"ignore_response,omitempty" yaml:"ignore_response,omitempty"`
	ValidateResponse bool           `json:"validate_response,omitempty" yaml:"validate_response,omitempty"`

	// Rate limiting
	RateLimit       int            `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`   // requests per minute
	BurstLimit      int            `json:"burst_limit,omitempty" yaml:"burst_limit,omitempty"` // burst capacity
	RateLimitWindow *time.Duration `json:"rate_limit_window,omitempty" yaml:"rate_limit_window,omitempty"`

	// Signature settings (for webhook signature verification)
	SignatureHeader string `json:"signature_header,omitempty" yaml:"signature_header,omitempty"` // Header name for signature
	SignatureAlgo   string `json:"signature_algo,omitempty" yaml:"signature_algo,omitempty"`     // "sha1", "sha256", "md5"
	SignaturePrefix string `json:"signature_prefix,omitempty" yaml:"signature_prefix,omitempty"` // Signature prefix (e.g., "sha256=")

	// Advanced settings
	FollowRedirects bool `json:"follow_redirects,omitempty" yaml:"follow_redirects,omitempty"`
	MaxRedirects    int  `json:"max_redirects,omitempty" yaml:"max_redirects,omitempty"`
	Compression     bool `json:"compression,omitempty" yaml:"compression,omitempty"`
}

// NewConfig creates a new webhook configuration with defaults
func NewConfig() *Config {
	return &Config{
		Method:          "POST",
		AuthType:        "none",
		ContentType:     "application/json",
		UserAgent:       "NotifyHub-Webhook/1.0",
		KeepAlive:       true,
		MaxIdleConns:    10,
		PayloadFormat:   "json",
		ExpectedStatus:  []int{200, 201, 202, 204},
		FollowRedirects: true,
		MaxRedirects:    10,
		Compression:     true,
		RateLimit:       60, // 60 requests per minute by default
		BurstLimit:      10, // burst of 10 requests
		SignatureAlgo:   "sha256",
		SignatureHeader: "X-Signature",
		SignaturePrefix: "sha256=",
	}
}

// Validate validates the webhook configuration
func (c *Config) Validate() error {
	// Validate required fields
	if c.WebhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}

	// Validate URL format
	if _, err := url.Parse(c.WebhookURL); err != nil {
		return fmt.Errorf("invalid webhook_url: %w", err)
	}

	// Validate HTTP method
	validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	methodValid := false
	for _, method := range validMethods {
		if strings.ToUpper(c.Method) == method {
			methodValid = true
			break
		}
	}
	if !methodValid {
		return fmt.Errorf("invalid method: %s (valid: %s)", c.Method, strings.Join(validMethods, ", "))
	}

	// Validate auth type
	validAuthTypes := []string{"none", "basic", "bearer", "api_key", "signature"}
	authValid := false
	for _, authType := range validAuthTypes {
		if c.AuthType == authType {
			authValid = true
			break
		}
	}
	if !authValid {
		return fmt.Errorf("invalid auth_type: %s (valid: %s)", c.AuthType, strings.Join(validAuthTypes, ", "))
	}

	// Validate auth configuration
	switch c.AuthType {
	case "basic":
		if c.Username == "" || c.Password == "" {
			return fmt.Errorf("username and password are required for basic auth")
		}
	case "bearer":
		if c.BearerToken == "" {
			return fmt.Errorf("bearer_token is required for bearer auth")
		}
	case "api_key":
		if c.APIKey == "" {
			return fmt.Errorf("api_key is required for api_key auth")
		}
		if c.APIKeyHeader == "" {
			c.APIKeyHeader = "X-API-Key" // default
		}
	case "signature":
		if c.Secret == "" {
			return fmt.Errorf("secret is required for signature auth")
		}
	}

	// Validate payload format
	if c.PayloadFormat != "" {
		validFormats := []string{"json", "form", "xml"}
		formatValid := false
		for _, format := range validFormats {
			if c.PayloadFormat == format {
				formatValid = true
				break
			}
		}
		if !formatValid {
			return fmt.Errorf("invalid payload_format: %s (valid: %s)", c.PayloadFormat, strings.Join(validFormats, ", "))
		}
	}

	// Validate signature algorithm
	if c.SignatureAlgo != "" {
		validAlgos := []string{"sha1", "sha256", "md5"}
		algoValid := false
		for _, algo := range validAlgos {
			if c.SignatureAlgo == algo {
				algoValid = true
				break
			}
		}
		if !algoValid {
			return fmt.Errorf("invalid signature_algo: %s (valid: %s)", c.SignatureAlgo, strings.Join(validAlgos, ", "))
		}
	}

	// Validate expected status codes
	if len(c.ExpectedStatus) == 0 {
		c.ExpectedStatus = []int{200, 201, 202, 204}
	}

	for _, status := range c.ExpectedStatus {
		if status < 100 || status >= 600 {
			return fmt.Errorf("invalid expected status code: %d", status)
		}
	}

	// Apply defaults for optional fields
	if c.Timeout == nil {
		timeout := 30 * time.Second
		c.Timeout = &timeout
	}

	if c.MaxRetries == nil {
		retries := 3
		c.MaxRetries = &retries
	}

	if c.RetryDelay == nil {
		delay := time.Second
		c.RetryDelay = &delay
	}

	if c.ResponseTimeout == nil {
		responseTimeout := 30 * time.Second
		c.ResponseTimeout = &responseTimeout
	}

	if c.RateLimitWindow == nil {
		window := time.Minute
		c.RateLimitWindow = &window
	}

	return nil
}

// GetTimeout returns the timeout duration
func (c *Config) GetTimeout() time.Duration {
	if c.Timeout != nil {
		return *c.Timeout
	}
	return 30 * time.Second
}

// GetMaxRetries returns the maximum retry attempts
func (c *Config) GetMaxRetries() int {
	if c.MaxRetries != nil {
		return *c.MaxRetries
	}
	return 3
}

// GetRetryDelay returns the retry delay
func (c *Config) GetRetryDelay() time.Duration {
	if c.RetryDelay != nil {
		return *c.RetryDelay
	}
	return time.Second
}

// GetResponseTimeout returns the response timeout
func (c *Config) GetResponseTimeout() time.Duration {
	if c.ResponseTimeout != nil {
		return *c.ResponseTimeout
	}
	return 30 * time.Second
}

// GetRateLimitWindow returns the rate limit window
func (c *Config) GetRateLimitWindow() time.Duration {
	if c.RateLimitWindow != nil {
		return *c.RateLimitWindow
	}
	return time.Minute
}

// IsAuthRequired returns true if authentication is configured
func (c *Config) IsAuthRequired() bool {
	return c.AuthType != "none" && c.AuthType != ""
}

// IsSignatureRequired returns true if signature verification is required
func (c *Config) IsSignatureRequired() bool {
	return c.AuthType == "signature" && c.Secret != ""
}

// GetEffectiveHeaders returns all effective headers
func (c *Config) GetEffectiveHeaders() map[string]string {
	headers := make(map[string]string)

	// Set default headers
	if c.ContentType != "" {
		headers["Content-Type"] = c.ContentType
	}
	if c.UserAgent != "" {
		headers["User-Agent"] = c.UserAgent
	}

	// Add custom headers
	for k, v := range c.Headers {
		headers[k] = v
	}

	// Add auth headers
	switch c.AuthType {
	case "bearer":
		if c.BearerToken != "" {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", c.BearerToken)
		}
	case "api_key":
		if c.APIKey != "" && c.APIKeyHeader != "" {
			headers[c.APIKeyHeader] = c.APIKey
		}
	}

	return headers
}

// IsStatusExpected returns true if the status code is expected
func (c *Config) IsStatusExpected(statusCode int) bool {
	for _, expected := range c.ExpectedStatus {
		if statusCode == expected {
			return true
		}
	}
	return false
}

// Clone creates a copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c

	// Deep copy pointers
	if c.Timeout != nil {
		timeout := *c.Timeout
		clone.Timeout = &timeout
	}

	if c.MaxRetries != nil {
		retries := *c.MaxRetries
		clone.MaxRetries = &retries
	}

	if c.RetryDelay != nil {
		delay := *c.RetryDelay
		clone.RetryDelay = &delay
	}

	if c.ResponseTimeout != nil {
		responseTimeout := *c.ResponseTimeout
		clone.ResponseTimeout = &responseTimeout
	}

	if c.RateLimitWindow != nil {
		window := *c.RateLimitWindow
		clone.RateLimitWindow = &window
	}

	// Deep copy maps and slices
	if c.Headers != nil {
		clone.Headers = make(map[string]string)
		for k, v := range c.Headers {
			clone.Headers[k] = v
		}
	}

	if c.FieldMapping != nil {
		clone.FieldMapping = make(map[string]string)
		for k, v := range c.FieldMapping {
			clone.FieldMapping[k] = v
		}
	}

	if c.ExpectedStatus != nil {
		clone.ExpectedStatus = make([]int, len(c.ExpectedStatus))
		copy(clone.ExpectedStatus, c.ExpectedStatus)
	}

	return &clone
}

// ToMap converts the configuration to a map
func (c *Config) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"webhook_url":    c.WebhookURL,
		"method":         c.Method,
		"auth_type":      c.AuthType,
		"content_type":   c.ContentType,
		"user_agent":     c.UserAgent,
		"payload_format": c.PayloadFormat,
		"rate_limit":     c.RateLimit,
		"burst_limit":    c.BurstLimit,
	}

	// Add optional fields if set
	if c.Secret != "" {
		result["secret"] = "[REDACTED]"
	}
	if c.Username != "" {
		result["username"] = c.Username
	}
	if c.BearerToken != "" {
		result["bearer_token"] = "[REDACTED]"
	}
	if c.APIKey != "" {
		result["api_key"] = "[REDACTED]"
	}
	if c.Timeout != nil {
		result["timeout"] = c.Timeout.String()
	}
	if c.MaxRetries != nil {
		result["max_retries"] = *c.MaxRetries
	}
	if c.Headers != nil {
		result["headers"] = c.Headers
	}
	if c.ExpectedStatus != nil {
		result["expected_status"] = c.ExpectedStatus
	}

	return result
}

// Predefined configurations for common webhook platforms

// SlackConfig returns a configuration for Slack webhooks
func SlackConfig(webhookURL string) *Config {
	config := NewConfig()
	config.WebhookURL = webhookURL
	config.Method = "POST"
	config.ContentType = "application/json"
	config.PayloadFormat = "json"
	config.ExpectedStatus = []int{200}
	return config
}

// DiscordConfig returns a configuration for Discord webhooks
func DiscordConfig(webhookURL string) *Config {
	config := NewConfig()
	config.WebhookURL = webhookURL
	config.Method = "POST"
	config.ContentType = "application/json"
	config.PayloadFormat = "json"
	config.ExpectedStatus = []int{204}
	return config
}

// TeamsConfig returns a configuration for Microsoft Teams webhooks
func TeamsConfig(webhookURL string) *Config {
	config := NewConfig()
	config.WebhookURL = webhookURL
	config.Method = "POST"
	config.ContentType = "application/json"
	config.PayloadFormat = "json"
	config.ExpectedStatus = []int{200}
	return config
}

// GenericConfig returns a generic webhook configuration
func GenericConfig(webhookURL string) *Config {
	config := NewConfig()
	config.WebhookURL = webhookURL
	return config
}
