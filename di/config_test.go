package di

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type testAppConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

type testDbConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type testConfig struct {
	App testAppConfig `mapstructure:"app"`
	DB  testDbConfig  `mapstructure:"db"`
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestConfigDefaultsMerge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	writeFile(t, path, "[app]\nport = 9000\n")

	ch := make(chan testAppConfig, 1)
	app := fx.New(
		App(
			Config[testAppConfig](
				"app",
				ConfigType("toml"),
				ConfigPath(dir),
				ConfigName("config"),
				ConfigDefault("app.name", "test"),
			),
			Invoke(func(cfg testAppConfig) {
				ch <- cfg
			}),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	select {
	case cfg := <-ch:
		if cfg.Name != "test" || cfg.Port != 9000 {
			t.Fatalf("unexpected config: %+v", cfg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for config")
	}
}

func TestConfigEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	writeFile(t, path, "[app]\nname = \"file\"\nport = 9000\n")

	t.Setenv("APP_NAME", "env")
	ch := make(chan testAppConfig, 1)
	app := fx.New(
		App(
			Config[testAppConfig](
				"app",
				ConfigType("toml"),
				ConfigPath(dir),
				ConfigName("config"),
			),
			Invoke(func(cfg testAppConfig) {
				ch <- cfg
			}),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	select {
	case cfg := <-ch:
		if cfg.Name != "env" {
			t.Fatalf("expected env override, got %+v", cfg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for config")
	}
}

func TestConfigBindSharedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	writeFile(t, path, "[app]\nname = \"demo\"\nport = 8080\n\n[db]\nhost = \"localhost\"\nport = 5432\n")

	appCh := make(chan testAppConfig, 1)
	dbCh := make(chan testDbConfig, 1)
	app := fx.New(
		App(
			ConfigFile(path, ConfigType("toml")),
			Config[testAppConfig]("app"),
			Config[testDbConfig]("db"),
			Invoke(func(cfg testAppConfig) { appCh <- cfg }),
			Invoke(func(cfg testDbConfig) { dbCh <- cfg }),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	select {
	case cfg := <-appCh:
		if cfg.Name != "demo" || cfg.Port != 8080 {
			t.Fatalf("unexpected app config: %+v", cfg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for app config")
	}
	select {
	case cfg := <-dbCh:
		if cfg.Host != "localhost" || cfg.Port != 5432 {
			t.Fatalf("unexpected db config: %+v", cfg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for db config")
	}
}

func TestConfigBindWithOptions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	writeFile(t, path, "[app]\nname = \"demo\"\nport = 8080\n")

	ch := make(chan testAppConfig, 1)
	app := fx.New(
		App(
			Config[testAppConfig](
				"app",
				ConfigType("toml"),
				ConfigPath(dir),
				ConfigName("config"),
			),
			Invoke(func(cfg testAppConfig) { ch <- cfg }),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	select {
	case cfg := <-ch:
		if cfg.Name != "demo" || cfg.Port != 8080 {
			t.Fatalf("unexpected config: %+v", cfg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for config")
	}
}

func TestConfigWatchRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	writeFile(t, path, "[app]\nname = \"demo\"\nport = 8080\n")

	restart := make(restartSignal, 1)
	app := fx.New(
		fx.Supply(restart),
		App(
			Provide(zap.NewNop),
			ConfigFile(path, ConfigType("toml")),
			Config[testAppConfig]("app", ConfigWatch()),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	writeFile(t, path, "[app]\nname = \"changed\"\nport = 8081\n")

	select {
	case <-restart:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatal("expected restart signal")
	}
}
