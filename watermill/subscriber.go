package watermill

import (
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

// NewSubscriber creates a Kafka subscriber bound to cfg.ConsumerGroup.
// All instances of a service sharing the same ConsumerGroup receive each
// message exactly once (Kafka load-balances across the group).
func NewSubscriber(cfg Config, log *zap.Logger) (message.Subscriber, error) {
	saramaCfg, err := buildSaramaConfig(cfg)
	if err != nil {
		return nil, err
	}
	sub, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               cfg.Brokers,
			ConsumerGroup:         cfg.ConsumerGroup,
			Unmarshaler:           kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaCfg,
		},
		newLogger(log),
	)
	if err != nil {
		return nil, Domain.Mark(err, ErrSubscriber)
	}
	return sub, nil
}
