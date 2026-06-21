package redis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"

	goredis "github.com/redis/go-redis/v9"
)

type Service struct {
	config Config
	client *goredis.Client
}

func New(cfg Config) *Service {
	return &Service{config: cfg}
}

func (s *Service) connect(ctx context.Context) error {
	opts := &goredis.Options{
		Addr:            s.config.Addr,
		Username:        s.config.Username,
		Password:        s.config.Password,
		DB:              s.config.DB,
		PoolSize:        s.config.PoolSize,
		MinIdleConns:    s.config.MinIdleConns,
		MaxIdleConns:    s.config.MaxIdleConns,
		ConnMaxLifetime: s.config.ConnMaxLifetime,
		ConnMaxIdleTime: s.config.ConnMaxIdleTime,
		DialTimeout:     s.config.DialTimeout,
		ReadTimeout:     s.config.ReadTimeout,
		WriteTimeout:    s.config.WriteTimeout,
	}

	if s.config.TLSEnabled {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: s.config.Insecure, //nolint:gosec
		}
		if s.config.CACert != "" {
			pem, err := os.ReadFile(s.config.CACert)
			if err != nil {
				return Domain.Wrap(err, "read ca cert")
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(pem) {
				return Domain.New("invalid ca cert pem")
			}
			tlsCfg.RootCAs = pool
		}
		opts.TLSConfig = tlsCfg
	}

	s.client = goredis.NewClient(opts)

	pingCtx := ctx
	if s.config.DialTimeout > 0 {
		var cancel context.CancelFunc
		pingCtx, cancel = context.WithTimeout(ctx, s.config.DialTimeout)
		defer cancel()
	}
	if err := s.client.Ping(pingCtx).Err(); err != nil {
		_ = s.client.Close()
		s.client = nil
		return Domain.Mark(err, ErrPingFailed)
	}

	return nil
}

// Client returns the underlying go-redis client. Safe to call after OnStart.
func (s *Service) Client() *goredis.Client {
	return s.client
}

func (s *Service) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// NewConnected creates a Service and immediately connects. Use in tests and CLIs.
func NewConnected(ctx context.Context, cfg Config) (*Service, error) {
	svc := New(cfg)
	if err := svc.connect(ctx); err != nil {
		return nil, err
	}
	return svc, nil
}
