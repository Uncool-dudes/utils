package redis

import "github.com/uncool-dudes/utils/errors"

var Domain = errors.NewDomain("redis")

var (
	ErrConnFailed = Domain.New("connection failed")
	ErrPingFailed = Domain.New("ping failed")
	ErrNil        = Domain.New("key not found")
)
