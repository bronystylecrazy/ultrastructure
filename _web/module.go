package web

import (
	"errors"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Features struct {
	Logger  bool
	Health  bool
	Etag    bool
	Monitor bool
	Swagger bool
	Static  bool
	Limiter bool
}

type Option func(*Features)

func defaultFeatures() Features {
	return Features{
		Logger:  true,
		Health:  true,
		Etag:    true,
		Monitor: false,
		Swagger: false,
		Static:  false,
		Limiter: true,
	}
}

func WithDefaults() Option {
	return func(features *Features) {
		*features = defaultFeatures()
	}
}

func WithLogger(enabled bool) Option {
	return func(features *Features) {
		features.Logger = enabled
	}
}

func WithHealth(enabled bool) Option {
	return func(features *Features) {
		features.Health = enabled
	}
}

func WithEtag(enabled bool) Option {
	return func(features *Features) {
		features.Etag = enabled
	}
}

func WithMonitor(enabled bool) Option {
	return func(features *Features) {
		features.Monitor = enabled
	}
}

func WithSwagger(enabled bool) Option {
	return func(features *Features) {
		features.Swagger = enabled
	}
}

func WithStatic(enabled bool) Option {
	return func(features *Features) {
		features.Static = enabled
	}
}

func WithLimiter(enabled bool) Option {
	return func(features *Features) {
		features.Limiter = enabled
	}
}

func Module(options ...Option) fx.Option {
	features := defaultFeatures()
	for _, opt := range options {
		if opt != nil {
			opt(&features)
		}
	}

	return fx.Options(
		fx.Supply(features),
		fx.Provide(
			NewApp,
			fx.Annotate(selectLoggerHandler, fx.ResultTags(`group:"web.handlers"`)),
			fx.Annotate(selectHealthHandler, fx.ResultTags(`group:"web.handlers"`)),
			fx.Annotate(selectEtagHandler, fx.ResultTags(`group:"web.handlers"`)),
			fx.Annotate(selectMonitorHandler, fx.ResultTags(`group:"web.handlers"`)),
			fx.Annotate(selectSwaggerHandler, fx.ResultTags(`group:"web.handlers"`)),
			fx.Annotate(selectStaticHandler, fx.ResultTags(`group:"web.handlers"`)),
			fx.Annotate(selectLimiterHandler, fx.ResultTags(`group:"web.handlers"`)),
		),
		fx.Invoke(
			fx.Annotate(SetupHandlers, fx.ParamTags(`group:"web.handlers"`)),
			registerLifecycle,
		),
	)
}

type loggerDeps struct {
	fx.In
	Logger *zap.Logger `optional:"true"`
}

type staticDeps struct {
	fx.In
	Assets *FS         `optional:"true"`
	Logger *zap.Logger `optional:"true"`
}

type swaggerDeps struct {
	fx.In
	Config *Config `optional:"true"`
}

func selectLoggerHandler(features Features, deps loggerDeps) (Handler, error) {
	if !features.Logger {
		return NopLoggerHandler, nil
	}
	if deps.Logger == nil {
		return nil, errors.New("web logger enabled but zap.Logger not provided")
	}
	return NewLoggerHandler(deps.Logger), nil
}

func selectHealthHandler(features Features) Handler {
	if !features.Health {
		return NopHealthHandler
	}
	return NewHealthHandler()
}

func selectEtagHandler(features Features) Handler {
	if !features.Etag {
		return NopEtagHandler
	}
	return NewEtagHandler()
}

func selectMonitorHandler(features Features) Handler {
	if !features.Monitor {
		return NopMonitorHandler
	}
	return NewMonitorHandler()
}

func selectSwaggerHandler(features Features, deps swaggerDeps) (Handler, error) {
	if !features.Swagger {
		return NopSwaggerHandler, nil
	}
	if deps.Config == nil {
		return nil, errors.New("swagger enabled but web.Config not provided")
	}
	return NewSwaggerHandler(*deps.Config)
}

func selectStaticHandler(features Features, deps staticDeps) (Handler, error) {
	if !features.Static {
		return NopStaticHandler, nil
	}
	if deps.Assets == nil {
		return nil, errors.New("static enabled but web.FS assets not provided")
	}
	if deps.Logger == nil {
		return nil, errors.New("static enabled but zap.Logger not provided")
	}
	return NewStaticHandler(*deps.Assets, deps.Logger), nil
}

func selectLimiterHandler(features Features) Handler {
	if !features.Limiter {
		return NopLimitterHandler
	}
	return NewLimitterHandler()
}
