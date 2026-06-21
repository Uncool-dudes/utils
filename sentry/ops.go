package sentry

import (
	"context"
	"time"

	sdk "github.com/getsentry/sentry-go"
)

// CaptureException sends err to Sentry as an exception event with stack trace.
func CaptureException(err error) {
	sdk.CaptureException(err)
}

// Recover captures a panic value from recover() as a Sentry exception.
func Recover(rv any) {
	sdk.CurrentHub().Recover(rv)
}

// RecoverWithContext captures a panic with request context so Sentry can attach
// request metadata (path, method, headers) to the exception event.
func RecoverWithContext(ctx context.Context, rv any) {
	sdk.CurrentHub().RecoverWithContext(ctx, rv)
}

// Flush waits up to 2s for buffered events to be sent. Call before os.Exit.
func Flush() {
	sdk.Flush(2 * time.Second)
}
