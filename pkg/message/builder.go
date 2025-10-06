// Package message provides message building functionality for NotifyHub
package message

import (
	"time"

	"github.com/kart-io/notifyhub/pkg/target"
	"github.com/kart-io/notifyhub/pkg/utils/idgen"
)

// Builder provides a fluent interface for building messages
type Builder struct {
	message *Message
}

// NewBuilder creates a new message builder
func NewBuilder() *Builder {
	return &Builder{
		message: &Message{
			ID:           idgen.GenerateMessageID(),
			Format:       FormatText,
			Priority:     PriorityNormal,
			CreatedAt:    time.Now(),
			Metadata:     make(map[string]interface{}),
			Variables:    make(map[string]interface{}),
			PlatformData: make(map[string]interface{}),
		},
	}
}

// NewMessage creates a new message builder (alias for NewBuilder)
func NewMessage() *Builder {
	return NewBuilder()
}

// SetID sets the message ID
func (b *Builder) SetID(id string) *Builder {
	b.message.ID = id
	return b
}

// SetTitle sets the message title
func (b *Builder) SetTitle(title string) *Builder {
	b.message.Title = title
	return b
}

// SetBody sets the message body
func (b *Builder) SetBody(body string) *Builder {
	b.message.Body = body
	return b
}

// SetFormat sets the message format
func (b *Builder) SetFormat(format Format) *Builder {
	b.message.Format = format
	return b
}

// SetPriority sets the message priority
func (b *Builder) SetPriority(priority Priority) *Builder {
	b.message.Priority = priority
	return b
}

// AddTarget adds a target to the message
func (b *Builder) AddTarget(target target.Target) *Builder {
	b.message.Targets = append(b.message.Targets, target)
	return b
}

// AddTargets adds multiple targets to the message
func (b *Builder) AddTargets(targets []target.Target) *Builder {
	b.message.Targets = append(b.message.Targets, targets...)
	return b
}

// SetTargets sets the message targets
func (b *Builder) SetTargets(targets []target.Target) *Builder {
	b.message.Targets = targets
	return b
}

// AddMetadata adds metadata to the message
func (b *Builder) AddMetadata(key string, value interface{}) *Builder {
	b.message.Metadata[key] = value
	return b
}

// SetMetadata sets the message metadata
func (b *Builder) SetMetadata(metadata map[string]interface{}) *Builder {
	b.message.Metadata = metadata
	return b
}

// AddVariable adds a template variable
func (b *Builder) AddVariable(key string, value interface{}) *Builder {
	b.message.Variables[key] = value
	return b
}

// SetVariables sets the template variables
func (b *Builder) SetVariables(variables map[string]interface{}) *Builder {
	b.message.Variables = variables
	return b
}

// AddPlatformData adds platform-specific data
func (b *Builder) AddPlatformData(platform string, data interface{}) *Builder {
	b.message.PlatformData[platform] = data
	return b
}

// SetPlatformData sets the platform-specific data
func (b *Builder) SetPlatformData(platformData map[string]interface{}) *Builder {
	b.message.PlatformData = platformData
	return b
}

// ScheduleAt sets when the message should be sent
func (b *Builder) ScheduleAt(scheduledAt time.Time) *Builder {
	b.message.ScheduledAt = &scheduledAt
	return b
}

// ScheduleAfter schedules the message to be sent after a duration
func (b *Builder) ScheduleAfter(duration time.Duration) *Builder {
	scheduledAt := time.Now().Add(duration)
	b.message.ScheduledAt = &scheduledAt
	return b
}

// Build returns the constructed message
func (b *Builder) Build() *Message {
	// Create a copy to avoid modification after build
	msg := *b.message

	// Deep copy slices and maps
	if len(b.message.Targets) > 0 {
		msg.Targets = make([]target.Target, len(b.message.Targets))
		copy(msg.Targets, b.message.Targets)
	}

	if len(b.message.Metadata) > 0 {
		msg.Metadata = make(map[string]interface{})
		for k, v := range b.message.Metadata {
			msg.Metadata[k] = v
		}
	}

	if len(b.message.Variables) > 0 {
		msg.Variables = make(map[string]interface{})
		for k, v := range b.message.Variables {
			msg.Variables[k] = v
		}
	}

	if len(b.message.PlatformData) > 0 {
		msg.PlatformData = make(map[string]interface{})
		for k, v := range b.message.PlatformData {
			msg.PlatformData[k] = v
		}
	}

	return &msg
}

// Reset resets the builder to build a new message
func (b *Builder) Reset() *Builder {
	b.message = &Message{
		ID:           idgen.GenerateMessageID(),
		Format:       FormatText,
		Priority:     PriorityNormal,
		CreatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
		Variables:    make(map[string]interface{}),
		PlatformData: make(map[string]interface{}),
	}
	return b
}

// Clone creates a new builder with a copy of the current message
func (b *Builder) Clone() *Builder {
	newBuilder := &Builder{
		message: b.Build(), // Build creates a copy
	}
	return newBuilder
}

// Convenience builder functions

// NewTextMessage creates a builder for a text message
func NewTextMessage(title, body string) *Builder {
	return NewBuilder().
		SetTitle(title).
		SetBody(body).
		SetFormat(FormatText)
}

// NewMarkdownMessage creates a builder for a markdown message
func NewMarkdownMessage(title, body string) *Builder {
	return NewBuilder().
		SetTitle(title).
		SetBody(body).
		SetFormat(FormatMarkdown)
}

// NewHTMLMessage creates a builder for an HTML message
func NewHTMLMessage(title, body string) *Builder {
	return NewBuilder().
		SetTitle(title).
		SetBody(body).
		SetFormat(FormatHTML)
}

// NewAlert creates a builder for an alert message
func NewAlert(title, body string) *Builder {
	return NewBuilder().
		SetTitle(title).
		SetBody(body).
		SetPriority(PriorityHigh).
		AddMetadata("type", "alert")
}

// NewNotification creates a builder for a notification message
func NewNotification(title, body string) *Builder {
	return NewBuilder().
		SetTitle(title).
		SetBody(body).
		SetPriority(PriorityNormal).
		AddMetadata("type", "notification")
}

// NewUrgentMessage creates a builder for an urgent message
func NewUrgentMessage(title, body string) *Builder {
	return NewBuilder().
		SetTitle(title).
		SetBody(body).
		SetPriority(PriorityUrgent).
		AddMetadata("type", "urgent")
}

// Template-based builders

// NewTemplateMessage creates a builder for a templated message
func NewTemplateMessage(templateID string, variables map[string]interface{}) *Builder {
	return NewBuilder().
		SetVariables(variables).
		AddMetadata("template_id", templateID)
}

// WithTemplate sets the template ID and variables
func (b *Builder) WithTemplate(templateID string, variables map[string]interface{}) *Builder {
	return b.
		SetVariables(variables).
		AddMetadata("template_id", templateID)
}
