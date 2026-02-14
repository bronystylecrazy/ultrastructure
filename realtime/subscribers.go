package realtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/bronystylecrazy/ultrastructure/otel"
	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"go.uber.org/fx"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var SubscribersGroupName = "mqtt_subscribers"
var TopicMiddlewaresGroupName = "mqtt_topic_middlewares"

var ErrTopicRegistrarStopped = errors.New("realtime: topic registrar is stopped")
var ErrTopicNotAllowed = errors.New("realtime: topic is not allowed by acl")
var ErrInvalidTopicRegistrationArgs = errors.New("realtime: invalid topic registration args")

type subscriptionEntry struct {
	filter string
	id     int
}

type ManagedPubSub struct {
	mu         sync.Mutex
	nextID     int
	stopped    bool
	items      []subscriptionEntry
	mws        []TopicMiddleware
	acl        TopicACLConfig
	sub        usmqtt.Subscriber
	pub        usmqtt.Publisher
	controller usmqtt.SessionController
	log        *zap.Logger
}

type newManagedPubSubIn struct {
	fx.In
	LC       fx.Lifecycle
	Sub      usmqtt.Subscriber
	Pub      usmqtt.Publisher `optional:"true"`
	Broker   usmqtt.Broker    `optional:"true"`
	Attached otel.Attached    `optional:"true"`
	Cfg      *Config          `optional:"true"`
}

func NewManagedPubSub(in newManagedPubSubIn) (*ManagedPubSub, error) {
	log := in.Attached.Logger
	if log == nil {
		log = zap.NewNop()
	}

	var cfg Config
	if in.Cfg != nil {
		cfg = *in.Cfg
	}

	m := &ManagedPubSub{
		nextID: 1,
		sub:    in.Sub,
		pub:    in.Pub,
		acl: TopicACLConfig{
			Enabled:         cfg.TopicACL.Enabled,
			AllowedPrefixes: append([]string(nil), cfg.TopicACL.AllowedPrefixes...),
		},
		log: log,
	}

	controller, err := resolveSessionController(cfg, in.Broker)
	if err != nil {
		return nil, err
	}
	m.controller = controller
	if m.acl.Enabled && len(m.acl.AllowedPrefixes) == 0 {
		log.Warn("mqtt topic acl is enabled with empty allowed_prefixes; all topic registrations will be denied")
	}

	in.LC.Append(fx.Hook{
		OnStop: m.Stop,
	})

	return m, nil
}

func resolveSessionController(cfg Config, broker usmqtt.Broker) (usmqtt.SessionController, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.SessionControl.Provider))
	switch provider {
	case "":
		if sc, ok := any(broker).(usmqtt.SessionController); ok {
			return sc, nil
		}
		return nil, nil
	case "emqx":
		tlsCfg, err := cfg.SessionControl.EMQX.TLS.Load()
		if err != nil {
			return nil, fmt.Errorf("realtime: load emqx session control tls config: %w", err)
		}
		return usmqtt.NewEMQXSessionController(usmqtt.EMQXSessionControllerConfig{
			Endpoint:    cfg.SessionControl.EMQX.Endpoint,
			Username:    cfg.SessionControl.EMQX.Username,
			Password:    cfg.SessionControl.EMQX.Password,
			BearerToken: cfg.SessionControl.EMQX.BearerToken,
			Timeout:     cfg.SessionControl.EMQX.Timeout,
			TLSConfig:   tlsCfg,
		})
	default:
		return nil, fmt.Errorf("realtime: unsupported session_control provider %q", cfg.SessionControl.Provider)
	}
}

func (m *ManagedPubSub) Topic(filter string, args ...any) error {
	handler, middlewares, err := parseTopicArgs(args...)
	if err != nil {
		return err
	}

	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return ErrTopicRegistrarStopped
	}
	if !m.isTopicAllowed(filter) {
		m.mu.Unlock()
		return fmt.Errorf("%w: %q", ErrTopicNotAllowed, filter)
	}

	id := m.nextID
	m.nextID++
	wrapped := applyTopicMiddlewares(handler, append(append([]TopicMiddleware(nil), m.mws...), middlewares...))
	inlineWrapped := topicHandlerToInlineSubFn(wrapped, m.pub, m.controller, func(err error, ctx Ctx) {
		m.log.Error(
			"mqtt topic handler error",
			zap.Error(err),
			zap.String("topic", ctx.Topic()),
			zap.String("filter", ctx.Filter()),
			zap.String("client_id", ctx.ClientID()),
		)
	})
	m.mu.Unlock()

	if err := m.sub.Subscribe(filter, id, inlineWrapped); err != nil {
		return err
	}

	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		if err := m.sub.Unsubscribe(filter, id); err != nil {
			return err
		}
		return ErrTopicRegistrarStopped
	}

	m.items = append(m.items, subscriptionEntry{
		filter: filter,
		id:     id,
	})
	m.mu.Unlock()

	return nil
}

func (m *ManagedPubSub) Use(middlewares ...TopicMiddleware) {
	if len(middlewares) == 0 {
		return
	}

	m.mu.Lock()
	m.mws = append(m.mws, middlewares...)
	m.mu.Unlock()
}

func (m *ManagedPubSub) Stop(context.Context) error {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return nil
	}
	m.stopped = true
	items := m.items
	m.items = nil
	m.mu.Unlock()

	var err error
	for _, item := range items {
		err = multierr.Append(err, m.sub.Unsubscribe(item.filter, item.id))
	}

	m.log.Debug("mqtt subscriptions cleaned", zap.Int("count", len(items)))
	return err
}

type setupTopicSubscribersIn struct {
	fx.In
	Log         *zap.Logger `optional:"true"`
	Manager     TopicRegistrar
	Subscribers []usmqtt.TopicSubscriber `group:"mqtt_subscribers"`
}

func SetupTopicSubscribers(in setupTopicSubscribersIn) error {
	if in.Log != nil {
		in.Log.Debug("setting up mqtt topic subscribers", zap.Int("count", len(in.Subscribers)))
	}
	for _, subscriber := range in.Subscribers {
		if err := subscriber.Subscribe(in.Manager); err != nil {
			return err
		}
	}

	return nil
}

type setupTopicMiddlewaresIn struct {
	fx.In
	Manager     TopicRegistrar
	Middlewares []TopicMiddleware `group:"mqtt_topic_middlewares"`
}

func SetupTopicMiddlewares(in setupTopicMiddlewaresIn) {
	in.Manager.Use(in.Middlewares...)
}

func applyTopicMiddlewares(handler TopicHandler, mws []TopicMiddleware) TopicHandler {
	if len(mws) == 0 {
		return handler
	}

	wrapped := handler
	for i := len(mws) - 1; i >= 0; i-- {
		wrapped = mws[i](wrapped)
	}
	return wrapped
}

func parseTopicArgs(args ...any) (TopicHandler, []TopicMiddleware, error) {
	if len(args) == 0 {
		return nil, nil, ErrInvalidTopicRegistrationArgs
	}

	handler, ok := toTopicHandler(args[len(args)-1])
	if !ok || handler == nil {
		return nil, nil, ErrInvalidTopicRegistrationArgs
	}

	mws := make([]TopicMiddleware, 0, len(args)-1)
	for i := 0; i < len(args)-1; i++ {
		mw, ok := toTopicMiddleware(args[i])
		if !ok || mw == nil {
			return nil, nil, ErrInvalidTopicRegistrationArgs
		}
		mws = append(mws, mw)
	}

	return handler, mws, nil
}

func toTopicMiddleware(v any) (TopicMiddleware, bool) {
	switch mw := v.(type) {
	case TopicMiddleware:
		return mw, true
	case func(TopicHandler) TopicHandler:
		return TopicMiddleware(mw), true
	case func(mqtt.InlineSubFn) mqtt.InlineSubFn:
		return wrapInlineSubMiddleware(mw), true
	default:
		return nil, false
	}
}

func toTopicHandler(v any) (TopicHandler, bool) {
	switch fn := v.(type) {
	case TopicHandler:
		return fn, true
	case func(Ctx) error:
		return TopicHandler(fn), true
	case func(Ctx):
		return func(ctx Ctx) error {
			fn(ctx)
			return nil
		}, true
	case mqtt.InlineSubFn:
		return inlineSubFnToTopicHandler(fn), true
	case func(*mqtt.Client, packets.Subscription, packets.Packet):
		return inlineSubFnToTopicHandler(mqtt.InlineSubFn(fn)), true
	default:
		return nil, false
	}
}

func topicHandlerToInlineSubFn(handler TopicHandler, pub usmqtt.Publisher, controller usmqtt.SessionController, onError func(error, Ctx)) mqtt.InlineSubFn {
	return func(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet) {
		ctx := &topicCtx{
			client:     cl,
			pub:        pub,
			controller: controller,
			sub:        sub,
			packet:     pk,
			ctx:        context.Background(),
		}

		if err := handler(ctx); err != nil && onError != nil {
			onError(err, ctx)
		}
	}
}

func inlineSubFnToTopicHandler(handler mqtt.InlineSubFn) TopicHandler {
	return func(ctx Ctx) error {
		handler(ctx.Client(), ctx.Subscription(), ctx.Packet())
		return nil
	}
}

func wrapInlineSubMiddleware(mw func(mqtt.InlineSubFn) mqtt.InlineSubFn) TopicMiddleware {
	return func(next TopicHandler) TopicHandler {
		return func(ctx Ctx) error {
			var nextErr error
			inlineNext := topicHandlerToInlineSubFn(next, nil, nil, func(err error, _ Ctx) {
				nextErr = err
			})
			inlineWrapped := mw(inlineNext)
			inlineWrapped(ctx.Client(), ctx.Subscription(), ctx.Packet())
			return nextErr
		}
	}
}

func (m *ManagedPubSub) isTopicAllowed(filter string) bool {
	if !m.acl.Enabled {
		return true
	}

	for _, prefix := range m.acl.AllowedPrefixes {
		if strings.HasPrefix(filter, prefix) {
			return true
		}
	}
	return false
}
