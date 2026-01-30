package lifecycle

import (
	"context"

	"github.com/bronystylecrazy/ultrastructure/otel"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type StartParams struct {
	fx.In

	Lc       fx.Lifecycle
	Tp       *otel.TracerProvider
	Logger   *zap.Logger
	Starters []Starter `group:"auto-starters"`
}

func RegisterStarters(params StartParams) {
	for _, starter := range params.Starters {
		params.Lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				ctx = otel.WithLogger(ctx, params.Logger)
				ctx = otel.WithTracer(ctx, params.Tp.Tracer("starter"))
				return starter.Start(ctx)
			},
		})
	}
}
