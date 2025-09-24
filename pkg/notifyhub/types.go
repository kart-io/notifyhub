// Package notifyhub provides type aliases and constants for backward compatibility.
// This file contains all type definitions and constants that maintain compatibility
// with existing code while delegating implementation to modular packages.
package notifyhub

import (
	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/core"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Type aliases for backward compatibility
type (
	Hub            = core.Hub
	HubConfig      = config.HubConfig
	PlatformConfig = config.PlatformConfig
	RetryPolicy    = config.RetryPolicy
	Message        = message.Message
	MessageBuilder = message.MessageBuilder
	Priority       = message.Priority
	Target         = target.Target
	Receipt        = receipt.Receipt
	AsyncReceipt   = receipt.AsyncReceipt
	PlatformResult = receipt.PlatformResult
	HealthStatus   = core.HealthStatus
	PlatformHealth = core.PlatformHealth
	QueueHealth    = core.QueueHealth
)

// Message priority constants for backward compatibility
const (
	PriorityLow    = message.PriorityLow
	PriorityNormal = message.PriorityNormal
	PriorityHigh   = message.PriorityHigh
	PriorityUrgent = message.PriorityUrgent
)

// Target type constants for backward compatibility
const (
	TargetTypeEmail   = target.TargetTypeEmail
	TargetTypePhone   = target.TargetTypePhone
	TargetTypeUser    = target.TargetTypeUser
	TargetTypeGroup   = target.TargetTypeGroup
	TargetTypeChannel = target.TargetTypeChannel
	TargetTypeWebhook = target.TargetTypeWebhook
)

// Platform constants for backward compatibility
const (
	PlatformFeishu  = target.PlatformFeishu
	PlatformEmail   = target.PlatformEmail
	PlatformSMS     = target.PlatformSMS
	PlatformWebhook = target.PlatformWebhook
	PlatformAuto    = target.PlatformAuto
)
