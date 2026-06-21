package httpserver

// Sentinel errors returned by the httpserver package.
var (
	ErrStartFailed = Domain.New("failed to start server")
	ErrShutdown    = Domain.New("server shutdown failed")
)
