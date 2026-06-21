package logger

// RotateConfig controls log file rotation. Only applies to file sinks (not stdout/stderr).
type RotateConfig struct {
	MaxSizeMB  int  `koanf:"max_size_mb"`  // rotate after N MB
	MaxBackups int  `koanf:"max_backups"`  // old files to keep
	MaxAgeDays int  `koanf:"max_age_days"` // delete after N days
	Compress   bool `koanf:"compress"`     // gzip rotated files
}

//nolint:gochecknoglobals
var RotateDefaults = RotateConfig{
	MaxSizeMB:  10,
	MaxBackups: 5,
	MaxAgeDays: 30,
	Compress:   true,
}

// SinkConfig describes one output target. Path is "stdout", "stderr", or a file path.
type SinkConfig struct {
	Path     string       `koanf:"path"     validate:"required"`
	Level    string       `koanf:"level"    validate:"omitempty,oneof=debug info warn error dpanic panic fatal"` // empty = follows root level
	Encoding string       `koanf:"encoding" validate:"omitempty,oneof=json console"`
	Rotate   RotateConfig `koanf:"rotate"` // only for file sinks
}

// Config is the top-level logger configuration.
type Config struct {
	Level           string       `koanf:"level"            validate:"required,oneof=debug info warn error dpanic panic fatal"`
	StacktraceLevel string       `koanf:"stacktrace_level" validate:"required,oneof=debug info warn error dpanic panic fatal"`
	Sinks           []SinkConfig `koanf:"sinks"            validate:"required,dive"`
	SamplingInitial int          `koanf:"sampling_initial"`    // 0 = disabled
	SamplingAfter   int          `koanf:"sampling_thereafter"` // log every Nth after initial
	Development     bool         `koanf:"development"`
	DisableCaller   bool         `koanf:"disable_caller"`
	DisableStack    bool         `koanf:"disable_stacktrace"`
}

//nolint:gochecknoglobals
var Defaults = Config{
	Level:           "info",
	StacktraceLevel: "error",
	SamplingInitial: 100,
	SamplingAfter:   100,
	Sinks: []SinkConfig{
		{Path: "stdout", Encoding: "console"},
	},
}
