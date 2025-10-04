// Package receipt provides message receipt structures and processing for NotifyHub
package receipt

import "time"

// Receipt represents a message delivery receipt
type Receipt struct {
	MessageID  string           `json:"message_id"`
	Status     string           `json:"status"`
	Results    []PlatformResult `json:"results"`
	Successful int              `json:"successful"`
	Failed     int              `json:"failed"`
	Total      int              `json:"total"`
	Timestamp  time.Time        `json:"timestamp"`
}

// PlatformResult represents the result of sending to a specific platform
type PlatformResult struct {
	Platform  string    `json:"platform"`
	Target    string    `json:"target"`
	Success   bool      `json:"success"`
	MessageID string    `json:"message_id,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Status constants
const (
	StatusSuccess    = "success"
	StatusPartial    = "partial"
	StatusFailed     = "failed"
	StatusPending    = "pending"
	StatusProcessing = "processing"
)

// New creates a new receipt
func New(messageID string) *Receipt {
	return &Receipt{
		MessageID: messageID,
		Status:    StatusPending,
		Results:   make([]PlatformResult, 0),
		Timestamp: time.Now(),
	}
}

// AddResult adds a platform result to the receipt
func (r *Receipt) AddResult(result PlatformResult) {
	r.Results = append(r.Results, result)
	r.Total = len(r.Results)

	// Update counters
	r.Successful = 0
	r.Failed = 0
	for _, res := range r.Results {
		if res.Success {
			r.Successful++
		} else {
			r.Failed++
		}
	}

	// Update overall status
	r.updateStatus()
}

// updateStatus updates the overall receipt status based on results
func (r *Receipt) updateStatus() {
	if r.Total == 0 {
		r.Status = StatusPending
		return
	}

	if r.Failed == 0 {
		r.Status = StatusSuccess
	} else if r.Successful == 0 {
		r.Status = StatusFailed
	} else {
		r.Status = StatusPartial
	}
}

// IsComplete returns true if all results have been received
func (r *Receipt) IsComplete() bool {
	return r.Status != StatusPending && r.Status != StatusProcessing
}

// IsSuccess returns true if all deliveries were successful
func (r *Receipt) IsSuccess() bool {
	return r.Status == StatusSuccess
}

// IsPartial returns true if some deliveries were successful
func (r *Receipt) IsPartial() bool {
	return r.Status == StatusPartial
}

// IsFailed returns true if all deliveries failed
func (r *Receipt) IsFailed() bool {
	return r.Status == StatusFailed
}

// GetSuccessRate returns the success rate as a percentage
func (r *Receipt) GetSuccessRate() float64 {
	if r.Total == 0 {
		return 0.0
	}
	return float64(r.Successful) / float64(r.Total) * 100
}

// GetErrors returns all error messages from failed results
func (r *Receipt) GetErrors() []string {
	errors := make([]string, 0, r.Failed)
	for _, result := range r.Results {
		if !result.Success && result.Error != "" {
			errors = append(errors, result.Error)
		}
	}
	return errors
}

// GetSuccessfulPlatforms returns the names of platforms that succeeded
func (r *Receipt) GetSuccessfulPlatforms() []string {
	platforms := make([]string, 0, r.Successful)
	for _, result := range r.Results {
		if result.Success {
			platforms = append(platforms, result.Platform)
		}
	}
	return platforms
}

// GetFailedPlatforms returns the names of platforms that failed
func (r *Receipt) GetFailedPlatforms() []string {
	platforms := make([]string, 0, r.Failed)
	for _, result := range r.Results {
		if !result.Success {
			platforms = append(platforms, result.Platform)
		}
	}
	return platforms
}
