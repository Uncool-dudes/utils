package pii

import (
	"net/netip"

	"github.com/cockroachdb/redact"
	"go.uber.org/zap/zapcore"
)

// IPAddress wraps netip.Addr. Validate with `validate:"ip"`.
type IPAddress struct{ addr netip.Addr }

func NewIPAddress(addr netip.Addr) IPAddress { return IPAddress{addr: addr} }

func (ip IPAddress) Addr() netip.Addr { return ip.addr }
func (ip IPAddress) String() string   { return ip.addr.String() }
func (ip IPAddress) Masked() string   { return maskIP(ip.addr) }
func (ip IPAddress) IsValid() bool    { return ip.addr.IsValid() }

func (ip IPAddress) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("value", string(redact.Sprint(ip.addr).Redact()))
	return nil
}
