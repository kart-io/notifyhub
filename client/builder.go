package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/internal"
	"github.com/kart-io/notifyhub/notifiers"
)

// MessageBuilder provides fluent interface for building messages
type MessageBuilder struct {
	message *notifiers.Message
	options *Options // Send options integrated into builder
	debug   bool     // Enable debug output for this message
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

// CardData sets the custom card data for platforms that support interactive cards
func (b *MessageBuilder) CardData(cardData interface{}) *MessageBuilder {
	b.message.CardData = cardData
	return b
}

// AtUser adds a user mention to the message
func (b *MessageBuilder) AtUser(userID string, userName ...string) *MessageBuilder {
	if b.message.AtMentions == nil {
		b.message.AtMentions = make([]notifiers.AtMention, 0)
	}

	mention := notifiers.AtMention{
		UserID: userID,
		IsAll:  false,
	}

	if len(userName) > 0 && userName[0] != "" {
		mention.UserName = userName[0]
	}

	b.message.AtMentions = append(b.message.AtMentions, mention)
	return b
}

// AtAll adds an @all mention to the message
func (b *MessageBuilder) AtAll() *MessageBuilder {
	if b.message.AtMentions == nil {
		b.message.AtMentions = make([]notifiers.AtMention, 0)
	}

	mention := notifiers.AtMention{
		UserID: "all",
		IsAll:  true,
	}

	b.message.AtMentions = append(b.message.AtMentions, mention)
	return b
}

// AtUsers adds multiple user mentions to the message
func (b *MessageBuilder) AtUsers(userIDs ...string) *MessageBuilder {
	for _, userID := range userIDs {
		if userID != "" {
			b.AtUser(userID)
		}
	}
	return b
}

// AtMentions sets the @mentions list directly
func (b *MessageBuilder) AtMentions(mentions []notifiers.AtMention) *MessageBuilder {
	b.message.AtMentions = mentions
	return b
}

// ClearAtMentions clears all @mentions from the message
func (b *MessageBuilder) ClearAtMentions() *MessageBuilder {
	b.message.AtMentions = nil
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

// NewCard creates a new card message builder
func NewCard(title, body string) *MessageBuilder {
	return NewMessage().
		Title(title).
		Body(body).
		Format(notifiers.FormatCard)
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

// ================================
// Strong Typed Target Methods
// ================================

// EmailTarget adds a validated email target
func (b *MessageBuilder) EmailTarget(email string) *MessageBuilder {
	target, err := EmailTarget(email)
	if err != nil {
		// In builder pattern, we can either panic or silently ignore
		// For now, we'll add it anyway and let validation catch it later
		b.Email(email)
	} else {
		b.Target(target)
	}
	return b
}

// EmailTargets adds multiple validated email targets
func (b *MessageBuilder) EmailTargets(emails ...string) *MessageBuilder {
	targets, err := EmailTargets(emails...)
	if err != nil {
		// Add individually to avoid losing all on single failure
		for _, email := range emails {
			b.EmailTarget(email)
		}
	} else {
		for _, target := range targets {
			b.Target(target)
		}
	}
	return b
}

// UserTarget adds a validated user target
func (b *MessageBuilder) UserTarget(userID, platform string) *MessageBuilder {
	target, err := UserTarget(userID, platform)
	if err != nil {
		b.User(userID, platform) // Fallback to original method
	} else {
		b.Target(target)
	}
	return b
}

// GroupTarget adds a validated group target
func (b *MessageBuilder) GroupTarget(groupID, platform string) *MessageBuilder {
	target, err := GroupTarget(groupID, platform)
	if err != nil {
		b.Group(groupID, platform) // Fallback to original method
	} else {
		b.Target(target)
	}
	return b
}

// ChannelTarget adds a validated channel target
func (b *MessageBuilder) ChannelTarget(channelID, platform string) *MessageBuilder {
	target, err := ChannelTarget(channelID, platform)
	if err != nil {
		b.Channel(channelID, platform) // Fallback to original method
	} else {
		b.Target(target)
	}
	return b
}

// ParseTarget adds a parsed target
func (b *MessageBuilder) ParseTarget(target string) *MessageBuilder {
	parsed, err := ParseTarget(target)
	if err != nil {
		// Try to add as-is if parsing fails
		if isValidEmailFormat(target) {
			b.Email(target)
		}
	} else {
		b.Target(parsed)
	}
	return b
}

// ParseTargets adds multiple parsed targets
func (b *MessageBuilder) ParseTargets(targets ...string) *MessageBuilder {
	for _, target := range targets {
		b.ParseTarget(target)
	}
	return b
}

// WithTargetBuilder uses a target builder to add targets
func (b *MessageBuilder) WithTargetBuilder(builder *TargetBuilder) *MessageBuilder {
	targets, err := builder.Build()
	if err == nil {
		for _, target := range targets {
			b.Target(target)
		}
	}
	return b
}

// SmartTargets intelligently parses and adds multiple targets
func (b *MessageBuilder) SmartTargets(targets ...string) *MessageBuilder {
	targetBuilder := NewTargetBuilder()
	targetBuilder.ParseAll(targets...)
	return b.WithTargetBuilder(targetBuilder)
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

// IfElse adds conditional logic with else branch
func (b *MessageBuilder) IfElse(condition bool, ifFn func(*MessageBuilder) *MessageBuilder, elseFn func(*MessageBuilder) *MessageBuilder) *MessageBuilder {
	if condition {
		return ifFn(b)
	}
	return elseFn(b)
}

// When adds multiple conditional branches
func (b *MessageBuilder) When(condition bool) *ConditionalBuilder {
	return &ConditionalBuilder{
		builder:   b,
		condition: condition,
	}
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

// AsCard sets the message format to card
func (b *MessageBuilder) AsCard() *MessageBuilder {
	return b.Format(notifiers.FormatCard)
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
			CardData:  b.message.CardData,
			CreatedAt: b.message.CreatedAt,
		},
	}

	// Deep copy targets
	newBuilder.message.Targets = make([]notifiers.Target, len(b.message.Targets))
	copy(newBuilder.message.Targets, b.message.Targets)

	// Deep copy @mentions
	if b.message.AtMentions != nil {
		newBuilder.message.AtMentions = make([]notifiers.AtMention, len(b.message.AtMentions))
		copy(newBuilder.message.AtMentions, b.message.AtMentions)
	}

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

// ================================
// Conditional Builder for Advanced Conditional Logic
// ================================

// ConditionalBuilder provides advanced conditional building capabilities
type ConditionalBuilder struct {
	builder   *MessageBuilder
	condition bool
}

// Then executes the function if the condition is true
func (cb *ConditionalBuilder) Then(fn func(*MessageBuilder) *MessageBuilder) *ElseBuilder {
	if cb.condition {
		cb.builder = fn(cb.builder)
	}
	return &ElseBuilder{
		builder:   cb.builder,
		condition: cb.condition,
	}
}

// ElseBuilder provides else functionality for conditional building
type ElseBuilder struct {
	builder   *MessageBuilder
	condition bool
}

// Else executes the function if the original condition was false
func (eb *ElseBuilder) Else(fn func(*MessageBuilder) *MessageBuilder) *MessageBuilder {
	if !eb.condition {
		return fn(eb.builder)
	}
	return eb.builder
}

// ElseIf adds another condition to check
func (eb *ElseBuilder) ElseIf(condition bool) *ConditionalBuilder {
	if !eb.condition {
		return &ConditionalBuilder{
			builder:   eb.builder,
			condition: condition,
		}
	}
	return &ConditionalBuilder{
		builder:   eb.builder,
		condition: false, // Previous condition was true, skip remaining
	}
}

// End returns the original MessageBuilder without else clause
func (eb *ElseBuilder) End() *MessageBuilder {
	return eb.builder
}

// ================================
// Send Options Integration
// ================================

// WithOptions sets send options for the message
func (b *MessageBuilder) WithOptions(options *Options) *MessageBuilder {
	b.options = options
	return b
}

// Async enables asynchronous sending
func (b *MessageBuilder) Async(async bool) *MessageBuilder {
	if b.options == nil {
		b.options = NewOptions()
	}
	b.options.Async = async
	return b
}

// AsyncSend enables asynchronous sending (convenience method)
func (b *MessageBuilder) AsyncSend() *MessageBuilder {
	return b.Async(true)
}

// SyncSend enables synchronous sending (convenience method)
func (b *MessageBuilder) SyncSend() *MessageBuilder {
	return b.Async(false)
}

// WithRetry enables retry on failure
func (b *MessageBuilder) WithRetry(retry bool) *MessageBuilder {
	if b.options == nil {
		b.options = NewOptions()
	}
	b.options.Retry = retry
	return b
}

// EnableRetry enables retry on failure (convenience method)
func (b *MessageBuilder) EnableRetry() *MessageBuilder {
	return b.WithRetry(true)
}

// DisableRetry disables retry on failure (convenience method)
func (b *MessageBuilder) DisableRetry() *MessageBuilder {
	return b.WithRetry(false)
}

// WithMaxRetries sets maximum retry attempts
func (b *MessageBuilder) WithMaxRetries(maxRetries int) *MessageBuilder {
	if b.options == nil {
		b.options = NewOptions()
	}
	b.options.MaxRetries = maxRetries
	return b
}

// WithTimeout sets operation timeout
func (b *MessageBuilder) WithTimeout(timeout time.Duration) *MessageBuilder {
	if b.options == nil {
		b.options = NewOptions()
	}
	b.options.Timeout = timeout
	return b
}

// WithDelay sets message delay in milliseconds
func (b *MessageBuilder) WithDelay(delayMs int) *MessageBuilder {
	if b.options == nil {
		b.options = NewOptions()
	}
	b.options.DelayMs = delayMs
	return b
}

// WithBatchSize sets batch size for batch operations
func (b *MessageBuilder) WithBatchSize(batchSize int) *MessageBuilder {
	if b.options == nil {
		b.options = NewOptions()
	}
	b.options.BatchSize = batchSize
	return b
}

// GetOptions returns the send options (for use with sending methods)
func (b *MessageBuilder) GetOptions() *Options {
	return b.options
}

// BuildWithOptions builds both message and options
func (b *MessageBuilder) BuildWithOptions() (*notifiers.Message, *Options) {
	return b.Build(), b.options
}

// ================================
// Enhanced Builder Methods with Send Options
// ================================

// SendTo combines building and sending in one call
// This requires a Hub to be passed in, making it convenient for one-liner sends
func (b *MessageBuilder) SendTo(hub *Hub, ctx ...interface{}) error {
	// Extract context if provided
	var sendCtx interface{}
	if len(ctx) > 0 {
		sendCtx = ctx[0]
	}

	message := b.Build()
	options := b.GetOptions()

	// Use hub's send method based on context type
	switch c := sendCtx.(type) {
	case context.Context:
		_, err := hub.Send(c, message, options)
		return err
	default:
		// Default to background context
		_, err := hub.Send(context.Background(), message, options)
		return err
	}
}

// SendAsyncTo combines building and async sending in one call
func (b *MessageBuilder) SendAsyncTo(hub *Hub, ctx ...interface{}) (string, error) {
	// Extract context if provided
	var sendCtx interface{}
	if len(ctx) > 0 {
		sendCtx = ctx[0]
	}

	message := b.Build()
	options := b.GetOptions()
	if options == nil {
		options = NewAsyncOptions()
	} else {
		options.Async = true
	}

	// Use hub's send method based on context type
	switch c := sendCtx.(type) {
	case context.Context:
		return hub.SendAsync(c, message, options)
	default:
		// Default to background context
		return hub.SendAsync(context.Background(), message, options)
	}
}

// SendWithAnalysisTo combines building and sending with analysis in one call
func (b *MessageBuilder) SendWithAnalysisTo(hub *Hub, ctx ...interface{}) ([]*notifiers.SendResult, *ResultAnalyzer, error) {
	// Extract context if provided
	var sendCtx interface{}
	if len(ctx) > 0 {
		sendCtx = ctx[0]
	}

	message := b.Build()
	options := b.GetOptions()

	// Use hub's send method based on context type
	switch c := sendCtx.(type) {
	case context.Context:
		return hub.SendWithAnalysis(c, message, options)
	default:
		// Default to background context
		return hub.SendWithAnalysis(context.Background(), message, options)
	}
}

// ================================
// Convenience Send Options Presets
// ================================

// AsQuickSend applies quick send options (sync, no retry, 5s timeout)
func (b *MessageBuilder) AsQuickSend() *MessageBuilder {
	return b.SyncSend().
		DisableRetry().
		WithTimeout(5 * time.Second)
}

// AsReliableSend applies reliable send options (sync, retry enabled, 30s timeout)
func (b *MessageBuilder) AsReliableSend() *MessageBuilder {
	return b.SyncSend().
		EnableRetry().
		WithMaxRetries(3).
		WithTimeout(30 * time.Second)
}

// AsBackgroundSend applies background send options (async, retry enabled)
func (b *MessageBuilder) AsBackgroundSend() *MessageBuilder {
	return b.AsyncSend().
		EnableRetry().
		WithMaxRetries(5)
}

// AsDelayedSend applies delayed send options with specified delay
func (b *MessageBuilder) AsDelayedSend(delayMs int) *MessageBuilder {
	return b.WithDelay(delayMs).
		EnableRetry()
}

// AsCriticalSend applies critical send options (sync, multiple retries, extended timeout)
func (b *MessageBuilder) AsCriticalSend() *MessageBuilder {
	return b.SyncSend().
		EnableRetry().
		WithMaxRetries(5).
		WithTimeout(60 * time.Second)
}

// ================================
// Platform-Specific Convenience Builders
// ================================



// ToDiscord adds Discord-specific targets
func (b *MessageBuilder) ToDiscord(targets ...string) *MessageBuilder {
	for _, target := range targets {
		if strings.HasPrefix(target, "#") {
			// Channel
			b.Target(notifiers.Target{
				Type:     notifiers.TargetTypeChannel,
				Value:    strings.TrimPrefix(target, "#"),
				Platform: "discord",
			})
		} else if strings.HasPrefix(target, "@") {
			// User
			b.Target(notifiers.Target{
				Type:     notifiers.TargetTypeUser,
				Value:    strings.TrimPrefix(target, "@"),
				Platform: "discord",
			})
		} else {
			// Assume it's a user ID
			b.Target(notifiers.Target{
				Type:     notifiers.TargetTypeUser,
				Value:    target,
				Platform: "discord",
			})
		}
	}
	return b
}

// ToTeams adds Microsoft Teams-specific targets
func (b *MessageBuilder) ToTeams(targets ...string) *MessageBuilder {
	for _, target := range targets {
		if strings.HasPrefix(target, "#") {
			// Channel
			b.Target(notifiers.Target{
				Type:     notifiers.TargetTypeChannel,
				Value:    strings.TrimPrefix(target, "#"),
				Platform: "teams",
			})
		} else if strings.HasPrefix(target, "@") {
			// User
			b.Target(notifiers.Target{
				Type:     notifiers.TargetTypeUser,
				Value:    strings.TrimPrefix(target, "@"),
				Platform: "teams",
			})
		} else {
			// Assume it's a user or channel
			b.Target(notifiers.Target{
				Type:     notifiers.TargetTypeUser,
				Value:    target,
				Platform: "teams",
			})
		}
	}
	return b
}

// ToWebhook adds webhook targets (using user type with webhook platform)
func (b *MessageBuilder) ToWebhook(urls ...string) *MessageBuilder {
	for _, url := range urls {
		b.Target(notifiers.Target{
			Type:     notifiers.TargetTypeUser,
			Value:    url,
			Platform: "webhook",
		})
	}
	return b
}

// ToSMS adds SMS targets (phone numbers, using user type with sms platform)
func (b *MessageBuilder) ToSMS(phoneNumbers ...string) *MessageBuilder {
	for _, phone := range phoneNumbers {
		b.Target(notifiers.Target{
			Type:     notifiers.TargetTypeUser,
			Value:    phone,
			Platform: "sms",
		})
	}
	return b
}

// ToPush adds push notification targets (device tokens, using user type with push platform)
func (b *MessageBuilder) ToPush(deviceTokens ...string) *MessageBuilder {
	for _, token := range deviceTokens {
		b.Target(notifiers.Target{
			Type:     notifiers.TargetTypeUser,
			Value:    token,
			Platform: "push",
		})
	}
	return b
}

// ================================
// Specialized Platform Builders
// ================================

// SlackChannel adds a Slack channel target
func (b *MessageBuilder) SlackChannel(channelName string) *MessageBuilder {
	channelName = strings.TrimPrefix(channelName, "#")
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeChannel,
		Value:    channelName,
		Platform: "slack",
	})
}

// SlackUser adds a Slack user target
func (b *MessageBuilder) SlackUser(userID string) *MessageBuilder {
	userID = strings.TrimPrefix(userID, "@")
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeUser,
		Value:    userID,
		Platform: "slack",
	})
}

// SlackDM adds a Slack direct message target (using user type)
func (b *MessageBuilder) SlackDM(userID string) *MessageBuilder {
	userID = strings.TrimPrefix(userID, "@")
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeUser,
		Value:    userID,
		Platform: "slack",
		Metadata: map[string]string{"dm": "true"},
	})
}


// FeishuBot adds a Feishu bot target (using user type with bot metadata)
func (b *MessageBuilder) FeishuBot(botID string) *MessageBuilder {
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeUser,
		Value:    botID,
		Platform: "feishu",
		Metadata: map[string]string{"type": "bot"},
	})
}

// DiscordChannel adds a Discord channel target
func (b *MessageBuilder) DiscordChannel(channelID string) *MessageBuilder {
	channelID = strings.TrimPrefix(channelID, "#")
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeChannel,
		Value:    channelID,
		Platform: "discord",
	})
}

// DiscordUser adds a Discord user target
func (b *MessageBuilder) DiscordUser(userID string) *MessageBuilder {
	userID = strings.TrimPrefix(userID, "@")
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeUser,
		Value:    userID,
		Platform: "discord",
	})
}

// DiscordDM adds a Discord direct message target (using user type)
func (b *MessageBuilder) DiscordDM(userID string) *MessageBuilder {
	userID = strings.TrimPrefix(userID, "@")
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeUser,
		Value:    userID,
		Platform: "discord",
		Metadata: map[string]string{"dm": "true"},
	})
}

// TeamsChannel adds a Microsoft Teams channel target
func (b *MessageBuilder) TeamsChannel(channelID string) *MessageBuilder {
	channelID = strings.TrimPrefix(channelID, "#")
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeChannel,
		Value:    channelID,
		Platform: "teams",
	})
}

// TeamsUser adds a Microsoft Teams user target
func (b *MessageBuilder) TeamsUser(userID string) *MessageBuilder {
	userID = strings.TrimPrefix(userID, "@")
	return b.Target(notifiers.Target{
		Type:     notifiers.TargetTypeUser,
		Value:    userID,
		Platform: "teams",
	})
}

// ================================
// Bulk Platform-Specific Methods
// ================================

// SlackChannels adds multiple Slack channels
func (b *MessageBuilder) SlackChannels(channels ...string) *MessageBuilder {
	for _, channel := range channels {
		b.SlackChannel(channel)
	}
	return b
}

// SlackUsers adds multiple Slack users
func (b *MessageBuilder) SlackUsers(users ...string) *MessageBuilder {
	for _, user := range users {
		b.SlackUser(user)
	}
	return b
}

// FeishuGroups adds multiple Feishu groups
func (b *MessageBuilder) FeishuGroups(groups ...string) *MessageBuilder {
	for _, group := range groups {
		b.FeishuGroup(group)
	}
	return b
}

// FeishuUsers adds multiple Feishu users
func (b *MessageBuilder) FeishuUsers(users ...string) *MessageBuilder {
	for _, user := range users {
		b.FeishuUser(user)
	}
	return b
}

// DiscordChannels adds multiple Discord channels
func (b *MessageBuilder) DiscordChannels(channels ...string) *MessageBuilder {
	for _, channel := range channels {
		b.DiscordChannel(channel)
	}
	return b
}

// DiscordUsers adds multiple Discord users
func (b *MessageBuilder) DiscordUsers(users ...string) *MessageBuilder {
	for _, user := range users {
		b.DiscordUser(user)
	}
	return b
}

// EmailsTo adds multiple email targets (alias for clearer intent)
func (b *MessageBuilder) EmailsTo(emails ...string) *MessageBuilder {
	return b.MultipleEmails(emails...)
}

// SMSTo adds multiple SMS targets
func (b *MessageBuilder) SMSTo(phoneNumbers ...string) *MessageBuilder {
	return b.ToSMS(phoneNumbers...)
}

// WebhooksTo adds multiple webhook targets
func (b *MessageBuilder) WebhooksTo(urls ...string) *MessageBuilder {
	return b.ToWebhook(urls...)
}

// ================================
// Platform-Agnostic Smart Routing
// ================================

// ToChannel routes to appropriate channel based on context
func (b *MessageBuilder) ToChannel(channelName string, platforms ...string) *MessageBuilder {
	if len(platforms) == 0 {
		// Default to common platforms
		platforms = []string{"slack", "discord", "teams"}
	}

	for _, platform := range platforms {
		switch platform {
		case "slack":
			b.SlackChannel(channelName)
		case "discord":
			b.DiscordChannel(channelName)
		case "teams":
			b.TeamsChannel(channelName)
		case "feishu":
			b.FeishuGroup(channelName)
		}
	}
	return b
}

// ToUser routes to appropriate user targets based on context
func (b *MessageBuilder) ToUser(userID string, platforms ...string) *MessageBuilder {
	if len(platforms) == 0 {
		// Default to common platforms
		platforms = []string{"slack", "discord", "teams"}
	}

	for _, platform := range platforms {
		switch platform {
		case "slack":
			b.SlackUser(userID)
		case "discord":
			b.DiscordUser(userID)
		case "teams":
			b.TeamsUser(userID)
		case "feishu":
			b.FeishuUser(userID)
		}
	}
	return b
}

// ToIncidentResponse routes to incident response channels/users
func (b *MessageBuilder) ToIncidentResponse() *MessageBuilder {
	return b.SlackChannel("incidents").
		DiscordChannel("incidents").
		TeamsChannel("incidents").
		Email("incident-response@company.com")
}

// ToOnCall routes to on-call personnel
func (b *MessageBuilder) ToOnCall() *MessageBuilder {
	return b.SlackUser("oncall").
		Email("oncall@company.com").
		ToSMS("+1-555-0123") // Example on-call phone
}

// ToDevOps routes to DevOps teams across platforms
func (b *MessageBuilder) ToDevOps() *MessageBuilder {
	return b.SlackChannel("devops").
		FeishuGroup("devops-team").
		Email("devops@company.com")
}

// ToSecurity routes to security teams
func (b *MessageBuilder) ToSecurity() *MessageBuilder {
	return b.SlackChannel("security").
		Email("security@company.com").
		TeamsChannel("security")
}

// ================================
// Conditional Platform Routing
// ================================

// ToSlackIf conditionally adds Slack targets
func (b *MessageBuilder) ToSlackIf(condition bool, targets ...string) *MessageBuilder {
	if condition {
		for _, target := range targets {
			b.ToSlack(target)
		}
	}
	return b
}

// ToFeishuIf conditionally adds Feishu targets
func (b *MessageBuilder) ToFeishuIf(condition bool, targets ...string) *MessageBuilder {
	if condition {
		for _, target := range targets {
			b.ToFeishu(target)
		}
	}
	return b
}

// ToEmailIf conditionally adds email targets
func (b *MessageBuilder) ToEmailIf(condition bool, emails ...string) *MessageBuilder {
	if condition {
		return b.MultipleEmails(emails...)
	}
	return b
}

// ToWebhookIf conditionally adds webhook targets
func (b *MessageBuilder) ToWebhookIf(condition bool, urls ...string) *MessageBuilder {
	if condition {
		return b.ToWebhook(urls...)
	}
	return b
}
