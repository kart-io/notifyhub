// Package notifyhub provides the public API for the NotifyHub notification system.
//
// This package serves as the main entry point for creating and using NotifyHub instances.
// It provides backward compatibility adapters and convenient factory functions for
// creating notification hubs with various platform configurations.
//
// This file has been refactored into multiple files for better organization:
//   - types.go: Type aliases and constants for backward compatibility
//   - hub_factory.go: Hub creation and configuration functions
//   - builders.go: Message and target builder adapter functions
//
// Usage Examples:
//
// Creating a hub with platform packages:
//
//	import "github.com/kart-io/notifyhub/pkg/platforms/feishu"
//	import "github.com/kart-io/notifyhub/pkg/platforms/email"
//
//	hub, err := NewHub(
//	    feishu.WithFeishu("webhook-url", feishu.WithFeishuSecret("secret")),
//	    email.WithEmail("smtp.example.com", 587, "from@example.com"),
//	)
//
// Creating and sending messages:
//
//	msg := NewAlert("System Alert").
//	    WithBody("Database connection failed").
//	    ToTarget(NewEmailTarget("admin@company.com")).
//	    Build()
//
//	receipt, err := hub.Send(ctx, msg)
//
// For more examples, see the examples/ directory in the repository.
package notifyhub

// All functionality has been moved to:
// - types.go: Type definitions and constants
// - hub_factory.go: Hub creation functions
// - builders.go: Builder adapter functions
// - extensions.go: Extension system (already exists)
