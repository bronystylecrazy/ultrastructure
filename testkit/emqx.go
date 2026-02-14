package testkit

import (
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultEMQXImage          = "emqx/emqx:5.8.4"
	defaultEMQXDashboardPort  = "18083/tcp"
	defaultEMQXMQTTPort       = "1883/tcp"
	defaultEMQXDashboardUser  = "admin"
	defaultEMQXDashboardPass  = "public"
	defaultEMQXStartupTimeout = 2 * time.Minute
)

// EMQXOptions controls how the EMQX container is started.
type EMQXOptions struct {
	Image          string
	Username       string
	Password       string
	StartupTimeout time.Duration
}

// EMQXContainer contains connection details for a started EMQX container.
type EMQXContainer struct {
	Container         testcontainers.Container
	Host              string
	DashboardPort     string
	MQTTPort          string
	DashboardEndpoint string
	Username          string
	Password          string
}

// StartEMQX starts an EMQX container configured for dashboard API authentication.
func (s *Suite) StartEMQX(opts EMQXOptions) *EMQXContainer {
	s.t.Helper()

	opts = withEMQXDefaults(opts)

	c := s.StartContainer(testcontainers.ContainerRequest{
		Image:        opts.Image,
		ExposedPorts: []string{defaultEMQXDashboardPort, defaultEMQXMQTTPort},
		Env: map[string]string{
			"EMQX_DASHBOARD__DEFAULT_USERNAME": opts.Username,
			"EMQX_DASHBOARD__DEFAULT_PASSWORD": opts.Password,
		},
		WaitingFor: wait.ForListeningPort(defaultEMQXDashboardPort).WithStartupTimeout(opts.StartupTimeout),
	})

	host := s.Host(c)
	dashboardPort := s.MappedPort(c, defaultEMQXDashboardPort)
	mqttPort := s.MappedPort(c, defaultEMQXMQTTPort)

	return &EMQXContainer{
		Container:         c,
		Host:              host,
		DashboardPort:     dashboardPort,
		MQTTPort:          mqttPort,
		DashboardEndpoint: fmt.Sprintf("http://%s:%s", host, dashboardPort),
		Username:          opts.Username,
		Password:          opts.Password,
	}
}

func withEMQXDefaults(opts EMQXOptions) EMQXOptions {
	if opts.Image == "" {
		opts.Image = defaultEMQXImage
	}
	if opts.Username == "" {
		opts.Username = defaultEMQXDashboardUser
	}
	if opts.Password == "" {
		opts.Password = defaultEMQXDashboardPass
	}
	if opts.StartupTimeout <= 0 {
		opts.StartupTimeout = defaultEMQXStartupTimeout
	}
	return opts
}
