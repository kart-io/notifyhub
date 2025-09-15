package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/internal"
	"github.com/kart-io/notifyhub/notifiers"
)

// MessageBuilder provides fluent interface for building messages
type MessageBuilder struct {
	message *notifiers.Message
	debug   bool // Enable debug output for this message
}

// NewMessage creates a new message builder
func NewMessage() *MessageBuilder {
	return &MessageBuilder{
		message: &notifiers.Message{
			Format:    notifiers.FormatText,
			Variables: make(map[string]interface{}),
			Metadata:  make(map[string]string),
			Priority:  3,
			CreatedAt: time.Now(),
		},
	}
}

// Title sets the message title
func (b *MessageBuilder) Title(title string) *MessageBuilder {
	b.message.Title = title
	return b
}

// Body sets the message body
func (b *MessageBuilder) Body(body string) *MessageBuilder {
	b.message.Body = body
	return b
}

// Format sets the message format
func (b *MessageBuilder) Format(format notifiers.MessageFormat) *MessageBuilder {
	b.message.Format = format
	return b
}

// Template sets the template name
func (b *MessageBuilder) Template(template string) *MessageBuilder {
	b.message.Template = template
	return b
}

// Variable adds a template variable
func (b *MessageBuilder) Variable(key string, value interface{}) *MessageBuilder {
	b.message.Variables[key] = value
	return b
}

// Variables adds multiple template variables
func (b *MessageBuilder) Variables(variables map[string]interface{}) *MessageBuilder {
	for k, v := range variables {
		b.message.Variables[k] = v
	}
	return b
}

// Metadata adds metadata
func (b *MessageBuilder) Metadata(key, value string) *MessageBuilder {
	b.message.Metadata[key] = value
	return b
}

// Priority sets the message priority
func (b *MessageBuilder) Priority(priority int) *MessageBuilder {
	b.message.Priority = priority
	return b
}

// Target adds a notification target
func (b *MessageBuilder) Target(target notifiers.Target) *MessageBuilder {
	b.message.Targets = append(b.message.Targets, target)
	return b
}

// Email adds an email target
func (b *MessageBuilder) Email(email string) *MessageBuilder {
	return b.Target(notifiers.Target{Type: notifiers.TargetTypeEmail, Value: email})
}

// FeishuGroup adds a Feishu group target
func (b *MessageBuilder) FeishuGroup(groupID string) *MessageBuilder {
	return b.Target(notifiers.Target{Type: notifiers.TargetTypeGroup, Value: groupID, Platform: "feishu"})
}

// FeishuUser adds a Feishu user target
func (b *MessageBuilder) FeishuUser(userID string) *MessageBuilder {
	return b.Target(notifiers.Target{Type: notifiers.TargetTypeUser, Value: userID, Platform: "feishu"})
}

// Build creates the final message
func (b *MessageBuilder) Build() *notifiers.Message {
	if b.message.ID == "" {
		b.message.ID = internal.GenerateID()
	}
	return b.message
}

// Convenience builders for different message types

// NewAlert creates a new alert message builder
func NewAlert(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Priority(4).
		Metadata("type", "alert").
		Format(notifiers.FormatText)
}

// NewNotice creates a new notice message builder
func NewNotice(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Priority(3).
		Metadata("type", "notice").
		Format(notifiers.FormatText)
}

// NewReport creates a new report message builder
func NewReport(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Priority(2).
		Metadata("type", "report").
		Format(notifiers.FormatText)
}

// NewMarkdown creates a new markdown message builder
func NewMarkdown(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Format(notifiers.FormatMarkdown)
}

// NewHTML creates a new HTML message builder
func NewHTML(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Format(notifiers.FormatHTML)
}

// Additional convenience methods for fluent API

// Delay sets the delay for message sending
func (b *MessageBuilder) Delay(delay time.Duration) *MessageBuilder {
	b.message.Delay = delay
	return b
}

// ID sets a custom message ID
func (b *MessageBuilder) ID(id string) *MessageBuilder {
	b.message.ID = id
	return b
}

// User adds a user target (platform agnostic)
func (b *MessageBuilder) User(userID, platform string) *MessageBuilder {
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeUser,
		Value:    userID,
		Platform: platform,
	})
}

// Group adds a group target (platform agnostic)
func (b *MessageBuilder) Group(groupID, platform string) *MessageBuilder {
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeGroup,
		Value:    groupID,
		Platform: platform,
	})
}

// Channel adds a channel target (platform agnostic)
func (b *MessageBuilder) Channel(channelID, platform string) *MessageBuilder {
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeChannel,
		Value:    channelID,
		Platform: platform,
	})
}

// MultipleEmails adds multiple email targets
func (b *MessageBuilder) MultipleEmails(emails ...string) *MessageBuilder {
	for _, email := range emails {
		b.Email(email)
	}
	return b
}

// MultipleUsers adds multiple user targets for a platform
func (b *MessageBuilder) MultipleUsers(platform string, userIDs ...string) *MessageBuilder {
	for _, userID := range userIDs {
		b.User(userID, platform)
	}
	return b
}

// WithTemplate is an alias for Template for better fluent API
func (b *MessageBuilder) WithTemplate(template string) *MessageBuilder {
	return b.Template(template)
}

// WithVariables is an alias for Variables for better fluent API
func (b *MessageBuilder) WithVariables(variables map[string]interface{}) *MessageBuilder {
	return b.Variables(variables)
}

// Urgent sets priority to 5 (urgent)
func (b *MessageBuilder) Urgent() *MessageBuilder {
	return b.Priority(5)
}

// High sets priority to 4 (high)
func (b *MessageBuilder) High() *MessageBuilder {
	return b.Priority(4)
}

// Normal sets priority to 3 (normal)
func (b *MessageBuilder) Normal() *MessageBuilder {
	return b.Priority(3)
}

// Low sets priority to 2 (low)
func (b *MessageBuilder) Low() *MessageBuilder {
	return b.Priority(2)
}

// Minimal sets priority to 1 (minimal)
func (b *MessageBuilder) Minimal() *MessageBuilder {
	return b.Priority(1)
}

// Validate validates the message being built
func (b *MessageBuilder) Validate() error {
	return notifiers.ValidateMessage(b.message)
}

// BuildAndValidate validates and builds the message
func (b *MessageBuilder) BuildAndValidate() (*notifiers.Message, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return b.Build(), nil
}

// Advanced Builder Methods for Common Scenarios

// QuickEmail builds a quick email message with smart defaults
func QuickEmail(title, body, email string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Email(email).
		Normal()
}

// QuickAlert builds a quick alert message with high priority
func QuickAlert(title, body string) *MessageBuilder {
	return NewAlert(title, body).
		High()
}

// QuickNotice builds a quick notice message with normal priority
func QuickNotice(title, body string) *MessageBuilder {
	return NewNotice(title, body).
		Normal()
}

// BatchEmail creates a message for multiple email recipients
func BatchEmail(title, body string, emails ...string) *MessageBuilder {
	builder := NewMessage().
		Title(title).
		Body(body).
		Normal()

	for _, email := range emails {
		builder.Email(email)
	}
	return builder
}

// TemplatedMessage creates a message with template and variables
func TemplatedMessage(template string, variables map[string]interface{}) *MessageBuilder {
	return NewMessage().
		WithTemplate(template).
		WithVariables(variables).
		Normal()
}

// Builder Validation Enhancements

// MustBuild builds the message and panics on validation error (useful for testing)
func (b *MessageBuilder) MustBuild() *notifiers.Message {
	message, err := b.BuildAndValidate()
	if err != nil {
		panic(fmt.Sprintf("Message validation failed: %v", err))
	}
	return message
}

// TryBuild attempts to build the message and returns nil on error (safe fallback)
func (b *MessageBuilder) TryBuild() *notifiers.Message {
	message, err := b.BuildAndValidate()
	if err != nil {
		return nil
	}
	return message
}

// Smart Target Methods

// ToEmail adds an email target with validation
func (b *MessageBuilder) ToEmail(email string) *MessageBuilder {
	return b.Email(email)
}

// ToUsers adds multiple user targets for a platform
func (b *MessageBuilder) ToUsers(platform string, userIDs ...string) *MessageBuilder {
	for _, userID := range userIDs {
		b.User(userID, platform)
	}
	return b
}

// ToChannels adds multiple channel targets for a platform
func (b *MessageBuilder) ToChannels(platform string, channelIDs ...string) *MessageBuilder {
	for _, channelID := range channelIDs {
		b.Channel(channelID, platform)
	}
	return b
}

// ToSlack adds Slack targets (users or channels)
func (b *MessageBuilder) ToSlack(target string) *MessageBuilder {
	if strings.HasPrefix(target, "#") {
		return b.Channel(strings.TrimPrefix(target, "#"), "slack")
	} else if strings.HasPrefix(target, "@") {
		return b.User(strings.TrimPrefix(target, "@"), "slack")
	} else {
		return b.User(target, "slack")
	}
}

// ToFeishu adds Feishu targets
func (b *MessageBuilder) ToFeishu(target string) *MessageBuilder {
	// Smart detection based on target format
	if strings.Contains(target, "@") {
		return b.Email(target)
	} else {
		return b.User(target, "feishu")
	}
}

// Conditional Building

// If adds a condition to the builder chain
func (b *MessageBuilder) If(condition bool, fn func(*MessageBuilder) *MessageBuilder) *MessageBuilder {
	if condition {
		return fn(b)
	}
	return b
}

// Unless adds a negative condition to the builder chain
func (b *MessageBuilder) Unless(condition bool, fn func(*MessageBuilder) *MessageBuilder) *MessageBuilder {
	if !condition {
		return fn(b)
	}
	return b
}

// Smart Formatting

// AsMarkdown sets the message format to markdown
func (b *MessageBuilder) AsMarkdown() *MessageBuilder {
	return b.Format(notifiers.FormatMarkdown)
}

// AsHTML sets the message format to HTML
func (b *MessageBuilder) AsHTML() *MessageBuilder {
	return b.Format(notifiers.FormatHTML)
}

// AsPlainText sets the message format to plain text
func (b *MessageBuilder) AsPlainText() *MessageBuilder {
	return b.Format(notifiers.FormatText)
}

// Clone creates a copy of the current builder
func (b *MessageBuilder) Clone() *MessageBuilder {
	newBuilder := &MessageBuilder{
		message: &notifiers.Message{
			ID:        b.message.ID,
			Title:     b.message.Title,
			Body:      b.message.Body,
			Format:    b.message.Format,
			Priority:  b.message.Priority,
			Template:  b.message.Template,
			Delay:     b.message.Delay,
			CreatedAt: b.message.CreatedAt,
		},
	}

	// Deep copy targets
	newBuilder.message.Targets = make([]notifiers.Target, len(b.message.Targets))
	copy(newBuilder.message.Targets, b.message.Targets)

	// Deep copy variables
	newBuilder.message.Variables = make(map[string]interface{})
	for k, v := range b.message.Variables {
		newBuilder.message.Variables[k] = v
	}

	// Deep copy metadata
	newBuilder.message.Metadata = make(map[string]string)
	for k, v := range b.message.Metadata {
		newBuilder.message.Metadata[k] = v
	}

	// Copy debug flag
	newBuilder.debug = b.debug

	return newBuilder
}

// ==========================================
// Debug and Inspection Methods
// ==========================================

// Debug enables debug mode for this message builder
// When enabled, it outputs detailed information about message construction
func (b *MessageBuilder) Debug() *MessageBuilder {
	b.debug = true
	fmt.Printf("🐛 [MessageBuilder] Debug mode enabled for message: %s\n", b.message.ID)
	fmt.Printf("🐛 [MessageBuilder] Current state:\n")
	fmt.Printf("   📋 Title: %q\n", b.message.Title)
	fmt.Printf("   📝 Body: %q\n", b.message.Body)
	fmt.Printf("   🎯 Targets: %d\n", len(b.message.Targets))
	for i, target := range b.message.Targets {
		fmt.Printf("      %d. %s:%s (%s)\n", i+1, target.Type, target.Value, target.Platform)
	}
	fmt.Printf("   🔢 Priority: %d\n", b.message.Priority)
	fmt.Printf("   📊 Format: %s\n", b.message.Format)
	if b.message.Template != "" {
		fmt.Printf("   📄 Template: %s\n", b.message.Template)
	}
	if len(b.message.Variables) > 0 {
		fmt.Printf("   🔧 Variables: %d\n", len(b.message.Variables))
		for k, v := range b.message.Variables {
			fmt.Printf("      %s: %v\n", k, v)
		}
	}
	if len(b.message.Metadata) > 0 {
		fmt.Printf("   🏷️  Metadata: %d\n", len(b.message.Metadata))
		for k, v := range b.message.Metadata {
			fmt.Printf("      %s: %s\n", k, v)
		}
	}
	if b.message.Delay > 0 {
		fmt.Printf("   ⏰ Delay: %v\n", b.message.Delay)
	}
	fmt.Printf("   🕐 Created: %v\n", b.message.CreatedAt)
	fmt.Println("🐛 [MessageBuilder] Debug output complete")
	return b
}

// Inspect returns a human-readable string representation of the message
func (b *MessageBuilder) Inspect() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Message ID: %s", b.message.ID))
	parts = append(parts, fmt.Sprintf("Title: %q", b.message.Title))
	parts = append(parts, fmt.Sprintf("Body: %q", b.message.Body))
	parts = append(parts, fmt.Sprintf("Targets: %d", len(b.message.Targets)))
	parts = append(parts, fmt.Sprintf("Priority: %d", b.message.Priority))
	parts = append(parts, fmt.Sprintf("Format: %s", b.message.Format))
	if b.message.Template != "" {
		parts = append(parts, fmt.Sprintf("Template: %s", b.message.Template))
	}
	if len(b.message.Variables) > 0 {
		parts = append(parts, fmt.Sprintf("Variables: %d", len(b.message.Variables)))
	}
	if len(b.message.Metadata) > 0 {
		parts = append(parts, fmt.Sprintf("Metadata: %d", len(b.message.Metadata)))
	}
	return strings.Join(parts, ", ")
}

// IsDebugEnabled returns whether debug mode is enabled for this builder
func (b *MessageBuilder) IsDebugEnabled() bool {
	return b.debug
}