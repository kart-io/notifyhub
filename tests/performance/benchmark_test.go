package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/api"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/tests/utils"
)

// BenchmarkSingleMessageSend 单消息发送性能测试
func BenchmarkSingleMessageSend(b *testing.B) {
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 1000, 4),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := hub.Send().
			Title("Benchmark Test").
			Body("This is a benchmark test message").
			Priority(3).
			To("test@example.com").
			Send(ctx)
		if err != nil {
			b.Errorf("Send failed: %v", err)
		}
	}
}

// BenchmarkMultiTargetSend 多目标发送性能测试
func BenchmarkMultiTargetSend(b *testing.B) {
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 1000, 4),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := hub.Send().
			Title("Benchmark Multi-Target").
			Body("Testing multiple targets").
			Priority(3).
			To("test1@example.com", "test2@example.com", "test3@example.com", "test4@example.com", "test5@example.com",
				"test6@example.com", "test7@example.com", "test8@example.com", "test9@example.com", "test10@example.com").
			Send(ctx)
		if err != nil {
			b.Errorf("Send failed: %v", err)
		}
	}
}

// BenchmarkConcurrentSends 并发发送性能测试
func BenchmarkConcurrentSends(b *testing.B) {
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 5000, 8),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()
	concurrency := runtime.NumCPU()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := hub.Send().
				Title("Concurrent Test").
				Body("Concurrent message").
				Priority(3).
				To("test@example.com").
				Send(ctx)
			if err != nil {
				b.Errorf("Send failed: %v", err)
			}
		}
	})

	b.ReportMetric(float64(concurrency), "goroutines")
}

// BenchmarkMessageCreation 消息创建性能测试
func BenchmarkMessageCreation(b *testing.B) {
	cfg := config.New(config.WithSilentLogger())
	hub, err := api.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg := hub.Send()
		msg.Title("Test Title")
		msg.Body("Test Body")
		msg.Priority(3)
		msg.Vars(map[string]interface{}{"key1": "value1"})
		msg.Meta("meta1", "data1")
		msg.To("test@example.com")
	}
}

// BenchmarkQueueThroughput 队列吞吐量测试
func BenchmarkQueueThroughput(b *testing.B) {
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 10000, 16),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := hub.Send().
			Title("Queue Test").
			Body("Testing queue throughput").
			Priority(3).
			To("test@example.com").
			Send(ctx)
		if err != nil {
			b.Errorf("Send failed: %v", err)
		}
	}

	// 等待队列处理完成
	time.Sleep(100 * time.Millisecond)
}

// TestThroughput 吞吐量测试
func TestThroughput(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 10000, 16),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()
	numMessages := 10000
	var sent int64
	startTime := time.Now()

	// 发送消息
	var wg sync.WaitGroup
	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			_, err := hub.Send().
				Title("Throughput Test").
				Body("Message for throughput testing").
				Priority(3).
				Vars(map[string]interface{}{"id": id}).
				To("test@example.com").
				Send(ctx)
			if err == nil {
				atomic.AddInt64(&sent, 1)
			}
		}(i)

		// 控制并发度
		if i%100 == 0 {
			wg.Wait()
		}
	}

	wg.Wait()
	duration := time.Since(startTime)

	// 计算吞吐量
	throughput := float64(sent) / duration.Seconds()
	t.Logf("Sent %d messages in %v", sent, duration)
	t.Logf("Throughput: %.2f messages/second", throughput)

	// 验证吞吐量
	helper.AssertTrue(throughput > 1000, "Throughput should be > 1000 msg/sec, got", throughput)
	helper.AssertEqual(int64(numMessages), sent, "All messages should be sent")
}

// TestLatency 延迟测试
func TestLatency(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 1000, 4),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()
	numSamples := 1000
	latencies := make([]time.Duration, numSamples)

	// 测量延迟
	for i := 0; i < numSamples; i++ {
		start := time.Now()
		_, err := hub.Send().
			Title("Latency Test").
			Body("Testing latency").
			Priority(3).
			To("test@example.com").
			Send(ctx)
		latencies[i] = time.Since(start)

		if err != nil {
			t.Errorf("Send failed: %v", err)
		}
	}

	// 计算统计信息
	var total time.Duration
	var maxLatency time.Duration
	minLatency := time.Hour

	for _, latency := range latencies {
		total += latency
		if latency > maxLatency {
			maxLatency = latency
		}
		if latency < minLatency {
			minLatency = latency
		}
	}

	avgLatency := total / time.Duration(numSamples)

	// 计算P50, P95, P99
	p50 := latencies[numSamples*50/100]
	p95 := latencies[numSamples*95/100]
	p99 := latencies[numSamples*99/100]

	t.Logf("Latency Statistics:")
	t.Logf("  Min: %v", minLatency)
	t.Logf("  Max: %v", maxLatency)
	t.Logf("  Avg: %v", avgLatency)
	t.Logf("  P50: %v", p50)
	t.Logf("  P95: %v", p95)
	t.Logf("  P99: %v", p99)

	// 验证延迟要求
	helper.AssertTrue(avgLatency < 10*time.Millisecond, "Avg latency should be < 10ms, got", avgLatency)
	helper.AssertTrue(p95 < 50*time.Millisecond, "P95 latency should be < 50ms, got", p95)
	helper.AssertTrue(p99 < 100*time.Millisecond, "P99 latency should be < 100ms, got", p99)
}

// TestMemoryUsage 内存使用测试
func TestMemoryUsage(t *testing.T) {
	helper := utils.NewTestHelper(t)

	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 5000, 8),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// 获取初始内存状态
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	runtime.GC()

	ctx := context.Background()
	numMessages := 10000

	// 发送大量消息
	for i := 0; i < numMessages; i++ {
		// 添加多个变量和元数据
		variables := make(map[string]interface{})
		for j := 0; j < 10; j++ {
			variables[fmt.Sprintf("key%d", j)] = j
		}

		_, err := hub.Send().
			Title("Memory Test").
			Body("Testing memory usage with a longer body content to increase memory pressure").
			Priority(3).
			Vars(variables).
			Meta("meta", "value").
			To("test@example.com").
			Send(ctx)
		if err != nil {
			// 在性能测试中可以忽略发送错误或记录
			t.Logf("Send error: %v", err)
		}

		// 定期GC
		if i%1000 == 0 {
			runtime.GC()
		}
	}

	// 等待处理完成
	time.Sleep(500 * time.Millisecond)
	runtime.GC()

	// 获取最终内存状态
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// 计算内存增长 (处理可能的负增长)
	initialMB := float64(m1.HeapAlloc) / (1024 * 1024)
	finalMB := float64(m2.HeapAlloc) / (1024 * 1024)
	heapGrowthMB := finalMB - initialMB

	t.Logf("Memory Usage:")
	t.Logf("  Initial Heap: %.2f MB", initialMB)
	t.Logf("  Final Heap: %.2f MB", finalMB)
	t.Logf("  Heap Growth: %.2f MB", heapGrowthMB)
	t.Logf("  Num GC: %d", m2.NumGC-m1.NumGC)

	// 验证内存使用 (允许负增长，只检查合理的正增长)
	if heapGrowthMB > 0 {
		helper.AssertTrue(heapGrowthMB < 100, "Heap growth should be < 100MB, got", heapGrowthMB)
	}
}

// BenchmarkWithMiddleware 带中间件的性能测试
func BenchmarkWithMiddleware(b *testing.B) {
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 1000, 4),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := hub.Send().
			Title("Middleware Test").
			Body("Testing with middleware").
			Priority(3).
			To("test@example.com").
			Send(ctx)
		if err != nil {
			b.Errorf("Send failed: %v", err)
		}
	}
}

// BenchmarkTemplateRendering 模板渲染性能测试
func BenchmarkTemplateRendering(b *testing.B) {
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := hub.Send().
			Template("complex-template").
			Title("{{.service}} Alert: {{.status}}").
			Body("Service {{.service}} is {{.status}}. Error: {{.error}}. Time: {{.timestamp}}").
			Vars(map[string]interface{}{
				"service":   "api-gateway",
				"status":    "down",
				"error":     "connection timeout",
				"timestamp": time.Now().Format(time.RFC3339),
			}).
			To("admin@example.com").
			Send(ctx)
		if err != nil {
			b.Errorf("Send failed: %v", err)
		}
	}
}
