package client

import (
	"context"
	"testing"
	"time"

	"github.com/kart-io/notifyhub/config"
	"github.com/kart-io/notifyhub/notifiers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHubCreation(t *testing.T) {
	// Test creating hub with test defaults
	hub, err := New(config.WithTestDefaults())
	require.NoError(t, err)
	require.NotNil(t, hub)

	// Clean up
	hub.Stop()
}

func TestQuickSendAPI(t *testing.T) {
	hub, err := New(config.WithTestDefaults())
	require.NoError(t, err)
	defer hub.Stop()

	ctx := context.Background()

	// Test QuickSend
	err = hub.QuickSend(ctx, "Test Title", "Test Body", "email:test@example.com@mock")
	assert.NoError(t, err)

	// Test Email shortcut
	err = hub.Email(ctx, "Email Test", "Email Body", "test@example.com")
	assert.NoError(t, err)

	// Test FeishuGroup shortcut
	err = hub.FeishuGroup(ctx, "Feishu Test", "Feishu Body", "test-group")
	assert.NoError(t, err)
}

func TestTargetList(t *testing.T) {
	// Test TargetList builder
	list := NewTargetList().
		AddEmails("user1@example.com", "user2@example.com").
		AddFeishuGroups("group1", "group2")

	targets := list.Build()
	assert.Len(t, targets, 4)
}

func TestEnhancedBatchBuilder(t *testing.T) {
	hub, err := New(config.WithTestDefaults())
	require.NoError(t, err)
	defer hub.Stop()

	// Test enhanced batch builder
	builder := hub.NewEnhancedBatch()
	assert.NotNil(t, builder)
}

func TestNotifyHubError(t *testing.T) {
	// Test error creation and methods
	err := NewConfigError("INVALID_CONFIG", "Configuration is invalid", "Check your settings")

	assert.Equal(t, ErrorCategoryConfig, err.Category)
	assert.Equal(t, "INVALID_CONFIG", err.Code)
	assert.Equal(t, "Configuration is invalid", err.Message)
	assert.False(t, err.IsRetryable())
	assert.Contains(t, err.GetSuggestions(), "Check your settings")
	assert.Contains(t, err.Error(), "CONFIG")
	assert.Contains(t, err.Error(), "INVALID_CONFIG")
}

func TestErrorCollector(t *testing.T) {
	collector := NewErrorCollector()

	// Initially no errors
	assert.False(t, collector.HasErrors())
	assert.Equal(t, 0, collector.Count())

	// Add some errors
	err1 := NewNetworkError("TIMEOUT", "Network timeout", nil, true)
	err2 := NewValidationError("INVALID_EMAIL", "Invalid email format", "user@", "Check email format")

	collector.Add(err1)
	collector.Add(err2)

	assert.True(t, collector.HasErrors())
	assert.Equal(t, 2, collector.Count())
	assert.Equal(t, err1, collector.FirstError())
	assert.Equal(t, err2, collector.LastError())

	categories := collector.Categories()
	assert.Equal(t, 1, categories[ErrorCategoryNetwork])
	assert.Equal(t, 1, categories[ErrorCategoryValidation])
}

func TestResultAnalyzer(t *testing.T) {
	// Create mock results
	results := []*notifiers.SendResult{
		{
			Platform: "email",
			Success:  true,
			Duration: 100 * time.Millisecond,
			Error:    "",
		},
		{
			Platform: "feishu",
			Success:  false,
			Duration: 50 * time.Millisecond,
			Error:    "Connection failed",
		},
	}

	analyzer := AnalyzeResults(results)

	// Test basic analysis
	assert.True(t, analyzer.HasSuccesses())
	assert.True(t, analyzer.HasFailures())
	assert.True(t, analyzer.IsPartialSuccess())
	assert.False(t, analyzer.IsFullSuccess())
	assert.False(t, analyzer.IsFullFailure())

	// Test success rate
	assert.Equal(t, 50.0, analyzer.SuccessRate())

	// Test platform filtering
	successful := analyzer.SuccessfulPlatforms()
	failed := analyzer.FailedPlatforms()
	assert.Contains(t, successful, "email")
	assert.Contains(t, failed, "feishu")

	// Test summary
	summary := analyzer.Summary()
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 1, summary.Successful)
	assert.Equal(t, 1, summary.Failed)
	assert.Equal(t, 0.5, summary.SuccessRate)
}

func TestConfigValidation(t *testing.T) {
	hub, err := New(config.WithTestDefaults())
	require.NoError(t, err)
	defer hub.Stop()

	ctx := context.Background()

	// Test validation
	result := hub.ValidateConfiguration(ctx)
	assert.NotNil(t, result)
}

func TestTemplateManager(t *testing.T) {
	hub, err := New(config.WithTestDefaults())
	require.NoError(t, err)
	defer hub.Stop()

	// Test template manager access
	templates := hub.Templates()
	assert.NotNil(t, templates)
}