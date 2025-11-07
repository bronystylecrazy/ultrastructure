package realtime

import (
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"go.uber.org/fx"
)

type Server interface {
	Start() error
	Send(topic string, payload any, retain bool, qos byte) error
	Subscribe(filter string, subscriptionId int, handler mqtt.InlineSubFn) error
	Unsubscribe(filter string, subscriptionId int) error
	AppendHooks(hooks ...mqtt.Hook) error
	AppendListeners(listeners ...listeners.Listener) error
	Close() error
}

func AsHook(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(mqtt.Hook)),
		fx.ResultTags(`group:"realtime.hooks"`),
	)
}

func AsListener(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(listeners.Listener)),
		fx.ResultTags(`group:"realtime.listeners"`),
	)
}

func WithHooks(f any) any {
	return fx.Annotate(f, fx.ParamTags(`group:"realtime.hooks"`))
}

func WithListeners(f any) any {
	return fx.Annotate(f, fx.ParamTags(`group:"realtime.listeners"`))
}

func AppendHooks(hooks []mqtt.Hook, server Server) error {
	return server.AppendHooks(hooks...)
}

func AppendListeners(listeners []listeners.Listener, server Server) error {
	return server.AppendListeners(listeners...)
}
