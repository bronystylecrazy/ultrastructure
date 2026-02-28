package goose

import (
	"context"
	"time"

	pgoose "github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

type MigrateDownCommand struct {
	runtime *MigrateRuntime
}

func NewMigrateDownCommand(runtime *MigrateRuntime) *MigrateDownCommand {
	return &MigrateDownCommand{runtime: runtime}
}

func (c *MigrateDownCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:           "down",
		Short:         "Roll back the most recent migration",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: c.runtime.runWithShutdown(func(cmd *cobra.Command, args []string) error {
			path, err := c.runtime.prepare()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			return pgoose.DownContext(ctx, c.runtime.db, path)
		}),
	}
}
