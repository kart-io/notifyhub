// Package health provides health monitoring integration for NotifyHub components
package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/errors"
)

// Monitor provides centralized health monitoring for NotifyHub
type Monitor struct {
	checker    *HealthChecker
	logger     logger.Logger
	components map[string]MonitoredComponent
	alerts     []AlertRule
	mutex      sync.RWMutex
	running    bool
	stopCh     chan struct{}
}

// MonitoredComponent represents a component being monitored
type MonitoredComponent struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // "platform", "queue", "template", "database", etc.
	Critical     bool                   `json:"critical"`
	Metadata     map[string]interface{} `json:"metadata"`
	LastStatus   Status                 `json:"last_status"`
	LastCheck    time.Time              `json:"last_check"`
	FailureCount int                    `json:"failure_count"`
	HealthCheck  Check                  `json:"-"`
}

// AlertRule defines when and how to alert on health issues
type AlertRule struct {
	Name             string        `json:"name"`
	ComponentType    string        `json:"component_type,omitempty"`
	ComponentName    string        `json:"component_name,omitempty"`
	StatusTrigger    Status        `json:"status_trigger"`
	FailureThreshold int           `json:"failure_threshold"`
	AlertInterval    time.Duration `json:"alert_interval"`
	LastAlert        time.Time     `json:"last_alert"`
}

// AlertEvent represents a health alert
type AlertEvent struct {
	RuleName  string                 `json:"rule_name"`
	Component string                 `json:"component"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details"`
	Timestamp time.Time              `json:"timestamp"`
	Critical  bool                   `json:"critical"`
}

// AlertHandler defines the interface for handling health alerts
type AlertHandler func(event AlertEvent)

// NewMonitor creates a new health monitor
func NewMonitor(logger logger.Logger) *Monitor {
	return &Monitor{
		checker:    NewHealthChecker(logger),
		logger:     logger,
		components: make(map[string]MonitoredComponent),
		alerts:     []AlertRule{},
		stopCh:     make(chan struct{}),
	}
}

// RegisterComponent registers a component for health monitoring
func (m *Monitor) RegisterComponent(name, componentType string, critical bool, healthCheck Check) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	component := MonitoredComponent{
		Name:        name,
		Type:        componentType,
		Critical:    critical,
		Metadata:    make(map[string]interface{}),
		LastStatus:  StatusUnknown,
		HealthCheck: healthCheck,
	}

	m.components[name] = component
	m.checker.RegisterCheck(name, healthCheck)

	m.logger.Info("Component registered for health monitoring",
		"component", name,
		"type", componentType,
		"critical", critical)
}

// UnregisterComponent removes a component from monitoring
func (m *Monitor) UnregisterComponent(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.components, name)
	m.checker.UnregisterCheck(name)

	m.logger.Info("Component unregistered from health monitoring", "component", name)
}

// AddAlertRule adds an alert rule
func (m *Monitor) AddAlertRule(rule AlertRule) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.alerts = append(m.alerts, rule)

	m.logger.Info("Alert rule added",
		"rule", rule.Name,
		"component_type", rule.ComponentType,
		"component_name", rule.ComponentName,
		"trigger", rule.StatusTrigger)
}

// GetSystemHealth returns the complete system health status
func (m *Monitor) GetSystemHealth(ctx context.Context) SystemHealth {
	health := m.checker.GetSystemHealth(ctx)

	// Update component information with monitoring metadata
	m.mutex.RLock()
	for name, component := range m.components {
		if healthInfo, exists := health.Components[name]; exists {
			healthInfo.Metadata = component.Metadata
			healthInfo.Metadata["type"] = component.Type
			healthInfo.Metadata["critical"] = component.Critical
			healthInfo.Metadata["failure_count"] = component.FailureCount
			health.Components[name] = healthInfo
		}
	}
	m.mutex.RUnlock()

	return health
}

// StartMonitoring starts the health monitoring process
func (m *Monitor) StartMonitoring(ctx context.Context, alertHandler AlertHandler) {
	m.mutex.Lock()
	if m.running {
		m.mutex.Unlock()
		return
	}
	m.running = true
	m.mutex.Unlock()

	m.logger.Info("Starting health monitoring")

	// Start periodic health checks
	m.checker.StartPeriodicChecks(ctx)

	// Start alert monitoring
	go m.runAlertMonitoring(ctx, alertHandler)
}

// StopMonitoring stops the health monitoring process
func (m *Monitor) StopMonitoring() {
	m.mutex.Lock()
	if !m.running {
		m.mutex.Unlock()
		return
	}
	m.running = false
	m.mutex.Unlock()

	m.logger.Info("Stopping health monitoring")

	m.checker.StopPeriodicChecks()
	close(m.stopCh)
	m.stopCh = make(chan struct{})
}

// runAlertMonitoring monitors health check results and triggers alerts
func (m *Monitor) runAlertMonitoring(ctx context.Context, alertHandler AlertHandler) {
	ticker := time.NewTicker(10 * time.Second) // Check for alerts every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkForAlerts(alertHandler)
		}
	}
}

// checkForAlerts evaluates alert rules against current health status
func (m *Monitor) checkForAlerts(alertHandler AlertHandler) {
	results := m.checker.GetLastResults()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, rule := range m.alerts {
		// Check if enough time has passed since last alert
		if time.Since(rule.LastAlert) < rule.AlertInterval {
			continue
		}

		// Find matching components
		var matchingComponents []string
		for name, component := range m.components {
			if m.componentMatchesRule(component, rule) {
				matchingComponents = append(matchingComponents, name)
			}
		}

		// Check each matching component
		for _, componentName := range matchingComponents {
			if result, exists := results[componentName]; exists {
				component := m.components[componentName]

				// Update failure count
				if result.Status == StatusUnhealthy || result.Status == StatusDegraded {
					component.FailureCount++
				} else {
					component.FailureCount = 0
				}
				component.LastStatus = result.Status
				component.LastCheck = result.Timestamp
				m.components[componentName] = component

				// Check if alert should be triggered
				if m.shouldTriggerAlert(rule, component, result) {
					alertEvent := AlertEvent{
						RuleName:  rule.Name,
						Component: componentName,
						Status:    result.Status,
						Message:   result.Message,
						Details:   result.Details,
						Timestamp: time.Now(),
						Critical:  component.Critical,
					}

					// Update last alert time
					for i, r := range m.alerts {
						if r.Name == rule.Name {
							m.alerts[i].LastAlert = time.Now()
							break
						}
					}

					m.logger.Warn("Health alert triggered",
						"rule", rule.Name,
						"component", componentName,
						"status", result.Status,
						"critical", component.Critical)

					// Send alert
					if alertHandler != nil {
						go alertHandler(alertEvent)
					}
				}
			}
		}
	}
}

// componentMatchesRule checks if a component matches an alert rule
func (m *Monitor) componentMatchesRule(component MonitoredComponent, rule AlertRule) bool {
	// Check component name match
	if rule.ComponentName != "" && rule.ComponentName != component.Name {
		return false
	}

	// Check component type match
	if rule.ComponentType != "" && rule.ComponentType != component.Type {
		return false
	}

	return true
}

// shouldTriggerAlert determines if an alert should be triggered
func (m *Monitor) shouldTriggerAlert(rule AlertRule, component MonitoredComponent, result CheckResult) bool {
	// Check status trigger
	if result.Status != rule.StatusTrigger {
		return false
	}

	// Check failure threshold
	if component.FailureCount < rule.FailureThreshold {
		return false
	}

	return true
}

// GetComponentHealth returns health information for a specific component
func (m *Monitor) GetComponentHealth(ctx context.Context, componentName string) (ComponentHealth, error) {
	m.mutex.RLock()
	component, exists := m.components[componentName]
	m.mutex.RUnlock()

	if !exists {
		return ComponentHealth{}, errors.NewSystemError(
			errors.ErrSystemUnavailable,
			"health_monitor",
			fmt.Sprintf("component %s not found", componentName),
		)
	}

	result := m.checker.RunCheck(ctx, componentName)

	return ComponentHealth{
		Name:         componentName,
		Status:       result.Status,
		LastCheck:    result.Timestamp,
		CheckResults: []CheckResult{result},
		Metadata: map[string]interface{}{
			"type":          component.Type,
			"critical":      component.Critical,
			"failure_count": component.FailureCount,
		},
	}, nil
}

// GetCriticalComponents returns all critical components and their health status
func (m *Monitor) GetCriticalComponents(ctx context.Context) map[string]ComponentHealth {
	m.mutex.RLock()
	criticalComponents := make([]string, 0)
	for name, component := range m.components {
		if component.Critical {
			criticalComponents = append(criticalComponents, name)
		}
	}
	m.mutex.RUnlock()

	result := make(map[string]ComponentHealth)
	for _, name := range criticalComponents {
		if health, err := m.GetComponentHealth(ctx, name); err == nil {
			result[name] = health
		}
	}

	return result
}

// IsSystemHealthy returns true if all critical components are healthy
func (m *Monitor) IsSystemHealthy(ctx context.Context) bool {
	criticalComponents := m.GetCriticalComponents(ctx)

	for _, health := range criticalComponents {
		if health.Status == StatusUnhealthy {
			return false
		}
	}

	return true
}

// GetHealthSummary returns a summary of system health
func (m *Monitor) GetHealthSummary(ctx context.Context) HealthSummary {
	systemHealth := m.GetSystemHealth(ctx)
	return systemHealth.Summary
}

// DefaultAlertRules returns a set of default alert rules
func DefaultAlertRules() []AlertRule {
	return []AlertRule{
		{
			Name:             "critical_component_failure",
			StatusTrigger:    StatusUnhealthy,
			FailureThreshold: 1,
			AlertInterval:    5 * time.Minute,
		},
		{
			Name:             "component_degraded",
			StatusTrigger:    StatusDegraded,
			FailureThreshold: 3,
			AlertInterval:    15 * time.Minute,
		},
		{
			Name:             "platform_rate_limit",
			ComponentType:    "platform",
			StatusTrigger:    StatusDegraded,
			FailureThreshold: 2,
			AlertInterval:    10 * time.Minute,
		},
		{
			Name:             "queue_failure",
			ComponentType:    "queue",
			StatusTrigger:    StatusUnhealthy,
			FailureThreshold: 1,
			AlertInterval:    2 * time.Minute,
		},
	}
}
