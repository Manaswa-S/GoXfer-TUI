package errs

type Errorf struct {
	Type      string
	Message   string
	Error     error
	ReturnRaw bool
}

// Generic Errors
const (
	ErrInternalServer  = "INTERNAL_SERVER_ERROR"
	ErrBadRequest      = "BAD_REQUEST"
	ErrUnauthorized    = "UNAUTHORIZED"
	ErrForbidden       = "FORBIDDEN"
	ErrNotFound        = "NOT_FOUND"
	ErrConflict        = "CONFLICT"
	ErrTooManyRequests = "TOO_MANY_REQUESTS"
	ErrEnvNotFound     = "ENV_NOT_FOUND"
)

// Validation & Input Errors
const (
	ErrInvalidInput  = "INVALID_INPUT"
	ErrMissingField  = "MISSING_FIELD"
	ErrBadForm       = "BAD_FORM"
	ErrInvalidFormat = "INVALID_FORMAT"
	ErrOutOfRange    = "OUT_OF_RANGE"
)

// Authentication & Authorization Errors
const (
	ErrInvalidCredentials = "INVALID_CREDENTIALS"
	ErrTokenExpired       = "TOKEN_EXPIRED"
	ErrTokenInvalid       = "TOKEN_INVALID"
	ErrPermissionDenied   = "PERMISSION_DENIED"
)

// Networking & API Errors
const (
	ErrTimeout            = "REQUEST_TIMEOUT"
	ErrServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrGatewayTimeout     = "GATEWAY_TIMEOUT"
	ErrRateLimited        = "RATE_LIMIT_EXCEEDED"
)

// File & Storage Errors
const (
	ErrFileNotFound        = "FILE_NOT_FOUND"
	ErrFileTooLarge        = "FILE_TOO_LARGE"
	ErrStorageFailed       = "STORAGE_OPERATION_FAILED"
	ErrInsufficientStorage = "INSUFFICIENT_STORAGE"
)

// Custom Business Logic Errors
const (
	ErrActionNotAllowed = "ACTION_NOT_ALLOWED"
	ErrResourceLocked   = "RESOURCE_LOCKED"
	ErrDependencyFailed = "DEPENDENCY_FAILED"
	ErrStateConflict    = "STATE_CONFLICT"
)

// HTTP Errors
const (
	ErrHTTPConnectionFailed = "HTTP_CONNECTION_FAILED"
	ErrHTTPTimeout          = "HTTP_TIMEOUT"          // Request timed out
	ErrHTTPHostUnreachable  = "HTTP_HOST_UNREACHABLE" // No route to host
	ErrHTTPNetworkReset     = "HTTP_NETWORK_RESET"    // Connection dropped mid-way
	ErrHTTPProxyError       = "HTTP_PROXY_ERROR"      // Error in proxy configuration
)
