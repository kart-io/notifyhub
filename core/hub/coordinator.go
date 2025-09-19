package hub

import (
	"context"
	"fmt"
	"sync"

	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/core/sending"
	"github.com/kart-io/notifyhub/logger"
)

// Coordinator coordinates message sending across platforms
// This implements the proposal's coordinator pattern
type Coordinator struct {
	platforms map[string]Platform
	mutex     sync.RWMutex
	logger    logger.Interface
}

// Platform represents a messaging platform
type Platform interface {
	Name() string
	Send(ctx context.Context, msg *message.Message, targets []sending.Target) (*sending.Result, error)
	Validate(msg *message.Message) error
	IsAvailable() bool
}

// NewCoordinator creates a new coordinator
func NewCoordinator(logger logger.Interface) *Coordinator {
	return &Coordinator{
		platforms: make(map[string]Platform),
		logger:    logger,
	}
}

// RegisterPlatform registers a platform with the coordinator
func (c *Coordinator) RegisterPlatform(platform Platform) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	name := platform.Name()
	if _, exists := c.platforms[name]; exists {
		return fmt.Errorf("platform %s already registered", name)
	}

	c.platforms[name] = platform
	c.logger.Info(context.Background(), "Platform registered", "platform", name)
	return nil
}

// Send coordinates sending a message to targets
func (c *Coordinator) Send(ctx context.Context, msg *message.Message, targets []sending.Target) (*sending.SendingResults, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets specified")
	}

	// Group targets by platform
	targetsByPlatform := make(map[string][]sending.Target)
	for _, target := range targets {
		platform := target.GetPlatform()
		targetsByPlatform[platform] = append(targetsByPlatform[platform], target)
	}

	// Send to each platform
	results := &sending.SendingResults{
		Results: make([]*sending.Result, 0, len(targets)),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for platformName, platformTargets := range targetsByPlatform {
		platform, exists := c.platforms[platformName]
		if !exists {
			c.logger.Warn(ctx, "Platform not found", "platform", platformName)
			continue
		}

		wg.Add(1)
		go func(p Platform, targets []sending.Target) {
			defer wg.Done()

			result, err := p.Send(ctx, msg, targets)
			if err != nil {
				c.logger.Error(ctx, "Failed to send message", "platform", p.Name(), "error", err)
				result = &sending.Result{
					Success: false,
					Error:   err,
				}
			}

			mu.Lock()
			results.Results = append(results.Results, result)
			mu.Unlock()
		}(platform, platformTargets)
	}

	wg.Wait()

	// Calculate summary
	for _, result := range results.Results {
		if result.Success {
			results.SuccessCount++
		} else {
			results.FailedCount++
		}
		results.TotalCount++
	}

	return results, nil
}

// GetPlatforms returns all registered platforms
func (c *Coordinator) GetPlatforms() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	names := make([]string, 0, len(c.platforms))
	for name := range c.platforms {
		names = append(names, name)
	}
	return names
}
