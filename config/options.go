package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/config/routing"
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
	queue        *QueueConfigOptions
	routing      []routing.Rule
	telemetry    *TelemetryConfig
	logger       logger.Interface
	mockNotifier *MockNotifierConfig

	// 外部队列实现支持
	queueConfigs         map[string]interface{}     // 队列类型特定配置
	externalQueue        queue.ExternalQueue        // 外部队列实例
	externalQueueFactory queue.ExternalQueueFactory // 外部队列工厂
}

// MockNotifierConfig holds null transport configuration for testing/development
// Note: Despite the "Mock" name for backward compatibility, this actually configures
// null transports that discard messages. For actual testing mocks, use tests/mocks package.
type MockNotifierConfig struct {
	Name             string
	ShouldFail       bool // Currently unused in null transport
	Delay            time.Duration
	SupportedTargets []string // Currently unused in null transport
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

// WithFeishuSimple configures Feishu webhook settings
func WithFeishuSimple(webhookURL string) Option {
	return optionFunc(func(c *Config) {
		c.feishu = &FeishuConfig{
			WebhookURL: webhookURL,
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

// WithMockNotifier configures null transports for testing/development
// Despite the "Mock" name (kept for backward compatibility), this creates null transports
// that discard messages but report successful delivery. For testing with verification, use tests/mocks.
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

// WithMockNotifierFailure configures null transport failure behavior
// Note: Currently unused as null transports always succeed
func WithMockNotifierFailure() Option {
	return optionFunc(func(c *Config) {
		if c.mockNotifier != nil {
			c.mockNotifier.ShouldFail = true
		}
	})
}

// WithMockNotifierDelay configures null transport delay
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

// QueueConfig holds basic queue configuration
type QueueConfig struct {
	Enabled    bool
	Type       string // "memory", "redis"
	BufferSize int
	Workers    int
}

// QueueConfigOptions extends the base QueueConfig with runtime options
type QueueConfigOptions struct {
	QueueConfig
	RetryPolicy *queue.RetryPolicy
}

// WithQueue configures queue settings
func WithQueue(queueType string, bufferSize, workers int) Option {
	return optionFunc(func(c *Config) {
		c.queue = &QueueConfigOptions{
			QueueConfig: QueueConfig{
				Enabled:    true,
				Type:       queueType,
				BufferSize: bufferSize,
				Workers:    workers,
			},
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
		c.queue = &QueueConfigOptions{
			QueueConfig: QueueConfig{
				Enabled:    true,
				Type:       getEnvOrDefault("NOTIFYHUB_QUEUE_TYPE", "memory"),
				BufferSize: getEnvIntOrDefault("NOTIFYHUB_QUEUE_BUFFER_SIZE", 1000),
				Workers:    getEnvIntOrDefault("NOTIFYHUB_QUEUE_WORKERS", 2),
			},
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

// WithRouting configures routing rules
func WithRouting(rules ...routing.Rule) Option {
	return optionFunc(func(c *Config) {
		c.routing = rules
	})
}

// WithDefaultRouting configures basic routing rules
func WithDefaultRouting() Option {
	return optionFunc(func(c *Config) {
		// Keep routing minimal for toolkit - users can add their own rules
		c.routing = []routing.Rule{}
	})
}

// ================================
// Telemetry Configuration Options
// ================================

type TelemetryConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	OTLPHeaders    map[string]string
	TracingEnabled bool
	SampleRate     float64
	MetricsEnabled bool
	Enabled        bool
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

// WithTelemetryFromEnv configures telemetry from environment variables (simplified)
func WithTelemetryFromEnv() Option {
	return optionFunc(func(c *Config) {
		if getEnvBoolOrDefault("NOTIFYHUB_TELEMETRY_ENABLED", false) {
			c.telemetry = &TelemetryConfig{
				ServiceName:    getEnvOrDefault("NOTIFYHUB_SERVICE_NAME", "notifyhub"),
				ServiceVersion: getEnvOrDefault("NOTIFYHUB_SERVICE_VERSION", "1.2.0"),
				Environment:    getEnvOrDefault("NOTIFYHUB_ENVIRONMENT", "development"),
				OTLPEndpoint:   getEnvOrDefault("NOTIFYHUB_OTLP_ENDPOINT", "http://localhost:4318"),
				TracingEnabled: true,
				MetricsEnabled: true,
				SampleRate:     1.0,
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

// WithDefaults applies sensible default configuration for quick setup
func WithDefaults() Option {
	return optionFunc(func(c *Config) {
		// Load from environment variables if available
		WithFeishuFromEnv().apply(c)
		WithEmailFromEnv().apply(c)
		WithQueueFromEnv().apply(c)

		// Apply default queue if none configured
		if c.queue == nil {
			WithQueue("memory", 1000, 4).apply(c)
		}

		// Apply default logger if none configured
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
func (c *Config) Queue() *QueueConfigOptions {
	return c.queue
}

// Routing returns routing rules
func (c *Config) Routing() []routing.Rule {
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

// QueueConfigs returns queue type specific configurations
func (c *Config) QueueConfigs() map[string]interface{} {
	return c.queueConfigs
}

// ExternalQueue returns external queue instance
func (c *Config) ExternalQueue() queue.ExternalQueue {
	return c.externalQueue
}

// ExternalQueueFactory returns external queue factory
func (c *Config) ExternalQueueFactory() queue.ExternalQueueFactory {
	return c.externalQueueFactory
}

// ================================
// External Queue Implementation Options
// ================================

// WithExternalQueueConfig 使用外部队列实现配置
func WithExternalQueueConfig(queueType string, config map[string]interface{}) Option {
	return optionFunc(func(c *Config) {
		// 提取基础配置
		bufferSize, _ := config["buffer_size"].(int)
		if bufferSize <= 0 {
			bufferSize = 1000
		}

		workers, _ := config["workers"].(int)
		if workers <= 0 {
			workers = 4
		}

		c.queue = &QueueConfigOptions{
			QueueConfig: QueueConfig{
				Enabled:    true,
				Type:       queueType,
				BufferSize: bufferSize,
				Workers:    workers,
			},
			RetryPolicy: queue.DefaultRetryPolicy(),
		}

		// 存储队列特定配置
		if c.queueConfigs == nil {
			c.queueConfigs = make(map[string]interface{})
		}
		c.queueConfigs[queueType] = config
	})
}

// WithExternalQueue 直接使用外部队列实例（最灵活）
func WithExternalQueue(instance queue.ExternalQueue) Option {
	return optionFunc(func(c *Config) {
		c.queue = &QueueConfigOptions{
			QueueConfig: QueueConfig{
				Enabled:    true,
				Type:       "external",
				BufferSize: 1000, // 默认值，实际由外部队列管理
				Workers:    4,    // 默认值，实际由外部队列管理
			},
			RetryPolicy: queue.DefaultRetryPolicy(),
		}

		// 存储外部队列实例
		c.externalQueue = instance
	})
}

// WithExternalQueueFactory 使用外部队列工厂
func WithExternalQueueFactory(factory queue.ExternalQueueFactory, config map[string]interface{}) Option {
	return optionFunc(func(c *Config) {
		queueType := factory.Name()

		// 提取基础配置
		bufferSize, _ := config["buffer_size"].(int)
		if bufferSize <= 0 {
			bufferSize = 1000
		}

		workers, _ := config["workers"].(int)
		if workers <= 0 {
			workers = 4
		}

		c.queue = &QueueConfigOptions{
			QueueConfig: QueueConfig{
				Enabled:    true,
				Type:       queueType,
				BufferSize: bufferSize,
				Workers:    workers,
			},
			RetryPolicy: queue.DefaultRetryPolicy(),
		}

		// 存储工厂和配置
		if c.queueConfigs == nil {
			c.queueConfigs = make(map[string]interface{})
		}
		c.queueConfigs[queueType] = config
		c.externalQueueFactory = factory
	})
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

func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
