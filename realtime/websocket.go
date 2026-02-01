package realtime

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync"

	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gorilla/websocket"
	"github.com/mochi-mqtt/server/v2/listeners"
)

const TypeWS = "ws"

var (
	// ErrInvalidMessage indicates that a message payload was not valid.
	ErrInvalidMessage = errors.New("message type not binary")
)

// Websocket is a listener for establishing websocket connections.
type Websocket struct { // [MQTT-4.2.0-1]
	sync.RWMutex
	id         string       // the internal id of the listener
	app        fiber.Router // the fiber router to register the websocket handler on
	authorizer Authorizer
	log        *slog.Logger          // server logging
	establish  listeners.EstablishFn // the server's establish connection handler
	upgrader   *websocket.Upgrader   //  upgrade the incoming http/tcp connection to a websocket compliant connection.
}

// NewWebsocket initializes and returns a new Websocket listener, listening on an address.
func NewWebsocket(app fiber.Router, authorizer Authorizer) *Websocket {
	return &Websocket{
		id:         "ws1",
		app:        app,
		authorizer: authorizer,
		upgrader: &websocket.Upgrader{
			Subprotocols: []string{"mqtt"},
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// ID returns the id of the listener.
func (l *Websocket) ID() string {
	return l.id
}

// Address returns the address of the listener.
func (l *Websocket) Address() string {
	return ""
}

// Protocol returns the address of the listener.
func (l *Websocket) Protocol() string {
	return "wss"
}

// Init initializes the listener.
func (l *Websocket) Init(log *slog.Logger) error {
	l.log = log
	l.app.All("/realtime", l.authorizer.Authorize(), adaptor.HTTPHandlerFunc(l.handler))
	return nil
}

// handler upgrades and handles an incoming websocket connection.
func (l *Websocket) handler(w http.ResponseWriter, r *http.Request) {
	c, err := l.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	err = l.establish(l.id, &wsConn{Conn: c.UnderlyingConn(), c: c})
	if err != nil {
		l.log.Warn("", "error", err)
	}
}

// Serve starts waiting for new Websocket connections, and calls the connection
// establishment callback for any received.
func (l *Websocket) Serve(establish listeners.EstablishFn) {
	l.establish = establish
}

// Close closes the listener and any client connections.
func (l *Websocket) Close(closeClients listeners.CloseFn) {
	l.Lock()
	defer l.Unlock()

	closeClients(l.id)
}

// wsConn is a websocket connection which satisfies the net.Conn interface.
type wsConn struct {
	net.Conn
	c *websocket.Conn

	// reader for the current message (can be nil)
	r io.Reader
}

// Read reads the next span of bytes from the websocket connection and returns the number of bytes read.
func (ws *wsConn) Read(p []byte) (int, error) {
	if ws.r == nil {
		op, r, err := ws.c.NextReader()
		if err != nil {
			return 0, err
		}

		if op != websocket.BinaryMessage {
			err = ErrInvalidMessage
			return 0, err
		}

		ws.r = r
	}

	var n int
	for {
		// buffer is full, return what we've read so far
		if n == len(p) {
			return n, nil
		}

		br, err := ws.r.Read(p[n:])
		n += br
		if err != nil {
			// when ANY error occurs, we consider this the end of the current message (either because it really is, via
			// io.EOF, or because something bad happened, in which case we want to drop the remainder)
			ws.r = nil

			if errors.Is(err, io.EOF) {
				err = nil
			}
			return n, err
		}
	}
}

// Write writes bytes to the websocket connection.
func (ws *wsConn) Write(p []byte) (int, error) {
	err := ws.c.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// Close signals the underlying websocket conn to close.
func (ws *wsConn) Close() error {
	return ws.Conn.Close()
}
