package apperror

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidTransition = errors.New("invalid payment status transition")
)
