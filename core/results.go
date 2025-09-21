package core

import (
	"time"
)

// SendingResults represents the results of sending messages to multiple targets
type SendingResults struct {
	MessageID string        `json:"message_id"`
	Results   []*Result     `json:"results"`
	Success   int           `json:"success"`
	Failed    int           `json:"failed"`
	Total     int           `json:"total"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
}

// NewSendingResults creates a new SendingResults instance
func NewSendingResults() *SendingResults {
	now := time.Now()
	return &SendingResults{
		Results:   make([]*Result, 0),
		StartTime: now,
		EndTime:   now,
		Duration:  0,
	}
}

// AddResult adds a result to the collection
func (sr *SendingResults) AddResult(result *Result) {
	sr.Results = append(sr.Results, result)
	sr.Total = len(sr.Results)

	if result.Success {
		sr.Success++
	} else {
		sr.Failed++
	}

	// Update timing
	sr.EndTime = time.Now()
	sr.Duration = sr.EndTime.Sub(sr.StartTime)
}

// IsSuccess returns true if all results were successful
func (sr *SendingResults) IsSuccess() bool {
	return sr.Failed == 0 && sr.Total > 0
}

// HasFailures returns true if there were any failures
func (sr *SendingResults) HasFailures() bool {
	return sr.Failed > 0
}

// GetSuccessfulResults returns only successful results
func (sr *SendingResults) GetSuccessfulResults() []*Result {
	successful := make([]*Result, 0, sr.Success)
	for _, result := range sr.Results {
		if result.Success {
			successful = append(successful, result)
		}
	}
	return successful
}

// GetFailedResults returns only failed results
func (sr *SendingResults) GetFailedResults() []*Result {
	failed := make([]*Result, 0, sr.Failed)
	for _, result := range sr.Results {
		if !result.Success {
			failed = append(failed, result)
		}
	}
	return failed
}

// GetResultsByPlatform returns results grouped by platform
func (sr *SendingResults) GetResultsByPlatform() map[string][]*Result {
	byPlatform := make(map[string][]*Result)
	for _, result := range sr.Results {
		platform := result.Platform
		byPlatform[platform] = append(byPlatform[platform], result)
	}
	return byPlatform
}

// GetResultsByStatus returns results grouped by status
func (sr *SendingResults) GetResultsByStatus() map[Status][]*Result {
	byStatus := make(map[Status][]*Result)
	for _, result := range sr.Results {
		status := result.Status
		byStatus[status] = append(byStatus[status], result)
	}
	return byStatus
}
