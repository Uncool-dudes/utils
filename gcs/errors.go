package gcs

import "github.com/uncool-dudes/utils/errors"

// Domain tags all errors from this package.
var Domain = errors.NewDomain("gcs")

// Sentinel errors returned by the gcs package.
var (
	ErrConnFailed   = Domain.New("connection failed")
	ErrUploadFailed = Domain.New("upload failed")
	ErrNotFound     = Domain.New("object not found")
)
