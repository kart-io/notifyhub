package notifyhub

import (
	"context"
	"time"

	"github.com/kart-io/notifyhub/core/hub"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/logger"
)

// Client 是 NotifyHub 的统一客户端，提供简化的API入口
type Client struct {
	hub      *hub.Hub
	logger   logger.Interface
	config   *Config
	resolver *TargetResolver
}

// Config 简化的配置结构
type Config struct {
	Platforms []PlatformConfig `json:"platforms"`
	Queue     *QueueConfig     `json:"queue,omitempty"`
	Retry     *RetryConfig     `json:"retry,omitempty"`
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty"`
	Logger    logger.Interface `json:"-"`
}

// PlatformConfig 平台配置抽象
type PlatformConfig struct {
	Type     PlatformType           `json:"type"`
	Name     string                 `json:"name"`
	Settings map[string]interface{} `json:"settings"`
	Enabled  bool                   `json:"enabled"`
}

// PlatformType 平台类型
type PlatformType string

const (
	PlatformFeishu PlatformType = "feishu"
	PlatformEmail  PlatformType = "email"
	PlatformSMS    PlatformType = "sms"
	PlatformSlack  PlatformType = "slack"
)

// QueueConfig 队列配置
type QueueConfig struct {
	Type        string `json:"type"` // "memory", "redis"
	Capacity    int    `json:"capacity"`
	Concurrency int    `json:"concurrency"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	Backoff     time.Duration `json:"backoff"`
	Jitter      bool          `json:"jitter"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Rate   int           `json:"rate"`  // 每秒请求数
	Burst  int           `json:"burst"` // 突发容量
	Window time.Duration `json:"window"`
}

// Option 配置选项
type Option func(*Config)

// New 创建 NotifyHub 客户端 - 统一入口
func New(options ...Option) (*Client, error) {
	cfg := &Config{
		Platforms: make([]PlatformConfig, 0),
		Queue: &QueueConfig{
			Type:        "memory",
			Capacity:    1000,
			Concurrency: 4,
		},
		Retry: &RetryConfig{
			MaxAttempts: 3,
			Backoff:     time.Second,
			Jitter:      true,
		},
		Logger: logger.Default,
	}

	// 应用配置选项
	for _, option := range options {
		option(cfg)
	}

	// 创建Hub选项
	hubOpts := &hub.Options{
		Logger: cfg.Logger,
	}

	// 创建核心Hub
	h := hub.NewHub(hubOpts)

	// 注册平台传输器
	if err := registerPlatformTransports(h, cfg.Platforms); err != nil {
		return nil, err
	}

	// 创建目标解析器
	resolver := &TargetResolver{}

	client := &Client{
		hub:      h,
		logger:   cfg.Logger,
		config:   cfg,
		resolver: resolver,
	}

	return client, nil
}

// Send 创建发送构建器 - 统一发送入口
func (c *Client) Send(ctx context.Context) *SendBuilder {
	return &SendBuilder{
		client:  c,
		ctx:     ctx,
		message: message.NewMessage(),
		targets: make([]sending.Target, 0),
	}
}

// Alert 创建告警构建器
func (c *Client) Alert(ctx context.Context) *AlertBuilder {
	return &AlertBuilder{
		SendBuilder: c.Send(ctx),
		priority:    message.PriorityHigh,
	}
}

// Notification 创建通知构建器
func (c *Client) Notification(ctx context.Context) *NotificationBuilder {
	return &NotificationBuilder{
		SendBuilder: c.Send(ctx),
		priority:    message.PriorityNormal,
	}
}

// Configure 动态配置变更
func (c *Client) Configure(options ...Option) error {
	for _, option := range options {
		option(c.config)
	}
	// TODO: 重新配置Hub
	return nil
}

// Health 健康检查
func (c *Client) Health() HealthStatus {
	hubStatus := c.hub.Health(context.Background())
	return HealthStatus{
		Healthy: hubStatus.Healthy,
		Details: hubStatus.Details,
	}
}

// Shutdown 优雅关闭
func (c *Client) Shutdown(ctx context.Context) error {
	return c.hub.Shutdown(ctx)
}

// HealthStatus 健康状态
type HealthStatus struct {
	Healthy bool                   `json:"healthy"`
	Details map[string]interface{} `json:"details"`
}

// TargetResolver 目标解析器
type TargetResolver struct {
	// 目标解析逻辑
}

// registerPlatformTransports 注册平台传输器
func registerPlatformTransports(h *hub.Hub, platforms []PlatformConfig) error {
	// TODO: 实现平台传输器注册逻辑
	return nil
}
