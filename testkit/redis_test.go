package testkit

import (
	"testing"
	"time"
)

func TestWithRedisDefaults(t *testing.T) {
	opts := withRedisDefaults(RedisOptions{})

	if opts.Image != defaultRedisImage {
		t.Fatalf("unexpected image: got=%q want=%q", opts.Image, defaultRedisImage)
	}
	if opts.StartupTimeout != defaultRedisStartupTimeout {
		t.Fatalf("unexpected startup timeout: got=%s want=%s", opts.StartupTimeout, defaultRedisStartupTimeout)
	}
}

func TestWithRedisDefaultsKeepsProvidedValues(t *testing.T) {
	in := RedisOptions{
		Image:          "redis:custom",
		Password:       "secret",
		StartupTimeout: 5 * time.Second,
	}
	opts := withRedisDefaults(in)
	if opts != in {
		t.Fatalf("options mutated unexpectedly: got=%+v want=%+v", opts, in)
	}
}

func TestRedisAddr(t *testing.T) {
	c := RedisContainer{Host: "localhost", Port: "63791"}
	if got, want := c.Addr(), "localhost:63791"; got != want {
		t.Fatalf("unexpected addr: got=%q want=%q", got, want)
	}
}
