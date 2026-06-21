package config

import "github.com/uncool-dudes/utils/errors"

// Domain tags all errors from this package.
var Domain = errors.NewDomain("config")

// Sentinel errors returned by config providers.
var (
	ErrNotFound  = Domain.New("file not found")
	ErrMalformed = Domain.New("malformed")
)
