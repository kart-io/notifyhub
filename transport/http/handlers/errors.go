package handlers

import "errors"

// Common HTTP handler errors
var (
	ErrEmptyMessage  = errors.New("message title and body cannot both be empty")
	ErrNoTargets     = errors.New("at least one target must be specified")
	ErrInvalidFormat = errors.New("invalid message format")
	ErrInvalidTarget = errors.New("invalid target specification")
	ErrUnauthorized  = errors.New("unauthorized access")
	ErrRateLimited   = errors.New("rate limit exceeded")
)
