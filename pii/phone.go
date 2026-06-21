package pii

import (
	"github.com/cockroachdb/redact"
	"go.uber.org/zap/zapcore"
)

// Phone is an E.164 phone number. Validate with `validate:"e164"`.
type Phone string

func (p Phone) String() string { return string(p) }
func (p Phone) Masked() string { return maskPhone(string(p)) }

func (p Phone) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(p).Redact()))
	return nil
}
