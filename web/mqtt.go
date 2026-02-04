package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/realtime"
	"go.uber.org/zap"
)

func UseMqttWebsocket() di.Node {
	return di.Options(
		di.Provide(realtime.NewWebsocket, Priority(Later), di.Params(``, di.Optional())),
		di.Invoke(func(log *zap.Logger) {
			log.Debug("use mqtt websocket")
		}),
	)
}

func RegisterMqttToWebsocket(ms *realtime.MqttServer, ws *realtime.Websocket) error {
	return ms.AddListener(ws)
}
