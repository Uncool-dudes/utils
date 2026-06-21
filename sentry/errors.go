package sentry

// Sentinel errors returned by the sentry package.
var (
	ErrInvalidConfig = Domain.New("invalid sentry config")
	ErrInit          = Domain.New("failed to initialise sentry")
)
