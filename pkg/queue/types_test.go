package queue

import (
	"testing"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
)

func TestNewQueueMessage(t *testing.T) {
	// Create a notification message
	notifyMsg := message.New()
	notifyMsg.Title = "Test Message"
	notifyMsg.Body = "Test body"
	notifyMsg.Priority = message.PriorityHigh
	notifyMsg.SetMetadata("source", "test")

	// Convert to queue message
	queueMsg := NewQueueMessage(notifyMsg)

	// Verify basic fields
	if queueMsg.ID == "" {
		t.Error("Expected queue message ID to be set")
	}

	if queueMsg.Payload != notifyMsg {
		t.Error("Expected payload to be the notification message")
	}

	if queueMsg.Priority != PriorityHigh {
		t.Errorf("Expected priority to be %v, got %v", PriorityHigh, queueMsg.Priority)
	}

	if queueMsg.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if queueMsg.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify metadata is copied
	if queueMsg.Metadata["source"] != "test" {
		t.Errorf("Expected metadata source to be 'test', got %v", queueMsg.Metadata["source"])
	}
}

func TestNewQueueMessageWithScheduling(t *testing.T) {
	// Create a scheduled notification message
	future := time.Now().Add(1 * time.Hour)
	notifyMsg := message.New()
	notifyMsg.ScheduledAt = &future

	// Convert to queue message
	queueMsg := NewQueueMessage(notifyMsg)

	// Verify scheduled time is preserved
	if queueMsg.ScheduledAt == nil {
		t.Error("Expected ScheduledAt to be set")
	}

	if !queueMsg.ScheduledAt.Equal(future) {
		t.Errorf("Expected ScheduledAt to be %v, got %v", future, *queueMsg.ScheduledAt)
	}
}

func TestNewQueueMessageWithPayload(t *testing.T) {
	payload := map[string]interface{}{
		"type": "webhook",
		"url":  "https://example.com/webhook",
	}

	queueMsg := NewQueueMessageWithPayload(payload, PriorityNormal)

	// Verify basic fields
	if queueMsg.ID == "" {
		t.Error("Expected queue message ID to be set")
	}

	// For maps and other complex types, check that payload was set (interface{} comparison)
	if queueMsg.Payload == nil {
		t.Error("Expected payload to be set")
	}

	if queueMsg.Priority != PriorityNormal {
		t.Errorf("Expected priority to be %v, got %v", PriorityNormal, queueMsg.Priority)
	}

	if queueMsg.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if queueMsg.Metadata == nil {
		t.Error("Expected Metadata to be initialized")
	}
}

func TestQueueMessageGetNotificationMessage(t *testing.T) {
	// Test with notification message payload
	notifyMsg := message.New()
	notifyMsg.Title = "Test Message"

	queueMsg := NewQueueMessage(notifyMsg)

	extractedMsg, ok := queueMsg.GetNotificationMessage()
	if !ok {
		t.Error("Expected GetNotificationMessage to return true for notification message payload")
	}

	if extractedMsg != notifyMsg {
		t.Error("Expected extracted message to be the same as original")
	}

	// Test with non-notification payload
	otherPayload := "some string payload"
	queueMsg = NewQueueMessageWithPayload(otherPayload, PriorityLow)

	extractedMsg, ok = queueMsg.GetNotificationMessage()
	if ok {
		t.Error("Expected GetNotificationMessage to return false for non-notification payload")
	}

	if extractedMsg != nil {
		t.Error("Expected extracted message to be nil for non-notification payload")
	}
}

func TestConvertPriority(t *testing.T) {
	tests := []struct {
		notifyPriority message.Priority
		queuePriority  Priority
	}{
		{message.PriorityLow, PriorityLow},
		{message.PriorityNormal, PriorityNormal},
		{message.PriorityHigh, PriorityHigh},
		{message.PriorityUrgent, PriorityUrgent},
		{message.Priority(99), PriorityNormal}, // Invalid priority defaults to normal
	}

	for _, tt := range tests {
		t.Run(tt.notifyPriority.String(), func(t *testing.T) {
			result := convertPriority(tt.notifyPriority)
			if result != tt.queuePriority {
				t.Errorf("Expected %v, got %v", tt.queuePriority, result)
			}
		})
	}
}

func TestQueueMessageRetryHandling(t *testing.T) {
	notifyMsg := message.New()
	queueMsg := NewQueueMessage(notifyMsg)

	// Test initial retry count
	if queueMsg.RetryCount != 0 {
		t.Errorf("Expected initial retry count to be 0, got %d", queueMsg.RetryCount)
	}

	// Simulate retry
	queueMsg.RetryCount++
	queueMsg.UpdatedAt = time.Now()

	if queueMsg.RetryCount != 1 {
		t.Errorf("Expected retry count to be 1 after increment, got %d", queueMsg.RetryCount)
	}
}

func TestQueueMessageMetadataHandling(t *testing.T) {
	notifyMsg := message.New()
	notifyMsg.SetMetadata("original", "value1")

	queueMsg := NewQueueMessage(notifyMsg)

	// Verify metadata is copied from notification message
	if queueMsg.Metadata["original"] != "value1" {
		t.Errorf("Expected metadata 'original' to be 'value1', got %v", queueMsg.Metadata["original"])
	}

	// Test that queue message metadata can be modified independently
	queueMsg.Metadata["queue_specific"] = "value2"

	// Verify notification message metadata wasn't affected
	if notifyMsg.Metadata["queue_specific"] != nil {
		t.Error("Expected notification message metadata to be unaffected by queue message changes")
	}
}

func TestPriorityConstants(t *testing.T) {
	// Test that priority constants are in correct order
	if PriorityLow >= PriorityNormal {
		t.Error("Expected PriorityLow to be less than PriorityNormal")
	}

	if PriorityNormal >= PriorityHigh {
		t.Error("Expected PriorityNormal to be less than PriorityHigh")
	}

	if PriorityHigh >= PriorityUrgent {
		t.Error("Expected PriorityHigh to be less than PriorityUrgent")
	}
}

func TestQueueMessageTimestamps(t *testing.T) {
	before := time.Now()
	notifyMsg := message.New()
	queueMsg := NewQueueMessage(notifyMsg)
	after := time.Now()

	// Check that timestamps are within expected range
	if queueMsg.Timestamp.Before(before) || queueMsg.Timestamp.After(after) {
		t.Errorf("Expected Timestamp to be between %v and %v, got %v", before, after, queueMsg.Timestamp)
	}

	if queueMsg.CreatedAt.Before(before) || queueMsg.CreatedAt.After(after) {
		t.Errorf("Expected CreatedAt to be between %v and %v, got %v", before, after, queueMsg.CreatedAt)
	}

	if queueMsg.UpdatedAt.Before(before) || queueMsg.UpdatedAt.After(after) {
		t.Errorf("Expected UpdatedAt to be between %v and %v, got %v", before, after, queueMsg.UpdatedAt)
	}
}

func TestCompleteMessageConversionFlow(t *testing.T) {
	// Create a comprehensive notification message
	notifyMsg := message.NewMessage("Test Notification").
		WithBody("This is a test notification").
		WithFormat(message.FormatMarkdown).
		WithPriority(message.PriorityHigh).
		WithMetadata("source", "unit_test").
		WithVariable("user", "testuser").
		ToEmail("test@example.com").
		ToFeishuUser("user123").
		Build()

	// Convert to queue message
	queueMsg := NewQueueMessage(notifyMsg)

	// Verify the conversion preserved all important data
	extractedMsg, ok := queueMsg.GetNotificationMessage()
	if !ok {
		t.Fatal("Failed to extract notification message from queue message")
	}

	// Verify all fields are preserved
	if extractedMsg.Title != "Test Notification" {
		t.Errorf("Expected title to be preserved, got %s", extractedMsg.Title)
	}

	if extractedMsg.Body != "This is a test notification" {
		t.Errorf("Expected body to be preserved, got %s", extractedMsg.Body)
	}

	if extractedMsg.Format != message.FormatMarkdown {
		t.Errorf("Expected format to be preserved, got %v", extractedMsg.Format)
	}

	if extractedMsg.Priority != message.PriorityHigh {
		t.Errorf("Expected priority to be preserved, got %v", extractedMsg.Priority)
	}

	if len(extractedMsg.Targets) != 2 {
		t.Errorf("Expected 2 targets to be preserved, got %d", len(extractedMsg.Targets))
	}

	if extractedMsg.Metadata["source"] != "unit_test" {
		t.Errorf("Expected metadata to be preserved, got %v", extractedMsg.Metadata["source"])
	}

	if extractedMsg.Variables["user"] != "testuser" {
		t.Errorf("Expected variables to be preserved, got %v", extractedMsg.Variables["user"])
	}

	// Verify queue-specific fields
	if queueMsg.Priority != PriorityHigh {
		t.Errorf("Expected queue priority to be converted correctly, got %v", queueMsg.Priority)
	}
}
