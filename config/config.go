// Package config provides a typed, generic config parser backed by koanf.
//
// It supports JSON, YAML, and TOML files, env var overrides, struct defaults,
// and hot reload via fsnotify.
//
// Typical usage:
//
//	type AppConfig struct {
//	    Host string `koanf:"host"`
//	    Port int    `koanf:"port"`
//	}
//
//	cp, err := config.New[AppConfig]("/etc/myapp/config.yaml",
//	    config.WithEnvPrefix("APP"),
//	    config.WithDefaultsFrom(config.Defaults, ""),
//	)
//	cfg := cp.Get()
//
// Hot reload:
//
//	cp.Watch(func(v AppConfig, err error) {
//	    if err != nil { log.Error("reload failed", err); return }
//	    applyNewConfig(v)
//	})
//
// Nested env var overrides use __ as the key delimiter:
//
//	APP_LOGGER__LEVEL=debug  →  cfg.Logger.Level = "debug"
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	mapstructure "github.com/go-viper/mapstructure/v2"
	kjson "github.com/knadh/koanf/parsers/json"
	ktoml "github.com/knadh/koanf/parsers/toml"
	kyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Parser holds the parsed config and manages hot reload.
type Parser[T any] struct {
	current  T
	k        *koanf.Koanf
	fp       *file.File
	parser   koanf.Parser
	opts     *options
	validate *validator.Validate
	mu       sync.RWMutex
}

// New parses the config file at path into T and returns a Parser.
//
// Returns [ErrNotFound] if the file does not exist, [ErrMalformed] if the file
// cannot be decoded into T, or a wrapped domain error for other failures.
func New[T any](path string, opts ...Option) (*Parser[T], error) {
	cfg := &options{
		envPrefix: "APP",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.err != nil {
		return nil, Domain.Wrap(cfg.err, "build options")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, Domain.Mark(err, ErrNotFound)
	}

	parser, err := parserFor(path)
	if err != nil {
		return nil, Domain.Wrap(err, "select parser")
	}

	k := koanf.New(".")

	if len(cfg.defaults) > 0 {
		if err := k.Load(confmap.Provider(cfg.defaults, "."), nil); err != nil {
			return nil, Domain.Wrap(err, "load defaults")
		}
	}

	fp := file.Provider(path)
	if err := k.Load(fp, parser); err != nil {
		return nil, Domain.Wrap(err, "read config")
	}

	if cfg.overlayEnvVarName != "" {
		if appEnv := os.Getenv(cfg.overlayEnvVarName); appEnv != "" {
			ext := filepath.Ext(path)
			base := strings.TrimSuffix(filepath.Base(path), ext)
			cfg.overlayPath = filepath.Join(filepath.Dir(path), base+"."+appEnv+ext)
		}
	}
	if cfg.overlayPath != "" {
		if _, statErr := os.Stat(cfg.overlayPath); statErr == nil {
			op, _ := parserFor(cfg.overlayPath)
			_ = k.Load(file.Provider(cfg.overlayPath), op)
		}
	}

	prefix := strings.ToUpper(cfg.envPrefix) + "_"
	if err := k.Load(env.Provider(prefix, ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, prefix)), "__", ".")
	}), nil); err != nil {
		return nil, Domain.Wrap(err, "load env")
	}

	cp := &Parser[T]{k: k, fp: fp, parser: parser, opts: cfg, validate: validator.New()}
	if err := unmarshal(k, &cp.current); err != nil {
		return nil, Domain.Mark(err, ErrMalformed)
	}
	if err := cp.validate.Struct(cp.current); err != nil {
		return nil, Domain.Mark(err, ErrMalformed)
	}

	return cp, nil
}

// Get returns the current parsed config. Safe to call from any goroutine.
func (cp *Parser[T]) Get() T {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.current
}

// Watch registers onChange to be called whenever the config file changes.
// onChange receives the new value and any unmarshal error. The Parser's
// internal state is only updated on a successful unmarshal.
// Returns an error if the file watcher cannot be registered.
func (cp *Parser[T]) Watch(onChange func(T, error)) error {
	return cp.fp.Watch(func(_ interface{}, err error) {
		if err != nil {
			var zero T
			onChange(zero, Domain.Wrap(err, "watch event"))
			return
		}

		k := koanf.New(".")
		if len(cp.opts.defaults) > 0 {
			_ = k.Load(confmap.Provider(cp.opts.defaults, "."), nil)
		}
		if loadErr := k.Load(cp.fp, cp.parser); loadErr != nil {
			var zero T
			onChange(zero, Domain.Wrap(loadErr, "reload config"))
			return
		}
		if cp.opts.overlayPath != "" {
			if _, statErr := os.Stat(cp.opts.overlayPath); statErr == nil {
				op, _ := parserFor(cp.opts.overlayPath)
				_ = k.Load(file.Provider(cp.opts.overlayPath), op)
			}
		}
		prefix := strings.ToUpper(cp.opts.envPrefix) + "_"
		_ = k.Load(env.Provider(prefix, ".", func(s string) string {
			return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, prefix)), "__", ".")
		}), nil)

		var val T
		if unmarshalErr := unmarshal(k, &val); unmarshalErr != nil {
			var zero T
			onChange(zero, Domain.Mark(unmarshalErr, ErrMalformed))
			return
		}
		if validateErr := cp.validate.Struct(val); validateErr != nil {
			var zero T
			onChange(zero, Domain.Mark(validateErr, ErrMalformed))
			return
		}
		cp.mu.Lock()
		cp.current = val
		cp.k = k
		cp.mu.Unlock()
		onChange(val, nil) // called outside lock — onChange may call Get()
	})
}

// --- helpers ---

func unmarshal[T any](k *koanf.Koanf, out *T) error {
	return k.UnmarshalWithConf("", out, koanf.UnmarshalConf{Tag: "koanf"})
}

func parserFor(path string) (koanf.Parser, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return kjson.Parser(), nil
	case ".yaml", ".yml":
		return kyaml.Parser(), nil
	case ".toml":
		return ktoml.Parser(), nil
	default:
		return nil, fmt.Errorf("unknown config format %q", filepath.Ext(path))
	}
}

// --- options ---

type options struct {
	err               error
	defaults          map[string]any
	envPrefix         string
	overlayEnvVarName string
	overlayPath       string // computed in New from overlayEnvVarName + primary path
}

// Option is a functional option for [New].
type Option func(*options)

// WithEnvPrefix sets the env var prefix for automatic env binding (default: "APP").
// Nested keys use __ as delimiter: APP_LOGGER__LEVEL → logger.level.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) { o.envPrefix = prefix }
}

// WithDefaults registers key/value pairs as koanf defaults, applied when the
// config file omits those keys.
func WithDefaults(d map[string]any) Option {
	return func(o *options) {
		if o.defaults == nil {
			o.defaults = make(map[string]any, len(d))
		}
		for k, v := range d {
			o.defaults[k] = v
		}
	}
}

// WithEnvOverlay enables environment-specific config overlays.
// It reads envVarName at startup (e.g. "APP_ENV") and, if set, loads a sibling file
// named config.<value>.yaml alongside the primary config. Missing overlay files are
// silently ignored. Overlay values take precedence over the primary file but are
// overridden by env vars.
//
//	config.New("config.yaml", config.WithEnvOverlay("APP_ENV"))
//	# APP_ENV=staging → loads config.staging.yaml if it exists
func WithEnvOverlay(envVarName string) Option {
	return func(o *options) { o.overlayEnvVarName = envVarName }
}

// WithDefaultsFrom serializes a struct into koanf defaults using mapstructure tags.
// Use prefix to scope nested configs (e.g. "server" for a server.Config nested under "server").
func WithDefaultsFrom(v any, prefix string) Option {
	return func(o *options) {
		var m map[string]any
		dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName: "koanf",
			Result:  &m,
		})
		if err != nil {
			o.err = err
			return
		}
		if err := dec.Decode(v); err != nil {
			o.err = err
			return
		}
		if o.defaults == nil {
			o.defaults = make(map[string]any)
		}
		for k, val := range m {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			o.defaults[key] = val
		}
	}
}
