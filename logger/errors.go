package logger

import "github.com/ratchio/utils/errors"

//nolint:gochecknoglobals
var Domain = errors.NewDomain("logger")

//nolint:gochecknoglobals,revive
var (
	ErrInvalidLevel    = Domain.New("invalid log level")
	ErrInvalidEncoding = Domain.New("invalid encoding")
)
