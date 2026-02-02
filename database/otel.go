package database

import (
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.uber.org/zap"
	"gorm.io/gorm"
	zapgorm "moul.io/zapgorm2"
)

func UseOtelTraceMetrics(opts ...di.Option) di.Node {
	return di.Invoke(func(db *gorm.DB, config otel.Config, tp *otel.TracerProvider) error {
		if !config.Disabled {
			if err := db.Use(otelgorm.NewPlugin(otelgorm.WithTracerProvider(tp))); err != nil {
				return fmt.Errorf("gorm otel plugin: %w", err)
			}
		}
		return nil
	}, opts...)
}

func UseOtelLogger(opts ...di.Option) di.Node {
	return di.Invoke(func(db *gorm.DB, config otel.Config, log *zap.Logger) error {
		if !config.Disabled {
			logger := zapgorm.New(log)
			logger.SetAsDefault()
			db.Logger = logger
		}
		return nil
	}, opts...)
}
