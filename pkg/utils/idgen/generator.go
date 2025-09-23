// Package idgen provides ID generation utilities for NotifyHub
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

// Generator defines the interface for ID generation
type Generator interface {
	// Generate creates a new unique ID
	Generate() string
	// GenerateWithPrefix creates a new unique ID with the given prefix
	GenerateWithPrefix(prefix string) string
}

// SimpleGenerator implements a simple ID generator using timestamp and random bytes
type SimpleGenerator struct {
	counter uint64
}

// NewSimpleGenerator creates a new simple ID generator
func NewSimpleGenerator() *SimpleGenerator {
	return &SimpleGenerator{}
}

// Generate creates a new unique ID in format: timestamp_counter_random
func (g *SimpleGenerator) Generate() string {
	return g.GenerateWithPrefix("")
}

// GenerateWithPrefix creates a new unique ID with the given prefix
func (g *SimpleGenerator) GenerateWithPrefix(prefix string) string {
	timestamp := time.Now().UnixNano()
	counter := atomic.AddUint64(&g.counter, 1)

	// Generate 4 bytes of random data
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to counter-based random if crypto/rand fails
		randomBytes = []byte{
			byte(counter >> 24),
			byte(counter >> 16),
			byte(counter >> 8),
			byte(counter),
		}
	}

	randomHex := hex.EncodeToString(randomBytes)

	if prefix != "" {
		return fmt.Sprintf("%s_%d_%d_%s", prefix, timestamp, counter, randomHex)
	}
	return fmt.Sprintf("%d_%d_%s", timestamp, counter, randomHex)
}

// SnowflakeGenerator implements a Snowflake-like ID generator
type SnowflakeGenerator struct {
	machineID uint64
	sequence  uint64
	epoch     int64 // Custom epoch timestamp
}

// NewSnowflakeGenerator creates a new Snowflake ID generator
func NewSnowflakeGenerator(machineID uint64) *SnowflakeGenerator {
	// Use NotifyHub epoch (2024-01-01 00:00:00 UTC)
	epoch := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	return &SnowflakeGenerator{
		machineID: machineID & 0x3FF, // 10 bits
		sequence:  0,
		epoch:     epoch,
	}
}

// Generate creates a new Snowflake ID
// Format: 1 bit (unused) + 41 bits timestamp + 10 bits machine + 12 bits sequence
func (g *SnowflakeGenerator) Generate() string {
	return g.GenerateWithPrefix("")
}

// GenerateWithPrefix creates a new Snowflake ID with prefix
func (g *SnowflakeGenerator) GenerateWithPrefix(prefix string) string {
	now := time.Now().UnixMilli()
	timestamp := now - g.epoch

	// Increment sequence
	seq := atomic.AddUint64(&g.sequence, 1) & 0xFFF // 12 bits

	// Construct the ID: timestamp(41) + machineID(10) + sequence(12)
	id := (uint64(timestamp) << 22) | (g.machineID << 12) | seq

	if prefix != "" {
		return fmt.Sprintf("%s_%d", prefix, id)
	}
	return fmt.Sprintf("%d", id)
}

// MessageIDGenerator is specialized for generating message IDs
type MessageIDGenerator struct {
	generator Generator
}

// NewMessageIDGenerator creates a new message ID generator
func NewMessageIDGenerator() *MessageIDGenerator {
	return &MessageIDGenerator{
		generator: NewSimpleGenerator(),
	}
}

// NewMessageIDGeneratorWithCustom creates a message ID generator with custom generator
func NewMessageIDGeneratorWithCustom(gen Generator) *MessageIDGenerator {
	return &MessageIDGenerator{
		generator: gen,
	}
}

// GenerateMessageID generates a message ID with "msg" prefix
func (g *MessageIDGenerator) GenerateMessageID() string {
	return g.generator.GenerateWithPrefix("msg")
}

// GenerateTaskID generates a task ID with "task" prefix
func (g *MessageIDGenerator) GenerateTaskID() string {
	return g.generator.GenerateWithPrefix("task")
}

// GenerateReceiptID generates a receipt ID with "rcpt" prefix
func (g *MessageIDGenerator) GenerateReceiptID() string {
	return g.generator.GenerateWithPrefix("rcpt")
}

// BatchIDGenerator generates batch IDs for related operations
type BatchIDGenerator struct {
	generator Generator
}

// NewBatchIDGenerator creates a new batch ID generator
func NewBatchIDGenerator() *BatchIDGenerator {
	return &BatchIDGenerator{
		generator: NewSimpleGenerator(),
	}
}

// GenerateBatchID generates a batch ID with "batch" prefix
func (g *BatchIDGenerator) GenerateBatchID() string {
	return g.generator.GenerateWithPrefix("batch")
}

// GenerateSessionID generates a session ID with "sess" prefix
func (g *BatchIDGenerator) GenerateSessionID() string {
	return g.generator.GenerateWithPrefix("sess")
}

// Default generators for global use
var (
	defaultMessageIDGen = NewMessageIDGenerator()
	defaultBatchIDGen   = NewBatchIDGenerator()
)

// Global convenience functions

// GenerateMessageID generates a global message ID
func GenerateMessageID() string {
	return defaultMessageIDGen.GenerateMessageID()
}

// GenerateTaskID generates a global task ID
func GenerateTaskID() string {
	return defaultMessageIDGen.GenerateTaskID()
}

// GenerateReceiptID generates a global receipt ID
func GenerateReceiptID() string {
	return defaultMessageIDGen.GenerateReceiptID()
}

// GenerateBatchID generates a global batch ID
func GenerateBatchID() string {
	return defaultBatchIDGen.GenerateBatchID()
}

// GenerateSessionID generates a global session ID
func GenerateSessionID() string {
	return defaultBatchIDGen.GenerateSessionID()
}

// GenerateSimpleID generates a simple ID without prefix
func GenerateSimpleID() string {
	gen := NewSimpleGenerator()
	return gen.Generate()
}

// GenerateSnowflakeID generates a Snowflake ID with machine ID 1
func GenerateSnowflakeID() string {
	gen := NewSnowflakeGenerator(1)
	return gen.Generate()
}
