package notifiers

import (
	"context"
	"fmt"
	"strings"
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
	TargetTypeGroup   TargetType = "group"   // Group/Channel notifications
	TargetTypeUser    TargetType = "user"    // Individual user notifications
	TargetTypeEmail   TargetType = "email"   // Email notifications
	TargetTypeChannel TargetType = "channel" // Channel notifications
)

// MessageFormat defines the format of message content
type MessageFormat string

const (
	FormatText     MessageFormat = "text"
	FormatMarkdown MessageFormat = "markdown"
	FormatHTML     MessageFormat = "html"
	FormatCard     MessageFormat = "card"
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
	Priority  int                    `json:"priority"`        // 1=low, 5=urgent
	Delay     time.Duration          `json:"delay,omitempty"` // 延迟发送时间
	CardData  interface{}            `json:"card_data,omitempty"` // 卡片数据，支持平台特定的卡片格式
	CreatedAt time.Time              `json:"created_at"`
}

// SendResult represents the result of a send operation
type SendResult struct {
	Target   Target        `json:"target"`
	Platform string        `json:"platform"`
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
	SentAt   time.Time     `json:"sent_at"`
	Attempts int           `json:"attempts"`
}

// ValidationError represents validation errors for messages and targets
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// Error implements the error interface for ValidationError
func (ve *ValidationError) Error() string {
	if ve.Value != "" {
		return fmt.Sprintf("validation failed for field '%s': %s (value: '%s')", ve.Field, ve.Message, ve.Value)
	}
	return fmt.Sprintf("validation failed for field '%s': %s", ve.Field, ve.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

// Error implements the error interface for ValidationErrors
func (ves ValidationErrors) Error() string {
	var messages []string
	for _, ve := range ves {
		messages = append(messages, ve.Error())
	}
	return strings.Join(messages, "; ")
}

// Add adds a validation error to the collection
func (ves *ValidationErrors) Add(field, message, value string) {
	*ves = append(*ves, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are validation errors
func (ves ValidationErrors) HasErrors() bool {
	return len(ves) > 0
}

// Notifier interface that all notification platforms must implement
type Notifier interface {
	Name() string
	Send(ctx context.Context, message *Message) ([]*SendResult, error)
	SupportsTarget(target Target) bool
	Health(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// ValidateMessage validates a message for basic requirements
func ValidateMessage(msg *Message) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	var errors ValidationErrors

	// Validate title
	if strings.TrimSpace(msg.Title) == "" {
		errors.Add("title", "title is required and cannot be empty", msg.Title)
	} else if len(msg.Title) > 200 {
		errors.Add("title", "title exceeds maximum length of 200 characters", msg.Title)
	}

	// Validate body
	if strings.TrimSpace(msg.Body) == "" {
		errors.Add("body", "body is required and cannot be empty", msg.Body)
	} else if len(msg.Body) > 5000 {
		errors.Add("body", "body exceeds maximum length of 5000 characters", msg.Body)
	}

	// Validate priority
	if msg.Priority < 1 || msg.Priority > 5 {
		errors.Add("priority", "priority must be between 1 and 5", fmt.Sprintf("%d", msg.Priority))
	}

	// Validate targets
	if len(msg.Targets) == 0 {
		errors.Add("targets", "at least one target is required", "")
	} else {
		for i, target := range msg.Targets {
			if err := ValidateTarget(target); err != nil {
				if validationErr, ok := err.(ValidationErrors); ok {
					for _, ve := range validationErr {
						errors.Add(fmt.Sprintf("targets[%d].%s", i, ve.Field), ve.Message, ve.Value)
					}
				} else {
					errors.Add(fmt.Sprintf("targets[%d]", i), err.Error(), "")
				}
			}
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// ValidateTarget validates a notification target
func ValidateTarget(target Target) error {
	var errors ValidationErrors

	// Validate target type
	validTypes := []TargetType{TargetTypeEmail, TargetTypeUser, TargetTypeGroup, TargetTypeChannel}
	validType := false
	for _, vt := range validTypes {
		if target.Type == vt {
			validType = true
			break
		}
	}
	if !validType {
		typeStrs := make([]string, len(validTypes))
		for i, vt := range validTypes {
			typeStrs[i] = string(vt)
		}
		errors.Add("type", fmt.Sprintf("invalid target type, must be one of: %s", strings.Join(typeStrs, ", ")), string(target.Type))
	}

	// Validate target value
	value := strings.TrimSpace(target.Value)
	if value == "" {
		errors.Add("value", "target value cannot be empty", target.Value)
	} else if len(value) > 255 {
		errors.Add("value", "target value exceeds maximum length of 255 characters", target.Value)
	}

	// Type-specific validation
	switch target.Type {
	case TargetTypeEmail:
		if !isValidEmail(value) {
			errors.Add("value", "invalid email format", value)
		}
	case TargetTypeUser, TargetTypeGroup, TargetTypeChannel:
		if len(value) < 1 {
			errors.Add("value", fmt.Sprintf("%s identifier cannot be empty", string(target.Type)), value)
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) == 0 || len(email) > 254 {
		return false
	}

	// Basic email format validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	local, domain := parts[0], parts[1]
	if len(local) == 0 || len(local) > 64 || len(domain) == 0 || len(domain) > 253 {
		return false
	}

	// Check for at least one dot in domain
	if !strings.Contains(domain, ".") {
		return false
	}

	return true
}
