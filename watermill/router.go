package watermill

import (
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/zap"
)

// NewRouter builds a message.Router pre-wired with the full middleware stack:
//
//  1. CorrelationID  — propagates trace IDs through message metadata
//  2. Logging        — structured per-message latency + outcome via zap
//  3. PoisonQueue    — dead-letters permanently failed messages to poisonTopic
//  4. Retry          — exponential backoff up to cfg.Retry.MaxRetries
//  5. Recoverer      — converts handler panics to errors (feeds into Retry)
func NewRouter(pub message.Publisher, cfg Config, log *zap.Logger) (*message.Router, error) {
	wlog := newLogger(log)

	router, err := message.NewRouter(message.RouterConfig{}, wlog)
	if err != nil {
		return nil, Domain.Mark(err, ErrRouter)
	}

	poisonMiddleware, err := middleware.PoisonQueue(pub, cfg.poisonTopic())
	if err != nil {
		return nil, Domain.Wrapf(err, "poison queue middleware for topic %s", cfg.poisonTopic())
	}

	initialInterval := time.Duration(cfg.Retry.InitialIntervalMs) * time.Millisecond
	if initialInterval == 0 {
		initialInterval = time.Duration(Defaults.Retry.InitialIntervalMs) * time.Millisecond
	}
	multiplier := cfg.Retry.Multiplier
	if multiplier == 0 {
		multiplier = Defaults.Retry.Multiplier
	}
	maxRetries := cfg.Retry.MaxRetries
	if maxRetries == 0 {
		maxRetries = Defaults.Retry.MaxRetries
	}

	retryMiddleware := middleware.Retry{
		MaxRetries:      maxRetries,
		InitialInterval: initialInterval,
		Multiplier:      multiplier,
		Logger:          wlog,
	}

	router.AddMiddleware(
		middleware.CorrelationID,
		loggingMiddleware(log),
		poisonMiddleware,
		retryMiddleware.Middleware,
		middleware.Recoverer,
	)

	return router, nil
}

func loggingMiddleware(log *zap.Logger) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			log.Info(
				"message received",
				zap.String("uuid", msg.UUID),
				zap.String("correlation_id", middleware.MessageCorrelationID(msg)),
			)
			out, err := h(msg)
			if err != nil {
				log.Error(
					"message handler failed",
					zap.String("uuid", msg.UUID),
					zap.Error(err),
				)
			}
			return out, err
		}
	}
}
