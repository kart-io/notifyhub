// Package target provides target routing functionality for NotifyHub
package target

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

// Router routes targets to appropriate platforms with load balancing
type Router interface {
	// RouteTargets routes targets to platforms with load balancing
	RouteTargets(targets []Target) (map[string][]Target, error)

	// UpdatePlatformHealth updates health status for a platform
	UpdatePlatformHealth(platform string, healthy bool, responseTime time.Duration)

	// GetPlatformHealth returns current health status of all platforms
	GetPlatformHealth() map[string]PlatformHealth

	// SetPlatformWeight sets weight for weighted load balancing
	SetPlatformWeight(platform string, weight int)

	// AddRule adds a routing rule
	AddRule(rule RoutingRule)

	// RemoveRule removes a routing rule
	RemoveRule(ruleID string)

	// Close gracefully shuts down the router
	Close() error
}

// RouterConfig configures the target router
type RouterConfig struct {
	LoadBalancing       string        `json:"load_balancing"` // "round_robin", "random", "weighted", "health"
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay"`
}

// PlatformHealth represents health status of a platform
type PlatformHealth struct {
	Healthy          bool          `json:"healthy"`
	LastChecked      time.Time     `json:"last_checked"`
	ResponseTime     time.Duration `json:"response_time"`
	FailureCount     int           `json:"failure_count"`
	SuccessCount     int           `json:"success_count"`
	ConsecutiveFails int           `json:"consecutive_fails"`
}

// RoutingRule defines routing rules for specific target types
type RoutingRule struct {
	ID                string                 `json:"id"`
	TargetType        string                 `json:"target_type"`
	PrimaryPlatforms  []string               `json:"primary_platforms"`
	FallbackPlatforms []string               `json:"fallback_platforms"`
	Priority          int                    `json:"priority"`
	Conditions        map[string]interface{} `json:"conditions,omitempty"`
}

// SmartRouter implements intelligent target routing
type SmartRouter struct {
	config          RouterConfig
	platformHealth  map[string]*PlatformHealth
	platformWeights map[string]int
	rules           []RoutingRule
	roundRobinIndex map[string]int
	logger          logger.Logger
	mutex           sync.RWMutex
}

// NewSmartRouter creates a new smart router
func NewSmartRouter(config RouterConfig, logger logger.Logger) *SmartRouter {
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

	return &SmartRouter{
		config:          config,
		platformHealth:  make(map[string]*PlatformHealth),
		platformWeights: make(map[string]int),
		rules:           make([]RoutingRule, 0),
		roundRobinIndex: make(map[string]int),
		logger:          logger,
	}
}

// RouteTargets routes targets to platforms with load balancing
func (r *SmartRouter) RouteTargets(targets []Target) (map[string][]Target, error) {
	r.logger.Debug("Routing targets", "count", len(targets), "strategy", r.config.LoadBalancing)

	result := make(map[string][]Target)

	// Group targets by type for rule application
	targetsByType := r.groupTargetsByType(targets)

	for targetType, typeTargets := range targetsByType {
		// Apply routing rules
		platforms := r.getPlatformsForTargetType(targetType)

		// Apply load balancing
		platformTargets := r.distributeTargets(typeTargets, platforms)

		// Merge results
		for platform, platformTargetList := range platformTargets {
			result[platform] = append(result[platform], platformTargetList...)
		}
	}

	r.logger.Debug("Targets routed", "platforms", len(result))
	return result, nil
}

// groupTargetsByType groups targets by their type
func (r *SmartRouter) groupTargetsByType(targets []Target) map[string][]Target {
	groups := make(map[string][]Target)
	for _, target := range targets {
		groups[target.Type] = append(groups[target.Type], target)
	}
	return groups
}

// getPlatformsForTargetType gets available platforms for a target type
func (r *SmartRouter) getPlatformsForTargetType(targetType string) []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Find matching rules
	var applicableRules []RoutingRule
	for _, rule := range r.rules {
		if rule.TargetType == targetType || rule.TargetType == "*" {
			applicableRules = append(applicableRules, rule)
		}
	}

	// Sort rules by priority
	sort.Slice(applicableRules, func(i, j int) bool {
		return applicableRules[i].Priority > applicableRules[j].Priority
	})

	// Get platforms from highest priority rule
	if len(applicableRules) > 0 {
		rule := applicableRules[0]
		platforms := make([]string, 0)

		// Add primary platforms if healthy
		for _, platform := range rule.PrimaryPlatforms {
			if r.isPlatformHealthy(platform) {
				platforms = append(platforms, platform)
			}
		}

		// Add fallback platforms if no primary platforms available
		if len(platforms) == 0 {
			for _, platform := range rule.FallbackPlatforms {
				if r.isPlatformHealthy(platform) {
					platforms = append(platforms, platform)
				}
			}
		}

		if len(platforms) > 0 {
			return platforms
		}
	}

	// Default: use platform from target if specified
	return []string{"default"}
}

// distributeTargets distributes targets across platforms using load balancing
func (r *SmartRouter) distributeTargets(targets []Target, platforms []string) map[string][]Target {
	if len(platforms) == 0 {
		return nil
	}

	switch r.config.LoadBalancing {
	case "round_robin":
		return r.roundRobinDistribution(targets, platforms)
	case "random":
		return r.randomDistribution(targets, platforms)
	case "weighted":
		return r.weightedDistribution(targets, platforms)
	case "health":
		return r.healthBasedDistribution(targets, platforms)
	default:
		return r.roundRobinDistribution(targets, platforms)
	}
}

// roundRobinDistribution distributes targets using round-robin
func (r *SmartRouter) roundRobinDistribution(targets []Target, platforms []string) map[string][]Target {
	result := make(map[string][]Target)
	platformKey := fmt.Sprintf("%v", platforms)

	for i, target := range targets {
		platformIndex := (r.roundRobinIndex[platformKey] + i) % len(platforms)
		platform := platforms[platformIndex]
		result[platform] = append(result[platform], target)
	}

	// Update round-robin index
	r.mutex.Lock()
	r.roundRobinIndex[platformKey] = (r.roundRobinIndex[platformKey] + len(targets)) % len(platforms)
	r.mutex.Unlock()

	return result
}

// randomDistribution distributes targets randomly
func (r *SmartRouter) randomDistribution(targets []Target, platforms []string) map[string][]Target {
	result := make(map[string][]Target)

	for _, target := range targets {
		platform := platforms[rand.Intn(len(platforms))]
		result[platform] = append(result[platform], target)
	}

	return result
}

// weightedDistribution distributes targets based on platform weights
func (r *SmartRouter) weightedDistribution(targets []Target, platforms []string) map[string][]Target {
	result := make(map[string][]Target)

	// Calculate total weight
	totalWeight := 0
	for _, platform := range platforms {
		weight := r.getPlatformWeight(platform)
		totalWeight += weight
	}

	if totalWeight == 0 {
		return r.roundRobinDistribution(targets, platforms)
	}

	// Create weighted platform list
	weightedPlatforms := make([]string, 0)
	for _, platform := range platforms {
		weight := r.getPlatformWeight(platform)
		for i := 0; i < weight; i++ {
			weightedPlatforms = append(weightedPlatforms, platform)
		}
	}

	// Distribute using weighted list
	for i, target := range targets {
		platformIndex := i % len(weightedPlatforms)
		platform := weightedPlatforms[platformIndex]
		result[platform] = append(result[platform], target)
	}

	return result
}

// healthBasedDistribution distributes targets based on platform health
func (r *SmartRouter) healthBasedDistribution(targets []Target, platforms []string) map[string][]Target {
	// Filter healthy platforms
	healthyPlatforms := make([]string, 0)
	for _, platform := range platforms {
		if r.isPlatformHealthy(platform) {
			healthyPlatforms = append(healthyPlatforms, platform)
		}
	}

	if len(healthyPlatforms) == 0 {
		// Use all platforms if none are healthy
		healthyPlatforms = platforms
	}

	return r.roundRobinDistribution(targets, healthyPlatforms)
}

// UpdatePlatformHealth updates health status for a platform
func (r *SmartRouter) UpdatePlatformHealth(platform string, healthy bool, responseTime time.Duration) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	health, exists := r.platformHealth[platform]
	if !exists {
		health = &PlatformHealth{}
		r.platformHealth[platform] = health
	}

	health.Healthy = healthy
	health.LastChecked = time.Now()
	health.ResponseTime = responseTime

	if healthy {
		health.SuccessCount++
		health.ConsecutiveFails = 0
	} else {
		health.FailureCount++
		health.ConsecutiveFails++
	}

	r.logger.Debug("Platform health updated", "platform", platform, "healthy", healthy, "response_time", responseTime)
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

	r.platformWeights[platform] = weight
	r.logger.Debug("Platform weight updated", "platform", platform, "weight", weight)
}

// AddRule adds a routing rule
func (r *SmartRouter) AddRule(rule RoutingRule) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.rules = append(r.rules, rule)
	r.logger.Debug("Routing rule added", "rule_id", rule.ID, "target_type", rule.TargetType)
}

// RemoveRule removes a routing rule
func (r *SmartRouter) RemoveRule(ruleID string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, rule := range r.rules {
		if rule.ID == ruleID {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			r.logger.Debug("Routing rule removed", "rule_id", ruleID)
			break
		}
	}
}

// Close gracefully shuts down the router
func (r *SmartRouter) Close() error {
	r.logger.Debug("Smart router closing")
	return nil
}

// Helper methods

// isPlatformHealthy checks if a platform is healthy
func (r *SmartRouter) isPlatformHealthy(platform string) bool {
	health, exists := r.platformHealth[platform]
	if !exists {
		return true // Assume healthy if no health data
	}

	// Consider unhealthy if too many consecutive failures
	if health.ConsecutiveFails >= 3 {
		return false
	}

	return health.Healthy
}

// getPlatformWeight gets the weight for a platform
func (r *SmartRouter) getPlatformWeight(platform string) int {
	weight, exists := r.platformWeights[platform]
	if !exists {
		return 1 // Default weight
	}
	return weight
}

// Convenience functions

// CreateEmailRule creates a routing rule for email targets
func CreateEmailRule(primaryPlatforms, fallbackPlatforms []string) RoutingRule {
	return RoutingRule{
		ID:                fmt.Sprintf("email-rule-%d", time.Now().Unix()),
		TargetType:        "email",
		PrimaryPlatforms:  primaryPlatforms,
		FallbackPlatforms: fallbackPlatforms,
		Priority:          100,
	}
}

// CreateWebhookRule creates a routing rule for webhook targets
func CreateWebhookRule(primaryPlatforms, fallbackPlatforms []string) RoutingRule {
	return RoutingRule{
		ID:                fmt.Sprintf("webhook-rule-%d", time.Now().Unix()),
		TargetType:        "webhook",
		PrimaryPlatforms:  primaryPlatforms,
		FallbackPlatforms: fallbackPlatforms,
		Priority:          100,
	}
}

// CreateDefaultRule creates a default routing rule for all target types
func CreateDefaultRule(primaryPlatforms, fallbackPlatforms []string) RoutingRule {
	return RoutingRule{
		ID:                fmt.Sprintf("default-rule-%d", time.Now().Unix()),
		TargetType:        "*",
		PrimaryPlatforms:  primaryPlatforms,
		FallbackPlatforms: fallbackPlatforms,
		Priority:          1,
	}
}
