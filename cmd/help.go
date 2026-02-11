package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

type HelpCommand struct {
	shutdowner fx.Shutdowner
}

func NewHelpCommand(shutdowner fx.Shutdowner) *HelpCommand {
	return &HelpCommand{
		shutdowner: shutdowner,
	}
}

func (s *HelpCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:           "help",
		Short:         "Display help information",
		SilenceErrors: true,
		RunE:          s.Run,
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return s.shutdowner.Shutdown()
		},
	}
}

func (s *HelpCommand) Run(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}
