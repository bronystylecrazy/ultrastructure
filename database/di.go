package database

import (
	"context"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Module(opts ...di.Node) di.Node {
	return di.Module(
		"us/database",
		di.Config[Config]("db"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
		di.Provide(NewPostgresDialector),
		di.Provide(NewGormDB),
		di.Invoke(func(cfg Config, db *gorm.DB, log *zap.Logger) error {
			if !cfg.Migrate {
				return nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			log.Debug("checking database connectivity")
			sqlDB, err := db.DB()
			if err != nil {
				log.Error("database init failed", zap.Error(err))
				return err
			}
			if err := sqlDB.PingContext(ctx); err != nil {
				log.Error("database unreachable (check VPN/connection)", zap.Error(err))
				return err
			}
			log.Debug("database connection successful")
			return nil
		}),
		di.Options(di.ConvertAnys(opts)...),
	)
}
