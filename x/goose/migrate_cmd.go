package goose

import (
	"database/sql"
	"errors"
	"io/fs"

	"github.com/bronystylecrazy/ultrastructure/database"
	pgoose "github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrNoDB = errors.New("goose: no database provided")

type MigrateCommand struct {
	runtime *MigrateRuntime
	status  *MigrateStatusCommand
	up      *MigrateUpCommand
	down    *MigrateDownCommand
	downTo  *MigrateDownToCommand
}

func NewMigrateCommand(runtime *MigrateRuntime, status *MigrateStatusCommand, up *MigrateUpCommand, down *MigrateDownCommand, downTo *MigrateDownToCommand) *MigrateCommand {
	return &MigrateCommand{
		runtime: runtime,
		status:  status,
		up:      up,
		down:    down,
		downTo:  downTo,
	}
}

func (m *MigrateCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "goose migrate",
		Short:         "Run migration commands",
		SilenceErrors: true,
		SilenceUsage:  true,
		PostRunE: func(cmd *cobra.Command, args []string) error {
			if m.runtime == nil || m.runtime.shutdowner == nil {
				return nil
			}
			return m.runtime.shutdowner.Shutdown()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	if m.status != nil {
		cmd.AddCommand(m.status.Command())
	}
	if m.up != nil {
		cmd.AddCommand(m.up.Command())
	}
	if m.down != nil {
		cmd.AddCommand(m.down.Command())
	}
	if m.downTo != nil {
		cmd.AddCommand(m.downTo.Command())
	}

	return cmd
}

type MigrateRuntime struct {
	db         *sql.DB
	goose      *Goose
	shutdowner fx.Shutdowner
	logger     *zap.Logger
}

func NewMigrateRuntime(shutdowner fx.Shutdowner, db *sql.DB, goose *Goose, logger *zap.Logger) *MigrateRuntime {
	return &MigrateRuntime{
		db:         db,
		goose:      goose,
		shutdowner: shutdowner,
		logger:     logger,
	}
}

func (m *MigrateRuntime) runWithShutdown(run func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
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

func (m *MigrateRuntime) log() *zap.Logger {
	if m.logger != nil {
		return m.logger.Named("goose")
	}
	return zap.L().Named("goose")
}

func (m *MigrateRuntime) prepare() (string, error) {
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

func (m *MigrateRuntime) configureGoose() error {
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

func (m *MigrateRuntime) migrationPath() string {
	if m.goose.FS == nil {
		return migrationRootPath(m.goose.Paths)
	}
	return "."
}

func (m *MigrateRuntime) migrationFS() (fs.FS, error) {
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
