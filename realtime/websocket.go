package realtime

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync"

	"log/slog"

	"github.com/bronystylecrazy/ultrastructure/web"
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
	Id          string       // the internal id of the listener
	App         fiber.Router // the fiber router to register the websocket handler on
	Authorizer  Authorizer
	Path        string
	Log         *slog.Logger          // server logging
	EstablishFn listeners.EstablishFn // the server's establish connection handler
	Upgrader    *websocket.Upgrader   //  upgrade the incoming http/tcp connection to a websocket compliant connection.
}

// NewWebsocket initializes and returns a new Websocket listener, listening on an address.
func NewWebsocket(app fiber.Router, authorizer Authorizer) *Websocket {
	return &Websocket{
		Id:         "ws1",
		App:        app,
		Authorizer: authorizer,
		Path:       "/realtime",
		Upgrader: &websocket.Upgrader{
			Subprotocols: []string{"mqtt"},
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// Option customizes a Websocket instance.
type Option func(*Websocket)

// NewWebsocketWithOptions initializes a Websocket with defaults and applies options.
func NewWebsocketWithOptions(opts ...Option) *Websocket {
	ws := NewWebsocket(nil, nil)
	for _, opt := range opts {
		if opt != nil {
			opt(ws)
		}
	}
	return ws
}

// WithId sets the Websocket listener id.
func WithId(id string) Option {
	return func(w *Websocket) {
		w.Id = id
	}
}

// WithApp sets the Fiber router for registering the websocket handler.
func WithApp(app fiber.Router) Option {
	return func(w *Websocket) {
		w.App = app
	}
}

// WithPath sets the route path for the websocket handler.
func WithPath(path string) Option {
	return func(w *Websocket) {
		w.Path = path
	}
}

// WithAuthorizer sets the authorizer for the websocket handler.
func WithAuthorizer(authorizer Authorizer) Option {
	return func(w *Websocket) {
		w.Authorizer = authorizer
	}
}

// WithUpgrader sets the websocket upgrader.
func WithUpgrader(upgrader *websocket.Upgrader) Option {
	return func(w *Websocket) {
		w.Upgrader = upgrader
	}
}

// ID returns the id of the listener.
func (l *Websocket) ID() string {
	return l.Id
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
	l.Log = log
	return nil
}

func (l *Websocket) Handle(r web.Router) {
	if l.Authorizer == nil {
		r.All(l.Path, adaptor.HTTPHandlerFunc(l.Handler))
	} else {
		r.All(l.Path, l.Authorizer.Authorize(), adaptor.HTTPHandlerFunc(l.Handler))
	}
}

// Handler upgrades and handles an incoming websocket connection.
func (l *Websocket) Handler(w http.ResponseWriter, r *http.Request) {
	c, err := l.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	err = l.EstablishFn(l.Id, &wsConn{Conn: c.UnderlyingConn(), c: c})
	if err != nil {
		l.Log.Warn("", "error", err)
	}
}

// Serve starts waiting for new Websocket connections, and calls the connection
// establishment callback for any received.
func (l *Websocket) Serve(establish listeners.EstablishFn) {
	l.EstablishFn = establish
}

// Close closes the listener and any client connections.
func (l *Websocket) Close(closeClients listeners.CloseFn) {
	l.Lock()
	defer l.Unlock()

	closeClients(l.Id)
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
