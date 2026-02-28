package goose

import (
	"context"
	"time"

	pgoose "github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

type MigrateStatusCommand struct {
	runtime *MigrateRuntime
}

func NewMigrateStatusCommand(runtime *MigrateRuntime) *MigrateStatusCommand {
	return &MigrateStatusCommand{runtime: runtime}
}

func (c *MigrateStatusCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:           "status",
		Short:         "Show migration status",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: c.runtime.runWithShutdown(func(cmd *cobra.Command, args []string) error {
			path, err := c.runtime.prepare()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			return pgoose.StatusContext(ctx, c.runtime.db, path)
		}),
	}
}
