// +build !redis

// Package async provides fallback implementation when Redis is not available
package async

import (
	"fmt"

	"github.com/kart-io/notifyhub/pkg/logger"
)

// RedisQueueConfig represents Redis queue configuration (fallback)
type RedisQueueConfig struct {
	Address       string
	Password      string
	DB            int
	StreamKey     string
	ConsumerGroup string
	ConsumerName  string
	MaxSize       int
}

// NewRedisAsyncQueue creates a new Redis-based async queue (fallback)
func NewRedisAsyncQueue(config RedisQueueConfig, logger logger.Logger) (AsyncQueue, error) {
	return nil, fmt.Errorf("Redis queue not available - rebuild with 'redis' build tag")
}