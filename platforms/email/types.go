package email

import "errors"

// Email-specific errors
var (
	ErrInvalidConfig      = errors.New("invalid email config")
	ErrEmptyMessage       = errors.New("message cannot be empty")
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrAttachmentTooLarge = errors.New("attachment too large")
)

// EmailPriority represents email priority levels
type EmailPriority string

const (
	PriorityLow    EmailPriority = "low"
	PriorityNormal EmailPriority = "normal"
	PriorityHigh   EmailPriority = "high"
	PriorityUrgent EmailPriority = "urgent"
)

// Attachment represents an email attachment
type Attachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"` // Base64 encoded content
	MimeType string `json:"mime_type"`
	Size     int    `json:"size"`
}

// EmailConfig holds Email-specific configuration
type EmailConfig struct {
	SMTPHost    string
	SMTPPort    int
	Username    string
	Password    string
	FromAddress string
	FromName    string
	UseTLS      bool
	UseSTARTTLS bool
	Timeout     int
}

// EmailTarget represents an Email-specific target
type EmailTarget struct {
	Type    string   `json:"type"`
	Address string   `json:"address"`
	Name    string   `json:"name,omitempty"`
	CCList  []string `json:"cc_list,omitempty"`
	BCCList []string `json:"bcc_list,omitempty"`
}

// EmailMessage represents a complete Email message
type EmailMessage struct {
	To          []string          `json:"to"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	BodyHTML    string            `json:"body_html,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	Priority    EmailPriority     `json:"priority,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}
