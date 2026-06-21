package resilience

import "github.com/uncool-dudes/utils/errors"

var Domain = errors.NewDomain("resilience")

var (
	ErrCircuitOpen = Domain.New("circuit open")
	ErrMaxRetries  = Domain.New("max retries exceeded")
	ErrTimeout     = Domain.New("timeout")
)
