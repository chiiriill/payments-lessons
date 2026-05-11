package apperror

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidTransition = errors.New("invalid payment status transition")
	ErrProviderTemporary = errors.New("provider temporary error")
	ErrProviderPermanent = errors.New("provider permanent error")
)
