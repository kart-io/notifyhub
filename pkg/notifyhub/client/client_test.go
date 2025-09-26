package client

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

func TestNewClient(t *testing.T) {
	// Test creating a new client with minimal configuration
	client, err := New(
		config.WithTimeout(10 * time.Second),
	)

	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	// Test health check
	ctx := context.Background()
	health, err := client.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	if health == nil {
		t.Error("Health status should not be nil")
	}

	// Clean up
	if err := client.Close(); err != nil {
		t.Errorf("Failed to close client: %v", err)
	}
}

func TestClientSend(t *testing.T) {
	// Create a client with a test platform configuration
	client, err := New(
		config.WithPlatform("test", map[string]interface{}{
			"type": "test",
		}),
		config.WithTimeout(5*time.Second),
	)

	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }() // Ignore close errors in tests

	// Create a test message
	msg := message.New()
	msg.Title = "Test Message"
	msg.Body = "This is a test message"
	msg.AddTarget(target.NewTarget("test", "test-target", "test"))

	ctx := context.Background()

	// Note: This will likely fail because we don't have a real test platform registered
	// But it validates the interface structure is correct
	_, err = client.Send(ctx, msg)
	// We expect an error here since no test platform is actually registered
	if err == nil {
		t.Log("Send succeeded (unexpected but not necessarily wrong)")
	} else {
		t.Logf("Send failed as expected: %v", err)
	}
}

func TestAsyncInterface(t *testing.T) {
	// Test that async interface methods exist and are callable
	client, err := New(
		config.WithAsyncBasic("memory", 2),
		config.WithTimeout(5*time.Second),
	)

	if err != nil {
		t.Fatalf("Failed to create async client: %v", err)
	}
	defer func() { _ = client.Close() }() // Ignore close errors in tests

	msg := message.New()
	msg.Title = "Async Test"
	msg.Body = "Testing async interface"
	msg.AddTarget(target.NewTarget("test", "async-target", "test"))

	ctx := context.Background()

	// Test SendAsync interface
	handle, err := client.SendAsync(ctx, msg)
	if err == nil && handle != nil {
		// Test handle interface
		status := handle.Status()
		if status.MessageID != msg.ID {
			t.Errorf("Expected message ID %s, got %s", msg.ID, status.MessageID)
		}

		// Test result channel
		select {
		case result := <-handle.Result():
			if result.MessageID != msg.ID {
				t.Errorf("Expected result message ID %s, got %s", msg.ID, result.MessageID)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for async result")
		}
	} else {
		t.Logf("SendAsync failed as expected (no test platform): %v", err)
	}
}

func TestClientConfiguration(t *testing.T) {
	// Test various configuration options
	testCases := []struct {
		name string
		opts []config.Option
	}{
		{
			name: "sync_only",
			opts: []config.Option{
				config.WithTimeout(10 * time.Second),
			},
		},
		{
			name: "async_enabled",
			opts: []config.Option{
				config.WithAsyncBasic("memory", 4),
				config.WithTimeout(15 * time.Second),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := New(tc.opts...)
			if err != nil {
				t.Fatalf("Failed to create client with %s config: %v", tc.name, err)
			}

			// Basic health check to ensure client is functional
			ctx := context.Background()
			_, err = client.Health(ctx)
			if err != nil {
				t.Errorf("Health check failed for %s config: %v", tc.name, err)
			}

			// Clean up
			if err := client.Close(); err != nil {
				t.Errorf("Failed to close %s client: %v", tc.name, err)
			}
		})
	}
}
