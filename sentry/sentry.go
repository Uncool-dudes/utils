package sentry

import (
	sdk "github.com/getsentry/sentry-go"
	"github.com/go-playground/validator/v10"

	"github.com/uncool-dudes/utils/errors"
)

// Domain tags all errors from this package.
var Domain = errors.NewDomain("sentry")

var validate = validator.New()

// Init initialises the global Sentry hub. Call once before any goroutines that
// may panic or call CaptureException. Omit the sentry config block or leave DSN
// empty to skip init (safe for local dev).
func Init(cfg Config) error {
	if cfg.DSN == "" {
		return nil
	}
	if err := validate.Struct(cfg); err != nil {
		return Domain.Mark(err, ErrInvalidConfig) //nolint:wrapcheck // Domain.Mark/New is the wrapping layer
	}

	if err := sdk.Init(sdk.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		ServerName:       cfg.ServerName,
		Tags:             cfg.Tags,
		AttachStacktrace: cfg.AttachStacktrace,
		MaxErrorDepth:    cfg.MaxErrorDepth,
		IgnoreErrors:     cfg.IgnoreErrors,
		Debug:            cfg.Debug,
		DisableLogs:      cfg.Logging.Disable,
	}); err != nil {
		return Domain.Mark(err, ErrInit) //nolint:wrapcheck // Domain.Mark/New is the wrapping layer
	}

	return nil
}
