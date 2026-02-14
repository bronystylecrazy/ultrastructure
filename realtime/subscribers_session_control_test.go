package realtime

import (
	"context"
	"errors"
	"testing"

	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	mqtt "github.com/mochi-mqtt/server/v2"
)

func TestResolveSessionControllerProviderInvalid(t *testing.T) {
	_, err := resolveSessionController(Config{
		SessionControl: SessionControlConfig{
			Provider: "nope",
		},
	}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveSessionControllerProviderEMQXRequiresEndpoint(t *testing.T) {
	_, err := resolveSessionController(Config{
		SessionControl: SessionControlConfig{
			Provider: "emqx",
		},
	}, nil)
	if !errors.Is(err, usmqtt.ErrEMQXEndpointRequired) {
		t.Fatalf("error mismatch: got=%v", err)
	}
}

type testSessionControllerBroker struct{}

func (testSessionControllerBroker) Publish(string, []byte, bool, byte) error { return nil }
func (testSessionControllerBroker) PublishJSON(string, any, bool, byte) error {
	return nil
}
func (testSessionControllerBroker) PublishString(string, string, bool, byte) error { return nil }
func (testSessionControllerBroker) Subscribe(string, int, mqtt.InlineSubFn) error {
	return nil
}
func (testSessionControllerBroker) Unsubscribe(string, int) error { return nil }
func (testSessionControllerBroker) DisconnectClient(context.Context, string, string) error {
	return nil
}

func TestResolveSessionControllerFallbackToBroker(t *testing.T) {
	sc, err := resolveSessionController(Config{}, testSessionControllerBroker{})
	if err != nil {
		t.Fatalf("resolveSessionController: %v", err)
	}
	if sc == nil {
		t.Fatal("expected session controller from broker fallback, got nil")
	}
}
