package realtime

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
)

var HooksGroupName = "mqtt_hooks"
var ListenersGroupName = "mqtt_listeners"

func Module(opts ...di.Node) di.Node {
	return di.Options(
		di.AutoGroup[mqtt.Hook](HooksGroupName),
		di.AutoGroup[listeners.Listener](ListenersGroupName),
		di.Provide(NewMqttServer, di.As[Server](), di.AsSelf()),
		di.Provide(NewAuthorizer),
		di.Options(di.ConvertAnys(opts)...),
		di.Invoke(AppendHooks, di.Params(``, di.Group(HooksGroupName))),
		di.Invoke(AppendListeners, di.Params(``, di.Group(ListenersGroupName))),
	)
}
