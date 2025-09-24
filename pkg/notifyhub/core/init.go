// Package core provides initialization for internal platforms
package core

import (
	"sync"
)

var (
	initOnce sync.Once
)

// ensureInternalPlatformsRegistered ensures that internal platforms are registered
// This function is no longer needed as platforms auto-register via extension system
// but kept for backward compatibility
func ensureInternalPlatformsRegistered() {
	initOnce.Do(func() {
		// With the new platform extension system, platforms auto-register
		// when their packages are imported. No manual registration needed.
	})
}
