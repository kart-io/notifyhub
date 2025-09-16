package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// ConfigValidationResult represents the result of configuration validation
type ConfigValidationResult struct {
	Valid     bool                       `json:"valid"`
	Errors    []ValidationError          `json:"errors,omitempty"`
	Warnings  []ValidationWarning        `json:"warnings,omitempty"`
	Summary   *ValidationSummary         `json:"summary"`
	Details   map[string]interface{}     `json:"details,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Component   string   `json:"component"`
	Field       string   `json:"field"`
	Message     string   `json:"message"`
	Severity    string   `json:"severity"` // "critical", "error", "warning"
	Suggestions []string `json:"suggestions,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Component string `json:"component"`
	Field     string `json:"field,omitempty"`
	Message   string `json:"message"`
	Impact    string `json:"impact"` // "performance", "reliability", "security"
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalChecks     int `json:"total_checks"`
	PassedChecks    int `json:"passed_checks"`
	ErrorCount      int `json:"error_count"`
	WarningCount    int `json:"warning_count"`
	CriticalErrors  int `json:"critical_errors"`
}

// ConfigValidator validates hub configuration
type ConfigValidator struct {
	hub     *Hub
	checks  []ValidationCheck
	timeout time.Duration
}

// ValidationCheck represents a single validation check
type ValidationCheck struct {
	Name        string
	Component   string
	CheckFunc   func(*Hub, *ValidationContext) *ConfigValidationResult
	Required    bool
	Description string
}

// ValidationContext provides context for validation checks
type ValidationContext struct {
	Timeout         time.Duration
	SkipNetworkTest bool
	Strict          bool
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(hub *Hub) *ConfigValidator {
	validator := &ConfigValidator{
		hub:     hub,
		checks:  make([]ValidationCheck, 0),
		timeout: 30 * time.Second,
	}

	// Register default validation checks
	validator.registerDefaultChecks()
	return validator
}

// WithTimeout sets validation timeout
func (cv *ConfigValidator) WithTimeout(timeout time.Duration) *ConfigValidator {
	cv.timeout = timeout
	return cv
}

// AddCheck adds a custom validation check
func (cv *ConfigValidator) AddCheck(check ValidationCheck) {
	cv.checks = append(cv.checks, check)
}

// Validate validates the hub configuration
func (cv *ConfigValidator) Validate(ctx context.Context, validationCtx *ValidationContext) *ConfigValidationResult {
	if validationCtx == nil {
		validationCtx = &ValidationContext{
			Timeout:         cv.timeout,
			SkipNetworkTest: false,
			Strict:          false,
		}
	}

	result := &ConfigValidationResult{
		Valid:   true,
		Errors:  make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
		Details: make(map[string]interface{}),
	}

	totalChecks := len(cv.checks)
	passedChecks := 0

	// Run all validation checks
	for _, check := range cv.checks {
		checkResult := check.CheckFunc(cv.hub, validationCtx)

		if checkResult != nil {
			// Merge errors and warnings
			result.Errors = append(result.Errors, checkResult.Errors...)
			result.Warnings = append(result.Warnings, checkResult.Warnings...)

			// Merge details
			for k, v := range checkResult.Details {
				result.Details[fmt.Sprintf("%s_%s", check.Component, k)] = v
			}

			if !checkResult.Valid && check.Required {
				result.Valid = false
			}
		} else {
			passedChecks++
		}

		// Check context timeout
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, ValidationError{
				Component: "validator",
				Field:     "timeout",
				Message:   "Validation timeout exceeded",
				Severity:  "error",
			})
			result.Valid = false
			break
		default:
		}
	}

	// Calculate summary
	errorCount := len(result.Errors)
	warningCount := len(result.Warnings)
	criticalErrors := 0

	for _, err := range result.Errors {
		if err.Severity == "critical" {
			criticalErrors++
		}
	}

	result.Summary = &ValidationSummary{
		TotalChecks:    totalChecks,
		PassedChecks:   passedChecks,
		ErrorCount:     errorCount,
		WarningCount:   warningCount,
		CriticalErrors: criticalErrors,
	}

	return result
}

// registerDefaultChecks registers all default validation checks
func (cv *ConfigValidator) registerDefaultChecks() {
	// Feishu configuration validation
	cv.AddCheck(ValidationCheck{
		Name:        "feishu_config",
		Component:   "feishu",
		CheckFunc:   cv.validateFeishuConfig,
		Required:    false,
		Description: "Validates Feishu webhook configuration",
	})

	// Email configuration validation
	cv.AddCheck(ValidationCheck{
		Name:        "email_config",
		Component:   "email",
		CheckFunc:   cv.validateEmailConfig,
		Required:    false,
		Description: "Validates email SMTP configuration",
	})

	// Queue configuration validation
	cv.AddCheck(ValidationCheck{
		Name:        "queue_config",
		Component:   "queue",
		CheckFunc:   cv.validateQueueConfig,
		Required:    true,
		Description: "Validates queue configuration",
	})

	// Template configuration validation
	cv.AddCheck(ValidationCheck{
		Name:        "template_config",
		Component:   "templates",
		CheckFunc:   cv.validateTemplateConfig,
		Required:    false,
		Description: "Validates template configuration",
	})

	// Routing configuration validation
	cv.AddCheck(ValidationCheck{
		Name:        "routing_config",
		Component:   "routing",
		CheckFunc:   cv.validateRoutingConfig,
		Required:    false,
		Description: "Validates routing rules",
	})

	// Logger configuration validation
	cv.AddCheck(ValidationCheck{
		Name:        "logger_config",
		Component:   "logger",
		CheckFunc:   cv.validateLoggerConfig,
		Required:    true,
		Description: "Validates logger configuration",
	})

	// Telemetry configuration validation
	cv.AddCheck(ValidationCheck{
		Name:        "telemetry_config",
		Component:   "telemetry",
		CheckFunc:   cv.validateTelemetryConfig,
		Required:    false,
		Description: "Validates telemetry configuration",
	})
}

// validateFeishuConfig validates Feishu configuration
func (cv *ConfigValidator) validateFeishuConfig(hub *Hub, ctx *ValidationContext) *ConfigValidationResult {
	feishuConfig := hub.config.Feishu()
	if feishuConfig == nil {
		return &ConfigValidationResult{
			Valid: true,
			Warnings: []ValidationWarning{
				{
					Component: "feishu",
					Message:   "Feishu configuration not provided",
					Impact:    "reliability",
				},
			},
			Details: map[string]interface{}{"configured": false},
		}
	}

	result := &ConfigValidationResult{
		Valid:   true,
		Errors:  make([]ValidationError, 0),
		Details: map[string]interface{}{"configured": true},
	}

	// Validate webhook URL
	if feishuConfig.WebhookURL == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Component:   "feishu",
			Field:       "webhook_url",
			Message:     "Webhook URL is required",
			Severity:    "critical",
			Suggestions: []string{"Set NOTIFYHUB_FEISHU_WEBHOOK_URL environment variable"},
		})
	} else {
		// Validate URL format
		parsedURL, err := url.Parse(feishuConfig.WebhookURL)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Component:   "feishu",
				Field:       "webhook_url",
				Message:     fmt.Sprintf("Invalid webhook URL format: %v", err),
				Severity:    "critical",
				Suggestions: []string{"Provide a valid HTTP/HTTPS URL"},
			})
		} else {
			if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
				result.Warnings = append(result.Warnings, ValidationWarning{
					Component: "feishu",
					Field:     "webhook_url",
					Message:   "Webhook URL should use HTTPS for security",
					Impact:    "security",
				})
			}

			result.Details["webhook_scheme"] = parsedURL.Scheme
			result.Details["webhook_host"] = parsedURL.Host
		}
	}

	// Validate timeout
	if feishuConfig.Timeout <= 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "feishu",
			Field:     "timeout",
			Message:   "Timeout should be positive",
			Impact:    "reliability",
		})
	} else if feishuConfig.Timeout > 60*time.Second {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "feishu",
			Field:     "timeout",
			Message:   "Timeout is very high, may cause performance issues",
			Impact:    "performance",
		})
	}

	result.Details["timeout"] = feishuConfig.Timeout.String()

	return result
}

// validateEmailConfig validates email configuration
func (cv *ConfigValidator) validateEmailConfig(hub *Hub, ctx *ValidationContext) *ConfigValidationResult {
	emailConfig := hub.config.Email()
	if emailConfig == nil {
		return &ConfigValidationResult{
			Valid: true,
			Warnings: []ValidationWarning{
				{
					Component: "email",
					Message:   "Email configuration not provided",
					Impact:    "reliability",
				},
			},
			Details: map[string]interface{}{"configured": false},
		}
	}

	result := &ConfigValidationResult{
		Valid:   true,
		Errors:  make([]ValidationError, 0),
		Details: map[string]interface{}{"configured": true},
	}

	// Validate SMTP host
	if emailConfig.Host == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Component:   "email",
			Field:       "host",
			Message:     "SMTP host is required",
			Severity:    "critical",
			Suggestions: []string{"Set NOTIFYHUB_SMTP_HOST environment variable"},
		})
	}

	// Validate port
	if emailConfig.Port <= 0 || emailConfig.Port > 65535 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Component:   "email",
			Field:       "port",
			Message:     "Invalid SMTP port",
			Severity:    "error",
			Suggestions: []string{"Use common ports: 25, 587, 465, 993, 995"},
		})
	} else {
		// Check for common secure ports
		if emailConfig.Port != 587 && emailConfig.Port != 465 && emailConfig.Port != 993 && emailConfig.Port != 995 {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Component: "email",
				Field:     "port",
				Message:   "Using non-standard SMTP port",
				Impact:    "reliability",
			})
		}
	}

	// Validate from email
	if emailConfig.From == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Component:   "email",
			Field:       "from",
			Message:     "From email address is required",
			Severity:    "critical",
			Suggestions: []string{"Set NOTIFYHUB_SMTP_FROM environment variable"},
		})
	} else if !strings.Contains(emailConfig.From, "@") {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Component:   "email",
			Field:       "from",
			Message:     "Invalid from email address format",
			Severity:    "error",
			Suggestions: []string{"Provide a valid email address"},
		})
	}

	// Validate authentication
	if emailConfig.Username != "" && emailConfig.Password == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "email",
			Field:     "password",
			Message:   "Username provided but password is empty",
			Impact:    "reliability",
		})
	}

	// Validate TLS usage
	if !emailConfig.UseTLS && emailConfig.Port != 25 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "email",
			Field:     "use_tls",
			Message:   "TLS not enabled for secure port",
			Impact:    "security",
		})
	}

	// Validate timeout
	if emailConfig.Timeout <= 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "email",
			Field:     "timeout",
			Message:   "Timeout should be positive",
			Impact:    "reliability",
		})
	}

	result.Details["host"] = emailConfig.Host
	result.Details["port"] = emailConfig.Port
	result.Details["use_tls"] = emailConfig.UseTLS
	result.Details["timeout"] = emailConfig.Timeout.String()

	return result
}

// validateQueueConfig validates queue configuration
func (cv *ConfigValidator) validateQueueConfig(hub *Hub, ctx *ValidationContext) *ConfigValidationResult {
	queueConfig := hub.config.Queue()

	result := &ConfigValidationResult{
		Valid:   true,
		Errors:  make([]ValidationError, 0),
		Details: make(map[string]interface{}),
	}

	if queueConfig == nil {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "queue",
			Message:   "Queue configuration not provided, using defaults",
			Impact:    "performance",
		})
		result.Details["configured"] = false
		result.Details["using_defaults"] = true
		return result
	}

	result.Details["configured"] = true

	// Validate buffer size
	if queueConfig.BufferSize <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Component:   "queue",
			Field:       "buffer_size",
			Message:     "Buffer size must be positive",
			Severity:    "error",
			Suggestions: []string{"Set buffer size to at least 100"},
		})
	} else if queueConfig.BufferSize < 10 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "queue",
			Field:     "buffer_size",
			Message:   "Buffer size is very small, may cause blocking",
			Impact:    "performance",
		})
	} else if queueConfig.BufferSize > 10000 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "queue",
			Field:     "buffer_size",
			Message:   "Buffer size is very large, may consume excessive memory",
			Impact:    "performance",
		})
	}

	// Validate worker count
	if queueConfig.Workers <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Component:   "queue",
			Field:       "workers",
			Message:     "Worker count must be positive",
			Severity:    "error",
			Suggestions: []string{"Set worker count to at least 1"},
		})
	} else if queueConfig.Workers > 50 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "queue",
			Field:     "workers",
			Message:   "High worker count may cause resource contention",
			Impact:    "performance",
		})
	}

	// Validate queue type
	if queueConfig.Type == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Component:   "queue",
			Field:       "type",
			Message:     "Queue type is required",
			Severity:    "critical",
			Suggestions: []string{"Use 'memory' for simple cases or 'redis' for production"},
		})
	} else {
		validTypes := []string{"memory", "redis"}
		valid := false
		for _, validType := range validTypes {
			if queueConfig.Type == validType {
				valid = true
				break
			}
		}
		if !valid {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Component:   "queue",
				Field:       "type",
				Message:     fmt.Sprintf("Invalid queue type: %s", queueConfig.Type),
				Severity:    "error",
				Suggestions: []string{"Use one of: " + strings.Join(validTypes, ", ")},
			})
		}
	}

	result.Details["type"] = queueConfig.Type
	result.Details["buffer_size"] = queueConfig.BufferSize
	result.Details["workers"] = queueConfig.Workers

	return result
}

// validateTemplateConfig validates template configuration
func (cv *ConfigValidator) validateTemplateConfig(hub *Hub, ctx *ValidationContext) *ConfigValidationResult {
	if hub.templates == nil {
		return &ConfigValidationResult{
			Valid: true,
			Warnings: []ValidationWarning{
				{
					Component: "templates",
					Message:   "Template engine not initialized",
					Impact:    "reliability",
				},
			},
			Details: map[string]interface{}{"initialized": false},
		}
	}

	result := &ConfigValidationResult{
		Valid:   true,
		Details: map[string]interface{}{"initialized": true},
	}

	// TODO: Add template-specific validation when template engine methods are available
	return result
}

// validateRoutingConfig validates routing configuration
func (cv *ConfigValidator) validateRoutingConfig(hub *Hub, ctx *ValidationContext) *ConfigValidationResult {
	routingConfig := hub.config.Routing()
	if routingConfig == nil {
		return &ConfigValidationResult{
			Valid: true,
			Warnings: []ValidationWarning{
				{
					Component: "routing",
					Message:   "Routing configuration not provided",
					Impact:    "reliability",
				},
			},
			Details: map[string]interface{}{"configured": false},
		}
	}

	result := &ConfigValidationResult{
		Valid:   true,
		Details: map[string]interface{}{"configured": true, "rules_count": len(routingConfig.Rules)},
	}

	// Validate routing rules
	for i, rule := range routingConfig.Rules {
		if rule.Name == "" {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Component: "routing",
				Field:     fmt.Sprintf("rules[%d].name", i),
				Message:   "Routing rule should have a name",
				Impact:    "reliability",
			})
		}

		if len(rule.Actions) == 0 {
			result.Errors = append(result.Errors, ValidationError{
				Component:   "routing",
				Field:       fmt.Sprintf("rules[%d].actions", i),
				Message:     "Routing rule must specify actions",
				Severity:    "error",
				Suggestions: []string{"Add at least one rule action"},
			})
			result.Valid = false
		} else {
			// Check that at least one action has platforms
			hasPlatforms := false
			for j, action := range rule.Actions {
				if len(action.Platforms) > 0 {
					hasPlatforms = true
					break
				} else {
					result.Warnings = append(result.Warnings, ValidationWarning{
						Component: "routing",
						Field:     fmt.Sprintf("rules[%d].actions[%d].platforms", i, j),
						Message:   "Action should specify target platforms",
						Impact:    "reliability",
					})
				}
			}
			if !hasPlatforms {
				result.Errors = append(result.Errors, ValidationError{
					Component:   "routing",
					Field:       fmt.Sprintf("rules[%d]", i),
					Message:     "Routing rule must have at least one action with platforms",
					Severity:    "error",
					Suggestions: []string{"Add target platforms to at least one action"},
				})
				result.Valid = false
			}
		}
	}

	return result
}

// validateLoggerConfig validates logger configuration
func (cv *ConfigValidator) validateLoggerConfig(hub *Hub, ctx *ValidationContext) *ConfigValidationResult {
	result := &ConfigValidationResult{
		Valid:   true,
		Details: make(map[string]interface{}),
	}

	if hub.logger == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Component:   "logger",
			Message:     "Logger not initialized",
			Severity:    "critical",
			Suggestions: []string{"Initialize logger in hub configuration"},
		})
		result.Details["initialized"] = false
	} else {
		result.Details["initialized"] = true
	}

	return result
}

// validateTelemetryConfig validates telemetry configuration
func (cv *ConfigValidator) validateTelemetryConfig(hub *Hub, ctx *ValidationContext) *ConfigValidationResult {
	telemetryConfig := hub.config.Telemetry()
	if telemetryConfig == nil {
		return &ConfigValidationResult{
			Valid: true,
			Warnings: []ValidationWarning{
				{
					Component: "telemetry",
					Message:   "Telemetry configuration not provided",
					Impact:    "reliability",
				},
			},
			Details: map[string]interface{}{"configured": false},
		}
	}

	result := &ConfigValidationResult{
		Valid:   true,
		Details: map[string]interface{}{"configured": true},
	}

	// Validate service info
	if telemetryConfig.ServiceName == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "telemetry",
			Field:     "service_name",
			Message:   "Service name not specified",
			Impact:    "reliability",
		})
	}

	if telemetryConfig.ServiceVersion == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component: "telemetry",
			Field:     "service_version",
			Message:   "Service version not specified",
			Impact:    "reliability",
		})
	}

	// Validate endpoints if telemetry is enabled
	if telemetryConfig.Enabled {
		if (telemetryConfig.TracingEnabled || telemetryConfig.MetricsEnabled) && telemetryConfig.OTLPEndpoint == "" {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Component: "telemetry",
				Field:     "otlp_endpoint",
				Message:   "Telemetry enabled but no OTLP endpoint specified",
				Impact:    "reliability",
			})
		}
	}

	result.Details["enabled"] = telemetryConfig.Enabled
	result.Details["tracing_enabled"] = telemetryConfig.TracingEnabled
	result.Details["metrics_enabled"] = telemetryConfig.MetricsEnabled

	return result
}

// Hub validation methods

// ValidateConfiguration validates hub configuration on startup
func (h *Hub) ValidateConfiguration(ctx context.Context) *ConfigValidationResult {
	validator := NewConfigValidator(h)
	return validator.Validate(ctx, nil)
}

// ValidateConfigurationStrict validates hub configuration with strict checks
func (h *Hub) ValidateConfigurationStrict(ctx context.Context) *ConfigValidationResult {
	validator := NewConfigValidator(h)
	return validator.Validate(ctx, &ValidationContext{
		Timeout:         30 * time.Second,
		SkipNetworkTest: false,
		Strict:          true,
	})
}

// ValidateAndReport validates configuration and logs results
func (h *Hub) ValidateAndReport(ctx context.Context) error {
	result := h.ValidateConfiguration(ctx)

	if len(result.Errors) > 0 {
		h.logger.Error(ctx, "Configuration validation failed:")
		for _, err := range result.Errors {
			h.logger.Error(ctx, "  - [%s] %s: %s", err.Component, err.Field, err.Message)
		}
	}

	if len(result.Warnings) > 0 {
		h.logger.Info(ctx, "Configuration validation warnings:")
		for _, warn := range result.Warnings {
			h.logger.Info(ctx, "  - [%s] %s: %s (Impact: %s)", warn.Component, warn.Field, warn.Message, warn.Impact)
		}
	}

	if !result.Valid {
		return fmt.Errorf("configuration validation failed with %d errors", len(result.Errors))
	}

	h.logger.Info(ctx, "Configuration validation passed: %d/%d checks passed, %d warnings",
		result.Summary.PassedChecks, result.Summary.TotalChecks, result.Summary.WarningCount)

	return nil
}