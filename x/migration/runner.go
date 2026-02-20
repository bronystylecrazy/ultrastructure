package migration

import (
	"context"
	"embed"
	"io/fs"
	"time"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var defaultPath = "migrations"

type Migration struct {
	FS    *embed.FS
	Paths []string

	config database.Config
	db     *gorm.DB
	logger *zap.Logger
}

func NewMigration(
	fs *embed.FS,
	config database.Config,
	db *gorm.DB,
	logger *zap.Logger,
	paths ...string,
) *Migration {
	return &Migration{
		FS:     fs,
		config: config,
		db:     db,
		logger: logger,
		Paths:  paths,
	}
}

func (m *Migration) getLogger() *zap.Logger {
	if m.logger != nil {
		return m.logger
	}
	return zap.L()
}

func (m *Migration) Run() error {
	logger := m.getLogger()

	if !m.config.Migrate || m.FS == nil {
		logger.Info("Skipping migrations")
		return nil
	}

	base := fs.FS(m.FS)
	path := defaultPath
	if len(m.Paths) > 0 && m.Paths[0] != "" {
		path = m.Paths[0]
	}

	sub, err := fs.Sub(m.FS, path)
	if err != nil {
		return err
	}

	base = sub
	path = "."
	goose.SetBaseFS(base)

	if err := goose.SetDialect(database.ParseDialect(m.config.Dialect)); err != nil {
		logger.Error("Failed to set dialect", zap.Error(err))
		return err
	}

	sqlDB, err := m.db.DB()
	if err != nil {
		logger.Error("Failed to get database connection", zap.Error(err))
		return err
	}

	goose.SetLogger(NewGooseZapLogger(logger))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = goose.UpContext(ctx, sqlDB, path)
	if err != nil {
		logger.Error("Failed to run migrations", zap.Error(err))
		return err
	}

	return nil
}
