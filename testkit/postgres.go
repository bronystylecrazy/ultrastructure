package testkit

import (
	"fmt"
	"net/url"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultPostgresImage          = "postgres:17-alpine"
	defaultPostgresPort           = "5432/tcp"
	defaultPostgresUser           = "postgres"
	defaultPostgresPassword       = "postgres"
	defaultPostgresDB             = "postgres"
	defaultPostgresStartupTimeout = 2 * time.Minute
)

// PostgresOptions controls how the PostgreSQL container is started.
type PostgresOptions struct {
	Image          string
	Username       string
	Password       string
	Database       string
	StartupTimeout time.Duration
}

// PostgresContainer contains connection details for a started PostgreSQL container.
type PostgresContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	Username  string
	Password  string
	Database  string
}

// StartPostgres starts a PostgreSQL container and returns its connection details.
func (s *Suite) StartPostgres(opts PostgresOptions) *PostgresContainer {
	s.t.Helper()

	opts = withPostgresDefaults(opts)

	c := s.StartContainer(testcontainers.ContainerRequest{
		Image:        opts.Image,
		ExposedPorts: []string{defaultPostgresPort},
		Env: map[string]string{
			"POSTGRES_USER":     opts.Username,
			"POSTGRES_PASSWORD": opts.Password,
			"POSTGRES_DB":       opts.Database,
		},
		WaitingFor: wait.ForListeningPort(defaultPostgresPort).WithStartupTimeout(opts.StartupTimeout),
	})

	return &PostgresContainer{
		Container: c,
		Host:      s.Host(c),
		Port:      s.MappedPort(c, defaultPostgresPort),
		Username:  opts.Username,
		Password:  opts.Password,
		Database:  opts.Database,
	}
}

// URL returns a postgres:// URL with sslmode disabled, suitable for local integration tests.
func (c *PostgresContainer) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		url.QueryEscape(c.Username),
		url.QueryEscape(c.Password),
		c.Host,
		c.Port,
		url.PathEscape(c.Database),
	)
}

func withPostgresDefaults(opts PostgresOptions) PostgresOptions {
	if opts.Image == "" {
		opts.Image = defaultPostgresImage
	}
	if opts.Username == "" {
		opts.Username = defaultPostgresUser
	}
	if opts.Password == "" {
		opts.Password = defaultPostgresPassword
	}
	if opts.Database == "" {
		opts.Database = defaultPostgresDB
	}
	if opts.StartupTimeout <= 0 {
		opts.StartupTimeout = defaultPostgresStartupTimeout
	}
	return opts
}
