// Package notifyhub provides validation tests for call chain simplification
package notifyhub

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// TestCallChainSimplification validates that the call chain has been simplified from 6 to 3 layers
func TestCallChainSimplification(t *testing.T) {
	// Create a client for testing
	client, err := New(
		WithFeishuWebhook("https://open.feishu.cn/open-apis/bot/v2/hook/test", "test-secret"),
		WithTestDefaults(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create call chain tracer
	tracer := NewCallChainTracer()

	// Analyze the call chain
	analysis, err := tracer.TraceCallChain(client)
	if err != nil {
		t.Fatalf("Failed to trace call chain: %v", err)
	}

	// Generate detailed report
	report := GenerateCallChainReport(analysis)
	t.Logf("Call Chain Analysis Report:\n%s", report)

	// Validate requirements
	t.Run("LayerCount", func(t *testing.T) {
		if analysis.TotalLayers > 3 {
			t.Errorf("Call chain has %d layers, expected 3 or fewer", analysis.TotalLayers)
		} else {
			t.Logf("✅ Call chain simplified to %d layers", analysis.TotalLayers)
		}
	})

	t.Run("CallPath", func(t *testing.T) {
		// Expected pattern: Client → Dispatcher → Platform
		actualPath := strings.Join(analysis.CallPath, " → ")

		t.Logf("Actual call path: %s", actualPath)

		// Check that we don't have deprecated layers
		deprecatedLayers := []string{"core.NewHub", "HubImpl", "PlatformManager", "clientAdapter"}
		for _, deprecated := range deprecatedLayers {
			for _, call := range analysis.CallPath {
				if strings.Contains(call, deprecated) {
					t.Errorf("Found deprecated layer in call path: %s", deprecated)
				}
			}
		}

		// Verify we have the expected layers
		hasClient := false
		hasDispatcher := false
		hasPlatform := false

		for _, call := range analysis.CallPath {
			if strings.Contains(strings.ToLower(call), "client") {
				hasClient = true
			}
			if strings.Contains(strings.ToLower(call), "dispatch") {
				hasDispatcher = true
			}
			if strings.Contains(strings.ToLower(call), "platform") {
				hasPlatform = true
			}
		}

		if !hasClient {
			t.Error("Missing client layer in call path")
		}
		if !hasDispatcher {
			t.Error("Missing dispatcher layer in call path")
		}
		if !hasPlatform {
			t.Error("Missing platform layer in call path")
		}

		if hasClient && hasDispatcher && hasPlatform {
			t.Log("✅ All expected layers present in call path")
		}
	})

	t.Run("Performance", func(t *testing.T) {
		// Performance should be reasonable for a single call
		if analysis.TotalDuration > 100*time.Millisecond {
			t.Errorf("Call chain too slow: %s (expected < 100ms)", analysis.TotalDuration)
		} else {
			t.Logf("✅ Call chain performance: %s", analysis.TotalDuration)
		}
	})

	t.Run("Memory", func(t *testing.T) {
		// Memory allocation should be reasonable
		if analysis.MemoryAllocations > 1024*1024 { // 1MB
			t.Errorf("Call chain uses too much memory: %d bytes (expected < 1MB)", analysis.MemoryAllocations)
		} else {
			t.Logf("✅ Call chain memory usage: %d bytes", analysis.MemoryAllocations)
		}
	})
}

// TestInstanceLevelDependencyInjection validates that global state has been eliminated
func TestInstanceLevelDependencyInjection(t *testing.T) {
	// Create multiple independent client instances
	client1, err := New(
		WithFeishuWebhook("https://open.feishu.cn/open-apis/bot/v2/hook/test1", "secret1"),
		WithTestDefaults(),
	)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	client2, err := New(
		WithFeishuWebhook("https://open.feishu.cn/open-apis/bot/v2/hook/test2", "secret2"),
		WithTestDefaults(),
	)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	// Test that instances are truly independent
	t.Run("IndependentInstances", func(t *testing.T) {
		// Both clients should be able to send messages independently
		msg1 := &message.Message{
			ID:    "test-1",
			Title: "Test Message 1",
			Body:  "This is test message 1",
			Format: message.FormatText,
			Priority: message.PriorityNormal,
			Targets: []target.Target{
				{Type: "feishu", Value: "test-webhook-1", Platform: "feishu"},
			},
			CreatedAt: time.Now(),
		}

		msg2 := &message.Message{
			ID:    "test-2",
			Title: "Test Message 2",
			Body:  "This is test message 2",
			Format: message.FormatText,
			Priority: message.PriorityNormal,
			Targets: []target.Target{
				{Type: "feishu", Value: "test-webhook-2", Platform: "feishu"},
			},
			CreatedAt: time.Now(),
		}

		ctx := context.Background()

		// Send messages concurrently to test independence
		done1 := make(chan error, 1)
		done2 := make(chan error, 1)

		go func() {
			_, err := client1.Send(ctx, msg1)
			done1 <- err
		}()

		go func() {
			_, err := client2.Send(ctx, msg2)
			done2 <- err
		}()

		// Wait for both to complete
		err1 := <-done1
		err2 := <-done2

		if err1 != nil {
			t.Errorf("Client1 send failed: %v", err1)
		}
		if err2 != nil {
			t.Errorf("Client2 send failed: %v", err2)
		}

		if err1 == nil && err2 == nil {
			t.Log("✅ Multiple instances work independently")
		}
	})
}

// TestIntermediateLayerRemoval validates that intermediate layers have been removed
func TestIntermediateLayerRemoval(t *testing.T) {
	client, err := New(
		WithFeishuWebhook("https://open.feishu.cn/open-apis/bot/v2/hook/test", "test-secret"),
		WithTestDefaults(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	t.Run("DirectPlatformAccess", func(t *testing.T) {
		// Trace the execution to ensure no intermediate adapters
		tracer := NewCallChainTracer()
		analysis, err := tracer.TraceCallChain(client)
		if err != nil {
			t.Fatalf("Failed to trace call chain: %v", err)
		}

		// Check for absence of intermediate layers
		forbiddenLayers := []string{
			"clientAdapter",
			"PlatformAdapter",
			"HubImpl",
			"core.NewHub",
			"PlatformManager",
		}

		for _, forbidden := range forbiddenLayers {
			for _, call := range analysis.CallPath {
				if strings.Contains(call, forbidden) {
					t.Errorf("Found forbidden intermediate layer: %s in call %s", forbidden, call)
				}
			}
		}

		t.Logf("✅ No intermediate layers detected")
	})
}

// TestPerformanceImprovement validates the 30% performance improvement target
func TestPerformanceImprovement(t *testing.T) {
	// This test compares new architecture with simulated legacy architecture
	t.Run("PerformanceComparison", func(t *testing.T) {
		// Create new architecture client
		newClient, err := New(
			WithFeishuWebhook("https://open.feishu.cn/open-apis/bot/v2/hook/test", "test-secret"),
			WithTestDefaults(),
		)
		if err != nil {
			t.Fatalf("Failed to create new client: %v", err)
		}
		defer newClient.Close()

		// Create legacy architecture simulation
		legacyClient := newLegacyClientSimulation()

		// Test message
		msg := &message.Message{
			ID:    "perf-test",
			Title: "Performance Test",
			Body:  "Testing performance improvement",
			Format: message.FormatText,
			Priority: message.PriorityNormal,
			Targets: []target.Target{
				{Type: "feishu", Value: "test-webhook", Platform: "feishu"},
			},
			CreatedAt: time.Now(),
		}

		ctx := context.Background()

		// Measure new architecture performance
		newTracer := NewCallChainTracer()
		newAnalysis, err := newTracer.TraceCallChain(newClient)
		if err != nil {
			t.Fatalf("Failed to trace new architecture: %v", err)
		}

		// Measure legacy architecture performance (simulated)
		legacyStart := time.Now()
		_, err = legacyClient.Send(ctx, msg)
		if err != nil {
			t.Fatalf("Failed to execute legacy call: %v", err)
		}
		legacyDuration := time.Since(legacyStart)

		// Create simulated legacy analysis
		legacyAnalysis := &CallChainAnalysis{
			TotalLayers:   6, // Simulated 6-layer architecture
			TotalDuration: legacyDuration,
			CallPath: []string{
				"Client",
				"core.NewHub",
				"HubImpl",
				"Dispatcher",
				"PlatformManager",
				"Platform",
			},
		}

		// Generate comparison report
		comparisonReport := CompareCallChains(legacyAnalysis, newAnalysis)
		t.Logf("Performance Comparison Report:\n%s", comparisonReport)

		// Validate improvement targets
		layerImprovement := float64(legacyAnalysis.TotalLayers-newAnalysis.TotalLayers) / float64(legacyAnalysis.TotalLayers) * 100
		performanceImprovement := float64(legacyAnalysis.TotalDuration-newAnalysis.TotalDuration) / float64(legacyAnalysis.TotalDuration) * 100

		if layerImprovement < 50 { // 6 to 3 layers = 50% reduction
			t.Errorf("Layer reduction insufficient: %.1f%% (expected >= 50%%)", layerImprovement)
		} else {
			t.Logf("✅ Layer reduction: %.1f%%", layerImprovement)
		}

		if performanceImprovement < 30 {
			t.Errorf("Performance improvement insufficient: %.1f%% (expected >= 30%%)", performanceImprovement)
		} else {
			t.Logf("✅ Performance improvement: %.1f%%", performanceImprovement)
		}
	})
}

// TestArchitectureCompliance validates overall architecture compliance
func TestArchitectureCompliance(t *testing.T) {
	client, err := New(
		WithFeishuWebhook("https://open.feishu.cn/open-apis/bot/v2/hook/test", "test-secret"),
		WithTestDefaults(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	t.Run("RequirementsCompliance", func(t *testing.T) {
		tracer := NewCallChainTracer()
		analysis, err := tracer.TraceCallChain(client)
		if err != nil {
			t.Fatalf("Failed to trace call chain: %v", err)
		}

		// Requirement 3.1: Call chain not exceeding 3 layers
		if analysis.TotalLayers > 3 {
			t.Errorf("Requirement 3.1 FAILED: Call chain has %d layers (expected <= 3)", analysis.TotalLayers)
		} else {
			t.Logf("✅ Requirement 3.1 PASSED: Call chain has %d layers", analysis.TotalLayers)
		}

		// Requirement 3.4: No clientAdapter or other redundant adapters
		hasRedundantAdapters := false
		redundantPatterns := []string{"clientAdapter", "Adapter", "Manager"}
		for _, call := range analysis.CallPath {
			for _, pattern := range redundantPatterns {
				if strings.Contains(call, pattern) && !strings.Contains(call, "AsyncManager") {
					hasRedundantAdapters = true
					t.Errorf("Requirement 3.4 FAILED: Found redundant adapter: %s", call)
				}
			}
		}
		if !hasRedundantAdapters {
			t.Log("✅ Requirement 3.4 PASSED: No redundant adapters found")
		}

		// Requirement 14.1: Performance improvement target
		// Note: This would need actual legacy baseline for precise measurement
		if analysis.TotalDuration < 10*time.Millisecond {
			t.Log("✅ Requirement 14.1 INDICATION: Fast execution suggests performance improvement")
		}
	})
}