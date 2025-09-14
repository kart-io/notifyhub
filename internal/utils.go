package internal

import (
	"fmt"
	"time"
)

// GenerateID generates a unique message ID
func GenerateID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%1000000)
}