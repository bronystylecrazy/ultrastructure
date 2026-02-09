package database

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func UseOtelTraceMetrics(opts ...di.Option) di.Node {
	return di.Invoke(attachOtelTraceMetricsToGorm, opts...)
}

func attachOtelTraceMetricsToGorm(db *gorm.DB, config otel.Config, tp *otel.TracerProvider) error {
	if config.Enabled {
		return db.Use(
			otelgorm.NewPlugin(
				otelgorm.WithTracerProvider(tp),
				otelgorm.WithoutQueryVariables(),
			),
		)
	}
	return nil
}

func UseOtelLogger(opts ...di.Option) di.Node {
	return di.Invoke(attachOtelLoggerToGorm, opts...)
}

func attachOtelLoggerToGorm(db *gorm.DB, config otel.Config, log *zap.Logger) error {
	if config.Enabled {
		logger := NewGormLogger(log)
		logger.Context = otel.ContextFunc
		logger.SetAsDefault()
		logger.IgnoreRecordNotFoundError = true
		db.Logger = logger
	}
	return nil
}
