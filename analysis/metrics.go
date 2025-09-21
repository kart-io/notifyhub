package analysis

import (
	"sync"
	"time"

	"github.com/kart-io/notifyhub/core"
)

// Metrics tracks sending metrics over time
type Metrics struct {
	totalSent       int64
	totalFailed     int64
	totalPending    int64
	platformMetrics map[string]*PlatformMetrics
	mutex           sync.RWMutex
	startTime       time.Time
}

// PlatformMetrics tracks metrics for a specific platform
type PlatformMetrics struct {
	Sent     int64     `json:"sent"`
	Failed   int64     `json:"failed"`
	Pending  int64     `json:"pending"`
	LastUsed time.Time `json:"last_used"`
}

// MetricsSnapshot provides a point-in-time view of metrics
type MetricsSnapshot struct {
	TotalSent       int64                       `json:"total_sent"`
	TotalFailed     int64                       `json:"total_failed"`
	TotalPending    int64                       `json:"total_pending"`
	SuccessRate     float64                     `json:"success_rate"`
	FailureRate     float64                     `json:"failure_rate"`
	PlatformMetrics map[string]*PlatformMetrics `json:"platform_metrics"`
	UptimeSeconds   int64                       `json:"uptime_seconds"`
	Timestamp       time.Time                   `json:"timestamp"`
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		platformMetrics: make(map[string]*PlatformMetrics),
		startTime:       time.Now(),
	}
}

// UpdateFromResults updates metrics from sending results
func (m *Metrics) UpdateFromResults(results *core.SendingResults) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	platformCounts := make(map[string]*PlatformMetrics)

	for _, result := range results.Results {
		platform := result.Target.Platform

		// Initialize platform metrics if not exists
		if _, exists := m.platformMetrics[platform]; !exists {
			m.platformMetrics[platform] = &PlatformMetrics{}
		}

		// Initialize platform counts for this batch
		if _, exists := platformCounts[platform]; !exists {
			platformCounts[platform] = &PlatformMetrics{}
		}

		// Count results by status
		switch result.Status {
		case core.StatusSent:
			m.totalSent++
			m.platformMetrics[platform].Sent++
			platformCounts[platform].Sent++
		case core.StatusFailed:
			m.totalFailed++
			m.platformMetrics[platform].Failed++
			platformCounts[platform].Failed++
		case core.StatusPending:
			m.totalPending++
			m.platformMetrics[platform].Pending++
			platformCounts[platform].Pending++
		}

		// Update last used time
		m.platformMetrics[platform].LastUsed = time.Now()
	}
}

// CalculateMetrics calculates metrics from current results
func (m *Metrics) CalculateMetrics(results *core.SendingResults) *MetricsSnapshot {
	snapshot := m.GetSnapshot()

	// Add current batch metrics
	snapshot.TotalSent += int64(results.Success)
	snapshot.TotalFailed += int64(results.Failed)
	snapshot.TotalPending += int64(results.Total - results.Success - results.Failed)

	// Recalculate rates
	total := snapshot.TotalSent + snapshot.TotalFailed + snapshot.TotalPending
	if total > 0 {
		snapshot.SuccessRate = float64(snapshot.TotalSent) / float64(total)
		snapshot.FailureRate = float64(snapshot.TotalFailed) / float64(total)
	}

	return snapshot
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() *MetricsSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Copy platform metrics
	platformMetrics := make(map[string]*PlatformMetrics)
	for platform, metrics := range m.platformMetrics {
		platformMetrics[platform] = &PlatformMetrics{
			Sent:     metrics.Sent,
			Failed:   metrics.Failed,
			Pending:  metrics.Pending,
			LastUsed: metrics.LastUsed,
		}
	}

	// Calculate rates
	total := m.totalSent + m.totalFailed + m.totalPending
	var successRate, failureRate float64
	if total > 0 {
		successRate = float64(m.totalSent) / float64(total)
		failureRate = float64(m.totalFailed) / float64(total)
	}

	return &MetricsSnapshot{
		TotalSent:       m.totalSent,
		TotalFailed:     m.totalFailed,
		TotalPending:    m.totalPending,
		SuccessRate:     successRate,
		FailureRate:     failureRate,
		PlatformMetrics: platformMetrics,
		UptimeSeconds:   int64(time.Since(m.startTime).Seconds()),
		Timestamp:       time.Now(),
	}
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.totalSent = 0
	m.totalFailed = 0
	m.totalPending = 0
	m.platformMetrics = make(map[string]*PlatformMetrics)
	m.startTime = time.Now()
}

// GetPlatformMetrics returns metrics for a specific platform
func (m *Metrics) GetPlatformMetrics(platform string) (*PlatformMetrics, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	metrics, exists := m.platformMetrics[platform]
	if !exists {
		return nil, false
	}

	// Return a copy
	return &PlatformMetrics{
		Sent:     metrics.Sent,
		Failed:   metrics.Failed,
		Pending:  metrics.Pending,
		LastUsed: metrics.LastUsed,
	}, true
}

// ListPlatforms returns all platforms with metrics
func (m *Metrics) ListPlatforms() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	platforms := make([]string, 0, len(m.platformMetrics))
	for platform := range m.platformMetrics {
		platforms = append(platforms, platform)
	}
	return platforms
}
