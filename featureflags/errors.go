package featureflags

import "github.com/uncool-dudes/utils/errors"

// Domain tags all errors from this package.
var Domain = errors.NewDomain("featureflags")

// Sentinel errors returned by the featureflags package.
var (
	ErrProviderFailed = Domain.New("provider initialization failed")
)
