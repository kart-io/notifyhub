package message

import (
	"time"

	"github.com/kart-io/notifyhub/internal"
)

// Format represents the format of a message
type Format string

const (
	FormatText     Format = "text"
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
	FormatJSON     Format = "json"
	FormatCard     Format = "card"
)

// TargetType represents the type of notification target
type TargetType string

const (
	TargetTypeEmail   TargetType = "email"
	TargetTypeUser    TargetType = "user"
	TargetTypeGroup   TargetType = "group"
	TargetTypeChannel TargetType = "channel"
	TargetTypeWebhook TargetType = "webhook"
)

// Target represents a notification destination
type Target struct {
	Type     TargetType        `json:"type"`
	Value    string            `json:"value"`
	Platform string            `json:"platform"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewTarget creates a new target
func NewTarget(targetType TargetType, value, platform string) Target {
	return Target{
		Type:     targetType,
		Value:    value,
		Platform: platform,
		Metadata: make(map[string]string),
	}
}

// AddMetadata adds metadata to the target
func (t *Target) AddMetadata(key, value string) *Target {
	if t.Metadata == nil {
		t.Metadata = make(map[string]string)
	}
	t.Metadata[key] = value
	return t
}

// GetMetadata retrieves metadata value
func (t *Target) GetMetadata(key string) (string, bool) {
	if t.Metadata == nil {
		return "", false
	}
	value, exists := t.Metadata[key]
	return value, exists
}

// String returns a string representation of the target
func (t Target) String() string {
	return t.Platform + ":" + string(t.Type) + ":" + t.Value
}

// GetPlatform returns the target platform
func (t *Target) GetPlatform() string {
	return t.Platform
}

// GetValue returns the target value
func (t *Target) GetValue() string {
	return t.Value
}

// GetType returns the target type
func (t *Target) GetType() TargetType {
	return t.Type
}

// Validate checks if the target is valid
func (t *Target) Validate() error {
	if t.Type == "" {
		return ErrInvalidTargetType
	}
	if t.Value == "" {
		return ErrEmptyTargetValue
	}
	if t.Platform == "" {
		return ErrEmptyPlatform
	}
	return nil
}

// Priority levels for messages
const (
	PriorityLow      = 1
	PriorityNormal   = 2
	PriorityMedium   = 3
	PriorityHigh     = 4
	PriorityCritical = 5
)

// Message represents a domain message entity
type Message struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Format    Format                 `json:"format"`
	Priority  int                    `json:"priority"`
	Template  string                 `json:"template,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	CardData  interface{}            `json:"card_data,omitempty"`
	Targets   []Target               `json:"targets,omitempty"`
	Delay     time.Duration          `json:"delay,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// NewMessage creates a new message with default values
func NewMessage() *Message {
	now := time.Now()
	return &Message{
		ID:        internal.GenerateID(),
		Format:    FormatText,
		Priority:  PriorityNormal,
		Variables: make(map[string]interface{}),
		Metadata:  make(map[string]string),
		Targets:   make([]Target, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetTitle sets the message title
func (m *Message) SetTitle(title string) *Message {
	m.Title = title
	m.UpdatedAt = time.Now()
	return m
}

// SetBody sets the message body
func (m *Message) SetBody(body string) *Message {
	m.Body = body
	m.UpdatedAt = time.Now()
	return m
}

// SetFormat sets the message format
func (m *Message) SetFormat(format Format) *Message {
	m.Format = format
	m.UpdatedAt = time.Now()
	return m
}

// SetPriority sets the message priority
func (m *Message) SetPriority(priority int) *Message {
	m.Priority = priority
	m.UpdatedAt = time.Now()
	return m
}

// SetTemplate sets the template name
func (m *Message) SetTemplate(template string) *Message {
	m.Template = template
	m.UpdatedAt = time.Now()
	return m
}

// AddVariable adds a template variable
func (m *Message) AddVariable(key string, value interface{}) *Message {
	m.Variables[key] = value
	m.UpdatedAt = time.Now()
	return m
}

// AddMetadata adds metadata
func (m *Message) AddMetadata(key, value string) *Message {
	m.Metadata[key] = value
	m.UpdatedAt = time.Now()
	return m
}

// GetMetadata returns all metadata
func (m *Message) GetMetadata() map[string]string {
	return m.Metadata
}

// SetMetadata sets metadata for a key
func (m *Message) SetMetadata(key, value string) *Message {
	m.Metadata[key] = value
	m.UpdatedAt = time.Now()
	return m
}

// SetCardData sets custom card data
func (m *Message) SetCardData(cardData interface{}) *Message {
	m.CardData = cardData
	m.UpdatedAt = time.Now()
	return m
}

// AddTarget adds a target to the message
func (m *Message) AddTarget(target Target) *Message {
	m.Targets = append(m.Targets, target)
	m.UpdatedAt = time.Now()
	return m
}

// GetTargets returns the message targets
func (m *Message) GetTargets() []Target {
	return m.Targets
}

// GetTitle returns the message title
func (m *Message) GetTitle() string {
	return m.Title
}

// GetBody returns the message body
func (m *Message) GetBody() string {
	return m.Body
}

// SetVariable sets a template variable
func (m *Message) SetVariable(key string, value interface{}) *Message {
	m.Variables[key] = value
	m.UpdatedAt = time.Now()
	return m
}

// GetVariables returns all template variables
func (m *Message) GetVariables() map[string]interface{} {
	return m.Variables
}

// SetVariables sets multiple template variables at once
func (m *Message) SetVariables(variables map[string]interface{}) *Message {
	m.Variables = make(map[string]interface{})
	for k, v := range variables {
		m.Variables[k] = v
	}
	m.UpdatedAt = time.Now()
	return m
}

// Clone creates a deep copy of the message
func (m *Message) Clone() *Message {
	clone := &Message{
		ID:        m.ID,
		Title:     m.Title,
		Body:      m.Body,
		Format:    m.Format,
		Priority:  m.Priority,
		Template:  m.Template,
		CardData:  m.CardData,
		Delay:     m.Delay,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	// Deep copy maps
	clone.Variables = make(map[string]interface{})
	for k, v := range m.Variables {
		clone.Variables[k] = v
	}

	clone.Metadata = make(map[string]string)
	for k, v := range m.Metadata {
		clone.Metadata[k] = v
	}

	// Deep copy targets
	clone.Targets = make([]Target, len(m.Targets))
	copy(clone.Targets, m.Targets)

	return clone
}

// Validate checks if the message is valid
func (m *Message) Validate() error {
	if m.Title == "" && m.Body == "" {
		return ErrEmptyMessage
	}
	if m.Priority < PriorityLow || m.Priority > PriorityCritical {
		return ErrInvalidPriority
	}
	return nil
}

// GetID returns the message ID
func (m *Message) GetID() string {
	return m.ID
}

// GetPriority returns the message priority
func (m *Message) GetPriority() int {
	return m.Priority
}
