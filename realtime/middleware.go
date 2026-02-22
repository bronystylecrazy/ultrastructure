package realtime

import (
	"fmt"
	"runtime/debug"

	"go.uber.org/zap"
)

func RecoverTopicMiddleware(log *zap.Logger) TopicMiddleware {
	if log == nil {
		log = zap.NewNop()
	}

	return func(next TopicHandler) TopicHandler {
		return func(ctx Ctx) (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error(
						"panic in mqtt handler",
						zap.Any("panic", rec),
						zap.String("filter", ctx.Filter()),
						zap.String("topic", ctx.Topic()),
						zap.ByteString("payload", ctx.Payload()),
						zap.ByteString("stack", debug.Stack()),
					)
					err = fmt.Errorf("%w: %v", ErrTopicHandlerPanic, rec)
				}
			}()

			return next(ctx)
		}
	}
}
