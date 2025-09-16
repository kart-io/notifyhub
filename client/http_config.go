package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/config"
)

// ================================
// HTTP Service Configuration Integration
// ================================

// HTTPServiceConfig holds complete HTTP service configuration
type HTTPServiceConfig struct {
	// Server configuration
	Server HTTPServerOptions

	// NotifyHub configuration
	NotifyHub *config.Config

	// Environment settings
	Environment string
	Profile     string

	// Framework integration
	Framework   string // gin, echo, chi, mux
	Integration *FrameworkIntegration

	// Configuration sources and validation
	Sources    []ConfigSource
	Validation *ConfigValidation
}

// FrameworkIntegration provides framework-specific integration options
type FrameworkIntegration struct {
	// Framework type detection
	AutoDetect      bool
	ManualFramework string

	// Middleware handling
	UseFrameworkMiddleware bool
	MiddlewareOrder        []string

	// Route integration
	RoutePrefixHandling string // "replace", "append", "auto"
	RouteRegistration   string // "manual", "auto"

	// Context propagation
	ContextPropagation bool
}

// ConfigSource represents a configuration source
type ConfigSource struct {
	Type     string // "env", "file", "api", "default"
	Priority int    // Higher priority overrides lower
	Location string // File path, API endpoint, etc.
	Format   string // "yaml", "json", "toml"
	Optional bool   // Whether this source is optional
}

// ConfigValidation holds validation settings and results
type ConfigValidation struct {
	Enabled         bool
	StrictMode      bool
	RequiredFields  []string
	ValidationRules map[string]ValidationRule
	Results         *ValidationResults
}

// ValidationRule defines validation criteria for a configuration field
type ValidationRule struct {
	Type        string      // "required", "range", "regex", "custom"
	Value       interface{} // Rule-specific value
	Message     string      // Custom error message
	Suggestions []string    // Suggested fixes
}

// ValidationResults holds validation results
type ValidationResults struct {
	Valid       bool
	Errors      []ConfigError
	Warnings    []ConfigWarning
	Suggestions []ConfigSuggestion
	Score       int // 0-100 confidence score
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field       string
	Message     string
	Value       interface{}
	Suggestions []string
}

// ConfigWarning represents a configuration warning
type ConfigWarning struct {
	Field   string
	Message string
	Value   interface{}
}

// ConfigSuggestion represents a configuration improvement suggestion
type ConfigSuggestion struct {
	Field     string
	Current   interface{}
	Suggested interface{}
	Reason    string
	Impact    string // "performance", "security", "reliability"
}

// ================================
// HTTP Service Configuration Builder
// ================================

// HTTPServiceConfigBuilder builds complete HTTP service configurations
type HTTPServiceConfigBuilder struct {
	config   *HTTPServiceConfig
	errors   []error
	warnings []string
	debug    bool
}

// NewHTTPServiceConfig creates a new HTTP service configuration builder
func NewHTTPServiceConfig() *HTTPServiceConfigBuilder {
	return &HTTPServiceConfigBuilder{
		config: &HTTPServiceConfig{
			Server: HTTPServerOptions{
				Addr:             ":8080",
				BasePath:         "/notify",
				ReadTimeout:      30 * time.Second,
				WriteTimeout:     30 * time.Second,
				IdleTimeout:      120 * time.Second,
				MaxHeaderBytes:   1 << 20,
				EnableKeepAlives: true,
			},
			Environment: "development",
			Profile:     "default",
			Framework:   "auto",
			Integration: &FrameworkIntegration{
				AutoDetect:             true,
				UseFrameworkMiddleware: true,
				RoutePrefixHandling:    "auto",
				RouteRegistration:      "auto",
				ContextPropagation:     true,
			},
			Sources: []ConfigSource{
				{Type: "env", Priority: 100, Optional: false},
				{Type: "default", Priority: 1, Optional: false},
			},
			Validation: &ConfigValidation{
				Enabled:         true,
				StrictMode:      false,
				RequiredFields:  []string{"server.addr"},
				ValidationRules: make(map[string]ValidationRule),
			},
		},
	}
}

// ================================
// Configuration Source Methods
// ================================

// FromEnvironment configures from environment variables
func (b *HTTPServiceConfigBuilder) FromEnvironment() *HTTPServiceConfigBuilder {
	b.config.Sources = append(b.config.Sources, ConfigSource{
		Type:     "env",
		Priority: 100,
		Optional: false,
	})

	// Load environment-based configuration
	b.loadFromEnvironment()
	return b
}

// FromFile configures from a configuration file
func (b *HTTPServiceConfigBuilder) FromFile(path string, format string) *HTTPServiceConfigBuilder {
	b.config.Sources = append(b.config.Sources, ConfigSource{
		Type:     "file",
		Priority: 80,
		Location: path,
		Format:   format,
		Optional: true,
	})

	// Load file-based configuration would be implemented here
	b.warnings = append(b.warnings, fmt.Sprintf("File configuration loading not yet implemented: %s", path))
	return b
}

// FromAPI configures from a remote API
func (b *HTTPServiceConfigBuilder) FromAPI(endpoint string) *HTTPServiceConfigBuilder {
	b.config.Sources = append(b.config.Sources, ConfigSource{
		Type:     "api",
		Priority: 60,
		Location: endpoint,
		Optional: true,
	})

	b.warnings = append(b.warnings, fmt.Sprintf("API configuration loading not yet implemented: %s", endpoint))
	return b
}

// loadFromEnvironment loads configuration from environment variables
func (b *HTTPServiceConfigBuilder) loadFromEnvironment() {
	// HTTP Server configuration
	if addr := os.Getenv("HTTP_SERVER_ADDR"); addr != "" {
		b.config.Server.Addr = addr
	}
	if basePath := os.Getenv("HTTP_SERVER_BASE_PATH"); basePath != "" {
		b.config.Server.BasePath = basePath
	}
	if readTimeout := os.Getenv("HTTP_SERVER_READ_TIMEOUT"); readTimeout != "" {
		if d, err := time.ParseDuration(readTimeout); err == nil {
			b.config.Server.ReadTimeout = d
		}
	}
	if writeTimeout := os.Getenv("HTTP_SERVER_WRITE_TIMEOUT"); writeTimeout != "" {
		if d, err := time.ParseDuration(writeTimeout); err == nil {
			b.config.Server.WriteTimeout = d
		}
	}

	// Environment detection
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		b.config.Environment = env
	}
	if profile := os.Getenv("PROFILE"); profile != "" {
		b.config.Profile = profile
	}

	// Framework detection
	if framework := os.Getenv("HTTP_FRAMEWORK"); framework != "" {
		b.config.Framework = framework
		b.config.Integration.AutoDetect = false
		b.config.Integration.ManualFramework = framework
	}

	// CORS configuration
	if cors := os.Getenv("HTTP_ENABLE_CORS"); cors != "" {
		b.config.Server.EnableCORS = strings.ToLower(cors) == "true"
	}

	// Middleware configuration
	if useFrameworkMiddleware := os.Getenv("HTTP_USE_FRAMEWORK_MIDDLEWARE"); useFrameworkMiddleware != "" {
		b.config.Integration.UseFrameworkMiddleware = strings.ToLower(useFrameworkMiddleware) == "true"
	}
}

// ================================
// Framework Integration Methods
// ================================

// ForFramework explicitly sets the target framework
func (b *HTTPServiceConfigBuilder) ForFramework(framework string) *HTTPServiceConfigBuilder {
	b.config.Framework = framework
	b.config.Integration.AutoDetect = false
	b.config.Integration.ManualFramework = framework

	// Apply framework-specific defaults
	switch strings.ToLower(framework) {
	case "gin":
		b.applyGinDefaults()
	case "echo":
		b.applyEchoDefaults()
	case "chi":
		b.applyChiDefaults()
	case "mux", "gorilla":
		b.applyGorillaDefaults()
	case "net/http", "http":
		b.applyNetHTTPDefaults()
	default:
		b.warnings = append(b.warnings, fmt.Sprintf("Unknown framework '%s', using generic defaults", framework))
	}

	return b
}

// WithAutoFrameworkDetection enables automatic framework detection
func (b *HTTPServiceConfigBuilder) WithAutoFrameworkDetection() *HTTPServiceConfigBuilder {
	b.config.Integration.AutoDetect = true
	b.detectFramework()
	return b
}

// detectFramework attempts to detect the framework in use
func (b *HTTPServiceConfigBuilder) detectFramework() {
	// Simple framework detection based on imports (would need actual implementation)
	detectedFramework := "net/http" // Default fallback

	b.config.Framework = detectedFramework
	b.config.Integration.ManualFramework = detectedFramework
}

// applyGinDefaults applies Gin-specific configuration defaults
func (b *HTTPServiceConfigBuilder) applyGinDefaults() {
	b.config.Integration.UseFrameworkMiddleware = true
	b.config.Integration.RoutePrefixHandling = "append"
	b.config.Integration.RouteRegistration = "auto"
	b.config.Integration.MiddlewareOrder = []string{"gin.Logger", "gin.Recovery", "notifyhub"}
}

// applyEchoDefaults applies Echo-specific configuration defaults
func (b *HTTPServiceConfigBuilder) applyEchoDefaults() {
	b.config.Integration.UseFrameworkMiddleware = true
	b.config.Integration.RoutePrefixHandling = "replace"
	b.config.Integration.RouteRegistration = "auto"
	b.config.Integration.MiddlewareOrder = []string{"echo.Logger", "echo.Recover", "notifyhub"}
}

// applyChiDefaults applies Chi-specific configuration defaults
func (b *HTTPServiceConfigBuilder) applyChiDefaults() {
	b.config.Integration.UseFrameworkMiddleware = true
	b.config.Integration.RoutePrefixHandling = "append"
	b.config.Integration.RouteRegistration = "manual"
	b.config.Integration.MiddlewareOrder = []string{"chi.Logger", "chi.Recoverer", "notifyhub"}
}

// applyGorillaDefaults applies Gorilla mux-specific configuration defaults
func (b *HTTPServiceConfigBuilder) applyGorillaDefaults() {
	b.config.Integration.UseFrameworkMiddleware = false
	b.config.Integration.RoutePrefixHandling = "manual"
	b.config.Integration.RouteRegistration = "manual"
	b.config.Integration.MiddlewareOrder = []string{"notifyhub"}
}

// applyNetHTTPDefaults applies net/http-specific configuration defaults
func (b *HTTPServiceConfigBuilder) applyNetHTTPDefaults() {
	b.config.Integration.UseFrameworkMiddleware = false
	b.config.Integration.RoutePrefixHandling = "manual"
	b.config.Integration.RouteRegistration = "manual"
	b.config.Integration.MiddlewareOrder = []string{"notifyhub"}
}

// ================================
// Environment Profile Methods
// ================================

// ForEnvironment sets the target environment
func (b *HTTPServiceConfigBuilder) ForEnvironment(env string) *HTTPServiceConfigBuilder {
	b.config.Environment = env

	// Apply environment-specific defaults
	switch strings.ToLower(env) {
	case "production", "prod":
		b.applyProductionDefaults()
	case "staging", "stage":
		b.applyStagingDefaults()
	case "development", "dev":
		b.applyDevelopmentDefaults()
	case "testing", "test":
		b.applyTestingDefaults()
	default:
		b.warnings = append(b.warnings, fmt.Sprintf("Unknown environment '%s', using development defaults", env))
		b.applyDevelopmentDefaults()
	}

	return b
}

// WithProfile sets the configuration profile
func (b *HTTPServiceConfigBuilder) WithProfile(profile string) *HTTPServiceConfigBuilder {
	b.config.Profile = profile

	switch strings.ToLower(profile) {
	case "minimal":
		b.applyMinimalProfile()
	case "standard":
		b.applyStandardProfile()
	case "comprehensive":
		b.applyComprehensiveProfile()
	case "performance":
		b.applyPerformanceProfile()
	case "security":
		b.applySecurityProfile()
	}

	return b
}

// applyProductionDefaults applies production environment settings
func (b *HTTPServiceConfigBuilder) applyProductionDefaults() {
	// Enable all production middleware
	b.config.Server.GlobalMiddleware = []func(http.Handler) http.Handler{
		RecoveryMiddleware(),
		LoggingMiddleware(),
		MetricsMiddleware(),
		CompressionMiddleware(),
	}

	// Strict validation
	b.config.Validation.StrictMode = true

	// Security headers
	b.config.Server.EnableCORS = false // Disabled by default in production

	// Performance settings
	b.config.Server.ReadTimeout = 10 * time.Second
	b.config.Server.WriteTimeout = 10 * time.Second
	b.config.Server.IdleTimeout = 60 * time.Second
}

// applyStagingDefaults applies staging environment settings
func (b *HTTPServiceConfigBuilder) applyStagingDefaults() {
	// Production-like but with more logging
	b.config.Server.GlobalMiddleware = []func(http.Handler) http.Handler{
		RecoveryMiddleware(),
		LoggingMiddleware(),
		MetricsMiddleware(),
	}

	b.config.Validation.StrictMode = true
	b.config.Server.EnableCORS = true
}

// applyDevelopmentDefaults applies development environment settings
func (b *HTTPServiceConfigBuilder) applyDevelopmentDefaults() {
	// Development-friendly middleware
	b.config.Server.GlobalMiddleware = []func(http.Handler) http.Handler{
		RecoveryMiddleware(),
		LoggingMiddleware(),
	}

	b.config.Validation.StrictMode = false
	b.config.Server.EnableCORS = true

	// Relaxed timeouts for debugging
	b.config.Server.ReadTimeout = 60 * time.Second
	b.config.Server.WriteTimeout = 60 * time.Second
}

// applyTestingDefaults applies testing environment settings
func (b *HTTPServiceConfigBuilder) applyTestingDefaults() {
	// Minimal middleware for testing
	b.config.Server.GlobalMiddleware = []func(http.Handler) http.Handler{
		RecoveryMiddleware(),
	}

	b.config.Validation.StrictMode = false
	b.config.Server.EnableCORS = true

	// Fast timeouts for tests
	b.config.Server.ReadTimeout = 5 * time.Second
	b.config.Server.WriteTimeout = 5 * time.Second
	b.config.Server.IdleTimeout = 5 * time.Second
}

// Profile application methods
func (b *HTTPServiceConfigBuilder) applyMinimalProfile() {
	b.config.Server.GlobalMiddleware = []func(http.Handler) http.Handler{
		RecoveryMiddleware(),
	}
}

func (b *HTTPServiceConfigBuilder) applyStandardProfile() {
	b.config.Server.GlobalMiddleware = []func(http.Handler) http.Handler{
		RecoveryMiddleware(),
		LoggingMiddleware(),
	}
}

func (b *HTTPServiceConfigBuilder) applyComprehensiveProfile() {
	b.config.Server.GlobalMiddleware = []func(http.Handler) http.Handler{
		RecoveryMiddleware(),
		LoggingMiddleware(),
		MetricsMiddleware(),
		CompressionMiddleware(),
	}
}

func (b *HTTPServiceConfigBuilder) applyPerformanceProfile() {
	b.config.Server.GlobalMiddleware = []func(http.Handler) http.Handler{
		RecoveryMiddleware(),
		CompressionMiddleware(),
	}

	// Performance-optimized timeouts
	b.config.Server.ReadTimeout = 5 * time.Second
	b.config.Server.WriteTimeout = 5 * time.Second
	b.config.Server.IdleTimeout = 30 * time.Second
}

func (b *HTTPServiceConfigBuilder) applySecurityProfile() {
	b.config.Server.GlobalMiddleware = []func(http.Handler) http.Handler{
		RecoveryMiddleware(),
		LoggingMiddleware(),
		// Would add security middleware like rate limiting, auth, etc.
	}

	// Security-focused settings
	b.config.Server.EnableCORS = false
	b.config.Validation.StrictMode = true
}

// ================================
// Configuration Validation Methods
// ================================

// WithValidation enables configuration validation
func (b *HTTPServiceConfigBuilder) WithValidation() *HTTPServiceConfigBuilder {
	b.config.Validation.Enabled = true
	return b
}

// WithStrictValidation enables strict validation mode
func (b *HTTPServiceConfigBuilder) WithStrictValidation() *HTTPServiceConfigBuilder {
	b.config.Validation.Enabled = true
	b.config.Validation.StrictMode = true
	return b
}

// AddValidationRule adds a custom validation rule
func (b *HTTPServiceConfigBuilder) AddValidationRule(field string, rule ValidationRule) *HTTPServiceConfigBuilder {
	if b.config.Validation.ValidationRules == nil {
		b.config.Validation.ValidationRules = make(map[string]ValidationRule)
	}
	b.config.Validation.ValidationRules[field] = rule
	return b
}

// validate performs configuration validation
func (b *HTTPServiceConfigBuilder) validate() *ValidationResults {
	results := &ValidationResults{
		Valid:       true,
		Errors:      []ConfigError{},
		Warnings:    []ConfigWarning{},
		Suggestions: []ConfigSuggestion{},
		Score:       100,
	}

	// Validate required fields
	for _, field := range b.config.Validation.RequiredFields {
		if !b.hasField(field) {
			results.Valid = false
			results.Errors = append(results.Errors, ConfigError{
				Field:   field,
				Message: fmt.Sprintf("Required field '%s' is missing", field),
				Suggestions: []string{
					fmt.Sprintf("Set %s environment variable", strings.ToUpper(strings.ReplaceAll(field, ".", "_"))),
					fmt.Sprintf("Use builder method to set %s", field),
				},
			})
			results.Score -= 20
		}
	}

	// Validate server address format
	if b.config.Server.Addr != "" {
		if !strings.Contains(b.config.Server.Addr, ":") {
			results.Valid = false
			results.Errors = append(results.Errors, ConfigError{
				Field:   "server.addr",
				Message: "Server address must include port (e.g., ':8080' or 'localhost:8080')",
				Value:   b.config.Server.Addr,
				Suggestions: []string{
					fmt.Sprintf(":%s", b.config.Server.Addr),
					fmt.Sprintf("localhost:%s", b.config.Server.Addr),
				},
			})
			results.Score -= 15
		}
	}

	// Validate timeout values
	if b.config.Server.ReadTimeout <= 0 {
		results.Warnings = append(results.Warnings, ConfigWarning{
			Field:   "server.read_timeout",
			Message: "Read timeout should be positive",
			Value:   b.config.Server.ReadTimeout,
		})
		results.Score -= 5
	}

	// Performance suggestions
	if b.config.Server.ReadTimeout > 60*time.Second {
		results.Suggestions = append(results.Suggestions, ConfigSuggestion{
			Field:     "server.read_timeout",
			Current:   b.config.Server.ReadTimeout,
			Suggested: 30 * time.Second,
			Reason:    "Shorter read timeout improves server responsiveness",
			Impact:    "performance",
		})
	}

	// Security suggestions
	if b.config.Environment == "production" && b.config.Server.EnableCORS {
		results.Suggestions = append(results.Suggestions, ConfigSuggestion{
			Field:     "server.enable_cors",
			Current:   true,
			Suggested: false,
			Reason:    "CORS should be explicitly configured in production",
			Impact:    "security",
		})
	}

	return results
}

// hasField checks if a configuration field is set
func (b *HTTPServiceConfigBuilder) hasField(field string) bool {
	parts := strings.Split(field, ".")
	switch parts[0] {
	case "server":
		if len(parts) > 1 {
			switch parts[1] {
			case "addr":
				return b.config.Server.Addr != ""
			case "base_path":
				return b.config.Server.BasePath != ""
			}
		}
	}
	return false
}

// ================================
// Server Configuration Methods
// ================================

// WithServer configures server options directly
func (b *HTTPServiceConfigBuilder) WithServer(options ...HTTPServerOption) *HTTPServiceConfigBuilder {
	for _, opt := range options {
		opt(&b.config.Server)
	}
	return b
}

// WithNotifyHub configures NotifyHub options
func (b *HTTPServiceConfigBuilder) WithNotifyHub(options ...config.Option) *HTTPServiceConfigBuilder {
	b.config.NotifyHub = config.New(options...)
	return b
}

// WithAddress sets the server address
func (b *HTTPServiceConfigBuilder) WithAddress(addr string) *HTTPServiceConfigBuilder {
	b.config.Server.Addr = addr
	return b
}

// WithBasePath sets the base path for NotifyHub routes
func (b *HTTPServiceConfigBuilder) WithBasePath(basePath string) *HTTPServiceConfigBuilder {
	b.config.Server.BasePath = basePath
	return b
}

// WithTimeouts sets server timeouts
func (b *HTTPServiceConfigBuilder) WithTimeouts(read, write, idle time.Duration) *HTTPServiceConfigBuilder {
	if read > 0 {
		b.config.Server.ReadTimeout = read
	}
	if write > 0 {
		b.config.Server.WriteTimeout = write
	}
	if idle > 0 {
		b.config.Server.IdleTimeout = idle
	}
	return b
}

// EnableDebug enables debug mode with additional logging
func (b *HTTPServiceConfigBuilder) EnableDebug() *HTTPServiceConfigBuilder {
	b.debug = true
	return b
}

// ================================
// Build Methods
// ================================

// Build creates the final HTTP service configuration
func (b *HTTPServiceConfigBuilder) Build() (*HTTPServiceConfig, error) {
	// Perform validation if enabled
	if b.config.Validation.Enabled {
		results := b.validate()
		b.config.Validation.Results = results

		if !results.Valid && b.config.Validation.StrictMode {
			return nil, fmt.Errorf("configuration validation failed: %d errors", len(results.Errors))
		}
	}

	// Set NotifyHub configuration if not set
	if b.config.NotifyHub == nil {
		b.config.NotifyHub = config.New(config.WithDefaults())
	}

	// Apply any accumulated errors
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("configuration build failed: %v", b.errors)
	}

	return b.config, nil
}

// MustBuild creates the configuration and panics on error
func (b *HTTPServiceConfigBuilder) MustBuild() *HTTPServiceConfig {
	cfg, err := b.Build()
	if err != nil {
		panic(err)
	}
	return cfg
}

// BuildServer creates a complete HTTP server using the configuration
func (b *HTTPServiceConfigBuilder) BuildServer(ctx context.Context) (*http.Server, *Hub, error) {
	cfg, err := b.Build()
	if err != nil {
		return nil, nil, err
	}

	// Create NotifyHub instance from existing config
	hub, err := NewFromConfig(cfg.NotifyHub)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create NotifyHub: %w", err)
	}
	if err := hub.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start NotifyHub: %v", err)
	}

	// Create HTTP server using the configuration
	serverOptions := []HTTPServerOption{
		WithAddress(cfg.Server.Addr),
		WithBasePath(cfg.Server.BasePath),
		WithTimeouts(cfg.Server.ReadTimeout, cfg.Server.WriteTimeout, cfg.Server.IdleTimeout),
	}

	// Add middleware
	if len(cfg.Server.GlobalMiddleware) > 0 {
		serverOptions = append(serverOptions, WithGlobalMiddleware(cfg.Server.GlobalMiddleware...))
	}

	// Add CORS if enabled
	if cfg.Server.EnableCORS {
		if len(cfg.Server.AllowedOrigins) > 0 {
			serverOptions = append(serverOptions, WithCORS(cfg.Server.AllowedOrigins, cfg.Server.AllowedMethods, cfg.Server.AllowedHeaders))
		} else {
			serverOptions = append(serverOptions, WithDefaultCORS())
		}
	}

	server := QuickHTTPServerWithOptions(hub, serverOptions...)

	return server, hub, nil
}

// ================================
// Convenience Functions
// ================================

// QuickDevelopmentConfig creates a development-ready configuration
func QuickDevelopmentConfig() *HTTPServiceConfigBuilder {
	return NewHTTPServiceConfig().
		ForEnvironment("development").
		WithProfile("standard").
		FromEnvironment().
		WithAutoFrameworkDetection()
}

// QuickProductionConfig creates a production-ready configuration
func QuickProductionConfig() *HTTPServiceConfigBuilder {
	return NewHTTPServiceConfig().
		ForEnvironment("production").
		WithProfile("comprehensive").
		FromEnvironment().
		WithStrictValidation().
		ForFramework("auto")
}

// QuickTestConfig creates a test-friendly configuration
func QuickTestConfig() *HTTPServiceConfigBuilder {
	return NewHTTPServiceConfig().
		ForEnvironment("testing").
		WithProfile("minimal").
		WithNotifyHub(config.WithTestDefaults())
}

// ================================
// Framework Integration Helpers
// ================================

// IntegrateWithGin provides Gin-specific integration helpers
func (cfg *HTTPServiceConfig) IntegrateWithGin() *GinIntegration {
	return &GinIntegration{config: cfg}
}

// IntegrateWithEcho provides Echo-specific integration helpers
func (cfg *HTTPServiceConfig) IntegrateWithEcho() *EchoIntegration {
	return &EchoIntegration{config: cfg}
}

// IntegrateWithChi provides Chi-specific integration helpers
func (cfg *HTTPServiceConfig) IntegrateWithChi() *ChiIntegration {
	return &ChiIntegration{config: cfg}
}

// GinIntegration provides Gin framework integration
type GinIntegration struct {
	config *HTTPServiceConfig
}

// EchoIntegration provides Echo framework integration
type EchoIntegration struct {
	config *HTTPServiceConfig
}

// ChiIntegration provides Chi framework integration
type ChiIntegration struct {
	config *HTTPServiceConfig
}

// Placeholder methods for framework integrations
func (gi *GinIntegration) RegisterRoutes(engine interface{}) error {
	// Implementation would register NotifyHub routes with Gin engine
	return fmt.Errorf("Gin integration not yet implemented")
}

func (ei *EchoIntegration) RegisterRoutes(echo interface{}) error {
	// Implementation would register NotifyHub routes with Echo instance
	return fmt.Errorf("Echo integration not yet implemented")
}

func (ci *ChiIntegration) RegisterRoutes(router interface{}) error {
	// Implementation would register NotifyHub routes with Chi router
	return fmt.Errorf("Chi integration not yet implemented")
}

// ================================
// Configuration Export/Import
// ================================

// ExportToYAML exports configuration to YAML format
func (cfg *HTTPServiceConfig) ExportToYAML() (string, error) {
	// Implementation would export to YAML
	return "", fmt.Errorf("YAML export not yet implemented")
}

// ExportToJSON exports configuration to JSON format
func (cfg *HTTPServiceConfig) ExportToJSON() (string, error) {
	// Implementation would export to JSON
	return "", fmt.Errorf("JSON export not yet implemented")
}

// ImportFromYAML imports configuration from YAML
func ImportHTTPServiceConfigFromYAML(yamlData string) (*HTTPServiceConfig, error) {
	// Implementation would import from YAML
	return nil, fmt.Errorf("YAML import not yet implemented")
}

// ImportFromJSON imports configuration from JSON
func ImportHTTPServiceConfigFromJSON(jsonData string) (*HTTPServiceConfig, error) {
	// Implementation would import from JSON
	return nil, fmt.Errorf("JSON import not yet implemented")
}
