package pii

import (
	"github.com/cockroachdb/redact"
	"go.uber.org/zap/zapcore"
)

// FirstName is a PII-redacting string type for a person's first name.
type FirstName string

// LastName is a PII-redacting string type for a person's last name.
type LastName string

// FullName is a PII-redacting string type for a person's full name.
type FullName string

func (n FirstName) String() string { return string(n) }

// Masked returns the first name with characters redacted.
func (n FirstName) Masked() string { return maskName(string(n)) }

func (n LastName) String() string { return string(n) }

// Masked returns the last name with characters redacted.
func (n LastName) Masked() string { return maskName(string(n)) }

func (n FullName) String() string { return string(n) }

// Masked returns the full name with characters redacted.
func (n FullName) Masked() string { return maskName(string(n)) }

// MarshalLogObject implements zapcore.ObjectMarshaler, logging the masked value.
func (n FirstName) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(n).Redact()))
	return nil
}

// MarshalLogObject implements zapcore.ObjectMarshaler, logging the masked value.
func (n LastName) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(n).Redact()))
	return nil
}

// MarshalLogObject implements zapcore.ObjectMarshaler, logging the masked value.
func (n FullName) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(n).Redact()))
	return nil
}
