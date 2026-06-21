package mailer

import "github.com/uncool-dudes/utils/errors"

// Domain tags all errors from this package.
var Domain = errors.NewDomain("mailer")

// Sentinel errors returned by the mailer package.
var (
	ErrConnFailed = Domain.New("connection failed")
	ErrSendFailed = Domain.New("send failed")
)
