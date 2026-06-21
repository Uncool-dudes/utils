package watermill

import (
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

// NewPublisher creates a Kafka publisher. The returned message.Publisher is safe
// for concurrent use and should be shared across the application.
func NewPublisher(cfg Config, log *zap.Logger) (message.Publisher, error) {
	saramaCfg, err := buildSaramaConfig(cfg)
	if err != nil {
		return nil, err
	}
	pub, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:               cfg.Brokers,
			Marshaler:             kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaCfg,
		},
		newLogger(log),
	)
	if err != nil {
		return nil, Domain.Mark(err, ErrPublisher)
	}
	return pub, nil
}
