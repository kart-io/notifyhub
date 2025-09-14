# Redis Queue Backend

Redis-based queue implementation using Redis Streams for persistent message queuing.

## Features

- Persistent message storage with Redis Streams
- Consumer group support for distributed processing
- Automatic message acknowledgment and retry handling
- Stream-based message IDs for ordering guarantees

## Configuration

```go
import "github.com/kart-io/notifyhub/queue/backends/redis"

// Redis queue configuration
config := &redis.Config{
    Addr:         "localhost:6379",
    Password:     "",
    DB:           0,
    StreamName:   "notifyhub:messages",
    ConsumerGroup: "workers",
    ConsumerName:  "worker-1",
}

// Create Redis queue
queue, err := redis.NewRedisQueue(config)
if err != nil {
    log.Fatal(err)
}
defer queue.Close()
```

## Requirements

- Redis 5.0+ (for Redis Streams support)
- Go Redis client (`github.com/go-redis/redis/v8`)

## Stream Structure

Messages are stored in Redis Streams with the following fields:
- `data` - JSON serialized message data
- `created_at` - Message creation timestamp
- `attempts` - Number of processing attempts