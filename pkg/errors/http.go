package errors

import "net/http"

// HTTPError represents an HTTP error with status code and message.
type HTTPError struct {
	Code       int
	Message    string
	StatusCode int
}

// NewHTTPError returns a new HTTPError with the given code, message, and status code.
// If statusCode is 0, it defaults to http.StatusBadRequest.
func NewHTTPError(code int, message string, statusCode int) *HTTPError {
	if statusCode == 0 {
		statusCode = http.StatusBadRequest
	}
	return &HTTPError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// NewUnauthorizedHTTPError returns a new unauthorized HTTP error.
func NewUnauthorizedHTTPError() *HTTPError {
	return &HTTPError{
		Code:       401,
		Message:    "Unauthorized",
		StatusCode: http.StatusUnauthorized,
	}
}

// NewForbiddenHTTPError returns a new forbidden HTTP error.
func NewForbiddenHTTPError() *HTTPError {
	return &HTTPError{
		Code:       403,
		Message:    "Forbidden",
		StatusCode: http.StatusForbidden,
	}
}

// Error returns the error message.
func (e *HTTPError) Error() string {
	return e.Message
}
