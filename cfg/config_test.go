package cfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestDecodeStructReadsFromEnv(t *testing.T) {
	t.Setenv("DB_DATASOURCE", "postgres://env")
	t.Setenv("DB_MIGRATE", "true")

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	type dbConfig struct {
		Datasource string `mapstructure:"datasource"`
		Migrate    bool   `mapstructure:"migrate"`
	}

	var out dbConfig
	if err := decode(v, "db", &out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.Datasource != "postgres://env" {
		t.Fatalf("expected datasource from env, got %q", out.Datasource)
	}
	if !out.Migrate {
		t.Fatalf("expected migrate from env to be true")
	}
}

func TestDecodeStructReadsFromToml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := []byte("[db]\ndatasource = \"postgres://file\"\nmigrate = true\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg := configState{
		sourceFile:   path,
		configType:   "toml",
		automaticEnv: true,
		keyReplacer:  strings.NewReplacer(".", "_", "-", "_"),
		optional:     false,
	}
	v, err := load(cfg, path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	type dbConfig struct {
		Datasource string `mapstructure:"datasource"`
		Migrate    bool   `mapstructure:"migrate"`
	}

	var out dbConfig
	if err := decode(v, "db", &out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.Datasource != "postgres://file" {
		t.Fatalf("expected datasource from file, got %q", out.Datasource)
	}
	if !out.Migrate {
		t.Fatalf("expected migrate from file to be true")
	}
}
