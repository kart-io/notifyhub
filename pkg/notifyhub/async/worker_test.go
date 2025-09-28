package async

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// TestLogger implements a simple logger for testing
type TestLogger struct{}

func (l *TestLogger) LogMode(level logger.LogLevel) logger.Logger { return l }
func (l *TestLogger) Debug(msg string, args ...any)              {}
func (l *TestLogger) Info(msg string, args ...any)               {}
func (l *TestLogger) Warn(msg string, args ...any)               {}
func (l *TestLogger) Error(msg string, args ...any)              {}

func NewTestLogger() logger.Logger {
	return &TestLogger{}
}

// MockDispatcher for testing
type MockDispatcher struct {
	processDelay time.Duration
	shouldFail   bool
	mutex        sync.RWMutex
}

func NewMockDispatcher() *MockDispatcher {
	return &MockDispatcher{
		processDelay: 50 * time.Millisecond,
		shouldFail:   false,
	}
}

func (m *MockDispatcher) Dispatch(ctx context.Context, msg *message.Message) (*receipt.Receipt, error) {
	m.mutex.RLock()
	delay := m.processDelay
	fail := m.shouldFail
	m.mutex.RUnlock()

	time.Sleep(delay)

	if fail {
		return nil, fmt.Errorf("test error")
	}

	return &receipt.Receipt{
		MessageID:  msg.ID,
		Status:     "success",
		Successful: 1,
		Failed:     0,
		Total:      1,
		Timestamp:  time.Now(),
	}, nil
}

func (m *MockDispatcher) Close() error {
	return nil
}

func (m *MockDispatcher) RegisterPlatform(name string, creator platform.PlatformCreator) {
	// Mock implementation
}

func (m *MockDispatcher) Health(ctx context.Context) (map[string]string, error) {
	return map[string]string{"test": "healthy"}, nil
}

func (m *MockDispatcher) SetProcessDelay(delay time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.processDelay = delay
}

func (m *MockDispatcher) SetShouldFail(fail bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.shouldFail = fail
}

func TestWorkerPoolConfig(t *testing.T) {
	config := DefaultWorkerPoolConfig()

	if config.MinWorkers != 2 {
		t.Errorf("Expected MinWorkers to be 2, got %d", config.MinWorkers)
	}

	if config.TargetLoad != 0.7 {
		t.Errorf("Expected TargetLoad to be 0.7, got %f", config.TargetLoad)
	}

	if config.TaskBatchSize != 10 {
		t.Errorf("Expected TaskBatchSize to be 10, got %d", config.TaskBatchSize)
	}
}

func TestWorkerPoolCreation(t *testing.T) {
	config := &WorkerPoolConfig{
		MinWorkers:      2,
		MaxWorkers:      4,
		TargetLoad:      0.7,
		ScaleUpDelay:    5 * time.Second,
		ScaleDownDelay:  30 * time.Second,
		HealthCheckTime: 10 * time.Second,
		MaxIdleTime:     60 * time.Second,
		TaskBatchSize:   5,
	}

	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)

	if pool == nil {
		t.Fatal("Expected worker pool to be created")
	}

	if len(pool.workers) != config.MinWorkers {
		t.Errorf("Expected %d initial workers, got %d", config.MinWorkers, len(pool.workers))
	}

	if pool.config.TaskBatchSize != 5 {
		t.Errorf("Expected TaskBatchSize to be 5, got %d", pool.config.TaskBatchSize)
	}
}

func TestWorkerPoolStartStop(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 2
	config.MaxWorkers = 4

	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)

	// Test start
	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}

	if !pool.IsHealthy() {
		t.Error("Expected worker pool to be healthy after start")
	}

	// Wait a bit for workers to initialize
	time.Sleep(100 * time.Millisecond)

	// Test stop
	err = pool.Stop(5 * time.Second)
	if err != nil {
		t.Fatalf("Failed to stop worker pool: %v", err)
	}

	if pool.IsHealthy() {
		t.Error("Expected worker pool to be unhealthy after stop")
	}
}

func TestWorkerPoolDynamicScaling(t *testing.T) {
	config := &WorkerPoolConfig{
		MinWorkers:      1,
		MaxWorkers:      5,
		TargetLoad:      0.7,
		ScaleUpDelay:    100 * time.Millisecond,
		ScaleDownDelay:  200 * time.Millisecond,
		HealthCheckTime: 50 * time.Millisecond,
		MaxIdleTime:     60 * time.Second,
		TaskBatchSize:   1,
	}

	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)

	// Start with minimum workers
	initialWorkers := len(pool.workers)
	if initialWorkers != config.MinWorkers {
		t.Errorf("Expected %d initial workers, got %d", config.MinWorkers, initialWorkers)
	}

	// Test scale up
	err := pool.AddWorker()
	if err != nil {
		t.Fatalf("Failed to add worker: %v", err)
	}

	if len(pool.workers) != initialWorkers+1 {
		t.Errorf("Expected %d workers after scaling up, got %d", initialWorkers+1, len(pool.workers))
	}

	// Test scale down
	var workerID int
	for id := range pool.workers {
		workerID = id
		break
	}

	err = pool.RemoveWorker(workerID)
	if err != nil {
		t.Fatalf("Failed to remove worker: %v", err)
	}

	if len(pool.workers) != initialWorkers {
		t.Errorf("Expected %d workers after scaling down, got %d", initialWorkers, len(pool.workers))
	}
}

func TestWorkerAffinity(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)

	// Get a worker ID
	var workerID int
	for id := range pool.workers {
		workerID = id
		break
	}

	// Test setting affinity
	affinity := WorkerAffinity{
		Platforms:    []string{"feishu", "email"},
		MessageTypes: []string{"alert", "notification"},
		Priorities:   []int{1, 2},
		Specialized:  true,
	}

	err := pool.SetWorkerAffinity(workerID, affinity)
	if err != nil {
		t.Fatalf("Failed to set worker affinity: %v", err)
	}

	// Test getting affinity
	retrievedAffinity, err := pool.GetWorkerAffinity(workerID)
	if err != nil {
		t.Fatalf("Failed to get worker affinity: %v", err)
	}

	if !retrievedAffinity.Specialized {
		t.Error("Expected worker to be specialized")
	}

	if len(retrievedAffinity.Platforms) != 2 {
		t.Errorf("Expected 2 platforms, got %d", len(retrievedAffinity.Platforms))
	}
}

func TestLoadBalancer(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)

	// Test load balancing strategies
	strategies := []LoadBalanceStrategy{
		RoundRobin,
		LeastConnections,
		WeightedRoundRobin,
		AffinityBased,
	}

	for _, strategy := range strategies {
		pool.SetLoadBalancingStrategy(strategy)
		retrievedStrategy := pool.GetLoadBalancingStrategy()

		if retrievedStrategy != strategy {
			t.Errorf("Expected strategy %v, got %v", strategy, retrievedStrategy)
		}

		// Test worker selection
		msg := &message.Message{
			ID:      "test-msg",
			Title:   "Test Message",
			Body:    "Test Body",
			Targets: []target.Target{{Platform: "feishu", Type: "webhook", Value: "test"}},
		}

		item := &QueueItem{Message: msg}
		worker := pool.GetBestWorkerForTask(item)

		// Worker should be selected (can be nil if no healthy workers)
		if len(pool.workers) > 0 && worker == nil {
			t.Errorf("Expected worker to be selected for strategy %v", strategy)
		}
	}
}

func TestWorkerStats(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 1

	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)

	// Get worker stats
	stats := pool.GetStats()
	if len(stats) != 1 {
		t.Errorf("Expected 1 worker stat, got %d", len(stats))
	}

	stat := stats[0]
	if stat.State != "idle" {
		t.Errorf("Expected worker state to be idle, got %s", stat.State)
	}

	if stat.Processed != 0 {
		t.Errorf("Expected 0 processed messages, got %d", stat.Processed)
	}

	// Test detailed stats
	detailedStats := pool.GetDetailedStats()
	if detailedStats == nil {
		t.Error("Expected detailed stats to be returned")
	}

	poolInfo, ok := detailedStats["pool_info"].(map[string]interface{})
	if !ok {
		t.Error("Expected pool_info to be present in detailed stats")
	}

	activeWorkers, ok := poolInfo["active_workers"].(int)
	if !ok || activeWorkers != 1 {
		t.Errorf("Expected 1 active worker, got %v", activeWorkers)
	}
}

func TestWorkerMonitor(t *testing.T) {
	config := &WorkerPoolConfig{
		MinWorkers:      1,
		MaxWorkers:      2,
		TargetLoad:      0.7,
		ScaleUpDelay:    100 * time.Millisecond,
		ScaleDownDelay:  200 * time.Millisecond,
		HealthCheckTime: 50 * time.Millisecond,
		MaxIdleTime:     200 * time.Millisecond,
		TaskBatchSize:   1,
	}

	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)
	monitor := pool.monitor

	// Test monitor start/stop
	err := monitor.Start()
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	// Wait for monitor to run
	time.Sleep(100 * time.Millisecond)

	err = monitor.Stop(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to stop monitor: %v", err)
	}
}

func TestWorkerScaler(t *testing.T) {
	config := &WorkerPoolConfig{
		MinWorkers:      1,
		MaxWorkers:      3,
		TargetLoad:      0.5,
		ScaleUpDelay:    50 * time.Millisecond,
		ScaleDownDelay:  100 * time.Millisecond,
		HealthCheckTime: 25 * time.Millisecond,
		MaxIdleTime:     60 * time.Second,
		TaskBatchSize:   1,
	}

	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)
	scaler := pool.scaler

	// Test scaler start/stop
	err := scaler.Start()
	if err != nil {
		t.Fatalf("Failed to start scaler: %v", err)
	}

	// Wait for scaler to run
	time.Sleep(100 * time.Millisecond)

	err = scaler.Stop(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to stop scaler: %v", err)
	}
}

func TestAsyncExecutorEnhanced(t *testing.T) {
	config := &WorkerPoolConfig{
		MinWorkers:      2,
		MaxWorkers:      4,
		TargetLoad:      0.7,
		ScaleUpDelay:    100 * time.Millisecond,
		ScaleDownDelay:  200 * time.Millisecond,
		HealthCheckTime: 50 * time.Millisecond,
		MaxIdleTime:     60 * time.Second,
		TaskBatchSize:   1,
	}

	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	executor := NewAsyncExecutor(100, config, dispatcher, testLogger)

	// Test start
	err := executor.Start()
	if err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}

	if !executor.IsHealthy() {
		t.Error("Expected executor to be healthy after start")
	}

	// Test configuration retrieval
	retrievedConfig := executor.GetWorkerPoolConfig()
	if retrievedConfig.MinWorkers != config.MinWorkers {
		t.Errorf("Expected MinWorkers %d, got %d", config.MinWorkers, retrievedConfig.MinWorkers)
	}

	// Test manual scaling
	err = executor.ScaleWorkers(3)
	if err != nil {
		t.Fatalf("Failed to scale workers: %v", err)
	}

	// Wait for scaling to complete
	time.Sleep(100 * time.Millisecond)

	stats := executor.GetStats()
	if stats == nil {
		t.Error("Expected stats to be returned")
	}

	// Test stop
	err = executor.Stop(5 * time.Second)
	if err != nil {
		t.Fatalf("Failed to stop executor: %v", err)
	}

	if executor.IsHealthy() {
		t.Error("Expected executor to be unhealthy after stop")
	}
}

func TestWorkerPerformanceMetrics(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 1

	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)

	// Get a worker
	var worker *Worker
	for _, w := range pool.workers {
		worker = w
		break
	}

	if worker == nil {
		t.Fatal("Expected at least one worker")
	}

	// Simulate processing
	duration := 100 * time.Millisecond
	worker.updatePerformanceMetrics(duration, true)

	// Check performance metrics
	worker.performance.mutex.RLock()
	avgProcessingTime := worker.performance.AvgProcessingTime
	worker.performance.mutex.RUnlock()

	if avgProcessingTime != duration {
		t.Errorf("Expected average processing time %v, got %v", duration, avgProcessingTime)
	}

	// Get worker stats
	stats := worker.getStats()
	if stats.Performance.AvgProcessingTime != duration {
		t.Errorf("Expected performance avg processing time %v, got %v", duration, stats.Performance.AvgProcessingTime)
	}
}

func TestWorkerBatchProcessing(t *testing.T) {
	config := &WorkerPoolConfig{
		MinWorkers:      1,
		MaxWorkers:      2,
		TargetLoad:      0.7,
		ScaleUpDelay:    5 * time.Second,
		ScaleDownDelay:  30 * time.Second,
		HealthCheckTime: 10 * time.Second,
		MaxIdleTime:     60 * time.Second,
		TaskBatchSize:   3, // Enable batching
	}

	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)

	// Get a worker
	var worker *Worker
	for _, w := range pool.workers {
		worker = w
		break
	}

	if worker == nil {
		t.Fatal("Expected at least one worker")
	}

	// Test batch processing
	batch := make([]*QueueItem, 2)
	for i := range batch {
		msg := &message.Message{
			ID:      fmt.Sprintf("test-msg-%d", i),
			Title:   "Test Message",
			Body:    "Test Body",
			Targets: []target.Target{{Platform: "feishu", Type: "webhook", Value: "test"}},
		}
		batch[i] = &QueueItem{Message: msg}
	}

	// This would normally be called internally
	worker.processBatch(batch)

	// Verify processing completed without errors
	if atomic.LoadInt64(&worker.processed) == 0 {
		t.Error("Expected worker to have processed messages")
	}
}

func TestWorkerStateTransitions(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 1

	queue := NewMemoryAsyncQueue(100, NewTestLogger())
	callbacks := NewCallbackRegistry(NewTestLogger())
	dispatcher := NewMockDispatcher()
	testLogger := NewTestLogger()

	pool := NewWorkerPool(config, queue, dispatcher, callbacks, testLogger)

	// Get a worker
	var worker *Worker
	for _, w := range pool.workers {
		worker = w
		break
	}

	if worker == nil {
		t.Fatal("Expected at least one worker")
	}

	// Test state transitions
	initialState := worker.state
	if initialState != WorkerStateIdle {
		t.Errorf("Expected initial state to be idle, got %s", initialState.String())
	}

	// Update state
	worker.updateState(WorkerStateProcessing)
	worker.stateMutex.RLock()
	currentState := worker.state
	worker.stateMutex.RUnlock()

	if currentState != WorkerStateProcessing {
		t.Errorf("Expected state to be processing, got %s", currentState.String())
	}

	// Test health check
	if !worker.isHealthy() {
		t.Error("Expected worker to be healthy")
	}
}