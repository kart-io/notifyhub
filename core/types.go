package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/internal"
)

// Message represents a unified notification message
type Message struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Format    Format                 `json:"format"`
	Priority  Priority               `json:"priority"`
	Targets   []Target               `json:"targets"`
	Template  string                 `json:"template,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	Delay     time.Duration          `json:"delay,omitempty"`
	CardData  interface{}            `json:"card_data,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Target represents a unified notification target
type Target struct {
	Type     TargetType        `json:"type"`
	Value    string            `json:"value"`
	Platform string            `json:"platform"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Result represents a unified sending result
type Result struct {
	MessageID string        `json:"message_id"`
	Target    Target        `json:"target"`
	Status    Status        `json:"status"`
	Platform  string        `json:"platform"`
	Error     error         `json:"error,omitempty"`
	Response  interface{}   `json:"response,omitempty"`
	SentAt    *time.Time    `json:"sent_at,omitempty"`
	Duration  time.Duration `json:"duration"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Timestamp time.Time     `json:"timestamp"`
	Success   bool          `json:"success"`
}

// Transport defines the unified interface for platform transports
type Transport interface {
	// Send sends a message to a target
	Send(ctx context.Context, msg *Message, target Target) (*Result, error)

	// Name returns the transport name
	Name() string

	// Health checks if the transport is healthy
	Health(ctx context.Context) error

	// Shutdown gracefully shuts down the transport
	Shutdown() error
}

// Format represents message format types
type Format string

const (
	FormatText     Format = "text"
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
	FormatCard     Format = "card"
)

// Priority represents message priority levels
type Priority int

const (
	PriorityLow      Priority = 1
	PriorityNormal   Priority = 2
	PriorityMedium   Priority = 3
	PriorityHigh     Priority = 4
	PriorityCritical Priority = 5
)

// TargetType represents target types
type TargetType string

const (
	TargetTypeEmail   TargetType = "email"
	TargetTypeUser    TargetType = "user"
	TargetTypeGroup   TargetType = "group"
	TargetTypeChannel TargetType = "channel"
	TargetTypeSMS     TargetType = "sms"
	TargetTypeOther   TargetType = "other"
)

// Status represents sending status
type Status string

const (
	StatusPending  Status = "pending"
	StatusSending  Status = "sending"
	StatusSent     Status = "sent"
	StatusFailed   Status = "failed"
	StatusRetrying Status = "retrying"
)

// NewMessage creates a new message with default values
func NewMessage() *Message {
	now := time.Now()
	return &Message{
		ID:        internal.GenerateID(),
		Format:    FormatText,
		Priority:  PriorityNormal,
		Targets:   make([]Target, 0),
		Variables: make(map[string]interface{}),
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
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

// NewResult creates a new result
func NewResult(messageID string, target Target) *Result {
	now := time.Now()
	return &Result{
		MessageID: messageID,
		Target:    target,
		Status:    StatusPending,
		Platform:  target.Platform,
		CreatedAt: now,
		UpdatedAt: now,
		Timestamp: now,
		Success:   false,
	}
}

// Validate validates the message
func (m *Message) Validate() error {
	if strings.TrimSpace(m.Title) == "" && strings.TrimSpace(m.Body) == "" {
		return fmt.Errorf("message must have either title or body")
	}

	if len(m.Title) > 200 {
		return fmt.Errorf("title exceeds maximum length of 200 characters")
	}

	if len(m.Body) > 5000 {
		return fmt.Errorf("body exceeds maximum length of 5000 characters")
	}

	if m.Priority < PriorityLow || m.Priority > PriorityCritical {
		return fmt.Errorf("priority must be between %d and %d", PriorityLow, PriorityCritical)
	}

	if len(m.Targets) == 0 {
		return fmt.Errorf("at least one target is required")
	}

	for i, target := range m.Targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf("target[%d]: %w", i, err)
		}
	}

	return nil
}

// Validate validates the target
func (t *Target) Validate() error {
	if t.Type == "" {
		return fmt.Errorf("target type is required")
	}

	if strings.TrimSpace(t.Value) == "" {
		return fmt.Errorf("target value is required")
	}

	if len(t.Value) > 255 {
		return fmt.Errorf("target value exceeds maximum length of 255 characters")
	}

	if t.Platform == "" {
		return fmt.Errorf("target platform is required")
	}

	// Type-specific validation
	switch t.Type {
	case TargetTypeEmail:
		if !isValidEmail(t.Value) {
			return fmt.Errorf("invalid email format: %s", t.Value)
		}
	case TargetTypeUser, TargetTypeGroup, TargetTypeChannel, TargetTypeSMS:
		if len(strings.TrimSpace(t.Value)) < 1 {
			return fmt.Errorf("%s identifier cannot be empty", string(t.Type))
		}
	default:
		return fmt.Errorf("unknown target type: %s", string(t.Type))
	}

	return nil
}

// String returns a string representation of the target
func (t *Target) String() string {
	return fmt.Sprintf("%s:%s@%s", t.Type, t.Value, t.Platform)
}

// AddMetadata adds metadata to the target
func (t *Target) AddMetadata(key, value string) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]string)
	}
	t.Metadata[key] = value
}

// SetStatus sets the result status
func (r *Result) SetStatus(status Status) {
	r.Status = status
	r.UpdatedAt = time.Now()

	if status == StatusSent {
		r.Success = true
		now := time.Now()
		r.SentAt = &now
	}
}

// SetError sets the result error
func (r *Result) SetError(err error) {
	r.Error = err
	r.Status = StatusFailed
	r.Success = false
	r.UpdatedAt = time.Now()
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) == 0 || len(email) > 254 {
		return false
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	local, domain := parts[0], parts[1]
	if len(local) == 0 || len(local) > 64 || len(domain) == 0 || len(domain) > 253 {
		return false
	}

	if !strings.Contains(domain, ".") {
		return false
	}

	return true
}
