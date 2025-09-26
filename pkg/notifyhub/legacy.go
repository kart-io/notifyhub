// Package notifyhub provides backward compatibility for existing NotifyHub users
// This file contains legacy API wrappers to ensure existing code continues to work
package notifyhub

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"github.com/kart-io/notifyhub/pkg/notifyhub/message"
	"github.com/kart-io/notifyhub/pkg/notifyhub/receipt"
	"github.com/kart-io/notifyhub/pkg/notifyhub/target"
)

// Deprecated: Legacy types and interfaces for backward compatibility
// These will be removed in a future major version

// LegacyHub represents the old hub interface (deprecated)
// Users should migrate to the new NotifyHub struct
type LegacyHub interface {
	Send(ctx context.Context, msg LegacyMessage) (*LegacyReceipt, error)
	SendBatch(ctx context.Context, msgs []LegacyMessage) ([]*LegacyReceipt, error)
	AddPlatform(name string, config map[string]interface{}) error
	RemovePlatform(name string) error
	Close() error
}

// LegacyMessage represents the old message format (deprecated)
type LegacyMessage struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Content  string                 `json:"content"` // Old name for Body
	Format   string                 `json:"format"`
	Priority int                    `json:"priority"`
	Targets  []LegacyTarget         `json:"targets"`
	Data     map[string]interface{} `json:"data"` // Old name for PlatformData
}

// LegacyTarget represents the old target format (deprecated)
type LegacyTarget struct {
	Type     string                 `json:"type"`
	Address  string                 `json:"address"` // Old name for Value
	Metadata map[string]interface{} `json:"metadata"`
}

// LegacyReceipt represents the old receipt format (deprecated)
type LegacyReceipt struct {
	MessageID string         `json:"message_id"`
	Status    string         `json:"status"`
	Results   []LegacyResult `json:"results"`
	Success   int            `json:"success"` // Old name for Successful
	Failure   int            `json:"failure"` // Old name for Failed
	Total     int            `json:"total"`
	Timestamp time.Time      `json:"timestamp"`
	Error     string         `json:"error,omitempty"`
}

// LegacyResult represents the old platform result format (deprecated)
type LegacyResult struct {
	Platform  string        `json:"platform"`
	Target    string        `json:"target"`
	Success   bool          `json:"success"`
	MessageID string        `json:"message_id,omitempty"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}

// LegacyHubWrapper wraps the new Client to provide legacy API compatibility
type LegacyHubWrapper struct {
	client Client
}

// CreateHub creates a new hub with legacy configuration format (deprecated)
// Users should migrate to notifyhub.New() with strong-typed configuration
func CreateHub(platforms map[string]map[string]interface{}) (LegacyHub, error) {
	logDeprecationWarning("CreateHub", "notifyhub.New() with strong-typed configuration")

	// Convert legacy platform configurations to new format
	var opts []config.Option

	for platformName, platformConfig := range platforms {
		switch platformName {
		case "email":
			emailConfig, err := convertLegacyEmailConfig(platformConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to convert email config: %w", err)
			}
			opts = append(opts, config.WithEmail(*emailConfig))

		case "feishu":
			feishuConfig, err := convertLegacyFeishuConfig(platformConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to convert feishu config: %w", err)
			}
			opts = append(opts, config.WithFeishu(*feishuConfig))

		case "sms":
			smsConfig, err := convertLegacySMSConfig(platformConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to convert sms config: %w", err)
			}
			opts = append(opts, config.WithSMS(*smsConfig))

		default:
			// Use legacy platform support for unknown platforms
			opts = append(opts, config.WithPlatform(platformName, platformConfig))
		}
	}

	// Convert config.Option to Option for the New function
	hubOpts := make([]Option, len(opts))
	for i, opt := range opts {
		// Both Option types have the same signature: func(*config.Config) error
		hubOpts[i] = Option(opt)
	}

	// Create new client
	c, err := New(hubOpts...)
	if err != nil {
		return nil, err
	}

	return &LegacyHubWrapper{client: c}, nil
}

// Send implements the legacy send interface
func (lhw *LegacyHubWrapper) Send(ctx context.Context, msg LegacyMessage) (*LegacyReceipt, error) {
	// Convert legacy message to new format
	newMsg := convertLegacyMessage(msg)

	// Send using new API
	newReceipt, err := lhw.client.Send(ctx, newMsg)
	if err != nil {
		return nil, err
	}

	// Convert new receipt back to legacy format
	return convertReceiptToLegacy(newReceipt), nil
}

// SendBatch implements the legacy batch send interface
func (lhw *LegacyHubWrapper) SendBatch(ctx context.Context, msgs []LegacyMessage) ([]*LegacyReceipt, error) {
	// Convert legacy messages to new format
	newMsgs := make([]*message.Message, len(msgs))
	for i, msg := range msgs {
		newMsgs[i] = convertLegacyMessage(msg)
	}

	// Send using new API
	newReceipts, err := lhw.client.SendBatch(ctx, newMsgs)
	if err != nil {
		return nil, err
	}

	// Convert new receipts back to legacy format
	legacyReceipts := make([]*LegacyReceipt, len(newReceipts))
	for i, receipt := range newReceipts {
		if receipt != nil {
			legacyReceipts[i] = convertReceiptToLegacy(receipt)
		}
	}

	return legacyReceipts, nil
}

// AddPlatform adds a platform using legacy configuration (deprecated)
func (lhw *LegacyHubWrapper) AddPlatform(name string, config map[string]interface{}) error {
	logDeprecationWarning("AddPlatform", "configure platforms during hub creation with strong-typed config")

	// This is limited in the new architecture since platforms are configured at creation time
	// We log a warning and return an error suggesting proper migration
	return fmt.Errorf("AddPlatform is deprecated and not supported in the new architecture. Please configure platforms during hub creation with notifyhub.New()")
}

// RemovePlatform removes a platform (deprecated)
func (lhw *LegacyHubWrapper) RemovePlatform(name string) error {
	logDeprecationWarning("RemovePlatform", "create a new hub with the desired platform configuration")

	// This is not supported in the new architecture
	return fmt.Errorf("RemovePlatform is deprecated and not supported in the new architecture. Please create a new hub with the desired platform configuration")
}

// Close closes the client
func (lhw *LegacyHubWrapper) Close() error {
	return lhw.client.Close()
}

// Conversion functions

// convertLegacyMessage converts a legacy message to the new format
func convertLegacyMessage(legacyMsg LegacyMessage) *message.Message {
	// Convert legacy targets to new format
	newTargets := make([]target.Target, len(legacyMsg.Targets))
	for i, legacyTarget := range legacyMsg.Targets {
		newTargets[i] = target.Target{
			Type:  legacyTarget.Type,
			Value: legacyTarget.Address, // Address -> Value
		}
	}

	return &message.Message{
		ID:           legacyMsg.ID,
		Title:        legacyMsg.Title,
		Body:         legacyMsg.Content, // Content -> Body
		Format:       message.Format(legacyMsg.Format),
		Priority:     message.Priority(legacyMsg.Priority),
		Targets:      newTargets,
		PlatformData: legacyMsg.Data, // Data -> PlatformData
		Metadata:     make(map[string]interface{}),
	}
}

// convertReceiptToLegacy converts a new receipt to legacy format
func convertReceiptToLegacy(newReceipt *receipt.Receipt) *LegacyReceipt {
	if newReceipt == nil {
		return nil
	}

	// Convert platform results to legacy format
	legacyResults := make([]LegacyResult, len(newReceipt.Results))
	for i, result := range newReceipt.Results {
		legacyResults[i] = LegacyResult{
			Platform:  result.Platform,
			Target:    result.Target,
			Success:   result.Success,
			MessageID: result.MessageID,
			Error:     result.Error,
			Timestamp: result.Timestamp,
			Duration:  result.Duration,
		}
	}

	errorStr := ""
	if newReceipt.Error != nil {
		errorStr = newReceipt.Error.Error()
	}

	return &LegacyReceipt{
		MessageID: newReceipt.MessageID,
		Status:    newReceipt.Status,
		Results:   legacyResults,
		Success:   newReceipt.Successful, // Successful -> Success
		Failure:   newReceipt.Failed,     // Failed -> Failure
		Total:     newReceipt.Total,
		Timestamp: newReceipt.Timestamp,
		Error:     errorStr,
	}
}

// convertLegacyEmailConfig converts legacy email config to new format
func convertLegacyEmailConfig(legacyConfig map[string]interface{}) (*config.EmailConfig, error) {
	emailConfig := &config.EmailConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RateLimit:  60,
	}

	if host, ok := legacyConfig["smtp_host"].(string); ok {
		emailConfig.SMTPHost = host
	} else {
		return nil, fmt.Errorf("smtp_host is required")
	}

	if port, ok := legacyConfig["smtp_port"].(int); ok {
		emailConfig.SMTPPort = port
	} else {
		return nil, fmt.Errorf("smtp_port is required")
	}

	if from, ok := legacyConfig["smtp_from"].(string); ok {
		emailConfig.SMTPFrom = from
	} else {
		return nil, fmt.Errorf("smtp_from is required")
	}

	if username, ok := legacyConfig["smtp_username"].(string); ok {
		emailConfig.SMTPUsername = username
	}

	if password, ok := legacyConfig["smtp_password"].(string); ok {
		emailConfig.SMTPPassword = password
	}

	if tls, ok := legacyConfig["smtp_tls"].(bool); ok {
		emailConfig.SMTPTLS = tls
	}

	if ssl, ok := legacyConfig["smtp_ssl"].(bool); ok {
		emailConfig.SMTPSSL = ssl
	}

	if timeout, ok := legacyConfig["timeout"].(time.Duration); ok {
		emailConfig.Timeout = timeout
	}

	return emailConfig, nil
}

// convertLegacyFeishuConfig converts legacy Feishu config to new format
func convertLegacyFeishuConfig(legacyConfig map[string]interface{}) (*config.FeishuConfig, error) {
	feishuConfig := &config.FeishuConfig{
		AuthType:   "webhook",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RateLimit:  100,
		SignVerify: false,
	}

	if webhookURL, ok := legacyConfig["webhook_url"].(string); ok {
		feishuConfig.WebhookURL = webhookURL
	} else {
		return nil, fmt.Errorf("webhook_url is required")
	}

	if secret, ok := legacyConfig["secret"].(string); ok && secret != "" {
		feishuConfig.Secret = secret
		feishuConfig.SignVerify = true
	}

	if appID, ok := legacyConfig["app_id"].(string); ok {
		feishuConfig.AppID = appID
		feishuConfig.AuthType = "app"
	}

	if appSecret, ok := legacyConfig["app_secret"].(string); ok {
		feishuConfig.AppSecret = appSecret
	}

	if timeout, ok := legacyConfig["timeout"].(time.Duration); ok {
		feishuConfig.Timeout = timeout
	}

	return feishuConfig, nil
}

// convertLegacySMSConfig converts legacy SMS config to new format
func convertLegacySMSConfig(legacyConfig map[string]interface{}) (*config.SMSConfig, error) {
	smsConfig := &config.SMSConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RateLimit:  30,
		Templates:  make(map[string]string),
	}

	if provider, ok := legacyConfig["provider"].(string); ok {
		smsConfig.Provider = provider
	} else {
		return nil, fmt.Errorf("provider is required")
	}

	if apiKey, ok := legacyConfig["api_key"].(string); ok {
		smsConfig.APIKey = apiKey
	} else {
		return nil, fmt.Errorf("api_key is required")
	}

	if from, ok := legacyConfig["from"].(string); ok {
		smsConfig.From = from
	} else {
		return nil, fmt.Errorf("from is required")
	}

	if apiSecret, ok := legacyConfig["api_secret"].(string); ok {
		smsConfig.APISecret = apiSecret
	}

	if region, ok := legacyConfig["region"].(string); ok {
		smsConfig.Region = region
	}

	if timeout, ok := legacyConfig["timeout"].(time.Duration); ok {
		smsConfig.Timeout = timeout
	}

	if templates, ok := legacyConfig["templates"].(map[string]string); ok {
		smsConfig.Templates = templates
	}

	return smsConfig, nil
}

// Utility functions

// logDeprecationWarning logs deprecation warnings to help users migrate
func logDeprecationWarning(deprecatedAPI, replacement string) {
	log.Printf("[DEPRECATED] %s is deprecated and will be removed in a future version. Please use %s instead. See migration guide: https://github.com/your-org/notifyhub/blob/main/MIGRATION.md", deprecatedAPI, replacement)
}

// Global legacy functions for backward compatibility

var legacyGlobalHub LegacyHub

// SetGlobalHub sets the global legacy hub (deprecated)
func SetGlobalHub(hub LegacyHub) {
	logDeprecationWarning("SetGlobalHub", "create explicit hub instances")
	legacyGlobalHub = hub
}

// GetGlobalHub returns the global legacy hub (deprecated)
func GetGlobalHub() LegacyHub {
	logDeprecationWarning("GetGlobalHub", "create explicit hub instances")
	return legacyGlobalHub
}

// Send sends a message using the global hub (deprecated)
func LegacySend(ctx context.Context, msg LegacyMessage) (*LegacyReceipt, error) {
	logDeprecationWarning("LegacySend", "create a hub instance and call Send()")
	if legacyGlobalHub == nil {
		return nil, fmt.Errorf("no global hub set. Call SetGlobalHub() first or create a hub instance")
	}
	return legacyGlobalHub.Send(ctx, msg)
}

// SendBatch sends multiple messages using the global hub (deprecated)
func LegacySendBatch(ctx context.Context, msgs []LegacyMessage) ([]*LegacyReceipt, error) {
	logDeprecationWarning("LegacySendBatch", "create a hub instance and call SendBatch()")
	if legacyGlobalHub == nil {
		return nil, fmt.Errorf("no global hub set. Call SetGlobalHub() first or create a hub instance")
	}
	return legacyGlobalHub.SendBatch(ctx, msgs)
}
