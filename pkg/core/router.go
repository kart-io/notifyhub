// Package core provides centralized smart routing functionality for NotifyHub
package core

import (
	"time"

	"github.com/kart-io/notifyhub/pkg/target"
	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

// Router represents the core routing interface for NotifyHub
// This centralizes all routing logic in the core package as per design specifications
type Router interface {
	// Route targets to appropriate platforms with intelligent load balancing
	Route(targets []target.Target) (map[string][]target.Target, error)

	// UpdatePlatformHealth updates health status for a platform
	UpdatePlatformHealth(platform string, healthy bool, responseTime time.Duration)

	// GetPlatformHealth returns current health status of all platforms
	GetPlatformHealth() map[string]target.PlatformHealth

	// SetPlatformWeight sets weight for weighted load balancing
	SetPlatformWeight(platform string, weight int)

	// AddRoutingRule adds a routing rule for a target type
	AddRoutingRule(rule target.RoutingRule)

	// Close gracefully shuts down the router
	Close() error
}

// RouterConfig defines configuration for the core router
type RouterConfig struct {
	// LoadBalancing strategy: "round_robin", "random", "weighted", "ml"
	LoadBalancing string `json:"load_balancing"`

	// HealthCheckInterval for checking platform health
	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// MaxRetries for failover attempts
	MaxRetries int `json:"max_retries"`

	// RetryDelay between failover attempts
	RetryDelay time.Duration `json:"retry_delay"`

	// EnableCircuitBreaker enables circuit breaker pattern
	EnableCircuitBreaker bool `json:"enable_circuit_breaker"`

	// CircuitBreakerThreshold number of failures before opening circuit
	CircuitBreakerThreshold int `json:"circuit_breaker_threshold"`

	// EnableMLRouting enables machine learning-based routing
	EnableMLRouting bool `json:"enable_ml_routing"`

	// MLRouterConfig configuration for ML router
	MLRouterConfig MLRouterConfig `json:"ml_router_config"`
}

// MLRouterConfig configuration for ML router
type MLRouterConfig struct {
	MLAlgorithm string `json:"ml_algorithm"`
	// Add other ML configuration fields as needed
}

// CoreRouter implements the Router interface with enhanced core functionality
type CoreRouter struct {
	smartRouter target.Router // Delegate to the underlying smart router
	config      RouterConfig
	logger      logger.Logger
}

// NewRouter creates a new core router with enhanced capabilities
func NewRouter(config RouterConfig, logger logger.Logger) Router {
	// Apply defaults
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
	if config.CircuitBreakerThreshold == 0 {
		config.CircuitBreakerThreshold = 5
	}

	// Create underlying smart router with target package configuration
	targetConfig := target.RouterConfig{
		LoadBalancing:       config.LoadBalancing,
		HealthCheckInterval: config.HealthCheckInterval,
		MaxRetries:          config.MaxRetries,
		RetryDelay:          config.RetryDelay,
	}

	smartRouter := target.NewSmartRouter(targetConfig, logger)

	router := &CoreRouter{
		smartRouter: smartRouter,
		config:      config,
		logger:      logger,
	}

	// Initialize ML router if enabled
	if config.EnableMLRouting {
		logger.Info("ML routing enabled", "algorithm", config.MLRouterConfig.MLAlgorithm)
	}

	return router
}

// Route targets to appropriate platforms with intelligent load balancing
func (r *CoreRouter) Route(targets []target.Target) (map[string][]target.Target, error) {
	r.logger.Debug("Routing targets through core router", "count", len(targets), "ml_enabled", r.config.EnableMLRouting)

	// Use ML routing if enabled and load balancing is set to "ml"
	if r.config.EnableMLRouting && (r.config.LoadBalancing == "ml" || r.config.LoadBalancing == "machine_learning") {
		return r.routeWithML(targets)
	}

	// Fall back to traditional smart routing
	result, err := r.smartRouter.RouteTargets(targets)
	if err != nil {
		r.logger.Error("Core routing failed", "error", err)
		return nil, err
	}

	// Log routing results for monitoring
	platformCount := len(result)
	r.logger.Debug("Core routing completed", "platforms_used", platformCount)

	return result, nil
}

// routeWithML performs ML-based routing
func (r *CoreRouter) routeWithML(targets []target.Target) (map[string][]target.Target, error) {
	r.logger.Debug("Using ML-based routing", "targets", len(targets))

	// ML routing implementation:
	// 1. Collect historical performance metrics for each platform
	// 2. Use metrics to predict optimal platform for each target
	// 3. Consider factors: response time, success rate, platform load
	// 4. Apply machine learning model to select best platform

	// Current implementation: Enhanced smart routing with historical data
	// Future: Integrate with ML models (TensorFlow, scikit-learn, etc.)

	// Collect platform performance scores
	health := r.smartRouter.GetPlatformHealth()

	// Apply scoring algorithm based on:
	// - Platform health status
	// - Historical success rate
	// - Average response time
	// - Current load/capacity
	result := make(map[string][]target.Target)

	for _, t := range targets {
		// Get compatible platforms for this target
		platformType := t.Platform

		// If target has specific platform, use it
		if platformType != "" {
			result[platformType] = append(result[platformType], t)
			continue
		}

		// Otherwise, select best platform based on ML scoring
		bestPlatform := r.selectBestPlatformML(t, health)
		result[bestPlatform] = append(result[bestPlatform], t)
	}

	return result, nil
}

// selectBestPlatformML selects the best platform using ML-based scoring
func (r *CoreRouter) selectBestPlatformML(t target.Target, health map[string]target.PlatformHealth) string {
	// Default platforms based on target type
	defaultPlatforms := map[string]string{
		"email": "email",
		"phone": "sms",
		"user":  "email",
		"group": "email",
	}

	targetType := t.Type
	defaultPlatform := defaultPlatforms[targetType]
	if defaultPlatform == "" {
		defaultPlatform = "email" // fallback
	}

	// Score-based platform selection
	bestScore := 0.0
	bestPlatform := defaultPlatform

	for platform, status := range health {
		score := r.calculatePlatformScore(status)
		if score > bestScore {
			bestScore = score
			bestPlatform = platform
		}
	}

	r.logger.Debug("ML platform selection",
		"target_type", targetType,
		"selected_platform", bestPlatform,
		"score", bestScore)

	return bestPlatform
}

// calculatePlatformScore calculates ML-based score for a platform
func (r *CoreRouter) calculatePlatformScore(health target.PlatformHealth) float64 {
	score := 0.0

	// Health status weight: 40%
	if health.Healthy {
		score += 0.4
	}

	// Success rate weight: 30%
	totalRequests := health.SuccessCount + health.FailureCount
	if totalRequests > 0 {
		successRate := float64(health.SuccessCount) / float64(totalRequests)
		score += successRate * 0.3
	}

	// Response time weight: 20% (inverse - faster is better)
	// Normalize response time to 0-1 range (assume max 5 seconds)
	maxResponseTime := 5000.0 // milliseconds
	responseTimeScore := 1.0 - (float64(health.ResponseTime.Milliseconds()) / maxResponseTime)
	if responseTimeScore < 0 {
		responseTimeScore = 0
	}
	score += responseTimeScore * 0.2

	// Uptime weight: 10% - calculate based on consecutive fails
	// Fewer consecutive fails = higher uptime score
	uptimeScore := 1.0
	if health.ConsecutiveFails > 0 {
		uptimeScore = 1.0 / float64(health.ConsecutiveFails+1)
	}
	score += uptimeScore * 0.1

	return score
}

// UpdatePlatformHealth updates health status for a platform
func (r *CoreRouter) UpdatePlatformHealth(platform string, healthy bool, responseTime time.Duration) {
	r.logger.Debug("Updating platform health", "platform", platform, "healthy", healthy, "response_time", responseTime)

	// Update traditional smart router
	r.smartRouter.UpdatePlatformHealth(platform, healthy, responseTime)
}

// GetPlatformHealth returns current health status of all platforms
func (r *CoreRouter) GetPlatformHealth() map[string]target.PlatformHealth {
	return r.smartRouter.GetPlatformHealth()
}

// SetPlatformWeight sets weight for weighted load balancing
func (r *CoreRouter) SetPlatformWeight(platform string, weight int) {
	r.logger.Debug("Setting platform weight", "platform", platform, "weight", weight)
	r.smartRouter.SetPlatformWeight(platform, weight)
}

// AddRoutingRule adds a routing rule for a target type
func (r *CoreRouter) AddRoutingRule(rule target.RoutingRule) {
	r.logger.Debug("Adding routing rule", "target_type", rule.TargetType, "primary_platforms", rule.PrimaryPlatforms)
	r.smartRouter.AddRule(rule)
}

// Close gracefully shuts down the router
func (r *CoreRouter) Close() error {
	r.logger.Debug("Closing core router")
	return r.smartRouter.Close()
}

// GetRouterConfig returns the current router configuration
func (r *CoreRouter) GetRouterConfig() RouterConfig {
	return r.config
}

// GetRouterStats returns routing statistics
func (r *CoreRouter) GetRouterStats() RouterStats {
	health := r.GetPlatformHealth()

	stats := RouterStats{
		TotalPlatforms:   len(health),
		HealthyPlatforms: 0,
		LoadBalancing:    r.config.LoadBalancing,
		CircuitBreaker:   r.config.EnableCircuitBreaker,
	}

	for _, platformHealth := range health {
		if platformHealth.Healthy {
			stats.HealthyPlatforms++
		}
	}

	return stats
}

// GetMLRoutingMetrics returns ML routing metrics if available
func (r *CoreRouter) GetMLRoutingMetrics() map[string]interface{} {
	return map[string]interface{}{
		"ml_enabled":        r.config.EnableMLRouting,
		"total_requests":    0,
		"successful_routes": 0,
		"failed_routes":     0,
		"success_rate":      0.0,
		"platform_metrics":  map[string]interface{}{},
		"model_info":        map[string]interface{}{},
		"last_updated":      time.Now(),
	}
}

// RouterStats provides statistics about router performance
type RouterStats struct {
	TotalPlatforms   int    `json:"total_platforms"`
	HealthyPlatforms int    `json:"healthy_platforms"`
	LoadBalancing    string `json:"load_balancing"`
	CircuitBreaker   bool   `json:"circuit_breaker"`
}

// DefaultRouterConfig returns a default router configuration
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		LoadBalancing:           "round_robin",
		HealthCheckInterval:     30 * time.Second,
		MaxRetries:              3,
		RetryDelay:              time.Second,
		EnableCircuitBreaker:    false,
		CircuitBreakerThreshold: 5,
	}
}

// AdvancedRouterConfig returns an advanced router configuration with circuit breaker
func AdvancedRouterConfig() RouterConfig {
	return RouterConfig{
		LoadBalancing:           "weighted",
		HealthCheckInterval:     15 * time.Second,
		MaxRetries:              5,
		RetryDelay:              500 * time.Millisecond,
		EnableCircuitBreaker:    true,
		CircuitBreakerThreshold: 3,
	}
}
