package cmd_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/lc"
	"go.uber.org/fx"
)

// TestHelpInvocationShutsDownApp is a regression test for the lifecycle hang:
// a flags-only `--help` invocation must boot the command root, print help, and
// then shut the fx app down instead of blocking forever. It uses no network
// listeners (no web module) so it is fully in-process and deterministic.
func TestHelpInvocationShutsDownApp(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"app", "--help"}
	t.Cleanup(func() { os.Args = origArgs })

	app := fx.New(
		di.App(
			lc.Providers(),
			cmd.Providers(),
			di.Invoke(cmd.RegisterCommands),
		).Build(),
		fx.NopLogger,
	)

	startCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		t.Fatalf("app.Start: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		_ = app.Stop(stopCtx)
	})

	select {
	case sig := <-app.Wait():
		if sig.ExitCode != 0 {
			t.Fatalf("help invocation exit code: got %d want 0", sig.ExitCode)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("app did not shut down after --help (hung)")
	}
}
