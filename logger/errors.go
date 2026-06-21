package logger

import "github.com/uncool-dudes/utils/errors"

// Domain tags all errors from this package.
//
//nolint:gochecknoglobals // package-level domain is intentional
var Domain = errors.NewDomain("logger")

// Sentinel errors returned by the logger package.
//
//nolint:gochecknoglobals // package-level sentinels are intentional
var (
	ErrInvalidLevel    = Domain.New("invalid log level")
	ErrInvalidEncoding = Domain.New("invalid encoding")
)
