package resilience

import "github.com/uncool-dudes/utils/errors"

// Domain tags all errors from this package.
var Domain = errors.NewDomain("resilience")

// Sentinel errors returned by the resilience package.
var (
	ErrCircuitOpen = Domain.New("circuit open")
	ErrMaxRetries  = Domain.New("max retries exceeded")
	ErrTimeout     = Domain.New("timeout")
)
