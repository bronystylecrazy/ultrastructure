package web

import (
	"github.com/spf13/cobra"
)

type ServeCommand struct {
	server Server
}

func NewServeCommand(server Server) *ServeCommand {
	return &ServeCommand{
		server: server,
	}
}

func (c *ServeCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:  "serve",
		RunE: c.RunE,
	}
}

func (c *ServeCommand) RunE(cmd *cobra.Command, args []string) error {
	// Block until the server stops so the app keeps running while serving.
	<-c.server.Done()
	return nil
}
