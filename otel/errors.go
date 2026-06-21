package otel

// Sentinel errors returned by the otel package.
var (
	ErrInvalidConfig = Domain.New("invalid otel config")
	ErrInit          = Domain.New("failed to initialise otel")
	ErrShutdown      = Domain.New("failed to shutdown otel")
)
