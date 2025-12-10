package flexinfra

import (
	"github.com/bronystylecrazy/flexinfra/logging"
	"github.com/bronystylecrazy/flexinfra/realtime"
	"github.com/bronystylecrazy/flexinfra/storage"
	"github.com/bronystylecrazy/flexinfra/web"
	"go.uber.org/fx"
)

var DefaultWires = fx.Options(
	fx.Provide(
		logging.NewDefaultLogger,
		logging.NewEventLogger,
	),
	fx.Provide(
		realtime.NewAuthorizer,
		realtime.NewMqttServer,
	),
	fx.Provide(
		// deps
		web.NewFiberApp,
		web.NewAuthorizer,
		web.NewValidator,
		// handlers
		web.NewEtagHandler,
		web.NewHealthHandler,
		web.NewLoggerHandler,
		web.NewMonitorHandler,
		web.NewStaticHandler,
		web.NewSwaggerHandler,
		web.NewLimitterHandler,
	),
	fx.Provide(
		storage.NewMinioStorage,
	),
)
