package api

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
)

// SendResult represents the result of a send operation (temporary)
type SendResult struct {
	MessageID string
	Success   bool
	Platform  string
	Target    message.Target
	SentAt    time.Time
	QueuedAt  time.Time
}

// DryRunResult represents the result of a dry run (temporary)
type DryRunResult struct {
	Valid     bool
	Message   message.Message
	Targets   []message.Target
	Timestamp time.Time
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
}

// BuilderInterface defines the common interface for all builders
type BuilderInterface interface {
	Title(string) BuilderInterface
	Body(string) BuilderInterface
	Priority(int) BuilderInterface
	Template(string) BuilderInterface
	Var(key string, value interface{}) BuilderInterface
	Vars(variables map[string]interface{}) BuilderInterface
	Meta(key, value string) BuilderInterface
	Send(ctx context.Context) (*sending.SendingResults, error)
	DryRun() (*DryRunResult, error)
}

// BaseBuilder provides common functionality for all builders
type BaseBuilder struct {
	client           *Client
	message          *message.Message
	targets          []message.Target
	targetDetector   *TargetDetector
	templateProcessor *TemplateProcessor
}

// NewBaseBuilder creates a new base builder
func NewBaseBuilder(client *Client) *BaseBuilder {
	return &BaseBuilder{
		client:            client,
		message:           message.NewMessage(),
		targets:           make([]message.Target, 0),
		targetDetector:    NewTargetDetector(),
		templateProcessor: NewTemplateProcessor(),
	}
}

// Title sets the message title
func (b *BaseBuilder) Title(title string) *BaseBuilder {
	b.message.Title = title
	return b
}

// Body sets the message body
func (b *BaseBuilder) Body(body string) *BaseBuilder {
	b.message.Body = body
	return b
}

// Priority sets the message priority
func (b *BaseBuilder) Priority(priority int) *BaseBuilder {
	b.message.Priority = priority
	return b
}

// Format sets the message format
func (b *BaseBuilder) Format(format message.Format) *BaseBuilder {
	b.message.Format = format
	return b
}

// Template sets the template name
func (b *BaseBuilder) Template(template string) *BaseBuilder {
	b.templateProcessor.SetTemplate(template)
	b.message.Template = template
	return b
}

// Var sets a template variable (single key-value)
func (b *BaseBuilder) Var(key string, value interface{}) *BaseBuilder {
	b.templateProcessor.Var(key, value)
	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}
	b.message.Variables[key] = value
	return b
}

// Vars sets multiple template variables
func (b *BaseBuilder) Vars(variables map[string]interface{}) *BaseBuilder {
	b.templateProcessor.Vars(variables)
	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}
	for k, v := range variables {
		b.message.Variables[k] = v
	}
	return b
}

// VarsFromKeyValue sets variables from alternating key-value pairs
func (b *BaseBuilder) VarsFromKeyValue(keyValues ...interface{}) *BaseBuilder {
	b.templateProcessor.VarsFromKeyValue(keyValues...)
	if len(keyValues)%2 != 0 {
		panic("VarsFromKeyValue requires an even number of arguments")
	}

	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}

	for i := 0; i < len(keyValues); i += 2 {
		key := fmt.Sprintf("%v", keyValues[i])
		value := keyValues[i+1]
		b.message.Variables[key] = value
	}
	return b
}

// Meta sets metadata
func (b *BaseBuilder) Meta(key, value string) *BaseBuilder {
	if b.message.Metadata == nil {
		b.message.Metadata = make(map[string]string)
	}
	b.message.Metadata[key] = value
	return b
}

// DelayBy sets a delay for message sending
func (b *BaseBuilder) DelayBy(delay time.Duration) *BaseBuilder {
	b.message.Delay = delay
	return b
}

// ScheduleAt schedules the message for a specific time
func (b *BaseBuilder) ScheduleAt(at time.Time) *BaseBuilder {
	// Store in metadata since ScheduledAt field doesn't exist in core message
	b.message.AddMetadata("scheduled_at", at.Format(time.RFC3339))
	return b
}

// Message sets the message directly (for compatibility)
func (b *BaseBuilder) Message(msg *message.Message) *BaseBuilder {
	b.message = msg
	return b
}

// Targets sets the targets directly (for compatibility)
func (b *BaseBuilder) Targets(targets ...message.Target) *BaseBuilder {
	b.targets = append(b.targets, targets...)
	return b
}

// DryRun enables dry run mode (validation only, no actual sending)
func (b *BaseBuilder) DryRun() (*DryRunResult, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	return &DryRunResult{
		Valid:     true,
		Message:   *b.message,
		Targets:   b.targets,
		Timestamp: time.Now(),
	}, nil
}

// Send sends the message (alias for Execute)
func (b *BaseBuilder) Send(ctx context.Context) (*sending.SendingResults, error) {
	return b.Execute(ctx)
}

// Execute sends the message
func (b *BaseBuilder) Execute(ctx context.Context) (*sending.SendingResults, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	// Generate message ID if not set
	if b.message.ID == "" {
		b.message.ID = generateMessageID()
	}

	// Convert message.Target to sending.Target
	sendingTargets := make([]sending.Target, len(b.targets))
	for i, t := range b.targets {
		sendingTargets[i] = sending.Target{
			Type:     sending.TargetType(t.Type),
			Value:    t.Value,
			Platform: t.Platform,
			Metadata: t.Metadata,
		}
	}

	return b.client.hub.Send(ctx, b.message, sendingTargets)
}

// validate checks if the message is ready to send
func (b *BaseBuilder) validate() error {
	if b.message.Title == "" && b.message.Body == "" {
		return &ValidationError{Field: "message", Message: "either title or body must be provided"}
	}

	if len(b.targets) == 0 {
		return &ValidationError{Field: "targets", Message: "at least one target must be specified"}
	}

	return nil
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// GetMessage returns the current message (for platform-specific builders)
func (b *BaseBuilder) GetMessage() *message.Message {
	return b.message
}

// GetClient returns the client (for platform-specific builders)
func (b *BaseBuilder) GetClient() *Client {
	return b.client
}

// AddTarget adds a target to the builder
func (b *BaseBuilder) AddTarget(target message.Target) *BaseBuilder {
	b.targets = append(b.targets, target)
	return b
}

// GetTargets returns the current targets
func (b *BaseBuilder) GetTargets() []message.Target {
	return b.targets
}

// To adds targets using smart detection
// Examples:
//   - To("user@example.com") -> email
//   - To("@john", "@jane") -> user mentions
//   - To("#alerts") -> channel
func (b *BaseBuilder) To(targets ...string) *BaseBuilder {
	detectedTargets := b.targetDetector.DetectTargets(targets...)
	b.targets = append(b.targets, detectedTargets...)
	return b
}

// ToEmail explicitly adds email targets
func (b *BaseBuilder) ToEmail(emails ...string) *BaseBuilder {
	for _, email := range emails {
		target := message.NewTarget(message.TargetTypeEmail, email, "email")
		b.targets = append(b.targets, target)
	}
	return b
}

// ToUser adds user targets (platform will be determined by context)
func (b *BaseBuilder) ToUser(users ...string) *BaseBuilder {
	for _, user := range users {
		target := message.NewTarget(message.TargetTypeUser, user, "")
		b.targets = append(b.targets, target)
	}
	return b
}

// ToGroup adds group targets
func (b *BaseBuilder) ToGroup(group string, platform string) *BaseBuilder {
	target := message.NewTarget(message.TargetTypeGroup, group, platform)
	b.targets = append(b.targets, target)
	return b
}

// ToChannel adds channel targets
func (b *BaseBuilder) ToChannel(channel string, platform string) *BaseBuilder {
	target := message.NewTarget(message.TargetTypeChannel, channel, platform)
	b.targets = append(b.targets, target)
	return b
}

// ToSlack adds Slack-specific targets
func (b *BaseBuilder) ToSlack(targets ...string) *BaseBuilder {
	slackDetector := NewSlackTargetDetector()
	detectedTargets := slackDetector.DetectTargets(targets...)
	b.targets = append(b.targets, detectedTargets...)
	return b
}

// ToFeishu adds Feishu-specific targets
func (b *BaseBuilder) ToFeishu(targets ...string) *BaseBuilder {
	feishuDetector := NewFeishuTargetDetector()
	detectedTargets := feishuDetector.DetectTargets(targets...)
	b.targets = append(b.targets, detectedTargets...)
	return b
}

// Via specifies which platforms to use for previously added targets
func (b *BaseBuilder) Via(platforms ...string) *BaseBuilder {
	b.targets = b.targetDetector.SetPlatformForTargets(b.targets, platforms...)
	return b
}

// GetTargetDetector returns the target detector (for advanced usage)
func (b *BaseBuilder) GetTargetDetector() *TargetDetector {
	return b.targetDetector
}

// GetTemplateProcessor returns the template processor (for advanced usage)
func (b *BaseBuilder) GetTemplateProcessor() *TemplateProcessor {
	return b.templateProcessor
}

// VarsFromStruct sets variables from struct fields using reflection
func (b *BaseBuilder) VarsFromStruct(data interface{}) *BaseBuilder {
	b.templateProcessor.VarsFromStruct(data)

	// Also update message variables for backward compatibility
	variables := b.templateProcessor.GetContext().GetVariables()
	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}
	for k, v := range variables {
		b.message.Variables[k] = v
	}
	return b
}
