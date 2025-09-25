package queue

import (
	"context"
	"sync"
	"time"
)

type HealthChecker struct {
	queue         ObservableQueue
	workerPool    WorkerPool
	checkInterval time.Duration
	thresholds    HealthThresholds
	status        *HealthStatus
	mutex         sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

type HealthThresholds struct {
	MaxQueueSize      int
	MaxErrorRate      float64
	DegradedErrorRate float64
	MaxProcessingTime time.Duration
	MinWorkers        int
}

func NewHealthChecker(queue ObservableQueue, pool WorkerPool, interval time.Duration) *HealthChecker {
	if interval == 0 {
		interval = 30 * time.Second
	}

	return &HealthChecker{
		queue:         queue,
		workerPool:    pool,
		checkInterval: interval,
		thresholds: HealthThresholds{
			MaxQueueSize:      10000,
			MaxErrorRate:      0.2,
			DegradedErrorRate: 0.05,
			MaxProcessingTime: 30 * time.Second,
			MinWorkers:        1,
		},
		status: &HealthStatus{
			Status:      "healthy",
			QueueSize:   0,
			Workers:     0,
			LastProcess: time.Now(),
			Errors:      []HealthError{},
			Uptime:      0,
		},
		stopCh: make(chan struct{}),
	}
}

func (hc *HealthChecker) Start(ctx context.Context) {
	hc.wg.Add(1)
	go hc.run(ctx)
}

func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

func (hc *HealthChecker) GetStatus() *HealthStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	statusCopy := *hc.status
	errorsCopy := make([]HealthError, len(hc.status.Errors))
	copy(errorsCopy, hc.status.Errors)
	statusCopy.Errors = errorsCopy

	return &statusCopy
}

func (hc *HealthChecker) SetThresholds(thresholds HealthThresholds) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.thresholds = thresholds
}

func (hc *HealthChecker) run(ctx context.Context) {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.performHealthCheck(startTime)
		}
	}
}

func (hc *HealthChecker) performHealthCheck(startTime time.Time) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	queueMetrics := hc.queue.GetMetrics()
	var poolHealth *HealthStatus
	if hc.workerPool != nil {
		poolHealth = hc.workerPool.GetHealth()
	}

	queueSize := int(queueMetrics.Size)
	errorCount := queueMetrics.RetryCount + queueMetrics.DeadLetterCount
	processedCount := queueMetrics.DequeuedCount

	var errorRate float64
	if processedCount > 0 {
		errorRate = float64(errorCount) / float64(processedCount)
	}

	status := "healthy"
	errors := []HealthError{}

	if errorRate > hc.thresholds.MaxErrorRate {
		status = "unhealthy"
		errors = append(errors, HealthError{
			Code:      "high_error_rate",
			Message:   "Error rate exceeds maximum threshold",
			Timestamp: time.Now(),
			Severity:  "critical",
		})
	} else if errorRate > hc.thresholds.DegradedErrorRate {
		if status == "healthy" {
			status = "degraded"
		}
		errors = append(errors, HealthError{
			Code:      "elevated_error_rate",
			Message:   "Error rate exceeds degraded threshold",
			Timestamp: time.Now(),
			Severity:  "warning",
		})
	}

	if queueSize > hc.thresholds.MaxQueueSize {
		status = "unhealthy"
		errors = append(errors, HealthError{
			Code:      "queue_size_exceeded",
			Message:   "Queue size exceeds maximum threshold",
			Timestamp: time.Now(),
			Severity:  "critical",
		})
	} else if queueSize > hc.thresholds.MaxQueueSize/2 {
		if status == "healthy" {
			status = "degraded"
		}
		errors = append(errors, HealthError{
			Code:      "high_queue_size",
			Message:   "Queue size approaching maximum threshold",
			Timestamp: time.Now(),
			Severity:  "warning",
		})
	}

	workers := 0
	if hc.workerPool != nil {
		workers = hc.workerPool.GetWorkerCount()
		if workers < hc.thresholds.MinWorkers {
			status = "unhealthy"
			errors = append(errors, HealthError{
				Code:      "insufficient_workers",
				Message:   "Worker count below minimum threshold",
				Timestamp: time.Now(),
				Severity:  "critical",
			})
		}
	}

	if poolHealth != nil && len(poolHealth.Errors) > 0 {
		recentErrors := poolHealth.Errors
		if len(recentErrors) > 10 {
			recentErrors = recentErrors[len(recentErrors)-10:]
		}
		errors = append(errors, recentErrors...)

		if status == "healthy" && len(poolHealth.Errors) > 0 {
			status = "degraded"
		}
	}

	hc.status = &HealthStatus{
		Status:      status,
		QueueSize:   queueSize,
		Workers:     workers,
		LastProcess: time.Now(),
		Errors:      errors,
		Uptime:      time.Since(startTime),
	}
}

func (hc *HealthChecker) IsHealthy() bool {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()
	return hc.status.Status == "healthy"
}

func (hc *HealthChecker) IsDegraded() bool {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()
	return hc.status.Status == "degraded"
}

func (hc *HealthChecker) IsUnhealthy() bool {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()
	return hc.status.Status == "unhealthy"
}
