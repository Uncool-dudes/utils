package httpserver

import "time"

// Config holds HTTP server settings.
type Config struct {
	Addr            string        `koanf:"addr"             validate:"required"`
	ReadTimeout     time.Duration `koanf:"read_timeout"`
	WriteTimeout    time.Duration `koanf:"write_timeout"`
	IdleTimeout     time.Duration `koanf:"idle_timeout"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
}

// Defaults provides sane out-of-the-box Config values.
var Defaults = Config{
	ReadTimeout:     5 * time.Second,
	WriteTimeout:    10 * time.Second,
	IdleTimeout:     60 * time.Second,
	ShutdownTimeout: 10 * time.Second,
}
