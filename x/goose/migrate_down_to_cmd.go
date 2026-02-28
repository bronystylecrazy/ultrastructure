package goose

import (
	"context"
	"fmt"
	"strconv"
	"time"

	pgoose "github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

type MigrateDownToCommand struct {
	runtime *MigrateRuntime
}

func NewMigrateDownToCommand(runtime *MigrateRuntime) *MigrateDownToCommand {
	return &MigrateDownToCommand{runtime: runtime}
}

func (c *MigrateDownToCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:           "down-to [version]",
		Short:         "Roll back migrations down to the specified version",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: c.runtime.runWithShutdown(func(cmd *cobra.Command, args []string) error {
			version, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid version %q: %w", args[0], err)
			}

			path, err := c.runtime.prepare()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			return pgoose.DownToContext(ctx, c.runtime.db, path, version)
		}),
	}
}
