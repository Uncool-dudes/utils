package db

import stderrors "errors"

// Sentinel errors returned by the db package.
var (
	ErrConnFailed         = Domain.New("failed to connect to database")
	ErrPingFailed         = Domain.New("database ping failed")
	ErrReloadNotSupported = stderrors.New("database reload requires restart — pool is immutable after init")
)
