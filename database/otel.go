package database

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func UseOtel() di.Node {
	return di.Invoke(func(g *GormOtel, log *zap.Logger) {
		log.Debug("initialized gorm opentelemetry!")
	})
}

type GormOtel struct {
	dbConfig   Config
	otelConfig otel.Config

	db     *gorm.DB
	logger *zap.Logger
	tp     *otel.TracerProvider
}

func NewGormOtel(dbConfig Config, otelConfig otel.Config, db *gorm.DB, logger *zap.Logger, tp *otel.TracerProvider) *GormOtel {
	o := &GormOtel{
		dbConfig:   dbConfig,
		otelConfig: otelConfig,
		db:         db,
		logger:     logger,
		tp:         tp,
	}
	o.append()
	return o
}

func (o *GormOtel) append() error {
	if isDbOtelEnabled(o.otelConfig.Enabled, o.dbConfig.Log.Enabled) {
		logger := NewGormLogger(o.logger)
		logger.Context = otel.ContextFunc
		logger.SetAsDefault()
		logger.LogLevel = parseGormLogLevel(o.dbConfig.Log.LogLevel)
		logger.SlowThreshold = o.dbConfig.Log.SlowThreshold
		logger.SkipCallerLookup = o.dbConfig.Log.SkipCallerLookup
		logger.IgnoreRecordNotFoundError = o.dbConfig.Log.IgnoreRecordNotFoundError
		logger.ParameterizedQueries = o.dbConfig.Log.ParameterizedQueries
		o.db.Logger = logger
	}

	if !isDbOtelEnabled(o.otelConfig.Enabled, o.dbConfig.Trace.Enabled) {
		return nil
	}

	opts := []otelgorm.Option{
		otelgorm.WithTracerProvider(o.tp),
	}
	if o.dbConfig.Trace.DBName != "" {
		opts = append(opts, otelgorm.WithDBName(o.dbConfig.Trace.DBName))
	}
	if len(o.dbConfig.Trace.Attributes) > 0 {
		attrs := make([]attribute.KeyValue, 0, len(o.dbConfig.Trace.Attributes))
		for k, v := range o.dbConfig.Trace.Attributes {
			attrs = append(attrs, attribute.String(k, v))
		}
		opts = append(opts, otelgorm.WithAttributes(attrs...))
	}
	if o.dbConfig.Trace.WithoutQueryVariables {
		opts = append(opts, otelgorm.WithoutQueryVariables())
	}
	if o.dbConfig.Trace.WithoutMetrics {
		opts = append(opts, otelgorm.WithoutMetrics())
	}

	if o.dbConfig.Trace.IncludeDryRunSpans {
		opts = append(opts, otelgorm.WithDryRunTx())
	}

	return o.db.Use(
		otelgorm.NewPlugin(opts...),
	)
}

func isDbOtelEnabled(global bool, local *bool) bool {
	if !global {
		return false
	}
	if local == nil {
		return true
	}
	return *local
}
