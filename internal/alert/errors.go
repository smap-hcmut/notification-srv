package alert

import "errors"

var (
	ErrDispatchFailed = errors.New("failed to dispatch alert")
	ErrInvalidInput   = errors.New("invalid alert input")
)
