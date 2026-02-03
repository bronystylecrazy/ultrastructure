package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/realtime"
)

func UseMqttWebsocket() di.Node {
	return di.Options(
		di.Provide(realtime.NewWebsocket, Priority(Later), di.Params(``, di.Optional())),
	)
}

func RegisterMqttToWebsocket(ms *realtime.MqttServer, ws *realtime.Websocket) error {
	return ms.AddListener(ws)
}
