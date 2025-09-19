package notifyhub

import (
	"time"

	"github.com/kart-io/notifyhub/logger"
)

// 平台配置选项

// WithFeishu 配置飞书平台
func WithFeishu(webhook, secret string) Option {
	return func(cfg *Config) {
		platform := PlatformConfig{
			Type:    PlatformFeishu,
			Name:    "feishu",
			Enabled: true,
			Settings: map[string]interface{}{
				"webhook": webhook,
				"secret":  secret,
			},
		}
		cfg.Platforms = append(cfg.Platforms, platform)
	}
}

// WithEmail 配置邮件平台
func WithEmail(host string, port int, username, password, from string) Option {
	return func(cfg *Config) {
		platform := PlatformConfig{
			Type:    PlatformEmail,
			Name:    "email",
			Enabled: true,
			Settings: map[string]interface{}{
				"host":     host,
				"port":     port,
				"username": username,
				"password": password,
				"from":     from,
				"use_tls":  true,
				"timeout":  30 * time.Second,
			},
		}
		cfg.Platforms = append(cfg.Platforms, platform)
	}
}

// WithEmailAdvanced 高级邮件配置
func WithEmailAdvanced(host string, port int, auth EmailAuth, config EmailConfig) Option {
	return func(cfg *Config) {
		platform := PlatformConfig{
			Type:    PlatformEmail,
			Name:    "email",
			Enabled: true,
			Settings: map[string]interface{}{
				"host":     host,
				"port":     port,
				"username": auth.Username,
				"password": auth.Password,
				"from":     config.From,
				"use_tls":  config.UseTLS,
				"timeout":  config.Timeout,
			},
		}
		cfg.Platforms = append(cfg.Platforms, platform)
	}
}

// WithSMS 配置短信平台
func WithSMS(provider, apiKey, apiSecret string) Option {
	return func(cfg *Config) {
		platform := PlatformConfig{
			Type:    PlatformSMS,
			Name:    "sms",
			Enabled: true,
			Settings: map[string]interface{}{
				"provider":   provider,
				"api_key":    apiKey,
				"api_secret": apiSecret,
			},
		}
		cfg.Platforms = append(cfg.Platforms, platform)
	}
}

// WithSlack 配置Slack平台
func WithSlack(webhook, token string) Option {
	return func(cfg *Config) {
		platform := PlatformConfig{
			Type:    PlatformSlack,
			Name:    "slack",
			Enabled: true,
			Settings: map[string]interface{}{
				"webhook": webhook,
				"token":   token,
			},
		}
		cfg.Platforms = append(cfg.Platforms, platform)
	}
}

// 高级配置选项

// WithPlatforms 批量配置平台
func WithPlatforms(platforms ...PlatformConfig) Option {
	return func(cfg *Config) {
		cfg.Platforms = append(cfg.Platforms, platforms...)
	}
}

// WithQueue 配置队列
func WithQueue(queueType string, capacity, concurrency int) Option {
	return func(cfg *Config) {
		cfg.Queue = &QueueConfig{
			Type:        queueType,
			Capacity:    capacity,
			Concurrency: concurrency,
		}
	}
}

// WithMemoryQueue 配置内存队列
func WithMemoryQueue(capacity, concurrency int) Option {
	return WithQueue("memory", capacity, concurrency)
}

// WithRedisQueue 配置Redis队列
func WithRedisQueue(addr, password string, db, capacity, concurrency int) Option {
	return func(cfg *Config) {
		cfg.Queue = &QueueConfig{
			Type:        "redis",
			Capacity:    capacity,
			Concurrency: concurrency,
		}
		// Redis特定配置可以添加到Platform中或单独处理
	}
}

// WithRetryPolicy 配置重试策略
func WithRetryPolicy(maxAttempts int, backoff time.Duration, jitter bool) Option {
	return func(cfg *Config) {
		cfg.Retry = &RetryConfig{
			MaxAttempts: maxAttempts,
			Backoff:     backoff,
			Jitter:      jitter,
		}
	}
}

// WithSimpleRetry 简单重试配置
func WithSimpleRetry(maxAttempts int) Option {
	return WithRetryPolicy(maxAttempts, time.Second, true)
}

// WithRateLimit 配置限流
func WithRateLimit(rate, burst int, window time.Duration) Option {
	return func(cfg *Config) {
		cfg.RateLimit = &RateLimitConfig{
			Rate:   rate,
			Burst:  burst,
			Window: window,
		}
	}
}

// WithLogger 配置日志器
func WithLogger(l logger.Interface) Option {
	return func(cfg *Config) {
		cfg.Logger = l
	}
}

// 预设配置

// WithProduction 生产环境预设
func WithProduction() Option {
	return func(cfg *Config) {
		// 生产环境优化配置
		cfg.Queue = &QueueConfig{
			Type:        "redis",
			Capacity:    10000,
			Concurrency: 16,
		}
		cfg.Retry = &RetryConfig{
			MaxAttempts: 5,
			Backoff:     2 * time.Second,
			Jitter:      true,
		}
		cfg.RateLimit = &RateLimitConfig{
			Rate:   100,
			Burst:  200,
			Window: time.Minute,
		}
		cfg.Logger = logger.NewStdLogger(logger.Warn)
	}
}

// WithDevelopment 开发环境预设
func WithDevelopment() Option {
	return func(cfg *Config) {
		// 开发环境配置
		cfg.Queue = &QueueConfig{
			Type:        "memory",
			Capacity:    1000,
			Concurrency: 4,
		}
		cfg.Retry = &RetryConfig{
			MaxAttempts: 3,
			Backoff:     time.Second,
			Jitter:      false,
		}
		cfg.Logger = logger.NewStdLogger(logger.Debug)
	}
}

// WithTesting 测试环境预设
func WithTesting() Option {
	return func(cfg *Config) {
		// 测试环境配置
		cfg.Queue = &QueueConfig{
			Type:        "memory",
			Capacity:    100,
			Concurrency: 1,
		}
		cfg.Retry = &RetryConfig{
			MaxAttempts: 1,
			Backoff:     100 * time.Millisecond,
			Jitter:      false,
		}
		cfg.Logger = logger.Discard
	}
}

// WithDefaults 默认配置
func WithDefaults() Option {
	return func(cfg *Config) {
		// 从环境变量自动配置
		// TODO: 实现环境变量读取逻辑
	}
}

// 配置结构体

// EmailAuth 邮件认证配置
type EmailAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	From    string        `json:"from"`
	UseTLS  bool          `json:"use_tls"`
	Timeout time.Duration `json:"timeout"`
}

// SMSAuth 短信认证配置
type SMSAuth struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

// 组合配置选项

// WithChatPlatform 通用聊天平台配置
func WithChatPlatform(platform PlatformType, webhook, secret string) Option {
	return func(cfg *Config) {
		platformConfig := PlatformConfig{
			Type:    platform,
			Name:    string(platform),
			Enabled: true,
			Settings: map[string]interface{}{
				"webhook": webhook,
				"secret":  secret,
			},
		}
		cfg.Platforms = append(cfg.Platforms, platformConfig)
	}
}

// WithEmailService 通用邮件服务配置
func WithEmailService(provider, from string, auth EmailAuth) Option {
	return func(cfg *Config) {
		settings := map[string]interface{}{
			"provider": provider,
			"from":     from,
			"username": auth.Username,
			"password": auth.Password,
			"use_tls":  true,
			"timeout":  30 * time.Second,
		}

		// 根据提供商设置默认配置
		switch provider {
		case "gmail":
			settings["host"] = "smtp.gmail.com"
			settings["port"] = 587
		case "outlook":
			settings["host"] = "smtp-mail.outlook.com"
			settings["port"] = 587
		case "qq":
			settings["host"] = "smtp.qq.com"
			settings["port"] = 587
		}

		platform := PlatformConfig{
			Type:     PlatformEmail,
			Name:     "email",
			Enabled:  true,
			Settings: settings,
		}
		cfg.Platforms = append(cfg.Platforms, platform)
	}
}

// WithSMSService 通用短信服务配置
func WithSMSService(provider string, auth SMSAuth) Option {
	return func(cfg *Config) {
		platform := PlatformConfig{
			Type:    PlatformSMS,
			Name:    "sms",
			Enabled: true,
			Settings: map[string]interface{}{
				"provider":   provider,
				"api_key":    auth.APIKey,
				"api_secret": auth.APISecret,
			},
		}
		cfg.Platforms = append(cfg.Platforms, platform)
	}
}
