package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/realtime"
	"github.com/mochi-mqtt/server/v2/listeners"
)

func UseMqttWebsocket() di.Node {
	return di.Options(
		di.Provide(realtime.NewWebsocket, di.AutoGroupIgnoreType[listeners.Listener]()),
		di.Invoke(RegisterMqttToWebsocket),
	)
}

func RegisterMqttToWebsocket(ms *realtime.MqttServer, ws *realtime.Websocket) error {
	return ms.AddListener(ws)
}
