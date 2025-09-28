// Package receipt provides receipt management for NotifyHub
package receipt

import "errors"

// Error definitions for receipt package
var (
	// ErrReceiptNotFound is returned when a receipt is not found
	ErrReceiptNotFound = errors.New("receipt not found")

	// ErrInvalidFilter is returned when a filter is invalid
	ErrInvalidFilter = errors.New("invalid filter parameters")

	// ErrStoreNotAvailable is returned when persistence store is not available
	ErrStoreNotAvailable = errors.New("persistence store not available")

	// ErrSerializationFailed is returned when serialization fails
	ErrSerializationFailed = errors.New("receipt serialization failed")

	// ErrDeserializationFailed is returned when deserialization fails
	ErrDeserializationFailed = errors.New("receipt deserialization failed")

	// ErrProcessorNotStarted is returned when processor operations are attempted before starting
	ErrProcessorNotStarted = errors.New("receipt processor not started")

	// ErrInvalidAggregationConfig is returned when aggregation config is invalid
	ErrInvalidAggregationConfig = errors.New("invalid aggregation configuration")
)