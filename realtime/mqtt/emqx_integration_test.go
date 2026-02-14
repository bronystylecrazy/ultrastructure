//go:build integration

package mqtt

import (
	"strings"
	"testing"
	"time"

	"github.com/bronystylecrazy/ultrastructure/testkit"
)

func TestEMQXSessionControllerWithTestcontainers(t *testing.T) {
	testkit.RequireIntegration(t)

	suite := testkit.NewSuite(t)
	emqx := suite.StartEMQX(testkit.EMQXOptions{
		StartupTimeout: 2 * time.Minute,
	})

	ctrl, err := NewEMQXSessionController(EMQXSessionControllerConfig{
		Endpoint: emqx.DashboardEndpoint,
		Username: emqx.Username,
		Password: emqx.Password,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewEMQXSessionController: %v", err)
	}

	// Kickout for a non-existent client should reach authenticated EMQX API and return a non-2xx business error.
	err = ctrl.DisconnectClient(suite.Context(), "non-existent-client", "integration")
	if err == nil {
		t.Fatal("expected emqx kickout error for non-existent client, got nil")
	}
	if strings.Contains(err.Error(), "status=401") || strings.Contains(err.Error(), "status=403") {
		t.Fatalf("unexpected auth failure from emqx api: %v", err)
	}
}
