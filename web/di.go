package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/zap"
)

func UseSwagger() di.Node {
	return di.Options(
		di.Provide(NewSwaggerHandler),
		di.Invoke(func(log *zap.Logger) {
			log.Debug("use swagger")
		}),
	)
}
