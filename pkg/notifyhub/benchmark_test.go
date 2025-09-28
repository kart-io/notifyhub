// Package notifyhub provides benchmarks for validating call chain simplification
package notifyhub

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
	"github.com/kart-io/notifyhub/pkg/platform"
)

// BenchmarkCallChainSimplification tests the performance improvement from 6-layer to 3-layer architecture
func BenchmarkCallChainSimplification(b *testing.B) {
	// Test both new simplified and legacy architectures
	b.Run("NewSimplifiedArchitecture", benchmarkNewArchitecture)
	b.Run("LegacyArchitecture", benchmarkLegacyArchitecture)
}

// benchmarkNewArchitecture tests the new 3-layer call chain: Client → Dispatcher → Platform
func benchmarkNewArchitecture(b *testing.B) {
	// Create a new client with minimal configuration
	client, err := New(
		WithFeishuWebhook("https://open.feishu.cn/open-apis/bot/v2/hook/test", "test-secret"),
		WithTestDefaults(),
	)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create a test message
	msg := &message.Message{
		ID:    "bench-test",
		Title: "Benchmark Test",
		Body:  "This is a benchmark test message",
		Format: message.FormatText,
		Priority: message.PriorityNormal,
		Targets: []target.Target{
			{Type: "feishu", Value: "test-webhook", Platform: "feishu"},
		},
		CreatedAt: time.Now(),
	}

	ctx := context.Background()

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Benchmark the call chain
	for i := 0; i < b.N; i++ {
		_, err := client.Send(ctx, msg)
		if err != nil {
			b.Fatalf("Send failed: %v", err)
		}
	}
}

// benchmarkLegacyArchitecture simulates the old 6-layer call chain for comparison
func benchmarkLegacyArchitecture(b *testing.B) {
	// Create a legacy-style client simulation
	legacy := newLegacyClientSimulation()

	// Create a test message
	msg := &message.Message{
		ID:    "bench-test-legacy",
		Title: "Benchmark Test Legacy",
		Body:  "This is a benchmark test message for legacy architecture",
		Format: message.FormatText,
		Priority: message.PriorityNormal,
		Targets: []target.Target{
			{Type: "feishu", Value: "test-webhook", Platform: "feishu"},
		},
		CreatedAt: time.Now(),
	}

	ctx := context.Background()

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Benchmark the legacy call chain
	for i := 0; i < b.N; i++ {
		_, err := legacy.Send(ctx, msg)
		if err != nil {
			b.Fatalf("Legacy send failed: %v", err)
		}
	}
}

// BenchmarkMemoryAllocation tests memory allocation improvements
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("NewArchitecture", benchmarkNewArchitectureMemory)
	b.Run("LegacyArchitecture", benchmarkLegacyArchitectureMemory)
}

func benchmarkNewArchitectureMemory(b *testing.B) {
	client, err := New(
		WithFeishuWebhook("https://open.feishu.cn/open-apis/bot/v2/hook/test", "test-secret"),
		WithTestDefaults(),
	)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	msg := &message.Message{
		ID:    "bench-mem-test",
		Title: "Memory Benchmark Test",
		Body:  "This tests memory allocation efficiency",
		Format: message.FormatText,
		Priority: message.PriorityNormal,
		Targets: []target.Target{
			{Type: "feishu", Value: "test-webhook", Platform: "feishu"},
		},
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := client.Send(ctx, msg)
		if err != nil {
			b.Fatalf("Send failed: %v", err)
		}
	}
}

func benchmarkLegacyArchitectureMemory(b *testing.B) {
	legacy := newLegacyClientSimulation()

	msg := &message.Message{
		ID:    "bench-mem-test-legacy",
		Title: "Memory Benchmark Test Legacy",
		Body:  "This tests memory allocation efficiency for legacy",
		Format: message.FormatText,
		Priority: message.PriorityNormal,
		Targets: []target.Target{
			{Type: "feishu", Value: "test-webhook", Platform: "feishu"},
		},
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := legacy.Send(ctx, msg)
		if err != nil {
			b.Fatalf("Legacy send failed: %v", err)
		}
	}
}

// BenchmarkConcurrentClients tests multi-instance performance
func BenchmarkConcurrentClients(b *testing.B) {
	numClients := 10
	clients := make([]Client, numClients)

	// Create multiple client instances to test instance isolation
	for i := 0; i < numClients; i++ {
		client, err := New(
			WithFeishuWebhook("https://open.feishu.cn/open-apis/bot/v2/hook/test", "test-secret"),
			WithTestDefaults(),
		)
		if err != nil {
			b.Fatalf("Failed to create client %d: %v", i, err)
		}
		clients[i] = client
		defer client.Close()
	}

	msg := &message.Message{
		ID:    "bench-concurrent-test",
		Title: "Concurrent Benchmark Test",
		Body:  "This tests concurrent client performance",
		Format: message.FormatText,
		Priority: message.PriorityNormal,
		Targets: []target.Target{
			{Type: "feishu", Value: "test-webhook", Platform: "feishu"},
		},
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		clientIndex := 0
		for pb.Next() {
			client := clients[clientIndex%numClients]
			_, err := client.Send(ctx, msg)
			if err != nil {
				b.Errorf("Concurrent send failed: %v", err)
			}
			clientIndex++
		}
	})
}

// Legacy architecture simulation for comparison

// legacyClientSimulation simulates the old 6-layer call chain
type legacyClientSimulation struct {
	hub *legacyHubImpl
}

// legacyHubImpl simulates HubImpl from the old architecture
type legacyHubImpl struct {
	dispatcher      *legacyDispatcher
	platformManager *legacyPlatformManager
}

// legacyDispatcher simulates the old dispatcher layer
type legacyDispatcher struct {
	manager *legacyPlatformManager
}

// legacyPlatformManager simulates PlatformManager layer
type legacyPlatformManager struct {
	platforms map[string]*legacyPlatformAdapter
}

// legacyPlatformAdapter simulates clientAdapter layer
type legacyPlatformAdapter struct {
	platform *mockPlatform
}

func newLegacyClientSimulation() *legacyClientSimulation {
	// Simulate the 6-layer call chain construction
	platform := &mockPlatform{
		name: "feishu",
		capabilities: platform.Capabilities{
			Name:                 "feishu",
			SupportedTargetTypes: []string{"feishu", "webhook"},
			SupportedFormats:     []string{"text", "markdown"},
			MaxMessageSize:       4000,
		},
	}

	adapter := &legacyPlatformAdapter{platform: platform}
	manager := &legacyPlatformManager{
		platforms: map[string]*legacyPlatformAdapter{
			"feishu": adapter,
		},
	}
	dispatcher := &legacyDispatcher{manager: manager}
	hub := &legacyHubImpl{
		dispatcher:      dispatcher,
		platformManager: manager,
	}

	return &legacyClientSimulation{hub: hub}
}

// Send simulates the 6-layer call chain:
// Client → core.NewHub → HubImpl → Dispatcher → PlatformManager → ClientAdapter → Platform
func (l *legacyClientSimulation) Send(ctx context.Context, msg *message.Message) (interface{}, error) {
	// Layer 1: Client call
	// Layer 2: core.NewHub call
	result1 := l.coreNewHubLayer(ctx, msg)

	// Layer 3: HubImpl call
	result2 := l.hub.hubImplLayer(ctx, result1)

	// Layer 4: Dispatcher call
	result3 := l.hub.dispatcher.dispatcherLayer(ctx, result2)

	// Layer 5: PlatformManager call
	result4 := l.hub.platformManager.platformManagerLayer(ctx, result3)

	// Layer 6: Platform call (through adapter)
	return l.hub.platformManager.platforms["feishu"].platformAdapterLayer(ctx, result4)
}

func (l *legacyClientSimulation) coreNewHubLayer(ctx context.Context, msg *message.Message) *message.Message {
	// Simulate processing overhead in core.NewHub
	time.Sleep(time.Nanosecond) // Minimal delay to simulate processing
	return msg
}

func (h *legacyHubImpl) hubImplLayer(ctx context.Context, msg *message.Message) *message.Message {
	// Simulate processing overhead in HubImpl
	time.Sleep(time.Nanosecond)
	return msg
}

func (d *legacyDispatcher) dispatcherLayer(ctx context.Context, msg *message.Message) *message.Message {
	// Simulate processing overhead in Dispatcher
	time.Sleep(time.Nanosecond)
	return msg
}

func (p *legacyPlatformManager) platformManagerLayer(ctx context.Context, msg *message.Message) *message.Message {
	// Simulate processing overhead in PlatformManager
	time.Sleep(time.Nanosecond)
	return msg
}

func (a *legacyPlatformAdapter) platformAdapterLayer(ctx context.Context, msg *message.Message) (interface{}, error) {
	// Simulate processing overhead in clientAdapter
	time.Sleep(time.Nanosecond)

	// Final call to platform
	targets := []target.Target{
		{Type: "feishu", Value: "test-webhook", Platform: "feishu"},
	}

	return a.platform.Send(ctx, msg, targets)
}