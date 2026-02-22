package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/bronystylecrazy/ultrastructure/meta"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

type Root struct {
	*cobra.Command
	shutdowner fx.Shutdowner
	stopOnce   sync.Once
}

func New(shutdowner fx.Shutdowner, cmd *cobra.Command) *Root {
	defaultCmd := &cobra.Command{
		Use:           meta.Name,
		SilenceErrors: true,
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return shutdowner.Shutdown()
		},
	}

	if cmd != nil {
		defaultCmd = cmd
	}

	return &Root{
		Command:    defaultCmd,
		shutdowner: shutdowner,
	}
}

func (r *Root) Start(ctx context.Context) error {
	go func() {
		if len(os.Args) <= 1 {
			if defaultUse, ok := r.defaultSubcommandUse(); ok {
				r.SetArgs([]string{defaultUse})
			}
		}

		err := r.Execute()
		r.stopOnce.Do(func() {
			if r.shutdowner == nil {
				return
			}
			if err != nil {
				_ = r.shutdowner.Shutdown(fx.ExitCode(1))
			}
		})
	}()
	return nil
}

func (r *Root) defaultSubcommandUse() (string, bool) {
	if r == nil || r.Command == nil {
		return "", false
	}
	defaultNameMu.RLock()
	use := strings.TrimSpace(currentDefaultCmd)
	defaultNameMu.RUnlock()
	if use == "" {
		use = defaultCommandName
	}
	for _, child := range r.Commands() {
		if child.Name() == use {
			return use, true
		}
	}
	return "", false
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
