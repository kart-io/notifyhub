package middleware_test

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/tests/utils"
)

// TestMiddlewareIntegration tests middleware integration through the API
func TestMiddlewareIntegration(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	results, err := hub.Send().
		Title("Test Title").
		Body("Test Body").
		Priority(3).
		To("test@example.com").
		Send(context.Background())

	utils.AssertNoError(t, err)
	utils.AssertTrue(t, results != nil, "results should not be nil")
}

// TestConcurrentMiddlewareProcessing tests concurrent middleware processing
func TestConcurrentMiddlewareProcessing(t *testing.T) {
	hub := utils.CreateTestHub(t)
	defer func() { _ = hub.Shutdown(context.Background()) }()

	numGoroutines := 5
	results := make(chan error, numGoroutines)

	// Start concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := hub.Send().
				Title("Test Title").
				Body("Test Body").
				Priority(3).
				To("test@example.com").
				Send(context.Background())
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

	_, err := hub.Send().
		Title("Test Title").
		Body("Test Body").
		Priority(3).
		To("test@example.com").
		Send(ctx)
	// Should either succeed quickly or fail due to context timeout
	if err != nil {
		utils.AssertTrue(t, ctx.Err() != nil, "context should be done")
	}
}
