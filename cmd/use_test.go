package cmd

import "testing"

// withDefaultCmd saves and restores the package-level currentDefaultCmd so
// tests that mutate it (directly or via WithDefaultName) do not leak state.
func withDefaultCmd(t *testing.T, name string) {
	t.Helper()
	defaultNameMu.Lock()
	prev := currentDefaultCmd
	currentDefaultCmd = name
	defaultNameMu.Unlock()
	t.Cleanup(func() {
		defaultNameMu.Lock()
		currentDefaultCmd = prev
		defaultNameMu.Unlock()
	})
}

func TestHasHelpFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "long help", args: []string{"--help"}, want: true},
		{name: "short help", args: []string{"-h"}, want: true},
		{name: "no args", args: nil, want: false},
		{name: "non-help flag", args: []string{"--some-flag"}, want: false},
		{name: "help after dashdash ignored", args: []string{"--", "-h"}, want: false},
		{name: "token then help", args: []string{"serve", "--help"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasHelpFlag(tt.args); got != tt.want {
				t.Fatalf("hasHelpFlag(%v)=%v want=%v", tt.args, got, tt.want)
			}
		})
	}
}

func TestCurrentCommandPathFromArgs(t *testing.T) {
	// Pin the default command name so the "flags-only falls back" cases are
	// deterministic regardless of prior WithDefaultName calls in the suite.
	withDefaultCmd(t, defaultCommandName)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "long help no fallback", args: []string{"--help"}, want: ""},
		{name: "short help no fallback", args: []string{"-h"}, want: ""},
		{name: "no args uses default", args: nil, want: "serve"},
		{name: "non-help flag falls back to default", args: []string{"--some-flag"}, want: "serve"},
		{name: "token wins over help", args: []string{"serve", "--help"}, want: "serve"},
		{name: "help after dashdash ignored uses default", args: []string{"--", "-h"}, want: "serve"},
		{name: "nested command path", args: []string{"user", "list"}, want: "user list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := currentCommandPathFromArgs(tt.args); got != tt.want {
				t.Fatalf("currentCommandPathFromArgs(%v)=%q want=%q", tt.args, got, tt.want)
			}
		})
	}
}

func TestWithDefaultNameChangesFallback(t *testing.T) {
	withDefaultCmd(t, defaultCommandName)

	// Empty/whitespace names reset to the built-in default.
	WithDefaultName("   ")
	if got := currentCommandPathFromArgs(nil); got != defaultCommandName {
		t.Fatalf("blank default name: got %q want %q", got, defaultCommandName)
	}

	WithDefaultName("worker")
	if got := currentCommandPathFromArgs(nil); got != "worker" {
		t.Fatalf("custom default name: got %q want %q", got, "worker")
	}
	// A help-only invocation still refuses to fall back to the custom default.
	if got := currentCommandPathFromArgs([]string{"--help"}); got != "" {
		t.Fatalf("help with custom default: got %q want empty", got)
	}
}
