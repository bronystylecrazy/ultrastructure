package realtime

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func UseAllowHook() di.Node {
	return di.Invoke(func(ms *MqttServer, log *zap.Logger) error {
		return ms.AddHook(new(auth.AllowHook), nil)
	})
}

func UseWebsocketListener(opts ...Option) di.Node {
	return di.Options(
		di.Provide(func(app fiber.Router, auth Authorizer) *Websocket {
			base := []Option{
				WithApp(app),
				WithAuthorizer(auth),
			}
			return NewWebsocketWithOptions(append(base, opts...)...)
		}, web.Priority(web.Later), di.Params(``, di.Optional())),
	)
}

func AppendHooks(ms *MqttServer, hooks ...mqtt.Hook) error {
	var err error
	for _, hook := range hooks {
		err = multierr.Append(err, ms.AddHook(hook, nil))
	}
	return err
}

func AppendListeners(ms *MqttServer, listeners ...listeners.Listener) error {
	var err error
	for _, listener := range listeners {
		err = multierr.Append(err, ms.AddListener(listener))
	}
	return err
}
