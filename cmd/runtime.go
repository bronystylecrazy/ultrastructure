package cmd

import (
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/meta"
	xservice "github.com/bronystylecrazy/ultrastructure/service"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const PreRunnersGroupName = "us/cmd/pre_runners"
const PostRunnersGroupName = "us/cmd/post_runners"

type PreRunner interface {
	PreRun(cmd *cobra.Command, args []string) (bool, error)
}

type PostRunner interface {
	PostRun(cmd *cobra.Command, args []string, runErr error) error
}

type serviceRuntimeHook struct {
	shutdowner fx.Shutdowner
	log        *zap.Logger
}

func NewServiceRuntimeHook(shutdowner fx.Shutdowner, log *zap.Logger) PreRunner {
	return &serviceRuntimeHook{
		shutdowner: shutdowner,
		log:        log,
	}
}

func (h *serviceRuntimeHook) PreRun(cmd *cobra.Command, args []string) (bool, error) {
	if cmd == nil {
		return false, nil
	}
	handled, err := xservice.MaybeRunWindowsService(meta.Name, h.log)
	if err != nil {
		return false, err
	}
	if !handled {
		return false, nil
	}
	if err := h.shutdowner.Shutdown(); err != nil {
		return true, fmt.Errorf("shutdown after service runtime: %w", err)
	}
	return true, nil
}
