# pii

`github.com/uncool-dudes/utils/pii`

Named types for personally identifiable information. All types are thin string (or struct) wrappers that redact automatically when logged via zap, and expose a `.Masked()` method for safe display.

## Types

| Type | Validate tag | Masked example |
|------|-------------|----------------|
| `pii.Email` | `validate:"email"` | `j***@example.com` |
| `pii.Phone` | `validate:"e164"` | `+91*******210` |
| `pii.IPAddress` | `validate:"ip"` | `192.168.1.xxx` |
| `pii.FirstName` | — | `J***n` |
| `pii.LastName` | — | `D**` |
| `pii.FullName` | — | `J** D**` |
| `pii.TaxID` | — | `AB***67` |

All types implement `zapcore.ObjectMarshaler` — raw values are never emitted to logs.

## Zap field helpers

```go
log.Info("user created",
    pii.EmailField("email", user.Email),
    pii.PhoneField("phone", user.Phone),
    pii.IPField("ip", user.IP),
    pii.FirstNameField("first_name", user.FirstName),
    pii.LastNameField("last_name", user.LastName),
    pii.FullNameField("full_name", user.FullName),
    pii.TaxIDField("tax_id", user.TaxID),
)
// → {"email": "j***@example.com", "phone": "+91*******210", ...}
```

## IPAddress construction

```go
addr, _ := netip.ParseAddr("192.168.1.42")
ip := pii.NewIPAddress(addr)
ip.Addr()    // netip.Addr
ip.Masked()  // "192.168.1.xxx"
ip.IsValid() // true
```

## Usage in request structs

```go
type CreateUserRequest struct {
    Email pii.Email `validate:"required,email"`
    Phone pii.Phone `validate:"required,e164"`
}
```

## Direct masking

Each type exposes `.Masked()` for safe display in responses or UI:

```go
user.Email.Masked()  // "j***@example.com"
user.Phone.Masked()  // "+91*******210"
```
