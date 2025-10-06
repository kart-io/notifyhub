package idgen

import (
	"strings"
	"testing"
)

func TestNewSimpleGenerator(t *testing.T) {
	gen := NewSimpleGenerator()
	if gen == nil {
		t.Fatal("NewSimpleGenerator() returned nil")
	}
	if gen.counter != 0 {
		t.Errorf("counter = %d, want 0", gen.counter)
	}
}

func TestSimpleGenerator_Generate(t *testing.T) {
	gen := NewSimpleGenerator()
	id1 := gen.Generate()
	id2 := gen.Generate()

	if id1 == "" {
		t.Error("Generate() returned empty string")
	}
	if id2 == "" {
		t.Error("Generate() returned empty string")
	}
	if id1 == id2 {
		t.Error("Generate() returned duplicate IDs")
	}

	// Check format: timestamp_counter_random (no prefix)
	parts := strings.Split(id1, "_")
	if len(parts) != 3 {
		t.Errorf("ID format = %d parts, want 3", len(parts))
	}
}

func TestSimpleGenerator_GenerateWithPrefix(t *testing.T) {
	gen := NewSimpleGenerator()

	tests := []struct {
		prefix    string
		wantParts int
	}{
		{"msg", 4},    // msg_timestamp_counter_random
		{"task", 4},   // task_timestamp_counter_random
		{"", 3},       // timestamp_counter_random
		{"prefix", 4}, // prefix_timestamp_counter_random
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			id := gen.GenerateWithPrefix(tt.prefix)
			if id == "" {
				t.Error("GenerateWithPrefix() returned empty string")
			}

			parts := strings.Split(id, "_")
			if len(parts) != tt.wantParts {
				t.Errorf("ID parts = %d, want %d (id=%s)", len(parts), tt.wantParts, id)
			}

			if tt.prefix != "" {
				if parts[0] != tt.prefix {
					t.Errorf("prefix = %s, want %s", parts[0], tt.prefix)
				}
			}
		})
	}
}

func TestSimpleGenerator_Uniqueness(t *testing.T) {
	gen := NewSimpleGenerator()
	ids := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		id := gen.Generate()
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("Generated %d unique IDs, want %d", len(ids), count)
	}
}

func TestNewSnowflakeGenerator(t *testing.T) {
	machineID := uint64(123)
	gen := NewSnowflakeGenerator(machineID)

	if gen == nil {
		t.Fatal("NewSnowflakeGenerator() returned nil")
	}
	// Machine ID should be masked to 10 bits
	if gen.machineID != (machineID & 0x3FF) {
		t.Errorf("machineID = %d, want %d", gen.machineID, machineID&0x3FF)
	}
	if gen.sequence != 0 {
		t.Errorf("sequence = %d, want 0", gen.sequence)
	}
}

func TestSnowflakeGenerator_Generate(t *testing.T) {
	gen := NewSnowflakeGenerator(1)
	id1 := gen.Generate()
	id2 := gen.Generate()

	if id1 == "" {
		t.Error("Generate() returned empty string")
	}
	if id2 == "" {
		t.Error("Generate() returned empty string")
	}
	if id1 == id2 {
		t.Error("Generate() returned duplicate IDs")
	}
}

func TestSnowflakeGenerator_GenerateWithPrefix(t *testing.T) {
	gen := NewSnowflakeGenerator(1)

	tests := []struct {
		prefix string
	}{
		{"msg"},
		{"task"},
		{""},
		{"prefix"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			id := gen.GenerateWithPrefix(tt.prefix)
			if id == "" {
				t.Error("GenerateWithPrefix() returned empty string")
			}

			if tt.prefix != "" {
				if !strings.HasPrefix(id, tt.prefix+"_") {
					t.Errorf("ID doesn't start with prefix: %s", id)
				}
			}
		})
	}
}

func TestSnowflakeGenerator_Uniqueness(t *testing.T) {
	gen := NewSnowflakeGenerator(1)
	ids := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		id := gen.Generate()
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("Generated %d unique IDs, want %d", len(ids), count)
	}
}

func TestNewMessageIDGenerator(t *testing.T) {
	gen := NewMessageIDGenerator()
	if gen == nil {
		t.Fatal("NewMessageIDGenerator() returned nil")
	}
	if gen.generator == nil {
		t.Error("generator is nil")
	}
}

func TestNewMessageIDGeneratorWithCustom(t *testing.T) {
	customGen := NewSnowflakeGenerator(1)
	gen := NewMessageIDGeneratorWithCustom(customGen)

	if gen == nil {
		t.Fatal("NewMessageIDGeneratorWithCustom() returned nil")
	}
	if gen.generator != customGen {
		t.Error("generator is not the custom one")
	}
}

func TestMessageIDGenerator_GenerateMessageID(t *testing.T) {
	gen := NewMessageIDGenerator()
	id := gen.GenerateMessageID()

	if id == "" {
		t.Error("GenerateMessageID() returned empty string")
	}
	if !strings.HasPrefix(id, "msg_") {
		t.Errorf("ID doesn't start with 'msg_': %s", id)
	}
}

func TestMessageIDGenerator_GenerateTaskID(t *testing.T) {
	gen := NewMessageIDGenerator()
	id := gen.GenerateTaskID()

	if id == "" {
		t.Error("GenerateTaskID() returned empty string")
	}
	if !strings.HasPrefix(id, "task_") {
		t.Errorf("ID doesn't start with 'task_': %s", id)
	}
}

func TestMessageIDGenerator_GenerateReceiptID(t *testing.T) {
	gen := NewMessageIDGenerator()
	id := gen.GenerateReceiptID()

	if id == "" {
		t.Error("GenerateReceiptID() returned empty string")
	}
	if !strings.HasPrefix(id, "rcpt_") {
		t.Errorf("ID doesn't start with 'rcpt_': %s", id)
	}
}

func TestNewBatchIDGenerator(t *testing.T) {
	gen := NewBatchIDGenerator()
	if gen == nil {
		t.Fatal("NewBatchIDGenerator() returned nil")
	}
	if gen.generator == nil {
		t.Error("generator is nil")
	}
}

func TestBatchIDGenerator_GenerateBatchID(t *testing.T) {
	gen := NewBatchIDGenerator()
	id := gen.GenerateBatchID()

	if id == "" {
		t.Error("GenerateBatchID() returned empty string")
	}
	if !strings.HasPrefix(id, "batch_") {
		t.Errorf("ID doesn't start with 'batch_': %s", id)
	}
}

func TestBatchIDGenerator_GenerateSessionID(t *testing.T) {
	gen := NewBatchIDGenerator()
	id := gen.GenerateSessionID()

	if id == "" {
		t.Error("GenerateSessionID() returned empty string")
	}
	if !strings.HasPrefix(id, "sess_") {
		t.Errorf("ID doesn't start with 'sess_': %s", id)
	}
}

func TestGlobalFunctions(t *testing.T) {
	tests := []struct {
		name   string
		fn     func() string
		prefix string
	}{
		{"GenerateMessageID", GenerateMessageID, "msg_"},
		{"GenerateTaskID", GenerateTaskID, "task_"},
		{"GenerateReceiptID", GenerateReceiptID, "rcpt_"},
		{"GenerateBatchID", GenerateBatchID, "batch_"},
		{"GenerateSessionID", GenerateSessionID, "sess_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.fn()
			if id == "" {
				t.Errorf("%s() returned empty string", tt.name)
			}
			if !strings.HasPrefix(id, tt.prefix) {
				t.Errorf("ID doesn't start with '%s': %s", tt.prefix, id)
			}
		})
	}
}

func TestGenerateSimpleID(t *testing.T) {
	id := GenerateSimpleID()
	if id == "" {
		t.Error("GenerateSimpleID() returned empty string")
	}

	// Simple ID has no prefix, format: timestamp_counter_random
	parts := strings.Split(id, "_")
	if len(parts) != 3 {
		t.Errorf("ID parts = %d, want 3", len(parts))
	}
}

func TestGenerateSnowflakeID(t *testing.T) {
	id := GenerateSnowflakeID()
	if id == "" {
		t.Error("GenerateSnowflakeID() returned empty string")
	}

	// Snowflake ID with no prefix should be a numeric string
	if strings.Contains(id, "_") {
		t.Errorf("Snowflake ID should not contain underscore: %s", id)
	}
}

func BenchmarkSimpleGenerator_Generate(b *testing.B) {
	gen := NewSimpleGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkSnowflakeGenerator_Generate(b *testing.B) {
	gen := NewSnowflakeGenerator(1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkGlobalGenerateMessageID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateMessageID()
	}
}
