package platforms_test

import (
	"context"
	"testing"

	"github.com/kart-io/notifyhub/tests/utils"
)

// TestPlatformIntegration tests basic platform integration
func TestPlatformIntegration(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.TODO()) }()

	// Get available transports
	transports := hub.GetTransports()
	utils.AssertTrue(t, transports != nil, "transports should not be nil")

	// The actual platform tests would depend on the specific platform implementations
	// For now, just verify the hub can be created and basic operations work
}

// TestPlatformConfiguration tests platform configuration
func TestPlatformConfiguration(t *testing.T) {
	// This test would verify platform-specific configuration
	// For now, just test that configuration doesn't break anything

	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.TODO()) }()

	msg := utils.CreateTestMessage()
	targets := utils.CreateTestTargets()

	// Send message (should work with mock setup)
	ctx := context.Background()
	results, err := hub.Send(ctx, msg, targets)
	utils.AssertNoError(t, err)
	utils.AssertTrue(t, results != nil, "results should not be nil")
}
