package web

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// Compile-time assertion that the concrete server still satisfies the updated
// Server interface (including the Done method added to fix the serve hang).
var _ Server = (*FiberServer)(nil)

// fakeServer drives ServeCommand.RunE through explicit channel signals.
type fakeServer struct {
	wait chan struct{}
	done chan struct{}
}

func newFakeServer() *fakeServer {
	return &fakeServer{
		wait: make(chan struct{}),
		done: make(chan struct{}),
	}
}

func (s *fakeServer) Listen() error         { return nil }
func (s *fakeServer) Wait() <-chan struct{} { return s.wait }
func (s *fakeServer) Done() <-chan struct{} { return s.done }

func TestServeCommandBlocksUntilDone(t *testing.T) {
	server := newFakeServer()
	cmd := NewServeCommand(server)

	returned := make(chan error, 1)
	go func() {
		returned <- cmd.RunE(&cobra.Command{}, nil)
	}()

	// Server "started" (Wait closes) must NOT unblock RunE; only Done should.
	close(server.wait)
	select {
	case err := <-returned:
		t.Fatalf("RunE returned after start (before Done): err=%v", err)
	case <-time.After(75 * time.Millisecond):
		// expected: still blocked
	}

	close(server.done)
	select {
	case err := <-returned:
		if err != nil {
			t.Fatalf("RunE returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunE did not return after Done closed")
	}
}

func TestFiberServerDoneClosedAfterListen(t *testing.T) {
	// Port far outside the valid uint16 range makes App.Listen fail immediately
	// without binding, so Listen returns fast and must still close Done.
	server := NewFiberServer(
		Config{Server: ServerConfig{Host: "127.0.0.1", Port: 70000}},
		FiberConfig{},
	)

	if err := server.Listen(); err == nil {
		t.Fatal("expected Listen to fail for invalid port")
	}

	select {
	case <-server.Done():
		// expected: Done closed once Listen returned
	default:
		t.Fatal("Done() not closed after Listen returned")
	}
}
