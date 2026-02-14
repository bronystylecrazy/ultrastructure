package mqtt_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
)

func TestEMQXSessionControllerDisconnectClientSuccessBasicAuth(t *testing.T) {
	var gotPath string
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ctrl, err := usmqtt.NewEMQXSessionController(usmqtt.EMQXSessionControllerConfig{
		Endpoint: srv.URL,
		Username: "api-user",
		Password: "api-pass",
		Timeout:  2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewEMQXSessionController: %v", err)
	}

	if err := ctrl.DisconnectClient(context.Background(), "client-1", "x"); err != nil {
		t.Fatalf("DisconnectClient: %v", err)
	}
	if gotPath != "/api/v5/clients/client-1/kickout" {
		t.Fatalf("path mismatch: got=%q", gotPath)
	}
	if gotAuth == "" {
		t.Fatal("expected authorization header, got empty")
	}
}

func TestEMQXSessionControllerDisconnectClientSuccessBearer(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctrl, err := usmqtt.NewEMQXSessionController(usmqtt.EMQXSessionControllerConfig{
		Endpoint:    srv.URL,
		BearerToken: "token-123",
	})
	if err != nil {
		t.Fatalf("NewEMQXSessionController: %v", err)
	}

	if err := ctrl.DisconnectClient(context.Background(), "client-2", "x"); err != nil {
		t.Fatalf("DisconnectClient: %v", err)
	}
	if gotAuth != "Bearer token-123" {
		t.Fatalf("authorization mismatch: got=%q", gotAuth)
	}
}

func TestEMQXSessionControllerDisconnectClientFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"BAD_REQUEST"}`))
	}))
	defer srv.Close()

	ctrl, err := usmqtt.NewEMQXSessionController(usmqtt.EMQXSessionControllerConfig{Endpoint: srv.URL})
	if err != nil {
		t.Fatalf("NewEMQXSessionController: %v", err)
	}

	if err := ctrl.DisconnectClient(context.Background(), "client-3", "x"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
