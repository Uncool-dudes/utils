package featureflags

import (
	"context"

	of "github.com/open-feature/go-sdk/openfeature"
)

// EvalContext carries optional targeting information for flag evaluation.
type EvalContext struct {
	TargetingKey string
	Attrs        map[string]any
}

// Service wraps an OpenFeature client and manages provider lifecycle.
type Service struct {
	provider of.FeatureProvider
	client   *of.Client
}

// New returns an uninitiated Service. Provider initialization is deferred to OnStart via fx.Module.
func New(provider of.FeatureProvider) *Service {
	return &Service{provider: provider}
}

func (s *Service) connect(ctx context.Context) error {
	if err := of.SetProviderWithContextAndWait(ctx, s.provider); err != nil {
		return Domain.Mark(err, ErrProviderFailed) //nolint:wrapcheck // Domain.Mark is the wrapping layer
	}
	s.client = of.NewDefaultClient()
	return nil
}

// Close shuts down the OpenFeature provider.
func (s *Service) Close(ctx context.Context) error {
	return of.ShutdownWithContext(ctx) //nolint:wrapcheck // openfeature shutdown, no domain wrapping needed
}

// Bool evaluates a boolean feature flag, returning defaultVal on provider error.
func (s *Service) Bool(ctx context.Context, flag string, defaultVal bool, evalCtx ...EvalContext) (bool, error) {
	val, err := s.client.BooleanValue(ctx, flag, defaultVal, toOFContext(evalCtx...))
	if err != nil {
		return defaultVal, Domain.Wrap(err, "evaluate bool flag")
	}
	return val, nil
}

// String evaluates a string feature flag, returning defaultVal on provider error.
func (s *Service) String(ctx context.Context, flag, defaultVal string, evalCtx ...EvalContext) (string, error) {
	val, err := s.client.StringValue(ctx, flag, defaultVal, toOFContext(evalCtx...))
	if err != nil {
		return defaultVal, Domain.Wrap(err, "evaluate string flag")
	}
	return val, nil
}

// Int evaluates an integer feature flag, returning defaultVal on provider error.
func (s *Service) Int(ctx context.Context, flag string, defaultVal int64, evalCtx ...EvalContext) (int64, error) {
	val, err := s.client.IntValue(ctx, flag, defaultVal, toOFContext(evalCtx...))
	if err != nil {
		return defaultVal, Domain.Wrap(err, "evaluate int flag")
	}
	return val, nil
}

// Float evaluates a float feature flag, returning defaultVal on provider error.
func (s *Service) Float(ctx context.Context, flag string, defaultVal float64, evalCtx ...EvalContext) (float64, error) {
	val, err := s.client.FloatValue(ctx, flag, defaultVal, toOFContext(evalCtx...))
	if err != nil {
		return defaultVal, Domain.Wrap(err, "evaluate float flag")
	}
	return val, nil
}

// NewConnected creates and immediately initializes a Service. Use in tests and CLIs.
func NewConnected(ctx context.Context, provider of.FeatureProvider) (*Service, error) {
	svc := New(provider)
	if err := svc.connect(ctx); err != nil {
		return nil, err
	}
	return svc, nil
}

func toOFContext(evalCtx ...EvalContext) of.EvaluationContext {
	if len(evalCtx) == 0 {
		return of.EvaluationContext{}
	}
	return of.NewEvaluationContext(evalCtx[0].TargetingKey, evalCtx[0].Attrs)
}
