package pii

import (
	"github.com/cockroachdb/redact"
	"go.uber.org/zap/zapcore"
)

type (
	FirstName string
	LastName  string
	FullName  string
)

func (n FirstName) String() string { return string(n) }
func (n FirstName) Masked() string { return maskName(string(n)) }
func (n LastName) String() string  { return string(n) }
func (n LastName) Masked() string  { return maskName(string(n)) }
func (n FullName) String() string  { return string(n) }
func (n FullName) Masked() string  { return maskName(string(n)) }

func (n FirstName) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(n).Redact()))
	return nil
}

func (n LastName) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(n).Redact()))
	return nil
}

func (n FullName) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(n).Redact()))
	return nil
}
