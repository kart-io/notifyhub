package internal

import (
	"fmt"
	"sync"
	"time"
)

var (
	idCounter int64
	idMutex   sync.Mutex
)

// GenerateID generates a unique message ID with better concurrency safety
func GenerateID() string {
	idMutex.Lock()
	defer idMutex.Unlock()

	now := time.Now().UnixNano()
	idCounter++

	return fmt.Sprintf("%d-%d", now, idCounter)
}
