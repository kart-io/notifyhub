// Package middleware provides metrics collection middleware for NotifyHub
package middleware

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// MetricsMiddleware collects and reports metrics for message operations
type MetricsMiddleware struct {
	BaseMiddleware
	collector MetricsCollector
	logger    logger.Logger
}

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	// Counter operations
	IncrementCounter(name string, tags map[string]string)
	AddToCounter(name string, value float64, tags map[string]string)

	// Histogram operations (for latency, duration)
	RecordHistogram(name string, value float64, tags map[string]string)

	// Gauge operations (for current values)
	SetGauge(name string, value float64, tags map[string]string)
	IncrementGauge(name string, tags map[string]string)
	DecrementGauge(name string, tags map[string]string)
}

// SimpleMetricsCollector is a basic in-memory metrics collector
type SimpleMetricsCollector struct {
	counters   map[string]*int64
	histograms map[string][]float64
	gauges     map[string]*int64
	logger     logger.Logger
}

// NewSimpleMetricsCollector creates a new simple metrics collector
func NewSimpleMetricsCollector(logger logger.Logger) *SimpleMetricsCollector {
	return &SimpleMetricsCollector{
		counters:   make(map[string]*int64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]*int64),
		logger:     logger,
	}
}

// IncrementCounter increments a counter metric
func (smc *SimpleMetricsCollector) IncrementCounter(name string, tags map[string]string) {
	smc.AddToCounter(name, 1, tags)
}

// AddToCounter adds a value to a counter metric
func (smc *SimpleMetricsCollector) AddToCounter(name string, value float64, tags map[string]string) {
	key := smc.buildKey(name, tags)

	if _, exists := smc.counters[key]; !exists {
		smc.counters[key] = new(int64)
	}

	atomic.AddInt64(smc.counters[key], int64(value))
}

// RecordHistogram records a value in a histogram metric
func (smc *SimpleMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	key := smc.buildKey(name, tags)

	if _, exists := smc.histograms[key]; !exists {
		smc.histograms[key] = make([]float64, 0, 1000) // Pre-allocate some capacity
	}

	smc.histograms[key] = append(smc.histograms[key], value)

	// Keep only recent values to prevent memory issues
	if len(smc.histograms[key]) > 10000 {
		smc.histograms[key] = smc.histograms[key][1000:] // Keep recent 9000 values
	}
}

// SetGauge sets a gauge metric to a specific value
func (smc *SimpleMetricsCollector) SetGauge(name string, value float64, tags map[string]string) {
	key := smc.buildKey(name, tags)

	if _, exists := smc.gauges[key]; !exists {
		smc.gauges[key] = new(int64)
	}

	atomic.StoreInt64(smc.gauges[key], int64(value))
}

// IncrementGauge increments a gauge metric
func (smc *SimpleMetricsCollector) IncrementGauge(name string, tags map[string]string) {
	key := smc.buildKey(name, tags)

	if _, exists := smc.gauges[key]; !exists {
		smc.gauges[key] = new(int64)
	}

	atomic.AddInt64(smc.gauges[key], 1)
}

// DecrementGauge decrements a gauge metric
func (smc *SimpleMetricsCollector) DecrementGauge(name string, tags map[string]string) {
	key := smc.buildKey(name, tags)

	if _, exists := smc.gauges[key]; !exists {
		smc.gauges[key] = new(int64)
	}

	atomic.AddInt64(smc.gauges[key], -1)
}

// buildKey builds a key from metric name and tags
func (smc *SimpleMetricsCollector) buildKey(name string, tags map[string]string) string {
	key := name
	for k, v := range tags {
		key += "," + k + "=" + v
	}
	return key
}

// GetCounterValue returns the current value of a counter
func (smc *SimpleMetricsCollector) GetCounterValue(name string, tags map[string]string) int64 {
	key := smc.buildKey(name, tags)
	if counter, exists := smc.counters[key]; exists {
		return atomic.LoadInt64(counter)
	}
	return 0
}

// GetGaugeValue returns the current value of a gauge
func (smc *SimpleMetricsCollector) GetGaugeValue(name string, tags map[string]string) int64 {
	key := smc.buildKey(name, tags)
	if gauge, exists := smc.gauges[key]; exists {
		return atomic.LoadInt64(gauge)
	}
	return 0
}

// GetHistogramValues returns the current values of a histogram
func (smc *SimpleMetricsCollector) GetHistogramValues(name string, tags map[string]string) []float64 {
	key := smc.buildKey(name, tags)
	if values, exists := smc.histograms[key]; exists {
		// Return a copy to avoid race conditions
		result := make([]float64, len(values))
		copy(result, values)
		return result
	}
	return []float64{}
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(collector MetricsCollector, l logger.Logger) *MetricsMiddleware {
	if l == nil {
		l = logger.Discard
	}

	if collector == nil {
		collector = NewSimpleMetricsCollector(l)
	}

	return &MetricsMiddleware{
		BaseMiddleware: NewBaseMiddleware("metrics"),
		collector:      collector,
		logger:         l,
	}
}

// HandleSend implements comprehensive metrics collection
func (mm *MetricsMiddleware) HandleSend(ctx context.Context, msg *message.Message, targets []target.Target, next SendHandler) (*receipt.Receipt, error) {
	startTime := time.Now()

	// Extract message metadata for tags
	tags := map[string]string{
		"format":   string(msg.Format),
		"priority": string(rune(msg.Priority)),
	}

	// Track active requests
	mm.collector.IncrementGauge("notifyhub_active_requests", tags)
	defer mm.collector.DecrementGauge("notifyhub_active_requests", tags)

	// Count total requests
	mm.collector.IncrementCounter("notifyhub_requests_total", tags)

	// Count targets by type
	targetCounts := make(map[string]int)
	for _, tgt := range targets {
		targetCounts[tgt.Type]++
	}

	for targetType, count := range targetCounts {
		targetTags := map[string]string{
			"target_type": targetType,
		}
		mm.collector.AddToCounter("notifyhub_targets_total", float64(count), targetTags)
	}

	// Execute the operation
	receipt, err := next(ctx, msg, targets)

	// Calculate duration
	duration := time.Since(startTime)
	mm.collector.RecordHistogram("notifyhub_request_duration_seconds", duration.Seconds(), tags)

	// Record results
	if err != nil {
		errorTags := map[string]string{
			"error_type": "send_error",
		}
		mm.collector.IncrementCounter("notifyhub_errors_total", errorTags)
		mm.collector.RecordHistogram("notifyhub_error_request_duration_seconds", duration.Seconds(), errorTags)
	} else if receipt != nil {
		// Success/failure metrics
		statusTags := map[string]string{
			"status": receipt.Status,
		}
		mm.collector.IncrementCounter("notifyhub_requests_by_status", statusTags)

		// Platform-specific metrics
		platformResults := make(map[string]map[string]int)
		for _, result := range receipt.Results {
			if _, exists := platformResults[result.Platform]; !exists {
				platformResults[result.Platform] = make(map[string]int)
			}

			if result.Success {
				platformResults[result.Platform]["success"]++
			} else {
				platformResults[result.Platform]["failed"]++
			}

			// Record platform-specific duration
			platformTags := map[string]string{
				"platform": result.Platform,
				"success":  fmt.Sprintf("%v", result.Success),
			}
			mm.collector.RecordHistogram("notifyhub_platform_duration_seconds",
				result.Duration.Seconds(), platformTags)
		}

		// Record platform results
		for platform, results := range platformResults {
			for status, count := range results {
				platformTags := map[string]string{
					"platform": platform,
					"result":   status,
				}
				mm.collector.AddToCounter("notifyhub_platform_results_total",
					float64(count), platformTags)
			}
		}

		// Overall success rate gauge
		if receipt.Total > 0 {
			successRate := float64(receipt.Successful) / float64(receipt.Total) * 100
			mm.collector.SetGauge("notifyhub_success_rate_percent", successRate, tags)
		}
	}

	mm.logger.Debug("Metrics recorded for message",
		"message_id", msg.ID,
		"duration", duration,
		"targets", len(targets))

	return receipt, err
}

// GetMetrics returns current metrics values (useful for monitoring endpoints)
func (mm *MetricsMiddleware) GetMetrics() map[string]interface{} {
	if smc, ok := mm.collector.(*SimpleMetricsCollector); ok {
		metrics := make(map[string]interface{})

		// Add counter metrics
		metrics["counters"] = make(map[string]int64)
		for key, counter := range smc.counters {
			if counter != nil {
				metrics["counters"].(map[string]int64)[key] = atomic.LoadInt64(counter)
			}
		}

		// Add gauge metrics
		metrics["gauges"] = make(map[string]int64)
		for key, gauge := range smc.gauges {
			if gauge != nil {
				metrics["gauges"].(map[string]int64)[key] = atomic.LoadInt64(gauge)
			}
		}

		// Add histogram summaries (count only to avoid large response)
		metrics["histograms"] = make(map[string]int)
		for key, values := range smc.histograms {
			metrics["histograms"].(map[string]int)[key] = len(values)
		}

		return metrics
	}

	return map[string]interface{}{
		"error": "metrics not available for this collector type",
	}
}

// ResetMetrics resets all metrics (useful for testing)
func (mm *MetricsMiddleware) ResetMetrics() {
	if smc, ok := mm.collector.(*SimpleMetricsCollector); ok {
		smc.counters = make(map[string]*int64)
		smc.histograms = make(map[string][]float64)
		smc.gauges = make(map[string]*int64)
		mm.logger.Debug("Metrics reset")
	}
}
