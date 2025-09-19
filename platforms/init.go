package platforms

import (
	"github.com/kart-io/notifyhub/platforms/email"
	"github.com/kart-io/notifyhub/platforms/feishu"
	"github.com/kart-io/notifyhub/platforms/registry"
	"github.com/kart-io/notifyhub/platforms/sms"
)

// RegisterAll registers all built-in platform adapters
// This implements the Phase 1 requirement for platform equality
func RegisterAll() error {
	// Register Feishu adapter
	if err := registry.Register(&feishu.FeishuAdapter{}); err != nil {
		return err
	}

	// Register Email adapter
	if err := registry.Register(&email.EmailAdapter{}); err != nil {
		return err
	}

	// Register SMS adapter
	if err := registry.Register(&sms.SMSAdapter{}); err != nil {
		return err
	}

	return nil
}

// init automatically registers all platforms on import
func init() {
	// Register all platforms during package initialization
	// This ensures platform equality from the start
	if err := RegisterAll(); err != nil {
		// Log error but don't panic to allow graceful handling
		// In production, this should be properly logged
		_ = err
	}
}
