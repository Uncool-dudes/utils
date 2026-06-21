package mailer

import "time"

// Config holds connection settings for the mailer.
type Config struct {
	Host     string `koanf:"host"     validate:"required"`
	Port     int    `koanf:"port"     validate:"required,min=1,max=65535"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	From     string `koanf:"from"`
	FromName string `koanf:"from_name"`

	TLSEnabled bool `koanf:"tls_enabled"`
	StartTLS   bool `koanf:"start_tls"`
	Insecure   bool `koanf:"insecure"`

	Timeout time.Duration `koanf:"timeout"`
}

// Defaults provides sane out-of-the-box Config values.
var Defaults = Config{
	Host:     "localhost",
	Port:     587,
	StartTLS: true,
	Timeout:  10 * time.Second,
}
