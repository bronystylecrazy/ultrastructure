package testkit

import (
	"testing"
	"time"
)

func TestWithMinIODefaults(t *testing.T) {
	opts := withMinIODefaults(MinIOOptions{})

	if opts.Image != defaultMinIOImage {
		t.Fatalf("unexpected image: got=%q want=%q", opts.Image, defaultMinIOImage)
	}
	if opts.AccessKey != defaultMinIOAccessKey {
		t.Fatalf("unexpected access key: got=%q want=%q", opts.AccessKey, defaultMinIOAccessKey)
	}
	if opts.SecretKey != defaultMinIOSecretKey {
		t.Fatalf("unexpected secret key: got=%q want=%q", opts.SecretKey, defaultMinIOSecretKey)
	}
	if opts.StartupTimeout != defaultMinIOStartupTimeout {
		t.Fatalf("unexpected startup timeout: got=%s want=%s", opts.StartupTimeout, defaultMinIOStartupTimeout)
	}
}

func TestWithMinIODefaultsKeepsProvidedValues(t *testing.T) {
	in := MinIOOptions{
		Image:          "minio/minio:latest",
		AccessKey:      "a",
		SecretKey:      "b",
		StartupTimeout: 7 * time.Second,
	}
	opts := withMinIODefaults(in)
	if opts != in {
		t.Fatalf("options mutated unexpectedly: got=%+v want=%+v", opts, in)
	}
}

func TestMinIOHelpers(t *testing.T) {
	c := MinIOContainer{
		Host:     "127.0.0.1",
		APIPort:  "19000",
		Endpoint: "http://127.0.0.1:19000",
	}

	if got, want := c.HostPort(), "127.0.0.1:19000"; got != want {
		t.Fatalf("unexpected host:port: got=%q want=%q", got, want)
	}
	if got, want := c.Region(), "us-east-1"; got != want {
		t.Fatalf("unexpected region: got=%q want=%q", got, want)
	}
	if got, want := c.Scheme(), "http"; got != want {
		t.Fatalf("unexpected scheme: got=%q want=%q", got, want)
	}
	if c.EndpointURL() == nil {
		t.Fatal("endpoint URL is nil")
	}
}
