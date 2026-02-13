package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type Ctx interface {
	context.Context

	Client() *mqtt.Client
	ClientID() string
	Username() string
	Subscription() packets.Subscription
	Packet() packets.Packet
	Filter() string
	Topic() string
	Payload() []byte
	DecodeJSON(v any) error
	BindJSON(v any) error
	Publish(topic string, payload []byte, retain bool, qos byte) error
	PublishJSON(topic string, payload any, retain bool, qos byte) error
	PublishString(topic string, payload string, retain bool, qos byte) error
	Context() context.Context
	SetContext(ctx context.Context)
	Identity() (ClientIdentity, bool)
	Disconnect() error
	Reject(reason string) error
}

type topicCtx struct {
	client *mqtt.Client
	pub    usmqtt.Publisher
	sub    packets.Subscription
	packet packets.Packet
	ctx    context.Context
}

var ErrTopicCtxNoClient = errors.New("realtime: topic context has no client")
var ErrTopicCtxNoPublisher = errors.New("realtime: topic context has no publisher")
var ErrTopicClientDisconnectedByHandler = errors.New("realtime: client disconnected by topic handler")
var ErrTopicClientRejectedByHandler = errors.New("realtime: client rejected by topic handler")

func (c *topicCtx) Client() *mqtt.Client {
	return c.client
}

func (c *topicCtx) ClientID() string {
	if c.client == nil {
		return ""
	}
	return c.client.ID
}

func (c *topicCtx) Username() string {
	if identity, ok := c.Identity(); ok && identity.Username != "" {
		return identity.Username
	}
	if c.client == nil {
		return ""
	}
	return string(c.client.Properties.Username)
}

func (c *topicCtx) Subscription() packets.Subscription {
	return c.sub
}

func (c *topicCtx) Packet() packets.Packet {
	return c.packet
}

func (c *topicCtx) Filter() string {
	return c.sub.Filter
}

func (c *topicCtx) Topic() string {
	return c.packet.TopicName
}

func (c *topicCtx) Payload() []byte {
	return c.packet.Payload
}

func (c *topicCtx) DecodeJSON(v any) error {
	return json.Unmarshal(c.packet.Payload, v)
}

func (c *topicCtx) BindJSON(v any) error {
	return c.DecodeJSON(v)
}

func BindJSON[T any](ctx Ctx) (T, error) {
	var out T
	err := ctx.DecodeJSON(&out)
	return out, err
}

func (c *topicCtx) Publish(topic string, payload []byte, retain bool, qos byte) error {
	if c.pub == nil {
		return ErrTopicCtxNoPublisher
	}
	return c.pub.Publish(topic, payload, retain, qos)
}

func (c *topicCtx) PublishJSON(topic string, payload any, retain bool, qos byte) error {
	if c.pub == nil {
		return ErrTopicCtxNoPublisher
	}
	return c.pub.PublishJSON(topic, payload, retain, qos)
}

func (c *topicCtx) PublishString(topic string, payload string, retain bool, qos byte) error {
	if c.pub == nil {
		return ErrTopicCtxNoPublisher
	}
	return c.pub.PublishString(topic, payload, retain, qos)
}

func (c *topicCtx) Context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *topicCtx) Deadline() (time.Time, bool) {
	return c.Context().Deadline()
}

func (c *topicCtx) Done() <-chan struct{} {
	return c.Context().Done()
}

func (c *topicCtx) Err() error {
	return c.Context().Err()
}

func (c *topicCtx) Value(key any) any {
	return c.Context().Value(key)
}

func (c *topicCtx) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *topicCtx) Identity() (ClientIdentity, bool) {
	return IdentityFromContext(c.Context())
}

func (c *topicCtx) Disconnect() error {
	if c.client == nil {
		return ErrTopicCtxNoClient
	}
	c.client.Stop(ErrTopicClientDisconnectedByHandler)
	return nil
}

func (c *topicCtx) Reject(reason string) error {
	if c.client == nil {
		return ErrTopicCtxNoClient
	}
	if reason == "" {
		reason = "unspecified"
	}
	c.client.Stop(fmt.Errorf("%w: %s", ErrTopicClientRejectedByHandler, reason))
	return nil
}

type TopicHandler func(ctx Ctx) error
