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

// SetBaseDir must anchor a RELATIVE source file away from the process cwd: an
// installed CLI reads its own config root, never whatever config.toml the
// current project carries — including one that would not even parse.
func TestSetBaseDirAnchorsRelativeSourceFiles(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "config.toml"), []byte("[db]\ndatasource = \"postgres://base\"\n"), 0o644); err != nil {
		t.Fatalf("write base config: %v", err)
	}
	cwd := t.TempDir()
	// A malformed cwd file proves it is not even opened.
	if err := os.WriteFile(filepath.Join(cwd, "config.toml"), []byte("not [valid toml %%\n"), 0o644); err != nil {
		t.Fatalf("write cwd config: %v", err)
	}
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	SetBaseDir(base)
	defer SetBaseDir("")

	cfg, _, err := parse([]any{WithSourceFile("config.toml"), WithType("toml"), WithOptional()})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	v, err := load(cfg, "config.toml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := v.GetString("db.datasource"); got != "postgres://base" {
		t.Fatalf("db.datasource = %q, want the base dir's value", got)
	}

	// An absolute path is untouched by the base dir.
	abs := filepath.Join(cwd, "config.toml")
	if got := resolveSourcePath(abs); got != abs {
		t.Fatalf("resolveSourcePath(%q) = %q, want unchanged", abs, got)
	}
}
