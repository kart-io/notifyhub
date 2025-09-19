package internal

import (
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

// GenerateID generates a unique message ID using UUID v4 to avoid concurrent conflicts
func GenerateID() string {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		// Fallback to a simple time-based approach if random fails
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	// Set version (4) and variant bits according to RFC 4122
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// GenerateIDWithPrefix generates a unique ID with a given prefix
func GenerateIDWithPrefix(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, GenerateID())
}
