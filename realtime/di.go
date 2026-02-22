package realtime

import (
	"context"
	"log/slog"

	"github.com/bronystylecrazy/ultrastructure/di"
	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	"github.com/bronystylecrazy/ultrastructure/web"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"go.uber.org/fx"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func Init() di.Node {
	return di.Options(
		di.Invoke(AppendHooks, di.Params(``, di.Group(HooksGroupName))),
		di.Invoke(AppendListeners, di.Params(``, di.Group(ListenersGroupName))),
		di.Invoke(RegisterBrokerLifecycle),
		di.Invoke(SetupTopicMiddlewares),
		di.Invoke(SetupTopicSubscribers),
		di.Invoke(func(lc fx.Lifecycle, broker usmqtt.Broker, logger *slog.Logger) {
			server, ok := broker.(*usmqtt.Server)
			if !ok || server == nil {
				return
			}

			server.Log = logger

			lc.Append(fx.Hook{
				OnStart: server.Start,
				OnStop:  server.Stop,
			})
		}),
	)
}

func UseAllowHook() di.Node {
	return di.Invoke(func(ms usmqtt.Broker, log *zap.Logger) error {
		srv, ok := ms.(*usmqtt.Server)
		if !ok || srv == nil {
			log.Debug("skipping mqtt allow hook: broker is not embedded mochi server")
			return nil
		}

		return srv.AddHook(new(auth.AllowHook), nil)
	})
}

func UseTCPListener() di.Node {
	return di.Invoke(func(ms usmqtt.Broker, cfg Config, log *zap.Logger) error {
		srv, ok := ms.(*usmqtt.Server)
		if !ok || srv == nil {
			log.Debug("skipping mqtt tcp listener: broker is external")
			return nil
		}

		id := cfg.TCPListener.ID
		if id == "" {
			id = "t1"
		}

		address := cfg.TCPListener.Address
		if address == "" {
			address = ":1883"
		}

		return srv.AddListener(listeners.NewTCP(listeners.Config{
			ID:      id,
			Address: address,
		}))
	})
}

func UseWebsocketListener(opts ...Option) di.Node {
	return di.Options(
		di.Provide(func(auth Authorizer, cfg Config, broker usmqtt.Broker) *Websocket {
			if _, ok := broker.(*usmqtt.Server); !ok {
				return nil
			}

			id := cfg.WebsocketListener.ID
			if id == "" {
				id = "ws1"
			}

			path := cfg.WebsocketListener.Path
			if path == "" {
				path = "/realtime"
			}

			base := []Option{
				WithAuthorizer(auth),
				WithId(id),
				WithPath(path),
			}
			return NewWebsocketWithOptions(append(base, opts...)...)
		}, web.Priority(web.Later), di.Params(``, di.Optional())),
	)
}

func AppendHooks(ms usmqtt.Broker, hooks ...mqtt.Hook) error {
	srv, ok := ms.(*usmqtt.Server)
	if !ok || srv == nil {
		return nil
	}

	var err error
	for _, hook := range hooks {
		err = multierr.Append(err, srv.AddHook(hook, nil))
	}
	return err
}

func AppendListeners(ms usmqtt.Broker, listeners ...listeners.Listener) error {
	srv, ok := ms.(*usmqtt.Server)
	if !ok || srv == nil {
		return nil
	}

	var err error
	for _, listener := range listeners {
		err = multierr.Append(err, srv.AddListener(listener))
	}
	return err
}

type startStopper interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

func RegisterBrokerLifecycle(lc fx.Lifecycle, broker usmqtt.Broker) {
	if _, ok := broker.(*usmqtt.Server); ok {
		// Embedded server already participates in lifecycle via lc.Module auto-grouping.
		return
	}

	ss, ok := broker.(startStopper)
	if !ok {
		return
	}

	lc.Append(fx.Hook{
		OnStart: ss.Start,
		OnStop:  ss.Stop,
	})
}
