// Package receipt provides receipt management for NotifyHub
package receipt

import (
	"strings"
	"time"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// AggregationConfig defines how to aggregate multi-platform results
type AggregationConfig struct {
	PartialFailureThreshold float64  `json:"partial_failure_threshold"` // 0.0-1.0, threshold for partial failure
	RequiredPlatforms       []string `json:"required_platforms,omitempty"`
	FailureTolerance        int      `json:"failure_tolerance"` // Max allowed failures
}

// ResultAggregator handles multi-platform result aggregation and status calculation
type ResultAggregator struct {
	Config AggregationConfig
	logger logger.Logger
}

// NewResultAggregator creates a new result aggregator
func NewResultAggregator(config AggregationConfig, logger logger.Logger) *ResultAggregator {
	return &ResultAggregator{
		Config: config,
		logger: logger,
	}
}

// AggregateMultiPlatformResults performs enhanced multi-platform result aggregation
func (a *ResultAggregator) AggregateMultiPlatformResults(receipt *Receipt) *Receipt {
	// Create a copy to avoid modifying original
	enhanced := &Receipt{
		MessageID: receipt.MessageID,
		Results:   make([]PlatformResult, len(receipt.Results)),
		Timestamp: receipt.Timestamp,
		Error:     receipt.Error,
	}
	copy(enhanced.Results, receipt.Results)

	// Group results by platform
	platformResults := make(map[string][]PlatformResult)
	for _, result := range enhanced.Results {
		platformResults[result.Platform] = append(platformResults[result.Platform], result)
	}

	// Aggregate counts
	var totalTargets, successfulTargets, failedTargets int
	platformStats := make(map[string]struct {
		Successful int
		Failed     int
		Total      int
	})

	for platform, results := range platformResults {
		stats := struct {
			Successful int
			Failed     int
			Total      int
		}{}

		for _, result := range results {
			stats.Total++
			if result.Success {
				stats.Successful++
				successfulTargets++
			} else {
				stats.Failed++
				failedTargets++
			}
			totalTargets++
		}

		platformStats[platform] = stats
	}

	enhanced.Total = totalTargets
	enhanced.Successful = successfulTargets
	enhanced.Failed = failedTargets

	a.logger.Debug("Aggregated multi-platform results",
		"message_id", enhanced.MessageID,
		"platforms", len(platformStats),
		"total_targets", totalTargets,
		"successful", successfulTargets,
		"failed", failedTargets)

	return enhanced
}

// CalculateOverallStatus determines the overall status based on partial failure logic
func (a *ResultAggregator) CalculateOverallStatus(receipt *Receipt) string {
	if receipt.Total == 0 {
		return string(StatusFailed)
	}

	successRate := float64(receipt.Successful) / float64(receipt.Total)

	// Check for complete success
	if receipt.Failed == 0 {
		return string(StatusCompleted)
	}

	// Check for complete failure
	if receipt.Successful == 0 {
		return string(StatusFailed)
	}

	// Check failure tolerance first
	if a.Config.FailureTolerance > 0 && receipt.Failed <= a.Config.FailureTolerance {
		return string(StatusCompleted)
	}

	// Check required platforms
	if len(a.Config.RequiredPlatforms) > 0 {
		requiredPlatformsOk := a.CheckRequiredPlatforms(receipt)
		if !requiredPlatformsOk {
			return string(StatusFailed)
		}
	}

	// Check for partial failure based on threshold
	if successRate >= a.Config.PartialFailureThreshold {
		return string(StatusPartialFailed)
	}

	// Below threshold - complete failure
	return string(StatusFailed)
}

// CheckRequiredPlatforms verifies that required platforms succeeded
func (a *ResultAggregator) CheckRequiredPlatforms(receipt *Receipt) bool {
	platformSuccess := make(map[string]bool)
	for _, result := range receipt.Results {
		if _, exists := platformSuccess[result.Platform]; !exists {
			platformSuccess[result.Platform] = true
		}
		if !result.Success {
			platformSuccess[result.Platform] = false
		}
	}

	for _, requiredPlatform := range a.Config.RequiredPlatforms {
		if success, exists := platformSuccess[requiredPlatform]; !exists || !success {
			return false
		}
	}

	return true
}

// AnalyzeFailurePatterns analyzes failure patterns in receipts
func (a *ResultAggregator) AnalyzeFailurePatterns(receipts []*Receipt) map[string]interface{} {
	failuresByPlatform := make(map[string]int)
	failuresByError := make(map[string]int)
	totalFailures := 0
	totalReceipts := len(receipts)

	for _, receipt := range receipts {
		if receipt.Failed > 0 {
			totalFailures++

			// Analyze by platform
			for _, result := range receipt.Results {
				if !result.Success {
					failuresByPlatform[result.Platform]++
					if result.Error != "" {
						// Group similar errors
						errorKey := a.categorizeError(result.Error)
						failuresByError[errorKey]++
					}
				}
			}
		}
	}

	failureRate := float64(0)
	if totalReceipts > 0 {
		failureRate = float64(totalFailures) / float64(totalReceipts) * 100
	}

	analysis := map[string]interface{}{
		"total_receipts":       totalReceipts,
		"total_failures":       totalFailures,
		"failure_rate":         failureRate,
		"failures_by_platform": failuresByPlatform,
		"failures_by_error":    failuresByError,
		"analysis_time":        time.Now().Format(time.RFC3339),
	}

	return analysis
}

// categorizeError categorizes error messages into common patterns
func (a *ResultAggregator) categorizeError(errorMsg string) string {
	errorLower := strings.ToLower(errorMsg)

	switch {
	case strings.Contains(errorLower, "timeout"):
		return "timeout_errors"
	case strings.Contains(errorLower, "network"):
		return "network_errors"
	case strings.Contains(errorLower, "auth"):
		return "authentication_errors"
	case strings.Contains(errorLower, "rate limit"):
		return "rate_limit_errors"
	case strings.Contains(errorLower, "invalid"):
		return "validation_errors"
	case strings.Contains(errorLower, "unavailable"):
		return "service_unavailable_errors"
	default:
		return "other_errors"
	}
}

// ExtractPlatforms extracts unique platform names from results
func (a *ResultAggregator) ExtractPlatforms(results []PlatformResult) []string {
	platforms := make(map[string]bool)
	for _, result := range results {
		platforms[result.Platform] = true
	}

	var platformList []string
	for platform := range platforms {
		platformList = append(platformList, platform)
	}

	return platformList
}