// Package message provides message structure and manipulation for NotifyHub
package message

import (
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
	"github.com/kart-io/notifyhub/pkg/utils/idgen"
)

// Message represents a notification message with fluent API support
type Message struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	Format       string                 `json:"format"` // "text", "markdown", "html"
	Priority     Priority               `json:"priority"`
	Targets      []target.Target        `json:"targets"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	ScheduledAt  *time.Time             `json:"scheduled_at,omitempty"`
	PlatformData map[string]interface{} `json:"platform_data,omitempty"`
}

// Helper methods for the Message type

// GetMetadata returns a copy of the metadata
func (m *Message) GetMetadata() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m.Metadata {
		result[k] = v
	}
	return result
}

// GetVariables returns a copy of the variables
func (m *Message) GetVariables() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m.Variables {
		result[k] = v
	}
	return result
}

// GetPlatformData returns a copy of the platform data
func (m *Message) GetPlatformData() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m.PlatformData {
		result[k] = v
	}
	return result
}

// AddTarget adds a target to the message
func (m *Message) AddTarget(t target.Target) {
	m.Targets = append(m.Targets, t)
}

// AddTargets adds multiple targets to the message
func (m *Message) AddTargets(targets ...target.Target) {
	m.Targets = append(m.Targets, targets...)
}

// SetMetadata sets a metadata value
func (m *Message) SetMetadata(key string, value interface{}) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	m.Metadata[key] = value
}

// SetVariable sets a template variable
func (m *Message) SetVariable(key string, value interface{}) {
	if m.Variables == nil {
		m.Variables = make(map[string]interface{})
	}
	m.Variables[key] = value
}

// SetPlatformData sets platform-specific data
func (m *Message) SetPlatformData(key string, value interface{}) {
	if m.PlatformData == nil {
		m.PlatformData = make(map[string]interface{})
	}
	m.PlatformData[key] = value
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return idgen.GenerateMessageID()
}