package middleware

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/kart-io/notifyhub/core/hub"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/logger"
)

// RetryMiddleware implements retry logic for failed message sending
type RetryMiddleware struct {
	maxRetries      int
	baseDelay       time.Duration
	maxDelay        time.Duration
	backoffFunction BackoffFunction
	logger          logger.Interface
}

// BackoffFunction defines the interface for backoff calculation
type BackoffFunction func(attempt int, baseDelay time.Duration) time.Duration

// NewRetryMiddleware creates a new retry middleware
func NewRetryMiddleware(maxRetries int, baseDelay time.Duration, logger logger.Interface) *RetryMiddleware {
	return &RetryMiddleware{
		maxRetries:      maxRetries,
		baseDelay:       baseDelay,
		maxDelay:        30 * time.Second,
		backoffFunction: ExponentialBackoff,
		logger:          logger,
	}
}

// SetBackoffFunction sets a custom backoff function
func (m *RetryMiddleware) SetBackoffFunction(fn BackoffFunction) {
	m.backoffFunction = fn
}

// SetMaxDelay sets the maximum delay between retries
func (m *RetryMiddleware) SetMaxDelay(maxDelay time.Duration) {
	m.maxDelay = maxDelay
}

// Process processes the message with retry logic
func (m *RetryMiddleware) Process(ctx context.Context, msg *message.Message, targets []sending.Target, next hub.ProcessFunc) (*sending.SendingResults, error) {
	// Initial attempt
	results, err := next(ctx, msg, targets)
	if err != nil {
		return results, err
	}

	// Check for failed results that can be retried
	failedResults := m.getRetryableFailures(results)
	if len(failedResults) == 0 {
		return results, nil
	}

	// Retry failed targets
	for attempt := 1; attempt <= m.maxRetries; attempt++ {
		if len(failedResults) == 0 {
			break
		}

		// Calculate delay
		delay := m.calculateDelay(attempt)
		if m.logger != nil {
			m.logger.Info(ctx, "retrying failed targets", "attempt", attempt, "count", len(failedResults), "delay", delay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case <-time.After(delay):
		}

		// Retry targets
		retryTargets := make([]sending.Target, len(failedResults))
		for i, result := range failedResults {
			retryTargets[i] = result.Target
			result.IncrementAttempt()
			result.SetStatus(sending.StatusRetrying)
		}

		retryResults, err := next(ctx, msg, retryTargets)
		if err != nil {
			// Update failed results with retry error
			for _, result := range failedResults {
				result.SetError(fmt.Errorf("retry failed: %w", err))
			}
			return results, err
		}

		// Update results with retry outcomes
		newFailedResults := make([]*sending.Result, 0)
		for i, retryResult := range retryResults.Results {
			originalResult := failedResults[i]

			if retryResult.IsSuccess() {
				// Retry succeeded, update original result
				originalResult.SetStatus(sending.StatusSent)
				originalResult.SetResponse(retryResult.Response)
				originalResult.Error = nil
				if m.logger != nil {
					m.logger.Info(ctx, "retry succeeded", "target", originalResult.Target.String(), "attempt", attempt)
				}
			} else {
				// Retry still failed
				originalResult.SetError(retryResult.Error)
				if m.shouldRetry(retryResult.Error, attempt) {
					newFailedResults = append(newFailedResults, originalResult)
				} else {
					// Give up retrying
					if m.logger != nil {
						m.logger.Error(ctx, "retry exhausted", "target", originalResult.Target.String(), "attempts", attempt, "error", retryResult.Error)
					}
				}
			}
		}

		failedResults = newFailedResults
	}

	return results, nil
}

// getRetryableFailures returns failed results that can be retried
func (m *RetryMiddleware) getRetryableFailures(results *sending.SendingResults) []*sending.Result {
	var failures []*sending.Result
	for _, result := range results.Results {
		if result.IsFailed() && m.shouldRetry(result.Error, 0) {
			failures = append(failures, result)
		}
	}
	return failures
}

// shouldRetry determines if an error is retryable
func (m *RetryMiddleware) shouldRetry(err error, attempt int) bool {
	if err == nil || attempt >= m.maxRetries {
		return false
	}

	// Don't retry certain errors
	if err == sending.ErrInvalidCredentials ||
		err == sending.ErrInvalidTargetType ||
		err == sending.ErrEmptyTargetValue {
		return false
	}

	return true
}

// calculateDelay calculates the delay for a retry attempt
func (m *RetryMiddleware) calculateDelay(attempt int) time.Duration {
	delay := m.backoffFunction(attempt, m.baseDelay)
	if delay > m.maxDelay {
		delay = m.maxDelay
	}
	return delay
}

// ExponentialBackoff implements exponential backoff with jitter
func ExponentialBackoff(attempt int, baseDelay time.Duration) time.Duration {
	// Exponential backoff: delay = baseDelay * 2^attempt
	delay := float64(baseDelay) * math.Pow(2, float64(attempt-1))

	// Add jitter (up to 25% of the delay)
	jitter := delay * 0.25 * (0.5 - float64(time.Now().UnixNano()%2)/2)

	return time.Duration(delay + jitter)
}

// LinearBackoff implements linear backoff
func LinearBackoff(attempt int, baseDelay time.Duration) time.Duration {
	return time.Duration(attempt) * baseDelay
}

// ConstantBackoff implements constant delay
func ConstantBackoff(attempt int, baseDelay time.Duration) time.Duration {
	return baseDelay
}
