# Message Scheduler

This package provides delayed message scheduling functionality using a min-heap for efficient time-based message delivery.

## Features

- Delayed message scheduling with precise timing
- Min-heap based priority queue for efficient scheduling
- Automatic message delivery at scheduled times
- Thread-safe operations with proper synchronization

## Usage

```go
import (
    "github.com/kart-io/notifyhub/queue/scheduler"
    "github.com/kart-io/notifyhub/queue/core"
)

// Create scheduler with a queue
scheduler := scheduler.NewMessageScheduler(queue)

// Schedule a delayed message
msg := &core.Message{
    Message: notifierMessage,
    // Message will be processed after delay
}
msg.Message.Delay = 5 * time.Minute

err := scheduler.ScheduleMessage(msg)

// Stop scheduler
scheduler.Stop()
```

## How it Works

1. Messages with `Delay > 0` are added to the scheduler's heap
2. A background goroutine checks the heap every second
3. When a message's scheduled time arrives, it's enqueued to the main queue
4. The scheduler uses a min-heap to efficiently find the next message to process