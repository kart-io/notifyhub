package monitoring

import (
	"sync"
	"time"
)

// ================================
// 监控与指标
// ================================

// Metrics holds notification metrics
type Metrics struct {
	mu sync.RWMutex

	TotalSent       int64             `json:"total_sent"`
	TotalFailed     int64             `json:"total_failed"`
	SendsByPlatform map[string]int64  `json:"sends_by_platform"`
	FailsByPlatform map[string]int64  `json:"fails_by_platform"`
	LastErrors      map[string]string `json:"last_errors"`
	AvgDuration     time.Duration     `json:"avg_duration"`
	MaxDuration     time.Duration     `json:"max_duration"`
	PlatformHealth  map[string]bool   `json:"platform_health"`
	StartTime       time.Time         `json:"start_time"`
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		SendsByPlatform: make(map[string]int64),
		FailsByPlatform: make(map[string]int64),
		LastErrors:      make(map[string]string),
		PlatformHealth:  make(map[string]bool),
		StartTime:       time.Now(),
	}
}

// RecordSend records a send operation result
func (m *Metrics) RecordSend(platform string, success bool, duration time.Duration, err string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if success {
		m.TotalSent++
		m.SendsByPlatform[platform]++
	} else {
		m.TotalFailed++
		m.FailsByPlatform[platform]++
		if err != "" {
			m.LastErrors[platform] = err
		}
	}

	// Update timing
	total := m.TotalSent + m.TotalFailed
	if total > 0 {
		m.AvgDuration = time.Duration((int64(m.AvgDuration)*(total-1) + int64(duration)) / total)
	}
	if duration > m.MaxDuration {
		m.MaxDuration = duration
	}
}

// RecordHealth records platform health status
func (m *Metrics) RecordHealth(platform string, healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PlatformHealth[platform] = healthy
}

// GetSuccessRate returns the overall success rate
func (m *Metrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := m.TotalSent + m.TotalFailed
	if total == 0 {
		return 1.0
	}
	return float64(m.TotalSent) / float64(total)
}

// GetUptime returns the uptime since metrics started
func (m *Metrics) GetUptime() time.Duration {
	return time.Since(m.StartTime)
}

// GetSnapshot returns a complete snapshot of current metrics
func (m *Metrics) GetSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_sent":        m.TotalSent,
		"total_failed":      m.TotalFailed,
		"success_rate":      m.GetSuccessRate(),
		"sends_by_platform": m.SendsByPlatform,
		"fails_by_platform": m.FailsByPlatform,
		"last_errors":       m.LastErrors,
		"avg_duration":      m.AvgDuration.String(),
		"max_duration":      m.MaxDuration.String(),
		"platform_health":   m.PlatformHealth,
		"uptime":            m.GetUptime().String(),
	}
}
