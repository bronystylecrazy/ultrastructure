package xgorm

import (
	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type GormOtel struct {
	dbConfig    database.Config
	logConfig   LogConfig
	traceConfig TraceConfig
	otelConfig  otel.Config

	db     *gorm.DB
	logger *zap.Logger
	tp     *otel.TracerProvider
}

func NewGormOtel(dbConfig database.Config, logConfig LogConfig, traceConfig TraceConfig, otelConfig otel.Config, db *gorm.DB, logger *zap.Logger, tp *otel.TracerProvider) *GormOtel {
	o := &GormOtel{
		dbConfig:    dbConfig,
		logConfig:   logConfig,
		traceConfig: traceConfig,
		otelConfig:  otelConfig,
		db:          db,
		logger:      logger,
		tp:          tp,
	}
	o.append()
	return o
}

func (o *GormOtel) append() error {
	if isDbOtelEnabled(o.otelConfig.Enabled, o.logConfig.Enabled) {
		logger := NewGormLogger(o.logger)
		logger.Context = otel.ContextFunc
		logger.SetAsDefault()
		logger.LogLevel = parseGormLogLevel(o.logConfig.LogLevel)
		logger.SlowThreshold = o.logConfig.SlowThreshold
		logger.SkipCallerLookup = o.logConfig.SkipCallerLookup
		logger.IgnoreRecordNotFoundError = o.logConfig.IgnoreRecordNotFoundError
		logger.ParameterizedQueries = o.logConfig.ParameterizedQueries
		o.db.Logger = logger
	}

	if !isDbOtelEnabled(o.otelConfig.Enabled, o.traceConfig.Enabled) {
		return nil
	}

	opts := []otelgorm.Option{
		otelgorm.WithTracerProvider(o.tp),
	}
	if o.traceConfig.DBName != "" {
		opts = append(opts, otelgorm.WithDBName(o.traceConfig.DBName))
	}
	if len(o.traceConfig.Attributes) > 0 {
		attrs := make([]attribute.KeyValue, 0, len(o.traceConfig.Attributes))
		for k, v := range o.traceConfig.Attributes {
			attrs = append(attrs, attribute.String(k, v))
		}
		opts = append(opts, otelgorm.WithAttributes(attrs...))
	}
	if o.traceConfig.WithoutQueryVariables {
		opts = append(opts, otelgorm.WithoutQueryVariables())
	}
	if o.traceConfig.WithoutMetrics {
		opts = append(opts, otelgorm.WithoutMetrics())
	}

	if o.traceConfig.IncludeDryRunSpans {
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
