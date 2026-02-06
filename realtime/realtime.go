package realtime

import (
	mqtt "github.com/mochi-mqtt/server/v2"
)

type Broker interface {
	Publisher
	Subscriber
}

type Publisher interface {
	Publish(topic string, payload any, retain bool, qos byte) error
}

type Subscriber interface {
	Subscribe(filter string, subscriptionId int, handler mqtt.InlineSubFn) error
	Unsubscribe(filter string, subscriptionId int) error
}
