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

// Ctx provides per-message topic handler context, including MQTT metadata,
// payload helpers, publishing helpers, and an overridable base context.
type Ctx interface {
	context.Context

	// Client returns the underlying MQTT client.
	// Example: client := ctx.Client()
	Client() *mqtt.Client
	// ClientID returns the MQTT client identifier.
	// Example: id := ctx.ClientID()
	ClientID() string
	// Username returns the authenticated username when available.
	// Example: user := ctx.Username()
	Username() string
	// Subscription returns the matched subscription for this handler invocation.
	// Example: sub := ctx.Subscription()
	Subscription() packets.Subscription
	// Packet returns the raw MQTT packet associated with this invocation.
	// Example: pk := ctx.Packet()
	Packet() packets.Packet
	// Filter returns the matched topic filter from the subscription.
	// Example: filter := ctx.Filter()
	Filter() string
	// Topic returns the packet topic name.
	// Example: topic := ctx.Topic()
	Topic() string
	// Payload returns the raw packet payload bytes.
	// Example: payload := ctx.Payload()
	Payload() []byte
	// DecodeJSON unmarshals the payload as JSON into v.
	// Example: err := ctx.DecodeJSON(&dst)
	DecodeJSON(v any) error
	// Publish sends raw payload bytes to topic.
	// Example: err := ctx.Publish("devices/ack", []byte("ok"), false, 1)
	Publish(topic string, payload []byte, retain bool, qos byte) error
	// PublishJSON marshals payload as JSON and publishes it to topic.
	// Example: err := ctx.PublishJSON("devices/ack", map[string]any{"ok": true}, false, 1)
	PublishJSON(topic string, payload any, retain bool, qos byte) error
	// PublishString publishes a string payload to topic.
	// Example: err := ctx.PublishString("devices/ack", "ok", false, 1)
	PublishString(topic string, payload string, retain bool, qos byte) error
	// Context returns the current base context used by the Ctx implementation.
	// Example: base := ctx.Context()
	Context() context.Context
	// SetContext replaces the base context used by this Ctx.
	// Example: ctx.SetContext(context.WithValue(ctx.Context(), key, value))
	SetContext(ctx context.Context)
	// Identity returns client identity from context when present.
	// Example: identity, ok := ctx.Identity()
	Identity() (ClientIdentity, bool)
	// Disconnect stops the underlying client connection.
	// Example: err := ctx.Disconnect()
	Disconnect() error
	// Reject stops the client and records a rejection reason.
	// Example: err := ctx.Reject("unauthorized topic")
	Reject(reason string) error
}

type topicCtx struct {
	client     *mqtt.Client
	pub        usmqtt.Publisher
	controller usmqtt.SessionController
	sub        packets.Subscription
	packet     packets.Packet
	ctx        context.Context
}

var ErrTopicCtxNoClient = errors.New("realtime: topic context has no client")
var ErrTopicCtxNoPublisher = errors.New("realtime: topic context has no publisher")
var ErrTopicCtxSessionControlUnsupported = errors.New("realtime: session control is unsupported")
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
	if c.controller != nil {
		if err := c.controller.DisconnectClient(c.Context(), c.client.ID, "disconnected by topic handler"); err != nil {
			if errors.Is(err, usmqtt.ErrSessionControlUnsupported) {
				return fmt.Errorf("%w: %w", ErrTopicCtxSessionControlUnsupported, err)
			}
			return err
		}
		return nil
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
	if c.controller != nil {
		if err := c.controller.DisconnectClient(c.Context(), c.client.ID, reason); err != nil {
			if errors.Is(err, usmqtt.ErrSessionControlUnsupported) {
				return fmt.Errorf("%w: %w", ErrTopicCtxSessionControlUnsupported, err)
			}
			return err
		}
		return nil
	}
	c.client.Stop(fmt.Errorf("%w: %s", ErrTopicClientRejectedByHandler, reason))
	return nil
}

type TopicHandler func(ctx Ctx) error
