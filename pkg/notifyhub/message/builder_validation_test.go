package message

import (
	"strings"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/errors"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

func TestMessageBuilderValidation(t *testing.T) {
	tests := []struct {
		name        string
		builderFunc func() (*Message, error)
		expectError bool
		errorCode   errors.Code
	}{
		{
			name: "valid message with all required fields",
			builderFunc: func() (*Message, error) {
				return NewMessage("Test Title").
					WithBody("Test body content").
					ToEmail("test@example.com").
					Build()
			},
			expectError: false,
		},
		{
			name: "empty title should fail",
			builderFunc: func() (*Message, error) {
				return NewMessage("").
					WithBody("Test body").
					ToEmail("test@example.com").
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrMissingRequired,
		},
		{
			name: "empty body should fail",
			builderFunc: func() (*Message, error) {
				return NewMessage("Test Title").
					WithBody("").
					ToEmail("test@example.com").
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrMissingRequired,
		},
		{
			name: "no targets should fail",
			builderFunc: func() (*Message, error) {
				return NewMessage("Test Title").
					WithBody("Test body").
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrMissingRequired,
		},
		{
			name: "title too long should fail",
			builderFunc: func() (*Message, error) {
				longTitle := strings.Repeat("a", MaxTitleLength+1)
				return NewMessage(longTitle).
					WithBody("Test body").
					ToEmail("test@example.com").
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrValueOutOfRange,
		},
		{
			name: "body too long should fail",
			builderFunc: func() (*Message, error) {
				longBody := strings.Repeat("a", MaxBodyLength+1)
				return NewMessage("Test Title").
					WithBody(longBody).
					ToEmail("test@example.com").
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrValueOutOfRange,
		},
		{
			name: "invalid email format should fail",
			builderFunc: func() (*Message, error) {
				return NewMessage("Test Title").
					WithBody("Test body").
					ToEmail("invalid-email").
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrInvalidFormat,
		},
		{
			name: "invalid phone number should fail",
			builderFunc: func() (*Message, error) {
				return NewMessage("Test Title").
					WithBody("Test body").
					ToPhone("invalid-phone").
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrInvalidFormat,
		},
		{
			name: "invalid webhook URL should fail",
			builderFunc: func() (*Message, error) {
				return NewMessage("Test Title").
					WithBody("Test body").
					ToWebhook("invalid-url").
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrInvalidFormat,
		},
		{
			name: "too many targets should fail",
			builderFunc: func() (*Message, error) {
				builder := NewMessage("Test Title").
					WithBody("Test body")

				// Add too many targets
				for i := 0; i <= MaxTargetCount; i++ {
					builder = builder.ToEmail("test@example.com")
				}

				return builder.Build()
			},
			expectError: true,
			errorCode:   errors.ErrValueOutOfRange,
		},
		{
			name: "invalid priority should fail",
			builderFunc: func() (*Message, error) {
				return NewMessage("Test Title").
					WithBody("Test body").
					WithPriority(Priority(999)).
					ToEmail("test@example.com").
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrValueOutOfRange,
		},
		{
			name: "invalid format should fail",
			builderFunc: func() (*Message, error) {
				return NewMessage("Test Title").
					WithBody("Test body").
					WithFormat(Format("invalid")).
					ToEmail("test@example.com").
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrInvalidFormat,
		},
		{
			name: "scheduled time in past should fail",
			builderFunc: func() (*Message, error) {
				pastTime := time.Now().Add(-1 * time.Hour)
				return NewMessage("Test Title").
					WithBody("Test body").
					ToEmail("test@example.com").
					ScheduleAt(pastTime).
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrValueOutOfRange,
		},
		{
			name: "negative schedule duration should fail",
			builderFunc: func() (*Message, error) {
				return NewMessage("Test Title").
					WithBody("Test body").
					ToEmail("test@example.com").
					ScheduleIn(-1 * time.Hour).
					Build()
			},
			expectError: true,
			errorCode:   errors.ErrValueOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := tt.builderFunc()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}

				// Check if it's a NotifyError with the expected code
				if notifyErr, ok := err.(*errors.NotifyError); ok {
					if notifyErr.Code != tt.errorCode {
						t.Errorf("Expected error code %s, got %s", tt.errorCode, notifyErr.Code)
					}
				} else {
					t.Errorf("Expected NotifyError, got %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
					return
				}

				if msg == nil {
					t.Errorf("Expected message but got nil")
				}
			}
		})
	}
}

func TestMessageBuilderValidationMethods(t *testing.T) {
	t.Run("HasErrors should track validation errors", func(t *testing.T) {
		builder := NewMessage("Test").WithBody("Test")

		// Initially no errors
		if builder.HasErrors() {
			t.Errorf("Expected no errors initially")
		}

		// Add invalid email
		builder.ToEmail("invalid-email")

		// Should have errors now
		if !builder.HasErrors() {
			t.Errorf("Expected to have errors after invalid email")
		}

		// Should have exactly one error
		errors := builder.GetErrors()
		if len(errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(errors))
		}
	})

	t.Run("ClearErrors should remove all errors", func(t *testing.T) {
		builder := NewMessage("Test").WithBody("Test")

		// Add some invalid data
		builder.ToEmail("invalid-email")
		builder.ToPhone("invalid-phone")

		// Should have errors
		if !builder.HasErrors() {
			t.Errorf("Expected to have errors")
		}

		// Clear errors
		builder.ClearErrors()

		// Should have no errors now
		if builder.HasErrors() {
			t.Errorf("Expected no errors after clearing")
		}
	})

	t.Run("Validate should check current state", func(t *testing.T) {
		builder := NewMessage("Test").WithBody("Test")

		// Should fail validation (no targets)
		err := builder.Validate()
		if err == nil {
			t.Errorf("Expected validation error for missing targets")
		}

		// Add target
		builder.ToEmail("test@example.com")

		// Should pass validation now
		err = builder.Validate()
		if err != nil {
			t.Errorf("Expected validation to pass, got: %v", err)
		}
	})

	t.Run("BuildUnsafe should bypass validation", func(t *testing.T) {
		// Create invalid message
		msg := NewMessage("").WithBody("").BuildUnsafe()

		// Should get message despite validation errors
		if msg == nil {
			t.Errorf("Expected message from BuildUnsafe")
		}

		if msg.Title != "" || msg.Body != "" {
			t.Errorf("Expected empty fields to remain empty")
		}
	})
}

func TestValidationHelpers(t *testing.T) {
	builder := NewMessage("Test")

	t.Run("validateEmailAddress", func(t *testing.T) {
		tests := []struct {
			email       string
			expectError bool
		}{
			{"test@example.com", false},
			{"user.name+tag@domain.co.uk", false},
			{"invalid-email", true},
			{"@domain.com", true},
			{"user@", true},
			{"", true},
		}

		for _, tt := range tests {
			err := builder.validateEmailAddress(tt.email)
			if (err != nil) != tt.expectError {
				t.Errorf("validateEmailAddress(%s): expected error=%v, got error=%v",
					tt.email, tt.expectError, err != nil)
			}
		}
	})

	t.Run("validatePhoneNumber", func(t *testing.T) {
		tests := []struct {
			phone       string
			expectError bool
		}{
			{"+1234567890", false},
			{"(123) 456-7890", false},
			{"123-456-7890", false},
			{"1234567890", false},
			{"12345", true}, // too short
			{"abc123", true}, // invalid chars
			{"", true},
		}

		for _, tt := range tests {
			err := builder.validatePhoneNumber(tt.phone)
			if (err != nil) != tt.expectError {
				t.Errorf("validatePhoneNumber(%s): expected error=%v, got error=%v",
					tt.phone, tt.expectError, err != nil)
			}
		}
	})

	t.Run("validateWebhookURL", func(t *testing.T) {
		tests := []struct {
			url         string
			expectError bool
		}{
			{"https://example.com/webhook", false},
			{"http://localhost:8080/hook", false},
			{"ftp://example.com", true}, // wrong scheme
			{"invalid-url", true},
			{"", true},
		}

		for _, tt := range tests {
			err := builder.validateWebhookURL(tt.url)
			if (err != nil) != tt.expectError {
				t.Errorf("validateWebhookURL(%s): expected error=%v, got error=%v",
					tt.url, tt.expectError, err != nil)
			}
		}
	})

	t.Run("validateTargetCount", func(t *testing.T) {
		tests := []struct {
			count       int
			expectError bool
		}{
			{1, false},
			{50, false},
			{MaxTargetCount, false},
			{0, true},
			{MaxTargetCount + 1, true},
		}

		for _, tt := range tests {
			err := builder.validateTargetCount(tt.count)
			if (err != nil) != tt.expectError {
				t.Errorf("validateTargetCount(%d): expected error=%v, got error=%v",
					tt.count, tt.expectError, err != nil)
			}
		}
	})
}

func TestBuilderChaining(t *testing.T) {
	t.Run("builder should support method chaining", func(t *testing.T) {
		msg, err := NewMessage("Test Title").
			WithBody("Test body content").
			WithFormat(FormatMarkdown).
			WithPriority(PriorityHigh).
			WithMetadata("key", "value").
			WithVariable("name", "John").
			ToEmail("test@example.com").
			ToPhone("+1234567890").
			ScheduleIn(1 * time.Hour).
			Build()

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if msg == nil {
			t.Errorf("Expected message but got nil")
		}

		// Verify all fields were set correctly
		if msg.Title != "Test Title" {
			t.Errorf("Expected title 'Test Title', got '%s'", msg.Title)
		}

		if msg.Format != FormatMarkdown {
			t.Errorf("Expected format markdown, got %s", msg.Format)
		}

		if msg.Priority != PriorityHigh {
			t.Errorf("Expected priority high, got %d", msg.Priority)
		}

		if len(msg.Targets) != 2 {
			t.Errorf("Expected 2 targets, got %d", len(msg.Targets))
		}

		if msg.ScheduledAt == nil {
			t.Errorf("Expected scheduled time to be set")
		}
	})

	t.Run("builder with validation errors should continue chaining", func(t *testing.T) {
		builder := NewMessage("Test").
			WithBody("Test").
			ToEmail("invalid-email"). // Should add error
			ToPhone("valid-phone").    // Should continue despite error
			WithPriority(Priority(999)) // Should add another error

		// Should have errors
		if !builder.HasErrors() {
			t.Errorf("Expected builder to have errors")
		}

		// Should have multiple errors
		errors := builder.GetErrors()
		if len(errors) < 2 {
			t.Errorf("Expected at least 2 errors, got %d", len(errors))
		}

		// Build should fail
		_, err := builder.Build()
		if err == nil {
			t.Errorf("Expected build to fail due to validation errors")
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("nil maps should be handled gracefully", func(t *testing.T) {
		builder := NewMessage("Test").WithBody("Test")

		// These should not panic and should add validation errors
		builder.WithVariables(nil)
		builder.WithPlatformData(nil)

		if !builder.HasErrors() {
			t.Errorf("Expected errors for nil maps")
		}
	})

	t.Run("whitespace-only fields should be treated as empty", func(t *testing.T) {
		_, err := NewMessage("   ").
			WithBody("   ").
			ToEmail("test@example.com").
			Build()

		if err == nil {
			t.Errorf("Expected error for whitespace-only title and body")
		}
	})

	t.Run("null characters should be rejected", func(t *testing.T) {
		builder := NewMessage("Title\x00").WithBody("Test").ToEmail("test@example.com")

		_, err := builder.Build()
		if err == nil {
			t.Errorf("Expected error for null character in title")
		}

		builder = NewMessage("Title").WithBody("Body\x00").ToEmail("test@example.com")

		_, err = builder.Build()
		if err == nil {
			t.Errorf("Expected error for null character in body")
		}
	})

	t.Run("empty target fields should be rejected", func(t *testing.T) {
		emptyTarget := target.Target{
			Type:     "",
			Value:    "test",
			Platform: "test",
		}

		builder := NewMessage("Test").WithBody("Test").ToTarget(emptyTarget)

		if !builder.HasErrors() {
			t.Errorf("Expected error for empty target type")
		}
	})
}