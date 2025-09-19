package common

// Remove imports to break circular dependency
// These interfaces will use generic interfaces instead

// PlatformAdapter defines the interface for platform adapters
// This is the core interface from the proposal
type PlatformAdapter interface {
	Name() string
	Supports() []string // Use string instead of message.Format to avoid circular import
	CreateBuilder(base *MessageBuilder) PlatformBuilder
	CreateTransport(config interface{}) (interface{}, error)
	Validate(msg interface{}) error // Use interface{} instead of specific type
}

// PlatformBuilder defines the interface for platform-specific builders
type PlatformBuilder interface {
	Platform() string
	Builder() *MessageBuilder // Returns base builder
}

// MessageBuilder represents the core message builder that platforms extend
// Using interface{} to avoid circular imports
type MessageBuilder struct {
	data map[string]interface{} // Generic data storage to avoid circular imports
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		data: make(map[string]interface{}),
	}
}

// Title sets the message title
func (mb *MessageBuilder) Title(title string) *MessageBuilder {
	mb.data["title"] = title
	return mb
}

// Body sets the message body
func (mb *MessageBuilder) Body(body string) *MessageBuilder {
	mb.data["body"] = body
	return mb
}

// AddTarget adds a target to the message
func (mb *MessageBuilder) AddTarget(target interface{}) *MessageBuilder {
	if targets, ok := mb.data["targets"].([]interface{}); ok {
		mb.data["targets"] = append(targets, target)
	} else {
		mb.data["targets"] = []interface{}{target}
	}
	return mb
}

// Build returns the built data
func (mb *MessageBuilder) Build() map[string]interface{} {
	return mb.data
}

// Mention defines the interface for platform mentions
// This implements the proposal's universal mention interface
type Mention interface {
	Platform() string
	Type() MentionType
	Target() string
	Render(format string) (string, error) // Use string instead of message.Format
}

// MentionType represents different types of mentions
type MentionType string

const (
	MentionTypeUser    MentionType = "user"
	MentionTypeGroup   MentionType = "group"
	MentionTypeChannel MentionType = "channel"
	MentionTypeAll     MentionType = "all"
)
