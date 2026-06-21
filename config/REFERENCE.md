# config

`github.com/uncool-dudes/utils/config`

Generic typed config parser backed by koanf. Supports JSON, YAML, and TOML files; env var overrides; struct defaults; and hot reload via fsnotify.

## Constructor

```go
cp, err := config.New[AppConfig]("/etc/myapp/config.yaml", ...opts)
cfg := cp.Get()
```

Returns `ErrNotFound` if the file does not exist; `ErrMalformed` if it cannot be decoded or fails struct validation.

## Methods

| Method | Description |
|--------|-------------|
| `cp.Get()` | Return the current config. Goroutine-safe. |
| `cp.Watch(func(T, error))` | Register a callback invoked on file changes. Internal state only updates on a successful reload. |

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithEnvPrefix(prefix)` | `"APP"` | Prefix for env var binding. Nested keys use `__` as delimiter: `APP_LOGGER__LEVEL` → `logger.level` |
| `WithDefaults(map[string]any)` | — | Raw koanf key/value defaults |
| `WithDefaultsFrom(v, prefix)` | — | Serialize a struct into koanf defaults. Use `prefix` to scope nested configs (e.g. `"server"`) |
| `WithEnvOverlay(envVarName)` | — | Read `envVarName` at startup (e.g. `APP_ENV=staging`) and load a sibling file `config.staging.yaml` if it exists. Overlay wins over primary file, loses to env vars. Missing overlay files are silently ignored. |

## Errors

| Sentinel | Meaning |
|----------|---------|
| `config.ErrNotFound` | Config file does not exist |
| `config.ErrMalformed` | File failed to decode or failed struct validation |

## Example

```go
type AppConfig struct {
    Host string `koanf:"host"`
    Port int    `koanf:"port"`
}

cp, err := config.New[AppConfig]("config.yaml",
    config.WithEnvPrefix("APP"),
    config.WithDefaultsFrom(myDefaults, ""),
    config.WithEnvOverlay("APP_ENV"),
)

cfg := cp.Get()

cp.Watch(func(v AppConfig, err error) {
    if err != nil { return }
    applyNewConfig(v)
})
```

Nested env override: `APP_LOGGER__LEVEL=debug` → `cfg.Logger.Level = "debug"`
