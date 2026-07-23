package gobind

import "net/http"

// BadRequestError creates a 400 Bad Request error with the given error body.
func BadRequestError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusBadRequest,
		Value:      v,
	}
}

// UnauthorizedError creates a 401 Unauthorized error with the given error body.
func UnauthorizedError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusUnauthorized,
		Value:      v,
	}
}

// ForbiddenError creates a 403 Forbidden error with the given error body.
func ForbiddenError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusForbidden,
		Value:      v,
	}
}

// NotFoundError creates a 404 Not Found error with the given error body.
func NotFoundError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusNotFound,
		Value:      v,
	}
}

// MethodNotAllowedError creates a 405 Method Not Allowed error with the given error body.
func MethodNotAllowedError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusMethodNotAllowed,
		Value:      v,
	}
}

// ConflictError creates a 409 Conflict error with the given error body.
func ConflictError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusConflict,
		Value:      v,
	}
}

// UnprocessableEntityError creates a 422 Unprocessable Entity error with the given error body.
func UnprocessableEntityError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusUnprocessableEntity,
		Value:      v,
	}
}

// TooManyRequestsError creates a 429 Too Many Requests error with the given error body.
func TooManyRequestsError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusTooManyRequests,
		Value:      v,
	}
}

// InternalServerError creates a 500 Internal Server Error error with the given error body.
func InternalServerError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusInternalServerError,
		Value:      v,
	}
}

// BadGatewayError creates a 502 Bad Gateway error with the given error body.
func BadGatewayError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusBadGateway,
		Value:      v,
	}
}

// ServiceUnavailableError creates a 503 Service Unavailable error with the given error body.
func ServiceUnavailableError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusServiceUnavailable,
		Value:      v,
	}
}

// GatewayTimeoutError creates a 504 Gateway Timeout error with the given error body.
func GatewayTimeoutError[T any](v T) *Error[T] {
	return &Error[T]{
		StatusCode: http.StatusGatewayTimeout,
		Value:      v,
	}
}
