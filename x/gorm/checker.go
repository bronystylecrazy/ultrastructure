package xgorm

import (
	"context"
	"time"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/gofiber/fiber/v3/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type GormChecker struct {
	otel.Telemetry

	cfg database.Config
	db  *gorm.DB
}

func NewGormChecker(cfg database.Config, db *gorm.DB) *GormChecker {
	return &GormChecker{
		Telemetry: otel.Nop(),

		cfg: cfg,
		db:  db,
	}
}

func (g *GormChecker) Check() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	g.Obs.Debug("checking database connectivity")

	sqlDB, err := g.db.DB()
	if err != nil {
		log.Error("database check failed", zap.Error(err))
		return err
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		log.Error("database unreachable (check VPN/connection)", zap.Error(err))
		return err
	}

	g.Obs.Debug("database connected")

	return nil
}
