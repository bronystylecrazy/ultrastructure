package xgorm

import (
	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Option interface {
	apply(*option)
}

type option struct {
	logger *zap.Logger
}

type optionFunc func(*option)

func (f optionFunc) apply(cfg *option) {
	f(cfg)
}

func WithLogger(logger *zap.Logger) Option {
	return optionFunc(func(cfg *option) {
		cfg.logger = logger
	})
}

func Use(opts ...Option) di.Node {
	useOpts := option{}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&useOpts)
		}
	}

	gormOtel := func(dbConfig database.Config, logConfig LogConfig, traceConfig TraceConfig, otelConfig otel.Config, db *gorm.DB, logger *zap.Logger, tp *otel.TracerProvider) *GormOtel {
		otelLogger := useOpts.logger
		if otelLogger == nil {
			otelLogger = logger
		}
		if otelLogger == nil {
			otelLogger = zap.L()
		}
		return NewGormOtel(dbConfig, logConfig, traceConfig, otelConfig, db, otelLogger, tp)
	}

	return di.Options(
		cfg.Config[LogConfig]("db.log", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),
		cfg.Config[TraceConfig]("db.trace", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),
		di.Provide(NewDialector),
		di.Provide(NewDB),
		di.Provide(NewSQLDB),
		di.Provide(NewChecker),
		di.Provide(gormOtel, di.Params(``, ``, ``, ``, ``, di.Optional(), ``)),
	)
}
