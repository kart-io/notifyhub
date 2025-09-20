package message

import (
	"testing"
	"time"

	"github.com/kart-io/notifyhub/notifiers"
	"github.com/stretchr/testify/assert"
)

func TestNewBuilder(t *testing.T) {
	t.Run("Create new builder", func(t *testing.T) {
		builder := NewBuilder()
		assert.NotNil(t, builder)
		assert.NotNil(t, builder.message)
		assert.Equal(t, notifiers.FormatText, builder.message.Format)
		assert.Equal(t, 3, builder.message.Priority)
		assert.NotNil(t, builder.message.Variables)
		assert.NotNil(t, builder.message.Metadata)
		assert.False(t, builder.message.CreatedAt.IsZero())
	})
}

func TestBasicBuilderMethods(t *testing.T) {
	t.Run("Title method", func(t *testing.T) {
		builder := NewBuilder()
		result := builder.Title("Test Title")

		assert.Equal(t, builder, result) // Should return self for chaining
		assert.Equal(t, "Test Title", builder.message.Title)
	})

	t.Run("Body method", func(t *testing.T) {
		builder := NewBuilder()
		result := builder.Body("Test Body")

		assert.Equal(t, builder, result)
		assert.Equal(t, "Test Body", builder.message.Body)
	})

	t.Run("Priority method", func(t *testing.T) {
		builder := NewBuilder()
		result := builder.Priority(5)

		assert.Equal(t, builder, result)
		assert.Equal(t, 5, builder.message.Priority)
	})

	t.Run("Format method", func(t *testing.T) {
		builder := NewBuilder()
		result := builder.Format(notifiers.FormatMarkdown)

		assert.Equal(t, builder, result)
		assert.Equal(t, notifiers.FormatMarkdown, builder.message.Format)
	})

	t.Run("AddTarget method", func(t *testing.T) {
		builder := NewBuilder()
		target := notifiers.Target{
			Type:     notifiers.TargetTypeEmail,
			Value:    "test@example.com",
			Platform: "email",
		}

		result := builder.AddTarget(target)
		assert.Equal(t, builder, result)
		assert.Len(t, builder.message.Targets, 1)
		assert.Equal(t, target, builder.message.Targets[0])
	})
}

func TestBuildMethod(t *testing.T) {
	t.Run("Build message", func(t *testing.T) {
		builder := NewBuilder()
		message := builder.
			Title("Test Message").
			Body("Test Body").
			Priority(4).
			Format(notifiers.FormatHTML).
			Build()

		assert.Equal(t, "Test Message", message.Title)
		assert.Equal(t, "Test Body", message.Body)
		assert.Equal(t, 4, message.Priority)
		assert.Equal(t, notifiers.FormatHTML, message.Format)
		assert.NotEmpty(t, message.ID) // Should generate ID
	})

	t.Run("Build with custom ID", func(t *testing.T) {
		builder := NewBuilder()
		customID := "custom-message-id"
		message := builder.
			ID(customID).
			Title("Test").
			Body("Test").
			Build()

		assert.Equal(t, customID, message.ID)
	})
}

func TestAdvancedBuilderMethods(t *testing.T) {
	t.Run("Variable method", func(t *testing.T) {
		builder := NewBuilder()
		result := builder.Variable("name", "John")

		assert.Equal(t, builder, result)
		assert.Equal(t, "John", builder.message.Variables["name"])
	})

	t.Run("Variables method", func(t *testing.T) {
		builder := NewBuilder()
		variables := map[string]interface{}{
			"user":   "Alice",
			"count":  5,
			"active": true,
		}

		result := builder.Variables(variables)
		assert.Equal(t, builder, result)
		assert.Equal(t, "Alice", builder.message.Variables["user"])
		assert.Equal(t, 5, builder.message.Variables["count"])
		assert.Equal(t, true, builder.message.Variables["active"])
	})

	t.Run("Metadata method", func(t *testing.T) {
		builder := NewBuilder()
		result := builder.Metadata("source", "api")

		assert.Equal(t, builder, result)
		assert.Equal(t, "api", builder.message.Metadata["source"])
	})

	t.Run("Template method", func(t *testing.T) {
		builder := NewBuilder()
		result := builder.Template("notification_template")

		assert.Equal(t, builder, result)
		assert.Equal(t, "notification_template", builder.message.Template)
	})

	t.Run("Delay method", func(t *testing.T) {
		builder := NewBuilder()
		delay := 5 * time.Minute
		result := builder.Delay(delay)

		assert.Equal(t, builder, result)
		assert.Equal(t, delay, builder.message.Delay)
	})

	t.Run("ID method", func(t *testing.T) {
		builder := NewBuilder()
		customID := "test-message-123"
		result := builder.ID(customID)

		assert.Equal(t, builder, result)
		assert.Equal(t, customID, builder.message.ID)
	})
}

func TestPlatformBuilders(t *testing.T) {
	t.Run("Feishu platform builder", func(t *testing.T) {
		builder := NewBuilder()
		platformBuilder := builder.Feishu()

		assert.NotNil(t, platformBuilder)
		// Platform builder should be able to return to base
		assert.Equal(t, builder, platformBuilder.Builder())
	})

	t.Run("Email platform builder", func(t *testing.T) {
		builder := NewBuilder()
		platformBuilder := builder.Email()

		assert.NotNil(t, platformBuilder)
		assert.Equal(t, builder, platformBuilder.Builder())
	})

	t.Run("SMS platform builder", func(t *testing.T) {
		builder := NewBuilder()
		platformBuilder := builder.SMS()

		assert.NotNil(t, platformBuilder)
		assert.Equal(t, builder, platformBuilder.Builder())
	})

	t.Run("Generic platform builder", func(t *testing.T) {
		builder := NewBuilder()
		platformBuilder := builder.Platform("test_platform")

		assert.NotNil(t, platformBuilder)
		assert.Equal(t, builder, platformBuilder.Builder())
	})
}

func TestConvenienceCreators(t *testing.T) {
	t.Run("Quick creator", func(t *testing.T) {
		builder := Quick("Quick Title", "Quick Body")
		message := builder.Build()

		assert.Equal(t, "Quick Title", message.Title)
		assert.Equal(t, "Quick Body", message.Body)
	})

	t.Run("Alert creator", func(t *testing.T) {
		builder := Alert("Alert Title", "Alert Body")
		message := builder.Build()

		assert.Equal(t, "Alert Title", message.Title)
		assert.Equal(t, "Alert Body", message.Body)
		assert.Equal(t, 4, message.Priority) // Alert priority
	})

	t.Run("Emergency creator", func(t *testing.T) {
		builder := Emergency("Emergency Title", "Emergency Body")
		message := builder.Build()

		assert.Equal(t, "Emergency Title", message.Title)
		assert.Equal(t, "Emergency Body", message.Body)
		assert.Equal(t, 5, message.Priority) // Emergency priority
	})

	t.Run("Notice creator", func(t *testing.T) {
		builder := Notice("Notice Title", "Notice Body")
		message := builder.Build()

		assert.Equal(t, "Notice Title", message.Title)
		assert.Equal(t, "Notice Body", message.Body)
		assert.Equal(t, 3, message.Priority) // Notice priority
	})

	t.Run("Markdown creator", func(t *testing.T) {
		builder := Markdown("Markdown Title", "**Bold** text")
		message := builder.Build()

		assert.Equal(t, "Markdown Title", message.Title)
		assert.Equal(t, "**Bold** text", message.Body)
		assert.Equal(t, notifiers.FormatMarkdown, message.Format)
	})

	t.Run("HTML creator", func(t *testing.T) {
		builder := HTML("HTML Title", "<b>Bold</b> text")
		message := builder.Build()

		assert.Equal(t, "HTML Title", message.Title)
		assert.Equal(t, "<b>Bold</b> text", message.Body)
		assert.Equal(t, notifiers.FormatHTML, message.Format)
	})

	t.Run("Card creator", func(t *testing.T) {
		builder := Card("Card Title", "Card content")
		message := builder.Build()

		assert.Equal(t, "Card Title", message.Title)
		assert.Equal(t, "Card content", message.Body)
		assert.Equal(t, notifiers.FormatCard, message.Format)
	})
}

func TestBuilderCloning(t *testing.T) {
	t.Run("Clone builder", func(t *testing.T) {
		original := NewBuilder()
		original.Title("Original Title").
			Body("Original Body").
			Priority(3)
		original.Variable("name", "John")
		original.Metadata("source", "test")

		clone := original.Clone()

		// Modify clone
		clone.Title("Cloned Title")
		clone.Variable("name", "Jane")
		clonedMessage := clone.Build()

		originalMessage := original.Build()

		// Verify original is unchanged
		assert.Equal(t, "Original Title", originalMessage.Title)
		assert.Equal(t, "John", originalMessage.Variables["name"])

		// Verify clone was modified
		assert.Equal(t, "Cloned Title", clonedMessage.Title)
		assert.Equal(t, "Jane", clonedMessage.Variables["name"])

		// Verify shared metadata is properly copied
		assert.Equal(t, "test", originalMessage.Metadata["source"])
		assert.Equal(t, "test", clonedMessage.Metadata["source"])
	})

	t.Run("Deep copy of slices and maps", func(t *testing.T) {
		original := NewBuilder()
		original.AddTarget(notifiers.Target{Type: notifiers.TargetTypeEmail, Value: "original@test.com"})
		original.Variable("shared", "original")
		original.Metadata("env", "test")

		clone := original.Clone()

		// Add different target to clone
		clone.AddTarget(notifiers.Target{Type: notifiers.TargetTypeUser, Value: "clone_user"})
		clone.Variable("shared", "clone")
		clonedMessage := clone.Build()

		originalMessage := original.Build()

		// Original should have only one target
		assert.Len(t, originalMessage.Targets, 1)
		assert.Equal(t, "original@test.com", originalMessage.Targets[0].Value)
		assert.Equal(t, "original", originalMessage.Variables["shared"])

		// Clone should have two targets
		assert.Len(t, clonedMessage.Targets, 2)
		assert.Equal(t, "clone", clonedMessage.Variables["shared"])
	})
}

func TestBuilderValidation(t *testing.T) {
	t.Run("Valid builder", func(t *testing.T) {
		builder := NewBuilder()
		builder.Title("Valid Title").
			Body("Valid Body").
			Priority(3)

		err := builder.Validate()
		assert.NoError(t, err)
	})

	t.Run("Empty title and body", func(t *testing.T) {
		builder := NewBuilder()
		builder.Priority(3)

		err := builder.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title or body")
	})

	t.Run("Invalid priority", func(t *testing.T) {
		builder := NewBuilder()
		builder.Title("Title").
			Body("Body").
			Priority(10) // Invalid priority

		err := builder.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "priority")
	})

	t.Run("Valid with only title", func(t *testing.T) {
		builder := NewBuilder()
		builder.Title("Only Title").
			Priority(3)

		err := builder.Validate()
		assert.NoError(t, err)
	})

	t.Run("Valid with only body", func(t *testing.T) {
		builder := NewBuilder()
		builder.Body("Only Body").
			Priority(3)

		err := builder.Validate()
		assert.NoError(t, err)
	})
}

func TestNoOpPlatformBuilder(t *testing.T) {
	t.Run("NoOp platform builder", func(t *testing.T) {
		// Create a CoreMessageBuilder for compatibility
		baseBuilder := NewBuilder()
		noOp := &noOpPlatformBuilder{base: baseBuilder}

		assert.Equal(t, "unknown", noOp.Platform())
		assert.Equal(t, baseBuilder, noOp.Builder())
		assert.Equal(t, baseBuilder.Build(), noOp.Builder().Build())
	})
}

func TestMessageIDGeneration(t *testing.T) {
	t.Run("Generate unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)

		// Generate 100 IDs and ensure they're unique
		for i := 0; i < 100; i++ {
			id := generateMessageID()
			assert.NotEmpty(t, id)
			assert.False(t, ids[id], "Generated duplicate ID: %s", id)
			ids[id] = true
		}
	})

	t.Run("ID format", func(t *testing.T) {
		id := generateMessageID()
		assert.True(t, len(id) > 4) // Should be longer than just "msg_"
		assert.Contains(t, id, "msg_")
	})
}

func TestGetMessage(t *testing.T) {
	t.Run("GetMessage returns current state", func(t *testing.T) {
		builder := NewBuilder()
		builder.Title("Test Title").
			Body("Test Body")

		message := builder.GetMessage()
		assert.Equal(t, "Test Title", message.Title)
		assert.Equal(t, "Test Body", message.Body)

		// Message should be the same instance
		assert.Equal(t, builder.message, message)
	})
}

func TestMessageBuilderInterfaceCompliance(t *testing.T) {
	t.Run("Implements CoreMessageBuilder interface", func(t *testing.T) {
		builder := NewBuilder()

		// Test interface methods
		result := builder.Title("Interface Test")
		assert.NotNil(t, result)

		result = builder.Body("Interface Body")
		assert.NotNil(t, result)

		result = builder.Priority(3)
		assert.NotNil(t, result)

		result = builder.Format(notifiers.FormatText)
		assert.NotNil(t, result)

		target := notifiers.Target{Type: notifiers.TargetTypeEmail, Value: "test@example.com"}
		result = builder.AddTarget(target)
		assert.NotNil(t, result)

		message := builder.Build()
		assert.NotNil(t, message)
		assert.Equal(t, "Interface Test", message.Title)
		assert.Equal(t, "Interface Body", message.Body)

		// Test platform-specific methods
		platformBuilder := builder.Feishu()
		assert.NotNil(t, platformBuilder)
		platformBuilder = builder.Email()
		assert.NotNil(t, platformBuilder)
		platformBuilder = builder.SMS()
		assert.NotNil(t, platformBuilder)
		platformBuilder = builder.Platform("test")
		assert.NotNil(t, platformBuilder)
	})
}

// Benchmark tests
func BenchmarkBuilderCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewBuilder()
	}
}

func BenchmarkMessageBuilding(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewBuilder().
			Title("Benchmark Test").
			Body("Benchmark test body").
			Priority(3).
			Format(notifiers.FormatText).
			Build()
	}
}

func BenchmarkBuilderCloning(b *testing.B) {
	original := NewBuilder()
	original.Title("Original").
		Body("Original body")
	original.Variable("test", "value")
	original.Metadata("env", "test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		original.Clone()
	}
}

func BenchmarkPlatformBuilderAccess(b *testing.B) {
	builder := NewBuilder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.Feishu()
		builder.Email()
		builder.SMS()
	}
}
