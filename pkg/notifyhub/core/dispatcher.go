// Package core provides the core message dispatcher for NotifyHub
package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Dispatcher represents the core message dispatcher interface
// This implements the simplified 3-layer calling chain: Client → Dispatcher → Platform
// eliminating the complex 6-layer chain identified in the architecture analysis
type Dispatcher interface {
	// RegisterPlatform registers a platform creator (Hub-level, not global)
	RegisterPlatform(name string, creator platform.PlatformCreator)

	// Dispatch sends a message through appropriate platforms
	Dispatch(ctx context.Context, msg *message.Message) (*receipt.Receipt, error)

	// Health checks the health of all registered platforms
	Health(ctx context.Context) (map[string]string, error)

	// Close gracefully shuts down the dispatcher
	Close() error
}

// PlatformManager interface for managing platforms at Hub level (no global registry)
type PlatformManager interface {
	// RegisterPlatform registers a platform creator function
	RegisterPlatform(name string, creator platform.PlatformCreator)
	// GetPlatform gets a platform by name, creating it if necessary
	GetPlatform(name string) (platform.Platform, error)
	// Health checks health of all platforms
	Health(ctx context.Context) (map[string]string, error)
	// ListPlatforms lists all registered platforms
	ListPlatforms() []string
	// Close closes all platforms
	Close() error
}

// HubPlatformManager implements PlatformManager without relying on global registry
type HubPlatformManager struct {
	creators  map[string]platform.PlatformCreator
	platforms map[string]platform.Platform
	config    *config.Config
	logger    logger.Logger
	mutex     sync.RWMutex
}

// NewPlatformManager creates a new Hub-level platform manager
func NewPlatformManager(config *config.Config, logger logger.Logger) PlatformManager {
	return &HubPlatformManager{
		creators:  make(map[string]platform.PlatformCreator),
		platforms: make(map[string]platform.Platform),
		config:    config,
		logger:    logger,
	}
}

// RegisterPlatform registers a platform creator function
func (m *HubPlatformManager) RegisterPlatform(name string, creator platform.PlatformCreator) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.creators[name] = creator
	m.logger.Debug("Platform creator registered", "platform", name)
}

// GetPlatform gets a platform by name, creating it if necessary
func (m *HubPlatformManager) GetPlatform(name string) (platform.Platform, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Return existing platform if available
	if p, exists := m.platforms[name]; exists {
		return p, nil
	}

	// Check if creator is registered
	creator, exists := m.creators[name]
	if !exists {
		m.logger.Error("Platform not registered", "platform", name)
		return nil, fmt.Errorf("platform %s not registered", name)
	}

	// Get platform configuration (use strong-typed config with fallback to legacy)
	platformConfig := m.config.GetPlatformConfig(name)
	if platformConfig == nil {
		m.logger.Error("Platform not configured", "platform", name)
		return nil, fmt.Errorf("platform %s not configured", name)
	}

	// Create platform instance
	p, err := creator(platformConfig, m.logger)
	if err != nil {
		m.logger.Error("Failed to create platform", "platform", name, "error", err)
		return nil, fmt.Errorf("failed to create platform %s: %w", name, err)
	}

	// Cache the platform
	m.platforms[name] = p
	m.logger.Debug("Platform created and cached", "platform", name)

	return p, nil
}

// Health checks health of all platforms
func (m *HubPlatformManager) Health(ctx context.Context) (map[string]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]string)

	for name, p := range m.platforms {
		if err := p.IsHealthy(ctx); err != nil {
			result[name] = "unhealthy: " + err.Error()
		} else {
			result[name] = "healthy"
		}
	}

	return result, nil
}

// ListPlatforms lists all registered platforms
func (m *HubPlatformManager) ListPlatforms() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	platforms := make([]string, 0, len(m.creators))
	for name := range m.creators {
		platforms = append(platforms, name)
	}
	return platforms
}

// Close closes all platforms
func (m *HubPlatformManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var lastErr error
	for name, p := range m.platforms {
		if err := p.Close(); err != nil {
			m.logger.Error("Failed to close platform", "platform", name, "error", err)
			lastErr = err
		}
	}

	// Clear the maps
	m.platforms = make(map[string]platform.Platform)

	return lastErr
}

// DispatcherImpl implements the Dispatcher interface
type DispatcherImpl struct {
	platformManager PlatformManager
	router          target.Router
	logger          logger.Logger
	config          *config.Config
}

// NewDispatcher creates a new message dispatcher with Hub-level platform management
func NewDispatcher(cfg *config.Config, logger logger.Logger) (Dispatcher, error) {
	// Create new Hub-level platform manager (no global registry dependency)
	platformManager := NewPlatformManager(cfg, logger)

	// Create smart router with load balancing and fault tolerance
	routerConfig := target.RouterConfig{
		LoadBalancing:       "round_robin", // Default strategy
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:          3,
		RetryDelay:          time.Second,
	}

	// Override router config from main config if available
	if cfg.AsyncConfig.QueueType != "" {
		// Use async config as router config (they share similar concepts)
		routerConfig.HealthCheckInterval = time.Duration(cfg.AsyncConfig.Workers) * time.Second
	}

	router := target.NewSmartRouter(routerConfig, logger)

	dispatcher := &DispatcherImpl{
		platformManager: platformManager,
		router:          router,
		logger:          logger,
		config:          cfg,
	}

	return dispatcher, nil
}

// RegisterPlatform registers a platform creator with the dispatcher's platform manager
func (d *DispatcherImpl) RegisterPlatform(name string, creator platform.PlatformCreator) {
	d.platformManager.RegisterPlatform(name, creator)
}

// Dispatch sends a message through the appropriate platforms using smart routing
func (d *DispatcherImpl) Dispatch(ctx context.Context, msg *message.Message) (*receipt.Receipt, error) {
	d.logger.Debug("Dispatching message", "message_id", msg.ID, "targets", len(msg.Targets))

	// Use smart router for intelligent platform selection with load balancing
	platformTargets, err := d.router.RouteTargets(msg.Targets)
	if err != nil {
		d.logger.Error("Failed to route targets", "message_id", msg.ID, "error", err)
		return nil, fmt.Errorf("target routing failed: %w", err)
	}

	// Send to each platform
	var allResults []receipt.PlatformResult
	successful := 0
	failed := 0

	for platformName, targets := range platformTargets {
		startTime := time.Now()

		platform, err := d.platformManager.GetPlatform(platformName)
		if err != nil {
			d.logger.Error("Platform not found", "platform", platformName, "error", err)
			failed += len(targets)

			// Update router about platform health
			d.router.UpdatePlatformHealth(platformName, false, time.Since(startTime))
			continue
		}

		// Send to platform using unified interface
		results, err := platform.Send(ctx, msg, targets)
		responseTime := time.Since(startTime)

		if err != nil {
			d.logger.Error("Platform send failed", "platform", platformName, "error", err)
			failed += len(targets)

			// Update router about platform health (failed)
			d.router.UpdatePlatformHealth(platformName, false, responseTime)
			continue
		}

		// Update router about platform health (successful)
		d.router.UpdatePlatformHealth(platformName, true, responseTime)

		// Convert platform results to receipt format
		for _, result := range results {
			platformResult := receipt.PlatformResult{
				Platform:  platformName,
				Target:    result.Target.Value,
				Success:   result.Success,
				MessageID: result.MessageID,
				Error:     result.Error,
				Timestamp: msg.CreatedAt,
			}
			allResults = append(allResults, platformResult)

			if result.Success {
				successful++
			} else {
				failed++
			}
		}
	}

	// Create receipt
	result := &receipt.Receipt{
		MessageID:  msg.ID,
		Status:     "completed",
		Results:    allResults,
		Successful: successful,
		Failed:     failed,
		Total:      successful + failed,
		Timestamp:  msg.CreatedAt,
	}

	d.logger.Debug("Message dispatched with smart routing", "message_id", msg.ID, "successful", successful, "failed", failed)
	return result, nil
}

// Health checks the health of all platforms
func (d *DispatcherImpl) Health(ctx context.Context) (map[string]string, error) {
	return d.platformManager.Health(ctx)
}

// Close gracefully shuts down the dispatcher
func (d *DispatcherImpl) Close() error {
	d.logger.Info("Closing dispatcher")

	// Close router first
	if err := d.router.Close(); err != nil {
		d.logger.Error("Failed to close router", "error", err)
	}

	// Close platform manager
	return d.platformManager.Close()
}
