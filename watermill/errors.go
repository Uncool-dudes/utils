package watermill

var (
	ErrInvalidConfig = Domain.New("invalid watermill config")
	ErrPublisher     = Domain.New("failed to create publisher")
	ErrSubscriber    = Domain.New("failed to create subscriber")
	ErrRouter        = Domain.New("failed to create router")
	ErrPublish       = Domain.New("publish failed")
	ErrMarshal       = Domain.New("message marshal failed")
	ErrUnmarshal     = Domain.New("message unmarshal failed")
)
