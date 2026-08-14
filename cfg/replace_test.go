package cfg_test

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type probeConfig struct {
	Name string `mapstructure:"name"`
}

func resolveProbe(t *testing.T, nodes ...any) probeConfig {
	t.Helper()
	var got probeConfig
	base := []any{cfg.Config[probeConfig]("probe", cfg.WithSourceFile("config.toml"), cfg.WithType("toml"))}
	app := fxtest.New(t, fx.Options(di.App(append(base, append(nodes, di.Populate(&got))...)...).Build()))
	app.RequireStart().RequireStop()
	return got
}

// Configuration is provided straight to fx, so di has to be told which type a
// config node exports before a replacement can reach it. Tests replace a whole
// config value rather than write a file.
func TestReplaceOverridesLoadedConfig(t *testing.T) {
	if got := resolveProbe(t, di.Replace(probeConfig{Name: "replaced"})); got.Name != "replaced" {
		t.Fatalf("got %#v", got)
	}
}

func TestConfigStandsWhenNothingReplacesIt(t *testing.T) {
	if got := resolveProbe(t); got.Name != "" {
		t.Fatalf("expected the loaded value, got %#v", got)
	}
}

func TestDefaultYieldsToLoadedConfig(t *testing.T) {
	if got := resolveProbe(t, di.Default(probeConfig{Name: "fallback"})); got.Name != "" {
		t.Fatalf("the config node should outrank a default, got %#v", got)
	}
}
