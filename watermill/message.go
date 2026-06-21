package watermill

import (
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// Publish marshals v as JSON and publishes to topic.
// A correlation ID is set on the message if not already present in ctx.
func Publish[T any](ctx context.Context, pub message.Publisher, topic string, v T) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return Domain.Mark(err, ErrMarshal) //nolint:wrapcheck // Domain.Mark/New is the wrapping layer
	}
	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.SetContext(ctx)
	if middleware.MessageCorrelationID(msg) == "" {
		middleware.SetCorrelationID(watermill.NewUUID(), msg)
	}
	if err := pub.Publish(topic, msg); err != nil {
		return Domain.Mark(err, ErrPublish) //nolint:wrapcheck // Domain.Mark/New is the wrapping layer
	}
	return nil
}

// Handle returns a watermill HandlerFunc that unmarshals the message payload
// into T and calls fn. Nack is sent automatically on unmarshal failure;
// fn errors propagate to the router middleware (retry / poison queue).
func Handle[T any](fn func(context.Context, T) error) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		var v T
		if err := json.Unmarshal(msg.Payload, &v); err != nil {
			return nil, Domain.Mark(err, ErrUnmarshal)
		}
		return nil, fn(msg.Context(), v)
	}
}
