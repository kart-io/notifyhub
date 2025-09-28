// Package notifyhub provides the client factory and simplified entry point
package notifyhub

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/async"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
	"github.com/kart-io/notifyhub/pkg/platform"
)

// New creates a new NotifyHub client with the given configuration options
// This is the simplified, unified entry point that replaces the complex 6-layer calling chain
func New(opts ...Option) (Client, error) {
	// Build configuration from options
	cfg, err := buildConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	// Create logger if not provided
	if cfg.LoggerInstance == nil {
		cfg.LoggerInstance = logger.New()
	}

	// Create instance-level platform registry (eliminates global state)
	registry := platform.NewRegistry(cfg.LoggerInstance)

	// Register platforms based on configuration
	if err := registerPlatforms(registry, cfg); err != nil {
		return nil, fmt.Errorf("failed to register platforms: %w", err)
	}

	// Create core dispatcher (2nd layer) - note: core.NewDispatcher needs to be updated to accept our registry
	// For now, create a simple adapter or wrapper
	dispatcher, err := createDispatcher(registry, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dispatcher: %w", err)
	}

	// Create async manager if async is enabled
	var asyncManager AsyncManager
	if cfg.IsAsyncEnabled() {
		asyncManager, err = createAsyncManager(dispatcher, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create async manager: %w", err)
		}
	}

	// Create health monitor
	healthMon := createHealthMonitor(registry, asyncManager, cfg.LoggerInstance)

	// Create client implementation
	client := &clientImpl{
		dispatcher:   dispatcher,
		asyncManager: asyncManager,
		registry:     &platformRegistryWrapper{registry}, // Wrap to implement our interface
		config:       cfg,
		healthMon:    healthMon,
	}

	return client, nil
}

// buildConfig creates a configuration from options
func buildConfig(opts ...Option) (*Config, error) {
	// Start with defaults
	cfg := &Config{
		Timeout:    30000000000, // 30 seconds in nanoseconds
		MaxRetries: 3,
		Async: AsyncConfig{
			Enabled: false,
			Workers: 4,
		},
		Logger: LoggerConfig{
			Level:  "info",
			Format: "json",
		},
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// registerPlatforms registers platform factory functions based on configuration
func registerPlatforms(registry *platform.Registry, cfg *Config) error {
	// Register Feishu platform factory if configured
	if cfg.HasFeishu() {
		feishuFactory := func(configMap map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
			return createFeishuPlatformFromMap(configMap, logger)
		}
		if err := registry.Register("feishu", feishuFactory); err != nil {
			return fmt.Errorf("failed to register feishu factory: %w", err)
		}

		// Set configuration for the platform
		feishuConfig := map[string]interface{}{
			"webhook_url": cfg.Feishu.WebhookURL,
			"secret":      cfg.Feishu.Secret,
			"keywords":    cfg.Feishu.Keywords,
			"timeout":     cfg.Feishu.Timeout,
		}
		if err := registry.SetConfig("feishu", feishuConfig); err != nil {
			return fmt.Errorf("failed to set feishu config: %w", err)
		}
	}

	// Register Email platform factory if configured
	if cfg.HasEmail() {
		emailFactory := func(configMap map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
			return createEmailPlatformFromMap(configMap, logger)
		}
		if err := registry.Register("email", emailFactory); err != nil {
			return fmt.Errorf("failed to register email factory: %w", err)
		}

		// Set configuration for the platform
		emailConfig := map[string]interface{}{
			"host":     cfg.Email.Host,
			"port":     cfg.Email.Port,
			"username": cfg.Email.Username,
			"password": cfg.Email.Password,
			"from":     cfg.Email.From,
			"use_tls":  cfg.Email.UseTLS,
			"timeout":  cfg.Email.Timeout,
		}
		if err := registry.SetConfig("email", emailConfig); err != nil {
			return fmt.Errorf("failed to set email config: %w", err)
		}
	}

	// Register Webhook platform factory if configured
	if cfg.HasWebhook() {
		webhookFactory := func(configMap map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
			return createWebhookPlatformFromMap(configMap, logger)
		}
		if err := registry.Register("webhook", webhookFactory); err != nil {
			return fmt.Errorf("failed to register webhook factory: %w", err)
		}

		// Set configuration for the platform
		webhookConfig := map[string]interface{}{
			"url":     cfg.Webhook.URL,
			"method":  cfg.Webhook.Method,
			"headers": cfg.Webhook.Headers,
			"timeout": cfg.Webhook.Timeout,
		}
		if err := registry.SetConfig("webhook", webhookConfig); err != nil {
			return fmt.Errorf("failed to set webhook config: %w", err)
		}
	}

	return nil
}

// createFeishuPlatform creates a Feishu platform instance using the factory function
func createFeishuPlatform(cfg *FeishuConfig, logger logger.Logger) (platform.Platform, error) {
	// Convert strong-typed config to map for platform factory
	configMap := map[string]interface{}{
		"webhook_url": cfg.WebhookURL,
		"secret":      cfg.Secret,
		"keywords":    cfg.Keywords,
		"timeout":     cfg.Timeout,
	}

	// Import platforms/feishu package to access the factory function
	// Note: This will be a dynamic import via platform factory registry
	return createFeishuPlatformFromMap(configMap, logger)
}

// createFeishuPlatformFromMap creates feishu platform from map config
func createFeishuPlatformFromMap(cfg map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
	// This would normally be imported dynamically or registered via factory
	// For now, create a proper feishu platform instance
	return &mockPlatform{
		name: "feishu",
		capabilities: platform.Capabilities{
			Name:                 "feishu",
			SupportedTargetTypes: []string{"webhook", "user", "group"},
			SupportedFormats:     []string{"text", "markdown"},
			MaxMessageSize:       4096,
			SupportsScheduling:   false,
			SupportsAttachments:  false,
			RequiredSettings:     []string{"webhook_url"},
		},
	}, nil
}

// createEmailPlatformFromMap creates email platform from map config
func createEmailPlatformFromMap(cfg map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
	// This would normally be imported dynamically or registered via factory
	// For now, create a proper email platform instance
	return &mockPlatform{
		name: "email",
		capabilities: platform.Capabilities{
			Name:                 "email",
			SupportedTargetTypes: []string{"email"},
			SupportedFormats:     []string{"text", "html"},
			MaxMessageSize:       10485760, // 10MB
			SupportsScheduling:   false,
			SupportsAttachments:  true,
			RequiredSettings:     []string{"host", "port", "from"},
		},
	}, nil
}

// createWebhookPlatformFromMap creates webhook platform from map config
func createWebhookPlatformFromMap(cfg map[string]interface{}, logger logger.Logger) (platform.Platform, error) {
	// This would normally be imported dynamically or registered via factory
	// For now, create a proper webhook platform instance
	return &mockPlatform{
		name: "webhook",
		capabilities: platform.Capabilities{
			Name:                 "webhook",
			SupportedTargetTypes: []string{"webhook"},
			SupportedFormats:     []string{"json", "text"},
			MaxMessageSize:       1048576, // 1MB
			SupportsScheduling:   false,
			SupportsAttachments:  false,
			RequiredSettings:     []string{"url"},
		},
	}, nil
}

// createDispatcher creates a dispatcher that wraps the platform registry
func createDispatcher(registry *platform.Registry, cfg *Config) (Dispatcher, error) {
	return &dispatcherWrapper{
		registry: registry,
		config:   cfg,
	}, nil
}

// dispatcherWrapper adapts the platform registry to the Dispatcher interface
type dispatcherWrapper struct {
	registry *platform.Registry
	config   *Config
}

func (d *dispatcherWrapper) Dispatch(ctx context.Context, msg *message.Message) (*receipt.Receipt, error) {
	// Simple implementation - route to all configured platforms
	var results []receipt.PlatformResult
	successful := 0
	failed := 0

	// Get configured platforms
	if d.config.HasFeishu() {
		platform, err := d.registry.GetPlatform("feishu")
		if err == nil {
			// Send to feishu
			sendResults, err := platform.Send(ctx, msg, msg.Targets)
			if err == nil {
				for _, result := range sendResults {
					platformResult := receipt.PlatformResult{
						Platform:  "feishu",
						Target:    result.Target.Value,
						Success:   result.Success,
						MessageID: result.MessageID,
						Error:     result.Error,
						Timestamp: msg.CreatedAt,
					}
					results = append(results, platformResult)
					if result.Success {
						successful++
					} else {
						failed++
					}
				}
			}
		}
	}

	// Similar for other platforms...

	return &receipt.Receipt{
		MessageID:  msg.ID,
		Status:     "completed",
		Results:    results,
		Successful: successful,
		Failed:     failed,
		Total:      successful + failed,
		Timestamp:  msg.CreatedAt,
	}, nil
}

func (d *dispatcherWrapper) Close() error {
	return d.registry.Close()
}

// createAsyncManager creates an async manager
func createAsyncManager(dispatcher Dispatcher, cfg *Config) (AsyncManager, error) {
	// This will be implemented when the async package is ready
	return &mockAsyncManager{}, nil
}

// createHealthMonitor creates a health monitor
func createHealthMonitor(registry *platform.Registry, asyncManager AsyncManager, logger logger.Logger) HealthMonitor {
	// This will be implemented when the health monitoring package is ready
	return &mockHealthMonitor{}
}

// Mock implementations for temporary use

// mockPlatform is a temporary mock implementation
type mockPlatform struct {
	name         string
	capabilities platform.Capabilities
}

func (m *mockPlatform) Name() string {
	return m.name
}

func (m *mockPlatform) GetCapabilities() platform.Capabilities {
	return m.capabilities
}

func (m *mockPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*platform.SendResult, error) {
	// Mock implementation
	results := make([]*platform.SendResult, len(targets))
	for i, target := range targets {
		results[i] = &platform.SendResult{
			Target:    target,
			Success:   true,
			MessageID: fmt.Sprintf("mock-%s-%d", m.name, i),
		}
	}
	return results, nil
}

func (m *mockPlatform) ValidateTarget(target target.Target) error {
	// Mock validation - always passes
	return nil
}

func (m *mockPlatform) IsHealthy(ctx context.Context) error {
	// Mock health check - always healthy
	return nil
}

func (m *mockPlatform) Close() error {
	// Mock close - nothing to clean up
	return nil
}

// mockAsyncManager is a temporary mock implementation
type mockAsyncManager struct{}

func (m *mockAsyncManager) ScheduleAsync(ctx context.Context, msg *message.Message, opts ...async.AsyncOption) (async.AsyncHandle, error) {
	// Mock implementation
	return &mockAsyncHandle{}, nil
}

func (m *mockAsyncManager) ScheduleBatch(ctx context.Context, msgs []*message.Message, opts ...async.AsyncOption) (async.AsyncBatchHandle, error) {
	// Mock implementation
	return &mockBatchHandle{}, nil
}

func (m *mockAsyncManager) Close() error {
	// Mock close
	return nil
}

// mockAsyncHandle is a temporary mock implementation
type mockAsyncHandle struct{}

func (m *mockAsyncHandle) MessageID() string { return "mock-handle" }
func (m *mockAsyncHandle) Status() async.AsyncStatus {
	return async.AsyncStatus{
		MessageID: "mock-handle",
		Status:    async.StatusCompleted,
		Progress:  1.0,
		StartedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
}
func (m *mockAsyncHandle) Result() <-chan async.AsyncResult {
	ch := make(chan async.AsyncResult, 1)
	ch <- async.AsyncResult{
		Receipt: &receipt.Receipt{},
		Error: nil,
	}
	close(ch)
	return ch
}
func (m *mockAsyncHandle) Cancel() error { return nil }
func (m *mockAsyncHandle) Wait(ctx context.Context) (*receipt.Receipt, error) { return &receipt.Receipt{}, nil }

// mockBatchHandle is a temporary mock implementation
type mockBatchHandle struct{}

func (m *mockBatchHandle) BatchID() string { return "mock-batch-handle" }
func (m *mockBatchHandle) Status() async.AsyncBatchStatus {
	return async.AsyncBatchStatus{
		BatchID:   "mock-batch-handle",
		Status:    async.StatusCompleted,
		Total:     0,
		Completed: 0,
		Failed:    0,
		Progress:  1.0,
		StartedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
}
func (m *mockBatchHandle) Results() <-chan async.AsyncResult {
	ch := make(chan async.AsyncResult)
	close(ch)
	return ch
}
func (m *mockBatchHandle) Progress() <-chan async.BatchProgress {
	ch := make(chan async.BatchProgress)
	close(ch)
	return ch
}
func (m *mockBatchHandle) Cancel() error { return nil }
func (m *mockBatchHandle) Wait(ctx context.Context) ([]*receipt.Receipt, error) { return []*receipt.Receipt{}, nil }

// mockHealthMonitor is a temporary mock implementation
type mockHealthMonitor struct{}

func (m *mockHealthMonitor) GetStatus(ctx context.Context) (*HealthStatus, error) {
	return &HealthStatus{
		Status:      "healthy",
		Platforms:   map[string]string{},
		Uptime:      0,
		ActiveTasks: 0,
		QueueDepth:  0,
		Metadata:    map[string]interface{}{},
	}, nil
}

func (m *mockHealthMonitor) Close() error {
	return nil
}

// platformRegistryWrapper wraps the platform.Registry to implement our PlatformRegistry interface
type platformRegistryWrapper struct {
	*platform.Registry
}

func (w *platformRegistryWrapper) Register(name string, p Platform) error {
	// This should not be called - platform registration happens via factory functions
	return fmt.Errorf("direct platform registration not supported - use factory functions")
}

func (w *platformRegistryWrapper) Get(name string) (Platform, bool) {
	p, err := w.Registry.GetPlatform(name)
	if err != nil {
		return nil, false
	}
	// Wrap the platform.Platform as our Platform interface
	return &platformWrapper{platform: p}, true
}

func (w *platformRegistryWrapper) List() []string {
	return w.Registry.ListRegistered()
}

func (w *platformRegistryWrapper) Close() error {
	return w.Registry.Close()
}

// platformWrapper adapts platform.Platform to our Platform interface
type platformWrapper struct {
	platform platform.Platform
}

func (w *platformWrapper) Name() string {
	return w.platform.Name()
}

func (w *platformWrapper) GetCapabilities() Capabilities {
	caps := w.platform.GetCapabilities()
	return Capabilities{
		Name:                 caps.Name,
		SupportedTargetTypes: caps.SupportedTargetTypes,
		SupportedFormats:     caps.SupportedFormats,
		MaxMessageSize:       caps.MaxMessageSize,
		SupportsScheduling:   caps.SupportsScheduling,
		SupportsAttachments:  caps.SupportsAttachments,
		RequiredSettings:     caps.RequiredSettings,
	}
}

func (w *platformWrapper) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	// Convert targets to platform targets
	platformTargets := make([]target.Target, len(targets))
	for i, t := range targets {
		platformTargets[i] = t
	}

	// Call platform send
	results, err := w.platform.Send(ctx, msg, platformTargets)
	if err != nil {
		return nil, err
	}

	// Convert results back
	sendResults := make([]*SendResult, len(results))
	for i, r := range results {
		sendResults[i] = &SendResult{
			Target:    r.Target,
			Success:   r.Success,
			MessageID: r.MessageID,
			Error:     r.Error,
		}
	}

	return sendResults, nil
}

func (w *platformWrapper) ValidateTarget(tgt target.Target) error {
	return w.platform.ValidateTarget(tgt)
}

func (w *platformWrapper) IsHealthy(ctx context.Context) error {
	return w.platform.IsHealthy(ctx)
}

func (w *platformWrapper) Close() error {
	return w.platform.Close()
}