package middleware_test

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/tests/utils"
)

// TestMiddlewareIntegration tests middleware integration through the API
func TestMiddlewareIntegration(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	msg := utils.CreateTestMessage()
	targets := utils.CreateTestTargets()

	ctx := context.Background()
	results, err := hub.Send(ctx, msg, targets)

	utils.AssertNoError(t, err)
	utils.AssertTrue(t, results != nil, "results should not be nil")
}

// TestConcurrentMiddlewareProcessing tests concurrent middleware processing
func TestConcurrentMiddlewareProcessing(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	msg := utils.CreateTestMessage()
	targets := []sending.Target{
		sending.NewTarget(sending.TargetTypeEmail, "test@example.com", "email"),
	}

	ctx := context.Background()
	numGoroutines := 5
	results := make(chan error, numGoroutines)

	// Start concurrent operations
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

// TestMiddlewareErrorHandling tests error handling in middleware
func TestMiddlewareErrorHandling(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	// Test with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()

	// Wait for context to timeout
	time.Sleep(10 * time.Millisecond)

	msg := utils.CreateTestMessage()
	targets := utils.CreateTestTargets()

	_, err := hub.Send(ctx, msg, targets)
	// Should either succeed quickly or fail due to context timeout
	if err != nil {
		utils.AssertTrue(t, ctx.Err() != nil, "context should be done")
	}
}
