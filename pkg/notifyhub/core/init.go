// Package core provides initialization for internal platforms
package core

import (
	"sync"

	// Import internal register to auto-register internal platforms
	_ "github.com/kart-io/notifyhub/internal/pkg/register"
)

var (
	initOnce sync.Once
)

// ensureInternalPlatformsRegistered ensures that internal platforms are registered
// This function is called during Hub creation to guarantee that internal platforms
// are available for use
func ensureInternalPlatformsRegistered() {
	initOnce.Do(func() {
		// The internal/pkg/register package automatically registers platforms
		// via its init() function when imported above
		// This ensures platforms are registered exactly once
	})
}