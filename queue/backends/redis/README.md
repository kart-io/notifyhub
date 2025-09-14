# Redis Queue Backend

Redis-based queue implementation using Redis Streams for persistent message queuing.

## Features

- Persistent message storage with Redis Streams
- Consumer group support for distributed processing
- Automatic message acknowledgment and retry handling
- Stream-based message IDs for ordering guarantees
- **Support for external Redis client management**
- **Flexible connection lifecycle management**

## Usage Methods

### Method 1: Using External Redis Client (Recommended)

**Use this when you already have a Redis connection pool or client**

```go
import (
    "github.com/redis/go-redis/v9"
    "github.com/kart-io/notifyhub/queue/backends/redis"
)

// Your existing Redis client
redisClient := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
    PoolSize: 20, // Your custom pool settings
})

// Queue-specific configuration
queueConfig := &redis.RedisQueueConfig{
    StreamName:      "notifyhub:messages",
    ConsumerGroup:   "workers",
    ConsumerName:    "worker-1",
    MaxLen:          10000,
    ClaimMinIdle:    5 * time.Minute,
    ProcessingLimit: 10,
}

// Create Redis queue with external client
queue, err := redis.NewRedisQueueWithClient(redisClient, queueConfig)
if err != nil {
    log.Fatal(err)
}

// Queue doesn't manage Redis client lifecycle
defer queue.Close()        // Only closes queue, not Redis client
defer redisClient.Close()  // You manage Redis client lifecycle
```

### Method 2: Full Options with Internal Connection

```go
// Complete configuration including connection
options := &redis.RedisQueueOptions{
    RedisConnectionConfig: &redis.RedisConnectionConfig{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    },
    RedisQueueConfig: &redis.RedisQueueConfig{
        StreamName:      "notifyhub:messages",
        ConsumerGroup:   "workers",
        ConsumerName:    "worker-1",
        MaxLen:          10000,
        ClaimMinIdle:    5 * time.Minute,
        ProcessingLimit: 10,
    },
}

// Create Redis queue with internal connection management
queue, err := redis.NewRedisQueueWithOptions(options)
if err != nil {
    log.Fatal(err)
}
defer queue.Close() // Closes both queue and internal Redis client
```

### Method 3: Simple Configuration (Legacy)

```go
// Use default configuration
queue, err := redis.NewRedisQueue(redis.DefaultRedisQueueOptions())
if err != nil {
    log.Fatal(err)
}
defer queue.Close()
```

## Real-world Usage Scenarios

### Scenario 1: Microservice with Shared Redis Pool

```go
// In your application initialization
func setupRedis() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     os.Getenv("REDIS_URL"),
        Password: os.Getenv("REDIS_PASSWORD"),
        DB:       0,
        PoolSize: 20,
        MinIdleConns: 5,
    })
}

func setupNotificationQueue(redisClient *redis.Client) (*redis.RedisQueue, error) {
    queueConfig := &redis.RedisQueueConfig{
        StreamName:      "notifications:queue",
        ConsumerGroup:   "notification-workers",
        ConsumerName:    os.Getenv("INSTANCE_ID"), // Unique per instance
        MaxLen:          50000,
        ClaimMinIdle:    2 * time.Minute,
        ProcessingLimit: 50,
    }

    return redis.NewRedisQueueWithClient(redisClient, queueConfig)
}

// In your main function
func main() {
    redisClient := setupRedis()
    defer redisClient.Close()

    // Use Redis client for caching, sessions, etc.
    cache := redis.NewCacheService(redisClient)

    // Use same Redis client for message queue
    queue, err := setupNotificationQueue(redisClient)
    if err != nil {
        log.Fatal(err)
    }
    defer queue.Close()

    // Start workers
    worker := worker.NewWorker(queue, sender, retryPolicy, 10)
    worker.Start(ctx)
    defer worker.Stop()
}
```

### Scenario 2: Multiple Queues with Single Redis Connection

```go
func setupMultipleQueues(redisClient *redis.Client) error {
    // High-priority notification queue
    highPriorityQueue, err := redis.NewRedisQueueWithClient(redisClient, &redis.RedisQueueConfig{
        StreamName:    "notifications:high-priority",
        ConsumerGroup: "high-priority-workers",
        ConsumerName:  "worker-1",
    })
    if err != nil {
        return err
    }

    // Low-priority notification queue
    lowPriorityQueue, err := redis.NewRedisQueueWithClient(redisClient, &redis.RedisQueueConfig{
        StreamName:    "notifications:low-priority",
        ConsumerGroup: "low-priority-workers",
        ConsumerName:  "worker-1",
    })
    if err != nil {
        return err
    }

    // Both queues share the same Redis connection
    defer highPriorityQueue.Close()
    defer lowPriorityQueue.Close()

    return nil
}
```

### Scenario 3: Redis Cluster Support

```go
func setupWithRedisCluster() (*redis.RedisQueue, error) {
    // Redis Cluster client
    clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
        Addrs: []string{
            "redis-node1:6379",
            "redis-node2:6379",
            "redis-node3:6379",
        },
        Password: os.Getenv("REDIS_PASSWORD"),
    })

    queueConfig := &redis.RedisQueueConfig{
        StreamName:    "notifications:cluster",
        ConsumerGroup: "cluster-workers",
        ConsumerName:  fmt.Sprintf("worker-%s", os.Getenv("POD_NAME")),
    }

    // Note: NewRedisQueueWithClient works with both regular and cluster clients
    return redis.NewRedisQueueWithClient(clusterClient, queueConfig)
}
```

## Migration Guide

### From Old API to New API

**Before:**
```go
// Old way (still supported but deprecated)
config := &redis.RedisQueueConfig{
    Addr:          "localhost:6379",
    StreamName:    "messages",
    ConsumerGroup: "workers",
}
queue, err := redis.NewRedisQueue(config)
```

**After:**
```go
// New recommended way
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
    // Add your custom Redis options here
})

queueConfig := &redis.RedisQueueConfig{
    StreamName:    "messages",
    ConsumerGroup: "workers",
    ConsumerName:  "worker-1",
}

queue, err := redis.NewRedisQueueWithClient(redisClient, queueConfig)
```

## Requirements

- Redis 5.0+ (for Redis Streams support)
- Go Redis client (`github.com/go-redis/redis/v8`)

## Stream Structure

Messages are stored in Redis Streams with the following fields:

- `data` - JSON serialized message data
- `created_at` - Message creation timestamp
- `attempts` - Number of processing attempts
