package realtime

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/lifecycle"
	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
)

var HooksGroupName = "mqtt_hooks"
var ListenersGroupName = "mqtt_listeners"

func Module(opts ...di.Node) di.Node {
	return di.Options(
		di.AutoGroup[mqtt.Hook](HooksGroupName),
		di.AutoGroup[listeners.Listener](ListenersGroupName),
		di.AutoGroup[usmqtt.TopicSubscriber](SubscribersGroupName),
		di.Config[Config]("realtime"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
		di.Provide(usmqtt.NewServer, di.AutoGroupIgnoreType[lifecycle.Starter](), di.AutoGroupIgnoreType[lifecycle.Stopper]()),
		di.Provide(
			NewBroker,
			di.As[usmqtt.Broker](),
			di.As[usmqtt.Publisher](),
			di.As[usmqtt.Subscriber](),
		),
		di.Provide(NewClientIdentityStore),
		di.Provide(
			NewClientIdentityHook,
			di.As[mqtt.Hook](`group:"mqtt_hooks"`),
		),
		di.Provide(NewClientConnectContextStore),
		di.Provide(
			NewClientConnectContextHook,
			di.As[mqtt.Hook](`group:"mqtt_hooks"`),
		),
		di.Provide(
			UseClientIdentityContext,
			di.As[TopicMiddleware](`group:"mqtt_topic_middlewares"`),
		),
		di.Provide(
			UseConnectedContext,
			di.As[TopicMiddleware](`group:"mqtt_topic_middlewares"`),
		),
		di.Provide(
			NewManagedPubSub,
			di.AsSelf[TopicRegistrar](),
		),
		di.Options(di.ConvertAnys(opts)...),
	)
}
