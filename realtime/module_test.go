package realtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"go.uber.org/fx/fxtest"
)

func TestModuleProvidesBrokerPublisherSubscriber(t *testing.T) {
	var broker usmqtt.Broker
	var publisher usmqtt.Publisher
	var subscriber usmqtt.Subscriber

	app := fxtest.New(t,
		di.App(
			Module(),
			di.Populate(&broker),
			di.Populate(&publisher),
			di.Populate(&subscriber),
		).Build(),
	)

	defer app.RequireStart().RequireStop()

	if broker == nil {
		t.Fatal("broker is nil")
	}
	if publisher == nil {
		t.Fatal("publisher is nil")
	}
	if subscriber == nil {
		t.Fatal("subscriber is nil")
	}

	brokerImpl, ok := broker.(*usmqtt.Server)
	if !ok {
		t.Fatalf("broker is %T, want *mqtt.Server", broker)
	}
	publisherImpl, ok := publisher.(*usmqtt.Server)
	if !ok {
		t.Fatalf("publisher is %T, want *mqtt.Server", publisher)
	}
	subscriberImpl, ok := subscriber.(*usmqtt.Server)
	if !ok {
		t.Fatalf("subscriber is %T, want *mqtt.Server", subscriber)
	}

	if brokerImpl != publisherImpl || brokerImpl != subscriberImpl {
		t.Fatal("broker, publisher, and subscriber are not the same instance")
	}

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		_ = brokerImpl.Start(serverCtx)
	}()
	defer func() {
		_ = brokerImpl.Stop(context.Background())
	}()

	received := make(chan []byte, 1)
	err := subscriber.Subscribe("test/di", 1, func(_ *mqtt.Client, _ packets.Subscription, pk packets.Packet) {
		select {
		case received <- pk.Payload:
		default:
		}
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	want := testPayload{Message: "from-di"}
	if err := publisher.PublishJSON("test/di", want, false, 0); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case payload := <-received:
		var got testPayload
		if err := json.Unmarshal(payload, &got); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if got != want {
			t.Fatalf("payload mismatch: got=%v want=%v", got, want)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for published message")
	}

	if err := subscriber.Unsubscribe("test/di", 1); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}

	if err := publisher.PublishJSON("test/di", testPayload{Message: "ignored"}, false, 0); err != nil {
		t.Fatalf("Publish after unsubscribe: %v", err)
	}

	select {
	case payload := <-received:
		t.Fatalf("unexpected message after unsubscribe: %s", string(payload))
	case <-time.After(150 * time.Millisecond):
	}
}
