package sentry

var (
	ErrInvalidConfig = Domain.New("invalid sentry config")
	ErrInit          = Domain.New("failed to initialise sentry")
)
