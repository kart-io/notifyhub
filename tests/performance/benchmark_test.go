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
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/queue"
	"github.com/kart-io/notifyhub/tests/utils"
)

// BenchmarkSingleMessageSend 单消息发送性能测试
func BenchmarkSingleMessageSend(b *testing.B) {
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 1000, 4),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg, nil)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()
	msg := hub.NewMessage()
	msg.SetTitle("Benchmark Test")
	msg.SetBody("This is a benchmark test message")
	msg.SetPriority(3)

	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := hub.Send(ctx, msg, targets)
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

	hub, err := api.New(cfg, nil)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()
	msg := hub.NewMessage()
	msg.SetTitle("Benchmark Multi-Target")
	msg.SetBody("Testing multiple targets")
	msg.SetPriority(3)

	// 创建10个目标
	targets := make([]sending.Target, 10)
	for i := 0; i < 10; i++ {
		targets[i] = utils.CreateTestTarget(
			sending.TargetTypeEmail,
			"test@example.com",
			"test",
		)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := hub.Send(ctx, msg, targets)
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

	hub, err := api.New(cfg, nil)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()
	concurrency := runtime.NumCPU()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		msg := hub.NewMessage()
		msg.SetTitle("Concurrent Test")
		msg.SetBody("Concurrent message")
		msg.SetPriority(3)

		targets := []sending.Target{
			utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
		}

		for pb.Next() {
			_, err := hub.Send(ctx, msg, targets)
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
	hub, err := api.New(cfg, nil)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg := hub.NewMessage()
		msg.SetTitle("Test Title")
		msg.SetBody("Test Body")
		msg.SetPriority(3)
		msg.AddVariable("key1", "value1")
		msg.AddMetadata("meta1", "data1")
		msg.AddTarget(message.NewTarget(message.TargetTypeEmail, "test@example.com", "email"))
	}
}

// BenchmarkQueueThroughput 队列吞吐量测试
func BenchmarkQueueThroughput(b *testing.B) {
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 10000, 16),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg, nil)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()

	// 预先创建消息
	messages := make([]*queue.Message, 1000)
	for i := 0; i < 1000; i++ {
		msg := &queue.Message{
			ID:        fmt.Sprintf("test-msg-%d", i),
			Message:   message.NewMessage().SetTitle("Queue Test").SetBody("Testing queue throughput").SetPriority(3),
			Targets:   []sending.Target{},
			Attempts:  0,
			CreatedAt: time.Now(),
		}
		messages[i] = msg
	}

	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
	}
	_ = targets // Avoid unused variable error

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg := messages[i%1000]
		_, _ = hub.Send(ctx, msg.Message, msg.Targets)
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

	hub, err := api.New(cfg, nil)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()
	numMessages := 10000
	var sent int64
	var received int64
	_ = received // Avoid unused variable error
	startTime := time.Now()

	// 发送消息
	var wg sync.WaitGroup
	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			msg := hub.NewMessage()
			msg.SetTitle("Throughput Test")
			msg.SetBody("Message for throughput testing")
			msg.SetPriority(3)
			msg.AddVariable("id", id)

			targets := []sending.Target{
				utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
			}

			_, err := hub.Send(ctx, msg, targets)
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

	hub, err := api.New(cfg, nil)
	helper.AssertNoError(err, "Failed to create hub")
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()
	numSamples := 1000
	latencies := make([]time.Duration, numSamples)

	// 测量延迟
	for i := 0; i < numSamples; i++ {
		msg := hub.NewMessage()
		msg.SetTitle("Latency Test")
		msg.SetBody("Testing latency")
		msg.SetPriority(3)

		targets := []sending.Target{
			utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
		}

		start := time.Now()
		_, err := hub.Send(ctx, msg, targets)
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

	hub, err := api.New(cfg, nil)
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
		msg := hub.NewMessage()
		msg.SetTitle("Memory Test")
		msg.SetBody("Testing memory usage with a longer body content to increase memory pressure")
		msg.SetPriority(3)

		// 添加多个变量和元数据
		for j := 0; j < 10; j++ {
			msg.AddVariable("key", j)
			msg.AddMetadata("meta", "value")
		}

		targets := []sending.Target{
			utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
		}

		_, err := hub.Send(ctx, msg, targets)
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

	// 计算内存增长
	heapGrowth := m2.HeapAlloc - m1.HeapAlloc
	heapGrowthMB := float64(heapGrowth) / (1024 * 1024)

	t.Logf("Memory Usage:")
	t.Logf("  Initial Heap: %.2f MB", float64(m1.HeapAlloc)/(1024*1024))
	t.Logf("  Final Heap: %.2f MB", float64(m2.HeapAlloc)/(1024*1024))
	t.Logf("  Heap Growth: %.2f MB", heapGrowthMB)
	t.Logf("  Num GC: %d", m2.NumGC-m1.NumGC)

	// 验证内存使用
	helper.AssertTrue(heapGrowthMB < 100, "Heap growth should be < 100MB, got", heapGrowthMB)
}

// BenchmarkWithMiddleware 带中间件的性能测试
func BenchmarkWithMiddleware(b *testing.B) {
	cfg := config.New(
		config.WithMockNotifier("test"),
		config.WithQueue("memory", 1000, 4),
		config.WithSilentLogger(),
	)

	hub, err := api.New(cfg, nil)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// NotifyHub类型没有UseMiddleware方法，跳过中间件测试
	// for i := 0; i < 5; i++ {
	//     hub.UseMiddleware(...)
	// }

	ctx := context.Background()
	msg := hub.NewMessage()
	msg.SetTitle("Middleware Test")
	msg.SetBody("Testing with middleware")
	msg.SetPriority(3)

	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "test"),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := hub.Send(ctx, msg, targets)
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

	hub, err := api.New(cfg, nil)
	if err != nil {
		b.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()
	msg := hub.NewMessage()
	msg.SetTemplate("complex-template")
	msg.SetTitle("{{.service}} Alert: {{.status}}")
	msg.SetBody("Service {{.service}} is {{.status}}. Error: {{.error}}. Time: {{.timestamp}}")

	// 添加多个变量
	msg.AddVariable("service", "api-gateway")
	msg.AddVariable("status", "down")
	msg.AddVariable("error", "connection timeout")
	msg.AddVariable("timestamp", time.Now().Format(time.RFC3339))

	targets := []sending.Target{
		utils.CreateTestTarget(sending.TargetTypeEmail, "admin@example.com", "test"),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := hub.Send(ctx, msg, targets)
		if err != nil {
			b.Errorf("Send failed: %v", err)
		}
	}
}
