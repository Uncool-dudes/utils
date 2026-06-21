package db

import "time"

// Config holds pgxpool connection settings.
type Config struct {
	URL string `koanf:"url" validate:"required"`

	// pool sizing
	MinConns int32 `koanf:"min_conns" validate:"min=0"`
	MaxConns int32 `koanf:"max_conns" validate:"min=1,gtefield=MinConns"`

	// connection lifecycle
	MaxConnLifetime       time.Duration `koanf:"max_conn_lifetime"`
	MaxConnLifetimeJitter time.Duration `koanf:"max_conn_lifetime_jitter"` // prevents thundering herd on expiry
	MaxConnIdleTime       time.Duration `koanf:"max_conn_idle_time"`
	ConnectTimeout        time.Duration `koanf:"connect_timeout"`
	PingTimeout           time.Duration `koanf:"ping_timeout"`

	// health check
	HealthCheckPeriod time.Duration `koanf:"health_check_period"`
}

// Defaults provides sane out-of-the-box Config values.
var Defaults = Config{
	MaxConns:              20,
	MinConns:              2,
	MaxConnLifetime:       time.Hour,
	MaxConnLifetimeJitter: 5 * time.Minute,
	MaxConnIdleTime:       30 * time.Minute,
	ConnectTimeout:        5 * time.Second,
	PingTimeout:           5 * time.Second,
	HealthCheckPeriod:     time.Minute,
}
