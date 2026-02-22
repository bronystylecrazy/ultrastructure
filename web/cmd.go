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
	<-c.server.Wait()
	return nil
}
