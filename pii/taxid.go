package pii

import (
	"github.com/cockroachdb/redact"
	"go.uber.org/zap/zapcore"
)

// TaxID is a locale-specific tax identifier (e.g. GST, VAT, EIN).
// No format validation — locale rules vary. Use a custom validator if needed.
type TaxID string

func (t TaxID) String() string { return string(t) }
func (t TaxID) Masked() string { return maskTaxID(string(t)) }

func (t TaxID) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(t).Redact()))
	return nil
}
