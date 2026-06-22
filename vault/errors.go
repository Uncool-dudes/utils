package vault

import "github.com/uncool-dudes/utils/errors"

// Domain tags all errors from this package.
var Domain = errors.NewDomain("vault")

// Sentinel errors returned by the vault package.
var (
	ErrConnFailed     = Domain.New("connection failed")
	ErrSecretNotFound = Domain.New("secret not found")
	ErrAccessDenied   = Domain.New("access denied")
)
