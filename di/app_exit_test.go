package di

import (
	"errors"
	"testing"

	"go.uber.org/fx"
)

func TestRunReturnsExitCode(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		wantCode int
	}{
		{name: "zero returns nil", code: 0},
		{name: "non-zero returns ExitError", code: 3, wantCode: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run(Invoke(func(s fx.Shutdowner) error {
				return s.Shutdown(fx.ExitCode(tt.code))
			}))

			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("Run: got %v want nil", err)
				}
				return
			}
			var exit *ExitError
			if !errors.As(err, &exit) || exit.Code != tt.wantCode {
				t.Fatalf("Run: got %v want exit status %d", err, tt.wantCode)
			}
		})
	}
}
