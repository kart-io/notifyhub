package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/notifiers"
)

// SendResult represents a streaming send result
type SendResult struct {
	*notifiers.SendResult
	MessageID string    `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
}

// ResultStream represents a stream of send results
type ResultStream struct {
	ch     chan *SendResult
	done   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
	closed bool
}

// NewResultStream creates a new result stream
func NewResultStream(ctx context.Context, bufferSize int) *ResultStream {
	streamCtx, cancel := context.WithCancel(ctx)
	return &ResultStream{
		ch:     make(chan *SendResult, bufferSize),
		done:   make(chan struct{}),
		ctx:    streamCtx,
		cancel: cancel,
	}
}

// Send adds a result to the stream
func (rs *ResultStream) Send(result *SendResult) bool {
	rs.mu.RLock()
	if rs.closed {
		rs.mu.RUnlock()
		return false
	}
	rs.mu.RUnlock()

	select {
	case rs.ch <- result:
		return true
	case <-rs.ctx.Done():
		return false
	default:
		// Channel is full, drop the result
		return false
	}
}

// Receive returns the channel for receiving results
func (rs *ResultStream) Receive() <-chan *SendResult {
	return rs.ch
}

// Close closes the stream
func (rs *ResultStream) Close() {
	rs.mu.Lock()
	if rs.closed {
		rs.mu.Unlock()
		return
	}
	rs.closed = true
	rs.mu.Unlock()

	rs.cancel()
	rs.wg.Wait()
	close(rs.ch)
	close(rs.done)
}

// Done returns a channel that's closed when the stream is done
func (rs *ResultStream) Done() <-chan struct{} {
	return rs.done
}

// Wait waits for all operations to complete
func (rs *ResultStream) Wait() {
	rs.wg.Wait()
}

// StreamingOptions configures streaming behavior
type StreamingOptions struct {
	BufferSize    int           `json:"buffer_size"`
	Timeout       time.Duration `json:"timeout"`
	BatchSize     int           `json:"batch_size"`
	ConcurrentOps int           `json:"concurrent_ops"`
	RetryOnFail   bool          `json:"retry_on_fail"`
}

// DefaultStreamingOptions returns default streaming options
func DefaultStreamingOptions() *StreamingOptions {
	return &StreamingOptions{
		BufferSize:    100,
		Timeout:       30 * time.Second,
		BatchSize:     10,
		ConcurrentOps: 4,
		RetryOnFail:   true,
	}
}

// StreamingBatchSender handles streaming batch sends
type StreamingBatchSender struct {
	hub     *Hub
	options *StreamingOptions
	results *ResultStream
}

// NewStreamingBatchSender creates a new streaming batch sender
func (h *Hub) NewStreamingBatchSender(ctx context.Context, options *StreamingOptions) *StreamingBatchSender {
	if options == nil {
		options = DefaultStreamingOptions()
	}

	return &StreamingBatchSender{
		hub:     h,
		options: options,
		results: NewResultStream(ctx, options.BufferSize),
	}
}

// SendBatchStream sends multiple messages and streams results
func (sbs *StreamingBatchSender) SendBatchStream(ctx context.Context, messages []*notifiers.Message, options *Options) *ResultStream {
	sbs.results.wg.Add(1)

	go func() {
		defer sbs.results.wg.Done()
		defer sbs.results.Close()

		// Process messages in batches
		for i := 0; i < len(messages); i += sbs.options.BatchSize {
			end := i + sbs.options.BatchSize
			if end > len(messages) {
				end = len(messages)
			}

			batch := messages[i:end]
			sbs.processBatch(ctx, batch, options)

			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return sbs.results
}

// processBatch processes a batch of messages
func (sbs *StreamingBatchSender) processBatch(ctx context.Context, messages []*notifiers.Message, options *Options) {
	sem := make(chan struct{}, sbs.options.ConcurrentOps)
	var wg sync.WaitGroup

	for _, message := range messages {
		wg.Add(1)
		go func(msg *notifiers.Message) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			// Send message
			results, err := sbs.hub.Send(ctx, msg, options)
			timestamp := time.Now()

			if err != nil {
				// Send error result
				errorResult := &SendResult{
					SendResult: &notifiers.SendResult{
						Platform: "error",
						Success:  false,
						Error:    err.Error(),
						Duration: 0,
					},
					MessageID: msg.ID,
					Timestamp: timestamp,
				}
				sbs.results.Send(errorResult)
				return
			}

			// Send successful results
			for _, result := range results {
				streamResult := &SendResult{
					SendResult: result,
					MessageID:  msg.ID,
					Timestamp:  timestamp,
				}
				sbs.results.Send(streamResult)
			}
		}(message)
	}

	wg.Wait()
}

// Results returns the result stream
func (sbs *StreamingBatchSender) Results() *ResultStream {
	return sbs.results
}

// Hub methods for streaming

// SendStream sends a single message and streams results as they come
func (h *Hub) SendStream(ctx context.Context, message *notifiers.Message, options *Options) *ResultStream {
	streamOptions := DefaultStreamingOptions()
	streamOptions.ConcurrentOps = len(message.Targets) // One per target

	sender := h.NewStreamingBatchSender(ctx, streamOptions)
	return sender.SendBatchStream(ctx, []*notifiers.Message{message}, options)
}

// SendBatchStream sends multiple messages and streams results
func (h *Hub) SendBatchStream(ctx context.Context, messages []*notifiers.Message, options *Options, streamingOptions *StreamingOptions) *ResultStream {
	sender := h.NewStreamingBatchSender(ctx, streamingOptions)
	return sender.SendBatchStream(ctx, messages, options)
}

// SendToTargetListStream sends a message to a target list and streams results
func (h *Hub) SendToTargetListStream(ctx context.Context, message *notifiers.Message, targetList *TargetList, options *Options) *ResultStream {
	// Create message with targets
	messageBuilder := NewMessage().
		Title(message.Title).
		Body(message.Body).
		Priority(message.Priority).
		Format(message.Format)

	// Add variables and metadata
	for k, v := range message.Variables {
		messageBuilder.Variable(k, v)
	}
	for k, v := range message.Metadata {
		messageBuilder.Metadata(k, v)
	}

	// Add targets from target list
	for _, target := range targetList.Build() {
		messageBuilder.Target(target)
	}

	finalMessage := messageBuilder.Build()
	return h.SendStream(ctx, finalMessage, options)
}

// StreamCollector collects and aggregates streaming results
type StreamCollector struct {
	results       []*SendResult
	successCount  int
	errorCount    int
	totalDuration time.Duration
	mu            sync.RWMutex
}

// NewStreamCollector creates a new stream collector
func NewStreamCollector() *StreamCollector {
	return &StreamCollector{
		results: make([]*SendResult, 0),
	}
}

// Collect collects results from a stream
func (sc *StreamCollector) Collect(stream *ResultStream) {
	for result := range stream.Receive() {
		sc.Add(result)
	}
}

// Add adds a result to the collector
func (sc *StreamCollector) Add(result *SendResult) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.results = append(sc.results, result)
	sc.totalDuration += result.Duration

	if result.Success {
		sc.successCount++
	} else {
		sc.errorCount++
	}
}

// Summary returns a summary of collected results
func (sc *StreamCollector) Summary() *StreamSummary {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	avgDuration := time.Duration(0)
	if len(sc.results) > 0 {
		avgDuration = sc.totalDuration / time.Duration(len(sc.results))
	}

	return &StreamSummary{
		TotalResults:    len(sc.results),
		SuccessCount:    sc.successCount,
		ErrorCount:      sc.errorCount,
		SuccessRate:     float64(sc.successCount) / float64(len(sc.results)),
		AverageDuration: avgDuration,
		TotalDuration:   sc.totalDuration,
	}
}

// Results returns all collected results
func (sc *StreamCollector) Results() []*SendResult {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	results := make([]*SendResult, len(sc.results))
	copy(results, sc.results)
	return results
}

// StreamSummary provides summary information about streaming results
type StreamSummary struct {
	TotalResults    int           `json:"total_results"`
	SuccessCount    int           `json:"success_count"`
	ErrorCount      int           `json:"error_count"`
	SuccessRate     float64       `json:"success_rate"`
	AverageDuration time.Duration `json:"average_duration"`
	TotalDuration   time.Duration `json:"total_duration"`
}

// String returns a string representation of the summary
func (ss *StreamSummary) String() string {
	return fmt.Sprintf("Stream Summary: %d total, %d success (%.1f%%), %d errors, avg duration: %v",
		ss.TotalResults, ss.SuccessCount, ss.SuccessRate*100, ss.ErrorCount, ss.AverageDuration)
}

// StreamProcessor provides advanced stream processing capabilities
type StreamProcessor struct {
	filters    []func(*SendResult) bool
	transforms []func(*SendResult) *SendResult
	mu         sync.RWMutex
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor() *StreamProcessor {
	return &StreamProcessor{
		filters:    make([]func(*SendResult) bool, 0),
		transforms: make([]func(*SendResult) *SendResult, 0),
	}
}

// AddFilter adds a filter function
func (sp *StreamProcessor) AddFilter(filter func(*SendResult) bool) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.filters = append(sp.filters, filter)
}

// AddTransform adds a transform function
func (sp *StreamProcessor) AddTransform(transform func(*SendResult) *SendResult) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.transforms = append(sp.transforms, transform)
}

// Process processes a stream with filters and transforms
func (sp *StreamProcessor) Process(input *ResultStream, bufferSize int) *ResultStream {
	output := NewResultStream(input.ctx, bufferSize)

	go func() {
		defer output.Close()

		for result := range input.Receive() {
			// Apply filters
			skip := false
			sp.mu.RLock()
			for _, filter := range sp.filters {
				if !filter(result) {
					skip = true
					break
				}
			}
			sp.mu.RUnlock()

			if skip {
				continue
			}

			// Apply transforms
			processedResult := result
			sp.mu.RLock()
			for _, transform := range sp.transforms {
				processedResult = transform(processedResult)
				if processedResult == nil {
					break
				}
			}
			sp.mu.RUnlock()

			if processedResult != nil {
				output.Send(processedResult)
			}
		}
	}()

	return output
}

// Common filter functions

// SuccessFilter filters only successful results
func SuccessFilter(result *SendResult) bool {
	return result.Success
}

// ErrorFilter filters only error results
func ErrorFilter(result *SendResult) bool {
	return !result.Success
}

// PlatformFilter creates a filter for specific platform
func PlatformFilter(platform string) func(*SendResult) bool {
	return func(result *SendResult) bool {
		return result.Platform == platform
	}
}

// DurationFilter creates a filter for results within duration threshold
func DurationFilter(maxDuration time.Duration) func(*SendResult) bool {
	return func(result *SendResult) bool {
		return result.Duration <= maxDuration
	}
}