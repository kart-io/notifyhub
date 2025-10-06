// Package email provides monitoring and logging utilities for email platform
package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart/notifyhub/pkg/utils/logger"
)

// EmailMetrics tracks email sending metrics
type EmailMetrics struct {
	mu             sync.RWMutex
	TotalSent      int64     `json:"total_sent"`
	TotalFailed    int64     `json:"total_failed"`
	LastSentTime   time.Time `json:"last_sent_time"`
	LastFailTime   time.Time `json:"last_fail_time"`
	AverageLatency float64   `json:"average_latency_ms"`
	SuccessRate    float64   `json:"success_rate"`

	// Provider-specific metrics
	ProviderMetrics map[string]*ProviderMetrics `json:"provider_metrics"`

	// Recent activity
	RecentActivity []ActivityRecord `json:"recent_activity,omitempty"`
}

// EmailMetricsSnapshot represents a snapshot of email metrics without mutex (safe for copying)
type EmailMetricsSnapshot struct {
	TotalSent       int64                       `json:"total_sent"`
	TotalFailed     int64                       `json:"total_failed"`
	LastSentTime    time.Time                   `json:"last_sent_time"`
	LastFailTime    time.Time                   `json:"last_fail_time"`
	AverageLatency  float64                     `json:"average_latency_ms"`
	SuccessRate     float64                     `json:"success_rate"`
	ProviderMetrics map[string]*ProviderMetrics `json:"provider_metrics"`
	RecentActivity  []ActivityRecord            `json:"recent_activity,omitempty"`
}

// ProviderMetrics tracks metrics for a specific email provider
type ProviderMetrics struct {
	Provider       string        `json:"provider"`
	TotalSent      int64         `json:"total_sent"`
	TotalFailed    int64         `json:"total_failed"`
	AverageLatency time.Duration `json:"average_latency"`
	LastError      string        `json:"last_error,omitempty"`
	LastErrorTime  time.Time     `json:"last_error_time,omitempty"`
}

// ActivityRecord represents a recent email activity
type ActivityRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Provider  string    `json:"provider"`
	Target    string    `json:"target"`
	Success   bool      `json:"success"`
	Latency   int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
}

// EmailMonitor monitors email platform performance and health
type EmailMonitor struct {
	metrics     *EmailMetrics
	logger      logger.Logger
	maxActivity int // Maximum number of activity records to keep
}

// NewEmailMonitor creates a new email monitor
func NewEmailMonitor(logger logger.Logger) *EmailMonitor {
	return &EmailMonitor{
		metrics: &EmailMetrics{
			ProviderMetrics: make(map[string]*ProviderMetrics),
			RecentActivity:  make([]ActivityRecord, 0),
		},
		logger:      logger,
		maxActivity: 100, // Keep last 100 activities
	}
}

// RecordSending records a successful email sending
func (em *EmailMonitor) RecordSending(provider, target string, latency time.Duration) {
	em.metrics.mu.Lock()
	defer em.metrics.mu.Unlock()

	// Update global metrics
	em.metrics.TotalSent++
	em.metrics.LastSentTime = time.Now()

	// Update average latency
	if em.metrics.AverageLatency == 0 {
		em.metrics.AverageLatency = float64(latency.Milliseconds())
	} else {
		// Exponential moving average
		em.metrics.AverageLatency = 0.9*em.metrics.AverageLatency + 0.1*float64(latency.Milliseconds())
	}

	// Update success rate
	total := em.metrics.TotalSent + em.metrics.TotalFailed
	if total > 0 {
		em.metrics.SuccessRate = float64(em.metrics.TotalSent) / float64(total) * 100
	}

	// Update provider metrics
	providerMetric := em.getOrCreateProviderMetric(provider)
	providerMetric.TotalSent++
	if providerMetric.AverageLatency == 0 {
		providerMetric.AverageLatency = latency
	} else {
		// Exponential moving average
		providerMetric.AverageLatency = time.Duration(
			0.9*float64(providerMetric.AverageLatency) + 0.1*float64(latency),
		)
	}

	// Record activity
	em.addActivity(ActivityRecord{
		Timestamp: time.Now(),
		Provider:  provider,
		Target:    em.maskEmail(target),
		Success:   true,
		Latency:   latency.Milliseconds(),
	})

	em.logger.Debug("邮件发送成功记录",
		"provider", provider,
		"target", em.maskEmail(target),
		"latency", latency,
		"total_sent", em.metrics.TotalSent)
}

// RecordFailure records a failed email sending
func (em *EmailMonitor) RecordFailure(provider, target string, err error, latency time.Duration) {
	em.metrics.mu.Lock()
	defer em.metrics.mu.Unlock()

	// Update global metrics
	em.metrics.TotalFailed++
	em.metrics.LastFailTime = time.Now()

	// Update success rate
	total := em.metrics.TotalSent + em.metrics.TotalFailed
	if total > 0 {
		em.metrics.SuccessRate = float64(em.metrics.TotalSent) / float64(total) * 100
	}

	// Update provider metrics
	providerMetric := em.getOrCreateProviderMetric(provider)
	providerMetric.TotalFailed++
	providerMetric.LastError = err.Error()
	providerMetric.LastErrorTime = time.Now()

	// Record activity
	em.addActivity(ActivityRecord{
		Timestamp: time.Now(),
		Provider:  provider,
		Target:    em.maskEmail(target),
		Success:   false,
		Latency:   latency.Milliseconds(),
		Error:     err.Error(),
	})

	em.logger.Error("邮件发送失败记录",
		"provider", provider,
		"target", em.maskEmail(target),
		"error", err,
		"latency", latency,
		"total_failed", em.metrics.TotalFailed)
}

// getOrCreateProviderMetric gets or creates provider metrics
func (em *EmailMonitor) getOrCreateProviderMetric(provider string) *ProviderMetrics {
	if metric, exists := em.metrics.ProviderMetrics[provider]; exists {
		return metric
	}

	metric := &ProviderMetrics{
		Provider: provider,
	}
	em.metrics.ProviderMetrics[provider] = metric
	return metric
}

// addActivity adds an activity record, maintaining the maximum limit
func (em *EmailMonitor) addActivity(activity ActivityRecord) {
	em.metrics.RecentActivity = append(em.metrics.RecentActivity, activity)

	// Keep only the most recent activities
	if len(em.metrics.RecentActivity) > em.maxActivity {
		em.metrics.RecentActivity = em.metrics.RecentActivity[len(em.metrics.RecentActivity)-em.maxActivity:]
	}
}

// maskEmail masks email address for privacy
func (em *EmailMonitor) maskEmail(email string) string {
	if len(email) < 5 {
		return "***"
	}

	atIndex := -1
	for i, char := range email {
		if char == '@' {
			atIndex = i
			break
		}
	}

	if atIndex <= 0 {
		return "***"
	}

	// Show first character and @ onwards
	if atIndex >= 3 {
		return email[:1] + "***" + email[atIndex:]
	}
	return "***" + email[atIndex:]
}

// GetMetrics returns current metrics (thread-safe)
func (em *EmailMonitor) GetMetrics() EmailMetricsSnapshot {
	em.metrics.mu.RLock()
	defer em.metrics.mu.RUnlock()

	// Deep copy to prevent race conditions
	metrics := EmailMetricsSnapshot{
		TotalSent:       em.metrics.TotalSent,
		TotalFailed:     em.metrics.TotalFailed,
		LastSentTime:    em.metrics.LastSentTime,
		LastFailTime:    em.metrics.LastFailTime,
		AverageLatency:  em.metrics.AverageLatency,
		SuccessRate:     em.metrics.SuccessRate,
		ProviderMetrics: make(map[string]*ProviderMetrics),
		RecentActivity:  make([]ActivityRecord, len(em.metrics.RecentActivity)),
	}

	// Copy provider metrics
	for k, v := range em.metrics.ProviderMetrics {
		metrics.ProviderMetrics[k] = &ProviderMetrics{
			Provider:       v.Provider,
			TotalSent:      v.TotalSent,
			TotalFailed:    v.TotalFailed,
			AverageLatency: v.AverageLatency,
			LastError:      v.LastError,
			LastErrorTime:  v.LastErrorTime,
		}
	}

	// Copy activity records
	copy(metrics.RecentActivity, em.metrics.RecentActivity)

	return metrics
}

// GetHealthStatus returns the health status of email platform
func (em *EmailMonitor) GetHealthStatus() HealthStatus {
	metrics := em.GetMetrics()

	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Metrics:   metrics,
		Issues:    []HealthIssue{},
	}

	// Check for health issues
	if metrics.SuccessRate < 80 && metrics.TotalSent+metrics.TotalFailed > 10 {
		status.Status = "degraded"
		status.Issues = append(status.Issues, HealthIssue{
			Type:        "low_success_rate",
			Severity:    "warning",
			Description: fmt.Sprintf("成功率较低: %.1f%%", metrics.SuccessRate),
			Suggestion:  "检查邮件配置和网络连接",
		})
	}

	if metrics.SuccessRate < 50 && metrics.TotalSent+metrics.TotalFailed > 10 {
		status.Status = "unhealthy"
		status.Issues = append(status.Issues, HealthIssue{
			Type:        "very_low_success_rate",
			Severity:    "error",
			Description: fmt.Sprintf("成功率极低: %.1f%%", metrics.SuccessRate),
			Suggestion:  "立即检查邮件服务配置",
		})
	}

	if metrics.AverageLatency > 30000 { // 30 seconds
		status.Issues = append(status.Issues, HealthIssue{
			Type:        "high_latency",
			Severity:    "warning",
			Description: fmt.Sprintf("平均延迟较高: %.1fms", metrics.AverageLatency),
			Suggestion:  "检查网络连接和SMTP服务器性能",
		})
	}

	// Check recent failures
	recentFailures := 0
	cutoff := time.Now().Add(-5 * time.Minute)
	for _, activity := range metrics.RecentActivity {
		if activity.Timestamp.After(cutoff) && !activity.Success {
			recentFailures++
		}
	}

	if recentFailures > 5 {
		status.Status = "degraded"
		status.Issues = append(status.Issues, HealthIssue{
			Type:        "recent_failures",
			Severity:    "warning",
			Description: fmt.Sprintf("最近5分钟内失败 %d 次", recentFailures),
			Suggestion:  "检查当前网络状况和服务器状态",
		})
	}

	if len(status.Issues) == 0 {
		status.Status = "healthy"
	}

	return status
}

// HealthStatus represents the health status of email platform
type HealthStatus struct {
	Status    string               `json:"status"` // healthy, degraded, unhealthy
	Timestamp time.Time            `json:"timestamp"`
	Metrics   EmailMetricsSnapshot `json:"metrics"`
	Issues    []HealthIssue        `json:"issues,omitempty"`
}

// HealthIssue represents a health issue
type HealthIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"` // info, warning, error
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// LogPerformanceReport logs a detailed performance report
func (em *EmailMonitor) LogPerformanceReport() {
	metrics := em.GetMetrics()

	em.logger.Info("📊 邮件平台性能报告",
		"total_sent", metrics.TotalSent,
		"total_failed", metrics.TotalFailed,
		"success_rate", fmt.Sprintf("%.1f%%", metrics.SuccessRate),
		"average_latency", fmt.Sprintf("%.1fms", metrics.AverageLatency))

	// Log provider-specific metrics
	for provider, providerMetric := range metrics.ProviderMetrics {
		em.logger.Info("📧 服务商统计",
			"provider", provider,
			"sent", providerMetric.TotalSent,
			"failed", providerMetric.TotalFailed,
			"avg_latency", providerMetric.AverageLatency)
	}

	// Log health status
	healthStatus := em.GetHealthStatus()
	if healthStatus.Status != "healthy" {
		em.logger.Warn("⚠️  邮件服务健康状况", "status", healthStatus.Status)
		for _, issue := range healthStatus.Issues {
			em.logger.Warn("健康问题",
				"type", issue.Type,
				"severity", issue.Severity,
				"description", issue.Description,
				"suggestion", issue.Suggestion)
		}
	} else {
		em.logger.Info("✅ 邮件服务健康状况良好")
	}
}

// StartPeriodicReporting starts periodic performance reporting
func (em *EmailMonitor) StartPeriodicReporting(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			em.LogPerformanceReport()
		case <-ctx.Done():
			em.logger.Info("停止邮件性能监控")
			return
		}
	}
}

// ResetMetrics resets all metrics
func (em *EmailMonitor) ResetMetrics() {
	em.metrics.mu.Lock()
	defer em.metrics.mu.Unlock()

	em.metrics.TotalSent = 0
	em.metrics.TotalFailed = 0
	em.metrics.AverageLatency = 0
	em.metrics.SuccessRate = 0
	em.metrics.ProviderMetrics = make(map[string]*ProviderMetrics)
	em.metrics.RecentActivity = make([]ActivityRecord, 0)

	em.logger.Info("邮件监控指标已重置")
}
