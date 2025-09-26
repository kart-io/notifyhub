// Package template provides factory functions for template engines and managers
package template

import (
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// EngineFactory creates template engines
type EngineFactory struct {
	logger logger.Logger
}

// NewEngineFactory creates a new engine factory
func NewEngineFactory(logger logger.Logger) *EngineFactory {
	return &EngineFactory{
		logger: logger,
	}
}

// CreateEngine creates a template engine of the specified type
func (f *EngineFactory) CreateEngine(engineType EngineType) (Engine, error) {
	switch engineType {
	case EngineGo:
		return NewGoEngine(f.logger), nil
	case EngineMustache:
		return NewMustacheEngine(f.logger), nil
	case EngineHandlebars:
		return NewHandlebarsEngine(f.logger), nil
	default:
		return nil, fmt.Errorf("unsupported template engine: %s", engineType)
	}
}

// GetSupportedEngines returns all supported engine types
func (f *EngineFactory) GetSupportedEngines() []EngineType {
	return []EngineType{
		EngineGo,
		EngineMustache,
		EngineHandlebars,
	}
}

// GetEngineCapabilities returns capabilities for all supported engines
func (f *EngineFactory) GetEngineCapabilities() map[EngineType]EngineCapabilities {
	capabilities := make(map[EngineType]EngineCapabilities)

	for _, engineType := range f.GetSupportedEngines() {
		engine, err := f.CreateEngine(engineType)
		if err == nil {
			capabilities[engineType] = engine.GetCapabilities()
		}
	}

	return capabilities
}

// RecommendEngine recommends the best engine for given requirements
func (f *EngineFactory) RecommendEngine(requirements EngineRequirements) EngineType {
	// Simple recommendation logic based on requirements

	if requirements.NeedsFunctions && requirements.PerformanceLevel == "high" {
		return EngineGo // Go templates have best function support and performance
	}

	if requirements.NeedsFunctions && !requirements.NeedsLogicLess {
		return EngineHandlebars // Handlebars has good function support
	}

	if requirements.NeedsLogicLess {
		return EngineMustache // Mustache is logic-less
	}

	if requirements.PerformanceLevel == "high" {
		return EngineGo // Go templates are fastest
	}

	// Default recommendation
	return EngineGo
}

// EngineRequirements represents requirements for engine selection
type EngineRequirements struct {
	NeedsFunctions   bool   // Requires custom functions/helpers
	NeedsLogicLess   bool   // Prefers logic-less templates
	NeedsPartials    bool   // Requires partial template support
	PerformanceLevel string // "high", "medium", "low"
	MaxTemplateSize  int    // Maximum template size
	SupportsCompile  bool   // Needs compilation support
}

// TemplateManagerBuilder helps build template managers with fluent interface
type TemplateManagerBuilder struct {
	config ManagerConfig
	logger logger.Logger
}

// NewTemplateManagerBuilder creates a new template manager builder
func NewTemplateManagerBuilder(logger logger.Logger) *TemplateManagerBuilder {
	return &TemplateManagerBuilder{
		config: DefaultManagerConfig(),
		logger: logger,
	}
}

// WithEngine sets the default template engine
func (b *TemplateManagerBuilder) WithEngine(engine EngineType) *TemplateManagerBuilder {
	b.config.DefaultEngine = engine
	return b
}

// WithCaching enables template caching
func (b *TemplateManagerBuilder) WithCaching(cacheType CacheType, ttl time.Duration) *TemplateManagerBuilder {
	b.config.EnableCache = true
	b.config.CacheConfig.Type = cacheType
	b.config.CacheConfig.TTL = ttl
	return b
}

// WithHotReloading enables hot reloading
func (b *TemplateManagerBuilder) WithHotReloading(paths ...string) *TemplateManagerBuilder {
	b.config.HotReload = true
	b.config.WatchPaths = paths
	return b
}

// WithValidation sets validation mode
func (b *TemplateManagerBuilder) WithValidation(mode ValidationMode) *TemplateManagerBuilder {
	b.config.ValidationMode = mode
	return b
}

// WithTimeout sets render timeout
func (b *TemplateManagerBuilder) WithTimeout(timeout time.Duration) *TemplateManagerBuilder {
	b.config.RenderTimeout = timeout
	return b
}

// ForTesting configures the manager for testing
func (b *TemplateManagerBuilder) ForTesting() *TemplateManagerBuilder {
	b.config = TestConfig()
	return b
}

// ForProduction configures the manager for production use
func (b *TemplateManagerBuilder) ForProduction() *TemplateManagerBuilder {
	b.config = PerformanceConfig()
	return b
}

// ForDevelopment configures the manager for development
func (b *TemplateManagerBuilder) ForDevelopment(watchPaths ...string) *TemplateManagerBuilder {
	b.config = DevelopmentConfig(watchPaths...)
	return b
}

// Build creates the template manager
func (b *TemplateManagerBuilder) Build() (Manager, error) {
	return NewManager(b.config, b.logger)
}

// Quick factory functions for common scenarios

// NewTestManager creates a template manager configured for testing
func NewTestManager(logger logger.Logger) (Manager, error) {
	return NewManagerWithOptions(logger,
		WithDefaultEngine(EngineGo),
		WithNoValidation(), // No validation overhead in tests
		WithMaxTemplateSize(64*1024),
		WithRenderTimeout(5*time.Second))
}

// NewProductionManager creates a template manager configured for production
func NewProductionManager(logger logger.Logger) (Manager, error) {
	return NewManagerWithOptions(logger,
		WithDefaultEngine(EngineGo),
		WithMultiLayerCache(10*time.Minute, time.Hour, 6*time.Hour), // 3-layer caching
		WithWriteThroughCache(),
		WithReadThroughCache(),
		WithNoValidation(), // No validation overhead in production
		WithMaxTemplateSize(2*1024*1024),
		WithRenderTimeout(10*time.Second))
}

// NewDevelopmentManager creates a template manager configured for development
func NewDevelopmentManager(logger logger.Logger, watchPaths ...string) (Manager, error) {
	return NewManagerWithOptions(logger,
		WithDefaultEngine(EngineGo),
		WithHotReload(watchPaths...),
		WithValidationMode(ValidationWarn),
		WithMaxTemplateSize(1024*1024),
		WithRenderTimeout(30*time.Second))
}
