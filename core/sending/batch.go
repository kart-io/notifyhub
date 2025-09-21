package sending

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/core/message"
)

// BatchSender handles batch message sending
// This implements the proposal's batch sending pattern
type BatchSender struct {
	transports     map[string]Transport
	logger         Logger
	maxConcurrency int
	batchSize      int
}

// BatchOptions configures batch sending behavior
type BatchOptions struct {
	MaxConcurrency int  // Maximum concurrent sends
	StopOnError    bool // Stop batch on first error
	Timeout        time.Duration
}

// BatchResult represents the result of a batch send
type BatchResult struct {
	TotalCount   int
	SuccessCount int
	FailedCount  int
	Results      []*Result
	Duration     time.Duration
	StoppedEarly bool
}

// NewBatchSender creates a new batch sender
func NewBatchSender(logger Logger, maxConcurrency int, batchSize int) *BatchSender {
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	if batchSize <= 0 {
		batchSize = 100
	}

	return &BatchSender{
		transports:     make(map[string]Transport),
		logger:         logger,
		maxConcurrency: maxConcurrency,
		batchSize:      batchSize,
	}
}

// RegisterTransport registers a transport for a platform
func (b *BatchSender) RegisterTransport(transport Transport) error {
	name := transport.Name()
	if _, exists := b.transports[name]; exists {
		return fmt.Errorf("transport %s already registered", name)
	}
	b.transports[name] = transport
	return nil
}

// SendBatch sends multiple messages in a batch
func (b *BatchSender) SendBatch(ctx context.Context, messages []*message.Message, targets []Target, opts *BatchOptions) (*BatchResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets provided")
	}

	if opts == nil {
		opts = &BatchOptions{
			MaxConcurrency: b.maxConcurrency,
			Timeout:        5 * time.Minute,
		}
	}

	startTime := time.Now()
	result := &BatchResult{
		TotalCount: len(messages) * len(targets),
		Results:    make([]*Result, 0, len(messages)*len(targets)),
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Create semaphore for concurrency control
	sem := make(chan struct{}, opts.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var stopFlag bool

	// Process each message-target combination
	for _, msg := range messages {
		for _, target := range targets {
			// Check if we should stop
			mu.Lock()
			if stopFlag {
				result.StoppedEarly = true
				mu.Unlock()
				break
			}
			mu.Unlock()

			wg.Add(1)
			sem <- struct{}{} // Acquire semaphore

			go func(m *message.Message, t Target) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore

				// Send message to target
				sendResult := b.sendToTarget(ctx, m, t)

				// Update results
				mu.Lock()
				result.Results = append(result.Results, sendResult)
				if sendResult.Success {
					result.SuccessCount++
				} else {
					result.FailedCount++
					if opts.StopOnError {
						stopFlag = true
					}
				}
				mu.Unlock()
			}(msg, target)
		}

		if stopFlag {
			break
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()

	result.Duration = time.Since(startTime)

	b.logger.Info(ctx, "Batch send completed", "total", result.TotalCount, "success", result.SuccessCount, "failed", result.FailedCount, "duration", result.Duration, "stopped", result.StoppedEarly)

	return result, nil
}

// sendToTarget sends a message to a single target
func (b *BatchSender) sendToTarget(ctx context.Context, msg *message.Message, target Target) *Result {
	result := NewResult(msg.ID, target)
	result.StartTime = time.Now()

	// Get transport for target platform
	transport, exists := b.transports[target.GetPlatform()]
	if !exists {
		result.Error = fmt.Errorf("no transport for platform: %s", target.GetPlatform())
		result.Status = StatusFailed
		result.EndTime = time.Now()
		return result
	}

	// Send through transport
	transportResult, err := transport.Send(ctx, msg, target)
	if err != nil {
		result.Error = err
		result.Status = StatusFailed
	} else {
		if transportResult != nil {
			result = transportResult
		} else {
			result.Status = StatusSent
			result.Success = true
		}
	}

	result.EndTime = time.Now()
	return result
}

// SendBatchWithGrouping sends messages grouped by platform for efficiency
func (b *BatchSender) SendBatchWithGrouping(ctx context.Context, messages []*message.Message, targets []Target, opts *BatchOptions) (*BatchResult, error) {
	// Group targets by platform
	platformGroups := make(map[string][]Target)
	for _, target := range targets {
		platform := target.GetPlatform()
		platformGroups[platform] = append(platformGroups[platform], target)
	}

	result := &BatchResult{
		TotalCount: len(messages) * len(targets),
		Results:    make([]*Result, 0, len(messages)*len(targets)),
	}

	startTime := time.Now()

	// Process each platform group
	var wg sync.WaitGroup
	var mu sync.Mutex

	for platform, platformTargets := range platformGroups {
		wg.Add(1)
		go func(p string, targets []Target) {
			defer wg.Done()

			_, exists := b.transports[p]
			if !exists {
				b.logger.Error(ctx, "Transport not found", "platform", p, "error", fmt.Errorf("no transport for platform: %s", p))
				return
			}

			// Send to all targets for this platform
			for _, msg := range messages {
				for _, target := range targets {
					sendResult := b.sendToTarget(ctx, msg, target)

					mu.Lock()
					result.Results = append(result.Results, sendResult)
					if sendResult.Success {
						result.SuccessCount++
					} else {
						result.FailedCount++
					}
					mu.Unlock()
				}
			}
		}(platform, platformTargets)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	return result, nil
}
