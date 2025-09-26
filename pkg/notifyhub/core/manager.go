// Package core provides platform management without depending on internal packages
package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/kart-io/notifyhub/pkg/logger"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// PublicPlatformManager manages platform senders using only public interfaces
type PublicPlatformManager struct {
	senders map[string]platform.ExternalSender
	mutex   sync.RWMutex
	logger  logger.Logger
}

// NewPublicPlatformManager creates a new public platform manager (deprecated)
func NewPublicPlatformManager() *PublicPlatformManager {
	return &PublicPlatformManager{
		senders: make(map[string]platform.ExternalSender),
		logger:  logger.New(), // Default to nop logger
	}
}

// SetLogger sets the logger for the platform manager
func (m *PublicPlatformManager) SetLogger(l logger.Logger) {
	if l != nil {
		m.logger = l
		m.logger.Debug("Logger set for platform manager")
	}
}

// CreateSender creates a sender for the specified platform using registered creators
func (m *PublicPlatformManager) CreateSender(platformName string, config map[string]interface{}, logger logger.Logger) (platform.ExternalSender, error) {
	m.logger.Debug("Creating sender for platform", "platform", platformName)

	// Use the global platform registry
	creators := platform.GetRegisteredCreators()
	creator, exists := creators[platformName]
	if !exists {
		m.logger.Error("Platform not registered", "platform", platformName)
		return nil, fmt.Errorf("platform %s not registered", platformName)
	}

	sender, err := creator(config, logger)
	if err != nil {
		m.logger.Error("Failed to create sender", "platform", platformName, "error", err)
		return nil, err
	}

	m.logger.Info("Successfully created sender", "platform", platformName)
	return sender, nil
}

// RegisterSender registers a sender with the manager
func (m *PublicPlatformManager) RegisterSender(sender platform.ExternalSender) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := sender.Name()
	if name == "" {
		m.logger.Error("Attempted to register sender with empty name")
		return fmt.Errorf("sender name cannot be empty")
	}

	if _, exists := m.senders[name]; exists {
		m.logger.Warn("Sender already registered", "platform", name)
		return fmt.Errorf("sender %s already registered", name)
	}

	m.senders[name] = sender
	m.logger.Info("Registered sender", "platform", name)
	return nil
}

// GetSender retrieves a sender by platform name
func (m *PublicPlatformManager) GetSender(platform string) (platform.ExternalSender, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sender, exists := m.senders[platform]
	if exists {
		m.logger.Debug("Retrieved sender", "platform", platform)
	} else {
		m.logger.Debug("Sender not found", "platform", platform)
	}
	return sender, exists
}

// ListSenders returns all registered sender names
func (m *PublicPlatformManager) ListSenders() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	names := make([]string, 0, len(m.senders))
	for name := range m.senders {
		names = append(names, name)
	}
	return names
}

// LocalSendResult is a local copy of SendResult to avoid import issues
type LocalSendResult struct {
	Target    target.Target          `json:"target"`
	Success   bool                   `json:"success"`
	MessageID string                 `json:"message_id,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Response  string                 `json:"response,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SendToAll sends a message to all targets across all relevant platforms
func (m *PublicPlatformManager) SendToAll(ctx context.Context, msg *message.Message, targets []target.Target) ([]*LocalSendResult, error) {
	m.logger.Debug("Starting SendToAll", "messageID", msg.ID, "targetCount", len(targets))

	// Group targets by platform
	platformTargets := make(map[string][]target.Target)
	for _, target := range targets {
		platformName := target.Platform
		platformTargets[platformName] = append(platformTargets[platformName], target)
	}

	m.logger.Debug("Grouped targets by platform", "messageID", msg.ID, "platformCount", len(platformTargets))
	for platform, targets := range platformTargets {
		m.logger.Debug("Platform target group", "messageID", msg.ID, "platform", platform, "targetCount", len(targets))
	}

	// Send to each platform in parallel
	type platformResult struct {
		platform string
		results  []*LocalSendResult
		err      error
	}

	resultChan := make(chan platformResult, len(platformTargets))
	var wg sync.WaitGroup

	for platformName, platformTargetList := range platformTargets {
		wg.Add(1)
		go func(platform string, targets []target.Target) {
			defer wg.Done()

			sender, exists := m.GetSender(platform)
			if !exists {
				m.logger.Error("Platform sender not found", "platform", platform, "messageID", msg.ID)
				// Create failed results for each target
				failedResults := make([]*LocalSendResult, len(targets))
				for i, target := range targets {
					failedResults[i] = &LocalSendResult{
						Target:  target,
						Success: false,
						Error:   fmt.Sprintf("platform %s not found", platform),
					}
				}
				resultChan <- platformResult{platform: platform, results: failedResults}
				return
			}

			// Send through platform
			m.logger.Debug("Sending message through platform", "platform", platform, "messageID", msg.ID, "targetCount", len(targets))
			results, err := sender.Send(ctx, msg, targets)

			if err != nil {
				m.logger.Error("Platform send failed", "platform", platform, "messageID", msg.ID, "error", err)
			} else {
				successCount := 0
				for _, r := range results {
					if r.Success {
						successCount++
					}
				}
				m.logger.Info("Platform send completed", "platform", platform, "messageID", msg.ID,
					"totalTargets", len(targets), "successCount", successCount)
			}

			// Convert platform.SendResult to LocalSendResult
			var localResults []*LocalSendResult
			if results != nil {
				localResults = make([]*LocalSendResult, len(results))
				for i, r := range results {
					localResults[i] = &LocalSendResult{
						Target:    r.Target,
						Success:   r.Success,
						MessageID: r.MessageID,
						Error:     r.Error,
						Response:  r.Response,
						Metadata:  r.Metadata,
					}
				}
			}

			resultChan <- platformResult{platform: platform, results: localResults, err: err}
		}(platformName, platformTargetList)
	}

	// Wait for all platforms to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var allResults []*LocalSendResult
	var lastError error

	for result := range resultChan {
		if result.err != nil {
			lastError = result.err
			m.logger.Error("Platform result with error", "platform", result.platform, "error", result.err)
			// Create failed results if platform send failed completely
			if result.results == nil {
				platformTargetList := platformTargets[result.platform]
				failedResults := make([]*LocalSendResult, len(platformTargetList))
				for i, target := range platformTargetList {
					failedResults[i] = &LocalSendResult{
						Target:  target,
						Success: false,
						Error:   result.err.Error(),
					}
				}
				allResults = append(allResults, failedResults...)
			}
		} else {
			allResults = append(allResults, result.results...)
		}
	}

	m.logger.Debug("SendToAll completed", "messageID", msg.ID, "totalResults", len(allResults), "hasError", lastError != nil)

	return allResults, lastError
}

// HealthCheck checks the health of all registered senders
func (m *PublicPlatformManager) HealthCheck(ctx context.Context) map[string]error {
	m.logger.Debug("Starting health check for all platforms")

	m.mutex.RLock()
	senders := make(map[string]platform.ExternalSender)
	for name, sender := range m.senders {
		senders[name] = sender
	}
	m.mutex.RUnlock()

	m.logger.Debug("Checking health for platforms", "count", len(senders))

	health := make(map[string]error)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for name, sender := range senders {
		wg.Add(1)
		go func(name string, sender platform.ExternalSender) {
			defer wg.Done()

			err := sender.IsHealthy(ctx)
			mutex.Lock()
			health[name] = err
			if err != nil {
				m.logger.Warn("Platform health check failed", "platform", name, "error", err)
			} else {
				m.logger.Debug("Platform health check passed", "platform", name)
			}
			mutex.Unlock()
		}(name, sender)
	}

	wg.Wait()
	m.logger.Debug("Health check completed", "platforms", len(health))
	return health
}

// Close shuts down all senders
func (m *PublicPlatformManager) Close() error {
	m.logger.Info("Closing platform manager")

	m.mutex.Lock()
	defer m.mutex.Unlock()

	var lastError error
	for name, sender := range m.senders {
		m.logger.Debug("Closing sender", "platform", name)
		if err := sender.Close(); err != nil {
			m.logger.Error("Failed to close sender", "platform", name, "error", err)
			lastError = fmt.Errorf("failed to close sender %s: %w", name, err)
		} else {
			m.logger.Debug("Successfully closed sender", "platform", name)
		}
	}

	// Clear the senders map
	m.senders = make(map[string]platform.ExternalSender)
	m.logger.Info("Platform manager closed")

	return lastError
}

// GetRegisteredPlatforms returns a list of all registered platform names
func GetRegisteredPlatforms() []string {
	creators := platform.GetRegisteredCreators()
	names := make([]string, 0, len(creators))
	for name := range creators {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a platform is registered
func IsRegistered(platformName string) bool {
	creators := platform.GetRegisteredCreators()
	_, exists := creators[platformName]
	return exists
}
