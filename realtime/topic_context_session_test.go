package realtime

import (
	"context"
	"errors"
	"testing"

	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	mqtt "github.com/mochi-mqtt/server/v2"
)

type recordingSessionController struct {
	clientID string
	reason   string
	err      error
}

func (r *recordingSessionController) DisconnectClient(_ context.Context, clientID string, reason string) error {
	r.clientID = clientID
	r.reason = reason
	return r.err
}

func TestTopicCtxDisconnectUsesSessionController(t *testing.T) {
	ctrl := &recordingSessionController{}
	ctx := &topicCtx{
		client:     &mqtt.Client{ID: "client-1"},
		controller: ctrl,
		ctx:        context.Background(),
	}

	if err := ctx.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if ctrl.clientID != "client-1" {
		t.Fatalf("controller client id mismatch: got=%q want=%q", ctrl.clientID, "client-1")
	}
	if ctrl.reason != "disconnected by topic handler" {
		t.Fatalf("controller reason mismatch: got=%q", ctrl.reason)
	}
}

func TestTopicCtxRejectUsesSessionController(t *testing.T) {
	ctrl := &recordingSessionController{}
	ctx := &topicCtx{
		client:     &mqtt.Client{ID: "client-2"},
		controller: ctrl,
		ctx:        context.Background(),
	}

	if err := ctx.Reject("unauthorized topic"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if ctrl.clientID != "client-2" {
		t.Fatalf("controller client id mismatch: got=%q want=%q", ctrl.clientID, "client-2")
	}
	if ctrl.reason != "unauthorized topic" {
		t.Fatalf("controller reason mismatch: got=%q want=%q", ctrl.reason, "unauthorized topic")
	}
}

func TestTopicCtxRejectUnsupportedSessionController(t *testing.T) {
	ctrl := &recordingSessionController{err: usmqtt.ErrSessionControlUnsupported}
	ctx := &topicCtx{
		client:     &mqtt.Client{ID: "client-3"},
		controller: ctrl,
		ctx:        context.Background(),
	}

	err := ctx.Reject("blocked")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrTopicCtxSessionControlUnsupported) {
		t.Fatalf("error mismatch: got=%v", err)
	}
}
