package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
)

type MqttServer struct {
	*mqtt.Server
}

func NewMqttServer() (*MqttServer, error) {

	server := &MqttServer{Server: mqtt.New(&mqtt.Options{
		InlineClient: true,
		Logger:       slog.New(slog.DiscardHandler),
	})}

	err := server.AddHook(new(auth.AllowHook), nil)
	if err != nil {
		return nil, fmt.Errorf("hook: %w", err)
	}

	return server, nil
}

func (m *MqttServer) Publish(topic string, payload any, retain bool, qos byte) error {

	p, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return m.Server.Publish(topic, p, retain, qos)
}

func (m *MqttServer) Start(context.Context) error {
	return m.Server.Serve()
}

func (m *MqttServer) Stop(context.Context) error {
	return m.Server.Close()
}
