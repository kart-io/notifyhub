package platform

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// manager implements the Manager interface for managing platform senders
type manager struct {
	senders   map[string]Sender
	mutex     sync.RWMutex
	factory   SenderFactory
	resolver  TargetResolver
	converter MessageConverter
	validator Validator
}

// NewManager creates a new platform manager
func NewManager(factory SenderFactory, resolver TargetResolver, converter MessageConverter, validator Validator) Manager {
	return &manager{
		senders:   make(map[string]Sender),
		factory:   factory,
		resolver:  resolver,
		converter: converter,
		validator: validator,
	}
}

// RegisterSender registers a sender with the manager
func (m *manager) RegisterSender(sender Sender) error {
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
func (m *manager) GetSender(platform string) (Sender, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sender, exists := m.senders[platform]
	return sender, exists
}

// ListSenders returns all registered sender names
func (m *manager) ListSenders() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	names := make([]string, 0, len(m.senders))
	for name := range m.senders {
		names = append(names, name)
	}
	return names
}

// SendToAll sends a message to all targets across all relevant platforms
func (m *manager) SendToAll(ctx context.Context, msg *InternalMessage, targets []InternalTarget) ([]*SendResult, error) {
	// Group targets by platform
	platformTargets := make(map[string][]InternalTarget)
	for _, target := range targets {
		platform := target.Platform
		platformTargets[platform] = append(platformTargets[platform], target)
	}

	// Send to each platform in parallel
	type platformResult struct {
		platform string
		results  []*SendResult
		err      error
	}

	resultChan := make(chan platformResult, len(platformTargets))
	var wg sync.WaitGroup

	for platform, platformTargetList := range platformTargets {
		wg.Add(1)
		go func(platform string, targets []InternalTarget) {
			defer wg.Done()

			sender, exists := m.GetSender(platform)
			if !exists {
				// Create a failed result for each target
				failedResults := make([]*SendResult, len(targets))
				for i, target := range targets {
					result := NewSendResult(target, false)
					result.Error = fmt.Sprintf("platform %s not found", platform)
					failedResults[i] = result
				}
				resultChan <- platformResult{platform: platform, results: failedResults}
				return
			}

			// Send through platform
			results, err := sender.Send(ctx, msg, targets)
			resultChan <- platformResult{platform: platform, results: results, err: err}
		}(platform, platformTargetList)
	}

	// Wait for all platforms to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var allResults []*SendResult
	var lastError error

	for result := range resultChan {
		if result.err != nil {
			lastError = result.err
			// Create failed results if platform send failed completely
			if result.results == nil {
				platformTargetList := platformTargets[result.platform]
				failedResults := make([]*SendResult, len(platformTargetList))
				for i, target := range platformTargetList {
					sendResult := NewSendResult(target, false)
					sendResult.Error = result.err.Error()
					failedResults[i] = sendResult
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
func (m *manager) HealthCheck(ctx context.Context) map[string]error {
	m.mutex.RLock()
	senders := make(map[string]Sender)
	for name, sender := range m.senders {
		senders[name] = sender
	}
	m.mutex.RUnlock()

	health := make(map[string]error)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for name, sender := range senders {
		wg.Add(1)
		go func(name string, sender Sender) {
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
func (m *manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var lastError error
	for name, sender := range m.senders {
		if err := sender.Close(); err != nil {
			lastError = fmt.Errorf("failed to close sender %s: %w", name, err)
		}
	}

	// Clear the senders map
	m.senders = make(map[string]Sender)

	return lastError
}

// SenderCreator is a function type that creates a sender with given configuration
type SenderCreator func(config map[string]interface{}) (Sender, error)

// Global registry for sender creators
var senderRegistry = make(map[string]SenderCreator)

// RegisterSenderCreator registers a sender creator for a platform
func RegisterSenderCreator(platform string, creator SenderCreator) {
	senderRegistry[platform] = creator
}

// defaultSenderFactory implements SenderFactory with support for common platforms
type defaultSenderFactory struct{}

// NewDefaultSenderFactory creates a default sender factory
func NewDefaultSenderFactory() SenderFactory {
	return &defaultSenderFactory{}
}

// CreateSender creates a new sender for the specified platform
func (f *defaultSenderFactory) CreateSender(platformName string, config map[string]interface{}) (Sender, error) {
	// First check if there's a registered creator for this platform
	if creator, exists := senderRegistry[platformName]; exists {
		return creator(config)
	}

	// Fall back to built-in implementations
	switch platformName {
	case "email":
		// Create email sender using the specific implementation
		return createEmailSender(config)
	case "feishu":
		// Create feishu sender using the specific implementation
		return createFeishuSender(config)
	case "sms":
		// Create SMS sender using the specific implementation
		return createSMSSender(config)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platformName)
	}
}

// GetSupportedPlatforms returns a list of platforms this factory can create
func (f *defaultSenderFactory) GetSupportedPlatforms() []string {
	return []string{"email", "feishu", "sms"}
}

// ValidateConfig validates configuration for a platform
func (f *defaultSenderFactory) ValidateConfig(platform string, config map[string]interface{}) error {
	switch platform {
	case "email":
		required := []string{"smtp_host", "smtp_port", "smtp_username", "smtp_password", "smtp_from"}
		return validateRequiredFields(config, required)
	case "feishu":
		required := []string{"webhook_url"}
		return validateRequiredFields(config, required)
	case "sms":
		required := []string{"provider", "api_key"}
		return validateRequiredFields(config, required)
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

// Factory methods have been removed - platform senders are created directly

// defaultMessageConverter implements MessageConverter
type defaultMessageConverter struct{}

// NewDefaultMessageConverter creates a default message converter
func NewDefaultMessageConverter() MessageConverter {
	return &defaultMessageConverter{}
}

// Convert performs placeholder conversion
func (c *defaultMessageConverter) Convert(input interface{}) (interface{}, error) {
	// Placeholder implementation - actual conversion logic would go here
	return input, nil
}

// ToInternal converts a message to internal format
func (c *defaultMessageConverter) ToInternal(message interface{}, platform string) (*InternalMessage, error) {
	// We need to import the Message type, but to avoid import cycles,
	// we'll use interface{} and type assertion
	msg, ok := message.(*InternalMessage)
	if ok {
		// Already converted
		return msg, nil
	}

	// For now, create a basic conversion
	// In a real implementation, this would convert from public Message to InternalMessage
	return &InternalMessage{
		ID:           generateMessageID(),
		Title:        "Converted Message",
		Body:         "Message body",
		Format:       "text",
		Priority:     2,
		Metadata:     make(map[string]interface{}),
		Variables:    make(map[string]interface{}),
		PlatformData: make(map[string]interface{}),
	}, nil
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// defaultValidator implements Validator
type defaultValidator struct{}

// NewDefaultValidator creates a default validator
func NewDefaultValidator() Validator {
	return &defaultValidator{}
}

// ValidateMessage validates a message for general requirements
func (v *defaultValidator) ValidateMessage(msg *InternalMessage) error {
	if msg.Title == "" && msg.Body == "" {
		return fmt.Errorf("message must have either title or body")
	}
	return nil
}

// ValidateTarget validates a target format
func (v *defaultValidator) ValidateTarget(target InternalTarget) error {
	if target.Value == "" {
		return fmt.Errorf("target value cannot be empty")
	}
	if target.Type == "" {
		return fmt.Errorf("target type cannot be empty")
	}
	if target.Platform == "" {
		return fmt.Errorf("target platform cannot be empty")
	}
	return nil
}

// ValidateMessageForPlatform validates a message for a specific platform
func (v *defaultValidator) ValidateMessageForPlatform(msg *InternalMessage, platform string, capabilities PlatformCapabilities) error {
	// Check message size
	if capabilities.MaxMessageSize > 0 {
		messageSize := len(msg.Title) + len(msg.Body)
		if messageSize > capabilities.MaxMessageSize {
			return fmt.Errorf("message size (%d bytes) exceeds platform limit (%d bytes)", messageSize, capabilities.MaxMessageSize)
		}
	}

	// Check format support
	if len(capabilities.SupportedFormats) > 0 {
		supported := false
		for _, format := range capabilities.SupportedFormats {
			if format == msg.Format {
				supported = true
				break
			}
		}
		if !supported {
			return fmt.Errorf("format %s not supported by platform %s", msg.Format, platform)
		}
	}

	return nil
}

// defaultTargetResolver implements TargetResolver
type defaultTargetResolver struct{}

// NewDefaultTargetResolver creates a default target resolver
func NewDefaultTargetResolver() TargetResolver {
	return &defaultTargetResolver{}
}

// Resolve performs placeholder resolution
func (r *defaultTargetResolver) Resolve(input interface{}) (interface{}, error) {
	// Placeholder implementation - actual resolution logic would go here
	return input, nil
}

// ResolveTargets determines which platform should handle each target
func (r *defaultTargetResolver) ResolveTargets(targets interface{}) map[string][]InternalTarget {
	// For now, create a basic implementation that handles common target types
	result := make(map[string][]InternalTarget)

	// Assuming targets is a slice, try to convert it
	if targetSlice, ok := targets.([]interface{}); ok {
		for _, target := range targetSlice {
			if targetMap, ok := target.(map[string]interface{}); ok {
				targetType, _ := targetMap["type"].(string)
				targetValue, _ := targetMap["value"].(string)
				platform, _ := targetMap["platform"].(string)

				if platform == "" {
					// Auto-detect platform based on target type
					switch targetType {
					case "email":
						platform = "email"
					case "phone":
						platform = "sms"
					case "user", "group", "webhook":
						platform = "feishu"
					}
				}

				if platform != "" {
					internalTarget := InternalTarget{
						Type:     targetType,
						Value:    targetValue,
						Platform: platform,
					}
					result[platform] = append(result[platform], internalTarget)
				}
			}
		}
	}

	return result
}

// ValidateTargetForPlatform checks if a target is valid for a platform
func (r *defaultTargetResolver) ValidateTargetForPlatform(target InternalTarget, platform string) error {
	switch platform {
	case "email":
		if target.Type != "email" {
			return fmt.Errorf("email platform only supports email targets")
		}
	case "sms":
		if target.Type != "phone" {
			return fmt.Errorf("sms platform only supports phone targets")
		}
	case "feishu":
		if target.Type != "user" && target.Type != "group" && target.Type != "webhook" {
			return fmt.Errorf("feishu platform only supports user, group, and webhook targets")
		}
	}
	return nil
}

// Utility functions

// validateRequiredFields validates that required fields are present in config
func validateRequiredFields(config map[string]interface{}, required []string) error {
	for _, field := range required {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("required field %s is missing", field)
		}
	}
	return nil
}

// Platform-specific sender creation functions

// createEmailSender creates an email sender with the given configuration
func createEmailSender(config map[string]interface{}) (Sender, error) {
	// Import the actual email sender implementation
	// We need to avoid import cycles, so we'll use internal imports
	// For now, create a basic implementation

	// Extract required configuration
	host, _ := config["smtp_host"].(string)
	port, _ := config["smtp_port"].(int)
	username, _ := config["smtp_username"].(string)
	password, _ := config["smtp_password"].(string)
	from, _ := config["smtp_from"].(string)
	useTLS, _ := config["smtp_tls"].(bool)

	if host == "" || username == "" || password == "" || from == "" {
		return nil, fmt.Errorf("missing required email configuration fields")
	}

	// Return a basic email sender implementation
	return &basicEmailSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		useTLS:   useTLS,
	}, nil
}

// createFeishuSender creates a feishu sender with the given configuration
func createFeishuSender(config map[string]interface{}) (Sender, error) {
	// This is a fallback implementation, the real feishu sender should be registered
	// during package initialization. This should not be called in normal operation.
	webhookURL, _ := config["webhook_url"].(string)
	if webhookURL == "" {
		return nil, fmt.Errorf("missing required feishu configuration: webhook_url")
	}
	return nil, fmt.Errorf("feishu sender not registered - ensure feishu package is imported")
}

// createSMSSender creates an SMS sender with the given configuration
func createSMSSender(config map[string]interface{}) (Sender, error) {
	provider, _ := config["provider"].(string)
	apiKey, _ := config["api_key"].(string)

	if provider == "" || apiKey == "" {
		return nil, fmt.Errorf("missing required SMS configuration fields")
	}

	return &basicSMSSender{
		provider: provider,
		apiKey:   apiKey,
	}, nil
}

// Basic sender implementations (placeholder implementations for now)

type basicEmailSender struct {
	host     string
	port     int
	username string
	password string
	from     string
	useTLS   bool
}

func (s *basicEmailSender) Name() string { return "email" }

func (s *basicEmailSender) Send(ctx context.Context, msg *InternalMessage, targets []InternalTarget) ([]*SendResult, error) {
	results := make([]*SendResult, len(targets))
	for i, target := range targets {
		result := NewSendResult(target, true)
		result.MessageID = fmt.Sprintf("email_%s_%d", msg.ID, i)
		// For demo purposes, mark as successful
		results[i] = result
	}
	return results, nil
}

func (s *basicEmailSender) ValidateTarget(target InternalTarget) error {
	if target.Type != "email" {
		return fmt.Errorf("email sender only supports email targets")
	}
	return nil
}

func (s *basicEmailSender) GetCapabilities() PlatformCapabilities {
	return PlatformCapabilities{
		Name:                 "email",
		SupportedTargetTypes: []string{"email"},
		SupportedFormats:     []string{"text", "html"},
		MaxMessageSize:       1024 * 1024, // 1MB
		SupportsScheduling:   true,
		SupportsAttachments:  true,
		RequiredSettings:     []string{"smtp_host", "smtp_port", "smtp_username", "smtp_password", "smtp_from"},
	}
}

func (s *basicEmailSender) IsHealthy(ctx context.Context) error { return nil }
func (s *basicEmailSender) Close() error                        { return nil }

type basicSMSSender struct {
	provider string
	apiKey   string
}

func (s *basicSMSSender) Name() string { return "sms" }

func (s *basicSMSSender) Send(ctx context.Context, msg *InternalMessage, targets []InternalTarget) ([]*SendResult, error) {
	results := make([]*SendResult, len(targets))
	for i, target := range targets {
		result := NewSendResult(target, true)
		result.MessageID = fmt.Sprintf("sms_%s_%d", msg.ID, i)
		// For demo purposes, mark as successful
		results[i] = result
	}
	return results, nil
}

func (s *basicSMSSender) ValidateTarget(target InternalTarget) error {
	if target.Type != "phone" {
		return fmt.Errorf("SMS sender only supports phone targets")
	}
	return nil
}

func (s *basicSMSSender) GetCapabilities() PlatformCapabilities {
	return PlatformCapabilities{
		Name:                 "sms",
		SupportedTargetTypes: []string{"phone"},
		SupportedFormats:     []string{"text"},
		MaxMessageSize:       160, // Standard SMS limit
		SupportsScheduling:   true,
		RequiredSettings:     []string{"provider", "api_key"},
	}
}

func (s *basicSMSSender) IsHealthy(ctx context.Context) error { return nil }
func (s *basicSMSSender) Close() error                        { return nil }
