package httpserver

var (
	ErrStartFailed = Domain.New("failed to start server")
	ErrShutdown    = Domain.New("server shutdown failed")
)
