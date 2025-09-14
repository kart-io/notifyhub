# Core Queue Implementation

This package contains the core queue interfaces and basic implementations.

## Components

- `interface.go` - Core Queue interface definition
- `message.go` - Message structure definition
- `simple.go` - In-memory queue implementation

## Usage

```go
import "github.com/kart-io/notifyhub/queue/core"

// Create a simple in-memory queue
queue := core.NewSimple(1000) // buffer size of 1000

// Enqueue a message
msg := &core.Message{
    Message: notifierMessage,
    // ... other fields
}
msgID, err := queue.Enqueue(ctx, msg)

// Dequeue a message
msg, err := queue.Dequeue(ctx)
```