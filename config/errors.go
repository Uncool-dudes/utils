package config

import "github.com/uncool-dudes/utils/errors"

var Domain = errors.NewDomain("config")

var (
	ErrNotFound  = Domain.New("file not found")
	ErrMalformed = Domain.New("malformed")
)
