package realtime

import (
	mqtt "github.com/mochi-mqtt/server/v2"
)

type Server interface {
	Send(topic string, payload any, retain bool, qos byte) error
	Subscribe(filter string, subscriptionId int, handler mqtt.InlineSubFn) error
	Unsubscribe(filter string, subscriptionId int) error
}
