package goose

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"time"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

var defaultPath = "migrations"

type Goose struct {
	FS    *embed.FS
	Paths []string

	config database.Config
	db     *sql.DB
	logger *zap.Logger
}

func NewGoose(
	fs *embed.FS,
	config database.Config,
	db *sql.DB,
	logger *zap.Logger,
	paths ...string,
) *Goose {
	return &Goose{
		FS:     fs,
		config: config,
		db:     db,
		logger: logger,
		Paths:  paths,
	}
}

func (m *Goose) getLogger() *zap.Logger {
	if m.logger != nil {
		return m.logger
	}
	return zap.L()
}

func (m *Goose) Run() error {
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

	if err := goose.SetDialect(database.ParseDialect(m.config.Driver)); err != nil {
		logger.Error("Failed to set dialect", zap.Error(err))
		return err
	}

	goose.SetLogger(NewGooseZapLogger(logger))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = goose.UpContext(ctx, m.db, path)
	if err != nil {
		logger.Error("Failed to run migrations", zap.Error(err))
		return err
	}

	return nil
}
