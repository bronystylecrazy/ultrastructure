package cmd

import (
	"fmt"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

const CommandersGroupName = "us/cmd/commanders"

type registerParams struct {
	fx.In

	Root     *Root
	Commands []Commander `group:"us/cmd/commanders"`
}

// Module wires root command creation and auto-registration for all Commander implementations.
func Module(extends ...di.Node) di.Node {
	return di.Module(
		"us/cmd",
		di.AutoGroup[Commander](CommandersGroupName),
		di.Provide(New, di.Params(di.Optional())),
		di.Invoke(RegisterCommands),
		di.Options(di.ConvertAnys(extends)...),
	)
}

func RegisterCommands(params registerParams) error {
	for _, c := range params.Commands {
		if err := registerOne(params.Root, c); err != nil {
			return err
		}
	}
	return nil
}

func registerOne(root *Root, c Commander) error {
	if root == nil || root.Command == nil {
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
	leafUse := leafUseFromPath(cmd.Use, len(parts))
	if leafUse == "" {
		leafUse = parts[len(parts)-1]
	}
	cmd.Use = leafUse
	parent := root.Command
	for _, part := range parts[:len(parts)-1] {
		parent = ensureSubCommandLocal(parent, part)
	}
	parent.AddCommand(cmd)
	return nil
}

func ensureSubCommandLocal(parent *cobra.Command, use string) *cobra.Command {
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

func commandPath(c Commander) (string, error) {
	if c == nil {
		return "", fmt.Errorf("commander is nil")
	}
	cmd := c.Command()
	if cmd == nil {
		return "", fmt.Errorf("command is nil")
	}
	path := pathFromUse(cmd.Use)
	if path == "" {
		path = strings.TrimSpace(cmd.Name())
	}
	if path == "" {
		return "", fmt.Errorf("command path is empty")
	}
	return path, nil
}

func pathFromUse(use string) string {
	fields := strings.Fields(strings.TrimSpace(use))
	if len(fields) == 0 {
		return ""
	}
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if strings.HasPrefix(f, "[") || strings.HasPrefix(f, "<") {
			break
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
}

func leafUseFromPath(use string, pathParts int) string {
	fields := strings.Fields(strings.TrimSpace(use))
	if len(fields) == 0 || pathParts <= 1 {
		return strings.TrimSpace(use)
	}
	idx := pathParts - 1
	if idx >= len(fields) {
		return ""
	}
	return strings.Join(fields[idx:], " ")
}
