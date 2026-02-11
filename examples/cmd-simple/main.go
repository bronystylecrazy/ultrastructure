package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	uscmd "github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

type Pinger interface {
	Ping(ctx context.Context, url string) error
}

type HTTPPinger struct {
	client *http.Client
}

func NewHTTPPinger() Pinger {
	return &HTTPPinger{
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (p *HTTPPinger) Ping(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

type PingCommand struct {
	pinger Pinger
}

func NewPingCommand(pinger Pinger) *PingCommand {
	return &PingCommand{pinger: pinger}
}

func (p *PingCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ping [url]",
		Short:         "Ping a URL",
		SilenceErrors: true,
		RunE:          p.Run,
	}
	cmd.Flags().String("url", "https://example.com", "URL to ping")
	return cmd
}

func (p *PingCommand) Run(cmd *cobra.Command, args []string) error {
	url, err := cmd.Flags().GetString("url")
	if err != nil {
		return err
	}
	if len(args) > 0 {
		url = args[0]
	}
	return p.pinger.Ping(cmd.Context(), url)
}

type UserListCommand struct{}

func NewUserListCommand() *UserListCommand {
	return &UserListCommand{}
}

func (u *UserListCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:           "user list",
		Short:         "List users (nested command example)",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("listing users...")
			return nil
		},
	}
}

func main() {
	var root *uscmd.Root

	app := fx.New(
		di.App(
			uscmd.Module(
				uscmd.WithDefaultName("ping"),
				uscmd.Use("ping",
					di.Provide(NewHTTPPinger),
					di.Provide(NewPingCommand),
				),
				uscmd.Use("user list",
					di.Provide(NewUserListCommand),
				),
			),
			di.Populate(&root),
		).Build(),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		panic(err)
	}
	defer func() {
		_ = app.Stop(ctx)
	}()

	if err := root.Start(ctx); err != nil {
		fmt.Println(err)
	}
}
