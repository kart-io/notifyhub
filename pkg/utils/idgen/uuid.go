// Package idgen provides UUID utilities
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// UUID represents a UUID value
type UUID [16]byte

// String returns the string representation of UUID in standard format
func (u UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

// Bytes returns the UUID as a byte slice
func (u UUID) Bytes() []byte {
	return u[:]
}

// IsZero returns true if the UUID is zero
func (u UUID) IsZero() bool {
	return u == UUID{}
}

// Version returns the UUID version
func (u UUID) Version() int {
	return int(u[6] >> 4)
}

// Variant returns the UUID variant
func (u UUID) Variant() int {
	switch {
	case (u[8] & 0x80) == 0x00:
		return 0 // NCS
	case (u[8] & 0xC0) == 0x80:
		return 1 // RFC 4122
	case (u[8] & 0xE0) == 0xC0:
		return 2 // Microsoft
	default:
		return 3 // Future
	}
}

// UUIDGenerator provides UUID generation functionality
type UUIDGenerator struct{}

// NewUUIDGenerator creates a new UUID generator
func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

// GenerateV4 generates a new UUID v4 (random)
func (g *UUIDGenerator) GenerateV4() (UUID, error) {
	var u UUID
	_, err := rand.Read(u[:])
	if err != nil {
		return UUID{}, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Set version (4) and variant bits
	u[6] = (u[6] & 0x0F) | 0x40 // Version 4
	u[8] = (u[8] & 0x3F) | 0x80 // Variant 10

	return u, nil
}

// GenerateV1 generates a new UUID v1 (timestamp + MAC)
// Note: This is a simplified implementation without real MAC address
func (g *UUIDGenerator) GenerateV1() (UUID, error) {
	var u UUID

	// Get current timestamp (100-nanosecond intervals since UUID epoch)
	now := time.Now()
	// UUID epoch is Oct 15, 1582
	uuidEpoch := time.Date(1582, 10, 15, 0, 0, 0, 0, time.UTC)
	timestamp := uint64(now.Sub(uuidEpoch).Nanoseconds() / 100)

	// Time low (32 bits)
	u[0] = byte(timestamp)
	u[1] = byte(timestamp >> 8)
	u[2] = byte(timestamp >> 16)
	u[3] = byte(timestamp >> 24)

	// Time mid (16 bits)
	u[4] = byte(timestamp >> 32)
	u[5] = byte(timestamp >> 40)

	// Time high and version (16 bits)
	u[6] = byte(timestamp >> 48)
	u[7] = byte(timestamp >> 56)
	u[6] = (u[6] & 0x0F) | 0x10 // Version 1

	// Clock sequence (14 bits) + variant (2 bits)
	clockSeq := uint16(now.UnixNano() & 0x3FFF)
	u[8] = byte(clockSeq>>8) | 0x80 // Variant 10
	u[9] = byte(clockSeq)

	// Generate pseudo-random node ID (48 bits)
	// In real implementation, this should be MAC address
	_, err := rand.Read(u[10:16])
	if err != nil {
		return UUID{}, fmt.Errorf("failed to generate node ID: %w", err)
	}
	// Set multicast bit to indicate this is not a real MAC
	u[10] |= 0x01

	return u, nil
}

// ParseUUID parses a UUID string into UUID struct
func ParseUUID(s string) (UUID, error) {
	var u UUID

	// Remove hyphens and convert to lowercase
	s = strings.ReplaceAll(strings.ToLower(s), "-", "")

	if len(s) != 32 {
		return u, fmt.Errorf("invalid UUID length: expected 32 characters, got %d", len(s))
	}

	// Decode hex string
	decoded, err := hex.DecodeString(s)
	if err != nil {
		return u, fmt.Errorf("invalid UUID format: %w", err)
	}

	copy(u[:], decoded)
	return u, nil
}

// MustParseUUID parses a UUID string and panics on error
func MustParseUUID(s string) UUID {
	u, err := ParseUUID(s)
	if err != nil {
		panic(err)
	}
	return u
}

// NewUUID generates a new UUID v4 using the default generator
func NewUUID() (UUID, error) {
	gen := NewUUIDGenerator()
	return gen.GenerateV4()
}

// MustNewUUID generates a new UUID v4 and panics on error
func MustNewUUID() UUID {
	u, err := NewUUID()
	if err != nil {
		panic(err)
	}
	return u
}

// NewUUIDString generates a new UUID v4 and returns it as string
func NewUUIDString() (string, error) {
	u, err := NewUUID()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// MustNewUUIDString generates a new UUID v4 string and panics on error
func MustNewUUIDString() string {
	s, err := NewUUIDString()
	if err != nil {
		panic(err)
	}
	return s
}

// Predefined UUIDs for testing
var (
	NilUUID = UUID{}
	TestUUID = MustParseUUID("550e8400-e29b-41d4-a716-446655440000")
)

// IsValidUUID checks if a string is a valid UUID format
func IsValidUUID(s string) bool {
	_, err := ParseUUID(s)
	return err == nil
}

// UUIDFromBytes creates a UUID from byte slice
func UUIDFromBytes(b []byte) (UUID, error) {
	var u UUID
	if len(b) != 16 {
		return u, fmt.Errorf("invalid byte slice length: expected 16, got %d", len(b))
	}
	copy(u[:], b)
	return u, nil
}

// GenerateShortUUID generates a shorter UUID-like ID for human readability
func GenerateShortUUID() (string, error) {
	// Generate 8 bytes of random data
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Format as short UUID: xxxxxxxx-xxxx
	return fmt.Sprintf("%x-%x", b[0:4], b[4:8]), nil
}

// MustGenerateShortUUID generates a short UUID and panics on error
func MustGenerateShortUUID() string {
	s, err := GenerateShortUUID()
	if err != nil {
		panic(err)
	}
	return s
}