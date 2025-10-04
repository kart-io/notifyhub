// Package metrics provides monitoring metrics functionality for NotifyHub
package metrics

import (
	"sync"
	"time"
)

// Metrics interface for collecting and reporting metrics
type Metrics interface {
	// Counter metrics
	IncrementCounter(name string, tags map[string]string)
	AddCounter(name string, value float64, tags map[string]string)

	// Gauge metrics
	SetGauge(name string, value float64, tags map[string]string)
	AddGauge(name string, value float64, tags map[string]string)

	// Histogram metrics
	RecordHistogram(name string, value float64, tags map[string]string)
	RecordTiming(name string, duration time.Duration, tags map[string]string)

	// Summary metrics
	RecordSummary(name string, value float64, tags map[string]string)

	// Custom metrics
	RecordCustom(name string, value interface{}, metricType string, tags map[string]string)

	// Reporting
	GetMetrics() map[string]MetricValue
	Reset()
	Close() error
}

// MetricValue represents a metric value with metadata
type MetricValue struct {
	Name      string                 `json:"name"`
	Value     interface{}            `json:"value"`
	Type      string                 `json:"type"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricType constants
const (
	MetricTypeCounter   = "counter"
	MetricTypeGauge     = "gauge"
	MetricTypeHistogram = "histogram"
	MetricTypeSummary   = "summary"
	MetricTypeTiming    = "timing"
	MetricTypeCustom    = "custom"
)

// MemoryMetrics implements an in-memory metrics collector
type MemoryMetrics struct {
	metrics map[string]MetricValue
	mutex   sync.RWMutex
}

// NewMemoryMetrics creates a new in-memory metrics collector
func NewMemoryMetrics() *MemoryMetrics {
	return &MemoryMetrics{
		metrics: make(map[string]MetricValue),
	}
}

// IncrementCounter increments a counter metric
func (m *MemoryMetrics) IncrementCounter(name string, tags map[string]string) {
	m.AddCounter(name, 1, tags)
}

// AddCounter adds to a counter metric
func (m *MemoryMetrics) AddCounter(name string, value float64, tags map[string]string) {
	key := m.buildKey(name, tags)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if existing, exists := m.metrics[key]; exists {
		if existingValue, ok := existing.Value.(float64); ok {
			value += existingValue
		}
	}

	m.metrics[key] = MetricValue{
		Name:      name,
		Value:     value,
		Type:      MetricTypeCounter,
		Tags:      tags,
		Timestamp: time.Now(),
	}
}

// SetGauge sets a gauge metric
func (m *MemoryMetrics) SetGauge(name string, value float64, tags map[string]string) {
	key := m.buildKey(name, tags)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.metrics[key] = MetricValue{
		Name:      name,
		Value:     value,
		Type:      MetricTypeGauge,
		Tags:      tags,
		Timestamp: time.Now(),
	}
}

// AddGauge adds to a gauge metric
func (m *MemoryMetrics) AddGauge(name string, value float64, tags map[string]string) {
	key := m.buildKey(name, tags)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if existing, exists := m.metrics[key]; exists {
		if existingValue, ok := existing.Value.(float64); ok {
			value += existingValue
		}
	}

	m.metrics[key] = MetricValue{
		Name:      name,
		Value:     value,
		Type:      MetricTypeGauge,
		Tags:      tags,
		Timestamp: time.Now(),
	}
}

// RecordHistogram records a histogram metric
func (m *MemoryMetrics) RecordHistogram(name string, value float64, tags map[string]string) {
	key := m.buildKey(name, tags)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// For memory implementation, just store the latest value
	// In a real implementation, this would maintain buckets/percentiles
	m.metrics[key] = MetricValue{
		Name:      name,
		Value:     value,
		Type:      MetricTypeHistogram,
		Tags:      tags,
		Timestamp: time.Now(),
	}
}

// RecordTiming records a timing metric
func (m *MemoryMetrics) RecordTiming(name string, duration time.Duration, tags map[string]string) {
	key := m.buildKey(name, tags)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.metrics[key] = MetricValue{
		Name:      name,
		Value:     duration.Milliseconds(),
		Type:      MetricTypeTiming,
		Tags:      tags,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
			"duration_ns": duration.Nanoseconds(),
		},
	}
}

// RecordSummary records a summary metric
func (m *MemoryMetrics) RecordSummary(name string, value float64, tags map[string]string) {
	key := m.buildKey(name, tags)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.metrics[key] = MetricValue{
		Name:      name,
		Value:     value,
		Type:      MetricTypeSummary,
		Tags:      tags,
		Timestamp: time.Now(),
	}
}

// RecordCustom records a custom metric
func (m *MemoryMetrics) RecordCustom(name string, value interface{}, metricType string, tags map[string]string) {
	key := m.buildKey(name, tags)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.metrics[key] = MetricValue{
		Name:      name,
		Value:     value,
		Type:      metricType,
		Tags:      tags,
		Timestamp: time.Now(),
	}
}

// GetMetrics returns all metrics
func (m *MemoryMetrics) GetMetrics() map[string]MetricValue {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]MetricValue)
	for k, v := range m.metrics {
		result[k] = v
	}
	return result
}

// Reset clears all metrics
func (m *MemoryMetrics) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.metrics = make(map[string]MetricValue)
}

// Close shuts down the metrics collector
func (m *MemoryMetrics) Close() error {
	m.Reset()
	return nil
}

// buildKey builds a unique key for a metric with tags
func (m *MemoryMetrics) buildKey(name string, tags map[string]string) string {
	key := name
	if len(tags) > 0 {
		key += "{"
		first := true
		for k, v := range tags {
			if !first {
				key += ","
			}
			key += k + "=" + v
			first = false
		}
		key += "}"
	}
	return key
}

// NoOpMetrics implements a no-operation metrics collector
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-op metrics collector
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

func (n *NoOpMetrics) IncrementCounter(name string, tags map[string]string)                     {}
func (n *NoOpMetrics) AddCounter(name string, value float64, tags map[string]string)            {}
func (n *NoOpMetrics) SetGauge(name string, value float64, tags map[string]string)              {}
func (n *NoOpMetrics) AddGauge(name string, value float64, tags map[string]string)              {}
func (n *NoOpMetrics) RecordHistogram(name string, value float64, tags map[string]string)       {}
func (n *NoOpMetrics) RecordTiming(name string, duration time.Duration, tags map[string]string) {}
func (n *NoOpMetrics) RecordSummary(name string, value float64, tags map[string]string)         {}
func (n *NoOpMetrics) RecordCustom(name string, value interface{}, metricType string, tags map[string]string) {
}
func (n *NoOpMetrics) GetMetrics() map[string]MetricValue { return make(map[string]MetricValue) }
func (n *NoOpMetrics) Reset()                             {}
func (n *NoOpMetrics) Close() error                       { return nil }

// Global metrics instance
var defaultMetrics Metrics = NewNoOpMetrics()

// SetDefaultMetrics sets the default global metrics instance
func SetDefaultMetrics(m Metrics) {
	defaultMetrics = m
}

// GetDefaultMetrics returns the default global metrics instance
func GetDefaultMetrics() Metrics {
	return defaultMetrics
}

// Convenience functions using the default metrics instance

// Counter increments a counter
func Counter(name string, tags map[string]string) {
	defaultMetrics.IncrementCounter(name, tags)
}

// CounterAdd adds to a counter
func CounterAdd(name string, value float64, tags map[string]string) {
	defaultMetrics.AddCounter(name, value, tags)
}

// Gauge sets a gauge
func Gauge(name string, value float64, tags map[string]string) {
	defaultMetrics.SetGauge(name, value, tags)
}

// Histogram records a histogram value
func Histogram(name string, value float64, tags map[string]string) {
	defaultMetrics.RecordHistogram(name, value, tags)
}

// Timing records a timing metric
func Timing(name string, duration time.Duration, tags map[string]string) {
	defaultMetrics.RecordTiming(name, duration, tags)
}

// Timer provides timing functionality
type Timer struct {
	name      string
	tags      map[string]string
	startTime time.Time
	metrics   Metrics
}

// NewTimer creates a new timer
func NewTimer(name string, tags map[string]string) *Timer {
	return &Timer{
		name:      name,
		tags:      tags,
		startTime: time.Now(),
		metrics:   defaultMetrics,
	}
}

// NewTimerWithMetrics creates a new timer with custom metrics instance
func NewTimerWithMetrics(name string, tags map[string]string, metrics Metrics) *Timer {
	return &Timer{
		name:      name,
		tags:      tags,
		startTime: time.Now(),
		metrics:   metrics,
	}
}

// Stop stops the timer and records the duration
func (t *Timer) Stop() time.Duration {
	duration := time.Since(t.startTime)
	t.metrics.RecordTiming(t.name, duration, t.tags)
	return duration
}

// Common metric names for NotifyHub
const (
	// Message metrics
	MetricMessagesTotal    = "notifyhub.messages.total"
	MetricMessagesSuccess  = "notifyhub.messages.success"
	MetricMessagesFailure  = "notifyhub.messages.failure"
	MetricMessagesDuration = "notifyhub.messages.duration"

	// Platform metrics
	MetricPlatformRequests = "notifyhub.platform.requests"
	MetricPlatformErrors   = "notifyhub.platform.errors"
	MetricPlatformLatency  = "notifyhub.platform.latency"

	// Queue metrics
	MetricQueueSize      = "notifyhub.queue.size"
	MetricQueueProcessed = "notifyhub.queue.processed"
	MetricQueueFailed    = "notifyhub.queue.failed"

	// System metrics
	MetricSystemMemory      = "notifyhub.system.memory"
	MetricSystemGoroutines  = "notifyhub.system.goroutines"
	MetricSystemConnections = "notifyhub.system.connections"
)

// Helper functions for common metrics

// RecordMessageSent records a successful message send
func RecordMessageSent(platform string, duration time.Duration) {
	tags := map[string]string{"platform": platform, "status": "success"}
	Counter(MetricMessagesTotal, tags)
	Counter(MetricMessagesSuccess, tags)
	Timing(MetricMessagesDuration, duration, tags)
}

// RecordMessageFailed records a failed message send
func RecordMessageFailed(platform string, errorCode string, duration time.Duration) {
	tags := map[string]string{"platform": platform, "status": "failure", "error": errorCode}
	Counter(MetricMessagesTotal, tags)
	Counter(MetricMessagesFailure, tags)
	Timing(MetricMessagesDuration, duration, tags)
}

// RecordPlatformRequest records a platform request
func RecordPlatformRequest(platform string, duration time.Duration, success bool) {
	tags := map[string]string{"platform": platform}
	Counter(MetricPlatformRequests, tags)
	Timing(MetricPlatformLatency, duration, tags)

	if !success {
		Counter(MetricPlatformErrors, tags)
	}
}
