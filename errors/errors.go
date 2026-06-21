// Package errors is a thin domain-aware wrapper around [github.com/cockroachdb/errors].
//
// Callers import only this package; the underlying library is an implementation detail.
// Errors carry stack traces, domain tags, hints, and safe details that surface in Sentry
// and structured logs without leaking PII.
//
// Typical usage:
//
//	var ErrNotFound = myDomain.New("not found")
//
//	func Lookup(id string) error {
//	    row, err := db.Query(id)
//	    if err != nil {
//	        return myDomain.Wrapf(err, "lookup %s", id)
//	    }
//	    if row == nil {
//	        return ErrNotFound
//	    }
//	    return nil
//	}
//
//	// Caller:
//	if errors.Is(err, ErrNotFound) { ... }
package errors

import (
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
)

// Is reports whether any error in err's chain matches target.
func Is(err, target error) bool { return errors.Is(err, target) }

// As finds the first error in err's chain matching target, and sets target to that error value.
func As(err error, target any) bool { return errors.As(err, target) }

// Unwrap returns the result of calling Unwrap on err, if err's type contains an Unwrap method returning error.
func Unwrap(err error) error { return errors.Unwrap(err) }

// Combine returns a combined error wrapping both. If either is nil, the other is returned.
func Combine(err, other error) error {
	return errors.CombineErrors(err, other) //nolint:wrapcheck // delegating to cockroachdb/errors which does not wrap
}

// Domain groups errors under a named tag for structured routing and Is-matching.
//
// Declare one Domain per package as a package-level var:
//
//	var dom = errors.NewDomain("billing")
//	var ErrPaymentDeclined = dom.New("payment declined")
type Domain struct {
	d errors.Domain
}

// NewDomain creates a named Domain for tagging errors from a package.
func NewDomain(name string) Domain {
	return Domain{d: errors.NamedDomain(name)}
}

// codeErr pairs a machine-readable code with a domain error.
// Participates in errors.Is matching via pointer identity; Unwrap walks the chain.
type codeErr struct {
	cause error
	code  string
}

func (c *codeErr) Error() string { return c.cause.Error() }
func (c *codeErr) Unwrap() error { return c.cause }

// NewCode creates a sentinel carrying a machine-readable code, stamped with this domain.
//
//	var ErrNotFound = reqDomain.NewCode("ERR_NOT_FOUND", "not found")
func (d Domain) NewCode(code, msg string) error {
	return &codeErr{
		code:  code,
		cause: errors.WithDomain(errors.NewWithDepth(1, msg), d.d),
	}
}

// New creates a new sentinel error stamped with this domain.
func (d Domain) New(msg string) error {
	return errors.WithDomain(errors.NewWithDepth(1, msg), d.d) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// Newf creates a new formatted sentinel error stamped with this domain.
func (d Domain) Newf(format string, args ...any) error {
	return errors.WithDomain(errors.NewWithDepthf(1, format, args...), d.d) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// NewWithDepth skips depth additional frames — use in helper constructors so they
// don't appear in the recorded stack trace.
func (d Domain) NewWithDepth(depth int, msg string) error {
	return errors.WithDomain(errors.NewWithDepth(depth+1, msg), d.d) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// NewWithDepthf creates a new formatted error, skipping depth extra stack frames.
func (d Domain) NewWithDepthf(depth int, format string, args ...any) error {
	return errors.WithDomain(errors.NewWithDepthf(depth+1, format, args...), d.d) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// Wrap annotates err with msg and stamps it with this domain.
func (d Domain) Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return errors.WithDomain(errors.WrapWithDepth(1, err, msg), d.d) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// Wrapf annotates err with a formatted message and stamps it with this domain.
func (d Domain) Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return errors.WithDomain(errors.WrapWithDepthf(1, err, format, args...), d.d) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// WrapWithDepth wraps err, skipping depth extra stack frames from the recorded trace.
func (d Domain) WrapWithDepth(depth int, err error, msg string) error {
	if err == nil {
		return nil
	}
	return errors.WithDomain(errors.WrapWithDepth(depth+1, err, msg), d.d) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// WrapWithDepthf wraps err with a formatted message, skipping depth extra stack frames.
func (d Domain) WrapWithDepthf(depth int, err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return errors.WithDomain(errors.WrapWithDepthf(depth+1, err, format, args...), d.d) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// Mark stamps err with the identity of sentinel so Is(err, sentinel) returns true,
// preserving the original message and stack.
//
//	if errors.Is(err, sql.ErrNoRows) {
//	    return dom.Mark(err, ErrNotFound)
//	}
func (d Domain) Mark(err, sentinel error) error {
	if err == nil {
		return nil
	}
	return errors.WithDomain(errors.Mark(err, sentinel), d.d) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// Has reports whether err belongs to this domain.
func (d Domain) Has(err error) bool {
	return errors.GetDomain(err) == d.d
}

// WithHint attaches a human-readable hint that surfaces in %+v and Sentry reports.
func WithHint(err error, hint string) error {
	return errors.WithHint(err, hint) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// WithSafeDetail attaches PII-free detail safe for telemetry and structured logging.
func WithSafeDetail(err error, format string, args ...any) error {
	return errors.WithSafeDetails(err, format, args...) //nolint:wrapcheck // cockroachdb/errors internals manage the chain
}

// Hints returns all hint strings attached to err's chain.
func Hints(err error) []string {
	return errors.GetAllHints(err)
}

// DomainOf returns the domain name tag of err, or empty string if none.
func DomainOf(err error) string {
	raw := string(errors.GetDomain(err))
	const prefix = "error domain: "
	if !strings.HasPrefix(raw, prefix) {
		return ""
	}
	quoted := raw[len(prefix):]
	if quoted == "<none>" {
		return ""
	}
	name, e := strconv.Unquote(quoted)
	if e != nil {
		return ""
	}
	return name
}
