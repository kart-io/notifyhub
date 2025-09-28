package feishu

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
)

// LoadFromEnvironment creates a FeishuConfig from environment variables
// This provides a convenient way to configure Feishu platform using environment variables
func LoadFromEnvironment() *config.FeishuConfig {
	cfg := &config.FeishuConfig{}

	// Load webhook URL from environment
	if webhookURL := os.Getenv("FEISHU_WEBHOOK_URL"); webhookURL != "" {
		cfg.WebhookURL = webhookURL
	}

	// Load secret from environment
	if secret := os.Getenv("FEISHU_SECRET"); secret != "" {
		cfg.Secret = secret
	}

	// Load app ID from environment
	if appID := os.Getenv("FEISHU_APP_ID"); appID != "" {
		cfg.AppID = appID
	}

	// Load app secret from environment
	if appSecret := os.Getenv("FEISHU_APP_SECRET"); appSecret != "" {
		cfg.AppSecret = appSecret
	}

	// Load auth type from environment
	if authType := os.Getenv("FEISHU_AUTH_TYPE"); authType != "" {
		cfg.AuthType = authType
	}

	// Load timeout from environment
	if timeoutStr := os.Getenv("FEISHU_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.Timeout = timeout
		}
	}

	// Load max retries from environment
	if maxRetriesStr := os.Getenv("FEISHU_MAX_RETRIES"); maxRetriesStr != "" {
		if maxRetries, err := strconv.Atoi(maxRetriesStr); err == nil && maxRetries >= 0 {
			cfg.MaxRetries = maxRetries
		}
	}

	// Load rate limit from environment
	if rateLimitStr := os.Getenv("FEISHU_RATE_LIMIT"); rateLimitStr != "" {
		if rateLimit, err := strconv.Atoi(rateLimitStr); err == nil && rateLimit >= 0 {
			cfg.RateLimit = rateLimit
		}
	}

	// Load sign verify from environment
	if signVerifyStr := os.Getenv("FEISHU_SIGN_VERIFY"); signVerifyStr != "" {
		cfg.SignVerify = parseBoolFromEnv(signVerifyStr)
	}

	// Load keywords from environment (comma-separated)
	if keywordsStr := os.Getenv("FEISHU_KEYWORDS"); keywordsStr != "" {
		keywords := strings.Split(keywordsStr, ",")
		for _, keyword := range keywords {
			trimmed := strings.TrimSpace(keyword)
			if trimmed != "" {
				cfg.Keywords = append(cfg.Keywords, trimmed)
			}
		}
	}

	// Set defaults and apply environment precedence
	SetDefaults(cfg)
	return cfg
}

// parseBoolFromEnv safely parses a boolean from environment variable string
func parseBoolFromEnv(str string) bool {
	switch strings.ToLower(str) {
	case "true", "1", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}