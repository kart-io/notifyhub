package feishu

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/platforms"
	"github.com/kart-io/notifyhub/transport/feishu"
)

// FeishuPlatform implements the Platform interface for Feishu
type FeishuPlatform struct {
	*platforms.BasePlatform
}

// NewFeishuPlatform creates a new Feishu platform implementation
func NewFeishuPlatform() *FeishuPlatform {
	capabilities := platforms.NewBaseCapabilities().
		WithFormats(platforms.FormatText, platforms.FormatMarkdown, platforms.FormatCard).
		WithFeatures(
			platforms.FeatureText,
			platforms.FeatureMarkdown,
			platforms.FeatureCard,
			platforms.FeatureMentions,
			platforms.FeatureBatch,
			platforms.FeatureRetry,
		).
		WithTargetTypes("group", "webhook", "user").
		WithMaxMessageSize(1024 * 100). // 100KB
		WithMaxTitleLength(256).
		WithMaxBodyLength(30000).
		WithRateLimits(platforms.RateLimitInfo{
			RequestsPerSecond: 20,
			BurstSize:         50,
			Window:            time.Second,
			Enforced:          true,
		})

	return &FeishuPlatform{
		BasePlatform: platforms.NewBasePlatform(
			"feishu",
			"Feishu (Lark) messaging platform for team collaboration",
			capabilities,
		),
	}
}

// ValidateConfig validates Feishu configuration
func (p *FeishuPlatform) ValidateConfig(config map[string]interface{}) error {
	if webhook, ok := config["webhook_url"].(string); !ok || webhook == "" {
		return fmt.Errorf("webhook_url is required")
	}

	// 验证安全设置：三选一模式
	securityCount := 0

	// 检查签名校验
	if secret, ok := config["secret"].(string); ok && secret != "" {
		securityCount++
	}

	// 检查IP白名单模式标识
	if useIPWhitelist, ok := config["use_ip_whitelist"].(bool); ok && useIPWhitelist {
		securityCount++
	}

	// 检查自定义关键词
	if keywords, ok := config["custom_keywords"]; ok && keywords != nil {
		switch v := keywords.(type) {
		case []string:
			if len(v) > 0 {
				securityCount++
			}
		case []interface{}:
			if len(v) > 0 {
				securityCount++
				// Validate each keyword
				for i, keyword := range v {
					if _, ok := keyword.(string); !ok {
						return fmt.Errorf("custom_keywords[%d] must be a string", i)
					}
				}
			}
		default:
			return fmt.Errorf("custom_keywords must be an array of strings")
		}
	}

	// 飞书安全设置只能选择一种，也可以都不选择（无安全验证）
	if securityCount > 1 {
		return fmt.Errorf("飞书安全设置只能选择一种：签名校验、IP白名单或自定义关键词")
	}

	return nil
}

// CreateTransport creates a Feishu transport instance
func (p *FeishuPlatform) CreateTransport(config map[string]interface{}) (platforms.Transport, error) {
	// Extract configuration
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required")
	}

	timeout := 30 * time.Second
	if t, ok := config["timeout"].(int); ok {
		timeout = time.Duration(t) * time.Second
	}

	// 创建Transport配置
	transportConfig := &feishu.Config{
		WebhookURL: webhookURL,
		Timeout:    timeout,
	}

	// 检查是否配置了代理
	if proxyURL, ok := config["proxy_url"].(string); ok && proxyURL != "" {
		transportConfig.ProxyURL = proxyURL
	}

	// 根据配置确定安全类型和相关参数
	secret, _ := config["secret"].(string)

	// Extract custom keywords
	var customKeywords []string
	if keywords, ok := config["custom_keywords"]; ok && keywords != nil {
		switch v := keywords.(type) {
		case []string:
			if len(v) > 0 {
				customKeywords = v
			}
		case []interface{}:
			for _, keyword := range v {
				if keywordStr, ok := keyword.(string); ok {
					customKeywords = append(customKeywords, keywordStr)
				}
			}
		}
	}

	// 设置安全配置（四选一）
	if secret != "" {
		transportConfig.SecurityType = feishu.SecurityTypeSignature
		transportConfig.Secret = secret
	} else if len(customKeywords) > 0 {
		transportConfig.SecurityType = feishu.SecurityTypeKeywords
		transportConfig.CustomKeywords = customKeywords
	} else {
		transportConfig.SecurityType = feishu.SecurityTypeNone
	}

	// Create the actual transport
	transport := feishu.NewTransport(transportConfig)

	return &FeishuTransportAdapter{transport: transport}, nil
}

// DefaultConfig returns default configuration for Feishu
func (p *FeishuPlatform) DefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"webhook_url":     "",
		"secret":          "",
		"timeout":         30,
		"ip_whitelist":    []string{}, // IP白名单，空表示不限制
		"custom_keywords": []string{}, // 自定义关键词，空表示不限制
	}
}

// FeishuTransportAdapter adapts the feishu transport to the Platform Transport interface
type FeishuTransportAdapter struct {
	transport *feishu.Transport
}

// Send sends a message through Feishu
func (t *FeishuTransportAdapter) Send(ctx context.Context, msg *core.Message, target core.Target) (*core.Result, error) {
	return t.transport.Send(ctx, msg, target)
}

// Name returns the transport name
func (t *FeishuTransportAdapter) Name() string {
	return t.transport.Name()
}

// Shutdown shuts down the transport
func (t *FeishuTransportAdapter) Shutdown() error {
	return t.transport.Shutdown()
}

// Health checks if the transport is healthy
func (t *FeishuTransportAdapter) Health(ctx context.Context) error {
	// The feishu transport doesn't have a Health method, so return nil
	return nil
}
