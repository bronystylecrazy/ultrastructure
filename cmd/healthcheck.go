package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

const defaultHealthcheckURL = "http://127.0.0.1:3000/healthz"

type HealthcheckCommand struct {
	shutdowner fx.Shutdowner
}

func NewHealthcheckCommand(shutdowner fx.Shutdowner) *HealthcheckCommand {
	return &HealthcheckCommand{
		shutdowner: shutdowner,
	}
}

func (s *HealthcheckCommand) Command() *cobra.Command {
	c := &cobra.Command{
		Use:           "healthcheck",
		Short:         "Check HTTP health endpoint and exit non-zero when unhealthy",
		SilenceErrors: true,
		RunE:          s.Run,
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return s.shutdowner.Shutdown()
		},
	}
	c.Flags().String("url", defaultHealthcheckURL, "health endpoint URL")
	c.Flags().Duration("timeout", 3*time.Second, "request timeout")
	return c
}

func (s *HealthcheckCommand) Run(cmd *cobra.Command, args []string) error {
	url, err := cmd.Flags().GetString("url")
	if err != nil {
		return err
	}
	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil {
		return err
	}
	if timeout <= 0 {
		return fmt.Errorf("timeout must be > 0")
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok")
	return err
}

