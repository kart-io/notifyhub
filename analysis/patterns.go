package analysis

import (
	"strings"

	"github.com/kart-io/notifyhub/core/sending"
)

// PatternDetector detects patterns in sending results
type PatternDetector struct {
	patterns []Pattern
}

// platformStats holds statistics for a platform
type platformStats struct {
	total   int
	success int
	failed  int
	pending int
	errors  map[string]int
}

// Pattern represents a detected pattern
type Pattern struct {
	Type        string                 `json:"type"`
	Severity    Severity               `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector() *PatternDetector {
	return &PatternDetector{
		patterns: make([]Pattern, 0),
	}
}

// DetectPatterns detects patterns in sending results
func (pd *PatternDetector) DetectPatterns(results *sending.SendingResults) []Pattern {
	var patterns []Pattern

	// Detect timeout patterns
	if pattern := pd.detectTimeoutPattern(results); pattern != nil {
		patterns = append(patterns, *pattern)
	}

	// Detect rate limiting patterns
	if pattern := pd.detectRateLimitPattern(results); pattern != nil {
		patterns = append(patterns, *pattern)
	}

	// Detect authentication patterns
	if pattern := pd.detectAuthPattern(results); pattern != nil {
		patterns = append(patterns, *pattern)
	}

	// Detect network patterns
	if pattern := pd.detectNetworkPattern(results); pattern != nil {
		patterns = append(patterns, *pattern)
	}

	// Detect platform-specific patterns
	if platformPatterns := pd.detectPlatformPatterns(results); len(platformPatterns) > 0 {
		patterns = append(patterns, platformPatterns...)
	}

	return patterns
}

// detectTimeoutPattern detects timeout-related failures
func (pd *PatternDetector) detectTimeoutPattern(results *sending.SendingResults) *Pattern {
	timeoutCount := 0
	totalFailed := 0

	for _, result := range results.Results {
		if result.IsFailed() {
			totalFailed++
			if result.Error != nil && strings.Contains(result.Error.Error(), "timeout") {
				timeoutCount++
			}
		}
	}

	if totalFailed > 0 && float64(timeoutCount)/float64(totalFailed) > 0.5 {
		return &Pattern{
			Type:        "timeout",
			Severity:    SeverityWarning,
			Title:       "High Timeout Rate",
			Description: "More than 50% of failures are due to timeouts",
			Metadata: map[string]interface{}{
				"timeout_count": timeoutCount,
				"total_failed":  totalFailed,
				"timeout_rate":  float64(timeoutCount) / float64(totalFailed),
			},
		}
	}

	return nil
}

// detectRateLimitPattern detects rate limiting failures
func (pd *PatternDetector) detectRateLimitPattern(results *sending.SendingResults) *Pattern {
	rateLimitCount := 0

	for _, result := range results.Results {
		if result.IsFailed() && result.Error == sending.ErrRateLimited {
			rateLimitCount++
		}
	}

	if rateLimitCount > 0 {
		return &Pattern{
			Type:        "rate_limit",
			Severity:    SeverityInfo,
			Title:       "Rate Limiting Detected",
			Description: "Some messages were rate limited",
			Metadata: map[string]interface{}{
				"rate_limited_count": rateLimitCount,
				"total_results":      len(results.Results),
			},
		}
	}

	return nil
}

// detectAuthPattern detects authentication-related failures
func (pd *PatternDetector) detectAuthPattern(results *sending.SendingResults) *Pattern {
	authFailureCount := 0

	for _, result := range results.Results {
		if result.IsFailed() && result.Error == sending.ErrInvalidCredentials {
			authFailureCount++
		}
	}

	if authFailureCount > 0 {
		return &Pattern{
			Type:        "authentication",
			Severity:    SeverityError,
			Title:       "Authentication Failures",
			Description: "Some messages failed due to authentication issues",
			Metadata: map[string]interface{}{
				"auth_failure_count": authFailureCount,
				"total_results":      len(results.Results),
			},
		}
	}

	return nil
}

// detectNetworkPattern detects network-related failures
func (pd *PatternDetector) detectNetworkPattern(results *sending.SendingResults) *Pattern {
	networkFailureCount := 0

	for _, result := range results.Results {
		if result.IsFailed() && result.Error == sending.ErrNetworkError {
			networkFailureCount++
		}
	}

	if networkFailureCount > 0 {
		return &Pattern{
			Type:        "network",
			Severity:    SeverityWarning,
			Title:       "Network Issues Detected",
			Description: "Some messages failed due to network issues",
			Metadata: map[string]interface{}{
				"network_failure_count": networkFailureCount,
				"total_results":         len(results.Results),
			},
		}
	}

	return nil
}

// detectPlatformPatterns detects platform-specific patterns
func (pd *PatternDetector) detectPlatformPatterns(results *sending.SendingResults) []Pattern {
	var patterns []Pattern

	platformStatsMap := make(map[string]*platformStats)

	// Collect platform statistics
	for _, result := range results.Results {
		platform := result.Target.Platform
		if _, exists := platformStatsMap[platform]; !exists {
			newStats := &platformStats{
				errors: make(map[string]int),
			}
			platformStatsMap[platform] = newStats
		}

		stats := platformStatsMap[platform]
		stats.total++

		if result.IsSuccess() {
			stats.success++
		} else if result.IsFailed() {
			stats.failed++
		} else {
			stats.pending++
		}

		// Track error types
		if result.Error != nil {
			if stats.errors == nil {
				stats.errors = make(map[string]int)
			}
			errorType := result.Error.Error()
			stats.errors[errorType]++
		}
	}

	// Analyze each platform
	for platform, stats := range platformStatsMap {
		if pattern := pd.analyzePlatformStats(platform, stats); pattern != nil {
			patterns = append(patterns, *pattern)
		}
	}

	return patterns
}

// analyzePlatformStats analyzes statistics for a specific platform
func (pd *PatternDetector) analyzePlatformStats(platform string, stats *platformStats) *Pattern {
	if stats.total == 0 {
		return nil
	}

	failureRate := float64(stats.failed) / float64(stats.total)

	// High failure rate for platform
	if failureRate > 0.8 && stats.failed > 1 {
		return &Pattern{
			Type:        "platform_failure",
			Severity:    SeverityError,
			Title:       "Platform Failure Pattern",
			Description: "High failure rate detected for platform: " + platform,
			Metadata: map[string]interface{}{
				"platform":     platform,
				"failure_rate": failureRate,
				"total":        stats.total,
				"failed":       stats.failed,
				"errors":       stats.errors,
			},
		}
	}

	// Perfect success for platform
	if stats.success == stats.total && stats.total > 0 {
		return &Pattern{
			Type:        "platform_success",
			Severity:    SeverityInfo,
			Title:       "Platform Success Pattern",
			Description: "Perfect success rate for platform: " + platform,
			Metadata: map[string]interface{}{
				"platform": platform,
				"total":    stats.total,
				"success":  stats.success,
			},
		}
	}

	return nil
}
