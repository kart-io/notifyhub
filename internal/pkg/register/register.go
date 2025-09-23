// Package register provides automatic registration of internal platforms
package register

import (
	"github.com/kart-io/notifyhub/internal/pkg/adapters"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// RegisterInternalPlatforms registers all internal platforms
// This should be called during application initialization
func RegisterInternalPlatforms() {
	// Register Feishu platform
	platform.RegisterPlatform("feishu", adapters.NewFeishuSender)

	// TODO: Add other internal platforms when their adapters are created
	// platform.RegisterPlatform("email", adapters.NewEmailSender)
	// platform.RegisterPlatform("sms", adapters.NewSMSSender)
}

// init automatically registers internal platforms
func init() {
	RegisterInternalPlatforms()
}
