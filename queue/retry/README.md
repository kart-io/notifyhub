# Retry Policies

This package provides various retry policies for handling message processing failures.

## Retry Policy Types

- `DefaultRetryPolicy()` - 3 retries with 30s initial interval, 2x multiplier
- `ExponentialBackoffPolicy(maxRetries, initialInterval, multiplier)` - Configurable exponential backoff
- `LinearBackoffPolicy(maxRetries, interval)` - Fixed interval retries
- `NoRetryPolicy()` - Disable retries
- `AggressiveRetryPolicy()` - 5 retries with 10s initial interval

## Features

- Exponential and linear backoff algorithms
- Jitter support to prevent thundering herd problems
- Configurable maximum retry counts and intervals

## Usage

```go
import "github.com/kart-io/notifyhub/queue/retry"

// Default policy
policy := retry.DefaultRetryPolicy()

// Custom exponential backoff
policy := retry.ExponentialBackoffPolicy(5, 10*time.Second, 2.0)

// Check if should retry
if policy.ShouldRetry(attempts) {
    nextRetry := policy.NextRetry(attempts)
    // Schedule retry at nextRetry time
}
```