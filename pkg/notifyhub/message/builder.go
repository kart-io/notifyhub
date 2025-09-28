// Package message provides message builder functionality for NotifyHub
package message

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/errors"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Validation constants
const (
	MaxTitleLength   = 200
	MaxBodyLength    = 4096
	MinTargetCount   = 1
	MaxTargetCount   = 100
	MaxKeywordLength = 50
)

// MessageBuilder provides fluent API for building messages with validation
type MessageBuilder struct {
	message *Message
	errors  []error
}

// NewMessage creates a new message builder with optional title
func NewMessage(title ...string) *MessageBuilder {
	msg := &Message{
		ID:           generateMessageID(),
		Format:       FormatText,
		Priority:     PriorityNormal,
		CreatedAt:    time.Now(),
		Targets:      make([]target.Target, 0),
		Metadata:     make(map[string]interface{}),
		Variables:    make(map[string]interface{}),
		PlatformData: make(map[string]interface{}),
	}

	if len(title) > 0 {
		msg.Title = title[0]
	}

	return &MessageBuilder{
		message: msg,
		errors:  make([]error, 0),
	}
}

// NewAlert creates a new alert message (high priority by default)
func NewAlert(title string) *MessageBuilder {
	return NewMessage(title).WithPriority(PriorityHigh)
}

// NewUrgent creates a new urgent message (urgent priority by default)
func NewUrgent(title string) *MessageBuilder {
	return NewMessage(title).WithPriority(PriorityUrgent)
}

// WithTitle sets the message title with validation
func (b *MessageBuilder) WithTitle(title string) *MessageBuilder {
	if err := b.validateTitle(title); err != nil {
		b.addError(err)
		return b
	}
	b.message.Title = title
	return b
}

// WithBody sets the message body with validation
func (b *MessageBuilder) WithBody(body string) *MessageBuilder {
	if err := b.validateBody(body); err != nil {
		b.addError(err)
		return b
	}
	b.message.Body = body
	return b
}

// WithText is an alias for WithBody for text messages
func (b *MessageBuilder) WithText(text string) *MessageBuilder {
	return b.WithBody(text)
}

// WithFormat sets the message format with validation
func (b *MessageBuilder) WithFormat(format Format) *MessageBuilder {
	if err := b.validateFormat(format); err != nil {
		b.addError(err)
		return b
	}
	b.message.Format = format
	return b
}

// WithMarkdown sets the message format to markdown and sets the body with validation
func (b *MessageBuilder) WithMarkdown(content string) *MessageBuilder {
	if err := b.validateBody(content); err != nil {
		b.addError(err)
		return b
	}
	b.message.Format = FormatMarkdown
	b.message.Body = content
	return b
}

// WithHTML sets the message format to HTML and sets the body with validation
func (b *MessageBuilder) WithHTML(content string) *MessageBuilder {
	if err := b.validateBody(content); err != nil {
		b.addError(err)
		return b
	}
	b.message.Format = FormatHTML
	b.message.Body = content
	return b
}

// WithPriority sets the message priority with validation
func (b *MessageBuilder) WithPriority(priority Priority) *MessageBuilder {
	if err := b.validatePriority(priority); err != nil {
		b.addError(err)
		return b
	}
	b.message.Priority = priority
	return b
}

// WithMetadata adds metadata to the message with validation
func (b *MessageBuilder) WithMetadata(key string, value interface{}) *MessageBuilder {
	if err := b.validateMetadataKey(key); err != nil {
		b.addError(err)
		return b
	}
	if b.message.Metadata == nil {
		b.message.Metadata = make(map[string]interface{})
	}
	b.message.Metadata[key] = value
	return b
}

// WithVariable adds a template variable with validation
func (b *MessageBuilder) WithVariable(key string, value interface{}) *MessageBuilder {
	if err := b.validateVariableKey(key); err != nil {
		b.addError(err)
		return b
	}
	if b.message.Variables == nil {
		b.message.Variables = make(map[string]interface{})
	}
	b.message.Variables[key] = value
	return b
}

// WithVariables adds multiple template variables with validation
func (b *MessageBuilder) WithVariables(variables map[string]interface{}) *MessageBuilder {
	if variables == nil {
		b.addError(errors.NewValidationError(errors.ErrInvalidType, "variables", "variables map cannot be nil"))
		return b
	}
	for k, v := range variables {
		if err := b.validateVariableKey(k); err != nil {
			b.addError(err)
			continue
		}
		if b.message.Variables == nil {
			b.message.Variables = make(map[string]interface{})
		}
		b.message.Variables[k] = v
	}
	return b
}

// WithPlatformData adds platform-specific data with validation
func (b *MessageBuilder) WithPlatformData(data map[string]interface{}) *MessageBuilder {
	if data == nil {
		b.addError(errors.NewValidationError(errors.ErrInvalidType, "platform_data", "platform data map cannot be nil"))
		return b
	}
	for k, v := range data {
		if err := b.validatePlatformDataKey(k); err != nil {
			b.addError(err)
			continue
		}
		if b.message.PlatformData == nil {
			b.message.PlatformData = make(map[string]interface{})
		}
		b.message.PlatformData[k] = v
	}
	return b
}

// ScheduleAt schedules the message for later delivery with validation
func (b *MessageBuilder) ScheduleAt(when time.Time) *MessageBuilder {
	if err := b.validateScheduleTime(when); err != nil {
		b.addError(err)
		return b
	}
	b.message.ScheduledAt = &when
	return b
}

// ScheduleIn schedules the message for delivery after a duration with validation
func (b *MessageBuilder) ScheduleIn(duration time.Duration) *MessageBuilder {
	if err := b.validateScheduleDuration(duration); err != nil {
		b.addError(err)
		return b
	}
	when := time.Now().Add(duration)
	b.message.ScheduledAt = &when
	return b
}

// ToTarget adds a generic target with validation
func (b *MessageBuilder) ToTarget(t target.Target) *MessageBuilder {
	if err := b.validateTarget(t); err != nil {
		b.addError(err)
		return b
	}
	if err := b.validateTargetCount(len(b.message.Targets) + 1); err != nil {
		b.addError(err)
		return b
	}
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToTargets adds multiple targets with validation
func (b *MessageBuilder) ToTargets(targets ...target.Target) *MessageBuilder {
	newCount := len(b.message.Targets) + len(targets)
	if err := b.validateTargetCount(newCount); err != nil {
		b.addError(err)
		return b
	}
	for _, t := range targets {
		if err := b.validateTarget(t); err != nil {
			b.addError(err)
			continue
		}
		b.message.Targets = append(b.message.Targets, t)
	}
	return b
}

// ToCustomTarget adds a custom target with type, value, and platform with validation
func (b *MessageBuilder) ToCustomTarget(targetType, value, platform string) *MessageBuilder {
	if err := b.validateTargetCount(len(b.message.Targets) + 1); err != nil {
		b.addError(err)
		return b
	}
	if err := b.validateTargetParams(targetType, value, platform); err != nil {
		b.addError(err)
		return b
	}
	t := target.NewTarget(targetType, value, platform)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToEmail adds an email target with validation
func (b *MessageBuilder) ToEmail(email string) *MessageBuilder {
	if err := b.validateTargetCount(len(b.message.Targets) + 1); err != nil {
		b.addError(err)
		return b
	}
	if err := b.validateEmailAddress(email); err != nil {
		b.addError(err)
		return b
	}
	t := target.NewEmailTarget(email)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToPhone adds a phone/SMS target with validation
func (b *MessageBuilder) ToPhone(phone string) *MessageBuilder {
	if err := b.validateTargetCount(len(b.message.Targets) + 1); err != nil {
		b.addError(err)
		return b
	}
	if err := b.validatePhoneNumber(phone); err != nil {
		b.addError(err)
		return b
	}
	t := target.NewPhoneTarget(phone)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToFeishu adds a Feishu target (can be user, group, or webhook) with validation
func (b *MessageBuilder) ToFeishu(identifier string) *MessageBuilder {
	if err := b.validateTargetCount(len(b.message.Targets) + 1); err != nil {
		b.addError(err)
		return b
	}
	if err := b.validateFeishuIdentifier(identifier); err != nil {
		b.addError(err)
		return b
	}
	t := target.AutoDetectTarget(identifier)
	if t.Platform == "" {
		t.Platform = "feishu"
	}
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToFeishuUser adds a specific Feishu user target with validation
func (b *MessageBuilder) ToFeishuUser(userID string) *MessageBuilder {
	if err := b.validateTargetCount(len(b.message.Targets) + 1); err != nil {
		b.addError(err)
		return b
	}
	if err := b.validateFeishuUserID(userID); err != nil {
		b.addError(err)
		return b
	}
	t := target.NewFeishuUserTarget(userID)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToFeishuGroup adds a specific Feishu group target with validation
func (b *MessageBuilder) ToFeishuGroup(groupID string) *MessageBuilder {
	if err := b.validateTargetCount(len(b.message.Targets) + 1); err != nil {
		b.addError(err)
		return b
	}
	if err := b.validateFeishuGroupID(groupID); err != nil {
		b.addError(err)
		return b
	}
	t := target.NewFeishuGroupTarget(groupID)
	b.message.Targets = append(b.message.Targets, t)
	return b
}

// ToWebhook adds a webhook target with validation
func (b *MessageBuilder) ToWebhook(url string) *MessageBuilder {
	if err := b.validateTargetCount(len(b.message.Targets) + 1); err != nil {
		b.addError(err)
		return b
	}
	if err := b.validateWebhookURL(url); err != nil {
		b.addError(err)
		return b
	}
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

// Build returns the constructed message after final validation
func (b *MessageBuilder) Build() (*Message, error) {
	// Check for builder errors first
	if len(b.errors) > 0 {
		return nil, b.combineErrors()
	}

	// Perform final comprehensive validation
	if err := b.validateFinal(); err != nil {
		return nil, err
	}

	// Generate ID if not set
	if b.message.ID == "" {
		b.message.ID = generateMessageID()
	}

	// Set created timestamp if not set
	if b.message.CreatedAt.IsZero() {
		b.message.CreatedAt = time.Now()
	}

	return b.message, nil
}

// BuildUnsafe returns the message without validation (for backward compatibility)
func (b *MessageBuilder) BuildUnsafe() *Message {
	// Generate ID if not set
	if b.message.ID == "" {
		b.message.ID = generateMessageID()
	}

	return b.message
}

// Validate checks if the current builder state is valid
func (b *MessageBuilder) Validate() error {
	if len(b.errors) > 0 {
		return b.combineErrors()
	}
	return b.validateFinal()
}

// HasErrors returns true if there are validation errors
func (b *MessageBuilder) HasErrors() bool {
	return len(b.errors) > 0
}

// GetErrors returns all accumulated validation errors
func (b *MessageBuilder) GetErrors() []error {
	return b.errors[:]
}

// ClearErrors clears all accumulated validation errors
func (b *MessageBuilder) ClearErrors() *MessageBuilder {
	b.errors = make([]error, 0)
	return b
}

// Validation helper methods

// addError adds a validation error to the builder
func (b *MessageBuilder) addError(err error) {
	b.errors = append(b.errors, err)
}

// combineErrors combines all errors into a single validation error
func (b *MessageBuilder) combineErrors() error {
	if len(b.errors) == 0 {
		return nil
	}
	if len(b.errors) == 1 {
		return b.errors[0]
	}

	var messages []string
	for _, err := range b.errors {
		messages = append(messages, err.Error())
	}

	return errors.NewValidationError(
		errors.ErrValidationFailed,
		"message_builder",
		fmt.Sprintf("multiple validation errors: %s", strings.Join(messages, "; ")),
	)
}

// validateFinal performs comprehensive final validation
func (b *MessageBuilder) validateFinal() error {
	// Validate required fields
	if strings.TrimSpace(b.message.Title) == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "title", "message title is required")
	}

	if strings.TrimSpace(b.message.Body) == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "body", "message body is required")
	}

	if len(b.message.Targets) == 0 {
		return errors.NewValidationError(errors.ErrMissingRequired, "targets", "at least one target is required")
	}

	// Validate field constraints
	if err := b.validateTitle(b.message.Title); err != nil {
		return err
	}

	if err := b.validateBody(b.message.Body); err != nil {
		return err
	}

	if err := b.validateFormat(b.message.Format); err != nil {
		return err
	}

	if err := b.validatePriority(b.message.Priority); err != nil {
		return err
	}

	if err := b.validateTargetCount(len(b.message.Targets)); err != nil {
		return err
	}

	// Validate all targets
	for i, target := range b.message.Targets {
		if err := b.validateTarget(target); err != nil {
			return errors.NewValidationError(
				errors.ErrInvalidTarget,
				fmt.Sprintf("targets[%d]", i),
				fmt.Sprintf("invalid target at index %d: %v", i, err),
			)
		}
	}

	return nil
}

// validateTitle validates the message title
func (b *MessageBuilder) validateTitle(title string) error {
	if len(title) == 0 {
		return errors.NewValidationError(errors.ErrMissingRequired, "title", "title cannot be empty")
	}

	if len(title) > MaxTitleLength {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"title",
			fmt.Sprintf("title length %d exceeds maximum %d characters", len(title), MaxTitleLength),
		)
	}

	// Check for invalid characters
	if strings.Contains(title, "\x00") {
		return errors.NewValidationError(errors.ErrInvalidFormat, "title", "title cannot contain null characters")
	}

	return nil
}

// validateBody validates the message body
func (b *MessageBuilder) validateBody(body string) error {
	if len(body) == 0 {
		return errors.NewValidationError(errors.ErrMissingRequired, "body", "body cannot be empty")
	}

	if len(body) > MaxBodyLength {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"body",
			fmt.Sprintf("body length %d exceeds maximum %d characters", len(body), MaxBodyLength),
		)
	}

	// Check for invalid characters
	if strings.Contains(body, "\x00") {
		return errors.NewValidationError(errors.ErrInvalidFormat, "body", "body cannot contain null characters")
	}

	return nil
}

// validateFormat validates the message format
func (b *MessageBuilder) validateFormat(format Format) error {
	if !format.IsValid() {
		return errors.NewValidationError(
			errors.ErrInvalidFormat,
			"format",
			fmt.Sprintf("invalid format '%s', must be one of: text, markdown, html", format),
		)
	}
	return nil
}

// validatePriority validates the message priority
func (b *MessageBuilder) validatePriority(priority Priority) error {
	if priority < PriorityLow || priority > PriorityUrgent {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"priority",
			fmt.Sprintf("invalid priority %d, must be between %d and %d", priority, PriorityLow, PriorityUrgent),
		)
	}
	return nil
}

// validateTargetCount validates the number of targets
func (b *MessageBuilder) validateTargetCount(count int) error {
	if count < MinTargetCount {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"targets",
			fmt.Sprintf("target count %d is below minimum %d", count, MinTargetCount),
		)
	}

	if count > MaxTargetCount {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"targets",
			fmt.Sprintf("target count %d exceeds maximum %d", count, MaxTargetCount),
		)
	}

	return nil
}

// validateTarget validates a single target
func (b *MessageBuilder) validateTarget(t target.Target) error {
	if strings.TrimSpace(t.Type) == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "target.type", "target type cannot be empty")
	}

	if strings.TrimSpace(t.Value) == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "target.value", "target value cannot be empty")
	}

	if strings.TrimSpace(t.Platform) == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "target.platform", "target platform cannot be empty")
	}

	return nil
}

// validateTargetParams validates target creation parameters
func (b *MessageBuilder) validateTargetParams(targetType, value, platform string) error {
	if strings.TrimSpace(targetType) == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "target_type", "target type cannot be empty")
	}

	if strings.TrimSpace(value) == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "target_value", "target value cannot be empty")
	}

	if strings.TrimSpace(platform) == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "target_platform", "target platform cannot be empty")
	}

	return nil
}

// validateEmailAddress validates an email address
func (b *MessageBuilder) validateEmailAddress(email string) error {
	if email == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "email", "email address cannot be empty")
	}

	// Basic email format validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.NewValidationError(
			errors.ErrInvalidFormat,
			"email",
			fmt.Sprintf("invalid email format: %s", email),
		)
	}

	return nil
}

// validatePhoneNumber validates a phone number
func (b *MessageBuilder) validatePhoneNumber(phone string) error {
	if phone == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "phone", "phone number cannot be empty")
	}

	// Basic phone number validation (allow digits, +, -, spaces, parentheses)
	phoneRegex := regexp.MustCompile(`^[\+]?[0-9\-\(\)\s]+$`)
	if !phoneRegex.MatchString(phone) {
		return errors.NewValidationError(
			errors.ErrInvalidFormat,
			"phone",
			fmt.Sprintf("invalid phone number format: %s", phone),
		)
	}

	// Check minimum length
	digitsOnly := regexp.MustCompile(`[0-9]`).FindAllString(phone, -1)
	if len(digitsOnly) < 6 {
		return errors.NewValidationError(
			errors.ErrInvalidFormat,
			"phone",
			"phone number must contain at least 6 digits",
		)
	}

	return nil
}

// validateWebhookURL validates a webhook URL
func (b *MessageBuilder) validateWebhookURL(webhookURL string) error {
	if webhookURL == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "webhook_url", "webhook URL cannot be empty")
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return errors.NewValidationError(
			errors.ErrInvalidFormat,
			"webhook_url",
			fmt.Sprintf("invalid URL format: %v", err),
		)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.NewValidationError(
			errors.ErrInvalidFormat,
			"webhook_url",
			"webhook URL must use http or https scheme",
		)
	}

	if parsedURL.Host == "" {
		return errors.NewValidationError(
			errors.ErrInvalidFormat,
			"webhook_url",
			"webhook URL must have a valid host",
		)
	}

	return nil
}

// validateFeishuIdentifier validates a Feishu identifier
func (b *MessageBuilder) validateFeishuIdentifier(identifier string) error {
	if identifier == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "feishu_identifier", "Feishu identifier cannot be empty")
	}

	// Check if it's a webhook URL
	if strings.HasPrefix(identifier, "http://") || strings.HasPrefix(identifier, "https://") {
		return b.validateWebhookURL(identifier)
	}

	// Check if it's an email
	if strings.Contains(identifier, "@") {
		return b.validateEmailAddress(identifier)
	}

	// For user/group IDs, just check they're not empty and reasonable length
	if len(identifier) > 100 {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"feishu_identifier",
			"Feishu identifier too long (max 100 characters)",
		)
	}

	return nil
}

// validateFeishuUserID validates a Feishu user ID
func (b *MessageBuilder) validateFeishuUserID(userID string) error {
	if userID == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "feishu_user_id", "Feishu user ID cannot be empty")
	}

	if len(userID) > 100 {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"feishu_user_id",
			"Feishu user ID too long (max 100 characters)",
		)
	}

	return nil
}

// validateFeishuGroupID validates a Feishu group ID
func (b *MessageBuilder) validateFeishuGroupID(groupID string) error {
	if groupID == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "feishu_group_id", "Feishu group ID cannot be empty")
	}

	if len(groupID) > 100 {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"feishu_group_id",
			"Feishu group ID too long (max 100 characters)",
		)
	}

	return nil
}

// validateMetadataKey validates a metadata key
func (b *MessageBuilder) validateMetadataKey(key string) error {
	if key == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "metadata_key", "metadata key cannot be empty")
	}

	if len(key) > MaxKeywordLength {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"metadata_key",
			fmt.Sprintf("metadata key too long (max %d characters)", MaxKeywordLength),
		)
	}

	return nil
}

// validateVariableKey validates a template variable key
func (b *MessageBuilder) validateVariableKey(key string) error {
	if key == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "variable_key", "variable key cannot be empty")
	}

	if len(key) > MaxKeywordLength {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"variable_key",
			fmt.Sprintf("variable key too long (max %d characters)", MaxKeywordLength),
		)
	}

	return nil
}

// validatePlatformDataKey validates a platform data key
func (b *MessageBuilder) validatePlatformDataKey(key string) error {
	if key == "" {
		return errors.NewValidationError(errors.ErrMissingRequired, "platform_data_key", "platform data key cannot be empty")
	}

	if len(key) > MaxKeywordLength {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"platform_data_key",
			fmt.Sprintf("platform data key too long (max %d characters)", MaxKeywordLength),
		)
	}

	return nil
}

// validateScheduleTime validates a scheduled time
func (b *MessageBuilder) validateScheduleTime(when time.Time) error {
	if when.IsZero() {
		return errors.NewValidationError(errors.ErrInvalidType, "schedule_time", "schedule time cannot be zero")
	}

	if when.Before(time.Now()) {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"schedule_time",
			"schedule time cannot be in the past",
		)
	}

	// Don't allow scheduling too far in the future (1 year)
	oneYearFromNow := time.Now().AddDate(1, 0, 0)
	if when.After(oneYearFromNow) {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"schedule_time",
			"schedule time cannot be more than 1 year in the future",
		)
	}

	return nil
}

// validateScheduleDuration validates a schedule duration
func (b *MessageBuilder) validateScheduleDuration(duration time.Duration) error {
	if duration <= 0 {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"schedule_duration",
			"schedule duration must be positive",
		)
	}

	// Don't allow scheduling more than 1 year in the future
	oneYear := 365 * 24 * time.Hour
	if duration > oneYear {
		return errors.NewValidationError(
			errors.ErrValueOutOfRange,
			"schedule_duration",
			"schedule duration cannot be more than 1 year",
		)
	}

	return nil
}
