package sending

import (
	"github.com/kart-io/notifyhub/core/errors"
)

// Re-export standard errors for backward compatibility
var (
	// ErrInvalidTargetType indicates an invalid target type
	ErrInvalidTargetType = errors.ErrInvalidTarget

	// ErrEmptyTargetValue indicates an empty target value
	ErrEmptyTargetValue = errors.ErrEmptyTarget

	// ErrEmptyPlatform indicates an empty platform
	ErrEmptyPlatform = errors.ErrInvalidPlatform

	// ErrUnsupportedPlatform indicates an unsupported platform
	ErrUnsupportedPlatform = errors.ErrInvalidPlatform

	// ErrSendingFailed indicates a general sending failure
	ErrSendingFailed = errors.ErrSendingFailed

	// ErrTimeout indicates a timeout during sending
	ErrTimeout = errors.ErrTimeout

	// ErrRateLimited indicates rate limiting
	ErrRateLimited = errors.ErrRateLimited

	// ErrInvalidCredentials indicates invalid credentials
	ErrInvalidCredentials = errors.ErrInvalidCredentials

	// ErrNetworkError indicates a network error
	ErrNetworkError = errors.ErrNetworkError
)
