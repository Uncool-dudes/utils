// Package logger is a zap wrapper with multi-sink fan-out, atomic level control,
// and file rotation via lumberjack.
package logger

import (
	"fmt"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/uncool-dudes/utils/errors"
)

// Service wraps a zap.Logger with atomic level control and hot-reload support.
type Service struct {
	log  *zap.Logger
	atom zap.AtomicLevel
	cfg  Config
	mu   sync.RWMutex
}

//nolint:gocritic
func New(cfg Config, opts ...Option) (*Service, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	atom, err := parseAtomicLevel(cfg.Level)
	if err != nil {
		return nil, Domain.Mark(err, ErrInvalidLevel)
	}

	stackLevel, err := parseLevel(cfg.StacktraceLevel)
	if err != nil {
		return nil, Domain.Mark(err, ErrInvalidLevel)
	}

	sinks := cfg.Sinks
	if len(sinks) == 0 {
		sinks = Defaults.Sinks
	}

	cores := make([]zapcore.Core, 0, len(sinks))
	for _, s := range sinks {
		core, err := buildCore(&s, atom, cfg)
		if err != nil {
			return nil, err
		}
		if cfg.SamplingInitial > 0 {
			samplerOpts := []zapcore.SamplerOption{}
			if o.samplingHook != nil {
				samplerOpts = append(samplerOpts, zapcore.SamplerHook(o.samplingHook))
			}
			core = zapcore.NewSamplerWithOptions(
				core,
				time.Second,
				cfg.SamplingInitial,
				cfg.SamplingAfter,
				samplerOpts...,
			)
		}
		cores = append(cores, core)
	}

	cores = append(cores, o.extraCores...)

	var tee zapcore.Core
	if len(cores) == 1 {
		tee = cores[0]
	} else {
		tee = zapcore.NewTee(cores...)
	}

	zapOpts := []zap.Option{}
	if !cfg.DisableStack {
		zapOpts = append(zapOpts, zap.AddStacktrace(stackLevel))
	}
	if !cfg.DisableCaller {
		zapOpts = append(zapOpts, zap.AddCaller())
	}
	if cfg.Development {
		zapOpts = append(zapOpts, zap.Development())
	}
	if o.preWriteHook != nil {
		zapOpts = append(zapOpts, zap.Hooks(o.preWriteHook))
	}

	log := zap.New(tee, zapOpts...)
	return &Service{atom: atom, cfg: cfg, log: log}, nil
}

func (s *Service) Logger() *zap.Logger {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.log
}

func (s *Service) Named(name string) *zap.Logger {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.log.Named(name)
}

func (s *Service) SetLevel(l zapcore.Level) { s.atom.SetLevel(l) }

//nolint:gocritic
func (s *Service) Reload(cfg Config, opts ...Option) error {
	lvl, err := parseLevel(cfg.Level)
	if err != nil {
		return Domain.Mark(err, ErrInvalidLevel)
	}
	s.atom.SetLevel(lvl)

	svc, err := New(cfg, opts...)
	if err != nil {
		return err
	}

	s.mu.Lock()
	oldLog := s.log
	s.cfg = cfg
	s.log = svc.log
	s.mu.Unlock()
	_ = oldLog.Sync()
	return nil
}

func (s *Service) Sync() error {
	s.mu.RLock()
	log := s.log
	s.mu.RUnlock()
	err := log.Sync()
	if err == nil {
		return nil
	}
	if errors.Is(err, syscall.ENOTTY) || errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.EBADF) {
		return nil
	}
	return Domain.Wrapf(err, "sync logger")
}

type options struct {
	samplingHook func(zapcore.Entry, zapcore.SamplingDecision)
	preWriteHook func(zapcore.Entry) error
	extraCores   []zapcore.Core
}

type Option func(*options)

func WithSamplingHook(fn func(zapcore.Entry, zapcore.SamplingDecision)) Option {
	return func(o *options) { o.samplingHook = fn }
}

func WithPreWriteHook(fn func(zapcore.Entry) error) Option {
	return func(o *options) { o.preWriteHook = fn }
}

func WithExtraCore(c zapcore.Core) Option {
	return func(o *options) { o.extraCores = append(o.extraCores, c) }
}

//nolint:gocritic
func buildCore(s *SinkConfig, atom zap.AtomicLevel, cfg Config) (zapcore.Core, error) {
	enc, err := buildEncoder(s.Encoding, cfg.Development)
	if err != nil {
		return nil, err
	}
	w, err := buildWriter(s)
	if err != nil {
		return nil, Domain.Wrapf(err, "open sink %q", s.Path)
	}
	if s.Level == "" {
		return zapcore.NewCore(enc, w, atom), nil
	}
	lvl, err := parseLevel(s.Level)
	if err != nil {
		return nil, Domain.Mark(err, ErrInvalidLevel)
	}
	return zapcore.NewCore(enc, w, zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= lvl && atom.Enabled(l)
	})), nil
}

func buildEncoder(encoding string, development bool) (zapcore.Encoder, error) {
	if encoding == "" {
		if development {
			encoding = "console"
		} else {
			encoding = "json"
		}
	}
	ecfg := zap.NewProductionEncoderConfig()
	ecfg.EncodeTime = zapcore.ISO8601TimeEncoder
	switch encoding {
	case "json":
		return zapcore.NewJSONEncoder(ecfg), nil
	case "console":
		ecfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(ecfg), nil
	default:
		return nil, Domain.Mark(fmt.Errorf("unknown encoding %q", encoding), ErrInvalidEncoding)
	}
}

func buildWriter(s *SinkConfig) (zapcore.WriteSyncer, error) {
	path := s.Path
	if path == "" {
		path = "stdout"
	}
	if path == "stdout" || path == "stderr" {
		ws, _, err := zap.Open(path)
		if err != nil {
			return ws, Domain.Wrapf(err, "open %s", path)
		}
		return ws, nil
	}
	r := s.Rotate
	if r.MaxSizeMB == 0 {
		r.MaxSizeMB = RotateDefaults.MaxSizeMB
	}
	if r.MaxBackups == 0 {
		r.MaxBackups = RotateDefaults.MaxBackups
	}
	if r.MaxAgeDays == 0 {
		r.MaxAgeDays = RotateDefaults.MaxAgeDays
	}
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   path,
		MaxSize:    r.MaxSizeMB,
		MaxBackups: r.MaxBackups,
		MaxAge:     r.MaxAgeDays,
		Compress:   r.Compress,
	}), nil
}

func parseAtomicLevel(s string) (zap.AtomicLevel, error) {
	if s == "" {
		return zap.NewAtomicLevelAt(zapcore.InfoLevel), nil
	}
	lvl, err := zapcore.ParseLevel(s)
	if err != nil {
		return zap.AtomicLevel{}, Domain.Mark(err, ErrInvalidLevel)
	}
	return zap.NewAtomicLevelAt(lvl), nil
}

func parseLevel(s string) (zapcore.Level, error) {
	if s == "" {
		return zapcore.InfoLevel, nil
	}
	lvl, err := zapcore.ParseLevel(s)
	if err != nil {
		return zapcore.InfoLevel, Domain.Mark(err, ErrInvalidLevel)
	}
	return lvl, nil
}
