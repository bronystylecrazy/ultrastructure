package database

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func GormCheck(cfg Config, db *gorm.DB, log *zap.Logger) error {
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
}
