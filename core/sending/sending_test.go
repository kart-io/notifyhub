package sending

import (
	"testing"
)

// Test Target creation
func TestNewTarget(t *testing.T) {
	target := NewTarget(TargetTypeEmail, "test@example.com", "email")

	if target.Type != TargetTypeEmail {
		t.Errorf("Expected type %s, got %s", TargetTypeEmail, target.Type)
	}

	if target.Value != "test@example.com" {
		t.Errorf("Expected value 'test@example.com', got '%s'", target.Value)
	}

	if target.Platform != "email" {
		t.Errorf("Expected platform 'email', got '%s'", target.Platform)
	}

	if target.Metadata == nil {
		t.Error("Expected metadata to be initialized")
	}
}

// Test Target metadata
func TestTargetMetadata(t *testing.T) {
	target := NewTarget(TargetTypeUser, "user123", "slack")

	// Test AddMetadata
	target.AddMetadata("channel", "#general")
	target.AddMetadata("priority", "high")

	// Test GetMetadata
	channel, exists := target.GetMetadata("channel")
	if !exists {
		t.Error("Expected channel metadata to exist")
	}
	if channel != "#general" {
		t.Errorf("Expected channel '#general', got '%s'", channel)
	}

	priority, exists := target.GetMetadata("priority")
	if !exists {
		t.Error("Expected priority metadata to exist")
	}
	if priority != "high" {
		t.Errorf("Expected priority 'high', got '%s'", priority)
	}

	// Test non-existent metadata
	_, exists = target.GetMetadata("nonexistent")
	if exists {
		t.Error("Expected non-existent metadata to not exist")
	}
}

// Test Target validation
func TestTargetValidation(t *testing.T) {
	tests := []struct {
		name    string
		target  Target
		wantErr bool
	}{
		{
			name: "Valid email target",
			target: Target{
				Type:     TargetTypeEmail,
				Value:    "test@example.com",
				Platform: "email",
			},
			wantErr: false,
		},
		{
			name: "Valid user target",
			target: Target{
				Type:     TargetTypeUser,
				Value:    "user123",
				Platform: "slack",
			},
			wantErr: false,
		},
		{
			name: "Empty type",
			target: Target{
				Value:    "test@example.com",
				Platform: "email",
			},
			wantErr: true,
		},
		{
			name: "Empty value",
			target: Target{
				Type:     TargetTypeEmail,
				Platform: "email",
			},
			wantErr: true,
		},
		{
			name: "Empty platform",
			target: Target{
				Type:  TargetTypeEmail,
				Value: "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "Invalid email format",
			target: Target{
				Type:     TargetTypeEmail,
				Value:    "invalid-email",
				Platform: "email",
			},
			wantErr: true,
		},
		{
			name: "Too long value",
			target: Target{
				Type:     TargetTypeUser,
				Value:    string(make([]byte, 256)), // 256 characters
				Platform: "slack",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.target.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Target.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test Target String representation
func TestTargetString(t *testing.T) {
	target := Target{
		Type:     TargetTypeEmail,
		Value:    "test@example.com",
		Platform: "email",
	}

	expected := "email:email:test@example.com"
	if target.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, target.String())
	}
}

// Test Target getters
func TestTargetGetters(t *testing.T) {
	target := Target{
		Type:     TargetTypeGroup,
		Value:    "dev-team",
		Platform: "slack",
	}

	if target.GetType() != TargetTypeGroup {
		t.Errorf("Expected type %s, got %s", TargetTypeGroup, target.GetType())
	}

	if target.GetValue() != "dev-team" {
		t.Errorf("Expected value 'dev-team', got '%s'", target.GetValue())
	}

	if target.GetPlatform() != "slack" {
		t.Errorf("Expected platform 'slack', got '%s'", target.GetPlatform())
	}
}

// Test Result creation
func TestNewResult(t *testing.T) {
	target := NewTarget(TargetTypeEmail, "test@example.com", "email")
	messageID := "test-msg-123"

	result := NewResult(messageID, target)

	if result.MessageID != messageID {
		t.Errorf("Expected message ID '%s', got '%s'", messageID, result.MessageID)
	}

	if result.Target.Type != target.Type {
		t.Error("Expected target type to match")
	}
	if result.Target.Value != target.Value {
		t.Error("Expected target value to match")
	}
	if result.Target.Platform != target.Platform {
		t.Error("Expected target platform to match")
	}

	if result.Status != StatusPending {
		t.Errorf("Expected status to be %s, got %s", StatusPending, result.Status)
	}

	if result.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

// Test SendingResults
func TestSendingResults(t *testing.T) {
	target1 := NewTarget(TargetTypeEmail, "test1@example.com", "email")
	target2 := NewTarget(TargetTypeEmail, "test2@example.com", "email")

	result1 := NewResult("msg-1", target1)
	result1.Success = true
	result1.Status = StatusSent

	result2 := NewResult("msg-2", target2)
	result2.Success = false
	result2.Status = StatusFailed

	results := &SendingResults{
		Results: []*Result{result1, result2},
	}

	if len(results.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results.Results))
	}

	// Test success count
	successCount := 0
	for _, result := range results.Results {
		if result.Success {
			successCount++
		}
	}

	if successCount != 1 {
		t.Errorf("Expected 1 successful result, got %d", successCount)
	}
}

// Test status constants
func TestStatusConstants(t *testing.T) {
	statuses := []Status{
		StatusPending,
		StatusSending,
		StatusSent,
		StatusFailed,
		StatusRetrying,
		StatusCancelled,
	}

	expectedValues := []string{
		"pending",
		"sending",
		"sent",
		"failed",
		"retrying",
		"cancelled",
	}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("Expected status '%s', got '%s'", expectedValues[i], string(status))
		}
	}
}

// Test target type constants
func TestTargetTypes(t *testing.T) {
	types := []TargetType{
		TargetTypeEmail,
		TargetTypeUser,
		TargetTypeGroup,
		TargetTypeChannel,
		TargetTypeWebhook,
		TargetTypeSMS,
		TargetTypeOther,
	}

	expectedValues := []string{
		"email",
		"user",
		"group",
		"channel",
		"webhook",
		"sms",
		"other",
	}

	for i, targetType := range types {
		if string(targetType) != expectedValues[i] {
			t.Errorf("Expected target type '%s', got '%s'", expectedValues[i], string(targetType))
		}
	}
}

// Test result with error
func TestResultWithError(t *testing.T) {
	target := NewTarget(TargetTypeEmail, "test@example.com", "email")
	messageID := "test-msg-123"

	result := NewResult(messageID, target)
	result.Success = false
	result.Status = StatusFailed
	result.Error = ErrSendingFailed

	if result.Success {
		t.Error("Expected success to be false")
	}

	if result.Error != ErrSendingFailed {
		t.Errorf("Expected error %v, got %v", ErrSendingFailed, result.Error)
	}
}

// Test concurrent target access
func TestConcurrentTargetAccess(t *testing.T) {
	target := NewTarget(TargetTypeUser, "user123", "slack")

	done := make(chan bool, 2)

	// Goroutine 1: Add metadata
	go func() {
		for i := 0; i < 100; i++ {
			target.AddMetadata("key1", "value1")
		}
		done <- true
	}()

	// Goroutine 2: Read metadata
	go func() {
		for i := 0; i < 100; i++ {
			target.GetMetadata("key1")
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state
	value, exists := target.GetMetadata("key1")
	if !exists {
		t.Error("Expected metadata to exist")
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got '%s'", value)
	}
}

// Test empty target metadata initialization
func TestEmptyTargetMetadata(t *testing.T) {
	target := Target{
		Type:     TargetTypeEmail,
		Value:    "test@example.com",
		Platform: "email",
		// Metadata is nil
	}

	// Test GetMetadata on nil metadata
	_, exists := target.GetMetadata("key")
	if exists {
		t.Error("Expected metadata key to not exist when metadata is nil")
	}

	// Test AddMetadata initializes metadata map
	target.AddMetadata("key", "value")
	value, exists := target.GetMetadata("key")
	if !exists {
		t.Error("Expected metadata key to exist after adding")
	}
	if value != "value" {
		t.Errorf("Expected value 'value', got '%s'", value)
	}
}
