// Package feishu provides configuration management for Feishu platform
// This file handles configuration validation and default value settings
package feishu

import (
	"fmt"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/config"
)

// ValidateConfig validates the Feishu configuration
func ValidateConfig(cfg *config.FeishuConfig) error {
	if cfg == nil {
		return fmt.Errorf("feishu config cannot be nil")
	}

	// Validate webhook URL
	if cfg.WebhookURL == "" {
		return fmt.Errorf("webhook_url is required for Feishu platform")
	}

	// Validate webhook URL format
	if !strings.HasPrefix(cfg.WebhookURL, "http://") && !strings.HasPrefix(cfg.WebhookURL, "https://") {
		return fmt.Errorf("webhook_url must start with http:// or https://")
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

	// No auth type needed - using webhook only

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

// extractString safely extracts a string from a map
func extractString(configMap map[string]interface{}, key string) string {
	if value, ok := configMap[key].(string); ok {
		return value
	}
	return ""
}

// extractInt safely extracts an int from a map (handles both int and float64)
func extractInt(configMap map[string]interface{}, key string) int {
	if value, ok := configMap[key].(int); ok {
		return value
	}
	if valueFloat, ok := configMap[key].(float64); ok {
		return int(valueFloat)
	}
	return 0
}

// extractBool safely extracts a bool from a map
func extractBool(configMap map[string]interface{}, key string) bool {
	if value, ok := configMap[key].(bool); ok {
		return value
	}
	return false
}

// extractTimeout safely extracts a timeout duration from a map
func extractTimeout(configMap map[string]interface{}, key string) time.Duration {
	// Try time.Duration first
	if timeout, ok := configMap[key].(time.Duration); ok {
		return timeout
	}

	// Try string format (e.g., "30s", "5m")
	if timeoutStr, ok := configMap[key].(string); ok {
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			return parsed
		}
	}

	// Try float64 (seconds)
	if timeoutFloat, ok := configMap[key].(float64); ok {
		return time.Duration(timeoutFloat) * time.Second
	}

	return 0
}

// extractKeywords safely extracts keywords from various formats
func extractKeywords(configMap map[string]interface{}, key string, existing []string) []string {
	keywordsInterface, ok := configMap[key]
	if !ok {
		return existing
	}

	// Try []string directly
	if keywordSlice, ok := keywordsInterface.([]string); ok {
		return keywordSlice
	}

	// Try []interface{} and convert each element
	if keywordSlice, ok := keywordsInterface.([]interface{}); ok {
		var keywords []string
		for _, keyword := range keywordSlice {
			if keywordStr, ok := keyword.(string); ok {
				keywords = append(keywords, keywordStr)
			}
		}
		return keywords
	}

	// Try comma-separated string format
	if keywordsStr, ok := keywordsInterface.(string); ok {
		if keywordsStr != "" {
			var keywords []string
			for _, keyword := range strings.Split(keywordsStr, ",") {
				trimmed := strings.TrimSpace(keyword)
				if trimmed != "" {
					keywords = append(keywords, trimmed)
				}
			}
			return keywords
		}
	}

	return existing
}

// NewConfigFromMap creates a FeishuConfig from a map (for backward compatibility)
func NewConfigFromMap(configMap map[string]interface{}) (*config.FeishuConfig, error) {
	cfg := &config.FeishuConfig{
		WebhookURL: extractString(configMap, "webhook_url"),
		Secret:     extractString(configMap, "secret"),
		MaxRetries: extractInt(configMap, "max_retries"),
		RateLimit:  extractInt(configMap, "rate_limit"),
		VerifySSL:  extractBool(configMap, "verify_ssl"),
		Timeout:    extractTimeout(configMap, "timeout"),
	}

	// Extract keywords with special handling
	cfg.Keywords = extractKeywords(configMap, "keywords", cfg.Keywords)

	// Set defaults and validate
	SetDefaults(cfg)
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
