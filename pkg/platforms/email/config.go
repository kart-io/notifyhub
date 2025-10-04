// Package email provides email platform configuration for NotifyHub
package email

import (
	"fmt"
	"time"
)

// Config represents email platform configuration
type Config struct {
	// SMTP server settings
	SMTPHost string `json:"smtp_host" yaml:"smtp_host"`
	SMTPPort int    `json:"smtp_port" yaml:"smtp_port"`

	// Authentication
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`

	// Email settings
	From       string `json:"from" yaml:"from"`
	FromName   string `json:"from_name,omitempty" yaml:"from_name,omitempty"`
	ReplyTo    string `json:"reply_to,omitempty" yaml:"reply_to,omitempty"`
	ReturnPath string `json:"return_path,omitempty" yaml:"return_path,omitempty"`

	// Security settings
	UseTLS         bool `json:"use_tls" yaml:"use_tls"`
	UseStartTLS    bool `json:"use_starttls" yaml:"use_starttls"`
	SkipCertVerify bool `json:"skip_cert_verify,omitempty" yaml:"skip_cert_verify,omitempty"`

	// Connection settings
	Timeout      *time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	MaxRetries   *int           `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
	KeepAlive    bool           `json:"keep_alive,omitempty" yaml:"keep_alive,omitempty"`
	PoolSize     int            `json:"pool_size,omitempty" yaml:"pool_size,omitempty"`
	MaxIdleConns int            `json:"max_idle_conns,omitempty" yaml:"max_idle_conns,omitempty"`

	// Message settings
	DefaultSubject string            `json:"default_subject,omitempty" yaml:"default_subject,omitempty"`
	Headers        map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Encoding       string            `json:"encoding,omitempty" yaml:"encoding,omitempty"`
	ContentType    string            `json:"content_type,omitempty" yaml:"content_type,omitempty"`

	// Template settings
	TemplateDir     string `json:"template_dir,omitempty" yaml:"template_dir,omitempty"`
	DefaultTemplate string `json:"default_template,omitempty" yaml:"default_template,omitempty"`

	// Rate limiting
	RateLimit       int            `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`   // emails per minute
	BurstLimit      int            `json:"burst_limit,omitempty" yaml:"burst_limit,omitempty"` // burst capacity
	RateLimitWindow *time.Duration `json:"rate_limit_window,omitempty" yaml:"rate_limit_window,omitempty"`

	// Advanced settings
	LocalName   string `json:"local_name,omitempty" yaml:"local_name,omitempty"`
	Helo        string `json:"helo,omitempty" yaml:"helo,omitempty"`
	AuthMethod  string `json:"auth_method,omitempty" yaml:"auth_method,omitempty"` // "plain", "login", "cram-md5"
	DSN         bool   `json:"dsn,omitempty" yaml:"dsn,omitempty"`                 // Delivery Status Notification
	TrackOpens  bool   `json:"track_opens,omitempty" yaml:"track_opens,omitempty"`
	TrackClicks bool   `json:"track_clicks,omitempty" yaml:"track_clicks,omitempty"`
}

// NewConfig creates a new email configuration with defaults
func NewConfig() *Config {
	return &Config{
		SMTPPort:     587,
		UseTLS:       true,
		UseStartTLS:  true,
		KeepAlive:    true,
		PoolSize:     10,
		MaxIdleConns: 5,
		Encoding:     "UTF-8",
		ContentType:  "text/html",
		AuthMethod:   "plain",
		RateLimit:    60, // 60 emails per minute by default
		BurstLimit:   10, // burst of 10 emails
	}
}

// Validate validates the email configuration
func (c *Config) Validate() error {
	// Validate required fields
	if c.SMTPHost == "" {
		return fmt.Errorf("smtp_host is required")
	}

	if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
		return fmt.Errorf("smtp_port must be between 1 and 65535")
	}

	if c.From == "" {
		return fmt.Errorf("from email address is required")
	}

	// Validate authentication if provided
	if c.Username != "" || c.Password != "" {
		if c.Username == "" {
			return fmt.Errorf("username is required when password is provided")
		}
		if c.Password == "" {
			return fmt.Errorf("password is required when username is provided")
		}
	}

	// Validate auth method
	if c.AuthMethod != "" {
		validMethods := []string{"plain", "login", "cram-md5"}
		valid := false
		for _, method := range validMethods {
			if c.AuthMethod == method {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid auth_method: %s (valid: plain, login, cram-md5)", c.AuthMethod)
		}
	}

	// Validate encoding
	if c.Encoding != "" {
		validEncodings := []string{"UTF-8", "ISO-8859-1", "US-ASCII"}
		valid := false
		for _, encoding := range validEncodings {
			if c.Encoding == encoding {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid encoding: %s", c.Encoding)
		}
	}

	// Validate content type
	if c.ContentType != "" {
		validTypes := []string{"text/plain", "text/html"}
		valid := false
		for _, contentType := range validTypes {
			if c.ContentType == contentType {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid content_type: %s", c.ContentType)
		}
	}

	// Validate rate limiting
	if c.RateLimit < 0 {
		return fmt.Errorf("rate_limit cannot be negative")
	}

	if c.BurstLimit < 0 {
		return fmt.Errorf("burst_limit cannot be negative")
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

// GetRateLimitWindow returns the rate limit window
func (c *Config) GetRateLimitWindow() time.Duration {
	if c.RateLimitWindow != nil {
		return *c.RateLimitWindow
	}
	return time.Minute
}

// IsAuthRequired returns true if authentication is configured
func (c *Config) IsAuthRequired() bool {
	return c.Username != "" && c.Password != ""
}

// GetServerAddress returns the complete server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.SMTPHost, c.SMTPPort)
}

// GetFromAddress returns the formatted from address
func (c *Config) GetFromAddress() string {
	if c.FromName != "" {
		return fmt.Sprintf("%s <%s>", c.FromName, c.From)
	}
	return c.From
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

	if c.RateLimitWindow != nil {
		window := *c.RateLimitWindow
		clone.RateLimitWindow = &window
	}

	// Deep copy maps
	if c.Headers != nil {
		clone.Headers = make(map[string]string)
		for k, v := range c.Headers {
			clone.Headers[k] = v
		}
	}

	return &clone
}

// ToMap converts the configuration to a map
func (c *Config) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"smtp_host":    c.SMTPHost,
		"smtp_port":    c.SMTPPort,
		"from":         c.From,
		"use_tls":      c.UseTLS,
		"use_starttls": c.UseStartTLS,
		"keep_alive":   c.KeepAlive,
		"encoding":     c.Encoding,
		"content_type": c.ContentType,
		"auth_method":  c.AuthMethod,
		"rate_limit":   c.RateLimit,
		"burst_limit":  c.BurstLimit,
	}

	// Add optional fields if set
	if c.FromName != "" {
		result["from_name"] = c.FromName
	}
	if c.ReplyTo != "" {
		result["reply_to"] = c.ReplyTo
	}
	if c.Username != "" {
		result["username"] = c.Username
	}
	if c.DefaultSubject != "" {
		result["default_subject"] = c.DefaultSubject
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

	return result
}

// Predefined configurations for common email providers

// GmailConfig returns a configuration for Gmail SMTP
func GmailConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.gmail.com"
	config.SMTPPort = 587
	config.Username = username
	config.Password = password
	config.UseTLS = true
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// OutlookConfig returns a configuration for Outlook SMTP
func OutlookConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp-mail.outlook.com"
	config.SMTPPort = 587
	config.Username = username
	config.Password = password
	config.UseTLS = true
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// SendGridConfig returns a configuration for SendGrid SMTP
func SendGridConfig(apiKey string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.sendgrid.net"
	config.SMTPPort = 587
	config.Username = "apikey"
	config.Password = apiKey
	config.UseTLS = true
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// NetEase163Config returns a configuration for 163.com SMTP
func NetEase163Config(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.163.com"
	config.SMTPPort = 25  // 163 supports both 25 and 587, 25 is more common for 163
	config.Username = username
	config.Password = password
	config.UseTLS = false      // 163 typically uses STARTTLS on port 25
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// NetEase126Config returns a configuration for 126.com SMTP
func NetEase126Config(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.126.com"
	config.SMTPPort = 25
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// NetEaseYeahConfig returns a configuration for yeah.net SMTP
func NetEaseYeahConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.yeah.net"
	config.SMTPPort = 25
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// QQMailConfig returns a configuration for QQ Mail SMTP
func QQMailConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.qq.com"
	config.SMTPPort = 587
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// SinaMailConfig returns a configuration for Sina Mail SMTP
func SinaMailConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.sina.com"
	config.SMTPPort = 25
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// SohuMailConfig returns a configuration for Sohu Mail SMTP
func SohuMailConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.sohu.com"
	config.SMTPPort = 25
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// YahooConfig returns a configuration for Yahoo Mail SMTP
func YahooConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.mail.yahoo.com"
	config.SMTPPort = 587
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// YahooJPConfig returns a configuration for Yahoo Japan SMTP
func YahooJPConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.mail.yahoo.co.jp"
	config.SMTPPort = 587
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// ZohoConfig returns a configuration for Zoho Mail SMTP
func ZohoConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.zoho.com"
	config.SMTPPort = 587
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// ProtonMailConfig returns a configuration for ProtonMail SMTP
func ProtonMailConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "127.0.0.1" // ProtonMail Bridge
	config.SMTPPort = 1025
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// AliMailConfig returns a configuration for Alibaba Mail SMTP (企业邮箱)
func AliMailConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.mxhichina.com"
	config.SMTPPort = 587
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}

// TencentExmailConfig returns a configuration for Tencent Enterprise Mail SMTP
func TencentExmailConfig(username, password string) *Config {
	config := NewConfig()
	config.SMTPHost = "smtp.exmail.qq.com"
	config.SMTPPort = 587
	config.Username = username
	config.Password = password
	config.UseTLS = false
	config.UseStartTLS = true
	config.AuthMethod = "plain"
	return config
}
