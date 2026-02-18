package realtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bronystylecrazy/ultrastructure/ditest"
	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/stretchr/testify/mock"
	"go.uber.org/fx"
)

func TestManagedPubSubAutoUnsubscribeOnStop(t *testing.T) {
	sub := NewMockSubscriber(t)

	handler := func(_ *mqtt.Client, _ packets.Subscription, _ packets.Packet) {}

	sub.EXPECT().Subscribe("hello/world", 1, mock.Anything).Return(nil).Once()
	sub.EXPECT().Subscribe("hello", 2, mock.Anything).Return(nil).Once()
	sub.EXPECT().Unsubscribe("hello/world", 1).Return(nil).Once()
	sub.EXPECT().Unsubscribe("hello", 2).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	app.RequireStart()
	if err := manager.Topic("hello/world", handler); err != nil {
		t.Fatalf("Topic(hello/world): %v", err)
	}
	if err := manager.Topic("hello", handler); err != nil {
		t.Fatalf("Topic(hello): %v", err)
	}
	app.RequireStop()
}

type recordingPubSubManager struct {
	topics []string
	err    error
}

func (m *recordingPubSubManager) Use(_ ...TopicMiddleware) {}

func (m *recordingPubSubManager) Topic(filter string, _ ...any) error {
	m.topics = append(m.topics, filter)
	return m.err
}

type testTopicSubscriber struct {
	topic string
}

func (h *testTopicSubscriber) Subscribe(r usmqtt.TopicRegistrar) error {
	return r.Topic(h.topic, func(_ *mqtt.Client, _ packets.Subscription, _ packets.Packet) {})
}

func TestSetupTopicSubscribers(t *testing.T) {
	manager := &recordingPubSubManager{}
	first := &testTopicSubscriber{topic: "/hello/world"}
	second := &testTopicSubscriber{topic: "/hello"}

	err := SetupTopicSubscribers(setupTopicSubscribersIn{
		Manager:     manager,
		Subscribers: []usmqtt.TopicSubscriber{first, second},
	})
	if err != nil {
		t.Fatalf("SetupTopicSubscribers: %v", err)
	}

	if len(manager.topics) != 2 {
		t.Fatalf("topics count mismatch: got=%d want=2", len(manager.topics))
	}
	if manager.topics[0] != "/hello/world" || manager.topics[1] != "/hello" {
		t.Fatalf("topics mismatch: got=%v", manager.topics)
	}
}

func TestManagedPubSubMiddlewareOrder(t *testing.T) {
	sub := NewMockSubscriber(t)
	calls := make([]string, 0, 5)

	mw1 := func(next TopicHandler) TopicHandler {
		return func(ctx Ctx) error {
			calls = append(calls, "mw1:before")
			if err := next(ctx); err != nil {
				return err
			}
			calls = append(calls, "mw1:after")
			return nil
		}
	}
	mw2 := func(next TopicHandler) TopicHandler {
		return func(ctx Ctx) error {
			calls = append(calls, "mw2:before")
			if err := next(ctx); err != nil {
				return err
			}
			calls = append(calls, "mw2:after")
			return nil
		}
	}

	sub.EXPECT().
		Subscribe("hello/mw", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(nil, packets.Subscription{Filter: "hello/mw"}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/mw", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	app.RequireStart()
	manager.Use(mw1, mw2)
	err := manager.Topic("hello/mw", func(_ *mqtt.Client, _ packets.Subscription, _ packets.Packet) {
		calls = append(calls, "handler")
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	want := []string{"mw1:before", "mw2:before", "handler", "mw2:after", "mw1:after"}
	if len(calls) != len(want) {
		t.Fatalf("calls length mismatch: got=%d want=%d calls=%v", len(calls), len(want), calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls mismatch at %d: got=%q want=%q all=%v", i, calls[i], want[i], calls)
		}
	}
}

func TestManagedPubSubTopicWithInlineMiddlewares(t *testing.T) {
	sub := NewMockSubscriber(t)
	calls := make([]string, 0, 7)

	global := func(next TopicHandler) TopicHandler {
		return func(ctx Ctx) error {
			calls = append(calls, "global:before")
			if err := next(ctx); err != nil {
				return err
			}
			calls = append(calls, "global:after")
			return nil
		}
	}
	local := func(next mqtt.InlineSubFn) mqtt.InlineSubFn {
		return func(cl *mqtt.Client, s packets.Subscription, pk packets.Packet) {
			calls = append(calls, "local:before")
			next(cl, s, pk)
			calls = append(calls, "local:after")
		}
	}

	sub.EXPECT().
		Subscribe("hello/inline", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(nil, packets.Subscription{Filter: "hello/inline"}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/inline", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	app.RequireStart()
	manager.Use(global)
	err := manager.Topic("hello/inline", local, func(_ *mqtt.Client, _ packets.Subscription, _ packets.Packet) {
		calls = append(calls, "handler")
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	want := []string{"global:before", "local:before", "handler", "local:after", "global:after"}
	if len(calls) != len(want) {
		t.Fatalf("calls length mismatch: got=%d want=%d calls=%v", len(calls), len(want), calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls mismatch at %d: got=%q want=%q all=%v", i, calls[i], want[i], calls)
		}
	}
}

func TestManagedPubSubTopicWithTopicContext(t *testing.T) {
	sub := NewMockSubscriber(t)

	type payload struct {
		Message string `json:"message"`
	}

	sub.EXPECT().
		Subscribe("hello/context", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(nil, packets.Subscription{Filter: "hello/+"}, packets.Packet{
				TopicName: "hello/context",
				Payload:   []byte(`{"message":"hello"}`),
			})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/context", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	var gotTopic string
	var gotFilter string
	var gotPayload payload

	app.RequireStart()
	err := manager.Topic("hello/context", func(ctx Ctx) {
		gotTopic = ctx.Topic()
		gotFilter = ctx.Filter()
		if decodeErr := ctx.DecodeJSON(&gotPayload); decodeErr != nil {
			t.Fatalf("DecodeJSON: %v", decodeErr)
		}
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	if gotTopic != "hello/context" {
		t.Fatalf("topic mismatch: got=%q", gotTopic)
	}
	if gotFilter != "hello/+" {
		t.Fatalf("filter mismatch: got=%q", gotFilter)
	}
	if gotPayload.Message != "hello" {
		t.Fatalf("payload mismatch: got=%q", gotPayload.Message)
	}
}

func TestManagedPubSubTopicArgsValidation(t *testing.T) {
	sub := NewMockSubscriber(t)
	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	app.RequireStart()
	if err := manager.Topic("hello/invalid"); !errors.Is(err, ErrInvalidTopicRegistrationArgs) {
		t.Fatalf("expected ErrInvalidTopicRegistrationArgs for empty args, got: %v", err)
	}
	if err := manager.Topic("hello/invalid", "not-middleware", func(_ *mqtt.Client, _ packets.Subscription, _ packets.Packet) {}); !errors.Is(err, ErrInvalidTopicRegistrationArgs) {
		t.Fatalf("expected ErrInvalidTopicRegistrationArgs for bad middleware, got: %v", err)
	}
	if err := manager.Topic("hello/invalid", "not-handler"); !errors.Is(err, ErrInvalidTopicRegistrationArgs) {
		t.Fatalf("expected ErrInvalidTopicRegistrationArgs for bad handler, got: %v", err)
	}
	app.RequireStop()
}

func TestManagedPubSubContextPassThroughMiddleware(t *testing.T) {
	sub := NewMockSubscriber(t)
	sub.EXPECT().
		Subscribe("hello/ctx", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(nil, packets.Subscription{}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/ctx", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	type key string
	const userIDKey key = "user-id"

	var gotUserID string
	mw := func(next TopicHandler) TopicHandler {
		return func(ctx Ctx) error {
			ctx.SetContext(context.WithValue(ctx.Context(), userIDKey, "u-123"))
			return next(ctx)
		}
	}

	app.RequireStart()
	manager.Use(mw)
	err := manager.Topic("hello/ctx", func(ctx Ctx) {
		v := ctx.Context().Value(userIDKey)
		if s, ok := v.(string); ok {
			gotUserID = s
		}
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	if gotUserID != "u-123" {
		t.Fatalf("context value mismatch: got=%q", gotUserID)
	}
}

func TestClientIdentityHookAndMiddleware(t *testing.T) {
	store := NewClientIdentityStore()
	hook := NewClientIdentityHook(store)

	cl := &mqtt.Client{ID: "c-1"}
	pk := packets.Packet{}
	pk.Connect.Username = []byte("alice")

	if err := hook.OnConnect(cl, pk); err != nil {
		t.Fatalf("OnConnect: %v", err)
	}

	sub := NewMockSubscriber(t)
	sub.EXPECT().
		Subscribe("hello/identity", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(cl, packets.Subscription{}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/identity", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	var got ClientIdentity
	var ok bool

	app.RequireStart()
	manager.Use(UseClientIdentityContext(store))
	err := manager.Topic("hello/identity", func(ctx Ctx) {
		got, ok = ctx.Identity()
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	if !ok {
		t.Fatal("expected identity in context")
	}
	if got.ClientID != "c-1" {
		t.Fatalf("client id mismatch: got=%q", got.ClientID)
	}
	if got.Username != "alice" {
		t.Fatalf("username mismatch: got=%q", got.Username)
	}

	hook.OnDisconnect(cl, nil, false)
	if _, exists := store.Get("c-1"); exists {
		t.Fatal("identity should be removed on disconnect")
	}
}

func TestManagedPubSubTopicACLAllowed(t *testing.T) {
	sub := NewMockSubscriber(t)
	sub.EXPECT().Subscribe("/public/hello", 1, mock.Anything).Return(nil).Once()
	sub.EXPECT().Unsubscribe("/public/hello", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Supply(&Config{
			TopicACL: TopicACLConfig{
				Enabled:         true,
				AllowedPrefixes: []string{"/public/", "/tenant-a/"},
			},
		}),
		fx.Populate(&manager),
	)

	app.RequireStart()
	err := manager.Topic("/public/hello", func(_ *mqtt.Client, _ packets.Subscription, _ packets.Packet) {})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()
}

func TestManagedPubSubTopicACLDenied(t *testing.T) {
	sub := NewMockSubscriber(t)

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Supply(&Config{
			TopicACL: TopicACLConfig{
				Enabled:         true,
				AllowedPrefixes: []string{"/public/", "/tenant-a/"},
			},
		}),
		fx.Populate(&manager),
	)

	app.RequireStart()
	err := manager.Topic("/private/hello", func(_ *mqtt.Client, _ packets.Subscription, _ packets.Packet) {})
	if err == nil {
		t.Fatal("expected ACL error, got nil")
	}
	if !errors.Is(err, ErrTopicNotAllowed) {
		t.Fatalf("expected ErrTopicNotAllowed, got: %v", err)
	}
	app.RequireStop()
}

func TestCtxDisconnect(t *testing.T) {
	sub := NewMockSubscriber(t)
	cl := &mqtt.Client{ID: "disconnect-client"}

	sub.EXPECT().
		Subscribe("hello/disconnect", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(cl, packets.Subscription{}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/disconnect", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	app.RequireStart()
	err := manager.Topic("hello/disconnect", func(ctx Ctx) {
		if disconnectErr := ctx.Disconnect(); disconnectErr != nil {
			t.Fatalf("Disconnect: %v", disconnectErr)
		}
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	if !errors.Is(cl.StopCause(), ErrTopicClientDisconnectedByHandler) {
		t.Fatalf("stop cause mismatch: got=%v", cl.StopCause())
	}
}

func TestCtxDisconnectWithoutClient(t *testing.T) {
	sub := NewMockSubscriber(t)

	sub.EXPECT().
		Subscribe("hello/disconnect-nil", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(nil, packets.Subscription{}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/disconnect-nil", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	var got error

	app.RequireStart()
	err := manager.Topic("hello/disconnect-nil", func(ctx Ctx) {
		got = ctx.Disconnect()
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	if !errors.Is(got, ErrTopicCtxNoClient) {
		t.Fatalf("disconnect error mismatch: got=%v", got)
	}
}

func TestCtxPublish(t *testing.T) {
	sub := NewMockSubscriber(t)
	pub := NewMockPublisher(t)

	sub.EXPECT().
		Subscribe("hello/publish", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(nil, packets.Subscription{}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/publish", 1).Return(nil).Once()

	pub.EXPECT().PublishJSON("hello/reply", mock.Anything, false, byte(0)).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			func() usmqtt.Publisher { return pub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	app.RequireStart()
	err := manager.Topic("hello/publish", func(ctx Ctx) {
		if pubErr := ctx.PublishJSON("hello/reply", map[string]string{"ok": "yes"}, false, 0); pubErr != nil {
			t.Fatalf("Publish: %v", pubErr)
		}
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()
}

func TestCtxPublishWithoutPublisher(t *testing.T) {
	sub := NewMockSubscriber(t)

	sub.EXPECT().
		Subscribe("hello/no-publisher", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(nil, packets.Subscription{}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/no-publisher", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	var got error
	app.RequireStart()
	err := manager.Topic("hello/no-publisher", func(ctx Ctx) {
		got = ctx.Publish("hello/reply", []byte("x"), false, 0)
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	if !errors.Is(got, ErrTopicCtxNoPublisher) {
		t.Fatalf("publish error mismatch: got=%v", got)
	}
}

func TestCtxDecodeJSON(t *testing.T) {
	sub := NewMockSubscriber(t)
	sub.EXPECT().
		Subscribe("hello/bind", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(nil, packets.Subscription{}, packets.Packet{Payload: []byte(`{"name":"alice"}`)})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/bind", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	type req struct {
		Name string `json:"name"`
	}
	var first req
	var second req

	app.RequireStart()
	err := manager.Topic("hello/bind", func(ctx Ctx) {
		if decodeErr := ctx.DecodeJSON(&first); decodeErr != nil {
			t.Fatalf("DecodeJSON (first): %v", decodeErr)
		}
		if decodeErr := ctx.DecodeJSON(&second); decodeErr != nil {
			t.Fatalf("DecodeJSON (second): %v", decodeErr)
		}
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	if first.Name != "alice" {
		t.Fatalf("first decode mismatch: got=%q", first.Name)
	}
	if second.Name != "alice" {
		t.Fatalf("second decode mismatch: got=%q", second.Name)
	}
}

func TestCtxUsernameClientIDAndReject(t *testing.T) {
	sub := NewMockSubscriber(t)
	cl := &mqtt.Client{ID: "client-123"}
	cl.Properties.Username = []byte("bob")

	sub.EXPECT().
		Subscribe("hello/reject", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(cl, packets.Subscription{}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/reject", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	var gotID string
	var gotUser string

	app.RequireStart()
	err := manager.Topic("hello/reject", func(ctx Ctx) {
		gotID = ctx.ClientID()
		gotUser = ctx.Username()
		if rejectErr := ctx.Reject("unauthorized"); rejectErr != nil {
			t.Fatalf("Reject: %v", rejectErr)
		}
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	if gotID != "client-123" {
		t.Fatalf("client id mismatch: got=%q", gotID)
	}
	if gotUser != "bob" {
		t.Fatalf("username mismatch: got=%q", gotUser)
	}
	if !errors.Is(cl.StopCause(), ErrTopicClientRejectedByHandler) {
		t.Fatalf("reject cause mismatch: got=%v", cl.StopCause())
	}
}

func TestTimeoutTopicMiddlewareCancelOnly(t *testing.T) {
	sub := NewMockSubscriber(t)
	cl := &mqtt.Client{ID: "timeout-cancel-only"}

	sub.EXPECT().
		Subscribe("hello/timeout-cancel", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(cl, packets.Subscription{}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/timeout-cancel", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	done := make(chan error, 1)

	app.RequireStart()
	manager.Use(TimeoutTopicMiddleware(20 * time.Millisecond))
	err := manager.Topic("hello/timeout-cancel", func(ctx Ctx) {
		<-ctx.Done()
		done <- ctx.Err()
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	select {
	case gotErr := <-done:
		if !errors.Is(gotErr, context.DeadlineExceeded) {
			t.Fatalf("ctx error mismatch: got=%v", gotErr)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for handler cancellation")
	}

	if cl.StopCause() != nil {
		t.Fatalf("client should not be disconnected, got stop cause: %v", cl.StopCause())
	}
}

func TestTimeoutTopicMiddlewareWithDisconnect(t *testing.T) {
	sub := NewMockSubscriber(t)
	cl := &mqtt.Client{ID: "timeout-disconnect"}

	sub.EXPECT().
		Subscribe("hello/timeout-disconnect", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(cl, packets.Subscription{}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/timeout-disconnect", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	done := make(chan error, 1)

	app.RequireStart()
	manager.Use(TimeoutTopicMiddleware(20*time.Millisecond, WithTimeoutDisconnect()))
	err := manager.Topic("hello/timeout-disconnect", func(ctx Ctx) {
		<-ctx.Done()
		done <- ctx.Err()
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	select {
	case gotErr := <-done:
		if !errors.Is(gotErr, context.DeadlineExceeded) {
			t.Fatalf("ctx error mismatch: got=%v", gotErr)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for handler cancellation")
	}

	if !errors.Is(cl.StopCause(), ErrTopicClientRejectedByHandler) {
		t.Fatalf("client stop cause mismatch: got=%v", cl.StopCause())
	}
}

type staticConnectClaimsFactory struct {
	claims Claims
}

func (f *staticConnectClaimsFactory) BuildConnectContext(_ *mqtt.Client, _ packets.Packet) context.Context {
	return WithClaims(context.Background(), f.claims)
}

func TestConnectContextClaimsFromConnectPhase(t *testing.T) {
	store := NewClientConnectContextStore()
	factory := &staticConnectClaimsFactory{
		claims: Claims{
			"role":   "admin",
			"tenant": "acme",
		},
	}
	hook := NewClientConnectContextHook(newClientConnectContextHookIn{
		Store:   store,
		Factory: factory,
	})

	cl := &mqtt.Client{ID: "claims-client"}
	if err := hook.OnConnect(cl, packets.Packet{}); err != nil {
		t.Fatalf("OnConnect: %v", err)
	}

	sub := NewMockSubscriber(t)
	sub.EXPECT().
		Subscribe("hello/claims", 1, mock.Anything).
		Run(func(_ string, _ int, h mqtt.InlineSubFn) {
			h(cl, packets.Subscription{}, packets.Packet{})
		}).
		Return(nil).
		Once()
	sub.EXPECT().Unsubscribe("hello/claims", 1).Return(nil).Once()

	var manager *ManagedPubSub
	app := ditest.New(t,
		fx.Provide(
			func() usmqtt.Subscriber { return sub },
			NewManagedPubSub,
		),
		fx.Populate(&manager),
	)

	var gotRole string
	var gotTenant string

	app.RequireStart()
	manager.Use(UseConnectedContext(store))
	err := manager.Topic("hello/claims", func(ctx Ctx) {
		connected, ok := ConnectedContext(ctx.Context())
		if !ok {
			t.Fatal("missing connected context")
		}

		claims, ok := ClaimsFromContext(connected)
		if !ok {
			t.Fatal("missing claims")
		}

		role, _ := claims["role"].(string)
		tenant, _ := claims["tenant"].(string)
		gotRole = role
		gotTenant = tenant
	})
	if err != nil {
		t.Fatalf("Topic: %v", err)
	}
	app.RequireStop()

	if gotRole != "admin" || gotTenant != "acme" {
		t.Fatalf("claims mismatch: role=%q tenant=%q", gotRole, gotTenant)
	}

	hook.OnDisconnect(cl, nil, false)
	if _, ok := store.Get(cl.ID); ok {
		t.Fatal("connected context should be removed on disconnect")
	}
}
