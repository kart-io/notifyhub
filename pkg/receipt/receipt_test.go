package receipt

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	messageID := "msg-123"
	receipt := New(messageID)

	if receipt.MessageID != messageID {
		t.Errorf("MessageID = %v, want %v", receipt.MessageID, messageID)
	}
	if receipt.Status != StatusPending {
		t.Errorf("Status = %v, want %v", receipt.Status, StatusPending)
	}
	if len(receipt.Results) != 0 {
		t.Errorf("Results length = %v, want 0", len(receipt.Results))
	}
	if receipt.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestReceipt_AddResult(t *testing.T) {
	receipt := New("msg-123")

	// Add successful result
	result1 := PlatformResult{
		Platform:  "email",
		Target:    "test@example.com",
		Success:   true,
		MessageID: "email-msg-1",
		Timestamp: time.Now(),
	}
	receipt.AddResult(result1)

	if receipt.Total != 1 {
		t.Errorf("Total = %v, want 1", receipt.Total)
	}
	if receipt.Successful != 1 {
		t.Errorf("Successful = %v, want 1", receipt.Successful)
	}
	if receipt.Failed != 0 {
		t.Errorf("Failed = %v, want 0", receipt.Failed)
	}
	if receipt.Status != StatusSuccess {
		t.Errorf("Status = %v, want %v", receipt.Status, StatusSuccess)
	}

	// Add failed result
	result2 := PlatformResult{
		Platform:  "feishu",
		Target:    "user123",
		Success:   false,
		Error:     "connection failed",
		Timestamp: time.Now(),
	}
	receipt.AddResult(result2)

	if receipt.Total != 2 {
		t.Errorf("Total = %v, want 2", receipt.Total)
	}
	if receipt.Successful != 1 {
		t.Errorf("Successful = %v, want 1", receipt.Successful)
	}
	if receipt.Failed != 1 {
		t.Errorf("Failed = %v, want 1", receipt.Failed)
	}
	if receipt.Status != StatusPartial {
		t.Errorf("Status = %v, want %v", receipt.Status, StatusPartial)
	}
}

func TestReceipt_UpdateStatus(t *testing.T) {
	tests := []struct {
		name           string
		results        []PlatformResult
		expectedStatus string
	}{
		{
			name:           "no results",
			results:        []PlatformResult{},
			expectedStatus: StatusPending,
		},
		{
			name: "all successful",
			results: []PlatformResult{
				{Platform: "email", Success: true},
				{Platform: "feishu", Success: true},
			},
			expectedStatus: StatusSuccess,
		},
		{
			name: "all failed",
			results: []PlatformResult{
				{Platform: "email", Success: false},
				{Platform: "feishu", Success: false},
			},
			expectedStatus: StatusFailed,
		},
		{
			name: "partial success",
			results: []PlatformResult{
				{Platform: "email", Success: true},
				{Platform: "feishu", Success: false},
			},
			expectedStatus: StatusPartial,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipt := New("msg-123")
			for _, result := range tt.results {
				receipt.AddResult(result)
			}
			if receipt.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", receipt.Status, tt.expectedStatus)
			}
		})
	}
}

func TestReceipt_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		results  []PlatformResult
		expected bool
	}{
		{
			name:     "no results",
			results:  []PlatformResult{},
			expected: false,
		},
		{
			name: "all successful",
			results: []PlatformResult{
				{Platform: "email", Success: true},
			},
			expected: true,
		},
		{
			name: "some failed",
			results: []PlatformResult{
				{Platform: "email", Success: true},
				{Platform: "feishu", Success: false},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipt := New("msg-123")
			for _, result := range tt.results {
				receipt.AddResult(result)
			}
			got := receipt.IsSuccess()
			if got != tt.expected {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReceipt_IsPartial(t *testing.T) {
	tests := []struct {
		name     string
		results  []PlatformResult
		expected bool
	}{
		{
			name:     "no results",
			results:  []PlatformResult{},
			expected: false,
		},
		{
			name: "all successful",
			results: []PlatformResult{
				{Platform: "email", Success: true},
			},
			expected: false,
		},
		{
			name: "some failed",
			results: []PlatformResult{
				{Platform: "email", Success: true},
				{Platform: "feishu", Success: false},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipt := New("msg-123")
			for _, result := range tt.results {
				receipt.AddResult(result)
			}
			got := receipt.IsPartial()
			if got != tt.expected {
				t.Errorf("IsPartial() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReceipt_IsFailed(t *testing.T) {
	tests := []struct {
		name     string
		results  []PlatformResult
		expected bool
	}{
		{
			name:     "no results",
			results:  []PlatformResult{},
			expected: false,
		},
		{
			name: "all failed",
			results: []PlatformResult{
				{Platform: "email", Success: false},
				{Platform: "feishu", Success: false},
			},
			expected: true,
		},
		{
			name: "some successful",
			results: []PlatformResult{
				{Platform: "email", Success: true},
				{Platform: "feishu", Success: false},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipt := New("msg-123")
			for _, result := range tt.results {
				receipt.AddResult(result)
			}
			got := receipt.IsFailed()
			if got != tt.expected {
				t.Errorf("IsFailed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusSuccess != "success" {
		t.Errorf("StatusSuccess = %v, want success", StatusSuccess)
	}
	if StatusPartial != "partial" {
		t.Errorf("StatusPartial = %v, want partial", StatusPartial)
	}
	if StatusFailed != "failed" {
		t.Errorf("StatusFailed = %v, want failed", StatusFailed)
	}
	if StatusPending != "pending" {
		t.Errorf("StatusPending = %v, want pending", StatusPending)
	}
	if StatusProcessing != "processing" {
		t.Errorf("StatusProcessing = %v, want processing", StatusProcessing)
	}
}
