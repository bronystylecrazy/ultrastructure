package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type testPayload struct {
	Message string `json:"message"`
}

func TestMqttServerPublishSubscribeRoundtrip(t *testing.T) {
	server, err := usmqtt.NewServer(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = server.Start(ctx)
	}()
	defer func() {
		_ = server.Stop(context.Background())
	}()

	received := make(chan []byte, 1)
	err = server.Subscribe("test/topic", 1, func(_ *mqtt.Client, _ packets.Subscription, pk packets.Packet) {
		select {
		case received <- pk.Payload:
		default:
		}
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	want := testPayload{Message: "hello"}
	if err := server.PublishJSON("test/topic", want, false, 0); err != nil {
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

	if err := server.Unsubscribe("test/topic", 1); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}

	if err := server.PublishJSON("test/topic", testPayload{Message: "ignored"}, false, 0); err != nil {
		t.Fatalf("Publish after unsubscribe: %v", err)
	}

	select {
	case payload := <-received:
		t.Fatalf("unexpected message after unsubscribe: %s", string(payload))
	case <-time.After(150 * time.Millisecond):
	}
}
