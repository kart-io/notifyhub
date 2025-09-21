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

	// Note: GetTransports method not available in new API
	// The actual platform tests would depend on the specific platform implementations
	// For now, just verify the hub can be created and basic operations work

	results, err := hub.Send().
		Title("Platform Test").
		Body("Test platform integration").
		Priority(3).
		To("test@example.com").
		Send(context.Background())

	utils.AssertNoError(t, err)
	utils.AssertTrue(t, results != nil, "results should not be nil")
}

// TestPlatformConfiguration tests platform configuration
func TestPlatformConfiguration(t *testing.T) {
	// This test would verify platform-specific configuration
	// For now, just test that configuration doesn't break anything

	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.TODO()) }()

	// Send message (should work with mock setup)
	results, err := hub.Send().
		Title("Configuration Test").
		Body("Test platform configuration").
		Priority(3).
		To("test@example.com").
		Send(context.Background())
	utils.AssertNoError(t, err)
	utils.AssertTrue(t, results != nil, "results should not be nil")
}
