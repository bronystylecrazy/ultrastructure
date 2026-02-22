package spa

import (
	"embed"
	"math"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web"
	"go.uber.org/zap"
)

func Use(assets *embed.FS, opts ...Option) di.Node {

	middleware := func(log *zap.Logger) (*Middleware, error) {
		base := []Option{WithAssets(assets), WithLogger(log)}
		return NewMiddleware(append(base, opts...)...)
	}

	return di.Provide(
		middleware,
		di.Params(di.Optional()),
		web.Priority(math.MaxInt32),
	)
}
