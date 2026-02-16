package cmd

import (
	"os"
	"strings"
	"sync"

	"github.com/bronystylecrazy/ultrastructure/di"
)

const defaultCommandName = "serve"

var (
	defaultNameMu     sync.RWMutex
	currentDefaultCmd = defaultCommandName
)

// Use conditionally includes nodes for the selected CLI command.
// Selection is inferred from process args (defaults to "serve" when no command is provided).
// Nested commands are supported via prefix matching, for example:
// Use("user") matches "user list", and Use("user list") matches "user list active".
func Use(name string, nodes ...di.Node) di.Node {
	selected := currentCommandPathFromArgs(os.Args[1:])
	want := normalizePath(name)
	return di.If(matchesPathPrefix(selected, want), di.Options(di.ConvertAnys(nodes)...))
}

// Serve is simply an alias for Use("serve", ...)
// This is the default command that is run when no command is provided.
func Run(nodes ...di.Node) di.Node {
	return Use("serve", di.Options(di.ConvertAnys(nodes)...))
}

// WithDefaultName configures the default command used when no subcommand is provided.
func WithDefaultName(name string) di.Node {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = defaultCommandName
	}
	defaultNameMu.Lock()
	currentDefaultCmd = trimmed
	defaultNameMu.Unlock()
	return di.Options()
}

func currentCommandPathFromArgs(args []string) string {
	var parts []string
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "-") {
			continue
		}
		parts = append(parts, trimmed)
	}
	if len(parts) > 0 {
		return normalizePath(strings.Join(parts, " "))
	}
	defaultNameMu.RLock()
	name := currentDefaultCmd
	defaultNameMu.RUnlock()
	return normalizePath(name)
}

func matchesPathPrefix(path string, want string) bool {
	if want == "" {
		return false
	}
	if path == want {
		return true
	}
	return strings.HasPrefix(path, want+" ")
}

func normalizePath(path string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(path)), " ")
}
