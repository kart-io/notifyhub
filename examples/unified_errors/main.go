package main

import (
	std_errors "errors"
	"fmt"

	"github.com/kart-io/notifyhub/core/errors"
)

// UnifiedErrorHandlingExample demonstrates how the standardized error system
// provides consistent error handling across different platforms and transports
func main() {
	fmt.Println("=== Unified Error Handling Example ===")

	// 1. Platform-specific errors are now standardized
	demonstratePlatformErrors()

	// 2. HTTP errors are mapped to standard error codes
	demonstrateHTTPErrorMapping()

	// 3. SMTP errors are mapped to standard error codes
	demonstrateSMTPErrorMapping()

	// 4. Error categorization and checking
	demonstrateErrorCategorization()

	// 5. Retry logic based on error types
	demonstrateRetryLogic()
}

func demonstratePlatformErrors() {
	fmt.Println("\n--- Platform-Specific Errors ---")

	// Feishu-specific errors now use standardized error system
	feishuConfigErr := errors.NewFeishuError(errors.CodeInvalidConfig, "webhook URL is invalid")
	fmt.Printf("Feishu Error: %s\n", feishuConfigErr)
	fmt.Printf("  - Code: %s\n", feishuConfigErr.Code)
	fmt.Printf("  - Category: %s\n", feishuConfigErr.Category)
	fmt.Printf("  - Platform: %s\n", feishuConfigErr.Platform)
	fmt.Printf("  - HTTP Status: %d\n", feishuConfigErr.HTTPStatusCode())

	// Email-specific errors
	emailErr := errors.NewEmailError(errors.CodeRateLimited, "SMTP rate limit exceeded")
	fmt.Printf("\nEmail Error: %s\n", emailErr)
	fmt.Printf("  - Code: %s\n", emailErr.Code)
	fmt.Printf("  - Category: %s\n", emailErr.Category)
	fmt.Printf("  - Platform: %s\n", emailErr.Platform)
	fmt.Printf("  - Is Retryable: %t\n", emailErr.IsRetryable())
}

func demonstrateHTTPErrorMapping() {
	fmt.Println("\n--- HTTP Error Mapping ---")

	// Simulate different HTTP error responses
	httpErrors := []struct {
		statusCode int
		body       string
		platform   string
	}{
		{401, "Authentication required", "feishu"},
		{429, "Rate limit exceeded", "feishu"},
		{500, "Internal server error", "email"},
		{404, "Webhook not found", "feishu"},
	}

	for _, httpErr := range httpErrors {
		mappedErr := errors.MapHTTPError(httpErr.statusCode, httpErr.body, httpErr.platform)
		fmt.Printf("HTTP %d (%s) -> %s: %s\n",
			httpErr.statusCode, httpErr.platform, mappedErr.Code, mappedErr.Message)
		fmt.Printf("  - Category: %s, Retryable: %t\n",
			mappedErr.Category, mappedErr.IsRetryable())
	}
}

func demonstrateSMTPErrorMapping() {
	fmt.Println("\n--- SMTP Error Mapping ---")

	// Simulate SMTP errors
	smtpErrors := []error{
		fmt.Errorf("535 authentication failed"),
		fmt.Errorf("421 too many connections"),
		fmt.Errorf("smtp timeout"),
		fmt.Errorf("550 invalid recipient address"),
	}

	for _, smtpErr := range smtpErrors {
		mappedErr := errors.MapSMTPError(smtpErr)
		fmt.Printf("SMTP '%s' -> %s: %s\n",
			smtpErr.Error(), mappedErr.Code, mappedErr.Message)
		fmt.Printf("  - Category: %s, Retryable: %t\n",
			mappedErr.Category, mappedErr.IsRetryable())
	}
}

func demonstrateErrorCategorization() {
	fmt.Println("\n--- Error Categorization ---")

	testErrors := []error{
		errors.ErrInvalidConfig,
		errors.ErrEmptyMessage,
		errors.ErrNetworkError,
		errors.ErrInvalidCredentials,
		errors.ErrRateLimited,
	}

	for _, err := range testErrors {
		fmt.Printf("Error: %s\n", err)
		fmt.Printf("  - Config Error: %t\n", errors.IsConfigurationError(err))
		fmt.Printf("  - Validation Error: %t\n", errors.IsValidationError(err))
		fmt.Printf("  - Network Error: %t\n", errors.IsNetworkError(err))
		fmt.Printf("  - Auth Error: %t\n", errors.IsAuthError(err))
		fmt.Printf("  - Rate Limit Error: %t\n", errors.IsRateLimitError(err))
		fmt.Printf("  - Retryable: %t\n", errors.IsRetryableError(err))
		fmt.Println()
	}
}

func demonstrateRetryLogic() {
	fmt.Println("\n--- Retry Logic Example ---")

	// Simulate different error scenarios and show retry decisions
	scenarios := []error{
		errors.ErrNetworkError,       // Should retry
		errors.ErrTimeout,            // Should retry
		errors.ErrRateLimited,        // Should retry
		errors.ErrInvalidCredentials, // Should NOT retry
		errors.ErrInvalidConfig,      // Should NOT retry
		errors.ErrEmptyMessage,       // Should NOT retry
	}

	for _, err := range scenarios {
		shouldRetry := errors.IsRetryableError(err)
		action := "ABORT"
		if shouldRetry {
			action = "RETRY"
		}

		fmt.Printf("Error: %s -> %s\n", err, action)
	}
}

// SimulateTransportErrorHandling shows how transport layers now use unified errors
func SimulateTransportErrorHandling() {
	fmt.Println("\n--- Transport Error Handling Simulation ---")

	// This would be actual transport code, but we'll simulate the error handling
	simulateFeishuErrors := []struct {
		httpStatus int
		apiCode    int
		apiMsg     string
	}{
		{200, 19001, "invalid app_id"},
		{200, 19003, "request too frequent"},
		{429, 0, "rate limited by gateway"},
		{500, 0, "internal server error"},
	}

	for _, scenario := range simulateFeishuErrors {
		var err error

		if scenario.httpStatus != 200 {
			// HTTP-level error
			err = errors.MapHTTPError(scenario.httpStatus, scenario.apiMsg, "feishu")
		} else if scenario.apiCode != 0 {
			// Feishu API-level error
			switch scenario.apiCode {
			case 19001:
				err = errors.NewFeishuError(errors.CodeInvalidCredentials, scenario.apiMsg)
			case 19003:
				err = errors.NewFeishuError(errors.CodeRateLimited, scenario.apiMsg)
			default:
				err = errors.NewFeishuError(errors.CodeSendingFailed, scenario.apiMsg)
			}
		}

		fmt.Printf("Scenario: HTTP %d, API %d -> %s\n",
			scenario.httpStatus, scenario.apiCode, err)

		// Unified error handling logic
		if notifyErr, ok := err.(*errors.NotifyError); ok {
			fmt.Printf("  - Should retry: %t\n", notifyErr.IsRetryable())
			fmt.Printf("  - HTTP status for API: %d\n", notifyErr.HTTPStatusCode())
		}
		fmt.Println()
	}

	// Example: Email transport SMTP error handling
	fmt.Println("--- Email SMTP Error Scenarios ---")
	smtpScenarios := []string{
		"535 authentication failed",
		"421 rate limit exceeded",
		"connection timeout",
		"550 recipient rejected",
	}

	for _, smtpErrStr := range smtpScenarios {
		smtpErr := std_errors.New(smtpErrStr)
		mappedErr := errors.MapSMTPError(smtpErr)

		fmt.Printf("SMTP: %s -> %s\n", smtpErrStr, mappedErr)
		fmt.Printf("  - Should retry: %t\n", mappedErr.IsRetryable())
		fmt.Printf("  - Category: %s\n", mappedErr.Category)
		fmt.Println()
	}
}
