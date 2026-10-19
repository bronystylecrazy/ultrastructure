package goose

import (
	"errors"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type recordingShutdowner struct {
	calls [][]fx.ShutdownOption
}

func (s *recordingShutdowner) Shutdown(opts ...fx.ShutdownOption) error {
	s.calls = append(s.calls, opts)
	return nil
}

// The first shutdown signal wins, so a failed migration must shut down with
// exit code 1 itself; the root's later ExitCode(1) would be dropped.
func TestRunWithShutdownExitCode(t *testing.T) {
	tests := []struct {
		name     string
		runErr   error
		wantCode int
	}{
		{name: "success exits zero", runErr: nil, wantCode: 0},
		{name: "failure exits one", runErr: errors.New("boom"), wantCode: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shutdowner := &recordingShutdowner{}
			runtime := NewMigrateRuntime(shutdowner, nil, nil, zap.NewNop())
			run := runtime.runWithShutdown(func(*cobra.Command, []string) error {
				return tt.runErr
			})

			if err := run(&cobra.Command{Use: "up"}, nil); !errors.Is(err, tt.runErr) {
				t.Fatalf("run: got %v want %v", err, tt.runErr)
			}
			want := [][]fx.ShutdownOption{{fx.ExitCode(tt.wantCode)}}
			if !reflect.DeepEqual(shutdowner.calls, want) {
				t.Fatalf("shutdown calls: got %#v want %#v", shutdowner.calls, want)
			}
		})
	}
}
