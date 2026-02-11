package cmd

import (
	"context"
	"fmt"
	"strings"

	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/spf13/cobra"
)

type Root struct {
	*cobra.Command
}

func New(cmd *cobra.Command) *Root {
	defaultCmd := &cobra.Command{
		Use: us.Name,
	}

	if cmd != nil {
		defaultCmd = cmd
	}

	return &Root{
		Command: defaultCmd,
	}
}

func (r *Root) Start(ctx context.Context) error {
	return r.Execute()
}

func (r *Root) Register(commands ...Commander) error {
	for _, command := range commands {
		if err := r.RegisterOne(command); err != nil {
			return err
		}
	}
	return nil
}

func (r *Root) RegisterOne(c Commander) error {
	if r == nil || r.Command == nil {
		return fmt.Errorf("root command is nil")
	}
	path, err := commandPath(c)
	if err != nil {
		return err
	}
	cmd := c.Command()
	parts := strings.Fields(path)
	if len(parts) == 0 {
		return fmt.Errorf("command path is empty")
	}
	if cmd.Use == "" {
		cmd.Use = parts[len(parts)-1]
	}
	parent := r.Command
	for _, part := range parts[:len(parts)-1] {
		parent = ensureSubCommand(parent, part)
	}
	parent.AddCommand(cmd)
	return nil
}

func ensureSubCommand(parent *cobra.Command, use string) *cobra.Command {
	name := strings.TrimSpace(use)
	if name == "" {
		return parent
	}
	for _, child := range parent.Commands() {
		if child.Name() == name {
			return child
		}
	}
	child := &cobra.Command{Use: name}
	parent.AddCommand(child)
	return child
}
