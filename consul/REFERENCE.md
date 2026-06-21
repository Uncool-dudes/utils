# consul

`github.com/uncool-dudes/utils/consul`

Consul service registration and discovery. Registers the service with `/healthz` and `/readyz` health checks, and includes the `prometheus` tag and `metrics_path` meta for Prometheus Consul SD.

## fx

```go
fx.Supply(consulCfg),
httpserver.Module,
consul.ModuleFor("my-service"),
// OnStart: Register; OnStop: Deregister
// Registration failures are logged as warnings, not fatal errors.
```

Port is derived automatically from `httpserver.Config.Addr`.

## Constructor (non-fx)

```go
c, err := consul.New(cfg)
err = c.Register("my-service", 8080)
addr, err := c.Lookup("other-service") // returns "host:port"
err = c.Deregister()
```

## Client methods

| Method | Description |
|--------|-------------|
| `c.Register(svcName, httpPort)` | Register service with liveness + readiness checks |
| `c.Deregister()` | Remove the registered service |
| `c.Lookup(svcName)` | Return `host:port` of the first healthy instance |

## Config

```go
type Config struct {
    Addr string            // Consul agent "host:port"
    Tags []string          // additional tags; "prometheus" is always ensured
    Meta map[string]string // included in registration; exposed to Prometheus SD
}
```

## Defaults

| Field | Default |
|-------|---------|
| `Addr` | `"localhost:8500"` |
| `Tags` | `["prometheus"]` |
| `Meta` | `{"metrics_path": "/metrics"}` |

## Errors

| Sentinel | Meaning |
|----------|---------|
| `consul.ErrInvalidAddr` | Addr is not a valid `host:port` |
| `consul.ErrConnect` | Consul client creation failed |
| `consul.ErrRegister` | Service registration failed |
| `consul.ErrDeregister` | Service deregistration failed |
| `consul.ErrLookup` | Health API call failed |
| `consul.ErrNoInstances` | No healthy instances found for the service |
