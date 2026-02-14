package testkit

import (
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultRedisImage          = "redis:7-alpine"
	defaultRedisPort           = "6379/tcp"
	defaultRedisStartupTimeout = 90 * time.Second
)

// RedisOptions controls how the Redis container is started.
type RedisOptions struct {
	Image          string
	Password       string
	StartupTimeout time.Duration
}

// RedisContainer contains connection details for a started Redis container.
type RedisContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	Password  string
}

// StartRedis starts a Redis container and returns its connection details.
func (s *Suite) StartRedis(opts RedisOptions) *RedisContainer {
	s.t.Helper()

	opts = withRedisDefaults(opts)

	req := testcontainers.ContainerRequest{
		Image:        opts.Image,
		ExposedPorts: []string{defaultRedisPort},
		WaitingFor: wait.ForListeningPort(defaultRedisPort).
			WithStartupTimeout(opts.StartupTimeout),
	}

	if opts.Password != "" {
		req.Cmd = []string{"redis-server", "--appendonly", "no", "--requirepass", opts.Password}
	}

	c := s.StartContainer(req)

	return &RedisContainer{
		Container: c,
		Host:      s.Host(c),
		Port:      s.MappedPort(c, defaultRedisPort),
		Password:  opts.Password,
	}
}

// Addr returns host:port for Redis clients.
func (c *RedisContainer) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func withRedisDefaults(opts RedisOptions) RedisOptions {
	if opts.Image == "" {
		opts.Image = defaultRedisImage
	}
	if opts.StartupTimeout <= 0 {
		opts.StartupTimeout = defaultRedisStartupTimeout
	}
	return opts
}
