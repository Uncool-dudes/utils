package pii

import (
	"github.com/cockroachdb/redact"
	"go.uber.org/zap/zapcore"
)

type Email string

func (e Email) String() string { return string(e) }
func (e Email) Masked() string { return maskEmail(string(e)) }

func (e Email) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(e).Redact()))
	return nil
}
