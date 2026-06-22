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
//
// Overlay files are loaded before the base config so the base wins:
//
//	HIRING_OVERLAY_FILES=/etc/shared/microservices.json:/etc/shared/secrets.json
//	config.New("hiring.json", config.WithOverlayFiles("HIRING_OVERLAY_FILES"))
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
		return nil, Domain.Mark(err, ErrNotFound) //nolint:wrapcheck // Domain.Mark is the wrapping layer
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

	if err := loadOverlays(k, cfg.overlayFiles); err != nil {
		return nil, err
	}

	fp := file.Provider(path)
	if err := k.Load(fp, parser); err != nil {
		return nil, Domain.Wrap(err, "read config")
	}

	prefix := strings.ToUpper(cfg.envPrefix) + "_"
	if err := k.Load(env.Provider(prefix, ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, prefix)), "__", ".")
	}), nil); err != nil {
		return nil, Domain.Wrap(err, "load env")
	}

	cp := &Parser[T]{k: k, fp: fp, parser: parser, opts: cfg, validate: validator.New()}
	if err := unmarshal(k, &cp.current); err != nil {
		return nil, Domain.Mark(err, ErrMalformed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	if err := cp.validate.Struct(cp.current); err != nil {
		return nil, Domain.Mark(err, ErrMalformed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
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
	if err := cp.fp.Watch(func(_ interface{}, err error) {
		if err != nil {
			var zero T
			onChange(zero, Domain.Wrap(err, "watch event"))
			return
		}

		k := koanf.New(".")
		if len(cp.opts.defaults) > 0 {
			_ = k.Load(confmap.Provider(cp.opts.defaults, "."), nil)
		}
		if loadErr := loadOverlays(k, cp.opts.overlayFiles); loadErr != nil {
			var zero T
			onChange(zero, loadErr)
			return
		}
		if loadErr := k.Load(cp.fp, cp.parser); loadErr != nil {
			var zero T
			onChange(zero, Domain.Wrap(loadErr, "reload config"))
			return
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
	}); err != nil {
		return Domain.Wrap(err, "register file watcher")
	}
	return nil
}

const tagName = "koanf"

// Load unmarshals the koanf subtree at key into T starting from defaults, then validates.
// Use an empty key to unmarshal from the root.
//
//	redisCfg, err := config.Load[redis.Config](k, "redis", redis.Defaults)
func Load[T any](k *koanf.Koanf, key string, defaults T) (T, error) {
	out := defaults
	if err := k.UnmarshalWithConf(key, &out, koanf.UnmarshalConf{Tag: tagName}); err != nil {
		return out, Domain.Mark(err, ErrMalformed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	if err := validator.New().Struct(out); err != nil {
		return out, Domain.Mark(err, ErrMalformed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	return out, nil
}

// --- helpers ---

// loadOverlays loads each path in order into k. Missing files are silently skipped.
func loadOverlays(k *koanf.Koanf, paths []string) error {
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		op, err := parserFor(p)
		if err != nil {
			return Domain.Wrap(err, "select overlay parser")
		}
		if err := k.Load(file.Provider(p), op); err != nil {
			return Domain.Wrap(err, "load overlay")
		}
	}
	return nil
}

func unmarshal[T any](k *koanf.Koanf, out *T) error {
	return Domain.Wrap(k.UnmarshalWithConf("", out, koanf.UnmarshalConf{Tag: tagName}), "unmarshal")
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
	err          error
	defaults     map[string]any
	envPrefix    string
	overlayFiles []string
}

// Option is a functional option for [New].
type Option func(*options)

// WithEnvPrefix sets the env var prefix for automatic env binding (default: "APP").
// Nested keys use __ as delimiter: APP_LOGGER__LEVEL → logger.level.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) { o.envPrefix = prefix }
}

// WithDefaults registers key/value pairs as koanf defaults, applied when neither
// overlay files nor the base config file provide those keys.
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

// WithOverlayFiles reads envVarName from the environment and interprets it as a
// colon-separated list of config file paths to load before the base config.
// Overlays are loaded in order; the base config overrides them; env vars override everything.
// Missing files are silently skipped.
//
//	config.New("hiring.json", config.WithOverlayFiles("HIRING_OVERLAY_FILES"))
//	# HIRING_OVERLAY_FILES=/etc/shared/microservices.json:/etc/shared/secrets.json
func WithOverlayFiles(envVarName string) Option {
	return func(o *options) {
		val := os.Getenv(envVarName)
		if val == "" {
			return
		}
		for _, p := range strings.Split(val, ":") {
			if p = strings.TrimSpace(p); p != "" {
				o.overlayFiles = append(o.overlayFiles, p)
			}
		}
	}
}

// WithDefaultsFrom serializes a struct into koanf defaults using mapstructure tags.
// Use prefix to scope nested configs (e.g. "server" for a server.Config nested under "server").
func WithDefaultsFrom(v any, prefix string) Option {
	return func(o *options) {
		var m map[string]any
		dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName: tagName,
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
