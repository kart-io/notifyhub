package message

import (
	"github.com/kart-io/notifyhub/core/errors"
)

// Re-export standard errors for backward compatibility
var (
	// ErrEmptyMessage indicates that the message has no title or body
	ErrEmptyMessage = errors.ErrEmptyMessage

	// ErrInvalidPriority indicates an invalid priority value
	ErrInvalidPriority = errors.ErrInvalidPriority

	// ErrInvalidFormat indicates an invalid message format
	ErrInvalidFormat = errors.ErrInvalidFormat

	// ErrMissingTemplate indicates that a template name is required
	ErrMissingTemplate = errors.ErrTemplateError

	// ErrTemplateRenderFailed indicates template rendering failure
	ErrTemplateRenderFailed = errors.ErrTemplateError

	// Target-related errors
	// ErrInvalidTargetType indicates an invalid target type
	ErrInvalidTargetType = errors.ErrInvalidTarget

	// ErrEmptyTargetValue indicates an empty target value
	ErrEmptyTargetValue = errors.ErrEmptyTarget

	// ErrEmptyPlatform indicates an empty platform
	ErrEmptyPlatform = errors.ErrInvalidPlatform
)
