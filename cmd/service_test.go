package cmd

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"go.uber.org/fx"
)

type testServiceController struct {
	calls         []string
	lastFollowArg bool
}

func (s *testServiceController) Install(context.Context) error {
	s.calls = append(s.calls, "install")
	return nil
}

func (s *testServiceController) Uninstall(context.Context) error {
	s.calls = append(s.calls, "uninstall")
	return nil
}

func (s *testServiceController) Start(context.Context) error {
	s.calls = append(s.calls, "start")
	return nil
}

func (s *testServiceController) Stop(context.Context) error {
	s.calls = append(s.calls, "stop")
	return nil
}

func (s *testServiceController) Restart(context.Context) error {
	s.calls = append(s.calls, "restart")
	return nil
}

func (s *testServiceController) Status(ctx context.Context, out io.Writer, follow bool) error {
	s.calls = append(s.calls, "status")
	s.lastFollowArg = follow
	_, _ = out.Write([]byte("status: running\n"))
	return nil
}

type testShutdowner struct {
	count int
}

func (s *testShutdowner) Shutdown(_ ...fx.ShutdownOption) error {
	s.count++
	return nil
}

func TestServiceCommandDispatchesActions(t *testing.T) {
	actions := []string{"install", "uninstall", "start", "stop", "restart", "status"}

	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			controller := &testServiceController{}
			shutdowner := &testShutdowner{}
			cmd := NewServiceCommand(serviceCommandParams{
				Shutdowner: shutdowner,
				Controller: controller,
			}).Command()
			cmd.SetContext(context.Background())
			cmd.SetArgs([]string{action})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute %q: %v", action, err)
			}

			if len(controller.calls) != 1 || controller.calls[0] != action {
				t.Fatalf("unexpected action call: got=%v want=%q", controller.calls, action)
			}
			if action == "status" && !controller.lastFollowArg {
				t.Fatal("expected status to follow logs by default")
			}
			if shutdowner.count != 1 {
				t.Fatalf("unexpected shutdown count: got=%d want=1", shutdowner.count)
			}
		})
	}
}

func TestServiceStatusFollowFlag(t *testing.T) {
	controller := &testServiceController{}
	shutdowner := &testShutdowner{}
	cmd := NewServiceCommand(serviceCommandParams{
		Shutdowner: shutdowner,
		Controller: controller,
	}).Command()
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"status", "--follow=false"})

	var out strings.Builder
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute status: %v", err)
	}
	if controller.lastFollowArg {
		t.Fatal("expected --follow=false to disable follow")
	}
	if !strings.Contains(out.String(), "status: running") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestServiceCommandReturnsErrorWhenControllerMissing(t *testing.T) {
	shutdowner := &testShutdowner{}
	cmd := NewServiceCommand(serviceCommandParams{
		Shutdowner: shutdowner,
	}).Command()
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"install"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrServiceControllerNotConfigured) {
		t.Fatalf("unexpected error: %v", err)
	}
	if shutdowner.count != 0 {
		t.Fatalf("shutdown should not run on command error: got=%d want=0", shutdowner.count)
	}
}

type controllerNoStatus struct{}

func (c *controllerNoStatus) Install(context.Context) error   { return nil }
func (c *controllerNoStatus) Uninstall(context.Context) error { return nil }
func (c *controllerNoStatus) Start(context.Context) error     { return nil }
func (c *controllerNoStatus) Stop(context.Context) error      { return nil }
func (c *controllerNoStatus) Restart(context.Context) error   { return nil }

func TestServiceStatusReturnsUnsupportedWhenMissingStatusController(t *testing.T) {
	shutdowner := &testShutdowner{}
	cmd := NewServiceCommand(serviceCommandParams{
		Shutdowner: shutdowner,
		Controller: &controllerNoStatus{},
	}).Command()
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"status"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrServiceStatusNotSupported) {
		t.Fatalf("unexpected error: %v", err)
	}
	if shutdowner.count != 0 {
		t.Fatalf("shutdown should not run on command error: got=%d want=0", shutdowner.count)
	}
}
