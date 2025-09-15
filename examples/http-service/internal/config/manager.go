package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Manager 统一配置管理器
type Manager struct {
	config *Config
}

// Config HTTP服务配置
type Config struct {
	Server    ServerConfig    `json:"server"`
	NotifyHub NotifyHubConfig `json:"notifyhub"`
	Auth      AuthConfig      `json:"auth"`
	Logging   LoggingConfig   `json:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port               int           `json:"port"`
	Host               string        `json:"host"`
	ReadTimeout        time.Duration `json:"read_timeout"`
	WriteTimeout       time.Duration `json:"write_timeout"`
	IdleTimeout        time.Duration `json:"idle_timeout"`
	MaxRequestSize     int64         `json:"max_request_size"`
	RateLimitPerMinute int           `json:"rate_limit_per_minute"`
}

// NotifyHubConfig NotifyHub相关配置
type NotifyHubConfig struct {
	UseDefaults   bool   `json:"use_defaults"`
	LogLevel      string `json:"log_level"`
	MockMode      bool   `json:"mock_mode"`
	AsyncDefault  bool   `json:"async_default"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	APIKey   string `json:"api_key"`
	Required bool   `json:"required"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

// NewManager 创建配置管理器
func NewManager() *Manager {
	return &Manager{}
}

// Load 加载配置
func (m *Manager) Load() (*Config, error) {
	config := m.getDefaultConfig()

	// 从环境变量加载配置
	if err := m.loadFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load from environment: %v", err)
	}

	// 验证配置
	if err := m.validateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	m.config = config
	return config, nil
}

// getDefaultConfig 获取默认配置
func (m *Manager) getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:               8080,
			Host:               "",
			ReadTimeout:        10 * time.Second,
			WriteTimeout:       10 * time.Second,
			IdleTimeout:        120 * time.Second,
			MaxRequestSize:     1024 * 1024, // 1MB
			RateLimitPerMinute: 60,
		},
		NotifyHub: NotifyHubConfig{
			UseDefaults:  true,
			LogLevel:     "info",
			MockMode:     false,
			AsyncDefault: false,
		},
		Auth: AuthConfig{
			APIKey:   "",
			Required: false,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
	}
}

// loadFromEnv 从环境变量加载配置
func (m *Manager) loadFromEnv(config *Config) error {
	// 服务器配置
	if port := getEnvInt("PORT", 0); port > 0 {
		config.Server.Port = port
	}
	if host := getEnvString("HOST", ""); host != "" {
		config.Server.Host = host
	}
	if readTimeout := getEnvDuration("READ_TIMEOUT", 0); readTimeout > 0 {
		config.Server.ReadTimeout = readTimeout
	}
	if writeTimeout := getEnvDuration("WRITE_TIMEOUT", 0); writeTimeout > 0 {
		config.Server.WriteTimeout = writeTimeout
	}
	if maxSize := getEnvInt64("MAX_REQUEST_SIZE", 0); maxSize > 0 {
		config.Server.MaxRequestSize = maxSize
	}
	if rateLimit := getEnvInt("RATE_LIMIT_PER_MINUTE", 0); rateLimit > 0 {
		config.Server.RateLimitPerMinute = rateLimit
	}

	// NotifyHub配置
	if logLevel := getEnvString("LOG_LEVEL", ""); logLevel != "" {
		config.NotifyHub.LogLevel = logLevel
	}
	config.NotifyHub.MockMode = getEnvBool("MOCK_MODE", false)
	config.NotifyHub.AsyncDefault = getEnvBool("ASYNC_DEFAULT", false)

	// 认证配置
	if apiKey := getEnvString("API_KEY", ""); apiKey != "" {
		config.Auth.APIKey = apiKey
		config.Auth.Required = true
	}

	// 日志配置
	if logLevel := getEnvString("LOGGING_LEVEL", ""); logLevel != "" {
		config.Logging.Level = logLevel
	}
	if logFormat := getEnvString("LOGGING_FORMAT", ""); logFormat != "" {
		config.Logging.Format = logFormat
	}

	return nil
}

// validateConfig 验证配置
func (m *Manager) validateConfig(config *Config) error {
	// 验证服务器配置
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Server.Port)
	}

	if config.Server.MaxRequestSize < 1024 {
		return fmt.Errorf("max_request_size too small: %d", config.Server.MaxRequestSize)
	}

	// 验证日志级别
	validLogLevels := []string{"silent", "error", "warn", "info", "debug"}
	if !contains(validLogLevels, config.NotifyHub.LogLevel) {
		return fmt.Errorf("invalid log level: %s", config.NotifyHub.LogLevel)
	}

	if !contains(validLogLevels, config.Logging.Level) {
		return fmt.Errorf("invalid logging level: %s", config.Logging.Level)
	}

	// 验证日志格式
	validFormats := []string{"text", "json"}
	if !contains(validFormats, config.Logging.Format) {
		return fmt.Errorf("invalid logging format: %s", config.Logging.Format)
	}

	return nil
}

// GetConfig 获取当前配置
func (m *Manager) GetConfig() *Config {
	return m.config
}

// 工具函数

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}