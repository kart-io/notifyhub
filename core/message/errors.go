package message

import "errors"

var (
	// ErrEmptyMessage indicates that the message has no title or body
	ErrEmptyMessage = errors.New("message must have either title or body")

	// ErrInvalidPriority indicates an invalid priority value
	ErrInvalidPriority = errors.New("priority must be between 1 and 5")

	// ErrInvalidFormat indicates an invalid message format
	ErrInvalidFormat = errors.New("invalid message format")

	// ErrMissingTemplate indicates that a template name is required
	ErrMissingTemplate = errors.New("template name is required")

	// ErrTemplateRenderFailed indicates template rendering failure
	ErrTemplateRenderFailed = errors.New("template rendering failed")

	// Target-related errors
	// ErrInvalidTargetType indicates an invalid target type
	ErrInvalidTargetType = errors.New("invalid target type")

	// ErrEmptyTargetValue indicates an empty target value
	ErrEmptyTargetValue = errors.New("target value cannot be empty")

	// ErrEmptyPlatform indicates an empty platform
	ErrEmptyPlatform = errors.New("platform cannot be empty")
)
