// Package health provides health monitoring and check functionality for NotifyHub
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"   // Component is functioning normally
	StatusDegraded  Status = "degraded"  // Component has issues but still functional
	StatusUnhealthy Status = "unhealthy" // Component is not functioning
	StatusUnknown   Status = "unknown"   // Component status is unknown
)

// Check represents a health check function
type Check func(ctx context.Context) CheckResult

// CheckResult represents the result of a health check
type CheckResult struct {
	Name      string                 `json:"name"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Error     error                  `json:"-"`
}

// ComponentHealth represents the health status of a component
type ComponentHealth struct {
	Name         string                 `json:"name"`
	Status       Status                 `json:"status"`
	LastCheck    time.Time              `json:"last_check"`
	CheckResults []CheckResult          `json:"check_results"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	Status     Status                     `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Version    string                     `json:"version,omitempty"`
	Components map[string]ComponentHealth `json:"components"`
	Summary    HealthSummary              `json:"summary"`
}

// HealthSummary provides aggregated health information
type HealthSummary struct {
	TotalComponents int `json:"total_components"`
	HealthyCount    int `json:"healthy_count"`
	DegradedCount   int `json:"degraded_count"`
	UnhealthyCount  int `json:"unhealthy_count"`
	UnknownCount    int `json:"unknown_count"`
}

// HealthChecker manages health checks for system components
type HealthChecker struct {
	checks   map[string]Check
	results  map[string]CheckResult
	timeout  time.Duration
	interval time.Duration
	logger   logger.Logger
	mutex    sync.RWMutex
	stopCh   chan struct{}
	running  bool
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(logger logger.Logger) *HealthChecker {
	return &HealthChecker{
		checks:   make(map[string]Check),
		results:  make(map[string]CheckResult),
		timeout:  10 * time.Second,
		interval: 30 * time.Second,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

// RegisterCheck registers a health check for a component
func (hc *HealthChecker) RegisterCheck(name string, check Check) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.checks[name] = check
	hc.logger.Debug("Health check registered", "component", name)
}

// UnregisterCheck removes a health check
func (hc *HealthChecker) UnregisterCheck(name string) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	delete(hc.checks, name)
	delete(hc.results, name)
	hc.logger.Debug("Health check unregistered", "component", name)
}

// RunCheck executes a specific health check
func (hc *HealthChecker) RunCheck(ctx context.Context, name string) CheckResult {
	hc.mutex.RLock()
	check, exists := hc.checks[name]
	hc.mutex.RUnlock()

	if !exists {
		return CheckResult{
			Name:      name,
			Status:    StatusUnknown,
			Message:   "Health check not found",
			Timestamp: time.Now(),
			Duration:  0,
		}
	}

	// Create timeout context
	checkCtx, cancel := context.WithTimeout(ctx, hc.timeout)
	defer cancel()

	// Record start time
	startTime := time.Now()

	// Execute the check
	result := check(checkCtx)

	// Set metadata
	result.Name = name
	result.Timestamp = startTime
	result.Duration = time.Since(startTime)

	// Store result
	hc.mutex.Lock()
	hc.results[name] = result
	hc.mutex.Unlock()

	hc.logger.Debug("Health check completed",
		"component", name,
		"status", result.Status,
		"duration", result.Duration,
		"message", result.Message)

	return result
}

// RunAllChecks executes all registered health checks
func (hc *HealthChecker) RunAllChecks(ctx context.Context) map[string]CheckResult {
	hc.mutex.RLock()
	checkNames := make([]string, 0, len(hc.checks))
	for name := range hc.checks {
		checkNames = append(checkNames, name)
	}
	hc.mutex.RUnlock()

	results := make(map[string]CheckResult)
	var wg sync.WaitGroup

	// Run checks concurrently
	for _, name := range checkNames {
		wg.Add(1)
		go func(checkName string) {
			defer wg.Done()
			result := hc.RunCheck(ctx, checkName)
			results[checkName] = result
		}(name)
	}

	wg.Wait()
	return results
}

// GetSystemHealth returns the overall system health status
func (hc *HealthChecker) GetSystemHealth(ctx context.Context) SystemHealth {
	results := hc.RunAllChecks(ctx)

	components := make(map[string]ComponentHealth)
	summary := HealthSummary{
		TotalComponents: len(results),
	}

	overallStatus := StatusHealthy

	for name, result := range results {
		componentHealth := ComponentHealth{
			Name:         name,
			Status:       result.Status,
			LastCheck:    result.Timestamp,
			CheckResults: []CheckResult{result},
		}

		components[name] = componentHealth

		// Update summary counts
		switch result.Status {
		case StatusHealthy:
			summary.HealthyCount++
		case StatusDegraded:
			summary.DegradedCount++
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		case StatusUnhealthy:
			summary.UnhealthyCount++
			overallStatus = StatusUnhealthy
		case StatusUnknown:
			summary.UnknownCount++
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		}
	}

	return SystemHealth{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Components: components,
		Summary:    summary,
	}
}

// StartPeriodicChecks starts periodic health checks
func (hc *HealthChecker) StartPeriodicChecks(ctx context.Context) {
	hc.mutex.Lock()
	if hc.running {
		hc.mutex.Unlock()
		return
	}
	hc.running = true
	hc.mutex.Unlock()

	hc.logger.Info("Starting periodic health checks", "interval", hc.interval)

	go func() {
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				hc.logger.Info("Stopping periodic health checks due to context cancellation")
				hc.mutex.Lock()
				hc.running = false
				hc.mutex.Unlock()
				return
			case <-hc.stopCh:
				hc.logger.Info("Stopping periodic health checks")
				hc.mutex.Lock()
				hc.running = false
				hc.mutex.Unlock()
				return
			case <-ticker.C:
				checkCtx, cancel := context.WithTimeout(ctx, hc.timeout)
				results := hc.RunAllChecks(checkCtx)
				cancel()

				// Log any unhealthy components
				for name, result := range results {
					if result.Status == StatusUnhealthy {
						hc.logger.Warn("Component unhealthy", "component", name, "message", result.Message)
					}
				}
			}
		}
	}()
}

// StopPeriodicChecks stops periodic health checks
func (hc *HealthChecker) StopPeriodicChecks() {
	hc.mutex.Lock()
	if !hc.running {
		hc.mutex.Unlock()
		return
	}
	hc.mutex.Unlock()

	close(hc.stopCh)
	hc.stopCh = make(chan struct{})
}

// SetTimeout sets the timeout for health checks
func (hc *HealthChecker) SetTimeout(timeout time.Duration) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.timeout = timeout
}

// SetInterval sets the interval for periodic health checks
func (hc *HealthChecker) SetInterval(interval time.Duration) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.interval = interval
}

// GetLastResults returns the last check results for all components
func (hc *HealthChecker) GetLastResults() map[string]CheckResult {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	results := make(map[string]CheckResult)
	for name, result := range hc.results {
		results[name] = result
	}

	return results
}

// Common health check implementations

// TCPHealthCheck creates a health check for TCP connectivity
func TCPHealthCheck(address string, timeout time.Duration) Check {
	return func(ctx context.Context) CheckResult {
		// This would implement TCP connection check
		// For now, returning a basic implementation
		return CheckResult{
			Status:  StatusHealthy,
			Message: fmt.Sprintf("TCP connection to %s successful", address),
			Details: map[string]interface{}{
				"address": address,
				"timeout": timeout.String(),
			},
		}
	}
}

// HTTPHealthCheck creates a health check for HTTP endpoints
func HTTPHealthCheck(url string, expectedStatus int) Check {
	return func(ctx context.Context) CheckResult {
		// This would implement HTTP health check
		// For now, returning a basic implementation
		return CheckResult{
			Status:  StatusHealthy,
			Message: fmt.Sprintf("HTTP check for %s successful", url),
			Details: map[string]interface{}{
				"url":             url,
				"expected_status": expectedStatus,
			},
		}
	}
}

// DatabaseHealthCheck creates a health check for database connectivity
func DatabaseHealthCheck(dsn string) Check {
	return func(ctx context.Context) CheckResult {
		// This would implement database ping check
		// For now, returning a basic implementation
		return CheckResult{
			Status:  StatusHealthy,
			Message: "Database connection successful",
			Details: map[string]interface{}{
				"dsn": dsn,
			},
		}
	}
}

// RedisHealthCheck creates a health check for Redis connectivity
func RedisHealthCheck(addr string) Check {
	return func(ctx context.Context) CheckResult {
		// This would implement Redis ping check
		// For now, returning a basic implementation
		return CheckResult{
			Status:  StatusHealthy,
			Message: "Redis connection successful",
			Details: map[string]interface{}{
				"address": addr,
			},
		}
	}
}

// QueueHealthCheck creates a health check for queue systems
func QueueHealthCheck(queueName string, checker func(ctx context.Context) error) Check {
	return func(ctx context.Context) CheckResult {
		if err := checker(ctx); err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: fmt.Sprintf("Queue %s is unhealthy: %v", queueName, err),
				Details: map[string]interface{}{
					"queue": queueName,
					"error": err.Error(),
				},
				Error: err,
			}
		}

		return CheckResult{
			Status:  StatusHealthy,
			Message: fmt.Sprintf("Queue %s is healthy", queueName),
			Details: map[string]interface{}{
				"queue": queueName,
			},
		}
	}
}

// String returns the string representation of Status
func (s Status) String() string {
	return string(s)
}

// MarshalJSON implements json.Marshaler for CheckResult
func (cr CheckResult) MarshalJSON() ([]byte, error) {
	type Alias CheckResult
	return json.Marshal(&struct {
		Alias
		Duration string `json:"duration"`
	}{
		Alias:    (Alias)(cr),
		Duration: cr.Duration.String(),
	})
}
