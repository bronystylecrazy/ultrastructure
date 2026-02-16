package cmd

import (
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/meta"
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
		meta.Name,
		meta.Version,
		meta.Commit,
		meta.BuildDate,
	)
	return err
}
