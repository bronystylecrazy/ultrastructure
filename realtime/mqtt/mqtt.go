package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	mqtt "github.com/mochi-mqtt/server/v2"
)

type Broker interface {
	Publisher
	Subscriber
}

type Publisher interface {
	Publish(topic string, payload []byte, retain bool, qos byte) error
	PublishJSON(topic string, payload any, retain bool, qos byte) error
	PublishString(topic string, payload string, retain bool, qos byte) error
}

type Subscriber interface {
	Subscribe(filter string, subscriptionId int, handler mqtt.InlineSubFn) error
	Unsubscribe(filter string, subscriptionId int) error
}

const (
	NoRetain = false
	Retain   = true
)

const (
	QoS0 byte = 0
	QoS1 byte = 1
	QoS2 byte = 2
)

type Server struct {
	*mqtt.Server
}

func NewServer() (*Server, error) {
	server := &Server{Server: mqtt.New(&mqtt.Options{
		InlineClient: true,
		Logger:       slog.New(slog.DiscardHandler),
	})}
	return server, nil
}

func (m *Server) Publish(topic string, payload []byte, retain bool, qos byte) error {
	return m.Server.Publish(topic, payload, retain, qos)
}

func (m *Server) PublishJSON(topic string, payload any, retain bool, qos byte) error {
	p, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return m.Server.Publish(topic, p, retain, qos)
}

func (m *Server) PublishString(topic string, payload string, retain bool, qos byte) error {
	return m.Publish(topic, []byte(payload), retain, qos)
}

func (m *Server) Start(context.Context) error {
	return m.Server.Serve()
}

func (m *Server) Stop(context.Context) error {
	return m.Server.Close()
}

func (m *Server) DisconnectClient(_ context.Context, clientID string, reason string) error {
	if strings.TrimSpace(clientID) == "" {
		return errors.New("realtime/mqtt: client id is required")
	}

	cl, ok := m.Clients.Get(clientID)
	if !ok || cl == nil {
		return fmt.Errorf("realtime/mqtt: client %q not found", clientID)
	}

	if reason == "" {
		reason = "session disconnected by controller"
	}
	cl.Stop(errors.New(reason))
	return nil
}
