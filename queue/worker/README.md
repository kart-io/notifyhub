# Message Worker

This package provides message processing workers that consume messages from queues and handle sending, retries, and callbacks.

## Features

- Concurrent message processing with configurable worker count
- Automatic retry handling with exponential backoff
- Callback execution for all processing events
- Graceful shutdown support

## Usage

```go
import (
    "github.com/kart-io/notifyhub/queue/worker"
    "github.com/kart-io/notifyhub/queue/retry"
)

// Create a worker
retryPolicy := retry.DefaultRetryPolicy()
w := worker.NewWorker(queue, sender, retryPolicy, 5) // 5 concurrent workers

// Start processing
err := w.Start(ctx)
if err != nil {
    log.Fatal(err)
}

// Graceful shutdown
defer w.Stop()
```

## Configuration

- `concurrency` - Number of concurrent worker goroutines
- `retryPolicy` - Retry behavior configuration
- `sender` - Message sender implementation
- `queue` - Queue to consume messages from