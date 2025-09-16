package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/kart-io/notifyhub/notifiers"
)

// ResultAnalyzer provides comprehensive analysis of send results
type ResultAnalyzer struct {
	results []*notifiers.SendResult
}

// AnalyzeResults creates a new result analyzer from send results
func AnalyzeResults(results []*notifiers.SendResult) *ResultAnalyzer {
	return &ResultAnalyzer{
		results: results,
	}
}

// Summary returns a summary of the results
type ResultSummary struct {
	Total           int                 `json:"total"`
	Successful      int                 `json:"successful"`
	Failed          int                 `json:"failed"`
	SuccessRate     float64             `json:"success_rate"`
	TotalDuration   time.Duration       `json:"total_duration"`
	AverageDuration time.Duration       `json:"average_duration"`
	FastestPlatform string              `json:"fastest_platform"`
	SlowestPlatform string              `json:"slowest_platform"`
	Platforms       map[string]bool     `json:"platforms"`
	Errors          map[string][]string `json:"errors"`
}

// Summary returns a comprehensive summary of all results
func (ra *ResultAnalyzer) Summary() *ResultSummary {
	if len(ra.results) == 0 {
		return &ResultSummary{
			Platforms: make(map[string]bool),
			Errors:    make(map[string][]string),
		}
	}

	summary := &ResultSummary{
		Total:     len(ra.results),
		Platforms: make(map[string]bool),
		Errors:    make(map[string][]string),
	}

	var totalDuration time.Duration
	var fastest, slowest time.Duration
	var fastestPlatform, slowestPlatform string

	for i, result := range ra.results {
		// Track platforms
		summary.Platforms[result.Platform] = true

		// Count success/failure
		if result.Success {
			summary.Successful++
		} else {
			summary.Failed++
			if result.Error != nil {
				summary.Errors[result.Platform] = append(summary.Errors[result.Platform], result.Error.Error())
			}
		}

		// Duration analysis
		totalDuration += result.Duration
		if i == 0 || result.Duration < fastest {
			fastest = result.Duration
			fastestPlatform = result.Platform
		}
		if i == 0 || result.Duration > slowest {
			slowest = result.Duration
			slowestPlatform = result.Platform
		}
	}

	summary.SuccessRate = float64(summary.Successful) / float64(summary.Total)
	summary.TotalDuration = totalDuration
	summary.AverageDuration = totalDuration / time.Duration(summary.Total)
	summary.FastestPlatform = fastestPlatform
	summary.SlowestPlatform = slowestPlatform

	return summary
}

// HasFailures returns true if any send operation failed
func (ra *ResultAnalyzer) HasFailures() bool {
	for _, result := range ra.results {
		if !result.Success {
			return true
		}
	}
	return false
}

// HasSuccesses returns true if any send operation succeeded
func (ra *ResultAnalyzer) HasSuccesses() bool {
	for _, result := range ra.results {
		if result.Success {
			return true
		}
	}
	return false
}

// SuccessRate returns the success rate as a percentage (0-100)
func (ra *ResultAnalyzer) SuccessRate() float64 {
	if len(ra.results) == 0 {
		return 0
	}
	successful := 0
	for _, result := range ra.results {
		if result.Success {
			successful++
		}
	}
	return float64(successful) / float64(len(ra.results)) * 100
}

// FailedPlatforms returns a list of platforms that failed
func (ra *ResultAnalyzer) FailedPlatforms() []string {
	platforms := make(map[string]bool)
	for _, result := range ra.results {
		if !result.Success {
			platforms[result.Platform] = true
		}
	}

	var failed []string
	for platform := range platforms {
		failed = append(failed, platform)
	}
	return failed
}

// SuccessfulPlatforms returns a list of platforms that succeeded
func (ra *ResultAnalyzer) SuccessfulPlatforms() []string {
	platforms := make(map[string]bool)
	for _, result := range ra.results {
		if result.Success {
			platforms[result.Platform] = true
		}
	}

	var successful []string
	for platform := range platforms {
		successful = append(successful, platform)
	}
	return successful
}

// ErrorsByPlatform returns errors grouped by platform
func (ra *ResultAnalyzer) ErrorsByPlatform() map[string][]string {
	errors := make(map[string][]string)
	for _, result := range ra.results {
		if !result.Success && result.Error != nil {
			errors[result.Platform] = append(errors[result.Platform], result.Error.Error())
		}
	}
	return errors
}

// AllErrors returns all error messages
func (ra *ResultAnalyzer) AllErrors() []string {
	var errors []string
	for _, result := range ra.results {
		if !result.Success && result.Error != nil {
			errors = append(errors, fmt.Sprintf("[%s] %s", result.Platform, result.Error.Error()))
		}
	}
	return errors
}

// FastestResult returns the result with the shortest duration
func (ra *ResultAnalyzer) FastestResult() *notifiers.SendResult {
	if len(ra.results) == 0 {
		return nil
	}

	fastest := ra.results[0]
	for _, result := range ra.results[1:] {
		if result.Duration < fastest.Duration {
			fastest = result
		}
	}
	return fastest
}

// SlowestResult returns the result with the longest duration
func (ra *ResultAnalyzer) SlowestResult() *notifiers.SendResult {
	if len(ra.results) == 0 {
		return nil
	}

	slowest := ra.results[0]
	for _, result := range ra.results[1:] {
		if result.Duration > slowest.Duration {
			slowest = result
		}
	}
	return slowest
}

// AverageDuration returns the average duration across all results
func (ra *ResultAnalyzer) AverageDuration() time.Duration {
	if len(ra.results) == 0 {
		return 0
	}

	var total time.Duration
	for _, result := range ra.results {
		total += result.Duration
	}
	return total / time.Duration(len(ra.results))
}

// FilterBySuccess returns results filtered by success status
func (ra *ResultAnalyzer) FilterBySuccess(success bool) []*notifiers.SendResult {
	var filtered []*notifiers.SendResult
	for _, result := range ra.results {
		if result.Success == success {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// FilterByPlatform returns results filtered by platform
func (ra *ResultAnalyzer) FilterByPlatform(platform string) []*notifiers.SendResult {
	var filtered []*notifiers.SendResult
	for _, result := range ra.results {
		if result.Platform == platform {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// String returns a human-readable summary
func (ra *ResultAnalyzer) String() string {
	summary := ra.Summary()
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📊 Results Summary:\n"))
	sb.WriteString(fmt.Sprintf("   Total: %d\n", summary.Total))
	sb.WriteString(fmt.Sprintf("   Successful: %d\n", summary.Successful))
	sb.WriteString(fmt.Sprintf("   Failed: %d\n", summary.Failed))
	sb.WriteString(fmt.Sprintf("   Success Rate: %.1f%%\n", summary.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("   Average Duration: %v\n", summary.AverageDuration))

	if summary.FastestPlatform != "" {
		sb.WriteString(fmt.Sprintf("   Fastest: %s\n", summary.FastestPlatform))
	}
	if summary.SlowestPlatform != "" {
		sb.WriteString(fmt.Sprintf("   Slowest: %s\n", summary.SlowestPlatform))
	}

	if len(summary.Errors) > 0 {
		sb.WriteString("   Errors:\n")
		for platform, errors := range summary.Errors {
			sb.WriteString(fmt.Sprintf("     %s: %s\n", platform, strings.Join(errors, ", ")))
		}
	}

	return sb.String()
}

// DetailedReport returns a detailed analysis report
func (ra *ResultAnalyzer) DetailedReport() string {
	var sb strings.Builder

	sb.WriteString("📋 Detailed Send Results Report\n")
	sb.WriteString("================================\n\n")

	summary := ra.Summary()
	sb.WriteString(fmt.Sprintf("Overall Statistics:\n"))
	sb.WriteString(fmt.Sprintf("  • Total operations: %d\n", summary.Total))
	sb.WriteString(fmt.Sprintf("  • Successful: %d (%.1f%%)\n", summary.Successful, summary.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("  • Failed: %d (%.1f%%)\n", summary.Failed, (1-summary.SuccessRate)*100))
	sb.WriteString(fmt.Sprintf("  • Total duration: %v\n", summary.TotalDuration))
	sb.WriteString(fmt.Sprintf("  • Average duration: %v\n", summary.AverageDuration))
	sb.WriteString("\n")

	// Platform breakdown
	sb.WriteString("Platform Breakdown:\n")
	platformStats := make(map[string]struct {
		success int
		failed  int
		total   int
	})

	for _, result := range ra.results {
		stats := platformStats[result.Platform]
		stats.total++
		if result.Success {
			stats.success++
		} else {
			stats.failed++
		}
		platformStats[result.Platform] = stats
	}

	for platform, stats := range platformStats {
		successRate := float64(stats.success) / float64(stats.total) * 100
		sb.WriteString(fmt.Sprintf("  • %s: %d/%d success (%.1f%%)\n",
			platform, stats.success, stats.total, successRate))
	}
	sb.WriteString("\n")

	// Performance analysis
	if len(ra.results) > 0 {
		fastest := ra.FastestResult()
		slowest := ra.SlowestResult()
		sb.WriteString("Performance Analysis:\n")
		sb.WriteString(fmt.Sprintf("  • Fastest: %s (%v)\n", fastest.Platform, fastest.Duration))
		sb.WriteString(fmt.Sprintf("  • Slowest: %s (%v)\n", slowest.Platform, slowest.Duration))
		sb.WriteString("\n")
	}

	// Error details
	if ra.HasFailures() {
		sb.WriteString("Error Details:\n")
		for platform, errors := range summary.Errors {
			sb.WriteString(fmt.Sprintf("  • %s:\n", platform))
			for _, err := range errors {
				sb.WriteString(fmt.Sprintf("    - %s\n", err))
			}
		}
	}

	return sb.String()
}

// IsComplete returns true if all operations completed (regardless of success)
func (ra *ResultAnalyzer) IsComplete() bool {
	return len(ra.results) > 0
}

// IsPartialSuccess returns true if some but not all operations succeeded
func (ra *ResultAnalyzer) IsPartialSuccess() bool {
	return ra.HasSuccesses() && ra.HasFailures()
}

// IsFullSuccess returns true if all operations succeeded
func (ra *ResultAnalyzer) IsFullSuccess() bool {
	return ra.HasSuccesses() && !ra.HasFailures()
}

// IsFullFailure returns true if all operations failed
func (ra *ResultAnalyzer) IsFullFailure() bool {
	return ra.HasFailures() && !ra.HasSuccesses()
}

// Recommendations returns suggestions based on the results
func (ra *ResultAnalyzer) Recommendations() []string {
	var recommendations []string

	if ra.IsFullFailure() {
		recommendations = append(recommendations, "All sends failed - check service configuration and network connectivity")
	} else if ra.IsPartialSuccess() {
		recommendations = append(recommendations, "Partial failure detected - review failed platforms for issues")

		failed := ra.FailedPlatforms()
		if len(failed) > 0 {
			recommendations = append(recommendations, fmt.Sprintf("Focus on fixing: %s", strings.Join(failed, ", ")))
		}
	} else if ra.IsFullSuccess() {
		if ra.AverageDuration() > 5*time.Second {
			recommendations = append(recommendations, "All sends succeeded but consider optimizing for better performance")
		}
	}

	// Performance recommendations
	avg := ra.AverageDuration()
	if avg > 10*time.Second {
		recommendations = append(recommendations, "Average send time is high - consider async sending or performance optimization")
	}

	successRate := ra.SuccessRate()
	if successRate < 95 && successRate > 0 {
		recommendations = append(recommendations, "Success rate below 95% - implement retry logic for better reliability")
	}

	return recommendations
}
