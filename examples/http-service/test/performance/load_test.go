package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/examples/http-service/internal/models"
)

const (
	testServerURL = "http://localhost:8080"
	testAPIKey    = "test-load-key"
)

// LoadTestConfig holds configuration for load tests
type LoadTestConfig struct {
	Duration     time.Duration
	Concurrency  int
	RequestsPerS int
	Endpoint     string
	Method       string
	Payload      interface{}
}

// LoadTestResult holds the results of a load test
type LoadTestResult struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalDuration      time.Duration
	AverageLatency     time.Duration
	MinLatency         time.Duration
	MaxLatency         time.Duration
	RequestsPerSecond  float64
	Errors             map[string]int64
}

func TestLoad_SendNotifications(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	payload := models.NotificationRequest{
		Title: "Load Test Notification",
		Body:  "This is a load test message",
		Targets: []models.TargetRequest{
			{
				Type:  "email",
				Value: "loadtest@example.com",
			},
		},
		Priority: 1,
	}

	config := LoadTestConfig{
		Duration:     30 * time.Second,
		Concurrency:  50,
		RequestsPerS: 100,
		Endpoint:     "/api/v1/notifications",
		Method:       http.MethodPost,
		Payload:      payload,
	}

	result := runLoadTest(t, config)

	// Print results
	t.Logf("Load Test Results:")
	t.Logf("Total Requests: %d", result.TotalRequests)
	t.Logf("Successful: %d (%.2f%%)", result.SuccessfulRequests,
		float64(result.SuccessfulRequests)/float64(result.TotalRequests)*100)
	t.Logf("Failed: %d (%.2f%%)", result.FailedRequests,
		float64(result.FailedRequests)/float64(result.TotalRequests)*100)
	t.Logf("Duration: %v", result.TotalDuration)
	t.Logf("Average Latency: %v", result.AverageLatency)
	t.Logf("Min Latency: %v", result.MinLatency)
	t.Logf("Max Latency: %v", result.MaxLatency)
	t.Logf("Requests/sec: %.2f", result.RequestsPerSecond)

	if len(result.Errors) > 0 {
		t.Logf("Errors:")
		for err, count := range result.Errors {
			t.Logf("  %s: %d", err, count)
		}
	}

	// Assert basic performance requirements
	if result.SuccessfulRequests == 0 {
		t.Error("No successful requests")
	}

	successRate := float64(result.SuccessfulRequests) / float64(result.TotalRequests)
	if successRate < 0.95 { // 95% success rate
		t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate*100)
	}

	if result.AverageLatency > 1*time.Second {
		t.Errorf("Average latency too high: %v (expected <= 1s)", result.AverageLatency)
	}

	if result.RequestsPerSecond < 50 {
		t.Errorf("Throughput too low: %.2f req/s (expected >= 50)", result.RequestsPerSecond)
	}
}

func TestLoad_BulkNotifications(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	payload := models.BulkNotificationRequest{
		Notifications: []models.NotificationRequest{
			{
				Title: "Bulk Load Test 1",
				Body:  "First bulk message",
				Targets: []models.TargetRequest{
					{Type: "email", Value: "bulk1@example.com"},
				},
			},
			{
				Title: "Bulk Load Test 2",
				Body:  "Second bulk message",
				Targets: []models.TargetRequest{
					{Type: "email", Value: "bulk2@example.com"},
				},
			},
		},
	}

	config := LoadTestConfig{
		Duration:     20 * time.Second,
		Concurrency:  20,
		RequestsPerS: 50,
		Endpoint:     "/api/v1/notifications/bulk",
		Method:       http.MethodPost,
		Payload:      payload,
	}

	result := runLoadTest(t, config)

	t.Logf("Bulk Load Test Results:")
	t.Logf("Total Requests: %d", result.TotalRequests)
	t.Logf("Successful: %d", result.SuccessfulRequests)
	t.Logf("Average Latency: %v", result.AverageLatency)
	t.Logf("Requests/sec: %.2f", result.RequestsPerSecond)

	// Bulk requests should handle lower throughput but higher latency
	if result.AverageLatency > 5*time.Second {
		t.Errorf("Bulk average latency too high: %v (expected <= 5s)", result.AverageLatency)
	}
}

func TestLoad_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		Duration:     10 * time.Second,
		Concurrency:  100,
		RequestsPerS: 500,
		Endpoint:     "/health",
		Method:       http.MethodGet,
		Payload:      nil,
	}

	result := runLoadTest(t, config)

	t.Logf("Health Check Load Test Results:")
	t.Logf("Total Requests: %d", result.TotalRequests)
	t.Logf("Successful: %d", result.SuccessfulRequests)
	t.Logf("Average Latency: %v", result.AverageLatency)
	t.Logf("Requests/sec: %.2f", result.RequestsPerSecond)

	// Health checks should be very fast
	if result.AverageLatency > 100*time.Millisecond {
		t.Errorf("Health check latency too high: %v (expected <= 100ms)", result.AverageLatency)
	}

	if result.RequestsPerSecond < 200 {
		t.Errorf("Health check throughput too low: %.2f req/s (expected >= 200)", result.RequestsPerSecond)
	}
}

func runLoadTest(t *testing.T, config LoadTestConfig) *LoadTestResult {
	var (
		totalRequests      int64
		successfulRequests int64
		failedRequests     int64
		totalLatency       int64
		minLatency         int64 = int64(time.Hour)
		maxLatency         int64
		errors             = make(map[string]int64)
		errorsMutex        sync.Mutex
	)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	// Rate limiter
	ticker := time.NewTicker(time.Second / time.Duration(config.RequestsPerS))
	defer ticker.Stop()

	var wg sync.WaitGroup

	// Worker goroutines
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					atomic.AddInt64(&totalRequests, 1)

					reqStart := time.Now()
					err := makeRequest(client, config)
					latency := time.Since(reqStart)

					// Update latency stats
					latencyNs := latency.Nanoseconds()
					atomic.AddInt64(&totalLatency, latencyNs)

					// Update min latency
					for {
						current := atomic.LoadInt64(&minLatency)
						if latencyNs >= current {
							break
						}
						if atomic.CompareAndSwapInt64(&minLatency, current, latencyNs) {
							break
						}
					}

					// Update max latency
					for {
						current := atomic.LoadInt64(&maxLatency)
						if latencyNs <= current {
							break
						}
						if atomic.CompareAndSwapInt64(&maxLatency, current, latencyNs) {
							break
						}
					}

					if err != nil {
						atomic.AddInt64(&failedRequests, 1)
						errorsMutex.Lock()
						errors[err.Error()]++
						errorsMutex.Unlock()
					} else {
						atomic.AddInt64(&successfulRequests, 1)
					}
				}
			}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	return &LoadTestResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		TotalDuration:      totalDuration,
		AverageLatency:     time.Duration(totalLatency / totalRequests),
		MinLatency:         time.Duration(minLatency),
		MaxLatency:         time.Duration(maxLatency),
		RequestsPerSecond:  float64(totalRequests) / totalDuration.Seconds(),
		Errors:             errors,
	}
}

func makeRequest(client *http.Client, config LoadTestConfig) error {
	var req *http.Request
	var err error

	url := testServerURL + config.Endpoint

	if config.Payload != nil {
		jsonPayload, err := json.Marshal(config.Payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}

		req, err = http.NewRequest(config.Method, url, bytes.NewBuffer(jsonPayload))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+testAPIKey)
	} else {
		req, err = http.NewRequest(config.Method, url, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// Stress test that gradually increases load
func TestStress_GradualIncrease(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	payload := models.NotificationRequest{
		Title: "Stress Test",
		Body:  "Stress test message",
		Targets: []models.TargetRequest{
			{Type: "email", Value: "stress@example.com"},
		},
	}

	// Test increasing load levels
	loadLevels := []struct {
		name        string
		concurrency int
		requestsPerS int
		duration    time.Duration
	}{
		{"Low Load", 10, 20, 10 * time.Second},
		{"Medium Load", 25, 50, 10 * time.Second},
		{"High Load", 50, 100, 10 * time.Second},
		{"Very High Load", 100, 200, 10 * time.Second},
	}

	for _, level := range loadLevels {
		t.Run(level.name, func(t *testing.T) {
			config := LoadTestConfig{
				Duration:     level.duration,
				Concurrency:  level.concurrency,
				RequestsPerS: level.requestsPerS,
				Endpoint:     "/api/v1/notifications",
				Method:       http.MethodPost,
				Payload:      payload,
			}

			result := runLoadTest(t, config)

			t.Logf("%s Results:", level.name)
			t.Logf("  Requests/sec: %.2f", result.RequestsPerSecond)
			t.Logf("  Success rate: %.2f%%",
				float64(result.SuccessfulRequests)/float64(result.TotalRequests)*100)
			t.Logf("  Average latency: %v", result.AverageLatency)

			// Gradually relaxed requirements as load increases
			successThreshold := 0.95 - (float64(level.concurrency)/100)*0.1 // Allow lower success rate at high load
			if successThreshold < 0.80 {
				successThreshold = 0.80 // Minimum 80% success rate
			}

			successRate := float64(result.SuccessfulRequests) / float64(result.TotalRequests)
			if successRate < successThreshold {
				t.Errorf("Success rate too low for %s: %.2f%% (expected >= %.2f%%)",
					level.name, successRate*100, successThreshold*100)
			}
		})

		// Brief pause between load levels
		time.Sleep(5 * time.Second)
	}
}

// Memory and resource usage monitoring during load test
func TestLoad_ResourceUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource usage test in short mode")
	}

	// This would require integration with monitoring tools
	// For now, we'll run a sustained load and check basic metrics

	payload := models.NotificationRequest{
		Title: "Resource Test",
		Body:  "Resource usage test",
		Targets: []models.TargetRequest{
			{Type: "email", Value: "resource@example.com"},
		},
	}

	config := LoadTestConfig{
		Duration:     60 * time.Second, // Longer test
		Concurrency:  30,
		RequestsPerS: 60,
		Endpoint:     "/api/v1/notifications",
		Method:       http.MethodPost,
		Payload:      payload,
	}

	result := runLoadTest(t, config)

	t.Logf("Resource Usage Test Results:")
	t.Logf("Duration: %v", result.TotalDuration)
	t.Logf("Total Requests: %d", result.TotalRequests)
	t.Logf("Requests/sec: %.2f", result.RequestsPerSecond)
	t.Logf("Average Latency: %v", result.AverageLatency)

	// Check for memory leaks or performance degradation
	// In a real scenario, you'd collect metrics from the server
	if result.AverageLatency > 2*time.Second {
		t.Errorf("Sustained load caused performance degradation: %v avg latency", result.AverageLatency)
	}
}

// Benchmark concurrent notification sending
func BenchmarkConcurrentNotifications(b *testing.B) {
	payload := models.NotificationRequest{
		Title: "Benchmark Test",
		Body:  "Benchmark message",
		Targets: []models.TargetRequest{
			{Type: "email", Value: "benchmark@example.com"},
		},
	}

	client := &http.Client{Timeout: 30 * time.Second}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jsonPayload, _ := json.Marshal(payload)
			req, _ := http.NewRequest(http.MethodPost, testServerURL+"/api/v1/notifications",
				bytes.NewBuffer(jsonPayload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+testAPIKey)

			resp, err := client.Do(req)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}