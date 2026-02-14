//go:build integration

package mqtt

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestEMQXSessionControllerWithTestcontainers(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "emqx/emqx:5.8.4",
		ExposedPorts: []string{"18083/tcp", "1883/tcp"},
		Env: map[string]string{
			"EMQX_DASHBOARD__DEFAULT_USERNAME": "admin",
			"EMQX_DASHBOARD__DEFAULT_PASSWORD": "public",
		},
		WaitingFor: wait.ForListeningPort("18083/tcp").WithStartupTimeout(2 * time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start emqx container: %v", err)
	}
	defer func() { _ = c.Terminate(ctx) }()

	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	port, err := c.MappedPort(ctx, "18083/tcp")
	if err != nil {
		t.Fatalf("mapped port: %v", err)
	}

	ctrl, err := NewEMQXSessionController(EMQXSessionControllerConfig{
		Endpoint: "http://" + host + ":" + port.Port(),
		Username: "admin",
		Password: "public",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewEMQXSessionController: %v", err)
	}

	// Kickout for a non-existent client should reach authenticated EMQX API and return a non-2xx business error.
	err = ctrl.DisconnectClient(ctx, "non-existent-client", "integration")
	if err == nil {
		t.Fatal("expected emqx kickout error for non-existent client, got nil")
	}
	if strings.Contains(err.Error(), "status=401") || strings.Contains(err.Error(), "status=403") {
		t.Fatalf("unexpected auth failure from emqx api: %v", err)
	}
}
