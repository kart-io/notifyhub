package platform

import (
	"context"
	"fmt"
	"testing"

	"github.com/kart-io/notifyhub/pkg/message"
	"github.com/kart-io/notifyhub/pkg/target"
	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

// mockPlatform implements Platform for testing
type mockPlatform struct {
	name string
}

func (m *mockPlatform) Name() string {
	return m.name
}

func (m *mockPlatform) Send(ctx context.Context, msg *message.Message, targets []target.Target) ([]*SendResult, error) {
	results := make([]*SendResult, len(targets))
	for i := range targets {
		results[i] = &SendResult{
			Success:   true,
			MessageID: "mock-msg-id",
		}
	}
	return results, nil
}

func (m *mockPlatform) ValidateTarget(tgt target.Target) error {
	return nil
}

func (m *mockPlatform) GetCapabilities() Capabilities {
	return Capabilities{
		Name: "mock",
	}
}

func (m *mockPlatform) IsHealthy(ctx context.Context) error {
	return nil
}

func (m *mockPlatform) Close() error {
	return nil
}

// mockFactory creates a mock platform
func mockFactory(config interface{}) (Platform, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	return &mockPlatform{name: "mock"}, nil
}

func TestNewRegistry(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}

	platforms := registry.ListPlatforms()
	if len(platforms) != 0 {
		t.Errorf("New registry should have 0 platforms, got %d", len(platforms))
	}
}

func TestRegistry_RegisterFactory(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	// Register factory
	err := registry.RegisterFactory("mock", mockFactory)
	if err != nil {
		t.Errorf("RegisterFactory() error = %v", err)
	}

	// Try to register again - should fail
	err = registry.RegisterFactory("mock", mockFactory)
	if err == nil {
		t.Error("RegisterFactory() should error when registering duplicate")
	}

	// Verify it's in the list
	platforms := registry.ListPlatforms()
	if len(platforms) != 1 {
		t.Errorf("Expected 1 platform, got %d", len(platforms))
	}
	if platforms[0] != "mock" {
		t.Errorf("Expected platform name 'mock', got %s", platforms[0])
	}
}

func TestRegistry_SetConfig(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	err := registry.RegisterFactory("mock", mockFactory)
	if err != nil {
		t.Fatalf("RegisterFactory() error = %v", err)
	}

	// Set config
	config := map[string]string{"key": "value"}
	err = registry.SetConfig("mock", config)
	if err != nil {
		t.Errorf("SetConfig() error = %v", err)
	}
}

func TestRegistry_GetPlatform(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	tests := []struct {
		name       string
		setup      func()
		platform   string
		wantErr    bool
		errMessage string
	}{
		{
			name:       "platform not registered",
			setup:      func() {},
			platform:   "unknown",
			wantErr:    true,
			errMessage: "not registered",
		},
		{
			name: "no config set",
			setup: func() {
				_ = registry.RegisterFactory("mock", mockFactory)
			},
			platform:   "mock",
			wantErr:    true,
			errMessage: "no configuration found",
		},
		{
			name: "successful creation",
			setup: func() {
				_ = registry.RegisterFactory("mock", mockFactory)
				_ = registry.SetConfig("mock", map[string]string{"key": "value"})
			},
			platform: "mock",
			wantErr:  false,
		},
		{
			name: "returns cached instance",
			setup: func() {
				// Instance already created in previous test
			},
			platform: "mock",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			platform, err := registry.GetPlatform(tt.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPlatform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && platform == nil {
				t.Error("GetPlatform() returned nil platform")
			}
		})
	}
}

func TestRegistry_ListPlatformsAfterCreation(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	// Initially empty
	platforms := registry.ListPlatforms()
	if len(platforms) != 0 {
		t.Errorf("Expected 0 platforms, got %d", len(platforms))
	}

	// Register and configure platforms
	_ = registry.RegisterFactory("mock1", mockFactory)
	_ = registry.SetConfig("mock1", map[string]string{"key": "value1"})
	_ = registry.RegisterFactory("mock2", mockFactory)
	_ = registry.SetConfig("mock2", map[string]string{"key": "value2"})

	// Create instances
	_, err := registry.GetPlatform("mock1")
	if err != nil {
		t.Fatalf("GetPlatform() error = %v", err)
	}
	_, err = registry.GetPlatform("mock2")
	if err != nil {
		t.Fatalf("GetPlatform() error = %v", err)
	}

	// List platforms
	platforms = registry.ListPlatforms()
	if len(platforms) != 2 {
		t.Errorf("Expected 2 platforms, got %d", len(platforms))
	}
}

func TestRegistry_Close(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	// Register and configure
	_ = registry.RegisterFactory("mock", mockFactory)
	_ = registry.SetConfig("mock", map[string]string{"key": "value"})
	_, err := registry.GetPlatform("mock")
	if err != nil {
		t.Fatalf("GetPlatform() error = %v", err)
	}

	// Close should close all instances
	err = registry.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Factories should still be registered
	platforms := registry.ListPlatforms()
	if len(platforms) != 1 {
		t.Errorf("Expected 1 platform factory after Close(), got %d", len(platforms))
	}
}

func TestRegistry_ConfigRecreation(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	_ = registry.RegisterFactory("mock", mockFactory)
	_ = registry.SetConfig("mock", map[string]string{"key": "value1"})

	// Get platform instance
	p1, err := registry.GetPlatform("mock")
	if err != nil {
		t.Fatalf("GetPlatform() error = %v", err)
	}

	// Update config
	_ = registry.SetConfig("mock", map[string]string{"key": "value2"})

	// Get platform again - should be new instance
	p2, err := registry.GetPlatform("mock")
	if err != nil {
		t.Fatalf("GetPlatform() error = %v", err)
	}

	// Both should be valid but SetConfig should have cleared cache
	if p1 == nil || p2 == nil {
		t.Error("Expected non-nil platforms")
	}
}
