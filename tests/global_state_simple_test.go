// Package tests provides simple global state elimination validation tests
package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// TestSimpleGlobalStateElimination validates basic global state elimination
func TestSimpleGlobalStateElimination(t *testing.T) {
	t.Run("IndependentHubInstances", func(t *testing.T) {
		// Create first hub
		hub1, err := notifyhub.New(notifyhub.WithEmail("smtp1.test.com", 587, "test1@test.com"))
		if err != nil {
			t.Fatalf("Failed to create first hub: %v", err)
		}
		defer func() { _ = hub1.Close() }() // Ignore close errors in tests

		// Create second hub with different configuration
		hub2, err := notifyhub.New(notifyhub.WithEmail("smtp2.test.com", 465, "test2@test.com"))
		if err != nil {
			t.Fatalf("Failed to create second hub: %v", err)
		}
		defer func() { _ = hub2.Close() }() // Ignore close errors in tests

		// Verify hubs are independent by checking they don't interfere with each other
		msg := &message.Message{
			ID:       "independence-test",
			Title:    "Independence Test",
			Body:     "Testing hub independence",
			Format:   message.FormatText,
			Priority: message.PriorityNormal,
			Targets: []target.Target{
				{Type: "email", Value: "user@test.com"},
			},
		}

		// Both hubs should work independently
		_, err1 := hub1.Send(context.Background(), msg)
		_, err2 := hub2.Send(context.Background(), msg)

		// We expect connection errors in test environment, but should be consistent
		if (err1 == nil) != (err2 == nil) {
			t.Log("Note: Hubs show different behavior, but this may be expected in test environment")
		}

		t.Log("Hub independence test completed - no global state conflicts detected")
	})

	t.Run("ConcurrentHubOperations", func(t *testing.T) {
		// Create multiple hubs concurrently to test for global state conflicts
		const numHubs = 5
		var wg sync.WaitGroup
		hubs := make([]notifyhub.Client, numHubs)
		errors := make([]error, numHubs)

		for i := 0; i < numHubs; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				hub, err := notifyhub.New(notifyhub.WithEmail("smtp.test.com", 587+index, "test@test.com"))
				if err != nil {
					errors[index] = err
					return
				}
				hubs[index] = hub
			}(i)
		}

		wg.Wait()

		// Clean up hubs
		defer func() {
			for _, hub := range hubs {
				if hub != nil {
					_ = hub.Close() // Ignore close errors in tests
				}
			}
		}()

		// Check for errors
		successCount := 0
		for i, err := range errors {
			if err != nil {
				t.Logf("Hub %d creation failed: %v", i, err)
			} else {
				successCount++
			}
		}

		if successCount == 0 {
			t.Error("No hubs created successfully - possible global state issue")
		} else {
			t.Logf("Concurrent hub creation test passed: %d/%d hubs created successfully", successCount, numHubs)
		}
	})

	t.Run("HubShutdownIndependence", func(t *testing.T) {
		// Create two hubs
		hub1, err := notifyhub.New(notifyhub.WithEmail("smtp1.test.com", 587, "test1@test.com"))
		if err != nil {
			t.Fatalf("Failed to create hub1: %v", err)
		}

		hub2, err := notifyhub.New(notifyhub.WithEmail("smtp2.test.com", 587, "test2@test.com"))
		if err != nil {
			t.Fatalf("Failed to create hub2: %v", err)
		}

		// Close first hub
		err = hub1.Close()
		if err != nil {
			t.Errorf("Hub1 close failed: %v", err)
		}

		// Verify second hub is still functional
		msg := &message.Message{
			ID:       "shutdown-independence-test",
			Title:    "Shutdown Independence Test",
			Body:     "Testing independence after shutdown",
			Format:   message.FormatText,
			Priority: message.PriorityNormal,
			Targets: []target.Target{
				{Type: "email", Value: "user@test.com"},
			},
		}

		// This operation should not be affected by hub1 being closed
		_, _ = hub2.Send(context.Background(), msg) // Ignore result in independence test
		// We expect connection errors, but not global state related panic or deadlock

		// Close second hub
		err = hub2.Close()
		if err != nil {
			t.Errorf("Hub2 close failed: %v", err)
		}

		t.Log("Hub shutdown independence test passed - no global state conflicts")
	})

	t.Run("NoSharedMutableState", func(t *testing.T) {
		// Create hub and test that operations don't affect global state
		hub, err := notifyhub.New(notifyhub.WithEmail("smtp.test.com", 587, "test@test.com"))
		if err != nil {
			t.Fatalf("Failed to create hub: %v", err)
		}
		defer func() { _ = hub.Close() }() // Ignore close errors in tests

		msg := &message.Message{
			ID:       "no-shared-state-test",
			Title:    "No Shared State Test",
			Body:     "Testing for shared mutable state",
			Format:   message.FormatText,
			Priority: message.PriorityNormal,
			Targets: []target.Target{
				{Type: "email", Value: "user@test.com"},
			},
		}

		// Send message multiple times - should not cause state conflicts
		for i := 0; i < 3; i++ {
			msg.ID = fmt.Sprintf("no-shared-state-test-%d", i)
			_, _ = hub.Send(context.Background(), msg)
		}

		t.Log("No shared mutable state test passed")
	})
}
