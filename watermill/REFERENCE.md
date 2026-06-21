# watermill

`github.com/uncool-dudes/utils/watermill`

Kafka pub/sub via Watermill. Supports SASL (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512) and mutual TLS. The router comes pre-wired with correlation ID propagation, logging, poison queue, retry, and panic recovery.

## fx

```go
fx.Supply(watermillCfg),
logger.Module,
watermill.Module,
// provides: message.Publisher, message.Subscriber, *message.Router
// OnStart: starts router, waits for Running(); OnStop: cancels context + closes router
```

## Publishing

```go
// Marshals v as JSON and publishes to topic.
// Sets a correlation ID on the message if not already present in ctx.
err := watermill.Publish(ctx, pub, "orders.created", myEvent)
```

## Subscribing

```go
// Handle returns a watermill.HandlerFunc that unmarshals the payload into T.
// Unmarshal failures Nack immediately; handler errors propagate to retry/poison queue.
router.AddHandler(
    "order-handler",
    "orders.created", sub,
    "orders.processed", pub,
    watermill.Handle(func(ctx context.Context, evt OrderCreatedEvent) error {
        return processOrder(ctx, evt)
    }),
)
```

## Router middleware stack

Applied automatically by `NewRouter` in this order:

1. **CorrelationID** — propagates trace IDs through message metadata
2. **Logging** — structured per-message latency and outcome via zap
3. **PoisonQueue** — dead-letters permanently failed messages to `cfg.PoisonQueueTopic`
4. **Retry** — exponential backoff up to `cfg.Retry.MaxRetries`
5. **Recoverer** — converts handler panics to errors (feeds into Retry)

## Config

```go
type Config struct {
    Brokers          []string    // required, min 1
    ConsumerGroup    string      // required
    PoisonQueueTopic string      // defaults to "<consumer_group>.failed"
    SASL             SASLConfig
    TLS              TLSConfig
    Retry            RetryConfig
}

type SASLConfig struct {
    Enable    bool
    Mechanism string // "PLAIN" | "SCRAM-SHA-256" | "SCRAM-SHA-512"
    Username  string
    Password  string
}

type TLSConfig struct {
    Enable             bool
    InsecureSkipVerify bool
    CACert             string // path to CA PEM file
    ClientCert         string // path to client cert PEM file
    ClientKey          string // path to client key PEM file
}

type RetryConfig struct {
    MaxRetries        int
    InitialIntervalMs int
    Multiplier        float64
}
```

## Defaults

| Field | Default |
|-------|---------|
| `Retry.MaxRetries` | `3` |
| `Retry.InitialIntervalMs` | `100` |
| `Retry.Multiplier` | `2.0` |
| `PoisonQueueTopic` | `"<consumer_group>.failed"` |

## Errors

| Sentinel | Meaning |
|----------|---------|
| `watermill.ErrInvalidConfig` | Config failed validation |
| `watermill.ErrPublisher` | Failed to create Kafka publisher |
| `watermill.ErrSubscriber` | Failed to create Kafka subscriber |
| `watermill.ErrRouter` | Failed to create router |
| `watermill.ErrPublish` | Publish call failed |
| `watermill.ErrMarshal` | JSON marshal of outgoing payload failed |
| `watermill.ErrUnmarshal` | JSON unmarshal of incoming payload failed |
