package testkit

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultMinIOImage          = "minio/minio:RELEASE.2025-07-23T15-54-02Z"
	defaultMinIOAPIPort        = "9000/tcp"
	defaultMinIOConsolePort    = "9001/tcp"
	defaultMinIOAccessKey      = "minioadmin"
	defaultMinIOSecretKey      = "minioadmin"
	defaultMinIOStartupTimeout = 2 * time.Minute
)

// MinIOOptions controls how the MinIO container is started.
type MinIOOptions struct {
	Image          string
	AccessKey      string
	SecretKey      string
	StartupTimeout time.Duration
}

// MinIOContainer contains connection details for a started MinIO container.
type MinIOContainer struct {
	Container       testcontainers.Container
	Host            string
	APIPort         string
	ConsolePort     string
	Endpoint        string
	ConsoleEndpoint string
	AccessKey       string
	SecretKey       string
}

// StartMinIO starts a MinIO container and returns API and console connection details.
func (s *Suite) StartMinIO(opts MinIOOptions) *MinIOContainer {
	s.t.Helper()

	opts = withMinIODefaults(opts)

	c := s.StartContainer(testcontainers.ContainerRequest{
		Image:        opts.Image,
		ExposedPorts: []string{defaultMinIOAPIPort, defaultMinIOConsolePort},
		Env: map[string]string{
			"MINIO_ROOT_USER":     opts.AccessKey,
			"MINIO_ROOT_PASSWORD": opts.SecretKey,
		},
		Cmd: []string{"server", "/data", "--console-address", ":9001"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(defaultMinIOAPIPort),
			wait.ForHTTP("/minio/health/ready").WithPort(defaultMinIOAPIPort),
		).WithStartupTimeout(opts.StartupTimeout),
	})

	host := s.Host(c)
	apiPort := s.MappedPort(c, defaultMinIOAPIPort)
	consolePort := s.MappedPort(c, defaultMinIOConsolePort)

	return &MinIOContainer{
		Container:       c,
		Host:            host,
		APIPort:         apiPort,
		ConsolePort:     consolePort,
		Endpoint:        fmt.Sprintf("http://%s:%s", host, apiPort),
		ConsoleEndpoint: fmt.Sprintf("http://%s:%s", host, consolePort),
		AccessKey:       opts.AccessKey,
		SecretKey:       opts.SecretKey,
	}
}

// HostPort returns "host:port" without scheme for S3 clients expecting a custom endpoint resolver.
func (c *MinIOContainer) HostPort() string {
	return fmt.Sprintf("%s:%s", c.Host, c.APIPort)
}

// Region returns the default MinIO region value when none is explicitly configured.
func (c *MinIOContainer) Region() string {
	return "us-east-1"
}

// EndpointURL returns the endpoint URL as a parsed value.
func (c *MinIOContainer) EndpointURL() *url.URL {
	u, _ := url.Parse(c.Endpoint)
	return u
}

// Scheme returns endpoint scheme, defaulting to "http".
func (c *MinIOContainer) Scheme() string {
	if strings.HasPrefix(c.Endpoint, "https://") {
		return "https"
	}
	return "http"
}

func withMinIODefaults(opts MinIOOptions) MinIOOptions {
	if opts.Image == "" {
		opts.Image = defaultMinIOImage
	}
	if opts.AccessKey == "" {
		opts.AccessKey = defaultMinIOAccessKey
	}
	if opts.SecretKey == "" {
		opts.SecretKey = defaultMinIOSecretKey
	}
	if opts.StartupTimeout <= 0 {
		opts.StartupTimeout = defaultMinIOStartupTimeout
	}
	return opts
}
