package redis

import "time"

// Config holds connection settings for the Redis/Valkey client.
type Config struct {
	Addr     string `koanf:"addr"     validate:"required"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`

	TLSEnabled bool   `koanf:"tls_enabled"`
	CACert     string `koanf:"ca_cert"`
	Insecure   bool   `koanf:"insecure"`

	PoolSize        int           `koanf:"pool_size"`
	MinIdleConns    int           `koanf:"min_idle_conns"`
	MaxIdleConns    int           `koanf:"max_idle_conns"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `koanf:"conn_max_idle_time"`

	DialTimeout  time.Duration `koanf:"dial_timeout"`
	ReadTimeout  time.Duration `koanf:"read_timeout"`
	WriteTimeout time.Duration `koanf:"write_timeout"`
}

// Defaults provides sane out-of-the-box Config values.
var Defaults = Config{
	Addr:            "localhost:6379",
	DB:              0,
	PoolSize:        10,
	MinIdleConns:    2,
	MaxIdleConns:    5,
	ConnMaxLifetime: 30 * time.Minute,
	ConnMaxIdleTime: 5 * time.Minute,
	DialTimeout:     5 * time.Second,
	ReadTimeout:     3 * time.Second,
	WriteTimeout:    3 * time.Second,
}
