package cmd

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// recordingShutdowner captures every Shutdown call (and its options) and
// signals a channel on the first call so tests can await it deterministically.
type recordingShutdowner struct {
	mu     sync.Mutex
	calls  [][]fx.ShutdownOption
	fired  chan struct{}
	closed bool
}

func newRecordingShutdowner() *recordingShutdowner {
	return &recordingShutdowner{fired: make(chan struct{})}
}

func (s *recordingShutdowner) Shutdown(opts ...fx.ShutdownOption) error {
	s.mu.Lock()
	s.calls = append(s.calls, opts)
	if !s.closed {
		s.closed = true
		close(s.fired)
	}
	s.mu.Unlock()
	return nil
}

func (s *recordingShutdowner) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func TestRootStartShutsDownWithExitCode(t *testing.T) {
	tests := []struct {
		name     string
		runErr   error
		wantCode int
	}{
		{name: "success exits zero", runErr: nil, wantCode: 0},
		{name: "error exits one", runErr: errors.New("boom"), wantCode: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shutdowner := newRecordingShutdowner()
			root := New(shutdowner, nil)
			root.SilenceUsage = true
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.AddCommand(&cobra.Command{
				Use: "child",
				RunE: func(*cobra.Command, []string) error {
					return tt.runErr
				},
			})
			root.SetArgs([]string{"child"})

			if err := root.Start(context.Background()); err != nil {
				t.Fatalf("Start returned error: %v", err)
			}

			select {
			case <-shutdowner.fired:
			case <-time.After(5 * time.Second):
				t.Fatal("shutdowner was not called within timeout")
			}

			// Give any erroneous extra shutdown a chance to arrive before asserting once.
			time.Sleep(50 * time.Millisecond)
			if got := shutdowner.callCount(); got != 1 {
				t.Fatalf("shutdown call count: got %d want 1", got)
			}

			shutdowner.mu.Lock()
			opts := shutdowner.calls[0]
			shutdowner.mu.Unlock()
			want := []fx.ShutdownOption{fx.ExitCode(tt.wantCode)}
			if !reflect.DeepEqual(opts, want) {
				t.Fatalf("shutdown options: got %#v want %#v", opts, want)
			}
		})
	}
}

func TestDefaultSubcommandUse(t *testing.T) {
	withDefaultCmd(t, defaultCommandName)

	t.Run("nil root", func(t *testing.T) {
		var r *Root
		if use, ok := r.defaultSubcommandUse(); ok || use != "" {
			t.Fatalf("nil root: got (%q,%v) want (\"\",false)", use, ok)
		}
	})

	t.Run("default child present", func(t *testing.T) {
		root := New(newRecordingShutdowner(), nil)
		root.AddCommand(&cobra.Command{Use: "serve"})
		use, ok := root.defaultSubcommandUse()
		if !ok || use != "serve" {
			t.Fatalf("got (%q,%v) want (\"serve\",true)", use, ok)
		}
	})

	t.Run("default child absent", func(t *testing.T) {
		root := New(newRecordingShutdowner(), nil)
		root.AddCommand(&cobra.Command{Use: "version"})
		if use, ok := root.defaultSubcommandUse(); ok || use != "" {
			t.Fatalf("got (%q,%v) want (\"\",false)", use, ok)
		}
	})

	t.Run("respects WithDefaultName", func(t *testing.T) {
		withDefaultCmd(t, defaultCommandName)
		WithDefaultName("worker")
		root := New(newRecordingShutdowner(), nil)
		root.AddCommand(&cobra.Command{Use: "worker"})
		use, ok := root.defaultSubcommandUse()
		if !ok || use != "worker" {
			t.Fatalf("got (%q,%v) want (\"worker\",true)", use, ok)
		}
	})
}
