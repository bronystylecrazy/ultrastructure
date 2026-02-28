package goose

import (
	"context"
	"time"

	pgoose "github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

type MigrateUpCommand struct {
	runtime *MigrateRuntime
}

func NewMigrateUpCommand(runtime *MigrateRuntime) *MigrateUpCommand {
	return &MigrateUpCommand{runtime: runtime}
}

func (c *MigrateUpCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:           "up",
		Short:         "Apply all pending migrations",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: c.runtime.runWithShutdown(func(cmd *cobra.Command, args []string) error {
			path, err := c.runtime.prepare()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			return pgoose.UpContext(ctx, c.runtime.db, path)
		}),
	}
}
