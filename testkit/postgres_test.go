package testkit

import (
	"strings"
	"testing"
	"time"
)

func TestWithPostgresDefaults(t *testing.T) {
	opts := withPostgresDefaults(PostgresOptions{})

	if opts.Image != defaultPostgresImage {
		t.Fatalf("unexpected image: got=%q want=%q", opts.Image, defaultPostgresImage)
	}
	if opts.Username != defaultPostgresUser {
		t.Fatalf("unexpected username: got=%q want=%q", opts.Username, defaultPostgresUser)
	}
	if opts.Password != defaultPostgresPassword {
		t.Fatalf("unexpected password: got=%q want=%q", opts.Password, defaultPostgresPassword)
	}
	if opts.Database != defaultPostgresDB {
		t.Fatalf("unexpected database: got=%q want=%q", opts.Database, defaultPostgresDB)
	}
	if opts.StartupTimeout != defaultPostgresStartupTimeout {
		t.Fatalf("unexpected startup timeout: got=%s want=%s", opts.StartupTimeout, defaultPostgresStartupTimeout)
	}
}

func TestWithPostgresDefaultsKeepsProvidedValues(t *testing.T) {
	in := PostgresOptions{
		Image:          "postgres:latest",
		Username:       "u",
		Password:       "p",
		Database:       "d",
		StartupTimeout: 10 * time.Second,
	}
	opts := withPostgresDefaults(in)
	if opts != in {
		t.Fatalf("options mutated unexpectedly: got=%+v want=%+v", opts, in)
	}
}

func TestPostgresContainerURL(t *testing.T) {
	c := PostgresContainer{
		Host:     "127.0.0.1",
		Port:     "15432",
		Username: "demo user",
		Password: "p@ss",
		Database: "db",
	}
	got := c.URL()

	if !strings.HasPrefix(got, "postgres://demo+user:p%40ss@127.0.0.1:15432/db?sslmode=disable") {
		t.Fatalf("unexpected URL: %s", got)
	}
}
