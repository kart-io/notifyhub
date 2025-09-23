// Package config provides configuration management for NotifyHub
package config

import (
	"time"
)

// HubConfig represents the main configuration for NotifyHub
type HubConfig struct {
	Platforms        map[string]PlatformConfig `json:"platforms"`
	DefaultTimeout   time.Duration             `json:"default_timeout"`
	RetryPolicy      RetryPolicy               `json:"retry_policy"`
	ValidationErrors []error                   `json:"-"` // Collect validation errors during configuration
}

// PlatformConfig represents configuration for a specific platform
type PlatformConfig map[string]interface{}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialInterval time.Duration `json:"initial_interval"`
	Multiplier      float64       `json:"multiplier"`
	MaxInterval     time.Duration `json:"max_interval"`
}