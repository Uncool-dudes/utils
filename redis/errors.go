package redis

import "github.com/uncool-dudes/utils/errors"

// Domain tags all errors from this package.
var Domain = errors.NewDomain("redis")

// Sentinel errors returned by the redis package.
var (
	ErrConnFailed = Domain.New("connection failed")
	ErrPingFailed = Domain.New("ping failed")
	ErrNil        = Domain.New("key not found")
)
