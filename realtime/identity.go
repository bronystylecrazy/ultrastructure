package realtime

import (
	"context"
	"sync"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type ClientIdentity struct {
	ClientID string
	Username string
}

type ClientIdentityStore struct {
	mu   sync.RWMutex
	data map[string]ClientIdentity
}

func NewClientIdentityStore() *ClientIdentityStore {
	return &ClientIdentityStore{
		data: make(map[string]ClientIdentity),
	}
}

func (s *ClientIdentityStore) Set(clientID string, identity ClientIdentity) {
	s.mu.Lock()
	s.data[clientID] = identity
	s.mu.Unlock()
}

func (s *ClientIdentityStore) Get(clientID string) (ClientIdentity, bool) {
	s.mu.RLock()
	identity, ok := s.data[clientID]
	s.mu.RUnlock()
	if !ok {
		return ClientIdentity{}, false
	}
	return identity, true
}

func (s *ClientIdentityStore) Delete(clientID string) {
	s.mu.Lock()
	delete(s.data, clientID)
	s.mu.Unlock()
}

type clientIdentityContextKey struct{}

func WithClientIdentity(ctx context.Context, identity ClientIdentity) context.Context {
	return context.WithValue(ctx, clientIdentityContextKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (ClientIdentity, bool) {
	identity, ok := ctx.Value(clientIdentityContextKey{}).(ClientIdentity)
	if !ok {
		return ClientIdentity{}, false
	}
	return identity, true
}

type ClientIdentityHook struct {
	mqtt.HookBase
	store *ClientIdentityStore
}

func NewClientIdentityHook(store *ClientIdentityStore) *ClientIdentityHook {
	return &ClientIdentityHook{store: store}
}

func (h *ClientIdentityHook) ID() string {
	return "us-client-identity-hook"
}

func (h *ClientIdentityHook) Provides(b byte) bool {
	return b == mqtt.OnConnect || b == mqtt.OnDisconnect
}

func (h *ClientIdentityHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	h.store.Set(cl.ID, ClientIdentity{
		ClientID: cl.ID,
		Username: string(pk.Connect.Username),
	})
	return nil
}

func (h *ClientIdentityHook) OnDisconnect(cl *mqtt.Client, _ error, _ bool) {
	h.store.Delete(cl.ID)
}

func UseClientIdentityContext(store *ClientIdentityStore) TopicMiddleware {
	return func(next TopicHandler) TopicHandler {
		return func(ctx Ctx) error {
			if cl := ctx.Client(); cl != nil {
				if identity, ok := store.Get(cl.ID); ok {
					ctx.SetContext(WithClientIdentity(ctx.Context(), identity))
				}
			}
			return next(ctx)
		}
	}
}
