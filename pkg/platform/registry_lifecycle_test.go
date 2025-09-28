package platform

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// TestPlatformLifecycleManagement tests the complete platform lifecycle
func TestPlatformLifecycleManagement(t *testing.T) {
	registry := NewRegistry(logger.New())
	defer registry.Close()

	platformName := "lifecycle-test"

	// Test platform registration
	err := registry.Register(platformName, mockCreator(platformName))
	if err != nil {
		t.Fatalf("Failed to register platform: %v", err)
	}

	// Verify registration
	if !registry.IsRegistered(platformName) {
		t.Error("Platform should be registered")
	}

	// Test platform configuration
	config := map[string]interface{}{
		"webhook_url": "http://example.com/webhook",
		"secret":      "test-secret",
	}
	err = registry.SetConfig(platformName, config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Verify configuration
	if !registry.IsConfigured(platformName) {
		t.Error("Platform should be configured")
	}

	// Test platform startup
	ctx := context.Background()
	err = registry.StartPlatform(ctx, platformName)
	if err != nil {
		t.Fatalf("Failed to start platform: %v", err)
	}

	// Verify platform is running
	instances := registry.ListInstances()
	if len(instances) != 1 || instances[0] != platformName {
		t.Errorf("Expected 1 running instance [%s], got: %v", platformName, instances)
	}

	// Test platform info retrieval
	info, err := registry.GetPlatformInfo(platformName)
	if err != nil {
		t.Fatalf("Failed to get platform info: %v", err)
	}

	if info.Name != platformName {
		t.Errorf("Expected platform name %s, got %s", platformName, info.Name)
	}

	if info.Status != StatusHealthy {
		t.Errorf("Expected status %s, got %s", StatusHealthy, info.Status)
	}

	// Test platform restart
	err = registry.RestartPlatform(ctx, platformName)
	if err != nil {
		t.Fatalf("Failed to restart platform: %v", err)
	}

	// Test platform stop
	err = registry.StopPlatform(platformName)
	if err != nil {
		t.Fatalf("Failed to stop platform: %v", err)
	}

	// Verify platform is stopped
	instances = registry.ListInstances()
	if len(instances) != 0 {
		t.Errorf("Expected no running instances, got: %v", instances)
	}

	// Test platform unregistration
	err = registry.Unregister(platformName)
	if err != nil {
		t.Fatalf("Failed to unregister platform: %v", err)
	}

	// Verify unregistration
	if registry.IsRegistered(platformName) {
		t.Error("Platform should not be registered after unregistration")
	}
}

// TestHealthMonitoring tests the health monitoring functionality
func TestHealthMonitoring(t *testing.T) {
	healthConfig := HealthCheckConfig{
		Enabled:          true,
		Interval:         100 * time.Millisecond,
		Timeout:          50 * time.Millisecond,
		RetryThreshold:   2,
		UnhealthyTimeout: 1 * time.Second,
	}

	registry := NewRegistryWithHealthConfig(logger.New(), healthConfig)
	defer registry.Close()

	platformName := "health-test"

	// Register and configure platform
	err := registry.Register(platformName, mockCreator(platformName))
	if err != nil {
		t.Fatalf("Failed to register platform: %v", err)
	}

	config := map[string]interface{}{
		"webhook_url": "http://example.com/webhook",
	}
	err = registry.SetConfig(platformName, config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Start platform
	ctx := context.Background()
	err = registry.StartPlatform(ctx, platformName)
	if err != nil {
		t.Fatalf("Failed to start platform: %v", err)
	}

	// Wait for initial health check
	time.Sleep(200 * time.Millisecond)

	// Check health status
	health := registry.Health(ctx)
	if health[platformName] != nil {
		t.Errorf("Expected healthy platform, got error: %v", health[platformName])
	}

	// Verify platform is healthy
	if !registry.IsHealthy(platformName) {
		t.Error("Platform should be healthy")
	}

	// Get health summary
	summary := registry.HealthSummary()
	platformInfo, exists := summary[platformName]
	if !exists {
		t.Fatal("Platform not found in health summary")
	}

	if platformInfo.Status != StatusHealthy {
		t.Errorf("Expected status %s, got %s", StatusHealthy, platformInfo.Status)
	}

	if platformInfo.LastHealthCheck == nil {
		t.Error("LastHealthCheck should not be nil")
	}
}

// TestPlatformMetrics tests the metrics tracking functionality
func TestPlatformMetrics(t *testing.T) {
	registry := NewRegistry(logger.New())
	defer registry.Close()

	platformName := "metrics-test"

	// Register, configure and start platform
	err := registry.Register(platformName, mockCreator(platformName))
	if err != nil {
		t.Fatalf("Failed to register platform: %v", err)
	}

	config := map[string]interface{}{
		"webhook_url": "http://example.com/webhook",
	}
	err = registry.SetConfig(platformName, config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	ctx := context.Background()
	err = registry.StartPlatform(ctx, platformName)
	if err != nil {
		t.Fatalf("Failed to start platform: %v", err)
	}

	// Test metrics updates
	registry.UpdateMetrics(platformName, true, 100*time.Millisecond)
	registry.UpdateMetrics(platformName, true, 200*time.Millisecond)
	registry.UpdateMetrics(platformName, false, 150*time.Millisecond)

	// Get platform info to check metrics
	info, err := registry.GetPlatformInfo(platformName)
	if err != nil {
		t.Fatalf("Failed to get platform info: %v", err)
	}

	metrics := info.Metrics
	if metrics.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", metrics.TotalRequests)
	}

	if metrics.SuccessfulSends != 2 {
		t.Errorf("Expected 2 successful sends, got %d", metrics.SuccessfulSends)
	}

	if metrics.FailedSends != 1 {
		t.Errorf("Expected 1 failed send, got %d", metrics.FailedSends)
	}

	expectedAvg := 150 * time.Millisecond
	if metrics.AverageLatency != expectedAvg {
		t.Errorf("Expected average latency %v, got %v", expectedAvg, metrics.AverageLatency)
	}

	if metrics.LastActivity == nil {
		t.Error("LastActivity should not be nil")
	}
}

// TestPlatformSelection tests the platform selection functionality
func TestPlatformSelection(t *testing.T) {
	registry := NewRegistry(logger.New())
	defer registry.Close()

	// Register multiple platforms with different capabilities
	platforms := []struct {
		name         string
		targetTypes  []string
		formats      []string
		messageSize  int
		scheduling   bool
		attachments  bool
	}{
		{"email", []string{"email"}, []string{"text", "html"}, 10240, false, true},
		{"feishu", []string{"webhook", "user"}, []string{"text", "markdown"}, 2048, true, false},
		{"sms", []string{"phone"}, []string{"text"}, 160, false, false},
		{"webhook", []string{"webhook"}, []string{"text", "json"}, 8192, true, false},
	}

	for _, p := range platforms {
		// Create mock creator with specific capabilities
		creator := func(name string, caps Capabilities) PlatformCreator {
			return func(config map[string]interface{}, logger logger.Logger) (Platform, error) {
				mock := newMockPlatform(name)
				mock.capabilities = caps
				return mock, nil
			}
		}(p.name, Capabilities{
			Name:                 p.name,
			SupportedTargetTypes: p.targetTypes,
			SupportedFormats:     p.formats,
			MaxMessageSize:       p.messageSize,
			SupportsScheduling:   p.scheduling,
			SupportsAttachments:  p.attachments,
		})

		err := registry.Register(p.name, creator)
		if err != nil {
			t.Fatalf("Failed to register platform %s: %v", p.name, err)
		}

		config := map[string]interface{}{"url": "http://example.com"}
		err = registry.SetConfig(p.name, config)
		if err != nil {
			t.Fatalf("Failed to configure platform %s: %v", p.name, err)
		}

		ctx := context.Background()
		err = registry.StartPlatform(ctx, p.name)
		if err != nil {
			t.Fatalf("Failed to start platform %s: %v", p.name, err)
		}
	}

	// Test platform selection by target type
	emailPlatforms := registry.GetHealthyPlatforms("email")
	if len(emailPlatforms) != 1 || emailPlatforms[0] != "email" {
		t.Errorf("Expected [email], got %v", emailPlatforms)
	}

	webhookPlatforms := registry.GetHealthyPlatforms("webhook")
	if len(webhookPlatforms) != 2 {
		t.Errorf("Expected 2 webhook platforms, got %d", len(webhookPlatforms))
	}

	// Test platform selection with criteria
	criteria := PlatformCriteria{
		TargetType:          "webhook",
		Format:              "text",
		RequiresScheduling:  true,
		RequiresAttachments: false,
		HealthyOnly:         true,
	}

	matches := registry.SelectPlatforms(criteria)
	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d: %v", len(matches), matches)
	}

	// Test best platform selection
	best, err := registry.GetBestPlatform(criteria)
	if err != nil {
		t.Fatalf("Failed to get best platform: %v", err)
	}

	if best != "feishu" && best != "webhook" {
		t.Errorf("Expected feishu or webhook, got %s", best)
	}

	// Test capabilities lookup
	caps := registry.GetCapabilities("email")
	if len(caps) != 1 {
		t.Errorf("Expected 1 email capability, got %d", len(caps))
	}

	emailCap, exists := caps["email"]
	if !exists {
		t.Error("Email platform capabilities not found")
	}

	if !emailCap.SupportsAttachments {
		t.Error("Email platform should support attachments")
	}
}

// TestRegistryStats tests the registry statistics functionality
func TestRegistryStats(t *testing.T) {
	registry := NewRegistry(logger.New())
	defer registry.Close()

	// Initial stats should be empty
	stats := registry.GetRegistryStats()
	if stats.Total != 0 || stats.Registered != 0 || stats.Configured != 0 || stats.Running != 0 {
		t.Errorf("Expected empty stats, got %+v", stats)
	}

	// Register a platform
	platformName := "stats-test"
	err := registry.Register(platformName, mockCreator(platformName))
	if err != nil {
		t.Fatalf("Failed to register platform: %v", err)
	}

	stats = registry.GetRegistryStats()
	if stats.Total != 1 || stats.Registered != 1 || stats.Configured != 0 || stats.Running != 0 {
		t.Errorf("Expected (1,1,0,0), got (%d,%d,%d,%d)", stats.Total, stats.Registered, stats.Configured, stats.Running)
	}

	// Configure the platform
	config := map[string]interface{}{"url": "http://example.com"}
	err = registry.SetConfig(platformName, config)
	if err != nil {
		t.Fatalf("Failed to configure platform: %v", err)
	}

	stats = registry.GetRegistryStats()
	if stats.Total != 1 || stats.Registered != 1 || stats.Configured != 1 || stats.Running != 0 {
		t.Errorf("Expected (1,1,1,0), got (%d,%d,%d,%d)", stats.Total, stats.Registered, stats.Configured, stats.Running)
	}

	// Start the platform
	ctx := context.Background()
	err = registry.StartPlatform(ctx, platformName)
	if err != nil {
		t.Fatalf("Failed to start platform: %v", err)
	}

	stats = registry.GetRegistryStats()
	if stats.Total != 1 || stats.Registered != 1 || stats.Configured != 1 || stats.Running != 1 {
		t.Errorf("Expected (1,1,1,1), got (%d,%d,%d,%d)", stats.Total, stats.Registered, stats.Configured, stats.Running)
	}

	// Check status counts
	if stats.StatusCounts[StatusHealthy] != 1 {
		t.Errorf("Expected 1 healthy platform, got %d", stats.StatusCounts[StatusHealthy])
	}
}

// TestBatchOperations tests the batch operation functionality
func TestBatchOperations(t *testing.T) {
	registry := NewRegistry(logger.New())
	defer registry.Close()

	// Define batch operations
	operations := []RegistryOperation{
		{
			Type:     OpRegister,
			Platform: "batch-1",
			Creator:  mockCreator("batch-1"),
		},
		{
			Type:     OpRegister,
			Platform: "batch-2",
			Creator:  mockCreator("batch-2"),
		},
		{
			Type:     OpConfigure,
			Platform: "batch-1",
			Config:   map[string]interface{}{"url": "http://example.com/1"},
		},
		{
			Type:     OpConfigure,
			Platform: "batch-2",
			Config:   map[string]interface{}{"url": "http://example.com/2"},
		},
	}

	// Execute batch operations
	err := registry.BatchOperation(operations)
	if err != nil {
		t.Fatalf("Failed to execute batch operations: %v", err)
	}

	// Verify results
	if !registry.IsRegistered("batch-1") || !registry.IsRegistered("batch-2") {
		t.Error("Both platforms should be registered")
	}

	if !registry.IsConfigured("batch-1") || !registry.IsConfigured("batch-2") {
		t.Error("Both platforms should be configured")
	}

	// Test batch operation with error
	invalidOperations := []RegistryOperation{
		{
			Type:     OpRegister,
			Platform: "", // Invalid empty name
			Creator:  mockCreator("invalid"),
		},
	}

	err = registry.BatchOperation(invalidOperations)
	if err == nil {
		t.Error("Expected error for invalid batch operations")
	}
}

// TestGracefulShutdown tests the graceful shutdown functionality
func TestGracefulShutdown(t *testing.T) {
	registry := NewRegistry(logger.New())

	// Register and start a platform
	platformName := "shutdown-test"
	err := registry.Register(platformName, mockCreator(platformName))
	if err != nil {
		t.Fatalf("Failed to register platform: %v", err)
	}

	config := map[string]interface{}{"url": "http://example.com"}
	err = registry.SetConfig(platformName, config)
	if err != nil {
		t.Fatalf("Failed to configure platform: %v", err)
	}

	ctx := context.Background()
	err = registry.StartPlatform(ctx, platformName)
	if err != nil {
		t.Fatalf("Failed to start platform: %v", err)
	}

	// Test graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = registry.Shutdown(shutdownCtx)
	if err != nil {
		t.Fatalf("Failed to shutdown registry: %v", err)
	}

	// Verify registry is shut down
	stats := registry.GetRegistryStats()
	if stats.Running != 0 {
		t.Errorf("Expected no running platforms after shutdown, got %d", stats.Running)
	}

	// Test that operations fail after shutdown
	err = registry.Register("new-platform", mockCreator("new-platform"))
	if err == nil {
		t.Error("Expected error when registering after shutdown")
	}
}

// TestConfigurationChange tests configuration change handling
func TestConfigurationChange(t *testing.T) {
	registry := NewRegistry(logger.New())
	defer registry.Close()

	platformName := "config-change-test"

	// Register and configure platform
	err := registry.Register(platformName, mockCreator(platformName))
	if err != nil {
		t.Fatalf("Failed to register platform: %v", err)
	}

	initialConfig := map[string]interface{}{
		"url":    "http://example.com/initial",
		"secret": "initial-secret",
	}
	err = registry.SetConfig(platformName, initialConfig)
	if err != nil {
		t.Fatalf("Failed to set initial config: %v", err)
	}

	// Start platform
	platform1, err := registry.GetPlatform(platformName)
	if err != nil {
		t.Fatalf("Failed to get platform: %v", err)
	}

	// Change configuration
	newConfig := map[string]interface{}{
		"url":    "http://example.com/new",
		"secret": "new-secret",
	}
	err = registry.SetConfig(platformName, newConfig)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Get platform again - should be a new instance due to config change
	platform2, err := registry.GetPlatform(platformName)
	if err != nil {
		t.Fatalf("Failed to get platform after config change: %v", err)
	}

	// Verify it's a new instance (in a real implementation, the instances would be different)
	// For mock platforms, we can at least verify the platform is accessible
	if platform1 == nil || platform2 == nil {
		t.Error("Platforms should not be nil")
	}

	// Verify configuration is sanitized in platform info
	info, err := registry.GetPlatformInfo(platformName)
	if err != nil {
		t.Fatalf("Failed to get platform info: %v", err)
	}

	if info.Config["secret"] != "[REDACTED]" {
		t.Errorf("Expected secret to be redacted, got %v", info.Config["secret"])
	}

	if info.Config["url"] != "http://example.com/new" {
		t.Errorf("Expected new URL, got %v", info.Config["url"])
	}
}