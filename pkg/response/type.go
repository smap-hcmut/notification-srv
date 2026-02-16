package response

import "smap-websocket/pkg/errors"

// Resp is the response format.
type Resp struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Errors    any    `json:"errors,omitempty"`
}

// ErrorMapping is a map of error to HTTPError.
type ErrorMapping map[error]*errors.HTTPError
