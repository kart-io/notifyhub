// Package notifyhub provides the public API for the NotifyHub notification system.
//
// This package serves as the main entry point for creating and using NotifyHub instances.
// It provides backward compatibility adapters and convenient factory functions for
// creating notification hubs with various platform configurations.
package notifyhub

import (
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Type aliases for backward compatibility
type (
	Hub            = core.Hub
	HubConfig      = config.HubConfig
	PlatformConfig = config.PlatformConfig
	RetryPolicy    = config.RetryPolicy
	Message        = message.Message
	MessageBuilder = message.MessageBuilder
	Priority       = message.Priority
	Target         = target.Target
	Receipt        = receipt.Receipt
	AsyncReceipt   = receipt.AsyncReceipt
	PlatformResult = receipt.PlatformResult
	HealthStatus   = core.HealthStatus
	PlatformHealth = core.PlatformHealth
	QueueHealth    = core.QueueHealth
)

// Constants for backward compatibility
const (
	PriorityLow    = message.PriorityLow
	PriorityNormal = message.PriorityNormal
	PriorityHigh   = message.PriorityHigh
	PriorityUrgent = message.PriorityUrgent

	TargetTypeEmail   = target.TargetTypeEmail
	TargetTypePhone   = target.TargetTypePhone
	TargetTypeUser    = target.TargetTypeUser
	TargetTypeGroup   = target.TargetTypeGroup
	TargetTypeChannel = target.TargetTypeChannel
	TargetTypeWebhook = target.TargetTypeWebhook

	PlatformFeishu  = target.PlatformFeishu
	PlatformEmail   = target.PlatformEmail
	PlatformSMS     = target.PlatformSMS
	PlatformWebhook = target.PlatformWebhook
	PlatformAuto    = target.PlatformAuto
)

// HubOption represents a configuration option for creating a hub.
// HubOptions are used with the functional options pattern to configure
// notification hubs in a flexible and extensible way.
//
// Example:
//
//	hub, err := NewHub(
//	    WithFeishu("webhook-url", "secret"),
//	    WithTimeout(30*time.Second),
//	)
type HubOption func(*HubConfig)

// NewHub creates a new notification hub with the given configuration options.
//
// NewHub uses the functional options pattern to provide a flexible way to
// configure notification hubs. It sets up default configurations and then
// applies the provided options.
//
// Parameters:
//   - opts: Variable number of HubOption functions to configure the hub
//
// Returns:
//   - Hub: The configured notification hub instance
//   - error: Any error that occurred during hub creation
//
// Example:
//
//	// Create a hub with Feishu platform
//	hub, err := NewHub(
//	    WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook", "your-secret"),
//	    WithTimeout(30*time.Second),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer hub.Close(context.Background())
//
//	// Create a hub with multiple platforms
//	hub, err := NewHub(
//	    WithFeishu("webhook-url", "secret"),
//	    WithEmail("smtp.example.com", 587, "user", "pass", "from@example.com", true, 30*time.Second),
//	)
func NewHub(opts ...HubOption) (Hub, error) {
	cfg := &HubConfig{
		Platforms:        make(map[string]PlatformConfig),
		DefaultTimeout:   30 * time.Second,
		ValidationErrors: make([]error, 0),
		RetryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			Multiplier:      2.0,
			MaxInterval:     30 * time.Second,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Check for validation errors
	if len(cfg.ValidationErrors) > 0 {
		return nil, fmt.Errorf("configuration validation failed: %v", cfg.ValidationErrors)
	}

	// Create real hub implementation
	return core.NewHub(cfg)
}

// Configuration option functions

// WithFeishu configures the Feishu (Lark) platform with a webhook URL and optional secret.
//
// Feishu is a popular enterprise collaboration platform. This function sets up
// the necessary configuration to send notifications through Feishu webhooks.
//
// Parameters:
//   - webhookURL: The Feishu webhook URL obtained from bot configuration
//   - secret: Optional secret key for webhook signature verification (can be empty)
//
// Returns:
//   - HubOption: Configuration function that can be passed to NewHub
//
// Example:
//
//	hub, err := NewHub(
//	    WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook", "your-secret"),
//	)
func WithFeishu(webhookURL, secret string) HubOption {
	return func(cfg *HubConfig) {
		cfg.Platforms["feishu"] = PlatformConfig{
			"webhook_url": webhookURL,
			"secret":      secret,
		}
	}
}

// WithFeishuFromMap configures Feishu platform from a map
func WithFeishuFromMap(settings map[string]interface{}) HubOption {
	return func(cfg *HubConfig) {
		cfg.Platforms["feishu"] = PlatformConfig(settings)
	}
}

// WithFeishuKeywords configures Feishu platform with keywords
func WithFeishuKeywords(keywords []string) HubOption {
	return func(cfg *HubConfig) {
		cfg.Platforms["feishu"] = PlatformConfig{
			"keywords": keywords,
		}
	}
}

// WithEmail configures the email platform with SMTP settings.
//
// This function sets up email notification capabilities using SMTP protocol.
// It supports both TLS and non-TLS connections.
//
// Parameters:
//   - host: SMTP server hostname (e.g., "smtp.gmail.com")
//   - port: SMTP server port (e.g., 587 for TLS, 25 for non-TLS)
//   - username: SMTP authentication username
//   - password: SMTP authentication password
//   - from: Default sender email address
//   - useTLS: Whether to use TLS encryption
//   - timeout: Connection timeout duration
//
// Returns:
//   - HubOption: Configuration function that can be passed to NewHub
//
// Example:
//
//	hub, err := NewHub(
//	    WithEmail("smtp.gmail.com", 587, "user@gmail.com", "password", "notifications@company.com", true, 30*time.Second),
//	)
func WithEmail(host string, port int, username, password, from string, useTLS bool, timeout time.Duration) HubOption {
	return func(cfg *HubConfig) {
		cfg.Platforms["email"] = PlatformConfig{
			"smtp_host":     host,
			"smtp_port":     port,
			"smtp_username": username,
			"smtp_password": password,
			"smtp_from":     from,
			"smtp_tls":      useTLS,
			"timeout":       timeout,
		}
	}
}

// WithEmailFromMap configures email platform from a map
func WithEmailFromMap(settings map[string]interface{}) HubOption {
	return func(cfg *HubConfig) {
		cfg.Platforms["email"] = PlatformConfig(settings)
	}
}

// WithSMS configures SMS platform
func WithSMS(provider, apiKey, from string) HubOption {
	return func(cfg *HubConfig) {
		cfg.Platforms["sms"] = PlatformConfig{
			"provider": provider,
			"api_key":  apiKey,
			"from":     from,
		}
	}
}

// WithSMSFromMap configures SMS platform from a map
func WithSMSFromMap(settings map[string]interface{}) HubOption {
	return func(cfg *HubConfig) {
		cfg.Platforms["sms"] = PlatformConfig(settings)
	}
}

// WithTimeout sets the default timeout
func WithTimeout(timeout time.Duration) HubOption {
	return func(cfg *HubConfig) {
		cfg.DefaultTimeout = timeout
	}
}

// WithPlatformConfig configures a platform with generic configuration
func WithPlatformConfig(platformName string, config map[string]interface{}) HubOption {
	return func(cfg *HubConfig) {
		cfg.Platforms[platformName] = PlatformConfig(config)
	}
}

// WithTestDefaults configures the hub with test defaults
func WithTestDefaults() HubOption {
	return func(cfg *HubConfig) {
		cfg.Platforms["feishu"] = PlatformConfig{
			"webhook_url": "https://example.com/webhook",
			"secret":      "test-secret",
		}
		cfg.Platforms["email"] = PlatformConfig{
			"smtp_host": "localhost",
			"smtp_port": 587,
		}
		cfg.Platforms["sms"] = PlatformConfig{
			"provider": "test",
		}
	}
}

// Message builder functions provide convenient ways to create notification messages.

// NewMessage creates a new message builder with an optional title.
//
// The message builder provides a fluent interface for constructing notification
// messages with various properties like body text, priority, targets, and metadata.
//
// Parameters:
//   - title: Optional message title. If not provided, defaults to empty string
//
// Returns:
//   - *MessageBuilder: A message builder instance for fluent configuration
//
// Example:
//
//	msg := NewMessage("System Alert").
//	    Body("Database connection failed").
//	    Priority(PriorityHigh).
//	    AddTarget(NewEmailTarget("admin@company.com")).
//	    Build()
func NewMessage(title ...string) *MessageBuilder {
	return message.NewMessage(title...)
}

// NewAlert creates a new alert message with high priority.
//
// Alert messages are pre-configured with high priority and are suitable
// for important notifications that require immediate attention.
//
// Parameters:
//   - title: The alert message title
//
// Returns:
//   - *MessageBuilder: A message builder configured for alerts
//
// Example:
//
//	msg := NewAlert("Security Breach Detected").
//	    Body("Unauthorized access attempt from IP: 192.168.1.100").
//	    AddTarget(NewEmailTarget("security@company.com")).
//	    Build()
func NewAlert(title string) *MessageBuilder {
	return message.NewAlert(title)
}

// NewUrgent creates a new urgent message with the highest priority.
//
// Urgent messages are pre-configured with urgent priority and are suitable
// for critical notifications that require immediate action.
//
// Parameters:
//   - title: The urgent message title
//
// Returns:
//   - *MessageBuilder: A message builder configured for urgent messages
//
// Example:
//
//	msg := NewUrgent("Service Down").
//	    Body("Payment service is currently unavailable").
//	    AddTarget(NewFeishuUserTarget("user123")).
//	    Build()
func NewUrgent(title string) *MessageBuilder {
	return message.NewUrgent(title)
}

// Target factory functions (adapter functions)
func NewTarget(targetType, value, platform string) Target {
	return target.NewTarget(targetType, value, platform)
}

func NewEmailTarget(email string) Target {
	return target.NewEmailTarget(email)
}

func NewPhoneTarget(phone string) Target {
	return target.NewPhoneTarget(phone)
}

func NewFeishuUserTarget(userID string) Target {
	return target.NewFeishuUserTarget(userID)
}

func NewFeishuGroupTarget(groupID string) Target {
	return target.NewFeishuGroupTarget(groupID)
}

func NewWebhookTarget(url string) Target {
	return target.NewWebhookTarget(url)
}

func AutoDetectTarget(value string) Target {
	return target.AutoDetectTarget(value)
}

// Error checking functions
func IsValidationError(err error) bool {
	// Simple validation error check
	if err == nil {
		return false
	}
	// You can implement more sophisticated error checking here
	return false
}

// Test configuration functions
func TestConfig() []HubOption {
	return []HubOption{WithTestDefaults()}
}

// QuickStart provides a simple way to create a working hub
type QuickStartBuilder struct {
	options []HubOption
}

func QuickStart() *QuickStartBuilder {
	return &QuickStartBuilder{
		options: []HubOption{WithTestDefaults()},
	}
}

func (q *QuickStartBuilder) WithFeishuWebhook(webhookURL, secret string) *QuickStartBuilder {
	q.options = append(q.options, WithFeishu(webhookURL, secret))
	return q
}

func (q *QuickStartBuilder) WithEmailSMTP(host string, port int, username, password, from string) *QuickStartBuilder {
	q.options = append(q.options, WithEmail(host, port, username, password, from, true, 30*time.Second))
	return q
}

func (q *QuickStartBuilder) Build() (Hub, error) {
	return NewHub(q.options...)
}

// ===== NEW EXTENSIBLE PLATFORM SYSTEM =====

// WithCustomPlatform configures a custom/external platform using the extension system
// This is the new recommended way to add external platforms
func WithCustomPlatform(platformName string, config map[string]interface{}) HubOption {
	return WithExtension(platformName, config)
}

// WithSlack adds Slack platform support (example of external platform)
func WithSlack(webhookURL string, options ...func(map[string]interface{})) HubOption {
	config := SimpleWebhookConfig(webhookURL, options...)
	return WithCustomPlatform("slack", config)
}

// WithDiscord adds Discord platform support (example of external platform)
func WithDiscord(webhookURL string, options ...func(map[string]interface{})) HubOption {
	config := SimpleWebhookConfig(webhookURL, options...)
	return WithCustomPlatform("discord", config)
}

// WithTelegram adds Telegram platform support (example of external platform)
func WithTelegram(botToken, chatID string) HubOption {
	config := map[string]interface{}{
		"bot_token": botToken,
		"chat_id":   chatID,
	}
	return WithCustomPlatform("telegram", config)
}

// WithWebhook adds generic webhook platform support
func WithWebhook(name, webhookURL string, options ...func(map[string]interface{})) HubOption {
	config := SimpleWebhookConfig(webhookURL, options...)
	return WithCustomPlatform(name, config)
}

// GetAvailablePlatforms returns information about all available platforms
func GetAvailablePlatforms() []PlatformInfo {
	return GetAllPlatformInfo()
}

// IsSupported checks if a platform is supported (either built-in or extension)
func IsSupported(platformName string) bool {
	// Check built-in platforms
	builtins := []string{"feishu", "email", "sms"}
	for _, builtin := range builtins {
		if platformName == builtin {
			return true
		}
	}

	// Check extensions
	return IsExtensionRegistered(platformName)
}

// GetPlatformTemplate returns a configuration template for a platform
func GetPlatformTemplate(platformName string) (*ConfigTemplate, error) {
	return GetConfigTemplate(platformName)
}

// ConfigBuilderFor creates a configuration builder for a specific platform
func ConfigBuilderFor(platformName string) *ConfigBuilder {
	builder := NewConfigBuilder()

	// Load defaults if extension exists
	if ext, exists := GetExtension(platformName); exists && ext.DefaultOpts != nil {
		builder.Merge(ext.DefaultOpts())
	}

	return builder
}
