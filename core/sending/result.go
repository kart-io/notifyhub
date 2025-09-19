package sending

import (
	"time"
)

// Status represents the sending status
type Status string

const (
	StatusPending   Status = "pending"
	StatusSending   Status = "sending"
	StatusSent      Status = "sent"
	StatusFailed    Status = "failed"
	StatusRetrying  Status = "retrying"
	StatusCancelled Status = "cancelled"
)

// Result represents the result of a message sending operation
type Result struct {
	MessageID    string            `json:"message_id"`
	Target       Target            `json:"target"`
	Platform     string            `json:"platform"`
	Status       Status            `json:"status"`
	Success      bool              `json:"success"`
	Error        error             `json:"error,omitempty"`
	Response     interface{}       `json:"response,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	AttemptCount int               `json:"attempt_count"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	SentAt       *time.Time        `json:"sent_at,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	Duration     time.Duration     `json:"duration"`
}

// NewResult creates a new sending result
func NewResult(messageID string, target Target) *Result {
	now := time.Now()
	return &Result{
		MessageID:    messageID,
		Target:       target,
		Status:       StatusPending,
		Metadata:     make(map[string]string),
		AttemptCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
		Timestamp:    now,
		Duration:     0,
	}
}

// SetStatus updates the result status
func (r *Result) SetStatus(status Status) *Result {
	r.Status = status
	r.UpdatedAt = time.Now()
	if status == StatusSent && r.SentAt == nil {
		now := time.Now()
		r.SentAt = &now
	}
	return r
}

// SetError sets the error and updates status to failed
func (r *Result) SetError(err error) *Result {
	r.Error = err
	r.Status = StatusFailed
	r.UpdatedAt = time.Now()
	return r
}

// SetResponse sets the platform response
func (r *Result) SetResponse(response interface{}) *Result {
	r.Response = response
	r.UpdatedAt = time.Now()
	return r
}

// AddMetadata adds metadata to the result
func (r *Result) AddMetadata(key, value string) *Result {
	if r.Metadata == nil {
		r.Metadata = make(map[string]string)
	}
	r.Metadata[key] = value
	r.UpdatedAt = time.Now()
	return r
}

// IncrementAttempt increments the attempt counter
func (r *Result) IncrementAttempt() *Result {
	r.AttemptCount++
	r.UpdatedAt = time.Now()
	return r
}

// IsSuccess returns true if the message was sent successfully
func (r *Result) IsSuccess() bool {
	return r.Status == StatusSent
}

// IsFailed returns true if the message sending failed
func (r *Result) IsFailed() bool {
	return r.Status == StatusFailed
}

// IsPending returns true if the message is pending
func (r *Result) IsPending() bool {
	return r.Status == StatusPending
}

// IsRetrying returns true if the message is being retried
func (r *Result) IsRetrying() bool {
	return r.Status == StatusRetrying
}

// SendingResults represents a collection of sending results
type SendingResults struct {
	Results      []*Result     `json:"results"`
	Total        int           `json:"total"`
	Success      int           `json:"success"`
	Failed       int           `json:"failed"`
	Pending      int           `json:"pending"`
	SuccessCount int           `json:"success_count"`
	FailedCount  int           `json:"failed_count"`
	TotalCount   int           `json:"total_count"`
	Duration     time.Duration `json:"duration"`
	CreatedAt    time.Time     `json:"created_at"`
}

// NewSendingResults creates a new sending results collection
func NewSendingResults() *SendingResults {
	return &SendingResults{
		Results:   make([]*Result, 0),
		CreatedAt: time.Now(),
	}
}

// AddResult adds a result to the collection
func (sr *SendingResults) AddResult(result *Result) {
	sr.Results = append(sr.Results, result)
	sr.updateCounts()
}

// updateCounts updates the count statistics
func (sr *SendingResults) updateCounts() {
	sr.Total = len(sr.Results)
	sr.Success = 0
	sr.Failed = 0
	sr.Pending = 0

	for _, result := range sr.Results {
		switch result.Status {
		case StatusSent:
			sr.Success++
		case StatusFailed:
			sr.Failed++
		case StatusPending, StatusSending, StatusRetrying:
			sr.Pending++
		}
	}
}

// GetSuccessResults returns all successful results
func (sr *SendingResults) GetSuccessResults() []*Result {
	var results []*Result
	for _, result := range sr.Results {
		if result.IsSuccess() {
			results = append(results, result)
		}
	}
	return results
}

// GetFailedResults returns all failed results
func (sr *SendingResults) GetFailedResults() []*Result {
	var results []*Result
	for _, result := range sr.Results {
		if result.IsFailed() {
			results = append(results, result)
		}
	}
	return results
}

// HasErrors returns true if any result has an error
func (sr *SendingResults) HasErrors() bool {
	return sr.Failed > 0
}

// IsCompleteSuccess returns true if all results are successful
func (sr *SendingResults) IsCompleteSuccess() bool {
	return sr.Total > 0 && sr.Success == sr.Total
}
