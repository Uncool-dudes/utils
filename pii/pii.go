// Package pii defines named types for personally identifiable information.
//
// Types are thin string wrappers — validation is handled by go-playground/validator
// at request boundaries using struct tags:
//
//	type CreateUserRequest struct {
//	    Email pii.Email `validate:"required,email"`
//	    Phone pii.Phone `validate:"required,e164"`
//	}
//
// All types implement zapcore.ObjectMarshaler so they redact automatically when
// logged via zap.Object("field", val). Raw values are never emitted to logs.
package pii

import "github.com/uncool-dudes/utils/errors"

//nolint:gochecknoglobals
var Domain = errors.NewDomain("pii")
