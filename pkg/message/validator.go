// Package message provides message validation functionality for NotifyHub
package message

import (
	"fmt"
	"strings"
	"time"

	"github.com/kart/notifyhub/pkg/errors"
)

// Validator provides message validation functionality
type Validator struct {
	config ValidatorConfig
}

// ValidatorConfig configures message validation
type ValidatorConfig struct {
	MaxTitleLength      int           `json:"max_title_length"`
	MaxBodyLength       int           `json:"max_body_length"`
	MaxTargets          int           `json:"max_targets"`
	RequireTitle        bool          `json:"require_title"`
	RequireBody         bool          `json:"require_body"`
	RequireTargets      bool          `json:"require_targets"`
	AllowEmptyTargets   bool          `json:"allow_empty_targets"`
	AllowFutureSchedule bool          `json:"allow_future_schedule"`
	MaxScheduleAhead    time.Duration `json:"max_schedule_ahead"`
}

// NewValidator creates a new message validator
func NewValidator(config ValidatorConfig) *Validator {
	// Apply defaults
	if config.MaxTitleLength == 0 {
		config.MaxTitleLength = 200
	}
	if config.MaxBodyLength == 0 {
		config.MaxBodyLength = 4000
	}
	if config.MaxTargets == 0 {
		config.MaxTargets = 100
	}
	if config.MaxScheduleAhead == 0 {
		config.MaxScheduleAhead = 24 * time.Hour
	}

	return &Validator{config: config}
}

// DefaultValidator creates a validator with default configuration
func DefaultValidator() *Validator {
	return NewValidator(ValidatorConfig{
		MaxTitleLength:      200,
		MaxBodyLength:       4000,
		MaxTargets:          100,
		RequireTitle:        true,
		RequireBody:         true,
		RequireTargets:      true,
		AllowEmptyTargets:   false,
		AllowFutureSchedule: true,
		MaxScheduleAhead:    24 * time.Hour,
	})
}

// StrictValidator creates a validator with strict validation rules
func StrictValidator() *Validator {
	return NewValidator(ValidatorConfig{
		MaxTitleLength:      100,
		MaxBodyLength:       2000,
		MaxTargets:          50,
		RequireTitle:        true,
		RequireBody:         true,
		RequireTargets:      true,
		AllowEmptyTargets:   false,
		AllowFutureSchedule: true,
		MaxScheduleAhead:    12 * time.Hour,
	})
}

// LenientValidator creates a validator with lenient validation rules
func LenientValidator() *Validator {
	return NewValidator(ValidatorConfig{
		MaxTitleLength:      500,
		MaxBodyLength:       10000,
		MaxTargets:          500,
		RequireTitle:        false,
		RequireBody:         false,
		RequireTargets:      false,
		AllowEmptyTargets:   true,
		AllowFutureSchedule: true,
		MaxScheduleAhead:    168 * time.Hour, // 1 week
	})
}

// Validate validates a message according to the configuration
func (v *Validator) Validate(msg *Message) error {
	if msg == nil {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message cannot be nil",
		}
	}

	// Validate message ID
	if err := v.validateID(msg); err != nil {
		return err
	}

	// Validate title
	if err := v.validateTitle(msg); err != nil {
		return err
	}

	// Validate body
	if err := v.validateBody(msg); err != nil {
		return err
	}

	// Validate format
	if err := v.validateFormat(msg); err != nil {
		return err
	}

	// Validate priority
	if err := v.validatePriority(msg); err != nil {
		return err
	}

	// Validate targets
	if err := v.validateTargets(msg); err != nil {
		return err
	}

	// Validate scheduling
	if err := v.validateScheduling(msg); err != nil {
		return err
	}

	// Validate metadata
	if err := v.validateMetadata(msg); err != nil {
		return err
	}

	return nil
}

// validateID validates the message ID
func (v *Validator) validateID(msg *Message) error {
	if msg.ID == "" {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message ID cannot be empty",
		}
	}

	// Check for valid characters
	if strings.ContainsAny(msg.ID, " \t\n\r") {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message ID cannot contain whitespace characters",
		}
	}

	return nil
}

// validateTitle validates the message title
func (v *Validator) validateTitle(msg *Message) error {
	if v.config.RequireTitle && msg.Title == "" {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message title is required",
		}
	}

	if len(msg.Title) > v.config.MaxTitleLength {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: fmt.Sprintf("message title length %d exceeds maximum %d", len(msg.Title), v.config.MaxTitleLength),
		}
	}

	return nil
}

// validateBody validates the message body
func (v *Validator) validateBody(msg *Message) error {
	if v.config.RequireBody && msg.Body == "" {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message body is required",
		}
	}

	if len(msg.Body) > v.config.MaxBodyLength {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: fmt.Sprintf("message body length %d exceeds maximum %d", len(msg.Body), v.config.MaxBodyLength),
		}
	}

	return nil
}

// validateFormat validates the message format
func (v *Validator) validateFormat(msg *Message) error {
	switch msg.Format {
	case FormatText, FormatMarkdown, FormatHTML:
		return nil
	default:
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: fmt.Sprintf("invalid message format: %s", msg.Format),
		}
	}
}

// validatePriority validates the message priority
func (v *Validator) validatePriority(msg *Message) error {
	switch msg.Priority {
	case PriorityLow, PriorityNormal, PriorityHigh, PriorityUrgent:
		return nil
	default:
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: fmt.Sprintf("invalid message priority: %d", msg.Priority),
		}
	}
}

// validateTargets validates the message targets
func (v *Validator) validateTargets(msg *Message) error {
	if v.config.RequireTargets && len(msg.Targets) == 0 {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message must have at least one target",
		}
	}

	if !v.config.AllowEmptyTargets && len(msg.Targets) == 0 {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "message targets cannot be empty",
		}
	}

	if len(msg.Targets) > v.config.MaxTargets {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: fmt.Sprintf("message has %d targets, exceeds maximum %d", len(msg.Targets), v.config.MaxTargets),
		}
	}

	// Validate each target
	for i, target := range msg.Targets {
		if err := v.validateTarget(target, i); err != nil {
			return err
		}
	}

	return nil
}

// validateTarget validates a single target
func (v *Validator) validateTarget(target interface{}, index int) error {
	// Note: target validation should be done by the target package
	// This is a basic validation to ensure targets are not nil
	if target == nil {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: fmt.Sprintf("target at index %d cannot be nil", index),
		}
	}

	return nil
}

// validateScheduling validates message scheduling
func (v *Validator) validateScheduling(msg *Message) error {
	if msg.ScheduledAt == nil {
		return nil
	}

	// Check if future scheduling is allowed
	if !v.config.AllowFutureSchedule {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "future scheduling is not allowed",
		}
	}

	// Check if scheduled time is in the past
	if msg.ScheduledAt.Before(time.Now()) {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: "scheduled time cannot be in the past",
		}
	}

	// Check maximum schedule ahead limit
	if time.Until(*msg.ScheduledAt) > v.config.MaxScheduleAhead {
		return &errors.NotifyError{
			Code:    errors.ErrInvalidMessage,
			Message: fmt.Sprintf("scheduled time is too far in the future (max: %v)", v.config.MaxScheduleAhead),
		}
	}

	return nil
}

// validateMetadata validates message metadata
func (v *Validator) validateMetadata(msg *Message) error {
	// Check for reserved metadata keys
	reservedKeys := []string{"_internal", "_system", "_notifyhub"}
	for key := range msg.Metadata {
		for _, reserved := range reservedKeys {
			if strings.HasPrefix(strings.ToLower(key), reserved) {
				return &errors.NotifyError{
					Code:    errors.ErrInvalidMessage,
					Message: fmt.Sprintf("metadata key '%s' uses reserved prefix '%s'", key, reserved),
				}
			}
		}
	}

	return nil
}

// ValidateQuick performs a quick validation with basic checks
func ValidateQuick(msg *Message) error {
	validator := NewValidator(ValidatorConfig{
		RequireTitle:   true,
		RequireBody:    true,
		RequireTargets: true,
	})
	return validator.Validate(msg)
}

// ValidateForPlatform validates a message for a specific platform
func ValidateForPlatform(msg *Message, platformName string, platformConfig map[string]interface{}) error {
	// Basic validation first
	if err := ValidateQuick(msg); err != nil {
		return err
	}

	// Platform-specific validation would be implemented here
	// This could include checking platform-specific limits, formats, etc.

	return nil
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ValidateWithResult validates a message and returns detailed results
func (v *Validator) ValidateWithResult(msg *Message) *ValidationResult {
	err := v.Validate(msg)
	if err == nil {
		return &ValidationResult{Valid: true}
	}

	result := &ValidationResult{
		Valid:  false,
		Errors: []string{err.Error()},
	}

	return result
}
