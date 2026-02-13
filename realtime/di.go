package realtime

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func UseAllowHook() di.Node {
	return di.Invoke(func(ms *usmqtt.Server, log *zap.Logger) error {
		return ms.AddHook(new(auth.AllowHook), nil)
	})
}

func UseTCPListener() di.Node {
	return di.Invoke(func(ms *usmqtt.Server, cfg Config, log *zap.Logger) error {
		id := cfg.TCPListener.ID
		if id == "" {
			id = "t1"
		}

		address := cfg.TCPListener.Address
		if address == "" {
			address = ":1883"
		}

		return ms.AddListener(listeners.NewTCP(listeners.Config{
			ID:      id,
			Address: address,
		}))
	})
}

func UseWebsocketListener(opts ...Option) di.Node {
	return di.Options(
		di.Provide(func(app fiber.Router, auth Authorizer, cfg Config) *Websocket {
			id := cfg.WebsocketListener.ID
			if id == "" {
				id = "ws1"
			}

			path := cfg.WebsocketListener.Path
			if path == "" {
				path = "/realtime"
			}

			base := []Option{
				WithApp(app),
				WithAuthorizer(auth),
				WithId(id),
				WithPath(path),
			}
			return NewWebsocketWithOptions(append(base, opts...)...)
		}, web.Priority(web.Later), di.Params(``, di.Optional())),
	)
}

func AppendHooks(ms *usmqtt.Server, hooks ...mqtt.Hook) error {
	var err error
	for _, hook := range hooks {
		err = multierr.Append(err, ms.AddHook(hook, nil))
	}
	return err
}

func AppendListeners(ms *usmqtt.Server, listeners ...listeners.Listener) error {
	var err error
	for _, listener := range listeners {
		err = multierr.Append(err, ms.AddListener(listener))
	}
	return err
}
