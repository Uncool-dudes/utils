package vault

import "time"

// Config holds connection and auth settings for the Vault client.
type Config struct {
	Addr      string        `koanf:"addr"      validate:"required"`
	Namespace string        `koanf:"namespace"`
	Timeout   time.Duration `koanf:"timeout"`
	Auth      AuthConfig    `koanf:"auth"`
	KV        KVConfig      `koanf:"kv"`
}

// AuthConfig defines how the client authenticates with Vault.
type AuthConfig struct {
	Method string `koanf:"method" validate:"required,oneof=token approle"`

	Token string `koanf:"token"`

	RoleID   string `koanf:"role_id"`
	SecretID string `koanf:"secret_id"`
}

// KVConfig holds KV v2 engine settings.
type KVConfig struct {
	MountPath string `koanf:"mount_path"`
}

// Defaults provides sane out-of-the-box Config values.
var Defaults = Config{
	Timeout: 10 * time.Second,
	KV:      KVConfig{MountPath: "secret"},
}
