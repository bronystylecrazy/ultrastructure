package us

// import (
// 	"github.com/bronystylecrazy/ultrastructure/core/logging"
// 	"github.com/bronystylecrazy/ultrastructure/realtime"
// 	"github.com/bronystylecrazy/ultrastructure/storage"
// 	"github.com/bronystylecrazy/ultrastructure/web"
// 	"go.uber.org/fx"
// )

// var DefaultWires = fx.Options(
// 	fx.Provide(
// 		logging.NewDefaultLogger,
// 		logging.NewEventLogger,
// 	),
// 	fx.Provide(
// 		realtime.NewAuthorizer,
// 		realtime.NewMqttServer,
// 	),
// 	fx.Provide(
// 		// deps
// 		web.NewApp,
// 		web.NewAuthorizer,
// 		web.NewValidator,
// 		// handlers
// 		web.NewEtagHandler,
// 		web.NewHealthHandler,
// 		web.NewLoggerHandler,
// 		web.NewMonitorHandler,
// 		web.NewStaticHandler,
// 		web.NewSwaggerHandler,
// 		web.NewLimitterHandler,
// 	),
// 	fx.Provide(
// 		storage.NewMinioStorage,
// 	),
// )
