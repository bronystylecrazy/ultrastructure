package testkit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

// IntegrationEnv is an optional environment variable gate for integration tests.
const IntegrationEnv = "RUN_INTEGRATION_TESTS"

// RequireIntegration skips the test unless RUN_INTEGRATION_TESTS is set to a truthy value.
func RequireIntegration(t testing.TB) {
	t.Helper()

	value := strings.TrimSpace(strings.ToLower(os.Getenv(IntegrationEnv)))
	if value == "1" || value == "true" || value == "yes" || value == "on" {
		return
	}

	t.Skipf("skipping integration test; set %s=1 to run", IntegrationEnv)
}

// RequireDocker skips the test when Docker is unavailable.
func RequireDocker(t testing.TB) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("docker is not available: %v", r)
		}
	}()

	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		t.Skipf("docker is not available: %v", err)
		return
	}
	_ = provider.Close()
}

// Suite is a test helper that owns the context and lifecycle of containers started during a test.
type Suite struct {
	t   testing.TB
	ctx context.Context
}

// NewSuite creates an integration test suite and verifies Docker availability.
func NewSuite(t testing.TB) *Suite {
	t.Helper()

	RequireDocker(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return &Suite{
		t:   t,
		ctx: ctx,
	}
}

// Context returns the suite context used to start and inspect containers.
func (s *Suite) Context() context.Context {
	return s.ctx
}

// StartContainer starts a test container and registers automatic cleanup.
func (s *Suite) StartContainer(req testcontainers.ContainerRequest) testcontainers.Container {
	s.t.Helper()

	c, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		s.t.Fatalf("start container %q: %v", req.Image, err)
	}

	s.t.Cleanup(func() {
		termCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_ = c.Terminate(termCtx)
	})

	return c
}

// Host returns the host where container ports are exposed.
func (s *Suite) Host(c testcontainers.Container) string {
	s.t.Helper()

	host, err := c.Host(s.ctx)
	if err != nil {
		s.t.Fatalf("container host: %v", err)
	}
	return host
}

// MappedPort returns the mapped host port for a container port.
func (s *Suite) MappedPort(c testcontainers.Container, port string) string {
	s.t.Helper()

	mapped, err := c.MappedPort(s.ctx, nat.Port(port))
	if err != nil {
		s.t.Fatalf("mapped port for %s: %v", port, err)
	}
	return mapped.Port()
}

// Endpoint builds a "<scheme>://<host>:<mapped-port>" endpoint for a container port.
func (s *Suite) Endpoint(c testcontainers.Container, scheme string, port string) string {
	s.t.Helper()

	return fmt.Sprintf("%s://%s:%s", scheme, s.Host(c), s.MappedPort(c, port))
}
