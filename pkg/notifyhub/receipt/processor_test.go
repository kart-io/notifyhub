package receipt

import (
	"bytes"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// MockLogger provides a simple mock logger for testing
type MockLogger struct{}

func (m *MockLogger) LogMode(level logger.LogLevel) logger.Logger { return m }
func (m *MockLogger) Debug(msg string, fields ...interface{})     {}
func (m *MockLogger) Info(msg string, fields ...interface{})      {}
func (m *MockLogger) Warn(msg string, fields ...interface{})      {}
func (m *MockLogger) Error(msg string, fields ...interface{})     {}

// TestProcessorCreation tests processor creation with options
func TestProcessorCreation(t *testing.T) {
	logger := &MockLogger{}

	// Test basic creation
	processor := NewProcessor(logger)
	if processor == nil {
		t.Fatal("Expected processor to be created")
	}

	// Test with options
	store := NewMemoryStore()
	config := AggregationConfig{
		PartialFailureThreshold: 0.7,
		FailureTolerance:        2,
		RequiredPlatforms:       []string{"feishu", "email"},
	}

	processor = NewProcessor(logger,
		WithPersistenceStore(store),
		WithAggregationConfig(config),
		WithRetentionPeriod(48*time.Hour),
	)

	if processor.store != store {
		t.Error("Expected persistence store to be set")
	}

	aggConfig := processor.GetAggregationConfig()
	if aggConfig.PartialFailureThreshold != 0.7 {
		t.Error("Expected aggregation config to be set")
	}

	if processor.GetRetentionPeriod() != 48*time.Hour {
		t.Error("Expected retention period to be set")
	}
}

// TestMultiPlatformAggregation tests enhanced multi-platform result aggregation
func TestMultiPlatformAggregation(t *testing.T) {
	processor := NewProcessor(&MockLogger{})

	receipt := &Receipt{
		MessageID: "test-msg-1",
		Results: []PlatformResult{
			{Platform: "feishu", Target: "user1", Success: true, Timestamp: time.Now()},
			{Platform: "feishu", Target: "user2", Success: false, Error: "rate limit", Timestamp: time.Now()},
			{Platform: "email", Target: "user@example.com", Success: true, Timestamp: time.Now()},
			{Platform: "webhook", Target: "http://example.com", Success: false, Error: "timeout", Timestamp: time.Now()},
		},
		Timestamp: time.Now(),
	}

	// Test aggregation
	enhanced := processor.aggregator.AggregateMultiPlatformResults(receipt)

	if enhanced.Total != 4 {
		t.Errorf("Expected total to be 4, got %d", enhanced.Total)
	}

	if enhanced.Successful != 2 {
		t.Errorf("Expected successful to be 2, got %d", enhanced.Successful)
	}

	if enhanced.Failed != 2 {
		t.Errorf("Expected failed to be 2, got %d", enhanced.Failed)
	}
}

// TestPartialFailureStatusCalculation tests partial failure status calculation
func TestPartialFailureStatusCalculation(t *testing.T) {
	tests := []struct {
		name      string
		config    AggregationConfig
		receipt   *Receipt
		expected  string
	}{
		{
			name: "Complete success",
			config: AggregationConfig{
				PartialFailureThreshold: 0.5,
			},
			receipt: &Receipt{
				Total:      3,
				Successful: 3,
				Failed:     0,
			},
			expected: string(StatusCompleted),
		},
		{
			name: "Complete failure",
			config: AggregationConfig{
				PartialFailureThreshold: 0.5,
			},
			receipt: &Receipt{
				Total:      3,
				Successful: 0,
				Failed:     3,
			},
			expected: string(StatusFailed),
		},
		{
			name: "Partial failure above threshold",
			config: AggregationConfig{
				PartialFailureThreshold: 0.5,
			},
			receipt: &Receipt{
				Total:      4,
				Successful: 3,
				Failed:     1,
			},
			expected: string(StatusPartialFailed),
		},
		{
			name: "Failure below threshold",
			config: AggregationConfig{
				PartialFailureThreshold: 0.8,
			},
			receipt: &Receipt{
				Total:      4,
				Successful: 3,
				Failed:     1,
			},
			expected: string(StatusFailed),
		},
		{
			name: "Within failure tolerance",
			config: AggregationConfig{
				PartialFailureThreshold: 0.5,
				FailureTolerance:        1,
			},
			receipt: &Receipt{
				Total:      4,
				Successful: 3,
				Failed:     1,
			},
			expected: string(StatusCompleted),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewProcessor(&MockLogger{}, WithAggregationConfig(tt.config))
			status := processor.aggregator.CalculateOverallStatus(tt.receipt)

			if status != tt.expected {
				t.Errorf("Expected status %s, got %s", tt.expected, status)
			}
		})
	}
}

// TestRequiredPlatforms tests required platform validation
func TestRequiredPlatforms(t *testing.T) {
	config := AggregationConfig{
		PartialFailureThreshold: 0.5,
		RequiredPlatforms:       []string{"feishu", "email"},
	}

	processor := NewProcessor(&MockLogger{}, WithAggregationConfig(config))

	tests := []struct {
		name     string
		receipt  *Receipt
		expected bool
	}{
		{
			name: "All required platforms successful",
			receipt: &Receipt{
				Results: []PlatformResult{
					{Platform: "feishu", Success: true},
					{Platform: "email", Success: true},
					{Platform: "webhook", Success: false},
				},
			},
			expected: true,
		},
		{
			name: "Required platform failed",
			receipt: &Receipt{
				Results: []PlatformResult{
					{Platform: "feishu", Success: false},
					{Platform: "email", Success: true},
					{Platform: "webhook", Success: true},
				},
			},
			expected: false,
		},
		{
			name: "Missing required platform",
			receipt: &Receipt{
				Results: []PlatformResult{
					{Platform: "webhook", Success: true},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.aggregator.CheckRequiredPlatforms(tt.receipt)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestPersistenceStore tests persistence store integration
func TestPersistenceStore(t *testing.T) {
	store := NewMemoryStore()
	processor := NewProcessor(&MockLogger{}, WithPersistenceStore(store))

	receipt := &Receipt{
		MessageID: "test-msg-1",
		Status:    "completed",
		Results: []PlatformResult{
			{Platform: "feishu", Target: "user1", Success: true, Timestamp: time.Now()},
		},
		Successful: 1,
		Failed:     0,
		Total:      1,
		Timestamp:  time.Now(),
	}

	// Test processing with persistence
	err := processor.ProcessReceipt(receipt)
	if err != nil {
		t.Fatalf("Failed to process receipt: %v", err)
	}

	// Test retrieval from memory
	retrieved, ok := processor.GetReceipt("test-msg-1")
	if !ok {
		t.Fatal("Receipt not found in memory")
	}

	if retrieved.MessageID != receipt.MessageID {
		t.Error("Retrieved receipt doesn't match original")
	}

	// Clear memory and test persistence fallback
	processor.receipts = make(map[string]*Receipt)

	// Should still be able to retrieve from store
	retrieved, ok = processor.GetReceipt("test-msg-1")
	if !ok {
		t.Fatal("Receipt not found in persistence store")
	}

	if retrieved.MessageID != receipt.MessageID {
		t.Error("Retrieved receipt from store doesn't match original")
	}
}

// TestSerializationDeserialization tests receipt serialization and deserialization
func TestSerializationDeserialization(t *testing.T) {
	processor := NewProcessor(&MockLogger{})

	receipt := &Receipt{
		MessageID: "test-msg-1",
		Status:    "completed",
		Results: []PlatformResult{
			{
				Platform:  "feishu",
				Target:    "user1",
				Success:   true,
				MessageID: "feishu-msg-1",
				Timestamp: time.Now(),
				Duration:  100 * time.Millisecond,
			},
		},
		Successful: 1,
		Failed:     0,
		Total:      1,
		Timestamp:  time.Now(),
	}

	// Test serialization
	data, err := processor.SerializeReceipt(receipt)
	if err != nil {
		t.Fatalf("Failed to serialize receipt: %v", err)
	}

	// Test deserialization
	deserialized, err := processor.DeserializeReceipt(data)
	if err != nil {
		t.Fatalf("Failed to deserialize receipt: %v", err)
	}

	if deserialized.MessageID != receipt.MessageID {
		t.Error("Deserialized receipt doesn't match original")
	}

	if len(deserialized.Results) != len(receipt.Results) {
		t.Error("Deserialized results count doesn't match original")
	}
}

// TestExportImport tests receipt export and import functionality
func TestExportImport(t *testing.T) {
	processor := NewProcessor(&MockLogger{})

	// Add some test receipts
	receipts := []*Receipt{
		{
			MessageID: "test-msg-1",
			Status:    "completed",
			Results: []PlatformResult{
				{Platform: "feishu", Target: "user1", Success: true, Timestamp: time.Now()},
			},
			Successful: 1,
			Failed:     0,
			Total:      1,
			Timestamp:  time.Now(),
		},
		{
			MessageID: "test-msg-2",
			Status:    "failed",
			Results: []PlatformResult{
				{Platform: "email", Target: "user@example.com", Success: false, Error: "timeout", Timestamp: time.Now()},
			},
			Successful: 0,
			Failed:     1,
			Total:      1,
			Timestamp:  time.Now(),
		},
	}

	for _, receipt := range receipts {
		processor.ProcessReceipt(receipt)
	}

	// Test export
	var buffer bytes.Buffer
	filter := ReceiptFilter{} // No filter, export all

	err := processor.ExportReceipts(&buffer, filter)
	if err != nil {
		t.Fatalf("Failed to export receipts: %v", err)
	}

	// Clear processor
	processor.receipts = make(map[string]*Receipt)

	// Test import
	err = processor.ImportReceipts(&buffer)
	if err != nil {
		t.Fatalf("Failed to import receipts: %v", err)
	}

	// Verify imported receipts
	if len(processor.receipts) != 2 {
		t.Errorf("Expected 2 imported receipts, got %d", len(processor.receipts))
	}

	for _, originalReceipt := range receipts {
		imported, ok := processor.GetReceipt(originalReceipt.MessageID)
		if !ok {
			t.Errorf("Imported receipt %s not found", originalReceipt.MessageID)
		} else if imported.Status != originalReceipt.Status {
			t.Errorf("Imported receipt status doesn't match original")
		}
	}
}

// TestFailurePatternAnalysis tests failure pattern analysis
func TestFailurePatternAnalysis(t *testing.T) {
	processor := NewProcessor(&MockLogger{})

	// Add receipts with various failure patterns
	receipts := []*Receipt{
		{
			MessageID: "msg-1",
			Status:    "completed",
			Results: []PlatformResult{
				{Platform: "feishu", Success: true},
			},
			Successful: 1,
			Failed:     0,
			Total:      1,
			Timestamp:  time.Now(),
		},
		{
			MessageID: "msg-2",
			Status:    "failed",
			Results: []PlatformResult{
				{Platform: "feishu", Success: false, Error: "rate limit exceeded"},
				{Platform: "email", Success: false, Error: "network timeout"},
			},
			Successful: 0,
			Failed:     2,
			Total:      2,
			Timestamp:  time.Now(),
		},
		{
			MessageID: "msg-3",
			Status:    "partial_failed",
			Results: []PlatformResult{
				{Platform: "feishu", Success: true},
				{Platform: "webhook", Success: false, Error: "authentication failed"},
			},
			Successful: 1,
			Failed:     1,
			Total:      2,
			Timestamp:  time.Now(),
		},
	}

	for _, receipt := range receipts {
		processor.ProcessReceipt(receipt)
	}

	// Analyze failure patterns
	analysis := processor.AnalyzeFailurePatterns(ReceiptFilter{})

	if analysis["total_receipts"].(int) != 3 {
		t.Error("Expected 3 total receipts in analysis")
	}

	if analysis["total_failures"].(int) != 2 {
		t.Error("Expected 2 total failures in analysis")
	}

	failuresByPlatform := analysis["failures_by_platform"].(map[string]int)
	if failuresByPlatform["feishu"] != 1 {
		t.Error("Expected 1 feishu failure")
	}

	failuresByError := analysis["failures_by_error"].(map[string]int)
	if failuresByError["rate_limit_errors"] != 1 {
		t.Error("Expected 1 rate limit error")
	}
	if failuresByError["timeout_errors"] != 1 {
		t.Error("Expected 1 timeout error")
	}
	if failuresByError["authentication_errors"] != 1 {
		t.Error("Expected 1 authentication error")
	}
}

// TestBatchProcessing tests batch receipt processing
func TestBatchProcessing(t *testing.T) {
	processor := NewProcessor(&MockLogger{})

	receipts := []*Receipt{
		{
			MessageID: "batch-msg-1",
			Results: []PlatformResult{
				{Platform: "feishu", Target: "user1", Success: true, Timestamp: time.Now()},
			},
			Timestamp: time.Now(),
		},
		{
			MessageID: "batch-msg-2",
			Results: []PlatformResult{
				{Platform: "email", Target: "user@example.com", Success: false, Error: "timeout", Timestamp: time.Now()},
			},
			Timestamp: time.Now(),
		},
	}

	err := processor.BatchProcessReceipts(receipts)
	if err != nil {
		t.Fatalf("Failed to batch process receipts: %v", err)
	}

	// Verify all receipts were processed
	for _, receipt := range receipts {
		processed, ok := processor.GetReceipt(receipt.MessageID)
		if !ok {
			t.Errorf("Batch processed receipt %s not found", receipt.MessageID)
		} else if processed.Status == "" {
			t.Error("Batch processed receipt should have status calculated")
		}
	}
}

// TestFilterMatching tests receipt filter matching
func TestFilterMatching(t *testing.T) {
	processor := NewProcessor(&MockLogger{})

	now := time.Now()
	receipt := &Receipt{
		MessageID: "filter-test-msg",
		Status:    "completed",
		Results: []PlatformResult{
			{Platform: "feishu", Target: "user1", Success: true},
		},
		Timestamp: now,
	}

	tests := []struct {
		name     string
		filter   ReceiptFilter
		expected bool
	}{
		{
			name:     "No filter",
			filter:   ReceiptFilter{},
			expected: true,
		},
		{
			name: "Time range match",
			filter: ReceiptFilter{
				StartTime: &[]time.Time{now.Add(-1 * time.Hour)}[0],
				EndTime:   &[]time.Time{now.Add(1 * time.Hour)}[0],
			},
			expected: true,
		},
		{
			name: "Time range no match",
			filter: ReceiptFilter{
				StartTime: &[]time.Time{now.Add(1 * time.Hour)}[0],
			},
			expected: false,
		},
		{
			name: "Status match",
			filter: ReceiptFilter{
				Status: []string{"completed", "failed"},
			},
			expected: true,
		},
		{
			name: "Status no match",
			filter: ReceiptFilter{
				Status: []string{"failed"},
			},
			expected: false,
		},
		{
			name: "Platform match",
			filter: ReceiptFilter{
				Platforms: []string{"feishu", "email"},
			},
			expected: true,
		},
		{
			name: "Platform no match",
			filter: ReceiptFilter{
				Platforms: []string{"webhook"},
			},
			expected: false,
		},
		{
			name: "Message ID match",
			filter: ReceiptFilter{
				MessageIDs: []string{"filter-test-msg", "other-msg"},
			},
			expected: true,
		},
		{
			name: "Message ID no match",
			filter: ReceiptFilter{
				MessageIDs: []string{"other-msg"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.serializer.MatchesFilter(receipt, tt.filter)
			if result != tt.expected {
				t.Errorf("Expected filter match %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestMetrics tests processor metrics tracking
func TestMetrics(t *testing.T) {
	processor := NewProcessor(&MockLogger{})

	receipt := &Receipt{
		MessageID: "metrics-test-msg",
		Status:    "completed",
		Results: []PlatformResult{
			{Platform: "feishu", Target: "user1", Success: true, Timestamp: time.Now()},
		},
		Successful: 1,
		Failed:     0,
		Total:      1,
		Timestamp:  time.Now(),
	}

	err := processor.ProcessReceipt(receipt)
	if err != nil {
		t.Fatalf("Failed to process receipt: %v", err)
	}

	metrics := processor.GetMetrics()
	if metrics.ProcessedCount != 1 {
		t.Errorf("Expected processed count to be 1, got %d", metrics.ProcessedCount)
	}

	if metrics.AverageProcessing == 0 {
		t.Error("Expected average processing time to be set")
	}

	stats := processor.GetStats()
	if stats["processed_count"].(int64) != 1 {
		t.Error("Expected processed count in stats to be 1")
	}
}