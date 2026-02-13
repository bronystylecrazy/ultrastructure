package realtime

import (
	"context"
	"errors"
	"time"
)

var ErrTopicHandlerTimeout = errors.New("realtime: topic handler timed out")

type TimeoutTopicMiddlewareOption func(*timeoutTopicMiddlewareConfig)

type timeoutTopicMiddlewareConfig struct {
	disconnectOnTimeout bool
}

func WithTimeoutDisconnect() TimeoutTopicMiddlewareOption {
	return func(cfg *timeoutTopicMiddlewareConfig) {
		cfg.disconnectOnTimeout = true
	}
}

func TimeoutTopicMiddleware(timeout time.Duration, opts ...TimeoutTopicMiddlewareOption) TopicMiddleware {
	cfg := timeoutTopicMiddlewareConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return func(next TopicHandler) TopicHandler {
		return func(ctx Ctx) error {
			if timeout <= 0 {
				return next(ctx)
			}

			timedCtx, cancel := context.WithTimeout(ctx.Context(), timeout)
			defer cancel()
			ctx.SetContext(timedCtx)

			err := next(ctx)

			if errors.Is(timedCtx.Err(), context.DeadlineExceeded) {
				if cfg.disconnectOnTimeout {
					_ = ctx.Reject(ErrTopicHandlerTimeout.Error())
				}
				// Standardize timeout outcome even if handler returned context.DeadlineExceeded.
				if err == nil || errors.Is(err, context.DeadlineExceeded) {
					return ErrTopicHandlerTimeout
				}
			}

			return err
		}
	}
}
