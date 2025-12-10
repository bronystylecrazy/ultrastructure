package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bronystylecrazy/flexinfra/logging"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"go.uber.org/zap"
)

type MqttServer struct {
	*mqtt.Server
	*logging.Log
}

func NewMqttServer() (Server, error) {
	server := mqtt.New(&mqtt.Options{
		InlineClient: true,
		Logger:       slog.New(slog.DiscardHandler),
	})

	err := server.AddHook(new(auth.AllowHook), nil)
	if err != nil {
		return nil, fmt.Errorf("hook: %w", err)
	}

	return &MqttServer{Server: server}, nil
}

func (m *MqttServer) AppendHooks(hooks ...mqtt.Hook) error {
	for _, hook := range hooks {
		if err := m.Server.AddHook(hook, nil); err != nil {
			return fmt.Errorf("hook: %w", err)
		}
		m.L().Info("registered hook", zap.String("id", hook.ID()))
	}
	return nil
}

func (m *MqttServer) AppendListeners(listeners ...listeners.Listener) error {
	for _, listener := range listeners {
		if err := m.Server.AddListener(listener); err != nil {
			return fmt.Errorf("listener: %w", err)
		}
		m.L().Info("registered listener", zap.String("id", listener.ID()))
	}
	return nil
}

func (m *MqttServer) Start(context.Context) error {
	return m.Server.Serve()
}

func (m *MqttServer) Send(topic string, payload any, retain bool, qos byte) error {

	p, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return m.Server.Publish(topic, p, retain, qos)
}

func (m *MqttServer) Stop(context.Context) error {
	return m.Server.Close()
}
