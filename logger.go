package us

import (
	"github.com/bronystylecrazy/ultrastructure/build"
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const LogConfigName = "log"

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func NewZapLogger(logCfg LogConfig) (*zap.Logger, error) {

	if build.IsDevelopment() {
		cfg := zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		switch logCfg.Level {
		case "debug":
			cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		case "info":
			cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		case "warn":
			cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
		case "error":
			cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
		case "fatal":
			cfg.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
		default:
			cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		}
		return cfg.Build()
	}

	return zap.NewProduction()
}

func NewEventLogger(log *zap.Logger) fxevent.Logger {
	return &fxevent.ZapLogger{Logger: log}
}

func LoggerModule(options ...any) di.Node {
	return di.Options(
		di.Provide(NewZapLogger),
		fx.WithLogger(NewEventLogger),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride()),
		di.Config[LogConfig](LogConfigName,
			di.Switch(
				di.Case(build.IsDevelopment(), di.ConfigDefault("log.level", "debug")),
				di.Case(build.IsProduction(), di.ConfigDefault("log.level", "info")),
				di.DefaultCase(di.ConfigDefault("log.level", "info")),
			),
		),
		di.Options(options...),
	)
}
