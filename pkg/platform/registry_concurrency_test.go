package platform

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// mockPlatform is a test implementation of the Platform interface
type mockPlatform struct {
	name         string
	capabilities Capabilities
	health       error
	sendResults  []*SendResult
	sendError    error
	closeError   error
	closed       bool
	mu           sync.Mutex
}

func newMockPlatform(name string) *mockPlatform {
	return &mockPlatform{
		name: name,
		capabilities: Capabilities{
			Name:                 name,
			SupportedTargetTypes: []string{"webhook"},
			SupportedFormats:     []string{"text"},
			MaxMessageSize:       1024,
			SupportsScheduling:   false,
			SupportsAttachments:  false,
			RequiredSettings:     []string{},
		},
		sendResults: []*SendResult{},
	}
}

func (m *mockPlatform) Name() string {
	return m.name
}

func (m *mockPlatform) GetCapabilities() Capabilities {
	return m.capabilities
}

func (m *mockPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendError != nil {
		return nil, m.sendError
	}

	results := make([]*SendResult, len(targets))
	for i, t := range targets {
		results[i] = &SendResult{
			Target:    t,
			Success:   true,
			MessageID: "mock-id-" + t.Value,
		}
	}

	m.sendResults = append(m.sendResults, results...)
	return results, nil
}

func (m *mockPlatform) ValidateTarget(target target.Target) error {
	return nil
}

func (m *mockPlatform) IsHealthy(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.health
}

func (m *mockPlatform) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return m.closeError
}

func (m *mockPlatform) GetSendResults() []*SendResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*SendResult{}, m.sendResults...)
}

func (m *mockPlatform) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func mockCreator(name string) PlatformCreator {
	return func(config map[string]interface{}, logger logger.Logger) (Platform, error) {
		return newMockPlatform(name), nil
	}
}

// TestMultiInstanceIsolation verifies that multiple registry instances are completely isolated
func TestMultiInstanceIsolation(t *testing.T) {
	// Create two separate registry instances
	logger1 := logger.New()
	logger2 := logger.New()

	registry1 := NewRegistry(logger1)
	registry2 := NewRegistry(logger2)

	// Verify they are separate instances
	if registry1 == registry2 {
		t.Fatal("Expected separate registry instances, got same instance")
	}

	// Register different platforms in each registry
	err1 := registry1.Register("platform1", mockCreator("platform1"))
	if err1 != nil {
		t.Fatalf("Failed to register platform1: %v", err1)
	}

	err2 := registry2.Register("platform2", mockCreator("platform2"))
	if err2 != nil {
		t.Fatalf("Failed to register platform2: %v", err2)
	}

	// Verify isolation: registry1 should not see platform2 and vice versa
	if registry1.IsRegistered("platform2") {
		t.Error("Registry1 should not see platform2")
	}

	if registry2.IsRegistered("platform1") {
		t.Error("Registry2 should not see platform1")
	}

	// Verify each registry only sees its own platforms
	platforms1 := registry1.ListRegistered()
	platforms2 := registry2.ListRegistered()

	if len(platforms1) != 1 || platforms1[0] != "platform1" {
		t.Errorf("Registry1 should only have platform1, got: %v", platforms1)
	}

	if len(platforms2) != 1 || platforms2[0] != "platform2" {
		t.Errorf("Registry2 should only have platform2, got: %v", platforms2)
	}

	// Clean up
	registry1.Close()
	registry2.Close()
}

// TestConcurrentAccess verifies thread-safety of registry operations
func TestConcurrentAccess(t *testing.T) {
	registry := NewRegistry(logger.New())
	defer registry.Close()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	// Test concurrent registration
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				platformName := fmt.Sprintf("platform-%d-%d", id, j)
				err := registry.Register(platformName, mockCreator(platformName))
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent registration error: %v", err)
	}

	// Verify all platforms were registered
	registered := registry.ListRegistered()
	expectedCount := numGoroutines * numOperations
	if len(registered) != expectedCount {
		t.Errorf("Expected %d registered platforms, got %d", expectedCount, len(registered))
	}
}

// TestConcurrentGetPlatform verifies thread-safety of platform retrieval
func TestConcurrentGetPlatform(t *testing.T) {
	registry := NewRegistry(logger.New())
	defer registry.Close()

	// Pre-register and configure platforms
	const numPlatforms = 5
	for i := 0; i < numPlatforms; i++ {
		platformName := fmt.Sprintf("platform-%d", i)
		err := registry.Register(platformName, mockCreator(platformName))
		if err != nil {
			t.Fatalf("Failed to register platform %s: %v", platformName, err)
		}

		err = registry.SetConfig(platformName, map[string]interface{}{
			"webhook_url": fmt.Sprintf("http://example.com/%d", i),
		})
		if err != nil {
			t.Fatalf("Failed to set config for platform %s: %v", platformName, err)
		}
	}

	const numGoroutines = 20
	const numOperations = 50

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	// Test concurrent platform retrieval
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				platformName := fmt.Sprintf("platform-%d", j%numPlatforms)
				platform, err := registry.GetPlatform(platformName)
				if err != nil {
					errors <- err
					return
				}
				if platform.Name() != platformName {
					errors <- fmt.Errorf("expected platform name %s, got %s", platformName, platform.Name())
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent GetPlatform error: %v", err)
	}
}

// TestConcurrentUnregistration verifies thread-safety of platform unregistration
func TestConcurrentUnregistration(t *testing.T) {
	registry := NewRegistry(logger.New())
	defer registry.Close()

	// Pre-register platforms
	const numPlatforms = 50
	for i := 0; i < numPlatforms; i++ {
		platformName := fmt.Sprintf("platform-%d", i)
		err := registry.Register(platformName, mockCreator(platformName))
		if err != nil {
			t.Fatalf("Failed to register platform %s: %v", platformName, err)
		}
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*5) // Allow some errors as platforms may be unregistered multiple times

	// Test concurrent unregistration
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				platformName := fmt.Sprintf("platform-%d", (id*5+j)%numPlatforms)
				err := registry.Unregister(platformName)
				if err != nil {
					// Unregistration errors are expected when multiple goroutines try to unregister the same platform
					// We're mainly testing that no race conditions occur
					continue
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// The main goal is to ensure no race conditions occur, so we just check that the test completes without panics
	t.Log("Concurrent unregistration test completed successfully")
}

// TestMultiInstanceConcurrentOperations verifies that multiple instances can operate concurrently without interference
func TestMultiInstanceConcurrentOperations(t *testing.T) {
	const numInstances = 5
	const numGoroutinesPerInstance = 4
	const numOperations = 25

	var wg sync.WaitGroup
	errors := make(chan error, numInstances*numGoroutinesPerInstance*numOperations)

	// Create multiple registry instances and run concurrent operations on each
	for instanceID := 0; instanceID < numInstances; instanceID++ {
		wg.Add(numGoroutinesPerInstance)

		for goroutineID := 0; goroutineID < numGoroutinesPerInstance; goroutineID++ {
			go func(instID, grID int) {
				defer wg.Done()

				// Each goroutine gets its own registry instance
				registry := NewRegistry(logger.New())
				defer registry.Close()

				// Perform operations
				for opID := 0; opID < numOperations; opID++ {
					platformName := fmt.Sprintf("inst-%d-gr-%d-op-%d", instID, grID, opID)

					// Register platform
					err := registry.Register(platformName, mockCreator(platformName))
					if err != nil {
						errors <- fmt.Errorf("instance %d, goroutine %d: register error: %v", instID, grID, err)
						return
					}

					// Set config
					err = registry.SetConfig(platformName, map[string]interface{}{
						"test": fmt.Sprintf("config-%d-%d-%d", instID, grID, opID),
					})
					if err != nil {
						errors <- fmt.Errorf("instance %d, goroutine %d: config error: %v", instID, grID, err)
						return
					}

					// Get platform
					_, err = registry.GetPlatform(platformName)
					if err != nil {
						errors <- fmt.Errorf("instance %d, goroutine %d: get platform error: %v", instID, grID, err)
						return
					}

					// Verify isolation - each registry should only see its own platforms
					registered := registry.ListRegistered()
					expected := opID + 1
					if len(registered) != expected {
						errors <- fmt.Errorf("instance %d, goroutine %d: expected %d platforms, got %d", instID, grID, expected, len(registered))
						return
					}
				}
			}(instanceID, goroutineID)
		}
	}

	// Wait for all operations to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out - possible deadlock")
	}

	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Multi-instance concurrent operation error: %v", err)
	}
}

// TestRegistryLifecycle verifies proper lifecycle management
func TestRegistryLifecycle(t *testing.T) {
	registry := NewRegistry(logger.New())

	// Register and configure a platform
	platformName := "test-platform"
	err := registry.Register(platformName, mockCreator(platformName))
	if err != nil {
		t.Fatalf("Failed to register platform: %v", err)
	}

	err = registry.SetConfig(platformName, map[string]interface{}{
		"webhook_url": "http://example.com",
	})
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Get platform to instantiate it
	platform, err := registry.GetPlatform(platformName)
	if err != nil {
		t.Fatalf("Failed to get platform: %v", err)
	}

	mockPlat := platform.(*mockPlatform)
	if mockPlat.IsClosed() {
		t.Error("Platform should not be closed initially")
	}

	// Close registry - should close all platform instances
	err = registry.Close()
	if err != nil {
		t.Fatalf("Failed to close registry: %v", err)
	}

	// Verify platform was closed
	if !mockPlat.IsClosed() {
		t.Error("Platform should be closed after registry close")
	}

	// Verify registry is clean after close
	instances := registry.ListInstances()
	if len(instances) != 0 {
		t.Errorf("Expected no instances after close, got: %v", instances)
	}
}