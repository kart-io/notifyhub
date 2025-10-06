// Package message provides unified message structures and types for NotifyHub
package message

import (
	"time"

	"github.com/kart-io/notifyhub/pkg/errors"
	"github.com/kart-io/notifyhub/pkg/target"
)

// Message represents a unified message structure
type Message struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	Format       Format                 `json:"format"`
	Priority     Priority               `json:"priority"`
	Targets      []target.Target        `json:"targets"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	PlatformData map[string]interface{} `json:"platform_data,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	ScheduledAt  *time.Time             `json:"scheduled_at,omitempty"`
}

// Format represents message format types
type Format string

const (
	FormatText     Format = "text"
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
)

// Priority represents message priority levels
type Priority int

const (
	PriorityLow    Priority = 0
	PriorityNormal Priority = 1
	PriorityHigh   Priority = 2
	PriorityUrgent Priority = 3
)

// New creates a new message with default values
func New() *Message {
	return &Message{
		ID:        generateID(),
		Format:    FormatText,
		Priority:  PriorityNormal,
		Targets:   make([]target.Target, 0),
		Metadata:  make(map[string]interface{}),
		Variables: make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
}

// SetTitle sets the message title
func (m *Message) SetTitle(title string) *Message {
	m.Title = title
	return m
}

// SetBody sets the message body
func (m *Message) SetBody(body string) *Message {
	m.Body = body
	return m
}

// SetFormat sets the message format
func (m *Message) SetFormat(format Format) *Message {
	m.Format = format
	return m
}

// SetPriority sets the message priority
func (m *Message) SetPriority(priority Priority) *Message {
	m.Priority = priority
	return m
}

// AddTarget adds a target to the message
func (m *Message) AddTarget(t target.Target) *Message {
	m.Targets = append(m.Targets, t)
	return m
}

// SetTargets sets all targets for the message
func (m *Message) SetTargets(targets []target.Target) *Message {
	m.Targets = targets
	return m
}

// SetMetadata sets metadata for the message
func (m *Message) SetMetadata(key string, value interface{}) *Message {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	m.Metadata[key] = value
	return m
}

// SetVariable sets a template variable
func (m *Message) SetVariable(key string, value interface{}) *Message {
	if m.Variables == nil {
		m.Variables = make(map[string]interface{})
	}
	m.Variables[key] = value
	return m
}

// SetPlatformData sets platform-specific data
func (m *Message) SetPlatformData(platform string, data interface{}) *Message {
	if m.PlatformData == nil {
		m.PlatformData = make(map[string]interface{})
	}
	m.PlatformData[platform] = data
	return m
}

// ScheduleAt schedules the message for later delivery
func (m *Message) ScheduleAt(at time.Time) *Message {
	m.ScheduledAt = &at
	return m
}

// IsScheduled returns true if the message is scheduled for later delivery
func (m *Message) IsScheduled() bool {
	return m.ScheduledAt != nil && m.ScheduledAt.After(time.Now())
}

// Validate validates the message
func (m *Message) Validate() error {
	if m.Title == "" && m.Body == "" {
		return errors.New(errors.ErrEmptyMessage, "message title and body cannot both be empty")
	}
	if len(m.Targets) == 0 {
		return errors.New(errors.ErrNoTargets, "message must have at least one target")
	}
	return nil
}

// generateID generates a unique message ID
func generateID() string {
	// Simple ID generation - in production, use proper UUID or timestamp-based ID
	return time.Now().Format("20060102150405") + "-" + "msg"
}
