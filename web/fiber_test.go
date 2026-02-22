package web

import (
	"context"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

type testFiberConfigurer struct {
	appName string
}

func (c *testFiberConfigurer) MutateFiberConfig(cfg *fiber.Config) {
	cfg.AppName = c.appName
}

func TestParseBodyLimit(t *testing.T) {
	tests := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{in: "4MB", want: 4_000_000},
		{in: "4MiB", want: 4 * 1024 * 1024},
		{in: "512KB", want: 512_000},
		{in: "512KiB", want: 512 * 1024},
		{in: "1GB", want: 1_000_000_000},
		{in: "4194304", want: 4194304},
		{in: "1.5MB", want: 1_500_000},
		{in: "12XB", wantErr: true},
		{in: "abc", wantErr: true},
	}

	for _, tc := range tests {
		got, err := ParseBodyLimit(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseBodyLimit(%q): expected error, got nil", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseBodyLimit(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseBodyLimit(%q): got=%d want=%d", tc.in, got, tc.want)
		}
	}
}

func TestFiberServerWaitSignalsWhenListening(t *testing.T) {
	server := NewFiberServer(
		Config{
			Server: ServerConfig{
				Host: "127.0.0.1",
				Port: 0,
			},
			Listen: ListenConfig{
				DisableStartupMessage: true,
				ShutdownTimeout:       time.Second,
			},
		},
		FiberConfig{},
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Listen()
	}()

	select {
	case <-server.Wait():
	case <-time.After(5 * time.Second):
		t.Fatal("wait signal was not emitted")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	if err := server.Stop(stopCtx); err != nil {
		t.Fatalf("stop server: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("listen returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("listen did not return after stop")
	}
}
