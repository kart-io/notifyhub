package monitoring

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()
	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.SendsByPlatform)
	assert.NotNil(t, metrics.FailsByPlatform)
	assert.NotNil(t, metrics.LastErrors)
	assert.NotNil(t, metrics.PlatformHealth)
	assert.Equal(t, int64(0), metrics.TotalSent)
	assert.Equal(t, int64(0), metrics.TotalFailed)
	assert.False(t, metrics.StartTime.IsZero())
}

func TestRecordSend(t *testing.T) {
	metrics := NewMetrics()

	// Test successful send
	metrics.RecordSend("feishu", true, 100*time.Millisecond, "")
	assert.Equal(t, int64(1), metrics.TotalSent)
	assert.Equal(t, int64(0), metrics.TotalFailed)
	assert.Equal(t, int64(1), metrics.SendsByPlatform["feishu"])
	assert.Equal(t, 100*time.Millisecond, metrics.AvgDuration)
	assert.Equal(t, 100*time.Millisecond, metrics.MaxDuration)

	// Test failed send
	metrics.RecordSend("email", false, 50*time.Millisecond, "connection timeout")
	assert.Equal(t, int64(1), metrics.TotalSent)
	assert.Equal(t, int64(1), metrics.TotalFailed)
	assert.Equal(t, int64(1), metrics.FailsByPlatform["email"])
	assert.Equal(t, "connection timeout", metrics.LastErrors["email"])
	
	// Average duration should be updated: (100 + 50) / 2 = 75ms
	assert.Equal(t, 75*time.Millisecond, metrics.AvgDuration)
	assert.Equal(t, 100*time.Millisecond, metrics.MaxDuration) // Max unchanged

	// Test another successful send with longer duration
	metrics.RecordSend("slack", true, 200*time.Millisecond, "")
	assert.Equal(t, int64(2), metrics.TotalSent)
	assert.Equal(t, int64(1), metrics.TotalFailed)
	assert.Equal(t, int64(1), metrics.SendsByPlatform["slack"])
	assert.Equal(t, 200*time.Millisecond, metrics.MaxDuration) // Max updated
}

func TestRecordHealth(t *testing.T) {
	metrics := NewMetrics()

	// Record health status for various platforms
	metrics.RecordHealth("feishu", true)
	metrics.RecordHealth("email", false)
	metrics.RecordHealth("slack", true)

	assert.True(t, metrics.PlatformHealth["feishu"])
	assert.False(t, metrics.PlatformHealth["email"])
	assert.True(t, metrics.PlatformHealth["slack"])

	// Update health status
	metrics.RecordHealth("email", true)
	assert.True(t, metrics.PlatformHealth["email"])
}

func TestGetSuccessRate(t *testing.T) {
	metrics := NewMetrics()

	// Initial success rate should be 1.0 (no sends)
	assert.Equal(t, 1.0, metrics.GetSuccessRate())

	// Record some successful sends
	metrics.RecordSend("feishu", true, 100*time.Millisecond, "")
	metrics.RecordSend("email", true, 150*time.Millisecond, "")
	assert.Equal(t, 1.0, metrics.GetSuccessRate())

	// Record a failed send
	metrics.RecordSend("slack", false, 50*time.Millisecond, "timeout")
	expectedRate := 2.0 / 3.0 // 2 successful out of 3 total
	assert.InDelta(t, expectedRate, metrics.GetSuccessRate(), 0.001)

	// Record another failed send
	metrics.RecordSend("webhook", false, 75*time.Millisecond, "error")
	expectedRate = 2.0 / 4.0 // 2 successful out of 4 total
	assert.Equal(t, expectedRate, metrics.GetSuccessRate())
}

func TestGetUptime(t *testing.T) {
	metrics := NewMetrics()
	
	// Sleep a small amount to ensure uptime > 0
	time.Sleep(10 * time.Millisecond)
	
	uptime := metrics.GetUptime()
	assert.True(t, uptime > 0)
	assert.True(t, uptime < time.Second) // Should be very small
}

func TestGetSnapshot(t *testing.T) {
	metrics := NewMetrics()

	// Record some data
	metrics.RecordSend("feishu", true, 100*time.Millisecond, "")
	metrics.RecordSend("email", false, 50*time.Millisecond, "error")
	metrics.RecordHealth("feishu", true)
	metrics.RecordHealth("email", false)

	snapshot := metrics.GetSnapshot()

	// Verify snapshot contains expected keys
	assert.Contains(t, snapshot, "total_sent")
	assert.Contains(t, snapshot, "total_failed")
	assert.Contains(t, snapshot, "success_rate")
	assert.Contains(t, snapshot, "sends_by_platform")
	assert.Contains(t, snapshot, "fails_by_platform")
	assert.Contains(t, snapshot, "last_errors")
	assert.Contains(t, snapshot, "avg_duration")
	assert.Contains(t, snapshot, "max_duration")
	assert.Contains(t, snapshot, "platform_health")
	assert.Contains(t, snapshot, "uptime")

	// Verify snapshot values
	assert.Equal(t, int64(1), snapshot["total_sent"])
	assert.Equal(t, int64(1), snapshot["total_failed"])
	assert.Equal(t, 0.5, snapshot["success_rate"])

	sendsByPlatform := snapshot["sends_by_platform"].(map[string]int64)
	assert.Equal(t, int64(1), sendsByPlatform["feishu"])

	failsByPlatform := snapshot["fails_by_platform"].(map[string]int64)
	assert.Equal(t, int64(1), failsByPlatform["email"])

	lastErrors := snapshot["last_errors"].(map[string]string)
	assert.Equal(t, "error", lastErrors["email"])

	platformHealth := snapshot["platform_health"].(map[string]bool)
	assert.True(t, platformHealth["feishu"])
	assert.False(t, platformHealth["email"])

	assert.Equal(t, "75ms", snapshot["avg_duration"])
	assert.Equal(t, "100ms", snapshot["max_duration"])
	assert.IsType(t, "", snapshot["uptime"])
}

func TestMetricsConcurrency(t *testing.T) {
	metrics := NewMetrics()
	const numGoroutines = 50
	const operationsPerGoroutine = 20

	var wg sync.WaitGroup

	// Test concurrent RecordSend operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				platform := "platform" + string(rune('0'+id%5)) // 5 different platforms
				success := j%2 == 0                              // Alternate success/failure
				duration := time.Duration(id*10+j) * time.Millisecond
				errorMsg := ""
				if !success {
					errorMsg = "test error"
				}
				metrics.RecordSend(platform, success, duration, errorMsg)
			}
		}(i)
	}

	// Test concurrent RecordHealth operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				platform := "health" + string(rune('0'+id%3)) // 3 different platforms
				healthy := j%2 == 0                            // Alternate healthy/unhealthy
				metrics.RecordHealth(platform, healthy)
			}
		}(i)
	}

	wg.Wait()

	// Verify final values
	totalOperations := int64(numGoroutines * operationsPerGoroutine)
	expectedSuccessful := totalOperations / 2 // Half should be successful
	expectedFailed := totalOperations - expectedSuccessful

	assert.Equal(t, expectedSuccessful, metrics.TotalSent)
	assert.Equal(t, expectedFailed, metrics.TotalFailed)

	// Check that platform data exists
	assert.True(t, len(metrics.SendsByPlatform) > 0)
	assert.True(t, len(metrics.FailsByPlatform) > 0)
	assert.True(t, len(metrics.PlatformHealth) > 0)

	// Verify success rate
	expectedRate := float64(expectedSuccessful) / float64(totalOperations)
	assert.InDelta(t, expectedRate, metrics.GetSuccessRate(), 0.001)
}


