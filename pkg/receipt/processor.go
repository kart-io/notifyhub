// Package receipt provides receipt processing functionality for NotifyHub
package receipt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kart-io/notifyhub/pkg/utils/logger"
)

// Processor processes and manages message receipts
type Processor interface {
	// ProcessReceipt processes a single receipt
	ProcessReceipt(ctx context.Context, receipt *Receipt) error

	// ProcessBatch processes multiple receipts
	ProcessBatch(ctx context.Context, receipts []*Receipt) error

	// AddHandler adds a receipt handler
	AddHandler(handler ReceiptHandler)

	// RemoveHandler removes a receipt handler
	RemoveHandler(handlerID string)

	// GetStats returns processing statistics
	GetStats() ProcessorStats

	// Start starts the processor
	Start(ctx context.Context) error

	// Stop stops the processor
	Stop() error
}

// ReceiptHandler handles specific types of receipt processing
type ReceiptHandler interface {
	// ID returns the handler ID
	ID() string

	// CanHandle checks if this handler can process the receipt
	CanHandle(receipt *Receipt) bool

	// Handle processes the receipt
	Handle(ctx context.Context, receipt *Receipt) error

	// Priority returns the handler priority (higher = more priority)
	Priority() int
}

// ProcessorConfig configures the receipt processor
type ProcessorConfig struct {
	Workers        int           `json:"workers"`
	BufferSize     int           `json:"buffer_size"`
	ProcessTimeout time.Duration `json:"process_timeout"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// ProcessorStats provides processing statistics
type ProcessorStats struct {
	TotalProcessed   int64     `json:"total_processed"`
	TotalFailed      int64     `json:"total_failed"`
	CurrentlyPending int64     `json:"currently_pending"`
	AverageTime      float64   `json:"average_time_ms"`
	LastProcessed    time.Time `json:"last_processed"`
	Handlers         int       `json:"handlers"`
}

// DefaultProcessor implements the Processor interface
type DefaultProcessor struct {
	config   ProcessorConfig
	handlers []ReceiptHandler
	queue    chan *Receipt
	stats    ProcessorStats
	logger   logger.Logger
	workers  []*worker
	mutex    sync.RWMutex
	running  bool
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewDefaultProcessor creates a new default receipt processor
func NewDefaultProcessor(config ProcessorConfig, logger logger.Logger) *DefaultProcessor {
	// Apply defaults
	if config.Workers <= 0 {
		config.Workers = 4
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}
	if config.ProcessTimeout == 0 {
		config.ProcessTimeout = 30 * time.Second
	}
	if config.RetryAttempts <= 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	return &DefaultProcessor{
		config:   config,
		handlers: make([]ReceiptHandler, 0),
		queue:    make(chan *Receipt, config.BufferSize),
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

// ProcessReceipt processes a single receipt
func (p *DefaultProcessor) ProcessReceipt(ctx context.Context, receipt *Receipt) error {
	if !p.running {
		return p.processReceiptSync(ctx, receipt)
	}

	select {
	case p.queue <- receipt:
		p.stats.CurrentlyPending++
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

// ProcessBatch processes multiple receipts
func (p *DefaultProcessor) ProcessBatch(ctx context.Context, receipts []*Receipt) error {
	if !p.running {
		// Process synchronously if not running
		for _, receipt := range receipts {
			if err := p.processReceiptSync(ctx, receipt); err != nil {
				return err
			}
		}
		return nil
	}

	// Process asynchronously
	for _, receipt := range receipts {
		select {
		case p.queue <- receipt:
			p.stats.CurrentlyPending++
		case <-ctx.Done():
			return ctx.Err()
		default:
			return ErrQueueFull
		}
	}

	return nil
}

// AddHandler adds a receipt handler
func (p *DefaultProcessor) AddHandler(handler ReceiptHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.handlers = append(p.handlers, handler)
	p.sortHandlersByPriority()
	p.stats.Handlers = len(p.handlers)

	p.logger.Debug("Receipt handler added", "handler_id", handler.ID(), "priority", handler.Priority())
}

// RemoveHandler removes a receipt handler
func (p *DefaultProcessor) RemoveHandler(handlerID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for i, handler := range p.handlers {
		if handler.ID() == handlerID {
			p.handlers = append(p.handlers[:i], p.handlers[i+1:]...)
			p.stats.Handlers = len(p.handlers)
			p.logger.Debug("Receipt handler removed", "handler_id", handlerID)
			break
		}
	}
}

// GetStats returns processing statistics
func (p *DefaultProcessor) GetStats() ProcessorStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.stats
}

// Start starts the processor
func (p *DefaultProcessor) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.running {
		return ErrAlreadyRunning
	}

	p.logger.Info("Starting receipt processor", "workers", p.config.Workers)

	// Start workers
	p.workers = make([]*worker, p.config.Workers)
	for i := 0; i < p.config.Workers; i++ {
		w := &worker{
			id:        i,
			processor: p,
			logger:    p.logger,
		}
		p.workers[i] = w
		p.wg.Add(1)
		go w.run(ctx)
	}

	p.running = true
	p.logger.Info("Receipt processor started")
	return nil
}

// Stop stops the processor
func (p *DefaultProcessor) Stop() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.running {
		return ErrNotRunning
	}

	p.logger.Info("Stopping receipt processor")

	close(p.stopCh)
	close(p.queue)
	p.wg.Wait()

	p.running = false
	p.logger.Info("Receipt processor stopped")
	return nil
}

// processReceiptSync processes a receipt synchronously
func (p *DefaultProcessor) processReceiptSync(ctx context.Context, receipt *Receipt) error {
	start := time.Now()

	handlers := p.getApplicableHandlers(receipt)
	if len(handlers) == 0 {
		p.logger.Debug("No handlers found for receipt", "message_id", receipt.MessageID)
		return nil
	}

	for _, handler := range handlers {
		if err := p.processWithHandler(ctx, receipt, handler); err != nil {
			p.updateStats(false, time.Since(start))
			return err
		}
	}

	p.updateStats(true, time.Since(start))
	return nil
}

// processWithHandler processes a receipt with a specific handler
func (p *DefaultProcessor) processWithHandler(ctx context.Context, receipt *Receipt, handler ReceiptHandler) error {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, p.config.ProcessTimeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt < p.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-time.After(p.config.RetryDelay):
			case <-timeoutCtx.Done():
				return timeoutCtx.Err()
			}
		}

		err := handler.Handle(timeoutCtx, receipt)
		if err == nil {
			return nil
		}

		lastErr = err
		p.logger.Debug("Handler processing failed", "handler_id", handler.ID(), "attempt", attempt+1, "error", err)
	}

	return lastErr
}

// getApplicableHandlers returns handlers that can process the receipt
func (p *DefaultProcessor) getApplicableHandlers(receipt *Receipt) []ReceiptHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var applicable []ReceiptHandler
	for _, handler := range p.handlers {
		if handler.CanHandle(receipt) {
			applicable = append(applicable, handler)
		}
	}

	return applicable
}

// sortHandlersByPriority sorts handlers by priority (descending)
func (p *DefaultProcessor) sortHandlersByPriority() {
	// Sort handlers by priority (higher priority first)
	for i := 0; i < len(p.handlers)-1; i++ {
		for j := i + 1; j < len(p.handlers); j++ {
			if p.handlers[i].Priority() < p.handlers[j].Priority() {
				p.handlers[i], p.handlers[j] = p.handlers[j], p.handlers[i]
			}
		}
	}
}

// updateStats updates processing statistics
func (p *DefaultProcessor) updateStats(success bool, duration time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if success {
		p.stats.TotalProcessed++
	} else {
		p.stats.TotalFailed++
	}

	if p.stats.CurrentlyPending > 0 {
		p.stats.CurrentlyPending--
	}

	// Update average time (simple moving average)
	totalProcessed := p.stats.TotalProcessed + p.stats.TotalFailed
	if totalProcessed > 0 {
		p.stats.AverageTime = (p.stats.AverageTime*float64(totalProcessed-1) + float64(duration.Milliseconds())) / float64(totalProcessed)
	}

	p.stats.LastProcessed = time.Now()
}

// worker represents a receipt processing worker
type worker struct {
	id        int
	processor *DefaultProcessor
	logger    logger.Logger
}

// run runs the worker loop
func (w *worker) run(ctx context.Context) {
	defer w.processor.wg.Done()

	w.logger.Debug("Receipt worker started", "worker_id", w.id)

	for {
		select {
		case receipt, ok := <-w.processor.queue:
			if !ok {
				w.logger.Debug("Receipt worker stopping", "worker_id", w.id)
				return
			}

			w.processReceipt(ctx, receipt)

		case <-w.processor.stopCh:
			w.logger.Debug("Receipt worker received stop signal", "worker_id", w.id)
			return

		case <-ctx.Done():
			w.logger.Debug("Receipt worker context cancelled", "worker_id", w.id)
			return
		}
	}
}

// processReceipt processes a single receipt
func (w *worker) processReceipt(ctx context.Context, receipt *Receipt) {
	start := time.Now()

	if err := w.processor.processReceiptSync(ctx, receipt); err != nil {
		w.logger.Error("Receipt processing failed", "worker_id", w.id, "message_id", receipt.MessageID, "error", err)
	}

	w.logger.Debug("Receipt processed", "worker_id", w.id, "message_id", receipt.MessageID, "duration", time.Since(start))
}

// Common receipt handlers

// LoggingHandler logs receipt information
type LoggingHandler struct {
	logger logger.Logger
}

// NewLoggingHandler creates a new logging handler
func NewLoggingHandler(logger logger.Logger) *LoggingHandler {
	return &LoggingHandler{logger: logger}
}

func (h *LoggingHandler) ID() string                      { return "logging" }
func (h *LoggingHandler) Priority() int                   { return 1 }
func (h *LoggingHandler) CanHandle(receipt *Receipt) bool { return true }

func (h *LoggingHandler) Handle(ctx context.Context, receipt *Receipt) error {
	h.logger.Info("Receipt processed",
		"message_id", receipt.MessageID,
		"status", receipt.Status,
		"successful", receipt.Successful,
		"failed", receipt.Failed,
		"total", receipt.Total)
	return nil
}

// MetricsHandler updates metrics based on receipts
type MetricsHandler struct {
	// Metrics implementation would go here
}

func (h *MetricsHandler) ID() string                      { return "metrics" }
func (h *MetricsHandler) Priority() int                   { return 10 }
func (h *MetricsHandler) CanHandle(receipt *Receipt) bool { return true }

func (h *MetricsHandler) Handle(ctx context.Context, receipt *Receipt) error {
	// Update metrics based on receipt
	return nil
}

// Error definitions
var (
	ErrQueueFull      = fmt.Errorf("receipt queue is full")
	ErrAlreadyRunning = fmt.Errorf("processor is already running")
	ErrNotRunning     = fmt.Errorf("processor is not running")
)

// Convenience functions

// ProcessSingle processes a single receipt with default settings
func ProcessSingle(ctx context.Context, receipt *Receipt, logger logger.Logger) error {
	processor := NewDefaultProcessor(ProcessorConfig{}, logger)
	processor.AddHandler(NewLoggingHandler(logger))
	return processor.ProcessReceipt(ctx, receipt)
}
