// Package platform provides instance-level platform registry for NotifyHub
package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Platform represents the unified platform interface that all notification platforms must implement
// This interface eliminates the previous dual-layer architecture and provides a single, consistent interface
type Platform interface {
	// Platform identification
	Name() string
	GetCapabilities() Capabilities

	// Message sending (core functionality)
	Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error)
	ValidateTarget(target target.Target) error

	// Lifecycle management
	IsHealthy(ctx context.Context) error
	Close() error
}

// SendResult represents the result of sending to a specific target
type SendResult struct {
	Target    target.Target          `json:"target"`
	Success   bool                   `json:"success"`
	MessageID string                 `json:"message_id,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Response  string                 `json:"response,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Capabilities describes what a platform supports
type Capabilities struct {
	Name                 string   `json:"name"`
	SupportedTargetTypes []string `json:"supported_target_types"`
	SupportedFormats     []string `json:"supported_formats"`
	MaxMessageSize       int      `json:"max_message_size"`
	SupportsScheduling   bool     `json:"supports_scheduling"`
	SupportsAttachments  bool     `json:"supports_attachments"`
	SupportsMentions     bool     `json:"supports_mentions"`
	SupportsRichContent  bool     `json:"supports_rich_content"`
	RequiredSettings     []string `json:"required_settings"`
}

// PlatformCreator is a function that creates a platform with given configuration
type PlatformCreator func(config map[string]interface{}, logger logger.Logger) (Platform, error)

// PlatformStatus represents the status of a platform instance
type PlatformStatus int

const (
	StatusUnknown PlatformStatus = iota
	StatusInitializing
	StatusHealthy
	StatusUnhealthy
	StatusShuttingDown
	StatusShutdown
)

func (s PlatformStatus) String() string {
	switch s {
	case StatusInitializing:
		return "initializing"
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	case StatusShuttingDown:
		return "shutting_down"
	case StatusShutdown:
		return "shutdown"
	default:
		return "unknown"
	}
}

// PlatformInfo contains metadata about a platform instance
type PlatformInfo struct {
	Name           string                 `json:"name"`
	Status         PlatformStatus         `json:"status"`
	Capabilities   Capabilities           `json:"capabilities"`
	LastHealthCheck *time.Time             `json:"last_health_check,omitempty"`
	HealthError    string                 `json:"health_error,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	Metrics        PlatformMetrics        `json:"metrics"`
	Config         map[string]interface{} `json:"config,omitempty"`
}

// PlatformMetrics contains platform performance metrics
type PlatformMetrics struct {
	TotalRequests   int64         `json:"total_requests"`
	SuccessfulSends int64         `json:"successful_sends"`
	FailedSends     int64         `json:"failed_sends"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastActivity    *time.Time    `json:"last_activity,omitempty"`
}

// HealthCheckConfig configures platform health monitoring
type HealthCheckConfig struct {
	Enabled          bool          `json:"enabled"`
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	RetryThreshold   int           `json:"retry_threshold"`
	UnhealthyTimeout time.Duration `json:"unhealthy_timeout"`
}

// DefaultHealthCheckConfig returns default health check configuration
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Enabled:          true,
		Interval:         30 * time.Second,
		Timeout:          5 * time.Second,
		RetryThreshold:   3,
		UnhealthyTimeout: 5 * time.Minute,
	}
}

// registryEntry represents an entry in the platform registry
type registryEntry struct {
	creator     PlatformCreator
	instance    Platform
	info        *PlatformInfo
	healthCheck *healthChecker
	config      map[string]interface{}
}

// healthChecker manages health monitoring for a platform
type healthChecker struct {
	platform       Platform
	config         HealthCheckConfig
	stopCh         chan struct{}
	lastCheck      time.Time
	consecutiveFails int
	mu             sync.RWMutex
}

// Registry represents an instance-level platform registry
// This replaces the global registry to support multi-instance usage
type Registry struct {
	entries       map[string]*registryEntry
	logger        logger.Logger
	healthConfig  HealthCheckConfig
	shutdownCh    chan struct{}
	shutdownOnce  sync.Once
	mu            sync.RWMutex
}

// NewRegistry creates a new instance-level platform registry
func NewRegistry(logger logger.Logger) *Registry {
	if logger == nil {
		logger = &noopLogger{} // Fallback to noop if no logger provided
	}

	return &Registry{
		entries:      make(map[string]*registryEntry),
		logger:       logger,
		healthConfig: DefaultHealthCheckConfig(),
		shutdownCh:   make(chan struct{}),
	}
}

// NewRegistryWithHealthConfig creates a registry with custom health check configuration
func NewRegistryWithHealthConfig(logger logger.Logger, healthConfig HealthCheckConfig) *Registry {
	if logger == nil {
		logger = &noopLogger{}
	}

	return &Registry{
		entries:      make(map[string]*registryEntry),
		logger:       logger,
		healthConfig: healthConfig,
		shutdownCh:   make(chan struct{}),
	}
}

// Register registers a platform creator with the registry
func (r *Registry) Register(name string, creator PlatformCreator) error {
	if name == "" {
		return fmt.Errorf("platform name cannot be empty")
	}
	if creator == nil {
		return fmt.Errorf("platform creator cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if registry is shutting down
	select {
	case <-r.shutdownCh:
		return fmt.Errorf("registry is shutting down")
	default:
	}

	if entry, exists := r.entries[name]; exists {
		r.logger.Debug("Platform creator already registered, overwriting", "platform", name)
		// Close existing instance if it exists
		if entry.instance != nil {
			r.shutdownPlatform(name, entry)
		}
	} else {
		r.entries[name] = &registryEntry{}
	}

	r.entries[name].creator = creator
	r.logger.Debug("Platform creator registered", "platform", name)

	return nil
}

// Unregister removes a platform creator and instance from the registry
func (r *Registry) Unregister(name string) error {
	if name == "" {
		return fmt.Errorf("platform name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.entries[name]
	if !exists {
		return fmt.Errorf("platform %s not registered", name)
	}

	// Shutdown platform instance and health monitoring
	r.shutdownPlatform(name, entry)

	// Remove entry completely
	delete(r.entries, name)

	r.logger.Debug("Platform unregistered", "platform", name)
	return nil
}

// SetConfig sets configuration for a platform
func (r *Registry) SetConfig(name string, config map[string]interface{}) error {
	if name == "" {
		return fmt.Errorf("platform name cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("platform config cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.entries[name]
	if !exists {
		r.entries[name] = &registryEntry{}
		entry = r.entries[name]
	}

	// If configuration changed and instance exists, recreate it
	if entry.instance != nil && !r.configEquals(entry.config, config) {
		r.logger.Info("Configuration changed, recreating platform instance", "platform", name)
		r.shutdownPlatform(name, entry)
	}

	entry.config = config
	r.logger.Debug("Platform config set", "platform", name)

	return nil
}

// GetPlatform gets or creates a platform instance
func (r *Registry) GetPlatform(name string) (Platform, error) {
	if name == "" {
		return nil, fmt.Errorf("platform name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if registry is shutting down
	select {
	case <-r.shutdownCh:
		return nil, fmt.Errorf("registry is shutting down")
	default:
	}

	entry, exists := r.entries[name]
	if !exists {
		r.logger.Error("Platform not registered", "platform", name)
		return nil, fmt.Errorf("platform %s not registered", name)
	}

	// Return existing healthy instance if available
	if entry.instance != nil && entry.info != nil && entry.info.Status == StatusHealthy {
		r.logger.Debug("Returning existing healthy platform instance", "platform", name)
		return entry.instance, nil
	}

	// Check if creator is available
	if entry.creator == nil {
		r.logger.Error("Platform creator not available", "platform", name)
		return nil, fmt.Errorf("platform %s creator not available", name)
	}

	// Check if platform is configured
	if entry.config == nil {
		r.logger.Error("Platform not configured", "platform", name)
		return nil, fmt.Errorf("platform %s not configured", name)
	}

	// Create new platform instance
	return r.createPlatformInstance(name, entry)
}

// createPlatformInstance creates a new platform instance with lifecycle management
func (r *Registry) createPlatformInstance(name string, entry *registryEntry) (Platform, error) {
	// Update status to initializing
	if entry.info == nil {
		entry.info = &PlatformInfo{
			Name:      name,
			CreatedAt: time.Now(),
			Metrics:   PlatformMetrics{},
		}
	}
	entry.info.Status = StatusInitializing

	r.logger.Info("Creating platform instance", "platform", name)

	// Create platform instance
	instance, err := entry.creator(entry.config, r.logger)
	if err != nil {
		r.logger.Error("Failed to create platform", "platform", name, "error", err)
		entry.info.Status = StatusUnhealthy
		return nil, fmt.Errorf("failed to create platform %s: %w", name, err)
	}

	// Store instance and update info
	entry.instance = instance
	entry.info.Capabilities = instance.GetCapabilities()
	entry.info.Status = StatusHealthy

	// Start health monitoring if enabled
	if r.healthConfig.Enabled {
		entry.healthCheck = r.startHealthMonitoring(name, instance)
	}

	r.logger.Info("Platform instance created successfully", "platform", name)
	return instance, nil
}

// shutdownPlatform gracefully shuts down a platform instance
func (r *Registry) shutdownPlatform(name string, entry *registryEntry) {
	if entry.info != nil {
		entry.info.Status = StatusShuttingDown
	}

	// Stop health monitoring
	if entry.healthCheck != nil {
		r.stopHealthMonitoring(entry.healthCheck)
		entry.healthCheck = nil
	}

	// Close platform instance
	if entry.instance != nil {
		if err := entry.instance.Close(); err != nil {
			r.logger.Warn("Failed to close platform during shutdown", "platform", name, "error", err)
		} else {
			r.logger.Debug("Platform closed successfully", "platform", name)
		}
		entry.instance = nil
	}

	if entry.info != nil {
		entry.info.Status = StatusShutdown
	}
}

// configEquals compares two configuration maps for equality
func (r *Registry) configEquals(config1, config2 map[string]interface{}) bool {
	if len(config1) != len(config2) {
		return false
	}

	for key, value1 := range config1 {
		value2, exists := config2[key]
		if !exists {
			return false
		}
		// Simple comparison - for complex types, this could be enhanced
		if fmt.Sprintf("%v", value1) != fmt.Sprintf("%v", value2) {
			return false
		}
	}

	return true
}

// IsRegistered checks if a platform creator is registered
func (r *Registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[name]
	return exists && entry.creator != nil
}

// IsConfigured checks if a platform is configured
func (r *Registry) IsConfigured(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[name]
	return exists && entry.config != nil
}

// IsHealthy checks if a platform instance is healthy
func (r *Registry) IsHealthy(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[name]
	return exists && entry.info != nil && entry.info.Status == StatusHealthy
}

// GetPlatformInfo returns detailed information about a platform
func (r *Registry) GetPlatformInfo(name string) (*PlatformInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[name]
	if !exists {
		return nil, fmt.Errorf("platform %s not found", name)
	}

	if entry.info == nil {
		return &PlatformInfo{
			Name:   name,
			Status: StatusUnknown,
		}, nil
	}

	// Create a copy to avoid concurrent access issues
	info := &PlatformInfo{
		Name:           entry.info.Name,
		Status:         entry.info.Status,
		Capabilities:   entry.info.Capabilities,
		HealthError:    entry.info.HealthError,
		CreatedAt:      entry.info.CreatedAt,
		Metrics:        entry.info.Metrics,
	}

	if entry.info.LastHealthCheck != nil {
		lastCheck := *entry.info.LastHealthCheck
		info.LastHealthCheck = &lastCheck
	}

	// Include sanitized config (without sensitive data)
	if entry.config != nil {
		info.Config = r.sanitizeConfig(entry.config)
	}

	return info, nil
}

// sanitizeConfig removes sensitive information from configuration
func (r *Registry) sanitizeConfig(config map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	sensitiveKeys := map[string]bool{
		"secret":     true,
		"password":   true,
		"token":      true,
		"api_key":    true,
		"private_key": true,
	}

	for key, value := range config {
		if sensitiveKeys[key] {
			sanitized[key] = "[REDACTED]"
		} else {
			sanitized[key] = value
		}
	}

	return sanitized
}

// ListRegistered returns all registered platform names
func (r *Registry) ListRegistered() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0)
	for name, entry := range r.entries {
		if entry.creator != nil {
			names = append(names, name)
		}
	}
	return names
}

// ListConfigured returns all configured platform names
func (r *Registry) ListConfigured() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0)
	for name, entry := range r.entries {
		if entry.config != nil {
			names = append(names, name)
		}
	}
	return names
}

// ListInstances returns all instantiated platform names
func (r *Registry) ListInstances() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0)
	for name, entry := range r.entries {
		if entry.instance != nil {
			names = append(names, name)
		}
	}
	return names
}

// ListByStatus returns platform names filtered by status
func (r *Registry) ListByStatus(status PlatformStatus) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0)
	for name, entry := range r.entries {
		if entry.info != nil && entry.info.Status == status {
			names = append(names, name)
		}
	}
	return names
}

// GetCapabilities returns capabilities for platforms that support specific features
func (r *Registry) GetCapabilities(targetType string) map[string]Capabilities {
	r.mu.RLock()
	defer r.mu.RUnlock()

	capabilities := make(map[string]Capabilities)
	for name, entry := range r.entries {
		if entry.info != nil {
			// Check if platform supports the target type
			for _, supportedType := range entry.info.Capabilities.SupportedTargetTypes {
				if supportedType == targetType {
					capabilities[name] = entry.info.Capabilities
					break
				}
			}
		}
	}
	return capabilities
}

// SelectPlatforms returns platforms that match the given criteria
func (r *Registry) SelectPlatforms(criteria PlatformCriteria) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []string
	for name, entry := range r.entries {
		if r.matchesCriteria(entry, criteria) {
			matches = append(matches, name)
		}
	}

	return matches
}

// PlatformCriteria defines criteria for platform selection
type PlatformCriteria struct {
	TargetType       string
	Format           string
	RequiresScheduling bool
	RequiresAttachments bool
	MinMessageSize   int
	HealthyOnly      bool
}

// matchesCriteria checks if a platform entry matches the given criteria
func (r *Registry) matchesCriteria(entry *registryEntry, criteria PlatformCriteria) bool {
	if entry.info == nil {
		return false
	}

	// Check health status if required
	if criteria.HealthyOnly && entry.info.Status != StatusHealthy {
		return false
	}

	caps := entry.info.Capabilities

	// Check target type support
	if criteria.TargetType != "" {
		found := false
		for _, supportedType := range caps.SupportedTargetTypes {
			if supportedType == criteria.TargetType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check format support
	if criteria.Format != "" {
		found := false
		for _, supportedFormat := range caps.SupportedFormats {
			if supportedFormat == criteria.Format {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check scheduling support
	if criteria.RequiresScheduling && !caps.SupportsScheduling {
		return false
	}

	// Check attachment support
	if criteria.RequiresAttachments && !caps.SupportsAttachments {
		return false
	}

	// Check message size
	if criteria.MinMessageSize > 0 && caps.MaxMessageSize < criteria.MinMessageSize {
		return false
	}

	return true
}

// Health checks the health of all platform instances
func (r *Registry) Health(ctx context.Context) map[string]error {
	if ctx == nil {
		ctx = context.Background()
	}

	r.mu.RLock()
	instances := make(map[string]Platform)
	for name, entry := range r.entries {
		if entry.instance != nil {
			instances[name] = entry.instance
		}
	}
	r.mu.RUnlock()

	health := make(map[string]error)
	for name, instance := range instances {
		// Create context with timeout for health check
		healthCtx, cancel := context.WithTimeout(ctx, r.healthConfig.Timeout)
		err := instance.IsHealthy(healthCtx)
		cancel()

		health[name] = err
		if err != nil {
			r.logger.Warn("Platform health check failed", "platform", name, "error", err)
			r.updatePlatformHealth(name, err)
		} else {
			r.logger.Debug("Platform health check passed", "platform", name)
			r.updatePlatformHealth(name, nil)
		}
	}

	return health
}

// HealthSummary returns a summary of platform health status
func (r *Registry) HealthSummary() map[string]PlatformInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	summary := make(map[string]PlatformInfo)
	for name, entry := range r.entries {
		if entry.info != nil {
			summary[name] = *entry.info
		} else {
			summary[name] = PlatformInfo{
				Name:   name,
				Status: StatusUnknown,
			}
		}
	}

	return summary
}

// updatePlatformHealth updates the health status of a platform
func (r *Registry) updatePlatformHealth(name string, healthErr error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.entries[name]
	if !exists || entry.info == nil {
		return
	}

	now := time.Now()
	entry.info.LastHealthCheck = &now

	if healthErr != nil {
		entry.info.Status = StatusUnhealthy
		entry.info.HealthError = healthErr.Error()
	} else {
		entry.info.Status = StatusHealthy
		entry.info.HealthError = ""
	}
}

// Close closes all platform instances and clears the registry
func (r *Registry) Close() error {
	r.shutdownOnce.Do(func() {
		close(r.shutdownCh)
	})

	r.mu.Lock()
	defer r.mu.Unlock()

	var lastError error
	for name, entry := range r.entries {
		r.logger.Debug("Closing platform", "platform", name)
		r.shutdownPlatform(name, entry)
	}

	// Clear all entries
	r.entries = make(map[string]*registryEntry)
	r.logger.Info("Platform registry closed")

	return lastError
}

// Shutdown gracefully shuts down the registry with timeout
func (r *Registry) Shutdown(ctx context.Context) error {
	select {
	case <-r.shutdownCh:
		return nil // Already shut down
	default:
	}

	r.logger.Info("Starting graceful registry shutdown")

	// Signal shutdown
	r.shutdownOnce.Do(func() {
		close(r.shutdownCh)
	})

	// Wait for ongoing operations to complete or timeout
	shutdownCh := make(chan error, 1)
	go func() {
		shutdownCh <- r.Close()
	}()

	select {
	case err := <-shutdownCh:
		r.logger.Info("Registry shutdown completed")
		return err
	case <-ctx.Done():
		r.logger.Warn("Registry shutdown timed out, forcing close")
		return r.Close()
	}
}

// RegisterBuiltinPlatforms registers all built-in platform creators
// This replaces the global init() pattern with explicit registration
func (r *Registry) RegisterBuiltinPlatforms() error {
	builtinPlatforms := map[string]PlatformCreator{
		// Built-in platforms will be registered here
		// This eliminates the need for init() functions in platform packages
	}

	for name, creator := range builtinPlatforms {
		if err := r.Register(name, creator); err != nil {
			return fmt.Errorf("failed to register builtin platform %s: %w", name, err)
		}
	}

	r.logger.Info("Built-in platforms registered", "count", len(builtinPlatforms))
	return nil
}

// StartPlatform explicitly starts a platform instance
func (r *Registry) StartPlatform(ctx context.Context, name string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	_, err := r.GetPlatform(name)
	if err != nil {
		return fmt.Errorf("failed to start platform %s: %w", name, err)
	}

	r.logger.Info("Platform started successfully", "platform", name)
	return nil
}

// StopPlatform stops a platform instance
func (r *Registry) StopPlatform(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.entries[name]
	if !exists {
		return fmt.Errorf("platform %s not found", name)
	}

	if entry.instance == nil {
		return fmt.Errorf("platform %s not running", name)
	}

	r.shutdownPlatform(name, entry)
	r.logger.Info("Platform stopped", "platform", name)
	return nil
}

// RestartPlatform restarts a platform instance
func (r *Registry) RestartPlatform(ctx context.Context, name string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	r.logger.Info("Restarting platform", "platform", name)

	// Stop the platform
	if err := r.StopPlatform(name); err != nil {
		r.logger.Warn("Failed to stop platform during restart, continuing", "platform", name, "error", err)
	}

	// Start the platform
	return r.StartPlatform(ctx, name)
}

// GetRegistryStats returns statistics about the registry
func (r *Registry) GetRegistryStats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RegistryStats{
		Total:      len(r.entries),
		StatusCounts: make(map[PlatformStatus]int),
	}

	for _, entry := range r.entries {
		if entry.creator != nil {
			stats.Registered++
		}
		if entry.config != nil {
			stats.Configured++
		}
		if entry.instance != nil {
			stats.Running++
		}
		if entry.info != nil {
			stats.StatusCounts[entry.info.Status]++
		} else {
			stats.StatusCounts[StatusUnknown]++
		}
	}

	return stats
}

// RegistryStats contains registry statistics
type RegistryStats struct {
	Total        int                       `json:"total"`
	Registered   int                       `json:"registered"`
	Configured   int                       `json:"configured"`
	Running      int                       `json:"running"`
	StatusCounts map[PlatformStatus]int    `json:"status_counts"`
}

// noopLogger is a fallback logger that does nothing
type noopLogger struct{}

// Health monitoring methods

// startHealthMonitoring starts health monitoring for a platform
func (r *Registry) startHealthMonitoring(name string, platform Platform) *healthChecker {
	if !r.healthConfig.Enabled {
		return nil
	}

	checker := &healthChecker{
		platform: platform,
		config:   r.healthConfig,
		stopCh:   make(chan struct{}),
	}

	// Start monitoring goroutine
	go r.runHealthMonitoring(name, checker)

	return checker
}

// stopHealthMonitoring stops health monitoring for a platform
func (r *Registry) stopHealthMonitoring(checker *healthChecker) {
	if checker != nil {
		close(checker.stopCh)
	}
}

// runHealthMonitoring runs the health monitoring loop
func (r *Registry) runHealthMonitoring(name string, checker *healthChecker) {
	ticker := time.NewTicker(checker.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-checker.stopCh:
			r.logger.Debug("Health monitoring stopped", "platform", name)
			return
		case <-r.shutdownCh:
			r.logger.Debug("Health monitoring stopped due to registry shutdown", "platform", name)
			return
		case <-ticker.C:
			r.performHealthCheck(name, checker)
		}
	}
}

// performHealthCheck performs a single health check
func (r *Registry) performHealthCheck(name string, checker *healthChecker) {
	checker.mu.Lock()
	defer checker.mu.Unlock()

	checker.lastCheck = time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), checker.config.Timeout)
	defer cancel()

	// Perform health check
	err := checker.platform.IsHealthy(ctx)

	if err != nil {
		checker.consecutiveFails++
		r.logger.Debug("Platform health check failed", "platform", name, "error", err, "consecutive_fails", checker.consecutiveFails)

		// Update platform status if threshold exceeded
		if checker.consecutiveFails >= checker.config.RetryThreshold {
			r.updatePlatformHealth(name, err)
		}
	} else {
		if checker.consecutiveFails > 0 {
			r.logger.Info("Platform health recovered", "platform", name, "previous_fails", checker.consecutiveFails)
		}
		checker.consecutiveFails = 0
		r.updatePlatformHealth(name, nil)
	}
}

// UpdateMetrics updates platform metrics
func (r *Registry) UpdateMetrics(name string, success bool, latency time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.entries[name]
	if !exists || entry.info == nil {
		return
	}

	entry.info.Metrics.TotalRequests++
	if success {
		entry.info.Metrics.SuccessfulSends++
	} else {
		entry.info.Metrics.FailedSends++
	}

	// Update average latency (simple moving average)
	if entry.info.Metrics.TotalRequests == 1 {
		entry.info.Metrics.AverageLatency = latency
	} else {
		// Calculate new average: (old_avg * (n-1) + new_value) / n
		n := entry.info.Metrics.TotalRequests
		oldAvg := entry.info.Metrics.AverageLatency
		newAvg := time.Duration((int64(oldAvg)*(n-1) + int64(latency)) / n)
		entry.info.Metrics.AverageLatency = newAvg
	}

	now := time.Now()
	entry.info.Metrics.LastActivity = &now
}

func (l *noopLogger) LogMode(level logger.LogLevel) logger.Logger { return l }
func (l *noopLogger) Debug(msg string, args ...any)               {}
func (l *noopLogger) Info(msg string, args ...any)                {}
func (l *noopLogger) Warn(msg string, args ...any)                {}
func (l *noopLogger) Error(msg string, args ...any)               {}

// Platform constants for built-in platforms
const (
	NameEmail   = "email"
	NameFeishu  = "feishu"
	NameSMS     = "sms"
	NameSlack   = "slack"
	NameDiscord = "discord"
	NameTeams   = "teams"
	NameWebhook = "webhook"
)

// Event system for registry changes

// RegistryEvent represents an event in the registry
type RegistryEvent struct {
	Type      EventType              `json:"type"`
	Platform  string                 `json:"platform"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// EventType represents the type of registry event
type EventType string

const (
	EventPlatformRegistered   EventType = "platform_registered"
	EventPlatformUnregistered EventType = "platform_unregistered"
	EventPlatformConfigured   EventType = "platform_configured"
	EventPlatformStarted      EventType = "platform_started"
	EventPlatformStopped      EventType = "platform_stopped"
	EventPlatformHealthy      EventType = "platform_healthy"
	EventPlatformUnhealthy    EventType = "platform_unhealthy"
)

// EventHandler is a function that handles registry events
type EventHandler func(event RegistryEvent)

// SetEventHandler sets an event handler for registry events
func (r *Registry) SetEventHandler(handler EventHandler) {
	// This is a placeholder for a future event system implementation
	// Events could be useful for monitoring, logging, and external integrations
}

// emitEvent emits a registry event (placeholder)
func (r *Registry) emitEvent(eventType EventType, platform string, data map[string]interface{}) {
	// This would emit events to registered handlers
	// For now, just log the event
	r.logger.Debug("Registry event", "type", eventType, "platform", platform)
}

// Export methods for testing and diagnostics

// ExportConfig exports the registry configuration for backup or migration
func (r *Registry) ExportConfig() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create export data structure
	exportData := struct {
		Version   string                              `json:"version"`
		Timestamp time.Time                         `json:"timestamp"`
		Platforms map[string]map[string]interface{} `json:"platforms"`
	}{
		Version:   "1.0",
		Timestamp: time.Now(),
		Platforms: make(map[string]map[string]interface{}),
	}

	// Export sanitized configurations
	for name, entry := range r.entries {
		if entry.config != nil {
			exportData.Platforms[name] = r.sanitizeConfig(entry.config)
		}
	}

	// Marshal to JSON
	return json.Marshal(exportData)
}

// ImportConfig imports registry configuration from backup
func (r *Registry) ImportConfig(data []byte) error {
	// This is a placeholder for configuration import functionality
	// Would parse the exported data and restore platform configurations
	return fmt.Errorf("import configuration not implemented yet")
}

// GetDiagnostics returns detailed diagnostic information about the registry
func (r *Registry) GetDiagnostics() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	diag := map[string]interface{}{
		"registry_stats":     r.GetRegistryStats(),
		"health_config":     r.healthConfig,
		"platform_details": make(map[string]interface{}),
	}

	// Add detailed platform information
	platformDetails := make(map[string]interface{})
	for name, entry := range r.entries {
		details := map[string]interface{}{
			"has_creator":  entry.creator != nil,
			"has_config":   entry.config != nil,
			"has_instance": entry.instance != nil,
		}

		if entry.info != nil {
			details["info"] = entry.info
		}

		platformDetails[name] = details
	}
	diag["platform_details"] = platformDetails

	return diag
}

// Backward compatibility aliases (deprecated)
// These are kept for backward compatibility and will be removed in a future version

// ExternalSender is deprecated: use Platform interface instead
type ExternalSender = Platform

// ExternalSenderCreator is deprecated: use PlatformCreator instead
type ExternalSenderCreator = PlatformCreator

// Deprecated global functions for backward compatibility
// These will be removed in a future version
// WARNING: These functions use global state and are not thread-safe for multi-instance usage

// Discovery and plugin loading methods

// DiscoverPlatforms discovers available platforms from registered plugins
func (r *Registry) DiscoverPlatforms() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	available := make([]string, 0)
	for name, entry := range r.entries {
		if entry.creator != nil {
			available = append(available, name)
		}
	}

	return available
}

// LoadPlatformPlugin loads a platform from a plugin (placeholder for future plugin system)
func (r *Registry) LoadPlatformPlugin(pluginPath, platformName string) error {
	// This is a placeholder for a future plugin loading system
	// For now, return an error indicating the feature is not implemented
	return fmt.Errorf("plugin loading not implemented yet")
}

// ConfigurePlatformFromEnv configures a platform from environment variables
func (r *Registry) ConfigurePlatformFromEnv(name, envPrefix string) error {
	// This would read configuration from environment variables
	// with the given prefix (e.g., NOTIFYHUB_FEISHU_WEBHOOK_URL)
	// For now, return an error indicating the feature needs implementation
	return fmt.Errorf("environment configuration not implemented yet")
}

// Platform load balancing and failover methods

// GetHealthyPlatforms returns all healthy platforms that support the given target type
func (r *Registry) GetHealthyPlatforms(targetType string) []string {
	criteria := PlatformCriteria{
		TargetType:  targetType,
		HealthyOnly: true,
	}
	return r.SelectPlatforms(criteria)
}

// GetBestPlatform returns the best platform for the given criteria
// Selection is based on health status, capabilities, and performance metrics
func (r *Registry) GetBestPlatform(criteria PlatformCriteria) (string, error) {
	matches := r.SelectPlatforms(criteria)
	if len(matches) == 0 {
		return "", fmt.Errorf("no platforms match the given criteria")
	}

	// Simple selection strategy: prefer healthy platforms with better metrics
	best := matches[0]
	bestScore := r.calculatePlatformScore(best)

	for _, name := range matches[1:] {
		score := r.calculatePlatformScore(name)
		if score > bestScore {
			best = name
			bestScore = score
		}
	}

	return best, nil
}

// calculatePlatformScore calculates a score for platform selection
func (r *Registry) calculatePlatformScore(name string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[name]
	if !exists || entry.info == nil {
		return 0.0
	}

	score := 0.0

	// Health status score
	switch entry.info.Status {
	case StatusHealthy:
		score += 100.0
	case StatusInitializing:
		score += 50.0
	case StatusUnhealthy:
		score += 10.0
	default:
		score += 0.0
	}

	// Performance metrics score
	metrics := entry.info.Metrics
	if metrics.TotalRequests > 0 {
		successRate := float64(metrics.SuccessfulSends) / float64(metrics.TotalRequests)
		score += successRate * 50.0

		// Lower latency is better
		if metrics.AverageLatency > 0 {
			latencyScore := 1000.0 / float64(metrics.AverageLatency.Milliseconds())
			score += latencyScore
		}
	}

	return score
}

var deprecatedGlobalRegistry = make(map[string]PlatformCreator)
var deprecatedMutex sync.RWMutex

// RegisterPlatform registers a platform creator globally (deprecated)
// Use Registry.Register() instead for instance-level registration
// WARNING: This function is deprecated and will be removed in v2.0
func RegisterPlatform(platformName string, creator PlatformCreator) {
	deprecatedMutex.Lock()
	defer deprecatedMutex.Unlock()
	deprecatedGlobalRegistry[platformName] = creator
}

// GetRegisteredCreators returns all registered platform creators (deprecated)
// Use Registry.ListRegistered() instead
func GetRegisteredCreators() map[string]PlatformCreator {
	deprecatedMutex.RLock()
	defer deprecatedMutex.RUnlock()

	result := make(map[string]PlatformCreator)
	for name, creator := range deprecatedGlobalRegistry {
		result[name] = creator
	}
	return result
}

// GetRegisteredPlatforms returns a list of all registered platform names (deprecated)
// Use Registry.ListRegistered() instead
func GetRegisteredPlatforms() []string {
	deprecatedMutex.RLock()
	defer deprecatedMutex.RUnlock()

	names := make([]string, 0, len(deprecatedGlobalRegistry))
	for name := range deprecatedGlobalRegistry {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a platform is registered (deprecated)
// Use Registry.IsRegistered() instead
func IsRegistered(platformName string) bool {
	deprecatedMutex.RLock()
	defer deprecatedMutex.RUnlock()

	_, exists := deprecatedGlobalRegistry[platformName]
	return exists
}

// Advanced registry operations

// BatchOperation performs multiple registry operations atomically
func (r *Registry) BatchOperation(operations []RegistryOperation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate all operations first
	for i, op := range operations {
		if err := r.validateOperation(op); err != nil {
			return fmt.Errorf("operation %d validation failed: %w", i, err)
		}
	}

	// Execute all operations
	for i, op := range operations {
		if err := r.executeOperation(op); err != nil {
			return fmt.Errorf("operation %d execution failed: %w", i, err)
		}
	}

	r.logger.Info("Batch operation completed", "operations", len(operations))
	return nil
}

// RegistryOperation represents a registry operation
type RegistryOperation struct {
	Type     OperationType              `json:"type"`
	Platform string                     `json:"platform"`
	Creator  PlatformCreator            `json:"-"`
	Config   map[string]interface{}     `json:"config,omitempty"`
}

// OperationType represents the type of registry operation
type OperationType string

const (
	OpRegister     OperationType = "register"
	OpUnregister   OperationType = "unregister"
	OpConfigure    OperationType = "configure"
	OpStart        OperationType = "start"
	OpStop         OperationType = "stop"
	OpRestart      OperationType = "restart"
)

// validateOperation validates a registry operation
func (r *Registry) validateOperation(op RegistryOperation) error {
	if op.Platform == "" {
		return fmt.Errorf("platform name cannot be empty")
	}

	switch op.Type {
	case OpRegister:
		if op.Creator == nil {
			return fmt.Errorf("creator cannot be nil for register operation")
		}
	case OpConfigure:
		if op.Config == nil {
			return fmt.Errorf("config cannot be nil for configure operation")
		}
	case OpUnregister, OpStart, OpStop, OpRestart:
		// No additional validation needed
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}

	return nil
}

// executeOperation executes a registry operation
func (r *Registry) executeOperation(op RegistryOperation) error {
	switch op.Type {
	case OpRegister:
		return r.register(op.Platform, op.Creator)
	case OpUnregister:
		return r.unregister(op.Platform)
	case OpConfigure:
		return r.setConfig(op.Platform, op.Config)
	case OpStart:
		_, err := r.createPlatformInstance(op.Platform, r.entries[op.Platform])
		return err
	case OpStop:
		entry := r.entries[op.Platform]
		if entry != nil {
			r.shutdownPlatform(op.Platform, entry)
		}
		return nil
	case OpRestart:
		entry := r.entries[op.Platform]
		if entry != nil {
			r.shutdownPlatform(op.Platform, entry)
			_, err := r.createPlatformInstance(op.Platform, entry)
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// register is the internal register method (without locking)
func (r *Registry) register(name string, creator PlatformCreator) error {
	if _, exists := r.entries[name]; !exists {
		r.entries[name] = &registryEntry{}
	}
	r.entries[name].creator = creator
	return nil
}

// unregister is the internal unregister method (without locking)
func (r *Registry) unregister(name string) error {
	entry, exists := r.entries[name]
	if !exists {
		return fmt.Errorf("platform %s not registered", name)
	}
	r.shutdownPlatform(name, entry)
	delete(r.entries, name)
	return nil
}

// setConfig is the internal setConfig method (without locking)
func (r *Registry) setConfig(name string, config map[string]interface{}) error {
	entry, exists := r.entries[name]
	if !exists {
		r.entries[name] = &registryEntry{}
		entry = r.entries[name]
	}

	// If configuration changed and instance exists, recreate it
	if entry.instance != nil && !r.configEquals(entry.config, config) {
		r.shutdownPlatform(name, entry)
	}

	entry.config = config
	return nil
}