// Package async provides basic tests for the enhanced callback management system
package async

import (
	"errors"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
)

func TestCallbackRegistry_BasicCreation(t *testing.T) {
	testLogger := logger.New()
	registry := NewCallbackRegistry(testLogger)
	defer registry.Shutdown(5 * time.Second)

	if registry == nil {
		t.Error("Expected registry to be created")
	}

	// Test that all components are initialized
	if registry.executor == nil {
		t.Error("Expected executor to be initialized")
	}

	if registry.tracker == nil {
		t.Error("Expected tracker to be initialized")
	}

	if registry.errorRecovery == nil {
		t.Error("Expected error recovery to be initialized")
	}

	if registry.performanceTracker == nil {
		t.Error("Expected performance tracker to be initialized")
	}
}

func TestCallbackRegistry_GlobalCallbacks(t *testing.T) {
	testLogger := logger.New()
	registry := NewCallbackRegistry(testLogger)
	defer registry.Shutdown(5 * time.Second)

	// Initially no global callbacks
	if registry.HasGlobalCallbacks() {
		t.Error("Expected no global callbacks initially")
	}

	// Register global callbacks
	callbacks := &Callbacks{
		OnResult: func(r *receipt.Receipt) {
			// Test callback
		},
	}
	registry.RegisterGlobalCallbacks(callbacks)

	// Should now have global callbacks
	if !registry.HasGlobalCallbacks() {
		t.Error("Expected global callbacks to be registered")
	}

	// Clear global callbacks
	registry.ClearGlobalCallbacks()
	if registry.HasGlobalCallbacks() {
		t.Error("Expected global callbacks to be cleared")
	}
}

func TestCallbackRegistry_MessageCallbacks(t *testing.T) {
	testLogger := logger.New()
	registry := NewCallbackRegistry(testLogger)
	defer registry.Shutdown(5 * time.Second)

	messageID := "test-message"

	// Initially no message callbacks
	if registry.GetCallbackCount() != 0 {
		t.Error("Expected no message callbacks initially")
	}

	// Register message callbacks
	callbacks := &Callbacks{
		OnResult: func(r *receipt.Receipt) {
			// Test callback
		},
	}
	registry.RegisterMessageCallbacks(messageID, callbacks)

	// Should now have message callbacks
	if registry.GetCallbackCount() != 1 {
		t.Error("Expected 1 message callback to be registered")
	}

	// Cleanup message callbacks
	registry.CleanupMessageCallbacks(messageID)
	if registry.GetCallbackCount() != 0 {
		t.Error("Expected message callbacks to be cleaned up")
	}
}

func TestCallbackExecutor_BasicExecution(t *testing.T) {
	testLogger := logger.New()
	executor := NewCallbackExecutor(testLogger)
	defer executor.Shutdown(5 * time.Second)

	callback := func(r *receipt.Receipt) {
		// Test callback execution - in real implementation this would be tracked
	}

	// Execute callback
	receipt := &receipt.Receipt{MessageID: "test"}
	execution := executor.ExecuteAsync(CallbackTypeResult, "test", callback, receipt)

	if execution == nil {
		t.Error("Expected execution to be returned")
	}

	if execution.Type != CallbackTypeResult {
		t.Error("Expected callback type to be result")
	}

	// Give time for execution
	time.Sleep(100 * time.Millisecond)

	// The callback should have been executed
	// Note: In a real test environment, we might need to verify this differently
	// since the execution is asynchronous
}

func TestCallbackTracker_BasicTracking(t *testing.T) {
	testLogger := logger.New()
	tracker := NewCallbackTracker(testLogger)

	// Create test execution
	execution := &CallbackExecution{
		ID:        "test-exec",
		MessageID: "test-msg",
		Type:      CallbackTypeResult,
		Status:    CallbackStatusSuccess,
		StartedAt: time.Now(),
		Duration:  10 * time.Millisecond,
	}

	// Track execution
	tracker.TrackExecution(execution)

	// Verify execution was tracked
	retrieved, exists := tracker.GetExecution("test-exec")
	if !exists {
		t.Error("Expected execution to be tracked")
	}

	if retrieved.ID != execution.ID {
		t.Error("Expected to retrieve the same execution")
	}

	// Test stats
	stats := tracker.GetStats()
	if stats["total_executions"].(int64) != 1 {
		t.Error("Expected total executions to be 1")
	}
}

func TestErrorRecoveryManager_BasicRetry(t *testing.T) {
	testLogger := logger.New()
	manager := NewErrorRecoveryManager(testLogger)
	defer manager.Shutdown(5 * time.Second)

	// Set retry policy
	policy := &CallbackRetryPolicy{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}

	manager.SetRetryPolicy(CallbackTypeResult, policy)

	// Create failed execution
	execution := &CallbackExecution{
		ID:         "test-retry",
		MessageID:  "test-msg",
		Type:       CallbackTypeResult,
		Status:     CallbackStatusFailed,
		RetryCount: 0,
	}

	// Handle failure
	testError := errors.New("test failure")
	manager.HandleFailure(execution, testError)

	// Verify failed callback was registered for retry
	failedCallbacks := manager.GetFailedCallbacks()
	if len(failedCallbacks) != 1 {
		t.Errorf("Expected 1 failed callback for retry, got %d", len(failedCallbacks))
	}
}

func TestPerformanceTracker_BasicMetrics(t *testing.T) {
	testLogger := logger.New()
	tracker := NewPerformanceTracker(testLogger)

	// Record execution
	execution := &CallbackExecution{
		Type:     CallbackTypeResult,
		Status:   CallbackStatusSuccess,
		Duration: 10 * time.Millisecond,
	}

	tracker.RecordExecution(execution)

	// Get metrics
	metrics, exists := tracker.GetMetrics(CallbackTypeResult)
	if !exists {
		t.Error("Expected metrics to exist for CallbackTypeResult")
	}

	if metrics.TotalExecutions != 1 {
		t.Errorf("Expected 1 total execution, got %d", metrics.TotalExecutions)
	}

	if metrics.SuccessfulExecutions != 1 {
		t.Errorf("Expected 1 successful execution, got %d", metrics.SuccessfulExecutions)
	}
}

func TestCallbackRegistry_EnhancedFeatures(t *testing.T) {
	testLogger := logger.New()
	registry := NewCallbackRegistry(testLogger)
	defer registry.Shutdown(5 * time.Second)

	messageID := "test-enhanced"
	callbacks := &Callbacks{
		OnResult: func(r *receipt.Receipt) {
			// Test callback
		},
	}

	registry.RegisterMessageCallbacks(messageID, callbacks)

	// Test enhanced features
	registry.SetCallbackTimeout(messageID, 5*time.Second)
	registry.SetCallbackPriority(messageID, 10)
	registry.AddCallbackMetadata(messageID, "test_key", "test_value")

	policy := &CallbackRetryPolicy{
		MaxRetries:      3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     10 * time.Second,
		Multiplier:      2.0,
		Jitter:          true,
	}
	registry.SetCallbackRetryPolicy(messageID, policy)

	// Test stats
	stats := registry.GetCallbackStats()
	if stats == nil {
		t.Error("Expected stats to be returned")
	}

	// Test health status
	healthStatus := registry.GetHealthStatus()
	if healthStatus == nil {
		t.Error("Expected health status to be returned")
	}

	// Test maintenance
	registry.PerformMaintenance()
}

func TestCallbackRegistry_TriggerCallbacks(t *testing.T) {
	testLogger := logger.New()
	registry := NewCallbackRegistry(testLogger)
	defer registry.Shutdown(5 * time.Second)

	// Register global callbacks
	callbacks := &Callbacks{
		OnResult: func(r *receipt.Receipt) {
			// Test callback
		},
		OnError: func(m *message.Message, err error) {
			// Test callback
		},
	}
	registry.RegisterGlobalCallbacks(callbacks)

	// Trigger callbacks
	receipt := &receipt.Receipt{MessageID: "test-msg"}
	registry.TriggerResult(receipt)

	msg := &message.Message{ID: "test-msg"}
	registry.TriggerError(msg, errors.New("test error"))

	registry.TriggerProgress(5, 10)

	summary := &BatchSummary{BatchID: "test-batch"}
	registry.TriggerComplete(summary)

	// Give callbacks time to execute
	time.Sleep(200 * time.Millisecond)

	// Verify callbacks were triggered (check executor stats)
	stats := registry.GetCallbackStats()
	executorStats := stats["executor"].(map[string]interface{})

	// Should have triggered at least some callbacks
	totalExecuted := executorStats["total_executed"].(int64)
	if totalExecuted == 0 {
		t.Error("Expected some callbacks to be executed")
	}
}

func TestCallbackRegistry_Shutdown(t *testing.T) {
	testLogger := logger.New()
	registry := NewCallbackRegistry(testLogger)

	// Register some callbacks
	callbacks := &Callbacks{
		OnResult: func(r *receipt.Receipt) {},
	}
	registry.RegisterGlobalCallbacks(callbacks)
	registry.RegisterMessageCallbacks("test-msg", callbacks)

	// Verify callbacks are registered
	if !registry.HasGlobalCallbacks() {
		t.Error("Expected global callbacks to be registered")
	}

	if registry.GetCallbackCount() != 1 {
		t.Error("Expected 1 message callback")
	}

	// Shutdown
	err := registry.Shutdown(5 * time.Second)
	if err != nil {
		t.Errorf("Expected clean shutdown, got error: %v", err)
	}

	// Verify callbacks were cleared
	if registry.HasGlobalCallbacks() {
		t.Error("Expected global callbacks to be cleared after shutdown")
	}

	if registry.GetCallbackCount() != 0 {
		t.Error("Expected message callbacks to be cleared after shutdown")
	}
}