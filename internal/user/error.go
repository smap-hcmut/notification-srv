package user

import "errors"

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUserExists    = errors.New("user already exists")
	ErrInvalidRole   = errors.New("invalid role")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrFieldRequired = errors.New("field required")
)
