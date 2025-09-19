package sms

import "errors"

// SMS-specific errors
var (
	ErrInvalidConfig   = errors.New("invalid SMS config")
	ErrEmptyMessage    = errors.New("message cannot be empty")
	ErrInvalidNumber   = errors.New("invalid phone number")
	ErrMessageTooLong  = errors.New("message too long for SMS")
	ErrInvalidProvider = errors.New("invalid SMS provider")
)

// SMSPriority represents SMS priority levels
type SMSPriority string

const (
	PriorityLow    SMSPriority = "low"
	PriorityNormal SMSPriority = "normal"
	PriorityHigh   SMSPriority = "high"
	PriorityUrgent SMSPriority = "urgent"
)

// SMSProvider represents different SMS service providers
type SMSProvider string

const (
	ProviderTwilio  SMSProvider = "twilio"
	ProviderAWSSNS  SMSProvider = "aws_sns"
	ProviderAliyun  SMSProvider = "aliyun"
	ProviderTencent SMSProvider = "tencent"
)

// SMSConfig holds SMS-specific configuration
type SMSConfig struct {
	Provider     SMSProvider `json:"provider"`
	AccountSID   string      `json:"account_sid"`
	AuthToken    string      `json:"auth_token"`
	FromNumber   string      `json:"from_number"`
	APIKey       string      `json:"api_key"`
	APISecret    string      `json:"api_secret"`
	Region       string      `json:"region,omitempty"`
	SignName     string      `json:"sign_name,omitempty"`     // For Chinese providers
	TemplateCode string      `json:"template_code,omitempty"` // For template SMS
}

// SMSTarget represents an SMS-specific target
type SMSTarget struct {
	Type        string `json:"type"`
	PhoneNumber string `json:"phone_number"`
	CountryCode string `json:"country_code,omitempty"`
	Name        string `json:"name,omitempty"`
}

// SMSMessage represents a complete SMS message
type SMSMessage struct {
	To             []string          `json:"to"`
	From           string            `json:"from"`
	Body           string            `json:"body"`
	Priority       SMSPriority       `json:"priority,omitempty"`
	DeliveryReport bool              `json:"delivery_report,omitempty"`
	FlashSMS       bool              `json:"flash_sms,omitempty"`
	ValidityPeriod int               `json:"validity_period,omitempty"` // in hours
	TemplateID     string            `json:"template_id,omitempty"`
	TemplateParams map[string]string `json:"template_params,omitempty"`
}
