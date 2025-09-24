// Package core provides platform management without depending on internal packages
package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/kart-io/notifyhub/pkg/notifyhub/platform"
)

// PublicPlatformManager manages platform senders using only public interfaces
type PublicPlatformManager struct {
	senders map[string]platform.ExternalSender
	mutex   sync.RWMutex
}

// NewPlatformManager creates a new public platform manager
func NewPlatformManager() *PublicPlatformManager {
	return &PublicPlatformManager{
		senders: make(map[string]platform.ExternalSender),
	}
}

// CreateSender creates a sender for the specified platform using registered creators
func (m *PublicPlatformManager) CreateSender(platformName string, config map[string]interface{}) (platform.ExternalSender, error) {
	// Use the global platform registry
	creators := platform.GetRegisteredCreators()
	creator, exists := creators[platformName]
	if !exists {
		return nil, fmt.Errorf("platform %s not registered", platformName)
	}

	return creator(config)
}

// RegisterSender registers a sender with the manager
func (m *PublicPlatformManager) RegisterSender(sender platform.ExternalSender) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := sender.Name()
	if name == "" {
		return fmt.Errorf("sender name cannot be empty")
	}

	if _, exists := m.senders[name]; exists {
		return fmt.Errorf("sender %s already registered", name)
	}

	m.senders[name] = sender
	return nil
}

// GetSender retrieves a sender by platform name
func (m *PublicPlatformManager) GetSender(platform string) (platform.ExternalSender, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sender, exists := m.senders[platform]
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
	Target    platform.Target        `json:"target"`
	Success   bool                   `json:"success"`
	MessageID string                 `json:"message_id,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Response  string                 `json:"response,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SendToAll sends a message to all targets across all relevant platforms
func (m *PublicPlatformManager) SendToAll(ctx context.Context, msg *platform.Message, targets []platform.Target) ([]*LocalSendResult, error) {
	// Group targets by platform
	platformTargets := make(map[string][]platform.Target)
	for _, target := range targets {
		platformName := target.Platform
		platformTargets[platformName] = append(platformTargets[platformName], target)
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
		go func(platform string, targets []platform.Target) {
			defer wg.Done()

			sender, exists := m.GetSender(platform)
			if !exists {
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
			results, err := sender.Send(ctx, msg, targets)

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

	return allResults, lastError
}

// HealthCheck checks the health of all registered senders
func (m *PublicPlatformManager) HealthCheck(ctx context.Context) map[string]error {
	m.mutex.RLock()
	senders := make(map[string]platform.ExternalSender)
	for name, sender := range m.senders {
		senders[name] = sender
	}
	m.mutex.RUnlock()

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
			mutex.Unlock()
		}(name, sender)
	}

	wg.Wait()
	return health
}

// Close shuts down all senders
func (m *PublicPlatformManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var lastError error
	for name, sender := range m.senders {
		if err := sender.Close(); err != nil {
			lastError = fmt.Errorf("failed to close sender %s: %w", name, err)
		}
	}

	// Clear the senders map
	m.senders = make(map[string]platform.ExternalSender)

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
