package testkit

import (
	"testing"
	"time"
)

func TestWithEMQXDefaults(t *testing.T) {
	opts := withEMQXDefaults(EMQXOptions{})

	if opts.Image != defaultEMQXImage {
		t.Fatalf("unexpected image: got=%q want=%q", opts.Image, defaultEMQXImage)
	}
	if opts.Username != defaultEMQXDashboardUser {
		t.Fatalf("unexpected username: got=%q want=%q", opts.Username, defaultEMQXDashboardUser)
	}
	if opts.Password != defaultEMQXDashboardPass {
		t.Fatalf("unexpected password: got=%q want=%q", opts.Password, defaultEMQXDashboardPass)
	}
	if opts.StartupTimeout != defaultEMQXStartupTimeout {
		t.Fatalf("unexpected startup timeout: got=%s want=%s", opts.StartupTimeout, defaultEMQXStartupTimeout)
	}
}

func TestWithEMQXDefaultsKeepsProvidedValues(t *testing.T) {
	in := EMQXOptions{
		Image:          "emqx/emqx:custom",
		Username:       "u",
		Password:       "p",
		StartupTimeout: 15 * time.Second,
	}

	opts := withEMQXDefaults(in)
	if opts != in {
		t.Fatalf("options mutated unexpectedly: got=%+v want=%+v", opts, in)
	}
}
