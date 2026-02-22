package cmd

import (
	"fmt"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/lc"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

const CommandersGroupName = "us/cmd/commanders"

type registerParams struct {
	fx.In

	Root        *Root
	Commands    []Commander  `group:"us/cmd/commanders"`
	PreRunners  []PreRunner  `group:"us/cmd/pre_runners"`
	PostRunners []PostRunner `group:"us/cmd/post_runners"`
}

// Module wires root command creation and auto-registration for all Commander implementations.
func Module(extends ...di.Node) di.Node {
	return di.Options(
		di.AutoGroup[Commander](CommandersGroupName),
		di.AutoGroup[PreRunner](PreRunnersGroupName),
		di.AutoGroup[PostRunner](PostRunnersGroupName),
		di.Module("us.cmd",
			di.Provide(New, di.Params(di.Optional(), di.Optional()), lc.StartPriority(lc.Latest)),
			di.Options(di.ConvertAnys(extends)...),
		),
	)
}

func RegisterCommands(params registerParams) error {
	if params.Root != nil {
		ensureCommandRunnable(params.Root.Command)
		applyRunners(params.Root.Command, params.PreRunners, params.PostRunners)
	}
	for _, c := range params.Commands {
		if err := registerOne(params.Root, c, params.PreRunners, params.PostRunners); err != nil {
			return err
		}
	}
	return nil
}

func registerOne(root *Root, c Commander, preRunners []PreRunner, postRunners []PostRunner) error {
	if root == nil || root.Command == nil {
		return fmt.Errorf("root command is nil")
	}
	path, err := commandPath(c)
	if err != nil {
		return err
	}
	cmd := c.Command()
	applyRunners(cmd, preRunners, postRunners)
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

func applyRunners(cmd *cobra.Command, preRunners []PreRunner, postRunners []PostRunner) {
	if cmd == nil || (len(preRunners) == 0 && len(postRunners) == 0) {
		return
	}

	origRunE := cmd.RunE
	origRun := cmd.Run
	if origRunE == nil && origRun == nil {
		return
	}
	if origRun != nil {
		cmd.Run = nil
	}

	cmd.RunE = func(c *cobra.Command, args []string) error {
		for _, runner := range preRunners {
			handled, err := runner.PreRun(c, args)
			if err != nil {
				return err
			}
			if handled {
				for _, post := range postRunners {
					if err := post.PostRun(c, args, nil); err != nil {
						return err
					}
				}
				return nil
			}
		}
		var runErr error
		if origRunE != nil {
			runErr = origRunE(c, args)
		} else if origRun != nil {
			origRun(c, args)
		}
		for _, post := range postRunners {
			if err := post.PostRun(c, args, runErr); err != nil {
				if runErr == nil {
					runErr = err
				}
			}
		}
		return runErr
	}
}

func ensureCommandRunnable(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	if cmd.Run != nil || cmd.RunE != nil {
		return
	}
	cmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}
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
