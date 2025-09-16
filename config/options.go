package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/logger"
	"github.com/kart-io/notifyhub/queue"
)

// ================================
// Option模式配置
// ================================

// Config holds NotifyHub configuration
type Config struct {
	feishu       *FeishuConfig
	email        *EmailConfig
	queue        *QueueConfig
	routing      *RoutingConfig
	telemetry    *TelemetryConfig
	logger       logger.Interface
	mockNotifier *MockNotifierConfig
}

// MockNotifierConfig holds mock notifier configuration
type MockNotifierConfig struct {
	Name             string
	ShouldFail       bool
	Delay            time.Duration
	SupportedTargets []string
}

// Option defines a configuration option
type Option interface {
	apply(*Config)
}

// optionFunc wraps a function to implement Option interface
type optionFunc func(*Config)

func (f optionFunc) apply(c *Config) {
	f(c)
}

// ================================
// Feishu Configuration Options
// ================================

type FeishuConfig struct {
	WebhookURL string
	Secret     string
	Timeout    time.Duration
}

// WithFeishu configures Feishu webhook settings
func WithFeishu(webhookURL string, secret string) Option {
	return optionFunc(func(c *Config) {
		c.feishu = &FeishuConfig{
			WebhookURL: webhookURL,
			Secret:     secret,
			Timeout:    30 * time.Second, // default timeout
		}
	})
}

// WithFeishuTimeout sets Feishu timeout
func WithFeishuTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *Config) {
		if c.feishu != nil {
			c.feishu.Timeout = timeout
		}
	})
}

// WithFeishuFromEnv configures Feishu from environment variables
func WithFeishuFromEnv() Option {
	return optionFunc(func(c *Config) {
		if url := os.Getenv("NOTIFYHUB_FEISHU_WEBHOOK_URL"); url != "" {
			c.feishu = &FeishuConfig{
				WebhookURL: url,
				Secret:     os.Getenv("NOTIFYHUB_FEISHU_SECRET"),
				Timeout:    getEnvDurationOrDefault("NOTIFYHUB_FEISHU_TIMEOUT", 30*time.Second),
			}
		}
	})
}

// ================================
// Email Configuration Options
// ================================

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
	Timeout  time.Duration
}

// WithEmail configures SMTP email settings
func WithEmail(host string, port int, username, password, from string) Option {
	return optionFunc(func(c *Config) {
		c.email = &EmailConfig{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			From:     from,
			UseTLS:   true, // default to TLS
			Timeout:  30 * time.Second,
		}
	})
}

// WithEmailTLS sets email TLS usage
func WithEmailTLS(useTLS bool) Option {
	return optionFunc(func(c *Config) {
		if c.email != nil {
			c.email.UseTLS = useTLS
		}
	})
}

// WithEmailTimeout sets email timeout
func WithEmailTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *Config) {
		if c.email != nil {
			c.email.Timeout = timeout
		}
	})
}

// WithEmailFromEnv configures email from environment variables
func WithEmailFromEnv() Option {
	return optionFunc(func(c *Config) {
		if host := os.Getenv("NOTIFYHUB_SMTP_HOST"); host != "" {
			c.email = &EmailConfig{
				Host:     host,
				Port:     getEnvIntOrDefault("NOTIFYHUB_SMTP_PORT", 587),
				Username: os.Getenv("NOTIFYHUB_SMTP_USERNAME"),
				Password: os.Getenv("NOTIFYHUB_SMTP_PASSWORD"),
				From:     os.Getenv("NOTIFYHUB_SMTP_FROM"),
				UseTLS:   getEnvBoolOrDefault("NOTIFYHUB_SMTP_USE_TLS", true),
				Timeout:  getEnvDurationOrDefault("NOTIFYHUB_SMTP_TIMEOUT", 30*time.Second),
			}
		}
	})
}

// ================================
// Mock Notifier Configuration Options (for testing)
// ================================

// WithMockNotifier configures a mock notifier for testing
func WithMockNotifier(name string) Option {
	return optionFunc(func(c *Config) {
		c.mockNotifier = &MockNotifierConfig{
			Name:             name,
			ShouldFail:       false,
			Delay:            10 * time.Millisecond,
			SupportedTargets: []string{"email", "user", "group", "channel"},
		}
	})
}

// WithMockNotifierFailure configures mock notifier to fail
func WithMockNotifierFailure() Option {
	return optionFunc(func(c *Config) {
		if c.mockNotifier != nil {
			c.mockNotifier.ShouldFail = true
		}
	})
}

// WithMockNotifierDelay configures mock notifier delay
func WithMockNotifierDelay(delay time.Duration) Option {
	return optionFunc(func(c *Config) {
		if c.mockNotifier != nil {
			c.mockNotifier.Delay = delay
		}
	})
}

// ================================
// Queue Configuration Options
// ================================

type QueueConfig struct {
	Type        string
	BufferSize  int
	Workers     int
	RetryPolicy *queue.RetryPolicy
}

// WithQueue configures queue settings
func WithQueue(queueType string, bufferSize, workers int) Option {
	return optionFunc(func(c *Config) {
		c.queue = &QueueConfig{
			Type:        queueType,
			BufferSize:  bufferSize,
			Workers:     workers,
			RetryPolicy: queue.DefaultRetryPolicy(),
		}
	})
}

// WithQueueRetryPolicy sets queue retry policy
func WithQueueRetryPolicy(policy *queue.RetryPolicy) Option {
	return optionFunc(func(c *Config) {
		if c.queue != nil {
			c.queue.RetryPolicy = policy
		}
	})
}

// WithQueueFromEnv configures queue from environment variables
func WithQueueFromEnv() Option {
	return optionFunc(func(c *Config) {
		c.queue = &QueueConfig{
			Type:       getEnvOrDefault("NOTIFYHUB_QUEUE_TYPE", "memory"),
			BufferSize: getEnvIntOrDefault("NOTIFYHUB_QUEUE_BUFFER_SIZE", 1000),
			Workers:    getEnvIntOrDefault("NOTIFYHUB_QUEUE_WORKERS", 2),
			RetryPolicy: &queue.RetryPolicy{
				MaxRetries:      getEnvIntOrDefault("NOTIFYHUB_RETRY_MAX", 3),
				InitialInterval: getEnvDurationOrDefault("NOTIFYHUB_RETRY_INTERVAL", 30*time.Second),
				Multiplier:      2.0,
			},
		}
	})
}

// ================================
// Routing Configuration Options
// ================================

type RoutingConfig struct {
	Rules []RoutingRule
}

type RoutingRule struct {
	Name       string
	Priority   int // Higher values = higher priority
	Enabled    bool
	Conditions RuleConditions
	Actions    []RuleAction
}

type RuleConditions struct {
	MessageType []string
	Priority    []int
	Metadata    map[string]string
}

type RuleAction struct {
	Type      string // "route"
	Platforms []string
}

// WithRouting configures routing rules
func WithRouting(rules ...RoutingRule) Option {
	return optionFunc(func(c *Config) {
		c.routing = &RoutingConfig{
			Rules: rules,
		}
	})
}

// WithDefaultRouting configures default routing rules
func WithDefaultRouting() Option {
	return optionFunc(func(c *Config) {
		c.routing = &RoutingConfig{
			Rules: []RoutingRule{
				{
					Name:     "high_priority_all",
					Priority: 100, // High priority rule
					Enabled:  true,
					Conditions: RuleConditions{
						Priority: []int{4, 5},
					},
					Actions: []RuleAction{
						{Type: "route", Platforms: []string{"feishu", "email"}},
					},
				},
				{
					Name:     "alerts_to_feishu",
					Priority: 50, // Medium priority rule
					Enabled:  true,
					Conditions: RuleConditions{
						Metadata: map[string]string{"type": "alert"},
					},
					Actions: []RuleAction{
						{Type: "route", Platforms: []string{"feishu"}},
					},
				},
			},
		}
	})
}

// ================================
// Telemetry Configuration Options
// ================================

type TelemetryConfig struct {
	ServiceName    string            `json:"service_name" yaml:"service_name"`
	ServiceVersion string            `json:"service_version" yaml:"service_version"`
	Environment    string            `json:"environment" yaml:"environment"`
	OTLPEndpoint   string            `json:"otlp_endpoint" yaml:"otlp_endpoint"`
	OTLPHeaders    map[string]string `json:"otlp_headers" yaml:"otlp_headers"`
	TracingEnabled bool              `json:"tracing_enabled" yaml:"tracing_enabled"`
	SampleRate     float64           `json:"sample_rate" yaml:"sample_rate"`
	MetricsEnabled bool              `json:"metrics_enabled" yaml:"metrics_enabled"`
	Enabled        bool              `json:"enabled" yaml:"enabled"`
}

// WithTelemetry configures telemetry settings
func WithTelemetry(serviceName, serviceVersion, environment string, otlpEndpoint string) Option {
	return optionFunc(func(c *Config) {
		c.telemetry = &TelemetryConfig{
			ServiceName:    serviceName,
			ServiceVersion: serviceVersion,
			Environment:    environment,
			OTLPEndpoint:   otlpEndpoint,
			TracingEnabled: true,
			MetricsEnabled: true,
			SampleRate:     1.0,
			Enabled:        true,
		}
	})
}

// WithTelemetryFromEnv configures telemetry from environment variables
func WithTelemetryFromEnv() Option {
	return optionFunc(func(c *Config) {
		if getEnvBoolOrDefault("NOTIFYHUB_TELEMETRY_ENABLED", false) {
			headers := make(map[string]string)
			if authHeader := getEnvOrDefault("NOTIFYHUB_OTLP_AUTH", ""); authHeader != "" {
				headers["Authorization"] = authHeader
			}

			c.telemetry = &TelemetryConfig{
				ServiceName:    getEnvOrDefault("NOTIFYHUB_SERVICE_NAME", "notifyhub"),
				ServiceVersion: getEnvOrDefault("NOTIFYHUB_SERVICE_VERSION", "1.2.0"),
				Environment:    getEnvOrDefault("NOTIFYHUB_ENVIRONMENT", "development"),
				OTLPEndpoint:   getEnvOrDefault("NOTIFYHUB_OTLP_ENDPOINT", "http://localhost:4318"),
				OTLPHeaders:    headers,
				TracingEnabled: getEnvBoolOrDefault("NOTIFYHUB_TRACING_ENABLED", true),
				MetricsEnabled: getEnvBoolOrDefault("NOTIFYHUB_METRICS_ENABLED", true),
				SampleRate:     getEnvFloatOrDefault("NOTIFYHUB_SAMPLE_RATE", 1.0),
				Enabled:        true,
			}
		}
	})
}

// WithTelemetryDisabled explicitly disables telemetry
func WithTelemetryDisabled() Option {
	return optionFunc(func(c *Config) {
		c.telemetry = &TelemetryConfig{
			Enabled: false,
		}
	})
}

// ================================
// Logger Configuration Options
// ================================

// WithLogger configures a custom logger
func WithLogger(logger logger.Interface) Option {
	return optionFunc(func(c *Config) {
		c.logger = logger
	})
}

// WithDefaultLogger configures the default logger with specified level
func WithDefaultLogger(level logger.LogLevel) Option {
	return optionFunc(func(c *Config) {
		c.logger = logger.Default.LogMode(level)
	})
}

// WithSilentLogger disables logging
func WithSilentLogger() Option {
	return optionFunc(func(c *Config) {
		c.logger = logger.Default.LogMode(logger.Silent)
	})
}

// ================================
// Preset Option Combinations
// ================================

// WithDefaults applies default configuration from environment variables
func WithDefaults() Option {
	return optionFunc(func(c *Config) {
		WithFeishuFromEnv().apply(c)
		WithEmailFromEnv().apply(c)
		WithQueueFromEnv().apply(c)
		WithTelemetryFromEnv().apply(c)
		WithDefaultRouting().apply(c)
		if c.logger == nil {
			WithDefaultLogger(logger.Warn).apply(c)
		}
	})
}

// WithTestDefaults applies test-friendly default configuration including a mock notifier
func WithTestDefaults() Option {
	return optionFunc(func(c *Config) {
		WithQueue("memory", 100, 1).apply(c)
		WithQueueRetryPolicy(queue.NoRetryPolicy()).apply(c)
		WithMockNotifier("test-mock").apply(c)
		if c.logger == nil {
			WithDefaultLogger(logger.Debug).apply(c)
		}
	})
}

// ================================
// Config Creation and Getters
// ================================

// New creates a new configuration with given options
func New(opts ...Option) *Config {
	c := &Config{}
	for _, opt := range opts {
		opt.apply(c)
	}

	// Apply defaults for any unset configurations
	if c.queue == nil {
		WithQueueFromEnv().apply(c)
	}

	// Apply default logger if not set
	if c.logger == nil {
		c.logger = logger.Default.LogMode(logger.Warn)
	}

	return c
}

// Getters for accessing configuration

// Feishu returns Feishu configuration
func (c *Config) Feishu() *FeishuConfig {
	return c.feishu
}

// Email returns Email configuration
func (c *Config) Email() *EmailConfig {
	return c.email
}

// Queue returns Queue configuration
func (c *Config) Queue() *QueueConfig {
	return c.queue
}

// Routing returns Routing configuration
func (c *Config) Routing() *RoutingConfig {
	return c.routing
}

// Telemetry returns Telemetry configuration
func (c *Config) Telemetry() *TelemetryConfig {
	return c.telemetry
}

// Logger returns Logger interface
func (c *Config) Logger() logger.Interface {
	return c.logger
}

// MockNotifier returns MockNotifier configuration
func (c *Config) MockNotifier() *MockNotifierConfig {
	return c.mockNotifier
}

// ================================
// Utility functions
// ================================

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}

func getEnvFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
