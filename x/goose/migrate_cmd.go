package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"time"

	"github.com/bronystylecrazy/ultrastructure/database"
	pgoose "github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrNoDB = errors.New("goose: no database provided")

type MigrateCommand struct {
	db         *sql.DB
	goose      *Goose
	shutdowner fx.Shutdowner
	logger     *zap.Logger
}

func NewMigrateCommand(shutdowner fx.Shutdowner, db *sql.DB, goose *Goose, logger *zap.Logger) *MigrateCommand {
	return &MigrateCommand{
		db:         db,
		goose:      goose,
		shutdowner: shutdowner,
		logger:     logger,
	}
}

func (m *MigrateCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "goose migrate",
		Short:         "Run migration commands",
		SilenceErrors: true,
		SilenceUsage:  true,
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return m.shutdowner.Shutdown()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		m.statusCommand(),
		m.upCommand(),
		m.downCommand(),
		m.downToCommand(),
	)

	return cmd
}

func (m *MigrateCommand) statusCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "status",
		Short:         "Show migration status",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: m.runWithShutdown(func(cmd *cobra.Command, args []string) error {
			path, err := m.prepare()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			return pgoose.StatusContext(ctx, m.db, path)
		}),
	}
}

func (m *MigrateCommand) upCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "up",
		Short:         "Apply all pending migrations",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: m.runWithShutdown(func(cmd *cobra.Command, args []string) error {
			path, err := m.prepare()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			return pgoose.UpContext(ctx, m.db, path)
		}),
	}
}

func (m *MigrateCommand) downCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "down",
		Short:         "Roll back the most recent migration",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: m.runWithShutdown(func(cmd *cobra.Command, args []string) error {
			path, err := m.prepare()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			return pgoose.DownContext(ctx, m.db, path)
		}),
	}
}

func (m *MigrateCommand) downToCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "down-to [version]",
		Short:         "Roll back migrations down to the specified version",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: m.runWithShutdown(func(cmd *cobra.Command, args []string) error {
			version, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid version %q: %w", args[0], err)
			}

			path, err := m.prepare()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			return pgoose.DownToContext(ctx, m.db, path, version)
		}),
	}
}

func (m *MigrateCommand) runWithShutdown(run func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		runErr := run(cmd, args)
		if runErr != nil {
			m.log().Error("goose migrate command failed", zap.String("command", cmd.CommandPath()), zap.Error(runErr))
		}
		if m.shutdowner == nil {
			return runErr
		}
		shutdownErr := m.shutdowner.Shutdown()
		if shutdownErr != nil {
			m.log().Error("failed to shutdown after goose migrate command", zap.String("command", cmd.CommandPath()), zap.Error(shutdownErr))
		}
		if runErr != nil {
			return runErr
		}
		return shutdownErr
	}
}

func (m *MigrateCommand) log() *zap.Logger {
	if m.logger != nil {
		return m.logger.Named("goose")
	}
	return zap.L().Named("goose")
}

func (m *MigrateCommand) prepare() (string, error) {
	if m.goose == nil {
		return "", ErrNoSource
	}
	if m.db == nil {
		return "", ErrNoDB
	}
	if err := m.configureGoose(); err != nil {
		return "", err
	}
	return m.migrationPath(), nil
}

func (m *MigrateCommand) configureGoose() error {
	if err := pgoose.SetDialect(database.ParseDialect(m.goose.config.Driver)); err != nil {
		return err
	}

	pgoose.SetLogger(NewGooseZapLogger(m.goose.getLogger()))

	baseFS, err := m.migrationFS()
	if err != nil {
		return err
	}
	pgoose.SetBaseFS(baseFS)

	return nil
}

func (m *MigrateCommand) migrationPath() string {
	if m.goose.FS == nil {
		return migrationRootPath(m.goose.Paths)
	}
	return "."
}

func (m *MigrateCommand) migrationFS() (fs.FS, error) {
	if m.goose.FS == nil {
		return nil, nil
	}

	path := migrationRootPath(m.goose.Paths)
	sub, err := fs.Sub(m.goose.FS, path)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func migrationRootPath(paths []string) string {
	if len(paths) > 0 && paths[0] != "" {
		return paths[0]
	}
	return defaultPath
}
