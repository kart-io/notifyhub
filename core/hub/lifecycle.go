package hub

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kart-io/notifyhub/logger"
)

// LifecycleManager manages the hub lifecycle
// This implements the proposal's lifecycle management
type LifecycleManager struct {
	hub      *Hub
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  atomic.Bool
	stopping atomic.Bool
	logger   logger.Interface
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(hub *Hub, logger logger.Interface) *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &LifecycleManager{
		hub:    hub,
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}
}

// Start starts the hub
func (lm *LifecycleManager) Start(ctx context.Context) error {
	if !lm.running.CompareAndSwap(false, true) {
		return fmt.Errorf("hub already running")
	}

	lm.logger.Info(ctx, "Starting hub")

	// Start background workers
	lm.wg.Add(1)
	go lm.healthCheckWorker()

	// Queue processing is handled separately by worker components

	lm.logger.Info(ctx, "Hub started")
	return nil
}

// Stop stops the hub gracefully
func (lm *LifecycleManager) Stop(timeout time.Duration) error {
	if !lm.stopping.CompareAndSwap(false, true) {
		return fmt.Errorf("hub already stopping")
	}

	lm.logger.Info(context.Background(), "Stopping hub")

	// Cancel context to signal workers to stop
	lm.cancel()

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		lm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		lm.logger.Info(context.Background(), "Hub stopped gracefully")
	case <-time.After(timeout):
		lm.logger.Warn(context.Background(), "Hub stop timeout, forcing shutdown")
		return fmt.Errorf("shutdown timeout")
	}

	lm.running.Store(false)
	return nil
}

// IsRunning returns true if the hub is running
func (lm *LifecycleManager) IsRunning() bool {
	return lm.running.Load()
}

// IsStopping returns true if the hub is stopping
func (lm *LifecycleManager) IsStopping() bool {
	return lm.stopping.Load()
}

// healthCheckWorker performs periodic health checks
func (lm *LifecycleManager) healthCheckWorker() {
	defer lm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-lm.ctx.Done():
			return
		case <-ticker.C:
			lm.performHealthCheck()
		}
	}
}

// performHealthCheck checks the health of all components
func (lm *LifecycleManager) performHealthCheck() {
	status := lm.hub.Health(lm.ctx)
	if status.Healthy {
		lm.logger.Debug(lm.ctx, "Health check passed", "status", status)
	} else {
		lm.logger.Warn(lm.ctx, "Health check failed", "status", status)
	}
}
