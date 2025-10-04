// Package errors provides error codes for NotifyHub
package errors

// ErrorCode represents a NotifyHub error code
type ErrorCode string

// Configuration Error Codes
const (
	// ErrInvalidConfig indicates invalid configuration
	ErrInvalidConfig ErrorCode = "INVALID_CONFIG"

	// ErrMissingPlatform indicates a required platform is missing
	ErrMissingPlatform ErrorCode = "MISSING_PLATFORM"

	// ErrInvalidPlatformConfig indicates invalid platform configuration
	ErrInvalidPlatformConfig ErrorCode = "INVALID_PLATFORM_CONFIG"

	// ErrMissingCredentials indicates missing authentication credentials
	ErrMissingCredentials ErrorCode = "MISSING_CREDENTIALS"

	// ErrInvalidCredentials indicates invalid authentication credentials
	ErrInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
)

// Message Error Codes
const (
	// ErrInvalidMessage indicates an invalid message
	ErrInvalidMessage ErrorCode = "INVALID_MESSAGE"

	// ErrMessageTooLarge indicates message exceeds size limits
	ErrMessageTooLarge ErrorCode = "MESSAGE_TOO_LARGE"

	// ErrEmptyMessage indicates an empty message
	ErrEmptyMessage ErrorCode = "EMPTY_MESSAGE"

	// ErrInvalidFormat indicates an invalid message format
	ErrInvalidFormat ErrorCode = "INVALID_FORMAT"

	// ErrMissingContent indicates missing message content
	ErrMissingContent ErrorCode = "MISSING_CONTENT"

	// ErrInvalidTemplate indicates an invalid template
	ErrInvalidTemplate ErrorCode = "INVALID_TEMPLATE"

	// ErrTemplateNotFound indicates a template was not found
	ErrTemplateNotFound ErrorCode = "TEMPLATE_NOT_FOUND"

	// ErrTemplateRenderFailed indicates template rendering failed
	ErrTemplateRenderFailed ErrorCode = "TEMPLATE_RENDER_FAILED"
)

// Target Error Codes
const (
	// ErrInvalidTarget indicates an invalid target
	ErrInvalidTarget ErrorCode = "INVALID_TARGET"

	// ErrEmptyTarget indicates an empty target
	ErrEmptyTarget ErrorCode = "EMPTY_TARGET"

	// ErrUnsupportedTarget indicates an unsupported target type
	ErrUnsupportedTarget ErrorCode = "UNSUPPORTED_TARGET"

	// ErrTargetResolutionFailed indicates target resolution failed
	ErrTargetResolutionFailed ErrorCode = "TARGET_RESOLUTION_FAILED"

	// ErrNoTargets indicates no targets were provided
	ErrNoTargets ErrorCode = "NO_TARGETS"

	// ErrTooManyTargets indicates too many targets were provided
	ErrTooManyTargets ErrorCode = "TOO_MANY_TARGETS"

	// ErrEmptyTargetType indicates target type is empty
	ErrEmptyTargetType ErrorCode = "EMPTY_TARGET_TYPE"

	// ErrEmptyTargetValue indicates target value is empty
	ErrEmptyTargetValue ErrorCode = "EMPTY_TARGET_VALUE"

	// ErrEmptyPlatform indicates platform is empty
	ErrEmptyPlatform ErrorCode = "EMPTY_PLATFORM"
)

// Platform Error Codes
const (
	// ErrPlatformUnavailable indicates a platform is unavailable
	ErrPlatformUnavailable ErrorCode = "PLATFORM_UNAVAILABLE"

	// ErrPlatformNotFound indicates a platform was not found
	ErrPlatformNotFound ErrorCode = "PLATFORM_NOT_FOUND"

	// ErrPlatformNotSupported indicates a platform is not supported
	ErrPlatformNotSupported ErrorCode = "PLATFORM_NOT_SUPPORTED"

	// ErrPlatformTimeout indicates a platform operation timed out
	ErrPlatformTimeout ErrorCode = "PLATFORM_TIMEOUT"

	// ErrPlatformError indicates a general platform error
	ErrPlatformError ErrorCode = "PLATFORM_ERROR"

	// ErrPlatformAuthFailed indicates platform authentication failed
	ErrPlatformAuthFailed ErrorCode = "PLATFORM_AUTH_FAILED"

	// ErrPlatformRejected indicates the platform rejected the request
	ErrPlatformRejected ErrorCode = "PLATFORM_REJECTED"
)

// Network Error Codes
const (
	// ErrNetworkTimeout indicates a network timeout
	ErrNetworkTimeout ErrorCode = "NETWORK_TIMEOUT"

	// ErrConnectionFailed indicates connection failure
	ErrConnectionFailed ErrorCode = "CONNECTION_FAILED"

	// ErrDNSResolutionFailed indicates DNS resolution failed
	ErrDNSResolutionFailed ErrorCode = "DNS_RESOLUTION_FAILED"

	// ErrSSLError indicates an SSL/TLS error
	ErrSSLError ErrorCode = "SSL_ERROR"

	// ErrProxyError indicates a proxy error
	ErrProxyError ErrorCode = "PROXY_ERROR"
)

// Rate Limiting Error Codes
const (
	// ErrRateLimitExceeded indicates rate limit was exceeded
	ErrRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"

	// ErrQuotaExceeded indicates quota was exceeded
	ErrQuotaExceeded ErrorCode = "QUOTA_EXCEEDED"

	// ErrThrottled indicates the request was throttled
	ErrThrottled ErrorCode = "THROTTLED"

	// ErrConcurrencyLimitExceeded indicates concurrency limit exceeded
	ErrConcurrencyLimitExceeded ErrorCode = "CONCURRENCY_LIMIT_EXCEEDED"
)

// Processing Error Codes
const (
	// ErrProcessingFailed indicates processing failed
	ErrProcessingFailed ErrorCode = "PROCESSING_FAILED"

	// ErrValidationFailed indicates validation failed
	ErrValidationFailed ErrorCode = "VALIDATION_FAILED"

	// ErrSerializationFailed indicates serialization failed
	ErrSerializationFailed ErrorCode = "SERIALIZATION_FAILED"

	// ErrDeserializationFailed indicates deserialization failed
	ErrDeserializationFailed ErrorCode = "DESERIALIZATION_FAILED"

	// ErrTransformationFailed indicates data transformation failed
	ErrTransformationFailed ErrorCode = "TRANSFORMATION_FAILED"
)

// System Error Codes
const (
	// ErrInternal indicates an internal system error
	ErrInternal ErrorCode = "INTERNAL_ERROR"

	// ErrNotImplemented indicates functionality is not implemented
	ErrNotImplemented ErrorCode = "NOT_IMPLEMENTED"

	// ErrUnavailable indicates service is unavailable
	ErrUnavailable ErrorCode = "UNAVAILABLE"

	// ErrResourceExhausted indicates resources are exhausted
	ErrResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"

	// ErrDeadlineExceeded indicates deadline was exceeded
	ErrDeadlineExceeded ErrorCode = "DEADLINE_EXCEEDED"

	// ErrCancelled indicates operation was cancelled
	ErrCancelled ErrorCode = "CANCELLED"

	// ErrNotFound indicates resource was not found
	ErrNotFound ErrorCode = "NOT_FOUND"

	// ErrAlreadyExists indicates resource already exists
	ErrAlreadyExists ErrorCode = "ALREADY_EXISTS"

	// ErrPermissionDenied indicates permission was denied
	ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"

	// ErrUnauthenticated indicates authentication is required
	ErrUnauthenticated ErrorCode = "UNAUTHENTICATED"
)

// Async Error Codes
const (
	// ErrAsyncOperationFailed indicates async operation failed
	ErrAsyncOperationFailed ErrorCode = "ASYNC_OPERATION_FAILED"

	// ErrHandleNotFound indicates async handle was not found
	ErrHandleNotFound ErrorCode = "HANDLE_NOT_FOUND"

	// ErrOperationCancelled indicates operation was cancelled
	ErrOperationCancelled ErrorCode = "OPERATION_CANCELLED"

	// ErrQueueFull indicates the queue is full
	ErrQueueFull ErrorCode = "QUEUE_FULL"

	// ErrWorkerUnavailable indicates no workers are available
	ErrWorkerUnavailable ErrorCode = "WORKER_UNAVAILABLE"
)

// Priority levels for error codes
const (
	PriorityLow      = 1
	PriorityNormal   = 2
	PriorityHigh     = 3
	PriorityCritical = 4
)

// ErrorCodeInfo provides information about an error code
type ErrorCodeInfo struct {
	Code        ErrorCode `json:"code"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Priority    int       `json:"priority"`
	Retryable   bool      `json:"retryable"`
	UserFacing  bool      `json:"user_facing"`
}

// GetErrorCodeInfo returns information about an error code
func GetErrorCodeInfo(code ErrorCode) ErrorCodeInfo {
	info, exists := errorCodeInfoMap[code]
	if !exists {
		return ErrorCodeInfo{
			Code:        code,
			Category:    "unknown",
			Description: "Unknown error code",
			Priority:    PriorityNormal,
			Retryable:   false,
			UserFacing:  false,
		}
	}
	return info
}

// IsRetryable checks if an error code is retryable
func IsRetryable(code ErrorCode) bool {
	info := GetErrorCodeInfo(code)
	return info.Retryable
}

// GetCategory returns the category of an error code
func GetCategory(code ErrorCode) string {
	info := GetErrorCodeInfo(code)
	return info.Category
}

// GetPriority returns the priority of an error code
func GetPriority(code ErrorCode) int {
	info := GetErrorCodeInfo(code)
	return info.Priority
}

// Error code information mapping
var errorCodeInfoMap = map[ErrorCode]ErrorCodeInfo{
	// Configuration errors
	ErrInvalidConfig: {
		Code: ErrInvalidConfig, Category: "configuration", Description: "Invalid configuration provided",
		Priority: PriorityHigh, Retryable: false, UserFacing: true,
	},
	ErrMissingPlatform: {
		Code: ErrMissingPlatform, Category: "configuration", Description: "Required platform configuration is missing",
		Priority: PriorityHigh, Retryable: false, UserFacing: true,
	},

	// Message errors
	ErrInvalidMessage: {
		Code: ErrInvalidMessage, Category: "message", Description: "Invalid message format or content",
		Priority: PriorityNormal, Retryable: false, UserFacing: true,
	},
	ErrMessageTooLarge: {
		Code: ErrMessageTooLarge, Category: "message", Description: "Message exceeds maximum size limit",
		Priority: PriorityNormal, Retryable: false, UserFacing: true,
	},

	// Target errors
	ErrInvalidTarget: {
		Code: ErrInvalidTarget, Category: "target", Description: "Invalid target specification",
		Priority: PriorityNormal, Retryable: false, UserFacing: true,
	},
	ErrNoTargets: {
		Code: ErrNoTargets, Category: "target", Description: "No targets specified for message",
		Priority: PriorityNormal, Retryable: false, UserFacing: true,
	},

	// Platform errors
	ErrPlatformUnavailable: {
		Code: ErrPlatformUnavailable, Category: "platform", Description: "Platform is temporarily unavailable",
		Priority: PriorityHigh, Retryable: true, UserFacing: true,
	},
	ErrPlatformTimeout: {
		Code: ErrPlatformTimeout, Category: "platform", Description: "Platform operation timed out",
		Priority: PriorityNormal, Retryable: true, UserFacing: true,
	},

	// Network errors
	ErrNetworkTimeout: {
		Code: ErrNetworkTimeout, Category: "network", Description: "Network operation timed out",
		Priority: PriorityNormal, Retryable: true, UserFacing: false,
	},
	ErrConnectionFailed: {
		Code: ErrConnectionFailed, Category: "network", Description: "Failed to establish connection",
		Priority: PriorityNormal, Retryable: true, UserFacing: false,
	},

	// Rate limiting errors
	ErrRateLimitExceeded: {
		Code: ErrRateLimitExceeded, Category: "rate_limit", Description: "Rate limit exceeded",
		Priority: PriorityNormal, Retryable: true, UserFacing: true,
	},
	ErrThrottled: {
		Code: ErrThrottled, Category: "rate_limit", Description: "Request was throttled",
		Priority: PriorityLow, Retryable: true, UserFacing: true,
	},

	// System errors
	ErrInternal: {
		Code: ErrInternal, Category: "system", Description: "Internal system error",
		Priority: PriorityCritical, Retryable: true, UserFacing: false,
	},
	ErrUnavailable: {
		Code: ErrUnavailable, Category: "system", Description: "Service is temporarily unavailable",
		Priority: PriorityHigh, Retryable: true, UserFacing: true,
	},
}

// GetAllErrorCodes returns all defined error codes
func GetAllErrorCodes() []ErrorCode {
	codes := make([]ErrorCode, 0, len(errorCodeInfoMap))
	for code := range errorCodeInfoMap {
		codes = append(codes, code)
	}
	return codes
}

// GetErrorCodesByCategory returns error codes for a specific category
func GetErrorCodesByCategory(category string) []ErrorCode {
	var codes []ErrorCode
	for code, info := range errorCodeInfoMap {
		if info.Category == category {
			codes = append(codes, code)
		}
	}
	return codes
}
