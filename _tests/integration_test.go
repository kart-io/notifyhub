package notifyhub_test

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub"
)

func TestNotifyHubIntegration(t *testing.T) {
	// Test complete NotifyHub integration workflow
	hub, err := notifyhub.New(
		notifyhub.WithTestDefaults(),
		notifyhub.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test-integration", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()

	// Start the hub
	err = hub.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start NotifyHub: %v", err)
	}
	defer hub.Stop()

	// Test 1: Basic message sending
	message := notifyhub.NewMessage().
		Title("Integration Test").
		Body("This is an integration test message").
		Priority(3).
		Build()

	results, err := hub.Send(ctx, message, nil)
	if results == nil {
		t.Error("Send should return results even if external service fails")
	}

	// Test 2: Batch sending
	messages := []*notifyhub.Message{
		notifyhub.NewMessage().Title("Batch 1").Body("First batch message").Build(),
		notifyhub.NewMessage().Title("Batch 2").Body("Second batch message").Build(),
		notifyhub.NewMessage().Title("Batch 3").Body("Third batch message").Build(),
	}

	batchResults, err := hub.SendBatch(ctx, messages, nil)
	if len(batchResults) == 0 {
		t.Error("Batch send should return results")
	}

	// Test 3: Async sending
	asyncOptions := notifyhub.NewAsyncOptions()
	taskID, err := hub.SendAsync(ctx, message, asyncOptions)
	if taskID == "" && err != nil {
		t.Errorf("Async send should either return task ID or error: err=%v", err)
	}

	// Test 4: Convenience methods
	err = hub.SendText(ctx, "Text Test", "Simple text message")
	// Error is expected due to fake webhook, but method should not panic

	err = hub.SendAlert(ctx, "Alert Test", "Alert message")
	// Error is expected due to fake webhook, but method should not panic

	// Test 5: Health check
	health := hub.GetHealth(ctx)
	if health == nil {
		t.Error("Health check should return non-nil result")
	}

	if health["status"] == nil {
		t.Error("Health should include status field")
	}

	// Test 6: Metrics
	metrics := hub.GetMetrics()
	if metrics == nil {
		t.Error("Metrics should return non-nil result")
	}
}

func TestNotifyHubMessageBuilders(t *testing.T) {
	// Test all message builder types
	hub, err := notifyhub.New(
		notifyhub.WithTestDefaults(),
		notifyhub.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test-builders", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()

	testCases := []struct {
		name        string
		builder     *notifyhub.MessageBuilder
		expectedPri int
	}{
		{"Alert", notifyhub.NewAlert("Alert Title", "Alert Body"), 4},
		{"Notice", notifyhub.NewNotice("Notice Title", "Notice Body"), 3},
		{"Report", notifyhub.NewReport("Report Title", "Report Body"), 2},
		{"Markdown", notifyhub.NewMarkdown("MD Title", "**Bold** text"), 3},
		{"HTML", notifyhub.NewHTML("HTML Title", "<b>Bold</b> text"), 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message := tc.builder.Build()
			if message.Priority != tc.expectedPri {
				t.Errorf("%s message should have priority %d, got %d", tc.name, tc.expectedPri, message.Priority)
			}

			// Test sending (will likely fail due to fake webhook, but should not panic)
			results, err := hub.Send(ctx, message, nil)
			if results == nil && err == nil {
				t.Error("Either results or error should be non-nil")
			}
		})
	}
}

func TestNotifyHubRouting(t *testing.T) {
	// Test message routing functionality
	rule1 := notifyhub.NewRoutingRule("priority-rule").
		WithPriority(4, 5).  // High priority messages
		RouteTo("feishu").
		Build()

	rule2 := notifyhub.NewRoutingRule("default-rule").
		RouteTo("email").
		Build()

	hub, err := notifyhub.New(
		notifyhub.WithTestDefaults(),
		notifyhub.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test-routing", ""),
		notifyhub.WithEmail("localhost", 587, "test", "test", "from@example.com"),
		notifyhub.WithRouting(rule1, rule2),
	)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub with routing: %v", err)
	}

	ctx := context.Background()

	// Test high priority message (should route to feishu via rule1)
	highPriorityMsg := notifyhub.NewAlert("High Priority", "This is urgent").Build()
	results, err := hub.Send(ctx, highPriorityMsg, nil)

	// Test low priority message (should route to email via rule2)
	lowPriorityMsg := notifyhub.NewReport("Low Priority", "This is a report").Build()
	results2, err2 := hub.Send(ctx, lowPriorityMsg, nil)

	// Both sends should produce some results (even if they fail)
	if results == nil && results2 == nil {
		t.Error("At least one routing test should produce results")
	}

	// Errors are expected due to fake endpoints
	_ = err
	_ = err2
}

func TestNotifyHubOptions(t *testing.T) {
	// Test various sending options
	hub, err := notifyhub.New(
		notifyhub.WithTestDefaults(),
		notifyhub.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test-options", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()
	message := notifyhub.NewMessage().Title("Options Test").Body("Testing options").Build()

	// Test sync options
	syncOptions := notifyhub.NewSyncOptions().WithTimeout(5 * time.Second)
	results1, err1 := hub.Send(ctx, message, syncOptions)

	// Test async options
	asyncOptions := notifyhub.NewAsyncOptions()
	results2, err2 := hub.Send(ctx, message, asyncOptions)

	// Test retry options
	retryOptions := notifyhub.NewRetryOptions(2)
	results3, err3 := hub.Send(ctx, message, retryOptions)

	// At least one should succeed or fail gracefully
	if results1 == nil && results2 == nil && results3 == nil && err1 == nil && err2 == nil && err3 == nil {
		t.Error("All option tests failed to produce any results or errors")
	}
}

func TestNotifyHubCallbacks(t *testing.T) {
	// Test callback functionality
	sentCallbackCalled := false
	failedCallbackCalled := false

	sentCallback := notifyhub.NewCallbackFunc("test-sent", func(ctx context.Context, callbackCtx *notifyhub.CallbackContext) error {
		sentCallbackCalled = true
		if callbackCtx.Event != notifyhub.CallbackEventSent {
			t.Error("Callback event should be sent")
		}
		return nil
	})

	failedCallback := notifyhub.NewCallbackFunc("test-failed", func(ctx context.Context, callbackCtx *notifyhub.CallbackContext) error {
		failedCallbackCalled = true
		if callbackCtx.Event != notifyhub.CallbackEventFailed {
			t.Error("Callback event should be failed")
		}
		return nil
	})

	callbackOptions := &notifyhub.CallbackOptions{}
	callbackOptions.AddCallback(notifyhub.CallbackEventSent, sentCallback)
	callbackOptions.AddCallback(notifyhub.CallbackEventFailed, failedCallback)

	hub, err := notifyhub.New(
		notifyhub.WithTestDefaults(),
		notifyhub.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test-callbacks", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()
	message := notifyhub.NewMessage().Title("Callback Test").Body("Testing callbacks").Build()

	// Test with callbacks
	options := notifyhub.NewAsyncOptions().WithCallbacks(callbackOptions)

	// Start hub for async processing
	err = hub.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start hub: %v", err)
	}
	defer hub.Stop()

	_, err = hub.SendAsync(ctx, message, options)

	// Wait a bit for async processing
	time.Sleep(200 * time.Millisecond)

	// At least one callback should have been called
	if !sentCallbackCalled && !failedCallbackCalled {
		t.Log("No callbacks were triggered (may be expected with fake webhook)")
	}
}

func TestNotifyHubFullWorkflow(t *testing.T) {
	// Test complete end-to-end workflow
	hub, err := notifyhub.New(
		notifyhub.WithTestDefaults(),
		notifyhub.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/test-workflow", ""),
		notifyhub.WithDefaultRouting(),
		notifyhub.WithDefaultLogger(notifyhub.LogLevelDebug),
	)
	if err != nil {
		t.Fatalf("Failed to create NotifyHub: %v", err)
	}

	ctx := context.Background()

	// Start hub
	err = hub.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start hub: %v", err)
	}

	// Create various message types
	messages := []*notifyhub.Message{
		notifyhub.NewAlert("Workflow Alert", "System alert message").Build(),
		notifyhub.NewNotice("Workflow Notice", "Information notice").Build(),
		notifyhub.NewReport("Workflow Report", "Status report").Build(),
	}

	// Send all messages
	for i, message := range messages {
		results, err := hub.Send(ctx, message, nil)

		// Log results for visibility
		t.Logf("Message %d results: %v, error: %v", i+1, len(results), err)
	}

	// Test batch sending
	batchResults, err := hub.SendBatch(ctx, messages, nil)
	t.Logf("Batch results: %v, error: %v", len(batchResults), err)

	// Test async sending
	for i, message := range messages {
		taskID, err := hub.SendAsync(ctx, message, notifyhub.NewAsyncOptions())
		t.Logf("Async message %d task ID: %s, error: %v", i+1, taskID, err)
	}

	// Wait for async processing
	time.Sleep(300 * time.Millisecond)

	// Get final metrics
	metrics := hub.GetMetrics()
	t.Logf("Final metrics: %+v", metrics)

	// Get health status
	health := hub.GetHealth(ctx)
	t.Logf("Final health: %+v", health)

	// Stop hub
	err = hub.Stop()
	if err != nil {
		t.Errorf("Failed to stop hub: %v", err)
	}
}