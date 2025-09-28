// Package target provides comprehensive performance testing for target functionality
package target

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// BenchmarkTargetCreation benchmarks target creation methods
func BenchmarkTargetCreation(b *testing.B) {
	b.Run("NewTarget", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewTarget("email", "user@example.com", "email")
		}
	})

	b.Run("NewEmailTarget", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewEmailTarget("user@example.com")
		}
	})

	b.Run("NewPhoneTarget", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewPhoneTarget("+1234567890")
		}
	})

	b.Run("NewFeishuUserTarget", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewFeishuUserTarget("ou_123456789")
		}
	})

	b.Run("NewWebhookTarget", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewWebhookTarget("https://example.com/webhook")
		}
	})
}

// BenchmarkTargetValidation benchmarks target validation methods
func BenchmarkTargetValidation(b *testing.B) {
	targets := []Target{
		{Type: "email", Value: "user@example.com"},
		{Type: "phone", Value: "+1234567890"},
		{Type: "webhook", Value: "https://example.com/webhook"},
		{Type: "user", Value: "ou_123456789"},
		{Type: "group", Value: "og_987654321"},
	}

	b.Run("Validate", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			target := targets[i%len(targets)]
			_ = target.Validate()
		}
	})

	b.Run("IsValid", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			target := targets[i%len(targets)]
			_ = target.IsValid()
		}
	})

	b.Run("String", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			target := targets[i%len(targets)]
			_ = target.String()
		}
	})
}

// BenchmarkTargetHelperMethods benchmarks target helper methods
func BenchmarkTargetHelperMethods(b *testing.B) {
	targets := []Target{
		{Type: "email", Value: "user@example.com", Platform: "email"},
		{Type: "phone", Value: "+1234567890", Platform: "sms"},
		{Type: "webhook", Value: "https://example.com/webhook", Platform: "webhook"},
		{Type: "user", Value: "ou_123456789", Platform: "feishu"},
	}

	b.Run("IsEmail", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			target := targets[i%len(targets)]
			_ = target.IsEmail()
		}
	})

	b.Run("IsPhone", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			target := targets[i%len(targets)]
			_ = target.IsPhone()
		}
	})

	b.Run("IsWebhook", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			target := targets[i%len(targets)]
			_ = target.IsWebhook()
		}
	})

	b.Run("IsFeishu", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			target := targets[i%len(targets)]
			_ = target.IsFeishu()
		}
	})
}

// BenchmarkTargetResolver_DetectionSpeed benchmarks auto-detection speed with various input sizes
func BenchmarkTargetResolver_DetectionSpeed(b *testing.B) {
	resolver := NewTargetResolver()

	// Test with inputs of different lengths
	emailSizes := []int{10, 50, 100, 200}
	for _, size := range emailSizes {
		email := strings.Repeat("a", size) + "@example.com"
		b.Run(fmt.Sprintf("EmailDetection_Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = resolver.AutoDetectTarget(email)
			}
		})
	}

	urlSizes := []int{20, 100, 500, 1000}
	for _, size := range urlSizes {
		url := "https://example.com/" + strings.Repeat("path/", size/5)
		b.Run(fmt.Sprintf("URLDetection_Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = resolver.AutoDetectTarget(url)
			}
		})
	}

	phoneSizes := []int{10, 12, 15}
	for _, size := range phoneSizes {
		phone := "+1" + strings.Repeat("2", size-2)
		b.Run(fmt.Sprintf("PhoneDetection_Size%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = resolver.AutoDetectTarget(phone)
			}
		})
	}
}

// BenchmarkTargetResolver_BatchPerformance benchmarks batch resolution with different batch sizes
func BenchmarkTargetResolver_BatchPerformance(b *testing.B) {
	resolver := NewTargetResolver()

	// Create test data for different batch sizes
	createBatch := func(size int) []string {
		batch := make([]string, size)
		for i := 0; i < size; i++ {
			switch i % 4 {
			case 0:
				batch[i] = fmt.Sprintf("user%d@example.com", i)
			case 1:
				batch[i] = fmt.Sprintf("+123456%04d", i)
			case 2:
				batch[i] = fmt.Sprintf("https://api%d.example.com/webhook", i)
			case 3:
				batch[i] = fmt.Sprintf("ou_%d123456789abcdef", i)
			}
		}
		return batch
	}

	batchSizes := []int{10, 100, 1000, 10000}
	for _, size := range batchSizes {
		batch := createBatch(size)
		b.Run(fmt.Sprintf("BatchSize_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = resolver.ResolveBatch(batch)
			}
		})
	}
}

// BenchmarkTargetResolver_StandardizationPerformance benchmarks standardization performance
func BenchmarkTargetResolver_StandardizationPerformance(b *testing.B) {
	resolver := NewTargetResolver()

	b.Run("EmailStandardization", func(b *testing.B) {
		inputs := []string{
			"User.Name+Tag@Gmail.Com",
			"user@EXAMPLE.COM",
			"test.email+alias@GMAIL.COM",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			input := inputs[i%len(inputs)]
			_ = resolver.StandardizeTarget(TargetTypeEmail, input)
		}
	})

	b.Run("PhoneStandardization", func(b *testing.B) {
		inputs := []string{
			"(555) 123-4567",
			"555-123-4567",
			"1 555 123 4567",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			input := inputs[i%len(inputs)]
			_ = resolver.StandardizeTarget(TargetTypePhone, input)
		}
	})

	b.Run("URLStandardization", func(b *testing.B) {
		inputs := []string{
			"HTTP://API.EXAMPLE.COM/webhook",
			"api.example.com/webhook",
			"HTTPS://EXAMPLE.COM:8080/webhook?param=value",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			input := inputs[i%len(inputs)]
			_ = resolver.StandardizeTarget(TargetTypeWebhook, input)
		}
	})
}

// BenchmarkTargetResolver_ConcurrentOperations benchmarks concurrent resolver operations
func BenchmarkTargetResolver_ConcurrentOperations(b *testing.B) {
	resolver := NewTargetResolver()

	testInputs := []string{
		"user@example.com",
		"+1234567890",
		"https://api.example.com/webhook",
		"ou_1234567890abcdef",
	}

	b.Run("ConcurrentAutoDetect", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				input := testInputs[i%len(testInputs)]
				_ = resolver.AutoDetectTarget(input)
				i++
			}
		})
	})

	b.Run("ConcurrentStandardize", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				input := testInputs[i%len(testInputs)]
				_ = resolver.StandardizeTarget(TargetTypeEmail, input)
				i++
			}
		})
	})
}

// BenchmarkTargetResolver_MemoryUsage benchmarks memory usage patterns
func BenchmarkTargetResolver_MemoryUsage(b *testing.B) {
	resolver := NewTargetResolver()

	b.Run("AutoDetectMemory", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = resolver.AutoDetectTarget("user@example.com")
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)

		allocPerOp := (m2.TotalAlloc - m1.TotalAlloc) / uint64(b.N)
		b.ReportMetric(float64(allocPerOp), "bytes/op")
	})

	b.Run("BatchResolveMemory", func(b *testing.B) {
		batch := []string{
			"user1@example.com",
			"user2@example.com",
			"+1234567890",
			"+1234567891",
		}

		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = resolver.ResolveBatch(batch)
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)

		allocPerOp := (m2.TotalAlloc - m1.TotalAlloc) / uint64(b.N)
		b.ReportMetric(float64(allocPerOp), "bytes/op")
	})
}

// TestTargetResolver_PerformanceThresholds tests that operations complete within acceptable time limits
func TestTargetResolver_PerformanceThresholds(t *testing.T) {
	resolver := NewTargetResolver()

	t.Run("AutoDetectPerformance", func(t *testing.T) {
		start := time.Now()
		iterations := 100000

		for i := 0; i < iterations; i++ {
			_ = resolver.AutoDetectTarget("user@example.com")
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		// Should average less than 1 microsecond per operation
		if avgTime > time.Microsecond {
			t.Errorf("AutoDetectTarget too slow: %v per operation (want < 1µs)", avgTime)
		}

		t.Logf("AutoDetectTarget: %d operations in %v (avg: %v)", iterations, duration, avgTime)
	})

	t.Run("BatchResolvePerformance", func(t *testing.T) {
		batch := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			batch[i] = fmt.Sprintf("user%d@example.com", i)
		}

		start := time.Now()
		iterations := 1000

		for i := 0; i < iterations; i++ {
			_, _ = resolver.ResolveBatch(batch)
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		// Should complete 1000-item batch in less than 10ms
		if avgTime > 10*time.Millisecond {
			t.Errorf("ResolveBatch too slow: %v per 1000-item batch (want < 10ms)", avgTime)
		}

		t.Logf("ResolveBatch: %d batches of 1000 in %v (avg: %v)", iterations, duration, avgTime)
	})

	t.Run("ValidationPerformance", func(t *testing.T) {
		target := Target{Type: "email", Value: "user@example.com"}
		start := time.Now()
		iterations := 1000000

		for i := 0; i < iterations; i++ {
			_ = target.Validate()
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		// Should average less than 10 microseconds per validation
		if avgTime > 10*time.Microsecond {
			t.Errorf("Target.Validate too slow: %v per operation (want < 10µs)", avgTime)
		}

		t.Logf("Target.Validate: %d operations in %v (avg: %v)", iterations, duration, avgTime)
	})
}

// TestTargetResolver_ConcurrentStressTest tests resolver under high concurrent load
func TestTargetResolver_ConcurrentStressTest(t *testing.T) {
	resolver := NewTargetResolver()

	numGoroutines := 100
	operationsPerGoroutine := 1000
	totalOps := numGoroutines * operationsPerGoroutine

	start := time.Now()

	var wg sync.WaitGroup
	var successCount int64
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			localSuccessCount := 0
			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix different types of operations
				switch j % 4 {
				case 0:
					target := resolver.AutoDetectTarget(fmt.Sprintf("user%d_%d@example.com", id, j))
					if target.Type == TargetTypeEmail {
						localSuccessCount++
					}
				case 1:
					target := resolver.AutoDetectTarget(fmt.Sprintf("+12345%05d", id*1000+j))
					if target.Type == TargetTypePhone {
						localSuccessCount++
					}
				case 2:
					standardized := resolver.StandardizeTarget(TargetTypeEmail, fmt.Sprintf("User%d@Example.Com", id))
					if strings.Contains(standardized, "@") {
						localSuccessCount++
					}
				case 3:
					batch := []string{
						fmt.Sprintf("user%d@example.com", id),
						fmt.Sprintf("+123456%04d", j),
					}
					targets, _ := resolver.ResolveBatch(batch)
					if len(targets) > 0 {
						localSuccessCount++
					}
				}
			}

			mu.Lock()
			successCount += int64(localSuccessCount)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Check that we got reasonable success rate
	successRate := float64(successCount) / float64(totalOps)
	if successRate < 0.95 {
		t.Errorf("success rate too low: %.2f%% (want >= 95%%)", successRate*100)
	}

	// Check that operations completed in reasonable time
	opsPerSecond := float64(totalOps) / duration.Seconds()
	if opsPerSecond < 10000 {
		t.Errorf("operations per second too low: %.0f (want >= 10000)", opsPerSecond)
	}

	t.Logf("Concurrent stress test: %d operations in %v (%.0f ops/sec, %.2f%% success)",
		totalOps, duration, opsPerSecond, successRate*100)
}

// TestTargetResolver_MemoryLeakDetection tests for memory leaks in resolver operations
func TestTargetResolver_MemoryLeakDetection(t *testing.T) {
	resolver := NewTargetResolver()

	// Capture initial memory stats
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Perform many operations
	for i := 0; i < 10000; i++ {
		// Auto-detect different types
		_ = resolver.AutoDetectTarget(fmt.Sprintf("user%d@example.com", i))
		_ = resolver.AutoDetectTarget(fmt.Sprintf("+123456%04d", i))
		_ = resolver.AutoDetectTarget(fmt.Sprintf("https://api%d.example.com/webhook", i))

		// Standardize targets
		_ = resolver.StandardizeTarget(TargetTypeEmail, fmt.Sprintf("User%d@Example.Com", i))
		_ = resolver.StandardizeTarget(TargetTypePhone, fmt.Sprintf("(%03d) %03d-%04d", i%1000, i%1000, i%10000))

		// Batch resolve
		batch := []string{
			fmt.Sprintf("user%d@example.com", i),
			fmt.Sprintf("+123456%04d", i),
		}
		_, _ = resolver.ResolveBatch(batch)
	}

	// Force garbage collection
	runtime.GC()
	runtime.GC() // Double GC to ensure cleanup

	// Capture final memory stats
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Check for excessive memory growth (handle potential negative values)
	var memGrowthMB float64
	if m2.Alloc > m1.Alloc {
		memGrowthMB = float64(m2.Alloc-m1.Alloc) / 1024 / 1024
	} else {
		memGrowthMB = 0 // Memory actually decreased (good!)
	}

	if memGrowthMB > 50 { // More reasonable threshold
		t.Errorf("excessive memory growth: %.2f MB (want < 50 MB)", memGrowthMB)
	}

	t.Logf("Memory usage: initial=%d KB, final=%d KB, growth=%.2f MB",
		m1.Alloc/1024, m2.Alloc/1024, memGrowthMB)
}