// Package target provides intelligent routing functionality for NotifyHub
package target

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// RouterConfig defines configuration for the target router
type RouterConfig struct {
	// LoadBalancing strategy: "round_robin", "random", "weighted"
	LoadBalancing string `json:"load_balancing"`

	// HealthCheckInterval for checking platform health
	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// MaxRetries for failover attempts
	MaxRetries int `json:"max_retries"`

	// RetryDelay between failover attempts
	RetryDelay time.Duration `json:"retry_delay"`
}

// PlatformHealth represents the health status of a platform
type PlatformHealth struct {
	Platform     string        `json:"platform"`
	Healthy      bool          `json:"healthy"`
	LastChecked  time.Time     `json:"last_checked"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorCount   int           `json:"error_count"`
	Weight       int           `json:"weight"` // For weighted load balancing
}

// RoutingRule defines how targets should be routed to platforms
type RoutingRule struct {
	TargetType        string   `json:"target_type"`        // email, phone, user, etc.
	PrimaryPlatforms  []string `json:"primary_platforms"`  // Preferred platforms in order
	FallbackPlatforms []string `json:"fallback_platforms"` // Backup platforms
	LoadBalancing     string   `json:"load_balancing"`     // Override global load balancing
}

// Router provides intelligent routing with load balancing and fault tolerance
type Router interface {
	// AddRule adds a routing rule for a target type
	AddRule(rule RoutingRule)

	// RouteTargets routes targets to appropriate platforms with load balancing
	RouteTargets(targets []Target) (map[string][]Target, error)

	// UpdatePlatformHealth updates health status for a platform
	UpdatePlatformHealth(platform string, healthy bool, responseTime time.Duration)

	// GetPlatformHealth returns current health status of all platforms
	GetPlatformHealth() map[string]PlatformHealth

	// SetPlatformWeight sets weight for weighted load balancing
	SetPlatformWeight(platform string, weight int)

	// Close gracefully shuts down the router
	Close() error
}

// SmartRouter implements the Router interface with advanced routing capabilities
type SmartRouter struct {
	config          RouterConfig
	rules           map[string]RoutingRule // target type -> routing rule
	platformHealth  map[string]*PlatformHealth
	roundRobinIndex map[string]int // platform -> current index for round robin
	logger          logger.Logger
	mutex           sync.RWMutex
	healthTicker    *time.Ticker
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// NewSmartRouter creates a new smart router with load balancing and fault tolerance
func NewSmartRouter(config RouterConfig, logger logger.Logger) Router {
	if config.LoadBalancing == "" {
		config.LoadBalancing = "round_robin"
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	router := &SmartRouter{
		config:          config,
		rules:           make(map[string]RoutingRule),
		platformHealth:  make(map[string]*PlatformHealth),
		roundRobinIndex: make(map[string]int),
		logger:          logger,
		stopCh:          make(chan struct{}),
	}

	// Add default routing rules
	router.addDefaultRules()

	// Start health monitoring if interval is set
	if config.HealthCheckInterval > 0 {
		router.startHealthMonitoring()
	}

	return router
}

// AddRule adds a routing rule for a target type
func (r *SmartRouter) AddRule(rule RoutingRule) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.rules[rule.TargetType] = rule
	r.logger.Debug("Routing rule added", "target_type", rule.TargetType, "primary_platforms", rule.PrimaryPlatforms)
}

// RouteTargets routes targets to appropriate platforms with load balancing
func (r *SmartRouter) RouteTargets(targets []Target) (map[string][]Target, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	platformTargets := make(map[string][]Target)

	for _, target := range targets {
		platform, err := r.selectPlatform(target)
		if err != nil {
			r.logger.Error("Failed to select platform for target", "target", target, "error", err)
			continue
		}

		platformTargets[platform] = append(platformTargets[platform], target)
	}

	return platformTargets, nil
}

// selectPlatform selects the best platform for a target using routing rules and load balancing
func (r *SmartRouter) selectPlatform(target Target) (string, error) {
	// If platform is explicitly specified and healthy, use it
	if target.Platform != "" && target.Platform != PlatformAuto {
		if r.isPlatformHealthy(target.Platform) {
			return target.Platform, nil
		}
	}

	// Auto-detect target type if not specified
	targetType := target.Type
	if targetType == "" {
		detectedTarget := DefaultResolver.AutoDetectTarget(target.Value)
		targetType = detectedTarget.Type
	}

	// Get routing rule for target type
	rule, exists := r.rules[targetType]
	if !exists {
		// Fallback to auto-detection
		detectedTarget := DefaultResolver.AutoDetectTarget(target.Value)
		return detectedTarget.Platform, nil
	}

	// Select from primary platforms using load balancing
	healthyPrimary := r.filterHealthyPlatforms(rule.PrimaryPlatforms)
	if len(healthyPrimary) > 0 {
		return r.selectWithLoadBalancing(healthyPrimary, rule.LoadBalancing), nil
	}

	// Fallback to fallback platforms
	healthyFallback := r.filterHealthyPlatforms(rule.FallbackPlatforms)
	if len(healthyFallback) > 0 {
		return r.selectWithLoadBalancing(healthyFallback, rule.LoadBalancing), nil
	}

	return "", fmt.Errorf("no healthy platforms available for target type: %s", targetType)
}

// filterHealthyPlatforms filters platforms by health status
func (r *SmartRouter) filterHealthyPlatforms(platforms []string) []string {
	var healthy []string
	for _, platform := range platforms {
		if r.isPlatformHealthy(platform) {
			healthy = append(healthy, platform)
		}
	}
	return healthy
}

// isPlatformHealthy checks if a platform is currently healthy
func (r *SmartRouter) isPlatformHealthy(platform string) bool {
	health, exists := r.platformHealth[platform]
	if !exists {
		// Unknown platform, assume healthy for now
		return true
	}
	return health.Healthy
}

// selectWithLoadBalancing selects a platform using the specified load balancing strategy
func (r *SmartRouter) selectWithLoadBalancing(platforms []string, strategy string) string {
	if len(platforms) == 0 {
		return ""
	}
	if len(platforms) == 1 {
		return platforms[0]
	}

	// Use rule-specific strategy if specified, otherwise use global config
	if strategy == "" {
		strategy = r.config.LoadBalancing
	}

	switch strategy {
	case "random":
		return platforms[rand.Intn(len(platforms))]

	case "weighted":
		return r.selectWeighted(platforms)

	case "round_robin":
		fallthrough
	default:
		// Round robin - get current index and increment
		key := fmt.Sprintf("%v", platforms) // Use platforms slice as key
		index := r.roundRobinIndex[key]
		selected := platforms[index%len(platforms)]
		r.roundRobinIndex[key] = index + 1
		return selected
	}
}

// selectWeighted selects a platform based on weight
func (r *SmartRouter) selectWeighted(platforms []string) string {
	// Calculate total weight
	totalWeight := 0
	weights := make([]int, len(platforms))

	for i, platform := range platforms {
		health := r.platformHealth[platform]
		weight := 1 // Default weight
		if health != nil {
			weight = health.Weight
		}
		weights[i] = weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		// No weights set, fallback to random
		return platforms[rand.Intn(len(platforms))]
	}

	// Select randomly based on weight
	random := rand.Intn(totalWeight)
	current := 0

	for i, weight := range weights {
		current += weight
		if random < current {
			return platforms[i]
		}
	}

	// Fallback (shouldn't reach here)
	return platforms[0]
}

// UpdatePlatformHealth updates health status for a platform
func (r *SmartRouter) UpdatePlatformHealth(platform string, healthy bool, responseTime time.Duration) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	health, exists := r.platformHealth[platform]
	if !exists {
		health = &PlatformHealth{
			Platform: platform,
			Weight:   1, // Default weight
		}
		r.platformHealth[platform] = health
	}

	oldHealthy := health.Healthy
	health.Healthy = healthy
	health.LastChecked = time.Now()
	health.ResponseTime = responseTime

	if healthy {
		health.ErrorCount = 0
	} else {
		health.ErrorCount++
	}

	// Log health changes
	if oldHealthy != healthy {
		r.logger.Info("Platform health changed",
			"platform", platform,
			"healthy", healthy,
			"response_time", responseTime,
			"error_count", health.ErrorCount)
	}
}

// GetPlatformHealth returns current health status of all platforms
func (r *SmartRouter) GetPlatformHealth() map[string]PlatformHealth {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]PlatformHealth)
	for platform, health := range r.platformHealth {
		result[platform] = *health
	}
	return result
}

// SetPlatformWeight sets weight for weighted load balancing
func (r *SmartRouter) SetPlatformWeight(platform string, weight int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	health, exists := r.platformHealth[platform]
	if !exists {
		health = &PlatformHealth{
			Platform: platform,
			Healthy:  true,
			Weight:   weight,
		}
		r.platformHealth[platform] = health
	} else {
		health.Weight = weight
	}

	r.logger.Debug("Platform weight updated", "platform", platform, "weight", weight)
}

// addDefaultRules adds default routing rules for common target types
func (r *SmartRouter) addDefaultRules() {
	// Email routing
	r.rules[TargetTypeEmail] = RoutingRule{
		TargetType:        TargetTypeEmail,
		PrimaryPlatforms:  []string{PlatformEmail},
		FallbackPlatforms: []string{}, // No fallback for email
	}

	// Phone routing
	r.rules[TargetTypePhone] = RoutingRule{
		TargetType:        TargetTypePhone,
		PrimaryPlatforms:  []string{PlatformSMS},
		FallbackPlatforms: []string{}, // No fallback for SMS
	}

	// User routing (prefer Feishu, fallback to email if available)
	r.rules[TargetTypeUser] = RoutingRule{
		TargetType:        TargetTypeUser,
		PrimaryPlatforms:  []string{PlatformFeishu},
		FallbackPlatforms: []string{PlatformEmail}, // If user has email
	}

	// Group routing
	r.rules[TargetTypeGroup] = RoutingRule{
		TargetType:        TargetTypeGroup,
		PrimaryPlatforms:  []string{PlatformFeishu},
		FallbackPlatforms: []string{},
	}

	// Channel routing
	r.rules[TargetTypeChannel] = RoutingRule{
		TargetType:        TargetTypeChannel,
		PrimaryPlatforms:  []string{PlatformFeishu},
		FallbackPlatforms: []string{},
	}

	// Webhook routing
	r.rules[TargetTypeWebhook] = RoutingRule{
		TargetType:        TargetTypeWebhook,
		PrimaryPlatforms:  []string{PlatformWebhook},
		FallbackPlatforms: []string{},
	}
}

// startHealthMonitoring starts background health monitoring
func (r *SmartRouter) startHealthMonitoring() {
	r.healthTicker = time.NewTicker(r.config.HealthCheckInterval)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-r.healthTicker.C:
				r.performHealthChecks()
			case <-r.stopCh:
				return
			}
		}
	}()

	r.logger.Debug("Health monitoring started", "interval", r.config.HealthCheckInterval)
}

// performHealthChecks performs health checks on all known platforms
func (r *SmartRouter) performHealthChecks() {
	r.mutex.RLock()
	platforms := make([]string, 0, len(r.platformHealth))
	for platform := range r.platformHealth {
		platforms = append(platforms, platform)
	}
	r.mutex.RUnlock()

	// TODO: Integrate with actual platform health checks
	// For now, we'll just mark stale entries as potentially unhealthy
	for _, platform := range platforms {
		r.mutex.RLock()
		health := r.platformHealth[platform]
		timeSinceLastCheck := time.Since(health.LastChecked)
		r.mutex.RUnlock()

		// Mark as potentially unhealthy if no updates for too long
		if timeSinceLastCheck > r.config.HealthCheckInterval*3 {
			r.logger.Warn("Platform health check stale", "platform", platform, "last_checked", health.LastChecked)
			// Could update health status here based on actual health check
		}
	}
}

// Close gracefully shuts down the router
func (r *SmartRouter) Close() error {
	r.logger.Debug("Shutting down smart router")

	// Stop health monitoring
	if r.healthTicker != nil {
		r.healthTicker.Stop()
	}

	// Signal shutdown
	close(r.stopCh)

	// Wait for background goroutines
	r.wg.Wait()

	r.logger.Debug("Smart router shut down complete")
	return nil
}
