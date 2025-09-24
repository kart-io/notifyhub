// Package queue provides metrics collection and monitoring for queue operations
package queue

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and manages queue metrics
type MetricsCollector struct {
	queueMetrics     *QueueMetrics
	workerMetrics    *WorkerMetrics
	healthStatus     *HealthStatus
	startTime        time.Time
	lastProcessTime  time.Time
	subscribers      map[string][]func(msg *Message)
	mutex            sync.RWMutex

	// Internal counters
	totalProcessingTime int64 // in nanoseconds
	processingCount     int64
	errorList           []HealthError
	errorMutex          sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	now := time.Now()
	return &MetricsCollector{
		queueMetrics: &QueueMetrics{
			Size:            0,
			EnqueuedCount:   0,
			DequeuedCount:   0,
			RetryCount:      0,
			DeadLetterCount: 0,
			AverageLatency:  0,
			ThroughputQPS:   0,
		},
		workerMetrics: &WorkerMetrics{
			ActiveWorkers:   0,
			IdleWorkers:     0,
			ProcessedCount:  0,
			ErrorCount:      0,
			AverageTime:     0,
		},
		healthStatus: &HealthStatus{
			Status:      "healthy",
			QueueSize:   0,
			Workers:     0,
			LastProcess: now,
			Errors:      []HealthError{},
			Uptime:      0,
		},
		startTime:       now,
		lastProcessTime: now,
		subscribers:     make(map[string][]func(msg *Message)),
	}
}

// RecordEnqueue records a message enqueue operation
func (mc *MetricsCollector) RecordEnqueue(msg *Message) {
	atomic.AddInt64(&mc.queueMetrics.EnqueuedCount, 1)
	atomic.AddInt64(&mc.queueMetrics.Size, 1)
	mc.notifySubscribers("enqueue", msg)
	mc.updateLastProcessTime()
}

// RecordDequeue records a message dequeue operation
func (mc *MetricsCollector) RecordDequeue(msg *Message) {
	atomic.AddInt64(&mc.queueMetrics.DequeuedCount, 1)
	atomic.AddInt64(&mc.queueMetrics.Size, -1)
	mc.notifySubscribers("dequeue", msg)
	mc.updateLastProcessTime()
}

// RecordRetry records a retry operation
func (mc *MetricsCollector) RecordRetry(msg *Message) {
	atomic.AddInt64(&mc.queueMetrics.RetryCount, 1)
	mc.notifySubscribers("retry", msg)
}

// RecordDeadLetter records a message moved to dead letter queue
func (mc *MetricsCollector) RecordDeadLetter(msg *Message) {
	atomic.AddInt64(&mc.queueMetrics.DeadLetterCount, 1)
	mc.notifySubscribers("dead_letter", msg)
}

// RecordProcessing records message processing metrics
func (mc *MetricsCollector) RecordProcessing(msg *Message, duration time.Duration, err error) {
	atomic.AddInt64(&mc.workerMetrics.ProcessedCount, 1)
	atomic.AddInt64(&mc.totalProcessingTime, duration.Nanoseconds())
	atomic.AddInt64(&mc.processingCount, 1)

	if err != nil {
		atomic.AddInt64(&mc.workerMetrics.ErrorCount, 1)
		mc.recordError("processing_error", err.Error(), "error")
	}

	// Update average processing time
	totalTime := atomic.LoadInt64(&mc.totalProcessingTime)
	count := atomic.LoadInt64(&mc.processingCount)
	if count > 0 {
		mc.workerMetrics.AverageTime = time.Duration(totalTime / count)
	}

	mc.updateLastProcessTime()
}

// UpdateWorkerCounts updates worker pool metrics
func (mc *MetricsCollector) UpdateWorkerCounts(active, idle int) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.workerMetrics.ActiveWorkers = active
	mc.workerMetrics.IdleWorkers = idle
	mc.healthStatus.Workers = active + idle
}

// UpdateQueueSize updates current queue size
func (mc *MetricsCollector) UpdateQueueSize(size int) {
	atomic.StoreInt64(&mc.queueMetrics.Size, int64(size))
	mc.healthStatus.QueueSize = size
}

// GetQueueMetrics returns current queue metrics
func (mc *MetricsCollector) GetQueueMetrics() *QueueMetrics {
	// Calculate throughput QPS
	uptime := time.Since(mc.startTime)
	if uptime > 0 {
		processed := atomic.LoadInt64(&mc.queueMetrics.DequeuedCount)
		mc.queueMetrics.ThroughputQPS = float64(processed) / uptime.Seconds()
	}

	// Calculate average latency (simplified)
	if mc.processingCount > 0 {
		avgTime := time.Duration(atomic.LoadInt64(&mc.totalProcessingTime) / atomic.LoadInt64(&mc.processingCount))
		mc.queueMetrics.AverageLatency = avgTime
	}

	// Return a copy to avoid concurrent modification
	return &QueueMetrics{
		Size:            atomic.LoadInt64(&mc.queueMetrics.Size),
		EnqueuedCount:   atomic.LoadInt64(&mc.queueMetrics.EnqueuedCount),
		DequeuedCount:   atomic.LoadInt64(&mc.queueMetrics.DequeuedCount),
		RetryCount:      atomic.LoadInt64(&mc.queueMetrics.RetryCount),
		DeadLetterCount: atomic.LoadInt64(&mc.queueMetrics.DeadLetterCount),
		AverageLatency:  mc.queueMetrics.AverageLatency,
		ThroughputQPS:   mc.queueMetrics.ThroughputQPS,
	}
}

// GetWorkerMetrics returns current worker metrics
func (mc *MetricsCollector) GetWorkerMetrics() *WorkerMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	return &WorkerMetrics{
		ActiveWorkers:   mc.workerMetrics.ActiveWorkers,
		IdleWorkers:     mc.workerMetrics.IdleWorkers,
		ProcessedCount:  atomic.LoadInt64(&mc.workerMetrics.ProcessedCount),
		ErrorCount:      atomic.LoadInt64(&mc.workerMetrics.ErrorCount),
		AverageTime:     mc.workerMetrics.AverageTime,
	}
}

// GetHealthStatus returns current health status
func (mc *MetricsCollector) GetHealthStatus() *HealthStatus {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	uptime := time.Since(mc.startTime)
	status := mc.determineHealthStatus()

	// Copy errors
	errors := make([]HealthError, len(mc.errorList))
	copy(errors, mc.errorList)

	return &HealthStatus{
		Status:      status,
		QueueSize:   mc.healthStatus.QueueSize,
		Workers:     mc.healthStatus.Workers,
		LastProcess: mc.lastProcessTime,
		Errors:      errors,
		Uptime:      uptime,
	}
}

// Subscribe registers a callback for queue events
func (mc *MetricsCollector) Subscribe(event string, callback func(msg *Message)) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.subscribers[event] == nil {
		mc.subscribers[event] = make([]func(msg *Message), 0)
	}
	mc.subscribers[event] = append(mc.subscribers[event], callback)
}

// Unsubscribe removes callbacks for queue events
func (mc *MetricsCollector) Unsubscribe(event string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	delete(mc.subscribers, event)
}

// Private methods

func (mc *MetricsCollector) updateLastProcessTime() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.lastProcessTime = time.Now()
}

func (mc *MetricsCollector) notifySubscribers(event string, msg *Message) {
	mc.mutex.RLock()
	callbacks := mc.subscribers[event]
	mc.mutex.RUnlock()

	// Execute callbacks asynchronously to avoid blocking
	for _, callback := range callbacks {
		go func(cb func(msg *Message), m *Message) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't crash
				}
			}()
			cb(m)
		}(callback, msg)
	}
}

func (mc *MetricsCollector) recordError(code, message, severity string) {
	mc.errorMutex.Lock()
	defer mc.errorMutex.Unlock()

	error := HealthError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Severity:  severity,
	}

	// Keep only last 100 errors
	if len(mc.errorList) >= 100 {
		mc.errorList = mc.errorList[1:]
	}

	mc.errorList = append(mc.errorList, error)
}

func (mc *MetricsCollector) determineHealthStatus() string {
	queueSize := int64(mc.healthStatus.QueueSize)
	errorCount := atomic.LoadInt64(&mc.workerMetrics.ErrorCount)
	processedCount := atomic.LoadInt64(&mc.workerMetrics.ProcessedCount)

	// Calculate error rate
	var errorRate float64
	if processedCount > 0 {
		errorRate = float64(errorCount) / float64(processedCount)
	}

	// Determine status based on queue size and error rate
	if errorRate > 0.2 || queueSize > 10000 {
		return "unhealthy"
	} else if errorRate > 0.05 || queueSize > 5000 {
		return "degraded"
	}

	return "healthy"
}