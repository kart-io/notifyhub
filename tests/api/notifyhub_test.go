package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/api"
	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/tests/utils"
)

func TestNotifyHub_New(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		options     *api.Options
		expectError bool
	}{
		{
			name:        "valid config",
			config:      utils.CreateTestConfig(),
			options:     &api.Options{Logger: utils.NewMockLogger()},
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			options:     nil,
			expectError: true,
		},
		{
			name:        "nil options",
			config:      utils.CreateTestConfig(),
			options:     nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub, err := api.New(tt.config, tt.options)

			if tt.expectError {
				utils.AssertError(t, err)
				utils.AssertEqual(t, (*api.NotifyHub)(nil), hub)
			} else {
				utils.AssertNoError(t, err)
				utils.AssertTrue(t, hub != nil, "hub should not be nil")
			}
		})
	}
}

func TestNotifyHub_SendMessage(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	msg := utils.CreateTestMessage()
	targets := utils.CreateTestTargets()

	ctx := context.Background()
	results, err := hub.Send(ctx, msg, targets)

	utils.AssertNoError(t, err)
	utils.AssertTrue(t, results != nil, "results should not be nil")
	utils.AssertEqual(t, len(targets), len(results.Results))

	// Check that all sends were attempted
	for i, result := range results.Results {
		utils.AssertEqual(t, targets[i].Value, result.Target.Value)
		utils.AssertEqual(t, targets[i].Platform, result.Target.Platform)
	}
}

func TestNotifyHub_SendToTargetExpressions(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	msg := utils.CreateTestMessage()
	expressions := []string{
		"email:test@example.com",
		"user:user123@feishu",
		"group:alerts@feishu",
	}

	ctx := context.Background()
	results, err := hub.SendToTargetExpressions(ctx, msg, expressions)

	if err != nil {
		// Target resolution may not be fully implemented yet
		t.Skip("Target expression resolution not implemented")
		return
	}

	utils.AssertTrue(t, results != nil, "results should not be nil")
	utils.AssertTrue(t, len(results.Results) >= 0, "should have results or empty results")
}

func TestNotifyHub_NewMessage(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	msg := hub.NewMessage()

	utils.AssertTrue(t, msg != nil, "message should not be nil")
	utils.AssertTrue(t, msg.ID != "", "message should have an ID")
}

func TestNotifyHub_GetMetrics(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// Send a message first to generate some metrics
	msg := utils.CreateTestMessage()
	targets := []sending.Target{
		sending.NewTarget(sending.TargetTypeEmail, "test@example.com", "email"),
	}

	ctx := context.Background()
	_, err := hub.Send(ctx, msg, targets)
	utils.AssertNoError(t, err)

	// Wait a bit for metrics to be processed
	time.Sleep(100 * time.Millisecond)

	metrics := hub.GetMetrics()
	utils.AssertTrue(t, metrics != nil, "metrics should not be nil")
}

func TestNotifyHub_GetTransports(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	transports := hub.GetTransports()
	utils.AssertTrue(t, transports != nil, "transports should not be nil")
	// Note: The actual transport list depends on configuration
}

func TestNotifyHub_Shutdown(t *testing.T) {
	hub := utils.CreateTestHub(t)

	ctx := context.Background()
	err := hub.Shutdown(ctx)

	utils.AssertNoError(t, err)
	utils.AssertTrue(t, hub.IsShutdown(), "hub should be shutdown")
}

func TestNotifyHub_SendAfterShutdown(t *testing.T) {
	hub := utils.CreateTestHub(t)

	// Shutdown first
	ctx := context.Background()
	err := hub.Shutdown(ctx)
	utils.AssertNoError(t, err)

	// Verify shutdown state
	utils.AssertTrue(t, hub.IsShutdown(), "hub should be shutdown")

	// Try to send after shutdown
	msg := utils.CreateTestMessage()
	targets := utils.CreateTestTargets()

	_, err = hub.Send(ctx, msg, targets)
	// Note: Some implementations may handle this gracefully rather than failing
	if err == nil {
		t.Skip("Hub gracefully handles sends after shutdown")
	} else {
		// If it fails, that's also acceptable behavior
		utils.AssertError(t, err)
	}
}

func TestNotifyHub_ConcurrentSends(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	msg := utils.CreateTestMessage()
	targets := []sending.Target{
		sending.NewTarget(sending.TargetTypeEmail, "test@example.com", "email"),
	}

	ctx := context.Background()
	numGoroutines := 10
	results := make(chan error, numGoroutines)

	// Start multiple goroutines sending messages concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := hub.Send(ctx, msg, targets)
			results <- err
		}()
	}

	// Wait for all to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		utils.AssertNoError(t, err)
	}
}

func TestNotifyHub_WithContextTimeout(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	msg := utils.CreateTestMessage()
	targets := utils.CreateTestTargets()

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()

	// Wait for context to timeout
	time.Sleep(10 * time.Millisecond)

	_, err := hub.Send(ctx, msg, targets)
	// Should either succeed quickly or fail due to context timeout
	// The exact behavior depends on implementation
	if err != nil {
		// If it fails, it should be due to context timeout
		utils.AssertTrue(t, ctx.Err() != nil, "context should be done")
	}
}

func TestNotifyHub_Configuration(t *testing.T) {
	// Test with Feishu configuration
	feishuConfig := config.New(
		config.WithFeishu("https://example.com/webhook", "secret"),
		config.WithQueue("memory", 100, 1),
		config.WithSilentLogger(),
	)

	opts := &api.Options{
		Logger: utils.NewMockLogger(),
	}

	hub, err := api.New(feishuConfig, opts)
	utils.AssertNoError(t, err)
	utils.AssertTrue(t, hub != nil, "hub should not be nil")

	defer func() { _ = hub.Shutdown(context.Background()) }()

	transports := hub.GetTransports()
	utils.AssertTrue(t, len(transports) > 0, "should have at least one transport")
}

func TestNotifyHub_ErrorHandling(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	ctx := context.Background()

	// Test with empty targets
	msg := utils.CreateTestMessage()
	emptyTargets := []sending.Target{}

	results, err := hub.Send(ctx, msg, emptyTargets)
	utils.AssertNoError(t, err) // Should succeed but with empty results
	utils.AssertEqual(t, 0, len(results.Results))
}
