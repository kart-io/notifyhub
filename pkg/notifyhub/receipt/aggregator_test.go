package receipt

import (
	"testing"
	"time"
)

// TestResultAggregator tests comprehensive receipt aggregation functionality
func TestResultAggregator(t *testing.T) {
	logger := &MockLogger{}
	config := AggregationConfig{
		PartialFailureThreshold: 0.6,
		FailureTolerance:        1,
		RequiredPlatforms:       []string{"feishu", "email"},
	}
	aggregator := NewResultAggregator(config, logger)

	t.Run("Multi-platform result aggregation", func(t *testing.T) {
		receipt := &Receipt{
			MessageID: "test-multi-platform",
			Results: []PlatformResult{
				{Platform: "feishu", Target: "user1", Success: true, Timestamp: time.Now()},
				{Platform: "feishu", Target: "user2", Success: false, Error: "rate limit", Timestamp: time.Now()},
				{Platform: "feishu", Target: "user3", Success: true, Timestamp: time.Now()},
				{Platform: "email", Target: "user1@example.com", Success: true, Timestamp: time.Now()},
				{Platform: "email", Target: "user2@example.com", Success: false, Error: "timeout", Timestamp: time.Now()},
				{Platform: "webhook", Target: "http://webhook1.com", Success: true, Timestamp: time.Now()},
				{Platform: "webhook", Target: "http://webhook2.com", Success: false, Error: "connection failed", Timestamp: time.Now()},
				{Platform: "sms", Target: "1234567890", Success: true, Timestamp: time.Now()},
			},
			Timestamp: time.Now(),
		}

		enhanced := aggregator.AggregateMultiPlatformResults(receipt)

		// Verify aggregation counts
		if enhanced.Total != 8 {
			t.Errorf("Expected total to be 8, got %d", enhanced.Total)
		}
		if enhanced.Successful != 5 {
			t.Errorf("Expected successful to be 5, got %d", enhanced.Successful)
		}
		if enhanced.Failed != 3 {
			t.Errorf("Expected failed to be 3, got %d", enhanced.Failed)
		}

		// Verify platform statistics extraction
		platforms := aggregator.ExtractPlatforms(enhanced.Results)
		expectedPlatforms := map[string]bool{
			"feishu":  true,
			"email":   true,
			"webhook": true,
			"sms":     true,
		}

		if len(platforms) != len(expectedPlatforms) {
			t.Errorf("Expected %d platforms, got %d", len(expectedPlatforms), len(platforms))
		}

		for _, platform := range platforms {
			if !expectedPlatforms[platform] {
				t.Errorf("Unexpected platform found: %s", platform)
			}
		}
	})

	t.Run("Status calculation - all success scenarios", func(t *testing.T) {

		tests := []struct {
			name     string
			receipt  *Receipt
			expected string
		}{
			{
				name: "Complete success",
				receipt: &Receipt{
					Total:      5,
					Successful: 5,
					Failed:     0,
				},
				expected: string(StatusCompleted),
			},
			{
				name: "Complete failure",
				receipt: &Receipt{
					Total:      5,
					Successful: 0,
					Failed:     5,
				},
				expected: string(StatusFailed),
			},
			{
				name: "Partial success above threshold",
				receipt: &Receipt{
					Total:      10,
					Successful: 7,
					Failed:     3,
				},
				expected: string(StatusPartialFailed),
			},
			{
				name: "Partial success below threshold",
				receipt: &Receipt{
					Total:      10,
					Successful: 5,
					Failed:     5,
				},
				expected: string(StatusFailed),
			},
			{
				name: "Within failure tolerance",
				receipt: &Receipt{
					Total:      5,
					Successful: 4,
					Failed:     1,
				},
				expected: string(StatusCompleted),
			},
			{
				name: "Beyond failure tolerance",
				receipt: &Receipt{
					Total:      5,
					Successful: 3,
					Failed:     2,
				},
				expected: string(StatusPartialFailed),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create a test aggregator without required platforms
				testConfig := AggregationConfig{
					PartialFailureThreshold: 0.6,
					FailureTolerance:        1,
				}
				testAggregator := NewResultAggregator(testConfig, logger)
				status := testAggregator.CalculateOverallStatus(tt.receipt)
				if status != tt.expected {
					t.Errorf("Expected status %s, got %s", tt.expected, status)
				}
			})
		}
	})

	t.Run("Required platforms validation", func(t *testing.T) {
		tests := []struct {
			name     string
			receipt  *Receipt
			expected bool
		}{
			{
				name: "All required platforms successful",
				receipt: &Receipt{
					Results: []PlatformResult{
						{Platform: "feishu", Success: true},
						{Platform: "email", Success: true},
						{Platform: "webhook", Success: false},
					},
				},
				expected: true,
			},
			{
				name: "Required platform failed",
				receipt: &Receipt{
					Results: []PlatformResult{
						{Platform: "feishu", Success: false},
						{Platform: "email", Success: true},
						{Platform: "webhook", Success: true},
					},
				},
				expected: false,
			},
			{
				name: "Missing required platform entirely",
				receipt: &Receipt{
					Results: []PlatformResult{
						{Platform: "webhook", Success: true},
						{Platform: "sms", Success: true},
					},
				},
				expected: false,
			},
			{
				name: "Partial success in required platform",
				receipt: &Receipt{
					Results: []PlatformResult{
						{Platform: "feishu", Success: true},
						{Platform: "feishu", Success: false},
						{Platform: "email", Success: true},
					},
				},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := aggregator.CheckRequiredPlatforms(tt.receipt)
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			})
		}
	})

	t.Run("Failure pattern analysis", func(t *testing.T) {
		receipts := []*Receipt{
			{
				MessageID: "msg-1",
				Failed:    0,
				Results:   []PlatformResult{{Platform: "feishu", Success: true}},
			},
			{
				MessageID: "msg-2",
				Failed:    2,
				Results: []PlatformResult{
					{Platform: "feishu", Success: false, Error: "rate limit exceeded"},
					{Platform: "email", Success: false, Error: "network timeout occurred"},
				},
			},
			{
				MessageID: "msg-3",
				Failed:    1,
				Results: []PlatformResult{
					{Platform: "webhook", Success: false, Error: "authentication failed"},
					{Platform: "sms", Success: true},
				},
			},
			{
				MessageID: "msg-4",
				Failed:    1,
				Results: []PlatformResult{
					{Platform: "email", Success: false, Error: "service unavailable"},
				},
			},
		}

		analysis := aggregator.AnalyzeFailurePatterns(receipts)

		// Verify analysis structure
		if analysis["total_receipts"].(int) != 4 {
			t.Errorf("Expected 4 total receipts, got %v", analysis["total_receipts"])
		}

		if analysis["total_failures"].(int) != 3 {
			t.Errorf("Expected 3 total failures, got %v", analysis["total_failures"])
		}

		// Check failure rate calculation
		expectedFailureRate := float64(3) / float64(4) * 100 // 75%
		if analysis["failure_rate"].(float64) != expectedFailureRate {
			t.Errorf("Expected failure rate %.2f, got %.2f", expectedFailureRate, analysis["failure_rate"].(float64))
		}

		// Verify platform failure counts
		platformFailures := analysis["failures_by_platform"].(map[string]int)
		expectedPlatformFailures := map[string]int{
			"feishu":  1,
			"email":   2,
			"webhook": 1,
		}

		for platform, expectedCount := range expectedPlatformFailures {
			if platformFailures[platform] != expectedCount {
				t.Errorf("Expected %d failures for platform %s, got %d", expectedCount, platform, platformFailures[platform])
			}
		}

		// Verify error categorization
		errorFailures := analysis["failures_by_error"].(map[string]int)
		expectedErrorTypes := map[string]int{
			"rate_limit_errors":          1,
			"timeout_errors":             1,
			"authentication_errors":      1,
			"service_unavailable_errors": 1,
		}

		for errorType, expectedCount := range expectedErrorTypes {
			if errorFailures[errorType] != expectedCount {
				t.Errorf("Expected %d failures for error type %s, got %d", expectedCount, errorType, errorFailures[errorType])
			}
		}
	})

	t.Run("Error categorization edge cases", func(t *testing.T) {
		tests := []struct {
			errorMsg string
			expected string
		}{
			{"Request timeout after 30 seconds", "timeout_errors"},
			{"NETWORK connection failed", "network_errors"},
			{"Auth token expired", "authentication_errors"},
			{"Rate limit exceeded for user", "rate_limit_errors"},
			{"Invalid request format", "validation_errors"},
			{"Service temporarily unavailable", "service_unavailable_errors"},
			{"Unknown error occurred", "other_errors"},
			{"", "other_errors"},
		}

		for _, tt := range tests {
			t.Run("Error: "+tt.errorMsg, func(t *testing.T) {
				category := aggregator.categorizeError(tt.errorMsg)
				if category != tt.expected {
					t.Errorf("Expected category %s for error '%s', got %s", tt.expected, tt.errorMsg, category)
				}
			})
		}
	})

	t.Run("Empty receipt handling", func(t *testing.T) {
		receipt := &Receipt{
			MessageID: "empty-receipt",
			Total:     0,
			Results:   []PlatformResult{},
			Timestamp: time.Now(),
		}

		enhanced := aggregator.AggregateMultiPlatformResults(receipt)
		if enhanced.Total != 0 {
			t.Errorf("Expected total to be 0 for empty receipt, got %d", enhanced.Total)
		}

		status := aggregator.CalculateOverallStatus(enhanced)
		if status != string(StatusFailed) {
			t.Errorf("Expected status to be failed for empty receipt, got %s", status)
		}
	})

	t.Run("Complex aggregation config scenarios", func(t *testing.T) {
		tests := []struct {
			name     string
			config   AggregationConfig
			receipt  *Receipt
			expected string
		}{
			{
				name: "High threshold with required platforms",
				config: AggregationConfig{
					PartialFailureThreshold: 0.9,
					RequiredPlatforms:       []string{"feishu"},
				},
				receipt: &Receipt{
					Total:      10,
					Successful: 8,
					Failed:     2,
					Results: []PlatformResult{
						{Platform: "feishu", Success: true},
						{Platform: "email", Success: false},
					},
				},
				expected: string(StatusFailed), // Success rate 0.8 < threshold 0.9
			},
			{
				name: "Zero threshold with tolerance",
				config: AggregationConfig{
					PartialFailureThreshold: 0.0,
					FailureTolerance:        5,
				},
				receipt: &Receipt{
					Total:      10,
					Successful: 7,
					Failed:     3,
				},
				expected: string(StatusCompleted), // Failed count (3) <= tolerance (5)
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				testAggregator := NewResultAggregator(tt.config, logger)
				status := testAggregator.CalculateOverallStatus(tt.receipt)
				if status != tt.expected {
					t.Errorf("Expected status %s, got %s", tt.expected, status)
				}
			})
		}
	})
}

// TestAggregatorPerformance tests aggregation performance with large datasets
func TestAggregatorPerformance(t *testing.T) {
	logger := &MockLogger{}
	config := AggregationConfig{
		PartialFailureThreshold: 0.5,
		FailureTolerance:        10,
	}
	aggregator := NewResultAggregator(config, logger)

	// Create a large receipt with many platform results
	results := make([]PlatformResult, 1000)
	for i := 0; i < 1000; i++ {
		results[i] = PlatformResult{
			Platform:  "platform" + string(rune(i%10+'0')),
			Target:    "target" + string(rune(i)),
			Success:   i%3 != 0, // 2/3 success rate
			Timestamp: time.Now(),
		}
	}

	receipt := &Receipt{
		MessageID: "performance-test",
		Results:   results,
		Timestamp: time.Now(),
	}

	start := time.Now()
	enhanced := aggregator.AggregateMultiPlatformResults(receipt)
	duration := time.Since(start)

	// Verify correctness
	expectedSuccessful := 0
	expectedFailed := 0
	for _, result := range results {
		if result.Success {
			expectedSuccessful++
		} else {
			expectedFailed++
		}
	}

	if enhanced.Successful != expectedSuccessful {
		t.Errorf("Expected %d successful, got %d", expectedSuccessful, enhanced.Successful)
	}
	if enhanced.Failed != expectedFailed {
		t.Errorf("Expected %d failed, got %d", expectedFailed, enhanced.Failed)
	}

	// Performance check - should complete within reasonable time
	if duration > 100*time.Millisecond {
		t.Errorf("Aggregation took too long: %v", duration)
	}

	t.Logf("Aggregated 1000 results in %v", duration)
}

// TestAggregatorEdgeCases tests edge cases and error conditions
func TestAggregatorEdgeCases(t *testing.T) {
	logger := &MockLogger{}

	t.Run("Nil receipt handling", func(t *testing.T) {
		config := AggregationConfig{PartialFailureThreshold: 0.5}
		aggregator := NewResultAggregator(config, logger)

		// This should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Aggregator panicked with nil receipt: %v", r)
			}
		}()

		// The aggregator should handle nil gracefully
		result := aggregator.AggregateMultiPlatformResults(nil)
		if result != nil {
			t.Errorf("Expected nil result for nil input, got %v", result)
		}
	})

	t.Run("Invalid threshold values", func(t *testing.T) {
		tests := []struct {
			threshold float64
			tolerance int
		}{
			{-0.5, 0},   // Negative threshold
			{1.5, 0},    // Threshold > 1
			{0.5, -5},   // Negative tolerance
		}

		for _, tt := range tests {
			config := AggregationConfig{
				PartialFailureThreshold: tt.threshold,
				FailureTolerance:        tt.tolerance,
			}
			aggregator := NewResultAggregator(config, logger)

			receipt := &Receipt{
				Total:      10,
				Successful: 7,
				Failed:     3,
			}

			// Should not panic and should return a valid status
			status := aggregator.CalculateOverallStatus(receipt)
			if status == "" {
				t.Errorf("Expected valid status for threshold %.2f, tolerance %d", tt.threshold, tt.tolerance)
			}
		}
	})

	t.Run("Very large failure analysis", func(t *testing.T) {
		config := AggregationConfig{PartialFailureThreshold: 0.5}
		aggregator := NewResultAggregator(config, logger)

		// Create many receipts for analysis
		receipts := make([]*Receipt, 1000)
		for i := 0; i < 1000; i++ {
			receipts[i] = &Receipt{
				MessageID: "analysis-" + string(rune(i)),
				Failed:    i % 5, // Varying failure counts
				Results: []PlatformResult{
					{Platform: "platform" + string(rune(i%3+'0')), Success: i%2 == 0, Error: "error" + string(rune(i%10+'0'))},
				},
			}
		}

		start := time.Now()
		analysis := aggregator.AnalyzeFailurePatterns(receipts)
		duration := time.Since(start)

		if analysis["total_receipts"].(int) != 1000 {
			t.Errorf("Expected 1000 receipts in analysis, got %v", analysis["total_receipts"])
		}

		// Should complete in reasonable time
		if duration > 500*time.Millisecond {
			t.Errorf("Failure analysis took too long: %v", duration)
		}

		t.Logf("Analyzed 1000 receipts in %v", duration)
	})
}