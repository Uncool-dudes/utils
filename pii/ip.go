package pii

import (
	"net/netip"

	"github.com/cockroachdb/redact"
	"go.uber.org/zap/zapcore"
)

// IPAddress wraps netip.Addr. Validate with `validate:"ip"`.
type IPAddress struct{ addr netip.Addr }

// NewIPAddress wraps addr in an IPAddress.
func NewIPAddress(addr netip.Addr) IPAddress { return IPAddress{addr: addr} }

// Addr returns the underlying netip.Addr.
func (ip IPAddress) Addr() netip.Addr { return ip.addr }
func (ip IPAddress) String() string   { return ip.addr.String() }

// Masked returns the IP address with the host portion redacted.
func (ip IPAddress) Masked() string { return maskIP(ip.addr) }

// IsValid reports whether the IP address is valid.
func (ip IPAddress) IsValid() bool { return ip.addr.IsValid() }

// MarshalLogObject implements zapcore.ObjectMarshaler, logging the masked value.
func (ip IPAddress) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(ip.addr).Redact()))
	return nil
}
