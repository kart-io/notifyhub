package sending

import "errors"

var (
	// ErrInvalidTargetType indicates an invalid target type
	ErrInvalidTargetType = errors.New("invalid target type")

	// ErrEmptyTargetValue indicates an empty target value
	ErrEmptyTargetValue = errors.New("target value cannot be empty")

	// ErrEmptyPlatform indicates an empty platform
	ErrEmptyPlatform = errors.New("platform cannot be empty")

	// ErrUnsupportedPlatform indicates an unsupported platform
	ErrUnsupportedPlatform = errors.New("unsupported platform")

	// ErrSendingFailed indicates a general sending failure
	ErrSendingFailed = errors.New("message sending failed")

	// ErrTimeout indicates a timeout during sending
	ErrTimeout = errors.New("sending timeout")

	// ErrRateLimited indicates rate limiting
	ErrRateLimited = errors.New("rate limited")

	// ErrInvalidCredentials indicates invalid credentials
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrNetworkError indicates a network error
	ErrNetworkError = errors.New("network error")
)
