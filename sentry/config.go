package sentry

// LoggingOptions controls Sentry's log integration behaviour.
type LoggingOptions struct {
	Disable bool `json:"disable"` // set true to opt out of Sentry log capture (enabled by default since v0.47)
}

// Config holds Sentry initialisation options sourced from the service config file.
type Config struct {
	DSN              string            `json:"dsn"               validate:"required"`
	Environment      string            `json:"environment"       validate:"omitempty,oneof=production staging development"`
	ServerName       string            `json:"server_name"`
	Tags             map[string]string `json:"tags"`
	AttachStacktrace bool              `json:"attach_stacktrace"`
	MaxErrorDepth    int               `json:"max_error_depth"   validate:"min=0"`
	IgnoreErrors     []string          `json:"ignore_errors"`
	Debug            bool              `json:"debug"`
	Logging          LoggingOptions    `json:"logging"`
}

// Defaults provides sane out-of-the-box Config values.
var Defaults = Config{
	Tags:             map[string]string{},
	AttachStacktrace: true,
	MaxErrorDepth:    10,
	IgnoreErrors:     []string{},
}
