package goose

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

type GooseCommand struct {
	shutdowner fx.Shutdowner
}

func NewCommand(shutdowner fx.Shutdowner) *GooseCommand {
	return &GooseCommand{shutdowner: shutdowner}
}

func (g *GooseCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:           "goose",
		Short:         "Database migration commands",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return g.shutdowner.Shutdown()
		},
	}
}
