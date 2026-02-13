package realtime

import (
	"context"
	"sync"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"go.uber.org/fx"
)

type ConnectContextFactory interface {
	BuildConnectContext(cl *mqtt.Client, pk packets.Packet) context.Context
}

type ClientConnectContextStore struct {
	mu   sync.RWMutex
	data map[string]context.Context
}

func NewClientConnectContextStore() *ClientConnectContextStore {
	return &ClientConnectContextStore{
		data: make(map[string]context.Context),
	}
}

func (s *ClientConnectContextStore) Set(clientID string, ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	s.data[clientID] = ctx
	s.mu.Unlock()
}

func (s *ClientConnectContextStore) Get(clientID string) (context.Context, bool) {
	s.mu.RLock()
	ctx, ok := s.data[clientID]
	s.mu.RUnlock()
	return ctx, ok
}

func (s *ClientConnectContextStore) Delete(clientID string) {
	s.mu.Lock()
	delete(s.data, clientID)
	s.mu.Unlock()
}

type newClientConnectContextHookIn struct {
	fx.In
	Store   *ClientConnectContextStore
	Factory ConnectContextFactory `optional:"true"`
}

type ClientConnectContextHook struct {
	mqtt.HookBase
	store   *ClientConnectContextStore
	factory ConnectContextFactory
}

func NewClientConnectContextHook(in newClientConnectContextHookIn) *ClientConnectContextHook {
	return &ClientConnectContextHook{
		store:   in.Store,
		factory: in.Factory,
	}
}

func (h *ClientConnectContextHook) ID() string {
	return "us-client-connect-context-hook"
}

func (h *ClientConnectContextHook) Provides(b byte) bool {
	return b == mqtt.OnConnect || b == mqtt.OnDisconnect
}

func (h *ClientConnectContextHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	ctx := context.Background()
	if h.factory != nil {
		if built := h.factory.BuildConnectContext(cl, pk); built != nil {
			ctx = built
		}
	}

	h.store.Set(cl.ID, ctx)
	return nil
}

func (h *ClientConnectContextHook) OnDisconnect(cl *mqtt.Client, _ error, _ bool) {
	h.store.Delete(cl.ID)
}

type connectedContextKey struct{}
type claimsKey struct{}

type Claims map[string]any

func WithConnectedContext(ctx context.Context, connected context.Context) context.Context {
	return context.WithValue(ctx, connectedContextKey{}, connected)
}

func ConnectedContext(ctx context.Context) (context.Context, bool) {
	connected, ok := ctx.Value(connectedContextKey{}).(context.Context)
	return connected, ok
}

func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsKey{}).(Claims)
	return claims, ok
}

func UseConnectedContext(store *ClientConnectContextStore) TopicMiddleware {
	return func(next TopicHandler) TopicHandler {
		return func(ctx Ctx) error {
			if cl := ctx.Client(); cl != nil {
				if connectedCtx, ok := store.Get(cl.ID); ok {
					ctx.SetContext(WithConnectedContext(ctx.Context(), connectedCtx))
				}
			}
			return next(ctx)
		}
	}
}
