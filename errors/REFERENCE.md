# errors

`github.com/uncool-dudes/utils/errors`

Domain-aware error wrapper around `cockroachdb/errors`. Errors carry stack traces, domain tags, hints, and safe details that surface in Sentry and structured logs without leaking PII. All other packages declare a `Domain` using this package.

## Domain

```go
var Domain = errors.NewDomain("mypackage")
```

Every package declares one `Domain` at package level. Errors created or wrapped through a domain are tagged with it, allowing `Domain.Has(err)` routing and structured Sentry grouping.

## Constructors

| Method | Description |
|--------|-------------|
| `Domain.New(msg)` | New sentinel error stamped with this domain |
| `Domain.Newf(format, args...)` | Formatted sentinel |
| `Domain.NewCode(code, msg)` | Sentinel with a machine-readable code string |
| `Domain.Wrap(err, msg)` | Wrap an existing error, adding message and stack |
| `Domain.Wrapf(err, format, args...)` | Formatted wrap |
| `Domain.Mark(err, sentinel)` | Stamp `err` so `errors.Is(err, sentinel)` returns true, preserving original message and stack |
| `Domain.Has(err)` | Report whether `err` belongs to this domain |

## Package-level helpers

| Function | Description |
|----------|-------------|
| `errors.Is(err, target)` | Delegates to `cockroachdb/errors` Is |
| `errors.As(err, target)` | Delegates to `cockroachdb/errors` As |
| `errors.Unwrap(err)` | Unwrap |
| `errors.Combine(err, other)` | Combine two errors; returns the non-nil one if only one is non-nil |
| `errors.WithHint(err, hint)` | Attach a human-readable hint (appears in `%+v` and Sentry) |
| `errors.WithSafeDetail(err, format, args...)` | Attach PII-free telemetry detail |
| `errors.Hints(err)` | Return all attached hints |
| `errors.DomainOf(err)` | Return the domain name of `err`, or empty string |

## Example

```go
var Domain = errors.NewDomain("billing")
var ErrPaymentDeclined = Domain.New("payment declined")

func Charge(id string) error {
    result, err := gateway.Charge(id)
    if err != nil {
        return Domain.Wrapf(err, "charge %s", id)
    }
    if result.Declined {
        return ErrPaymentDeclined
    }
    return nil
}

// Caller:
if errors.Is(err, ErrPaymentDeclined) { ... }
```
