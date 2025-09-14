package notifiers

import (
	"context"
	"time"
)

// Target represents a notification destination
type Target struct {
	Type     TargetType        `json:"type" yaml:"type"`
	Value    string            `json:"value" yaml:"value"`       // Email, user ID, group ID
	Platform string            `json:"platform" yaml:"platform"` // "feishu", "email"
	Metadata map[string]string `json:"metadata" yaml:"metadata"` // Platform-specific data
}

// TargetType defines the type of notification target
type TargetType string

const (
	TargetTypeGroup TargetType = "group" // Group/Channel notifications
	TargetTypeUser  TargetType = "user"  // Individual user notifications
	TargetTypeEmail TargetType = "email" // Email notifications
)

// MessageFormat defines the format of message content
type MessageFormat string

const (
	FormatText     MessageFormat = "text"
	FormatMarkdown MessageFormat = "markdown"
	FormatHTML     MessageFormat = "html"
)

// Message represents a notification message
type Message struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Format    MessageFormat          `json:"format"`
	Targets   []Target               `json:"targets"`
	Template  string                 `json:"template,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	Priority  int                    `json:"priority"` // 1=low, 5=urgent
	Delay     time.Duration          `json:"delay,omitempty"`     // 延迟发送时间
	CreatedAt time.Time              `json:"created_at"`
}

// SendResult represents the result of a send operation
type SendResult struct {
	Target    Target        `json:"target"`
	Platform  string        `json:"platform"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	SentAt    time.Time     `json:"sent_at"`
	Attempts  int           `json:"attempts"`
}

// Notifier interface that all notification platforms must implement
type Notifier interface {
	Name() string
	Send(ctx context.Context, message *Message) ([]*SendResult, error)
	SupportsTarget(target Target) bool
	Health(ctx context.Context) error
	Shutdown(ctx context.Context) error
}