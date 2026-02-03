package realtime

import (
	"fmt"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
)

func AllowHook(ms *MqttServer) error {
	err := ms.AddHook(new(auth.AllowHook), nil)
	if err != nil {
		return fmt.Errorf("hook: %w", err)
	}
	return nil
}

func AppendHooks(ms *MqttServer, hooks ...mqtt.Hook) error {
	for _, hook := range hooks {
		if err := ms.AddHook(hook, nil); err != nil {
			return fmt.Errorf("hook: %w", err)
		}
	}
	return nil
}

func AppendListeners(ms *MqttServer, listeners ...listeners.Listener) error {
	for _, listener := range listeners {
		if err := ms.AddListener(listener); err != nil {
			return fmt.Errorf("listener: %w", err)
		}
	}
	return nil
}
