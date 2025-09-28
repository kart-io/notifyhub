// Package feishu provides configuration management for Feishu platform
// This file handles configuration validation and default value settings
package feishu

import (
	"fmt"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
)

// ValidateConfig validates the Feishu configuration
func ValidateConfig(cfg *config.FeishuConfig) error {
	if cfg == nil {
		return fmt.Errorf("feishu config cannot be nil")
	}

	// Validate authentication configuration based on auth type
	switch cfg.AuthType {
	case "", "webhook":
		if cfg.WebhookURL == "" {
			return fmt.Errorf("webhook_url is required for webhook authentication")
		}
		// Validate webhook URL format
		if !strings.HasPrefix(cfg.WebhookURL, "http://") && !strings.HasPrefix(cfg.WebhookURL, "https://") {
			return fmt.Errorf("webhook_url must start with http:// or https://")
		}
	case "app":
		if cfg.AppID == "" {
			return fmt.Errorf("app_id is required for app authentication")
		}
		if cfg.AppSecret == "" {
			return fmt.Errorf("app_secret is required for app authentication")
		}
	default:
		return fmt.Errorf("invalid auth_type: %s (must be 'webhook' or 'app')", cfg.AuthType)
	}

	// Validate timeout
	if cfg.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	// Validate max retries
	if cfg.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}

	// Validate rate limit
	if cfg.RateLimit < 0 {
		return fmt.Errorf("rate_limit cannot be negative")
	}

	// Validate keywords (if provided)
	for i, keyword := range cfg.Keywords {
		trimmed := strings.TrimSpace(keyword)
		if trimmed == "" {
			return fmt.Errorf("keyword at index %d cannot be empty", i)
		}
		// Update the keyword with trimmed value
		cfg.Keywords[i] = trimmed
	}

	return nil
}

// SetDefaults sets default values for Feishu configuration
func SetDefaults(cfg *config.FeishuConfig) {
	// Set default timeout
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// Set default max retries
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	// Set default rate limit (requests per minute)
	if cfg.RateLimit == 0 {
		cfg.RateLimit = 60
	}

	// Set default auth type
	if cfg.AuthType == "" {
		if cfg.WebhookURL != "" {
			cfg.AuthType = "webhook"
		} else if cfg.AppID != "" {
			cfg.AuthType = "app"
		} else {
			cfg.AuthType = "webhook" // Default to webhook
		}
	}

	// Clean up keywords: trim whitespace and remove empty ones
	cleanKeywords := make([]string, 0, len(cfg.Keywords))
	for _, keyword := range cfg.Keywords {
		trimmed := strings.TrimSpace(keyword)
		if trimmed != "" {
			cleanKeywords = append(cleanKeywords, trimmed)
		}
	}
	cfg.Keywords = cleanKeywords
}

// NewConfigFromMap creates a FeishuConfig from a map (for backward compatibility)
func NewConfigFromMap(configMap map[string]interface{}) (*config.FeishuConfig, error) {
	cfg := &config.FeishuConfig{}

	// Extract webhook configuration
	if webhookURL, ok := configMap["webhook_url"].(string); ok {
		cfg.WebhookURL = webhookURL
	}
	if secret, ok := configMap["secret"].(string); ok {
		cfg.Secret = secret
	}

	// Extract app configuration
	if appID, ok := configMap["app_id"].(string); ok {
		cfg.AppID = appID
	}
	if appSecret, ok := configMap["app_secret"].(string); ok {
		cfg.AppSecret = appSecret
	}
	if authType, ok := configMap["auth_type"].(string); ok {
		cfg.AuthType = authType
	}

	// Extract timeout
	if timeout, ok := configMap["timeout"].(time.Duration); ok {
		cfg.Timeout = timeout
	} else if timeoutStr, ok := configMap["timeout"].(string); ok {
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.Timeout = parsed
		}
	} else if timeoutFloat, ok := configMap["timeout"].(float64); ok {
		cfg.Timeout = time.Duration(timeoutFloat) * time.Second
	}

	// Extract max retries
	if maxRetries, ok := configMap["max_retries"].(int); ok {
		cfg.MaxRetries = maxRetries
	} else if maxRetriesFloat, ok := configMap["max_retries"].(float64); ok {
		cfg.MaxRetries = int(maxRetriesFloat)
	}

	// Extract rate limit
	if rateLimit, ok := configMap["rate_limit"].(int); ok {
		cfg.RateLimit = rateLimit
	} else if rateLimitFloat, ok := configMap["rate_limit"].(float64); ok {
		cfg.RateLimit = int(rateLimitFloat)
	}

	// Extract sign verify
	if signVerify, ok := configMap["sign_verify"].(bool); ok {
		cfg.SignVerify = signVerify
	}

	// Extract keywords
	if keywordsInterface, ok := configMap["keywords"]; ok {
		if keywordSlice, ok := keywordsInterface.([]interface{}); ok {
			for _, keyword := range keywordSlice {
				if keywordStr, ok := keyword.(string); ok {
					cfg.Keywords = append(cfg.Keywords, keywordStr)
				}
			}
		} else if keywordsStr, ok := keywordsInterface.(string); ok {
			// Support comma-separated string format
			if keywordsStr != "" {
				keywords := strings.Split(keywordsStr, ",")
				for _, keyword := range keywords {
					trimmed := strings.TrimSpace(keyword)
					if trimmed != "" {
						cfg.Keywords = append(cfg.Keywords, trimmed)
					}
				}
			}
		} else if keywordSlice, ok := keywordsInterface.([]string); ok {
			cfg.Keywords = keywordSlice
		}
	}

	// Set defaults and validate
	SetDefaults(cfg)
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}



