package sentry

import (
	sdk "github.com/getsentry/sentry-go"
	"github.com/go-playground/validator/v10"

	"github.com/ratchio/utils/errors"
)

var (
	Domain   = errors.NewDomain("sentry")
	validate = validator.New()
)

// Init initialises the global Sentry hub. Call once before any goroutines that
// may panic or call CaptureException. Omit the sentry config block to skip init.
func Init(cfg Config) error {
	if err := validate.Struct(cfg); err != nil {
		return Domain.Mark(err, ErrInvalidConfig)
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
		return Domain.Mark(err, ErrInit)
	}

	return nil
}
