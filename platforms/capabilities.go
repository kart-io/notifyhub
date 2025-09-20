package platforms

import (
	"time"
)

// Format represents a message format type
type Format string

const (
	FormatText     Format = "text"
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
	FormatCard     Format = "card"
)

// Capabilities defines what a platform can do
type Capabilities interface {
	// Message format support
	SupportedFormats() []Format
	SupportsFormat(format Format) bool

	// Message size limits
	MaxMessageSize() int
	MaxTitleLength() int
	MaxBodyLength() int

	// Rate limiting information
	RateLimits() RateLimitInfo

	// Feature support
	Features() []Feature
	SupportsFeature(feature Feature) bool

	// Target types supported
	SupportedTargetTypes() []string
	SupportsTargetType(targetType string) bool
}

// RateLimitInfo contains rate limiting information
type RateLimitInfo struct {
	// Maximum requests per second
	RequestsPerSecond int

	// Burst capacity
	BurstSize int

	// Time window for rate limiting
	Window time.Duration

	// Whether rate limiting is enforced
	Enforced bool
}

// Feature represents a platform feature
type Feature string

const (
	// Core features
	FeatureText     Feature = "text"
	FeatureMarkdown Feature = "markdown"
	FeatureHTML     Feature = "html"
	FeatureCard     Feature = "card"
	FeatureTemplate Feature = "template"

	// Advanced features
	FeatureAttachments Feature = "attachments"
	FeatureMentions    Feature = "mentions"
	FeatureReactions   Feature = "reactions"
	FeatureThreading   Feature = "threading"
	FeatureScheduling  Feature = "scheduling"
	FeaturePriority    Feature = "priority"
	FeatureBatch       Feature = "batch"

	// Delivery features
	FeatureDeliveryReceipt Feature = "delivery_receipt"
	FeatureReadReceipt     Feature = "read_receipt"
	FeatureRetry           Feature = "retry"
)

// BaseCapabilities provides a standard implementation of Capabilities
type BaseCapabilities struct {
	formats        []Format
	features       []Feature
	targetTypes    []string
	maxMessageSize int
	maxTitleLength int
	maxBodyLength  int
	rateLimits     RateLimitInfo
}

// NewBaseCapabilities creates a new BaseCapabilities instance
func NewBaseCapabilities() *BaseCapabilities {
	return &BaseCapabilities{
		formats:        []Format{FormatText},
		features:       []Feature{FeatureText},
		targetTypes:    []string{"user", "group"},
		maxMessageSize: 1024 * 1024, // 1MB default
		maxTitleLength: 256,
		maxBodyLength:  4096,
		rateLimits: RateLimitInfo{
			RequestsPerSecond: 10,
			BurstSize:         20,
			Window:            time.Second,
			Enforced:          false,
		},
	}
}

// Builder pattern for BaseCapabilities
func (c *BaseCapabilities) WithFormats(formats ...Format) *BaseCapabilities {
	c.formats = formats
	return c
}

func (c *BaseCapabilities) WithFeatures(features ...Feature) *BaseCapabilities {
	c.features = features
	return c
}

func (c *BaseCapabilities) WithTargetTypes(types ...string) *BaseCapabilities {
	c.targetTypes = types
	return c
}

func (c *BaseCapabilities) WithMaxMessageSize(size int) *BaseCapabilities {
	c.maxMessageSize = size
	return c
}

func (c *BaseCapabilities) WithMaxTitleLength(length int) *BaseCapabilities {
	c.maxTitleLength = length
	return c
}

func (c *BaseCapabilities) WithMaxBodyLength(length int) *BaseCapabilities {
	c.maxBodyLength = length
	return c
}

func (c *BaseCapabilities) WithRateLimits(limits RateLimitInfo) *BaseCapabilities {
	c.rateLimits = limits
	return c
}

// Interface implementation
func (c *BaseCapabilities) SupportedFormats() []Format {
	return c.formats
}

func (c *BaseCapabilities) SupportsFormat(format Format) bool {
	for _, f := range c.formats {
		if f == format {
			return true
		}
	}
	return false
}

func (c *BaseCapabilities) MaxMessageSize() int {
	return c.maxMessageSize
}

func (c *BaseCapabilities) MaxTitleLength() int {
	return c.maxTitleLength
}

func (c *BaseCapabilities) MaxBodyLength() int {
	return c.maxBodyLength
}

func (c *BaseCapabilities) RateLimits() RateLimitInfo {
	return c.rateLimits
}

func (c *BaseCapabilities) Features() []Feature {
	return c.features
}

func (c *BaseCapabilities) SupportsFeature(feature Feature) bool {
	for _, f := range c.features {
		if f == feature {
			return true
		}
	}
	return false
}

func (c *BaseCapabilities) SupportedTargetTypes() []string {
	return c.targetTypes
}

func (c *BaseCapabilities) SupportsTargetType(targetType string) bool {
	for _, t := range c.targetTypes {
		if t == targetType {
			return true
		}
	}
	return false
}
