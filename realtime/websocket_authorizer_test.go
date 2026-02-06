package realtime

import (
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gorilla/websocket"
	"log/slog"
)

type rejectAuthorizer struct {
	called *atomic.Bool
}

func (r rejectAuthorizer) Authorize() fiber.Handler {
	return func(c fiber.Ctx) error {
		if r.called != nil {
			r.called.Store(true)
		}
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}
}

func TestWebsocketAuthorizeRejectsBlocksEstablish(t *testing.T) {
	app := fiber.New()

	var authCalled atomic.Bool
	ws := NewWebsocket(app, rejectAuthorizer{called: &authCalled})
	if err := ws.Init(slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("Init: %v", err)
	}

	var establishCalled atomic.Bool
	ws.Serve(func(id string, c net.Conn) error {
		establishCalled.Store(true)
		return nil
	})
	ws.Handle(app)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	go func() {
		_ = app.Listener(ln)
	}()
	defer func() {
		_ = app.Shutdown()
	}()

	url := "ws://" + ln.Addr().String() + "/realtime"
	dialer := websocket.Dialer{Subprotocols: []string{"mqtt"}}

	var resp *http.Response
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, resp, err = dialer.Dial(url, nil)
		if err == nil {
			t.Fatal("expected websocket upgrade to be rejected")
		}
		if resp != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if resp == nil {
		t.Fatalf("expected http response, got error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if !authCalled.Load() {
		t.Fatal("authorizer was not called")
	}
	if establishCalled.Load() {
		t.Fatal("establish called despite authorization failure")
	}
}
