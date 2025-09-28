// Package errors provides retry strategies and mechanisms for NotifyHub
package errors

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// RetryStrategy defines the interface for retry strategies
type RetryStrategy interface {
	// ShouldRetry determines if an operation should be retried based on the error and attempt count
	ShouldRetry(err error, attempt int) bool
	// GetDelay calculates the delay before the next retry attempt
	GetDelay(attempt int) time.Duration
	// GetMaxAttempts returns the maximum number of retry attempts
	GetMaxAttempts() int
	// GetStatistics returns retry statistics
	GetStatistics() *RetryStatistics
	// Reset resets the strategy statistics
	Reset()
	// UpdatePlatformConfig updates platform-specific retry configuration
	UpdatePlatformConfig(platform string, config *PlatformRetryConfig) error
	// GetPlatformConfig gets platform-specific retry configuration
	GetPlatformConfig(platform string) *PlatformRetryConfig
}

// RetryableError defines an interface for errors that can provide custom retry behavior
type RetryableError interface {
	error
	// IsRetryable returns true if the error should be retried
	IsRetryable() bool
	// ShouldRetry returns true if the error should be retried for the given attempt
	ShouldRetry(attempt int) bool
	// GetRetryDelay returns a custom retry delay for this error
	GetRetryDelay(attempt int) time.Duration
	// GetRetryContext returns context information for retry decisions
	GetRetryContext() map[string]interface{}
}

// AdvancedRetryableError extends RetryableError with additional capabilities
type AdvancedRetryableError interface {
	RetryableError
	// GetPlatformRetryConfig returns platform-specific retry configuration
	GetPlatformRetryConfig() *PlatformRetryConfig
	// ShouldUseCircuitBreaker returns true if circuit breaker should be applied
	ShouldUseCircuitBreaker() bool
	// GetCircuitBreakerThreshold returns the failure threshold for circuit breaker
	GetCircuitBreakerThreshold() int
}

// PlatformRetryConfig holds platform-specific retry configuration
type PlatformRetryConfig struct {
	Platform    string        `json:"platform"`
	MaxAttempts int           `json:"max_attempts"`
	BaseDelay   time.Duration `json:"base_delay"`
	MaxDelay    time.Duration `json:"max_delay"`
	Multiplier  float64       `json:"multiplier"`
	Jitter      float64       `json:"jitter"`
	Enabled     bool          `json:"enabled"`
}

// RetryStatistics tracks retry performance and patterns
type RetryStatistics struct {
	mu                sync.RWMutex
	TotalAttempts     int64         `json:"total_attempts"`
	TotalRetries      int64         `json:"total_retries"`
	SuccessfulRetries int64         `json:"successful_retries"`
	FailedRetries     int64         `json:"failed_retries"`
	AverageDelay      time.Duration `json:"average_delay"`
	MinDelay          time.Duration `json:"min_delay"`
	MaxDelay          time.Duration `json:"max_delay"`
	ErrorCounts       map[Code]int64 `json:"error_counts"`
	PlatformStats     map[string]*PlatformRetryStats `json:"platform_stats"`
	LastRetryTime     time.Time     `json:"last_retry_time"`
	StartTime         time.Time     `json:"start_time"`
}

// PlatformRetryStats tracks retry statistics per platform
type PlatformRetryStats struct {
	Platform          string        `json:"platform"`
	Attempts          int64         `json:"attempts"`
	Retries           int64         `json:"retries"`
	Successes         int64         `json:"successes"`
	Failures          int64         `json:"failures"`
	AverageDelay      time.Duration `json:"average_delay"`
	CircuitBreakerTrips int64       `json:"circuit_breaker_trips"`
}

// NewRetryStatistics creates a new retry statistics tracker
func NewRetryStatistics() *RetryStatistics {
	return &RetryStatistics{
		ErrorCounts:   make(map[Code]int64),
		PlatformStats: make(map[string]*PlatformRetryStats),
		StartTime:     time.Now(),
		MinDelay:      time.Duration(0),
		MaxDelay:      time.Duration(0),
	}
}

// RecordAttempt records a retry attempt
func (s *RetryStatistics) RecordAttempt(platform string, delay time.Duration, err error, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalAttempts++
	s.LastRetryTime = time.Now()

	// Update delay statistics
	if s.MinDelay == 0 || delay < s.MinDelay {
		s.MinDelay = delay
	}
	if delay > s.MaxDelay {
		s.MaxDelay = delay
	}

	// Calculate running average delay
	if s.TotalRetries > 0 {
		s.AverageDelay = time.Duration((int64(s.AverageDelay)*s.TotalRetries + int64(delay)) / (s.TotalRetries + 1))
	} else {
		s.AverageDelay = delay
	}

	if delay > 0 {
		s.TotalRetries++
		if success {
			s.SuccessfulRetries++
		} else {
			s.FailedRetries++
		}
	}

	// Track error codes
	if err != nil {
		if notifyErr, ok := err.(*NotifyError); ok {
			s.ErrorCounts[notifyErr.Code]++
		}
	}

	// Track platform statistics
	if platform != "" {
		if _, exists := s.PlatformStats[platform]; !exists {
			s.PlatformStats[platform] = &PlatformRetryStats{
				Platform: platform,
			}
		}
		pStats := s.PlatformStats[platform]
		pStats.Attempts++
		if delay > 0 {
			pStats.Retries++
			if pStats.AverageDelay == 0 {
				pStats.AverageDelay = delay
			} else {
				pStats.AverageDelay = time.Duration((int64(pStats.AverageDelay)*pStats.Retries + int64(delay)) / (pStats.Retries + 1))
			}
		}
		if success {
			pStats.Successes++
		} else {
			pStats.Failures++
		}
	}
}

// GetSuccessRate returns the overall success rate
func (s *RetryStatistics) GetSuccessRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TotalAttempts == 0 {
		return 0.0
	}
	return float64(s.TotalAttempts-s.FailedRetries) / float64(s.TotalAttempts)
}

// GetRetryRate returns the retry rate
func (s *RetryStatistics) GetRetryRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TotalAttempts == 0 {
		return 0.0
	}
	return float64(s.TotalRetries) / float64(s.TotalAttempts)
}

// Reset resets all statistics
func (s *RetryStatistics) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalAttempts = 0
	s.TotalRetries = 0
	s.SuccessfulRetries = 0
	s.FailedRetries = 0
	s.AverageDelay = 0
	s.MinDelay = 0
	s.MaxDelay = 0
	s.ErrorCounts = make(map[Code]int64)
	s.PlatformStats = make(map[string]*PlatformRetryStats)
	s.StartTime = time.Now()
}

// JitterType defines the type of jitter algorithm to use
type JitterType int

const (
	// UniformJitter applies uniform random jitter
	UniformJitter JitterType = iota
	// FullJitter applies full random jitter
	FullJitter
	// ExponentialJitter applies exponential jitter
	ExponentialJitter
	// DecorrelatedJitter applies decorrelated jitter
	DecorrelatedJitter
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	// CircuitClosed - normal operation
	CircuitClosed CircuitBreakerState = iota
	// CircuitOpen - circuit is open, failing fast
	CircuitOpen
	// CircuitHalfOpen - testing if service has recovered
	CircuitHalfOpen
)

// CircuitBreaker implements circuit breaker pattern for retry strategies
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitBreakerState
	failureCount    int64
	lastFailureTime time.Time
	nextRetryTime   time.Time
	threshold       int64
	timeout         time.Duration
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int64, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:     CircuitClosed,
		threshold: threshold,
		timeout:   timeout,
	}
}

// ShouldAllow determines if an operation should be allowed
func (cb *CircuitBreaker) ShouldAllow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		return time.Now().After(cb.nextRetryTime)
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == CircuitClosed && cb.failureCount >= cb.threshold {
		cb.state = CircuitOpen
		cb.nextRetryTime = time.Now().Add(cb.timeout)
	} else if cb.state == CircuitHalfOpen {
		cb.state = CircuitOpen
		cb.nextRetryTime = time.Now().Add(cb.timeout)
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == CircuitOpen && time.Now().After(cb.nextRetryTime) {
		cb.mu.RUnlock()
		cb.mu.Lock()
		if cb.state == CircuitOpen && time.Now().After(cb.nextRetryTime) {
			cb.state = CircuitHalfOpen
		}
		cb.mu.Unlock()
		cb.mu.RLock()
	}

	return cb.state
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failureCount = 0
	cb.lastFailureTime = time.Time{}
	cb.nextRetryTime = time.Time{}
}

// PerformanceMonitor tracks retry performance and optimizes strategies
type PerformanceMonitor struct {
	mu                sync.RWMutex
	PlatformMetrics   map[string]*PlatformPerformanceMetrics
	OverallMetrics    *OverallPerformanceMetrics
	OptimizationRules []OptimizationRule
}

// PlatformPerformanceMetrics tracks performance metrics per platform
type PlatformPerformanceMetrics struct {
	Platform           string        `json:"platform"`
	TotalOperations    int64         `json:"total_operations"`
	SuccessfulOps      int64         `json:"successful_ops"`
	FailedOps          int64         `json:"failed_ops"`
	AverageLatency     time.Duration `json:"average_latency"`
	MinLatency         time.Duration `json:"min_latency"`
	MaxLatency         time.Duration `json:"max_latency"`
	P50Latency         time.Duration `json:"p50_latency"`
	P95Latency         time.Duration `json:"p95_latency"`
	P99Latency         time.Duration `json:"p99_latency"`
	SuccessRate        float64       `json:"success_rate"`
	ThroughputPerMin   float64       `json:"throughput_per_min"`
	LastUpdated        time.Time     `json:"last_updated"`
	LatencyHistory     []time.Duration // Rolling window of latencies
}

// OverallPerformanceMetrics tracks overall retry system performance
type OverallPerformanceMetrics struct {
	TotalRetryOperations   int64         `json:"total_retry_operations"`
	SuccessfulRetries      int64         `json:"successful_retries"`
	FailedRetries          int64         `json:"failed_retries"`
	AverageRetryLatency    time.Duration `json:"average_retry_latency"`
	OverallSuccessRate     float64       `json:"overall_success_rate"`
	MostReliablePlatform   string        `json:"most_reliable_platform"`
	LeastReliablePlatform  string        `json:"least_reliable_platform"`
	RecommendedStrategy    string        `json:"recommended_strategy"`
	LastOptimized          time.Time     `json:"last_optimized"`
}

// OptimizationRule defines a rule for strategy optimization
type OptimizationRule struct {
	Name      string                              `json:"name"`
	Condition func(*PlatformPerformanceMetrics) bool
	Action    func(*PlatformRetryConfig) *PlatformRetryConfig
	Priority  int `json:"priority"`
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		PlatformMetrics: make(map[string]*PlatformPerformanceMetrics),
		OverallMetrics:  &OverallPerformanceMetrics{},
		OptimizationRules: []OptimizationRule{
			{
				Name:     "HighFailureRateRule",
				Priority: 1,
				Condition: func(metrics *PlatformPerformanceMetrics) bool {
					return metrics.SuccessRate < 0.5 && metrics.TotalOperations > 10
				},
				Action: func(config *PlatformRetryConfig) *PlatformRetryConfig {
					// Increase retry attempts and delay for problematic platforms
					newConfig := *config
					newConfig.MaxAttempts = int(math.Min(float64(config.MaxAttempts*2), 10))
					newConfig.BaseDelay = time.Duration(float64(config.BaseDelay) * 1.5)
					return &newConfig
				},
			},
			{
				Name:     "HighLatencyRule",
				Priority: 2,
				Condition: func(metrics *PlatformPerformanceMetrics) bool {
					return metrics.P95Latency > 10*time.Second && metrics.TotalOperations > 5
				},
				Action: func(config *PlatformRetryConfig) *PlatformRetryConfig {
					// Reduce retry frequency for high-latency platforms
					newConfig := *config
					newConfig.BaseDelay = time.Duration(float64(config.BaseDelay) * 2)
					newConfig.MaxDelay = time.Duration(float64(config.MaxDelay) * 1.5)
					return &newConfig
				},
			},
		},
	}
}

// RecordOperation records a retry operation
func (pm *PerformanceMonitor) RecordOperation(platform string, latency time.Duration, success bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Initialize platform metrics if not exists
	if _, exists := pm.PlatformMetrics[platform]; !exists {
		pm.PlatformMetrics[platform] = &PlatformPerformanceMetrics{
			Platform:       platform,
			MinLatency:     latency,
			MaxLatency:     latency,
			LatencyHistory: make([]time.Duration, 0, 100), // Keep last 100 latencies
		}
	}

	metrics := pm.PlatformMetrics[platform]
	metrics.TotalOperations++
	metrics.LastUpdated = time.Now()

	// Update latency metrics
	metrics.LatencyHistory = append(metrics.LatencyHistory, latency)
	if len(metrics.LatencyHistory) > 100 {
		metrics.LatencyHistory = metrics.LatencyHistory[1:] // Keep rolling window
	}

	// Calculate latency statistics
	pm.updateLatencyStats(metrics)

	// Update success/failure counts
	if success {
		metrics.SuccessfulOps++
	} else {
		metrics.FailedOps++
	}

	// Update success rate
	metrics.SuccessRate = float64(metrics.SuccessfulOps) / float64(metrics.TotalOperations)

	// Update overall metrics
	pm.OverallMetrics.TotalRetryOperations++
	if success {
		pm.OverallMetrics.SuccessfulRetries++
	} else {
		pm.OverallMetrics.FailedRetries++
	}

	pm.OverallMetrics.OverallSuccessRate = float64(pm.OverallMetrics.SuccessfulRetries) / float64(pm.OverallMetrics.TotalRetryOperations)
}

// updateLatencyStats calculates latency percentiles and statistics
func (pm *PerformanceMonitor) updateLatencyStats(metrics *PlatformPerformanceMetrics) {
	if len(metrics.LatencyHistory) == 0 {
		return
	}

	// Sort latencies for percentile calculation
	latencies := make([]time.Duration, len(metrics.LatencyHistory))
	copy(latencies, metrics.LatencyHistory)

	// Simple bubble sort for small arrays
	for i := 0; i < len(latencies); i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	// Calculate percentiles
	n := len(latencies)
	metrics.MinLatency = latencies[0]
	metrics.MaxLatency = latencies[n-1]
	metrics.P50Latency = latencies[n/2]
	metrics.P95Latency = latencies[int(float64(n)*0.95)]
	metrics.P99Latency = latencies[int(float64(n)*0.99)]

	// Calculate average
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	metrics.AverageLatency = total / time.Duration(n)
}

// OptimizeStrategy analyzes performance and suggests optimizations
func (pm *PerformanceMonitor) OptimizeStrategy(platform string, currentConfig *PlatformRetryConfig) *PlatformRetryConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metrics, exists := pm.PlatformMetrics[platform]
	if !exists {
		return currentConfig // No data to optimize
	}

	optimizedConfig := *currentConfig

	// Apply optimization rules in priority order
	for _, rule := range pm.OptimizationRules {
		if rule.Condition(metrics) {
			optimizedConfig = *rule.Action(&optimizedConfig)
		}
	}

	return &optimizedConfig
}

// GetRecommendations returns performance-based recommendations
func (pm *PerformanceMonitor) GetRecommendations() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var recommendations []string

	for platform, metrics := range pm.PlatformMetrics {
		if metrics.SuccessRate < 0.7 && metrics.TotalOperations > 10 {
			recommendations = append(recommendations,
				fmt.Sprintf("Platform %s has low success rate (%.2f%%). Consider increasing retry attempts or reviewing platform configuration.",
					platform, metrics.SuccessRate*100))
		}

		if metrics.P95Latency > 30*time.Second {
			recommendations = append(recommendations,
				fmt.Sprintf("Platform %s has high latency (P95: %v). Consider optimizing retry delays or platform-specific timeouts.",
					platform, metrics.P95Latency))
		}
	}

	return recommendations
}

// Reset resets all performance metrics
func (pm *PerformanceMonitor) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.PlatformMetrics = make(map[string]*PlatformPerformanceMetrics)
	pm.OverallMetrics = &OverallPerformanceMetrics{}
}

// ExponentialBackoffStrategy implements exponential backoff with jitter
type ExponentialBackoffStrategy struct {
	BaseDelay         time.Duration            // Base delay for first retry
	MaxDelay          time.Duration            // Maximum delay between retries
	Multiplier        float64                  // Exponential multiplier (typically 2.0)
	Jitter            float64                  // Jitter factor (0.0 to 1.0)
	JitterType        JitterType               // Type of jitter algorithm
	MaxAttempts       int                      // Maximum retry attempts
	RetryableCodes    map[Code]bool            // Map of retryable error codes
	PlatformConfigs   map[string]*PlatformRetryConfig // Platform-specific configurations
	CircuitBreakers   map[string]*CircuitBreaker // Platform-specific circuit breakers
	Statistics        *RetryStatistics         // Retry statistics tracker
	ConditionalPolicies []ConditionalRetryPolicy // Conditional retry policies
	lastDelayUsed     time.Duration            // Last delay used (for decorrelated jitter)
	PerformanceMonitor *PerformanceMonitor     // Performance monitoring and optimization
	mu                sync.RWMutex             // Mutex for thread safety
}

// ConditionalRetryPolicy defines a conditional retry policy
type ConditionalRetryPolicy struct {
	Name      string                                    `json:"name"`
	Condition func(err error, attempt int) bool       `json:"-"`
	Action    func(err error, attempt int) RetryAction `json:"-"`
}

// RetryAction defines the action to take for a retry
type RetryAction struct {
	ShouldRetry bool          `json:"should_retry"`
	Delay       time.Duration `json:"delay"`
	MaxAttempts int           `json:"max_attempts"`
}

// NewExponentialBackoffStrategy creates a new exponential backoff strategy
func NewExponentialBackoffStrategy() *ExponentialBackoffStrategy {
	retryableCodes := make(map[Code]bool)

	// Platform errors that are retryable
	retryableCodes[ErrPlatformUnavailable] = true
	retryableCodes[ErrPlatformRateLimit] = true
	retryableCodes[ErrPlatformTimeout] = true
	retryableCodes[ErrPlatformInternal] = true
	retryableCodes[ErrPlatformMaintenance] = true

	// Network errors that are retryable
	retryableCodes[ErrNetworkTimeout] = true
	retryableCodes[ErrNetworkConnection] = true
	retryableCodes[ErrNetworkDNS] = true

	// Queue errors that are retryable
	retryableCodes[ErrQueueFull] = true
	retryableCodes[ErrQueueTimeout] = true
	retryableCodes[ErrQueueConnection] = true
	retryableCodes[ErrQueueWorkerFailed] = true

	// System errors that are retryable
	retryableCodes[ErrSystemUnavailable] = true
	retryableCodes[ErrResourceExhausted] = true
	retryableCodes[ErrSystemTimeout] = true
	retryableCodes[ErrSystemOverload] = true

	// Message errors that are retryable
	retryableCodes[ErrMessageSendFailed] = true
	retryableCodes[ErrMessageTimeout] = true

	// Template errors that are retryable
	retryableCodes[ErrTemplateCacheError] = true

	return &ExponentialBackoffStrategy{
		BaseDelay:       1 * time.Second,
		MaxDelay:        30 * time.Second,
		Multiplier:      2.0,
		Jitter:          0.1,
		JitterType:      UniformJitter,
		MaxAttempts:     5,
		RetryableCodes:  retryableCodes,
		PlatformConfigs: make(map[string]*PlatformRetryConfig),
		CircuitBreakers: make(map[string]*CircuitBreaker),
		Statistics:      NewRetryStatistics(),
		PerformanceMonitor: NewPerformanceMonitor(),
	}
}

// ShouldRetry determines if an operation should be retried
func (s *ExponentialBackoffStrategy) ShouldRetry(err error, attempt int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if attempt >= s.MaxAttempts {
		return false
	}

	// Check if it's a NotifyError with a retryable code
	if notifyErr, ok := err.(*NotifyError); ok {
		// Check platform-specific configuration first
		if platformConfig, exists := s.PlatformConfigs[notifyErr.Platform]; exists {
			if !platformConfig.Enabled {
				return false
			}
			if attempt >= platformConfig.MaxAttempts {
				return false
			}
		}

		// Check circuit breaker for the platform
		if cb, exists := s.CircuitBreakers[notifyErr.Platform]; exists {
			if !cb.ShouldAllow() {
				return false
			}
		}

		// Check conditional policies
		for _, policy := range s.ConditionalPolicies {
			if policy.Condition(err, attempt) {
				action := policy.Action(err, attempt)
				return action.ShouldRetry
			}
		}

		return s.RetryableCodes[notifyErr.Code]
	}

	// For non-NotifyError, don't retry by default
	return false
}

// GetDelay calculates the delay with exponential backoff and jitter
func (s *ExponentialBackoffStrategy) GetDelay(attempt int) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	if attempt <= 0 {
		return s.BaseDelay
	}

	// Calculate exponential delay
	delay := float64(s.BaseDelay) * math.Pow(s.Multiplier, float64(attempt))

	// Apply maximum delay limit
	if delay > float64(s.MaxDelay) {
		delay = float64(s.MaxDelay)
	}

	// Apply jitter based on strategy type
	if s.Jitter > 0 {
		switch s.JitterType {
		case UniformJitter:
			jitterAmount := delay * s.Jitter
			jitter := rand.Float64()*2*jitterAmount - jitterAmount
			delay += jitter
		case FullJitter:
			delay = rand.Float64() * delay
		case ExponentialJitter:
			delay = delay * (1 + s.Jitter*rand.Float64())
		case DecorrelatedJitter:
			if s.lastDelayUsed > 0 {
				delay = float64(s.BaseDelay) + rand.Float64()*float64(s.lastDelayUsed*3)
			} else {
				delay = float64(s.BaseDelay)
			}
		}
	}

	// Ensure delay is not negative
	if delay < 0 {
		delay = float64(s.BaseDelay)
	}

	finalDelay := time.Duration(delay)
	s.lastDelayUsed = finalDelay
	return finalDelay
}

// GetMaxAttempts returns the maximum number of retry attempts
func (s *ExponentialBackoffStrategy) GetMaxAttempts() int {
	return s.MaxAttempts
}

// GetStatistics returns retry statistics
func (s *ExponentialBackoffStrategy) GetStatistics() *RetryStatistics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Statistics
}

// Reset resets the strategy statistics
func (s *ExponentialBackoffStrategy) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Statistics.Reset()
	for _, cb := range s.CircuitBreakers {
		cb.Reset()
	}
	s.PerformanceMonitor.Reset()
}

// UpdatePlatformConfig updates platform-specific retry configuration
func (s *ExponentialBackoffStrategy) UpdatePlatformConfig(platform string, config *PlatformRetryConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config == nil {
		return fmt.Errorf("platform config cannot be nil")
	}

	s.PlatformConfigs[platform] = config

	// Create or update circuit breaker for platform
	if config.Enabled {
		threshold := int64(config.MaxAttempts)
		timeout := config.MaxDelay
		s.CircuitBreakers[platform] = NewCircuitBreaker(threshold, timeout)
	}

	return nil
}

// GetPlatformConfig gets platform-specific retry configuration
func (s *ExponentialBackoffStrategy) GetPlatformConfig(platform string) *PlatformRetryConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PlatformConfigs[platform]
}

// RecordAttempt records a retry attempt with platform context
func (s *ExponentialBackoffStrategy) RecordAttempt(platform string, delay time.Duration, err error, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Record in statistics
	s.Statistics.RecordAttempt(platform, delay, err, success)

	// Update circuit breaker
	if cb, exists := s.CircuitBreakers[platform]; exists {
		if success {
			cb.RecordSuccess()
		} else {
			cb.RecordFailure()
			if s.Statistics.PlatformStats[platform] != nil {
				s.Statistics.PlatformStats[platform].CircuitBreakerTrips++
			}
		}
	}

	// Update performance monitor
	s.PerformanceMonitor.RecordOperation(platform, delay, success)
}

// LinearBackoffStrategy implements linear backoff
type LinearBackoffStrategy struct {
	BaseDelay      time.Duration
	DelayIncrement time.Duration
	MaxDelay       time.Duration
	MaxAttempts    int
	RetryableCodes map[Code]bool
	Statistics     *RetryStatistics
}

// NewLinearBackoffStrategy creates a new linear backoff strategy
func NewLinearBackoffStrategy() *LinearBackoffStrategy {
	strategy := NewExponentialBackoffStrategy()
	return &LinearBackoffStrategy{
		BaseDelay:      2 * time.Second,
		DelayIncrement: 1 * time.Second,
		MaxDelay:       20 * time.Second,
		MaxAttempts:    3,
		RetryableCodes: strategy.RetryableCodes,
		Statistics:     NewRetryStatistics(),
	}
}

// ShouldRetry determines if an operation should be retried
func (s *LinearBackoffStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= s.MaxAttempts {
		return false
	}

	if notifyErr, ok := err.(*NotifyError); ok {
		return s.RetryableCodes[notifyErr.Code]
	}

	return false
}

// GetDelay calculates the delay with linear backoff
func (s *LinearBackoffStrategy) GetDelay(attempt int) time.Duration {
	delay := s.BaseDelay + time.Duration(attempt)*s.DelayIncrement
	if delay > s.MaxDelay {
		delay = s.MaxDelay
	}
	return delay
}

// GetMaxAttempts returns the maximum number of retry attempts
func (s *LinearBackoffStrategy) GetMaxAttempts() int {
	return s.MaxAttempts
}

// GetStatistics returns retry statistics
func (s *LinearBackoffStrategy) GetStatistics() *RetryStatistics {
	return s.Statistics
}

// Reset resets the strategy statistics
func (s *LinearBackoffStrategy) Reset() {
	s.Statistics.Reset()
}

// UpdatePlatformConfig updates platform-specific retry configuration (no-op for linear strategy)
func (s *LinearBackoffStrategy) UpdatePlatformConfig(platform string, config *PlatformRetryConfig) error {
	// Linear strategy doesn't support platform-specific configs
	return nil
}

// GetPlatformConfig gets platform-specific retry configuration (no-op for linear strategy)
func (s *LinearBackoffStrategy) GetPlatformConfig(platform string) *PlatformRetryConfig {
	// Linear strategy doesn't support platform-specific configs
	return nil
}

// FixedDelayStrategy implements fixed delay between retries
type FixedDelayStrategy struct {
	Delay          time.Duration
	MaxAttempts    int
	RetryableCodes map[Code]bool
	Statistics     *RetryStatistics
}

// NewFixedDelayStrategy creates a new fixed delay strategy
func NewFixedDelayStrategy(delay time.Duration, maxAttempts int) *FixedDelayStrategy {
	strategy := NewExponentialBackoffStrategy()
	return &FixedDelayStrategy{
		Delay:          delay,
		MaxAttempts:    maxAttempts,
		RetryableCodes: strategy.RetryableCodes,
		Statistics:     NewRetryStatistics(),
	}
}

// ShouldRetry determines if an operation should be retried
func (s *FixedDelayStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= s.MaxAttempts {
		return false
	}

	if notifyErr, ok := err.(*NotifyError); ok {
		return s.RetryableCodes[notifyErr.Code]
	}

	return false
}

// GetDelay returns the fixed delay
func (s *FixedDelayStrategy) GetDelay(attempt int) time.Duration {
	return s.Delay
}

// GetMaxAttempts returns the maximum number of retry attempts
func (s *FixedDelayStrategy) GetMaxAttempts() int {
	return s.MaxAttempts
}

// GetStatistics returns retry statistics
func (s *FixedDelayStrategy) GetStatistics() *RetryStatistics {
	return s.Statistics
}

// Reset resets the strategy statistics
func (s *FixedDelayStrategy) Reset() {
	s.Statistics.Reset()
}

// UpdatePlatformConfig updates platform-specific retry configuration (no-op for fixed strategy)
func (s *FixedDelayStrategy) UpdatePlatformConfig(platform string, config *PlatformRetryConfig) error {
	// Fixed delay strategy doesn't support platform-specific configs
	return nil
}

// GetPlatformConfig gets platform-specific retry configuration (no-op for fixed strategy)
func (s *FixedDelayStrategy) GetPlatformConfig(platform string) *PlatformRetryConfig {
	// Fixed delay strategy doesn't support platform-specific configs
	return nil
}

// RetryExecutor executes operations with retry logic
type RetryExecutor struct {
	strategy RetryStrategy
	logger   logger.Logger
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(strategy RetryStrategy, logger logger.Logger) *RetryExecutor {
	return &RetryExecutor{
		strategy: strategy,
		logger:   logger,
	}
}

// Execute executes a function with retry logic
func (r *RetryExecutor) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < r.strategy.GetMaxAttempts(); attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return NewSystemError(ErrSystemTimeout, "retry_executor", "context cancelled during retry")
		default:
		}

		// Execute the operation
		if err := operation(); err != nil {
			lastErr = err

			// Check if we should retry
			if !r.strategy.ShouldRetry(err, attempt) {
				r.logger.Debug("Operation failed, not retryable",
					"attempt", attempt+1,
					"error", err.Error())
				return err
			}

			// Calculate delay for next attempt
			delay := r.strategy.GetDelay(attempt)

			r.logger.Warn("Operation failed, retrying",
				"attempt", attempt+1,
				"max_attempts", r.strategy.GetMaxAttempts(),
				"delay", delay,
				"error", err.Error())

			// Wait for delay or context cancellation
			select {
			case <-ctx.Done():
				return NewSystemError(ErrSystemTimeout, "retry_executor", "context cancelled during retry delay")
			case <-time.After(delay):
				// Continue to next attempt
			}
		} else {
			// Operation succeeded
			if attempt > 0 {
				r.logger.Info("Operation succeeded after retries", "attempts", attempt+1)
			}
			return nil
		}
	}

	// All retry attempts exhausted
	r.logger.Error("Operation failed after all retry attempts",
		"attempts", r.strategy.GetMaxAttempts(),
		"last_error", lastErr.Error())

	return Wrap(lastErr, ErrSystemTimeout, "operation failed after all retry attempts")
}

// ExecuteWithCallback executes a function with retry logic and callbacks
func (r *RetryExecutor) ExecuteWithCallback(
	ctx context.Context,
	operation func() error,
	onRetry func(attempt int, err error, delay time.Duration),
) error {
	var lastErr error

	for attempt := 0; attempt < r.strategy.GetMaxAttempts(); attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return NewSystemError(ErrSystemTimeout, "retry_executor", "context cancelled during retry")
		default:
		}

		// Execute the operation
		if err := operation(); err != nil {
			lastErr = err

			// Check if we should retry
			if !r.strategy.ShouldRetry(err, attempt) {
				return err
			}

			// Calculate delay for next attempt
			delay := r.strategy.GetDelay(attempt)

			// Call retry callback if provided
			if onRetry != nil {
				onRetry(attempt, err, delay)
			}

			// Wait for delay or context cancellation
			select {
			case <-ctx.Done():
				return NewSystemError(ErrSystemTimeout, "retry_executor", "context cancelled during retry delay")
			case <-time.After(delay):
				// Continue to next attempt
			}
		} else {
			// Operation succeeded
			return nil
		}
	}

	// All retry attempts exhausted
	return Wrap(lastErr, ErrSystemTimeout, "operation failed after all retry attempts")
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	Strategy    string        `json:"strategy" yaml:"strategy"`         // "exponential", "linear", "fixed"
	BaseDelay   time.Duration `json:"base_delay" yaml:"base_delay"`     // Base delay
	MaxDelay    time.Duration `json:"max_delay" yaml:"max_delay"`       // Maximum delay
	MaxAttempts int           `json:"max_attempts" yaml:"max_attempts"` // Maximum attempts
	Multiplier  float64       `json:"multiplier" yaml:"multiplier"`     // Exponential multiplier
	Jitter      float64       `json:"jitter" yaml:"jitter"`             // Jitter factor
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Strategy:    "exponential",
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		MaxAttempts: 5,
		Multiplier:  2.0,
		Jitter:      0.1,
	}
}

// CreateRetryStrategy creates a retry strategy from configuration
func CreateRetryStrategy(config RetryConfig) RetryStrategy {
	switch config.Strategy {
	case "linear":
		strategy := NewLinearBackoffStrategy()
		strategy.BaseDelay = config.BaseDelay
		strategy.MaxDelay = config.MaxDelay
		strategy.MaxAttempts = config.MaxAttempts
		return strategy

	case "fixed":
		return NewFixedDelayStrategy(config.BaseDelay, config.MaxAttempts)

	default: // "exponential" or unknown defaults to exponential
		strategy := NewExponentialBackoffStrategy()
		strategy.BaseDelay = config.BaseDelay
		strategy.MaxDelay = config.MaxDelay
		strategy.MaxAttempts = config.MaxAttempts
		strategy.Multiplier = config.Multiplier
		strategy.Jitter = config.Jitter
		return strategy
	}
}
