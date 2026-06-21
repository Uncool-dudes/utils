# redis

`github.com/uncool-dudes/utils/redis`

Redis/Valkey client with connection pooling, TLS, and fx lifecycle. Thin wrapper over [go-redis v9](https://github.com/redis/go-redis) — no opinionated helpers, callers use `Client()` directly.

## fx wiring

```go
type AppConfig struct {
    Redis redis.Config `koanf:"redis"`
}

fx.New(
    fx.Supply(cfg.Redis),
    redis.Module,
    // redis.Service is now available for injection
)
```

## config.yaml

```yaml
redis:
  addr: "localhost:6379"
  # username, password: omit for unauthenticated local dev
  # db: 0
```

All fields with defaults:

| Field | Default | Notes |
|---|---|---|
| `addr` | `localhost:6379` | Required |
| `db` | `0` | |
| `pool_size` | `10` | |
| `min_idle_conns` | `2` | |
| `max_idle_conns` | `5` | |
| `conn_max_lifetime` | `30m` | |
| `conn_max_idle_time` | `5m` | |
| `dial_timeout` | `5s` | Also used as ping timeout on connect |
| `read_timeout` | `3s` | |
| `write_timeout` | `3s` | |
| `tls_enabled` | `false` | Set `true` for Memorystore |
| `ca_cert` | `""` | Path to PEM; required if server uses a private CA |
| `insecure` | `false` | Skip TLS verification — dev only |

## using the client

```go
type MyService struct {
    redis *goredis.Client
}

func New(redisSvc *redis.Service) *MyService {
    return &MyService{redis: redisSvc.Client()}
}

func (s *MyService) Set(ctx context.Context, key, val string, ttl time.Duration) error {
    return s.redis.Set(ctx, key, val, ttl).Err()
}

func (s *MyService) Get(ctx context.Context, key string) (string, error) {
    val, err := s.redis.Get(ctx, key).Result()
    if errors.Is(err, goredis.Nil) {
        return "", nil // key not found
    }
    return val, err
}
```

## tests / CLIs (no fx)

```go
svc, err := redis.NewConnected(ctx, redis.Config{Addr: "localhost:6379"})
if err != nil {
    return err
}
defer svc.Close()
client := svc.Client()
```

## TLS (GCP Memorystore)

```yaml
redis:
  addr: "10.0.0.1:6378"
  tls_enabled: true
  # ca_cert: omit — uses system roots for Memorystore managed certs
```

## errors

```go
redis.ErrConnFailed  // connection could not be established
redis.ErrPingFailed  // connected but ping failed
redis.ErrNil         // key not found (wraps goredis.Nil)
```
