package pii

import (
	"github.com/cockroachdb/redact"
	"go.uber.org/zap/zapcore"
)

// Email is a PII-redacting string type for email addresses.
type Email string

func (e Email) String() string { return string(e) }

// Masked returns the email with the local part redacted.
func (e Email) Masked() string { return maskEmail(string(e)) }

// MarshalLogObject implements zapcore.ObjectMarshaler, logging the masked value.
func (e Email) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(e).Redact()))
	return nil
}
