package cmd

import (
	"fmt"

	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

type VersionCommand struct {
	shutdowner fx.Shutdowner
}

func NewVersionCommand(shutdowner fx.Shutdowner) *VersionCommand {
	return &VersionCommand{
		shutdowner: shutdowner,
	}
}

func (s *VersionCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print build version information",
		SilenceErrors: true,
		RunE:          s.Run,
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return s.shutdowner.Shutdown()
		},
	}
}

func (s *VersionCommand) Run(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	_, err := fmt.Fprintf(
		out,
		"%s\n  Version   %s\n  Commit    %s\n  BuildDate %s\n",
		us.Name,
		us.Version,
		us.Commit,
		us.BuildDate,
	)
	return err
}
