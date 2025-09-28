// Package message provides message structure and manipulation for NotifyHub
package message

import (
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
	"github.com/kart-io/notifyhub/pkg/utils/idgen"
)

// Format represents message format types
type Format string

const (
	FormatText     Format = "text"
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
)

// String returns the string representation of format
func (f Format) String() string {
	return string(f)
}

// IsValid returns true if the format is valid
func (f Format) IsValid() bool {
	switch f {
	case FormatText, FormatMarkdown, FormatHTML:
		return true
	default:
		return false
	}
}

// Message represents a unified notification message structure
// This is the canonical Message definition that replaces all other Message types in the codebase
type Message struct {
	ID           string                 `json:"id" validate:"required,min=1"`
	Title        string                 `json:"title" validate:"required,max=200"`
	Body         string                 `json:"body" validate:"required,max=4096"`
	Format       Format                 `json:"format" validate:"required,oneof=text markdown html"`
	Priority     Priority               `json:"priority" validate:"min=0,max=3"`
	Targets      []target.Target        `json:"targets" validate:"required,min=1,max=100,dive"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	PlatformData map[string]interface{} `json:"platform_data,omitempty"`
	CreatedAt    time.Time              `json:"created_at" validate:"required"`
	ScheduledAt  *time.Time             `json:"scheduled_at,omitempty"`
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

// New creates a new message with auto-generated ID and CreatedAt timestamp
func New() *Message {
	return &Message{
		ID:           generateMessageID(),
		Format:       FormatText,     // default format
		Priority:     PriorityNormal, // default priority
		CreatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
		Variables:    make(map[string]interface{}),
		PlatformData: make(map[string]interface{}),
	}
}

// NewWithID creates a new message with the specified ID
func NewWithID(id string) *Message {
	return &Message{
		ID:           id,
		Format:       FormatText,
		Priority:     PriorityNormal,
		CreatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
		Variables:    make(map[string]interface{}),
		PlatformData: make(map[string]interface{}),
	}
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return idgen.GenerateMessageID()
}
