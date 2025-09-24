package idgen

import (
	"strings"
	"testing"
)

func TestSimpleGenerator(t *testing.T) {
	gen := NewSimpleGenerator()

	// Test basic generation
	id1 := gen.Generate()
	id2 := gen.Generate()

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	if len(id1) == 0 {
		t.Error("Generated ID should not be empty")
	}

	// Test with prefix
	prefixedID := gen.GenerateWithPrefix("test")
	if !strings.HasPrefix(prefixedID, "test_") {
		t.Errorf("Expected prefixed ID to start with 'test_', got: %s", prefixedID)
	}
}

func TestSnowflakeGenerator(t *testing.T) {
	gen := NewSnowflakeGenerator(1)

	// Test basic generation
	id1 := gen.Generate()
	id2 := gen.Generate()

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	if len(id1) == 0 {
		t.Error("Generated ID should not be empty")
	}

	// Test with prefix
	prefixedID := gen.GenerateWithPrefix("snow")
	if !strings.HasPrefix(prefixedID, "snow_") {
		t.Errorf("Expected prefixed ID to start with 'snow_', got: %s", prefixedID)
	}
}

func TestMessageIDGenerator(t *testing.T) {
	gen := NewMessageIDGenerator()

	// Test message ID generation
	msgID := gen.GenerateMessageID()
	if !strings.HasPrefix(msgID, "msg_") {
		t.Errorf("Expected message ID to start with 'msg_', got: %s", msgID)
	}

	// Test task ID generation
	taskID := gen.GenerateTaskID()
	if !strings.HasPrefix(taskID, "task_") {
		t.Errorf("Expected task ID to start with 'task_', got: %s", taskID)
	}

	// Test receipt ID generation
	receiptID := gen.GenerateReceiptID()
	if !strings.HasPrefix(receiptID, "rcpt_") {
		t.Errorf("Expected receipt ID to start with 'rcpt_', got: %s", receiptID)
	}
}

func TestBatchIDGenerator(t *testing.T) {
	gen := NewBatchIDGenerator()

	// Test batch ID generation
	batchID := gen.GenerateBatchID()
	if !strings.HasPrefix(batchID, "batch_") {
		t.Errorf("Expected batch ID to start with 'batch_', got: %s", batchID)
	}

	// Test session ID generation
	sessionID := gen.GenerateSessionID()
	if !strings.HasPrefix(sessionID, "sess_") {
		t.Errorf("Expected session ID to start with 'sess_', got: %s", sessionID)
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Test global message ID generation
	msgID := GenerateMessageID()
	if !strings.HasPrefix(msgID, "msg_") {
		t.Errorf("Expected message ID to start with 'msg_', got: %s", msgID)
	}

	// Test global simple ID generation
	simpleID := GenerateSimpleID()
	if len(simpleID) == 0 {
		t.Error("Generated simple ID should not be empty")
	}

	// Test global Snowflake ID generation
	snowflakeID := GenerateSnowflakeID()
	if len(snowflakeID) == 0 {
		t.Error("Generated Snowflake ID should not be empty")
	}
}

func TestIDUniqueness(t *testing.T) {
	gen := NewSimpleGenerator()
	ids := make(map[string]bool)
	count := 1000

	// Generate many IDs quickly to test uniqueness
	for i := 0; i < count; i++ {
		id := gen.Generate()
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

func TestSnowflakeUniqueness(t *testing.T) {
	gen := NewSnowflakeGenerator(1)
	ids := make(map[string]bool)
	count := 1000

	// Generate many IDs quickly to test uniqueness
	for i := 0; i < count; i++ {
		id := gen.Generate()
		if ids[id] {
			t.Errorf("Duplicate Snowflake ID generated: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("Expected %d unique Snowflake IDs, got %d", count, len(ids))
	}
}

func TestConcurrentGeneration(t *testing.T) {
	gen := NewSimpleGenerator()
	idCh := make(chan string, 100)
	done := make(chan bool, 10)

	// Start multiple goroutines generating IDs
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				idCh <- gen.Generate()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	close(idCh)

	// Collect all IDs and check for uniqueness
	ids := make(map[string]bool)
	for id := range idCh {
		if ids[id] {
			t.Errorf("Duplicate ID generated in concurrent test: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != 100 {
		t.Errorf("Expected 100 unique IDs, got %d", len(ids))
	}
}

func BenchmarkSimpleGenerator(b *testing.B) {
	gen := NewSimpleGenerator()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkSnowflakeGenerator(b *testing.B) {
	gen := NewSnowflakeGenerator(1)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkMessageIDGenerator(b *testing.B) {
	gen := NewMessageIDGenerator()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gen.GenerateMessageID()
	}
}

func BenchmarkGlobalMessageID(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		GenerateMessageID()
	}
}
