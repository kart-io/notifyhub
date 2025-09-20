package api

import (
	"context"
	"encoding/json"

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/platforms/feishu"
)

// PlatformType represents the platform type
type PlatformType string

const (
	PlatformEmail  PlatformType = "email"
	PlatformFeishu PlatformType = "feishu"
	PlatformSlack  PlatformType = "slack"
)

// PlatformData interface for platform-specific data
type PlatformData interface {
	MarshalMetadata() (string, error)
	GetTargets() []message.Target
	Validate() error
}

// EmailPlatformData contains email-specific data
type EmailPlatformData struct {
	To       []string
	CC       []string
	BCC      []string
	Subject  string
	TextBody string
	HTMLBody string
	Headers  map[string]string
}

func (d *EmailPlatformData) MarshalMetadata() (string, error) {
	data, err := json.Marshal(d)
	return string(data), err
}

func (d *EmailPlatformData) GetTargets() []message.Target {
	var targets []message.Target
	for _, to := range d.To {
		targets = append(targets, message.NewTarget(message.TargetTypeEmail, to, "email"))
	}
	for _, cc := range d.CC {
		target := message.NewTarget(message.TargetTypeEmail, cc, "email")
		target.AddMetadata("type", "cc")
		targets = append(targets, target)
	}
	for _, bcc := range d.BCC {
		target := message.NewTarget(message.TargetTypeEmail, bcc, "email")
		target.AddMetadata("type", "bcc")
		targets = append(targets, target)
	}
	return targets
}

func (d *EmailPlatformData) Validate() error {
	if len(d.To) == 0 {
		return &ValidationError{Field: "to", Message: "at least one recipient must be specified"}
	}
	return nil
}

// FeishuPlatformData contains Feishu-specific data
type FeishuPlatformData struct {
	Groups  []string
	Users   []string
	AtAll   bool
	AtUsers []string
	Card    *feishu.FeishuCard
	Secret  string
	Webhook string
}

func (d *FeishuPlatformData) MarshalMetadata() (string, error) {
	data, err := json.Marshal(d)
	return string(data), err
}

func (d *FeishuPlatformData) GetTargets() []message.Target {
	var targets []message.Target
	for _, group := range d.Groups {
		targets = append(targets, message.NewTarget(message.TargetTypeGroup, group, "feishu"))
	}
	for _, user := range d.Users {
		targets = append(targets, message.NewTarget(message.TargetTypeUser, user, "feishu"))
	}
	return targets
}

func (d *FeishuPlatformData) Validate() error {
	if len(d.Groups) == 0 && len(d.Users) == 0 && d.Webhook == "" {
		return &ValidationError{Field: "targets", Message: "at least one group, user, or webhook must be specified"}
	}
	return nil
}

// SlackPlatformData contains Slack-specific data
type SlackPlatformData struct {
	Channels  []string
	Users     []string
	ThreadTs  string
	Broadcast bool
	LinkNames bool
	Webhook   string
}

func (d *SlackPlatformData) MarshalMetadata() (string, error) {
	data, err := json.Marshal(d)
	return string(data), err
}

func (d *SlackPlatformData) GetTargets() []message.Target {
	var targets []message.Target
	for _, channel := range d.Channels {
		targets = append(targets, message.NewTarget(message.TargetTypeChannel, channel, "slack"))
	}
	for _, user := range d.Users {
		targets = append(targets, message.NewTarget(message.TargetTypeUser, user, "slack"))
	}
	return targets
}

func (d *SlackPlatformData) Validate() error {
	if len(d.Channels) == 0 && len(d.Users) == 0 && d.Webhook == "" {
		return &ValidationError{Field: "targets", Message: "at least one channel, user, or webhook must be specified"}
	}
	return nil
}

// UnifiedPlatformBuilder provides unified platform-specific functionality
type UnifiedPlatformBuilder struct {
	*BaseBuilder
	platform     PlatformType
	platformData PlatformData
}

// NewUnifiedPlatformBuilder creates a new unified platform builder
func NewUnifiedPlatformBuilder(client *Client, platform PlatformType) *UnifiedPlatformBuilder {
	builder := &UnifiedPlatformBuilder{
		BaseBuilder: NewBaseBuilder(client),
		platform:    platform,
	}

	// Initialize platform-specific data based on platform type
	switch platform {
	case PlatformEmail:
		builder.platformData = &EmailPlatformData{Headers: make(map[string]string)}
	case PlatformFeishu:
		builder.platformData = &FeishuPlatformData{}
	case PlatformSlack:
		builder.platformData = &SlackPlatformData{}
	}

	return builder
}

// Title sets the message title
func (b *UnifiedPlatformBuilder) Title(title string) *UnifiedPlatformBuilder {
	b.BaseBuilder.Title(title)
	return b
}

// Body sets the message body
func (b *UnifiedPlatformBuilder) Body(body string) *UnifiedPlatformBuilder {
	b.BaseBuilder.Body(body)
	return b
}

// Priority sets the message priority
func (b *UnifiedPlatformBuilder) Priority(priority int) *UnifiedPlatformBuilder {
	b.BaseBuilder.Priority(priority)
	return b
}

// Template sets a template
func (b *UnifiedPlatformBuilder) Template(template string) *UnifiedPlatformBuilder {
	b.BaseBuilder.Template(template)
	return b
}

// Var sets template variables
func (b *UnifiedPlatformBuilder) Var(key string, value interface{}) *UnifiedPlatformBuilder {
	b.BaseBuilder.Var(key, value)
	return b
}

// Send sends the message
func (b *UnifiedPlatformBuilder) Send(ctx context.Context) (*sending.SendingResults, error) {
	if err := b.platformData.Validate(); err != nil {
		return nil, err
	}

	// Add platform-specific targets
	platformTargets := b.platformData.GetTargets()
	for _, target := range platformTargets {
		b.AddTarget(target)
	}

	// Store platform-specific metadata
	if b.message.Metadata == nil {
		b.message.Metadata = make(map[string]string)
	}

	metadata, err := b.platformData.MarshalMetadata()
	if err != nil {
		return nil, err
	}
	b.message.Metadata[string(b.platform)+"_data"] = metadata

	return b.Execute(ctx)
}

// DryRun validates the message without sending
func (b *UnifiedPlatformBuilder) DryRun() (*DryRunResult, error) {
	if err := b.platformData.Validate(); err != nil {
		return nil, err
	}

	targets := b.platformData.GetTargets()

	return &DryRunResult{
		Valid:   true,
		Message: *b.GetMessage(),
		Targets: targets,
	}, nil
}

// GetPlatformData returns the platform-specific data (for type assertions)
func (b *UnifiedPlatformBuilder) GetPlatformData() PlatformData {
	return b.platformData
}

// Email-specific convenience methods
func (b *UnifiedPlatformBuilder) To(emails ...string) *UnifiedPlatformBuilder {
	if emailData, ok := b.platformData.(*EmailPlatformData); ok {
		emailData.To = append(emailData.To, emails...)
	}
	return b
}

func (b *UnifiedPlatformBuilder) CC(emails ...string) *UnifiedPlatformBuilder {
	if emailData, ok := b.platformData.(*EmailPlatformData); ok {
		emailData.CC = append(emailData.CC, emails...)
	}
	return b
}

func (b *UnifiedPlatformBuilder) BCC(emails ...string) *UnifiedPlatformBuilder {
	if emailData, ok := b.platformData.(*EmailPlatformData); ok {
		emailData.BCC = append(emailData.BCC, emails...)
	}
	return b
}

func (b *UnifiedPlatformBuilder) Subject(subject string) *UnifiedPlatformBuilder {
	if emailData, ok := b.platformData.(*EmailPlatformData); ok {
		emailData.Subject = subject
	}
	return b
}

func (b *UnifiedPlatformBuilder) HTMLBody(htmlBody string) *UnifiedPlatformBuilder {
	if emailData, ok := b.platformData.(*EmailPlatformData); ok {
		emailData.HTMLBody = htmlBody
		b.GetMessage().Format = message.FormatHTML
	}
	return b
}

// Feishu-specific convenience methods
func (b *UnifiedPlatformBuilder) ToGroup(groups ...string) *UnifiedPlatformBuilder {
	if feishuData, ok := b.platformData.(*FeishuPlatformData); ok {
		feishuData.Groups = append(feishuData.Groups, groups...)
	}
	return b
}

func (b *UnifiedPlatformBuilder) ToUser(users ...string) *UnifiedPlatformBuilder {
	if feishuData, ok := b.platformData.(*FeishuPlatformData); ok {
		feishuData.Users = append(feishuData.Users, users...)
	} else if slackData, ok := b.platformData.(*SlackPlatformData); ok {
		slackData.Users = append(slackData.Users, users...)
	}
	return b
}

func (b *UnifiedPlatformBuilder) AtAll() *UnifiedPlatformBuilder {
	if feishuData, ok := b.platformData.(*FeishuPlatformData); ok {
		feishuData.AtAll = true
	}
	return b
}

func (b *UnifiedPlatformBuilder) AtUser(users ...string) *UnifiedPlatformBuilder {
	if feishuData, ok := b.platformData.(*FeishuPlatformData); ok {
		feishuData.AtUsers = append(feishuData.AtUsers, users...)
	}
	return b
}

func (b *UnifiedPlatformBuilder) Card(card *feishu.FeishuCard) *UnifiedPlatformBuilder {
	if feishuData, ok := b.platformData.(*FeishuPlatformData); ok {
		feishuData.Card = card
		b.GetMessage().Format = message.FormatCard
	}
	return b
}

// Slack-specific convenience methods
func (b *UnifiedPlatformBuilder) ToChannel(channels ...string) *UnifiedPlatformBuilder {
	if slackData, ok := b.platformData.(*SlackPlatformData); ok {
		slackData.Channels = append(slackData.Channels, channels...)
	}
	return b
}

func (b *UnifiedPlatformBuilder) InThread(threadTs string) *UnifiedPlatformBuilder {
	if slackData, ok := b.platformData.(*SlackPlatformData); ok {
		slackData.ThreadTs = threadTs
	}
	return b
}

func (b *UnifiedPlatformBuilder) Broadcast() *UnifiedPlatformBuilder {
	if slackData, ok := b.platformData.(*SlackPlatformData); ok {
		slackData.Broadcast = true
	}
	return b
}

func (b *UnifiedPlatformBuilder) LinkNames() *UnifiedPlatformBuilder {
	if slackData, ok := b.platformData.(*SlackPlatformData); ok {
		slackData.LinkNames = true
	}
	return b
}

// Common webhook methods
func (b *UnifiedPlatformBuilder) WithWebhook(webhook string, secret ...string) *UnifiedPlatformBuilder {
	switch data := b.platformData.(type) {
	case *FeishuPlatformData:
		data.Webhook = webhook
		if len(secret) > 0 {
			data.Secret = secret[0]
		}
	case *SlackPlatformData:
		data.Webhook = webhook
	}
	return b
}