package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var ErrServiceControllerNotConfigured = errors.New("service controller is not configured")
var ErrServiceStatusNotSupported = errors.New("service status is not supported by controller")

type ServiceController interface {
	Install(ctx context.Context) error
	Uninstall(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
}

type ServiceStatusController interface {
	Status(ctx context.Context, out io.Writer, follow bool) error
}

type serviceCommandParams struct {
	fx.In

	Shutdowner fx.Shutdowner
	Controller ServiceController       `optional:"true"`
	Status     ServiceStatusController `optional:"true"`
}

type ServiceCommand struct {
	shutdowner       fx.Shutdowner
	controller       ServiceController
	statusController ServiceStatusController
}

func NewServiceCommand(in serviceCommandParams) *ServiceCommand {
	statusController := in.Status
	if statusController == nil && in.Controller != nil {
		if c, ok := in.Controller.(ServiceStatusController); ok {
			statusController = c
		}
	}

	return &ServiceCommand{
		shutdowner:       in.Shutdowner,
		controller:       in.Controller,
		statusController: statusController,
	}
}

func (s *ServiceCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "service",
		Short:         "Manage service registration and runtime state",
		SilenceErrors: true,
	}

	cmd.AddCommand(
		s.actionCommand("install", "Install service", func(ctx context.Context) error {
			return s.controller.Install(ctx)
		}),
		s.actionCommand("uninstall", "Uninstall service", func(ctx context.Context) error {
			return s.controller.Uninstall(ctx)
		}),
		s.actionCommand("start", "Start service", func(ctx context.Context) error {
			return s.controller.Start(ctx)
		}),
		s.actionCommand("stop", "Stop service", func(ctx context.Context) error {
			return s.controller.Stop(ctx)
		}),
		s.actionCommand("restart", "Restart service", func(ctx context.Context) error {
			return s.controller.Restart(ctx)
		}),
		s.statusCommand(),
	)

	return cmd
}

func (s *ServiceCommand) statusCommand() *cobra.Command {
	var follow bool
	cmd := &cobra.Command{
		Use:           "status",
		Short:         "Show service status and optionally stream daemon logs",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.statusController != nil {
				return s.statusController.Status(cmd.Context(), cmd.OutOrStdout(), follow)
			}
			if s.controller == nil {
				return fmt.Errorf("%w: provide cmd.ServiceController", ErrServiceControllerNotConfigured)
			}
			return fmt.Errorf("%w: provide cmd.ServiceStatusController", ErrServiceStatusNotSupported)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return s.shutdowner.Shutdown()
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", true, "follow daemon log output")
	return cmd
}

func (s *ServiceCommand) actionCommand(use string, short string, run func(ctx context.Context) error) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.controller == nil {
				return fmt.Errorf("%w: provide cmd.ServiceController", ErrServiceControllerNotConfigured)
			}
			return run(cmd.Context())
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return s.shutdowner.Shutdown()
		},
	}
}
