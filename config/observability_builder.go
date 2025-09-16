package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ================================
// Enhanced Observability Configuration Builder
// ================================

// ObservabilityBuilder provides a fluent API for configuring observability
type ObservabilityBuilder struct {
	config *TelemetryConfig
	errors []error
}

// NewObservabilityBuilder creates a new observability configuration builder
func NewObservabilityBuilder() *ObservabilityBuilder {
	return &ObservabilityBuilder{
		config: &TelemetryConfig{
			ServiceName:    "notifyhub",
			ServiceVersion: "1.2.0",
			Environment:    "development",
			OTLPEndpoint:   "http://localhost:4318",
			OTLPHeaders:    make(map[string]string),
			TracingEnabled: true,
			MetricsEnabled: true,
			SampleRate:     1.0,
			Enabled:        true,
		},
		errors: make([]error, 0),
	}
}

// ServiceName sets the service name
func (ob *ObservabilityBuilder) ServiceName(name string) *ObservabilityBuilder {
	if strings.TrimSpace(name) == "" {
		ob.errors = append(ob.errors, fmt.Errorf("service name cannot be empty"))
		return ob
	}
	ob.config.ServiceName = strings.TrimSpace(name)
	return ob
}

// ServiceVersion sets the service version
func (ob *ObservabilityBuilder) ServiceVersion(version string) *ObservabilityBuilder {
	if strings.TrimSpace(version) == "" {
		ob.errors = append(ob.errors, fmt.Errorf("service version cannot be empty"))
		return ob
	}
	ob.config.ServiceVersion = strings.TrimSpace(version)
	return ob
}

// Environment sets the environment (dev, staging, prod, etc.)
func (ob *ObservabilityBuilder) Environment(env string) *ObservabilityBuilder {
	if strings.TrimSpace(env) == "" {
		ob.errors = append(ob.errors, fmt.Errorf("environment cannot be empty"))
		return ob
	}
	ob.config.Environment = strings.TrimSpace(env)
	return ob
}

// OTLPEndpoint sets the OTLP endpoint URL
func (ob *ObservabilityBuilder) OTLPEndpoint(endpoint string) *ObservabilityBuilder {
	if strings.TrimSpace(endpoint) == "" {
		ob.errors = append(ob.errors, fmt.Errorf("OTLP endpoint cannot be empty"))
		return ob
	}

	// Basic URL validation
	endpoint = strings.TrimSpace(endpoint)
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		ob.errors = append(ob.errors, fmt.Errorf("OTLP endpoint must start with http:// or https://"))
		return ob
	}

	ob.config.OTLPEndpoint = endpoint
	return ob
}

// OTLPHeader adds an OTLP header
func (ob *ObservabilityBuilder) OTLPHeader(key, value string) *ObservabilityBuilder {
	if strings.TrimSpace(key) == "" {
		ob.errors = append(ob.errors, fmt.Errorf("OTLP header key cannot be empty"))
		return ob
	}
	if strings.TrimSpace(value) == "" {
		ob.errors = append(ob.errors, fmt.Errorf("OTLP header value cannot be empty"))
		return ob
	}
	ob.config.OTLPHeaders[strings.TrimSpace(key)] = strings.TrimSpace(value)
	return ob
}

// OTLPHeaders sets multiple OTLP headers
func (ob *ObservabilityBuilder) OTLPHeaders(headers map[string]string) *ObservabilityBuilder {
	for key, value := range headers {
		ob.OTLPHeader(key, value)
	}
	return ob
}

// Authentication adds authentication header
func (ob *ObservabilityBuilder) Authentication(auth string) *ObservabilityBuilder {
	return ob.OTLPHeader("Authorization", auth)
}

// BearerToken adds bearer token authentication
func (ob *ObservabilityBuilder) BearerToken(token string) *ObservabilityBuilder {
	if strings.TrimSpace(token) == "" {
		ob.errors = append(ob.errors, fmt.Errorf("bearer token cannot be empty"))
		return ob
	}
	return ob.OTLPHeader("Authorization", "Bearer "+strings.TrimSpace(token))
}

// APIKey adds API key authentication
func (ob *ObservabilityBuilder) APIKey(key string) *ObservabilityBuilder {
	if strings.TrimSpace(key) == "" {
		ob.errors = append(ob.errors, fmt.Errorf("API key cannot be empty"))
		return ob
	}
	return ob.OTLPHeader("X-API-Key", strings.TrimSpace(key))
}

// EnableTracing enables distributed tracing
func (ob *ObservabilityBuilder) EnableTracing() *ObservabilityBuilder {
	ob.config.TracingEnabled = true
	return ob
}

// DisableTracing disables distributed tracing
func (ob *ObservabilityBuilder) DisableTracing() *ObservabilityBuilder {
	ob.config.TracingEnabled = false
	return ob
}

// EnableMetrics enables metrics collection
func (ob *ObservabilityBuilder) EnableMetrics() *ObservabilityBuilder {
	ob.config.MetricsEnabled = true
	return ob
}

// DisableMetrics disables metrics collection
func (ob *ObservabilityBuilder) DisableMetrics() *ObservabilityBuilder {
	ob.config.MetricsEnabled = false
	return ob
}

// SampleRate sets the tracing sample rate (0.0 to 1.0)
func (ob *ObservabilityBuilder) SampleRate(rate float64) *ObservabilityBuilder {
	if rate < 0.0 || rate > 1.0 {
		ob.errors = append(ob.errors, fmt.Errorf("sample rate must be between 0.0 and 1.0, got %f", rate))
		return ob
	}
	ob.config.SampleRate = rate
	return ob
}

// Enable enables observability
func (ob *ObservabilityBuilder) Enable() *ObservabilityBuilder {
	ob.config.Enabled = true
	return ob
}

// Disable disables observability
func (ob *ObservabilityBuilder) Disable() *ObservabilityBuilder {
	ob.config.Enabled = false
	return ob
}

// ================================
// Environment-based Configuration
// ================================

// FromEnvironment configures from environment variables
func (ob *ObservabilityBuilder) FromEnvironment() *ObservabilityBuilder {
	if enabled := os.Getenv("NOTIFYHUB_TELEMETRY_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			if val {
				ob.Enable()
			} else {
				ob.Disable()
			}
		}
	}

	if serviceName := os.Getenv("NOTIFYHUB_SERVICE_NAME"); serviceName != "" {
		ob.ServiceName(serviceName)
	}

	if serviceVersion := os.Getenv("NOTIFYHUB_SERVICE_VERSION"); serviceVersion != "" {
		ob.ServiceVersion(serviceVersion)
	}

	if environment := os.Getenv("NOTIFYHUB_ENVIRONMENT"); environment != "" {
		ob.Environment(environment)
	}

	if endpoint := os.Getenv("NOTIFYHUB_OTLP_ENDPOINT"); endpoint != "" {
		ob.OTLPEndpoint(endpoint)
	}

	if auth := os.Getenv("NOTIFYHUB_OTLP_AUTH"); auth != "" {
		ob.Authentication(auth)
	}

	if apiKey := os.Getenv("NOTIFYHUB_OTLP_API_KEY"); apiKey != "" {
		ob.APIKey(apiKey)
	}

	if tracing := os.Getenv("NOTIFYHUB_TRACING_ENABLED"); tracing != "" {
		if val, err := strconv.ParseBool(tracing); err == nil {
			if val {
				ob.EnableTracing()
			} else {
				ob.DisableTracing()
			}
		}
	}

	if metrics := os.Getenv("NOTIFYHUB_METRICS_ENABLED"); metrics != "" {
		if val, err := strconv.ParseBool(metrics); err == nil {
			if val {
				ob.EnableMetrics()
			} else {
				ob.DisableMetrics()
			}
		}
	}

	if sampleRate := os.Getenv("NOTIFYHUB_SAMPLE_RATE"); sampleRate != "" {
		if val, err := strconv.ParseFloat(sampleRate, 64); err == nil {
			ob.SampleRate(val)
		}
	}

	return ob
}

// ================================
// Preset Configurations
// ================================

// Development configures for development environment
func (ob *ObservabilityBuilder) Development() *ObservabilityBuilder {
	return ob.
		Environment("development").
		OTLPEndpoint("http://localhost:4318").
		SampleRate(1.0).
		EnableTracing().
		EnableMetrics().
		Enable()
}

// Production configures for production environment
func (ob *ObservabilityBuilder) Production() *ObservabilityBuilder {
	return ob.
		Environment("production").
		SampleRate(0.1). // Lower sample rate for production
		EnableTracing().
		EnableMetrics().
		Enable()
}

// Staging configures for staging environment
func (ob *ObservabilityBuilder) Staging() *ObservabilityBuilder {
	return ob.
		Environment("staging").
		SampleRate(0.5). // Medium sample rate for staging
		EnableTracing().
		EnableMetrics().
		Enable()
}

// Testing configures for testing environment
func (ob *ObservabilityBuilder) Testing() *ObservabilityBuilder {
	return ob.
		Environment("testing").
		SampleRate(1.0).
		EnableTracing().
		EnableMetrics().
		Enable()
}

// Minimal configures minimal observability (metrics only)
func (ob *ObservabilityBuilder) Minimal() *ObservabilityBuilder {
	return ob.
		DisableTracing().
		EnableMetrics().
		SampleRate(0.0).
		Enable()
}

// Full configures full observability
func (ob *ObservabilityBuilder) Full() *ObservabilityBuilder {
	return ob.
		EnableTracing().
		EnableMetrics().
		SampleRate(1.0).
		Enable()
}

// ================================
// Cloud Provider Presets
// ================================

// Jaeger configures for Jaeger
func (ob *ObservabilityBuilder) Jaeger(endpoint string) *ObservabilityBuilder {
	if endpoint == "" {
		endpoint = "http://localhost:14268/api/traces"
	}
	return ob.
		OTLPEndpoint(endpoint).
		EnableTracing().
		EnableMetrics()
}

// Honeycomb configures for Honeycomb
func (ob *ObservabilityBuilder) Honeycomb(apiKey string) *ObservabilityBuilder {
	return ob.
		OTLPEndpoint("https://api.honeycomb.io").
		OTLPHeader("x-honeycomb-team", apiKey).
		EnableTracing().
		EnableMetrics()
}

// DataDog configures for DataDog
func (ob *ObservabilityBuilder) DataDog(apiKey string) *ObservabilityBuilder {
	return ob.
		OTLPEndpoint("https://api.datadoghq.com").
		OTLPHeader("DD-API-KEY", apiKey).
		EnableTracing().
		EnableMetrics()
}

// NewRelic configures for New Relic
func (ob *ObservabilityBuilder) NewRelic(licenseKey string) *ObservabilityBuilder {
	return ob.
		OTLPEndpoint("https://otlp.nr-data.net:4317").
		OTLPHeader("api-key", licenseKey).
		EnableTracing().
		EnableMetrics()
}

// ================================
// Conditional Configuration
// ================================

// If applies configuration conditionally
func (ob *ObservabilityBuilder) If(condition bool, fn func(*ObservabilityBuilder) *ObservabilityBuilder) *ObservabilityBuilder {
	if condition {
		return fn(ob)
	}
	return ob
}

// Unless applies configuration conditionally (opposite of If)
func (ob *ObservabilityBuilder) Unless(condition bool, fn func(*ObservabilityBuilder) *ObservabilityBuilder) *ObservabilityBuilder {
	if !condition {
		return fn(ob)
	}
	return ob
}

// ================================
// Build and Validation
// ================================

// Build creates the final telemetry configuration
func (ob *ObservabilityBuilder) Build() (*TelemetryConfig, error) {
	if len(ob.errors) > 0 {
		return nil, fmt.Errorf("observability configuration errors: %v", ob.errors)
	}

	// Additional validation
	if ob.config.Enabled {
		if ob.config.TracingEnabled && ob.config.OTLPEndpoint == "" {
			return nil, fmt.Errorf("OTLP endpoint is required when tracing is enabled")
		}
	}

	return ob.config, nil
}

// MustBuild creates the configuration and panics on error
func (ob *ObservabilityBuilder) MustBuild() *TelemetryConfig {
	config, err := ob.Build()
	if err != nil {
		panic(err)
	}
	return config
}

// Validate checks the configuration without building
func (ob *ObservabilityBuilder) Validate() error {
	_, err := ob.Build()
	return err
}

// HasErrors returns true if there are configuration errors
func (ob *ObservabilityBuilder) HasErrors() bool {
	return len(ob.errors) > 0
}

// Errors returns all configuration errors
func (ob *ObservabilityBuilder) Errors() []error {
	return ob.errors
}

// ================================
// Integration with Config Options
// ================================

// WithObservability creates a configuration option from the builder
func WithObservability() *ObservabilityBuilder {
	return NewObservabilityBuilder()
}

// AsOption converts the builder to a configuration option
func (ob *ObservabilityBuilder) AsOption() Option {
	return optionFunc(func(c *Config) {
		config, err := ob.Build()
		if err == nil {
			c.telemetry = config
		}
	})
}

// ================================
// Quick Configuration Functions
// ================================

// QuickObservability creates a quick observability configuration
func QuickObservability(serviceName, environment string) Option {
	return WithObservability().
		ServiceName(serviceName).
		Environment(environment).
		FromEnvironment().
		AsOption()
}

// DevObservability creates development observability configuration
func DevObservability(serviceName string) Option {
	return WithObservability().
		ServiceName(serviceName).
		Development().
		AsOption()
}

// ProdObservability creates production observability configuration
func ProdObservability(serviceName, endpoint string) Option {
	return WithObservability().
		ServiceName(serviceName).
		OTLPEndpoint(endpoint).
		Production().
		AsOption()
}
