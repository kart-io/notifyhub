// Package message provides message builder functionality for NotifyHub
package message

import (
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// MessageBuilder provides fluent API for building messages
type MessageBuilder struct {
	message *Message
}

// NewMessage creates a new message builder with optional title
func NewMessage(title ...string) *MessageBuilder {
	msg := &Message{
		ID:           generateMessageID(),
		Format:       "text",
		Priority:     PriorityNormal,
		Targets:      make([]target.Target, 0),
		Metadata:     make(map[string]interface{}),
		Variables:    make(map[string]interface{}),
		PlatformData: make(map[string]interface{}),
	}

	if len(title) > 0 {
		msg.Title = title[0]
	}

	return &MessageBuilder{message: msg}
}

// NewAlert creates a new alert message (high priority by default)
func NewAlert(title string) *MessageBuilder {
	return NewMessage(title).WithPriority(PriorityHigh)
}

// NewUrgent creates a new urgent message (urgent priority by default)
func NewUrgent(title string) *MessageBuilder {
	return NewMessage(title).WithPriority(PriorityUrgent)
}

// WithTitle sets the message title
func (b *MessageBuilder) WithTitle(title string) *MessageBuilder {
	b.message.Title = title
	return b
}

// WithBody sets the message body
func (b *MessageBuilder) WithBody(body string) *MessageBuilder {
	b.message.Body = body
	return b
}

// WithText is an alias for WithBody for text messages
func (b *MessageBuilder) WithText(text string) *MessageBuilder {
	return b.WithBody(text)
}

// WithFormat sets the message format
func (b *MessageBuilder) WithFormat(format string) *MessageBuilder {
	b.message.Format = format
	return b
}

// WithMarkdown sets the message format to markdown and sets the body
func (b *MessageBuilder) WithMarkdown(content string) *MessageBuilder {
	b.message.Format = "markdown"
	b.message.Body = content
	return b
}

// WithHTML sets the message format to HTML and sets the body
func (b *MessageBuilder) WithHTML(content string) *MessageBuilder {
	b.message.Format = "html"
	b.message.Body = content
	return b
}

// WithPriority sets the message priority
func (b *MessageBuilder) WithPriority(priority Priority) *MessageBuilder {
	b.message.Priority = priority
	return b
}

// WithMetadata adds metadata to the message
func (b *MessageBuilder) WithMetadata(key string, value interface{}) *MessageBuilder {
	b.message.Metadata[key] = value
	return b
}

// WithVariable adds a template variable
func (b *MessageBuilder) WithVariable(key string, value interface{}) *MessageBuilder {
	b.message.Variables[key] = value
	return b
}

// WithVariables adds multiple template variables
func (b *MessageBuilder) WithVariables(variables map[string]interface{}) *MessageBuilder {
	for k, v := range variables {
		b.message.Variables[k] = v
	}
	return b
}

// WithPlatformData adds platform-specific data
func (b *MessageBuilder) WithPlatformData(data map[string]interface{}) *MessageBuilder {
	for k, v := range data {
		b.message.PlatformData[k] = v
	}
	return b
}

// ScheduleAt schedules the message for later delivery
func (b *MessageBuilder) ScheduleAt(when time.Time) *MessageBuilder {
	b.message.ScheduledAt = &when
	return b
}

// ScheduleIn schedules the message for delivery after a duration
func (b *MessageBuilder) ScheduleIn(duration time.Duration) *MessageBuilder {
	when := time.Now().Add(duration)
	b.message.ScheduledAt = &when
	return b
}

// ToTarget adds a generic target
func (b *MessageBuilder) ToTarget(t target.Target) *MessageBuilder {
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToTargets adds multiple targets
func (b *MessageBuilder) ToTargets(targets ...target.Target) *MessageBuilder {
	b.message.Targets = append(b.message.Targets, targets...)
	return b
}

// ToCustomTarget adds a custom target with type, value, and platform
func (b *MessageBuilder) ToCustomTarget(targetType, value, platform string) *MessageBuilder {
	t := target.NewTarget(targetType, value, platform)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToEmail adds an email target
func (b *MessageBuilder) ToEmail(email string) *MessageBuilder {
	t := target.NewEmailTarget(email)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToPhone adds a phone/SMS target
func (b *MessageBuilder) ToPhone(phone string) *MessageBuilder {
	t := target.NewPhoneTarget(phone)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToFeishu adds a Feishu target (can be user, group, or webhook)
func (b *MessageBuilder) ToFeishu(identifier string) *MessageBuilder {
	t := target.AutoDetectTarget(identifier)
	if t.Platform == "" {
		t.Platform = "feishu"
	}
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToFeishuUser adds a specific Feishu user target
func (b *MessageBuilder) ToFeishuUser(userID string) *MessageBuilder {
	t := target.NewFeishuUserTarget(userID)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToFeishuGroup adds a specific Feishu group target
func (b *MessageBuilder) ToFeishuGroup(groupID string) *MessageBuilder {
	t := target.NewFeishuGroupTarget(groupID)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToWebhook adds a webhook target
func (b *MessageBuilder) ToWebhook(url string) *MessageBuilder {
	t := target.NewWebhookTarget(url)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// Platform-specific fluent methods

// WithFeishuCard adds Feishu card content
func (b *MessageBuilder) WithFeishuCard(card interface{}) *MessageBuilder {
	b.message.PlatformData["feishu_card"] = card
	return b
}

// WithFeishuMentions adds Feishu user mentions
func (b *MessageBuilder) WithFeishuMentions(userIDs ...string) *MessageBuilder {
	mentions := make([]map[string]interface{}, len(userIDs))
	for i, userID := range userIDs {
		mentions[i] = map[string]interface{}{"user_id": userID}
	}
	b.message.PlatformData["feishu_mentions"] = mentions
	return b
}

// WithFeishuMentionAll adds Feishu mention all
func (b *MessageBuilder) WithFeishuMentionAll() *MessageBuilder {
	b.message.PlatformData["feishu_mention_all"] = true
	return b
}

// WithEmailCC adds email CC recipients
func (b *MessageBuilder) WithEmailCC(emails ...string) *MessageBuilder {
	b.message.PlatformData["email_cc"] = emails
	return b
}

// WithEmailBCC adds email BCC recipients
func (b *MessageBuilder) WithEmailBCC(emails ...string) *MessageBuilder {
	b.message.PlatformData["email_bcc"] = emails
	return b
}

// WithEmailPriority sets email priority
func (b *MessageBuilder) WithEmailPriority(priority string) *MessageBuilder {
	b.message.PlatformData["email_priority"] = priority
	return b
}

// WithEmailAttachments adds email attachments
func (b *MessageBuilder) WithEmailAttachments(attachments []string) *MessageBuilder {
	b.message.PlatformData["email_attachments"] = attachments
	return b
}

// WithSMSTemplate sets SMS template ID
func (b *MessageBuilder) WithSMSTemplate(templateID string) *MessageBuilder {
	b.message.PlatformData["sms_template"] = templateID
	return b
}

// WithSMSVariables sets SMS template variables
func (b *MessageBuilder) WithSMSVariables(variables map[string]interface{}) *MessageBuilder {
	b.message.PlatformData["sms_variables"] = variables
	return b
}

// Build returns the constructed message
func (b *MessageBuilder) Build() *Message {
	// Generate ID if not set
	if b.message.ID == "" {
		b.message.ID = generateMessageID()
	}

	return b.message
}