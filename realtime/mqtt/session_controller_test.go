package mqtt_test

import (
	"context"
	"errors"
	"testing"

	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
)

func TestServerDisconnectClient(t *testing.T) {
	srv, err := usmqtt.NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	cl := srv.Server.NewClient(nil, "tcp1", "client-x", false)
	srv.Clients.Add(cl)

	err = srv.DisconnectClient(context.Background(), "client-x", "integration-test-disconnect")
	if err != nil {
		t.Fatalf("DisconnectClient: %v", err)
	}

	cause := cl.StopCause()
	if cause == nil {
		t.Fatal("expected stop cause, got nil")
	}
	if cause.Error() != "integration-test-disconnect" {
		t.Fatalf("stop cause mismatch: got=%q want=%q", cause.Error(), "integration-test-disconnect")
	}
}

func TestExternalDisconnectClientUnsupported(t *testing.T) {
	ext, err := usmqtt.NewExternal(usmqtt.ExternalConfig{
		Endpoint: "127.0.0.1:1883",
		ClientID: "client-y",
	})
	if err != nil {
		t.Fatalf("NewExternal: %v", err)
	}

	err = ext.DisconnectClient(context.Background(), "client-y", "nope")
	if !errors.Is(err, usmqtt.ErrSessionControlUnsupported) {
		t.Fatalf("error mismatch: got=%v", err)
	}
}
