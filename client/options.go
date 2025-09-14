package client

import (
	"time"

	"github.com/kart-io/notifyhub/queue"
)

// Options provides options for sending messages
type Options struct {
	// Async enables asynchronous sending
	Async bool `json:"async,omitempty"`
	// Retry enables retry on failure
	Retry bool `json:"retry,omitempty"`
	// MaxRetries overrides default max retries
	MaxRetries int `json:"max_retries,omitempty"`
	// Timeout for sending operation
	Timeout time.Duration `json:"timeout,omitempty"`
	// Callbacks for message lifecycle events
	Callbacks *queue.CallbackOptions `json:"callbacks,omitempty"`
}

// WithAsync enables asynchronous sending
func (o *Options) WithAsync(async bool) *Options {
	if o == nil {
		o = &Options{}
	}
	o.Async = async
	return o
}

// WithRetry enables retry on failure
func (o *Options) WithRetry(retry bool) *Options {
	if o == nil {
		o = &Options{}
	}
	o.Retry = retry
	return o
}

// WithMaxRetries sets maximum retry attempts
func (o *Options) WithMaxRetries(maxRetries int) *Options {
	if o == nil {
		o = &Options{}
	}
	o.MaxRetries = maxRetries
	return o
}

// WithTimeout sets operation timeout
func (o *Options) WithTimeout(timeout time.Duration) *Options {
	if o == nil {
		o = &Options{}
	}
	o.Timeout = timeout
	return o
}

// WithCallbacks sets callback options
func (o *Options) WithCallbacks(callbacks *queue.CallbackOptions) *Options {
	if o == nil {
		o = &Options{}
	}
	o.Callbacks = callbacks
	return o
}

// NewOptions creates new send options
func NewOptions() *Options {
	return &Options{}
}

// NewAsyncOptions creates options for async sending
func NewAsyncOptions() *Options {
	return &Options{
		Async: true,
		Retry: true,
	}
}

// NewSyncOptions creates options for sync sending
func NewSyncOptions() *Options {
	return &Options{
		Async:   false,
		Timeout: 30 * time.Second,
	}
}

// NewRetryOptions creates options with retry enabled
func NewRetryOptions(maxRetries int) *Options {
	return &Options{
		Retry:      true,
		MaxRetries: maxRetries,
		Timeout:    30 * time.Second,
	}
}