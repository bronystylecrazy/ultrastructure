package cfg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type option interface {
	apply(*configState)
}

type optionFunc func(*configState)

func (f optionFunc) apply(s *configState) { f(s) }

type watchOption interface {
	applyWatch(*watchState)
}

type watchOptionFunc func(*watchState)

func (f watchOptionFunc) applyWatch(s *watchState) { f(s) }

type disableEnvOverride struct{}
type disableOptional struct{}

type watchAll struct {
	cfg watchState
}

func (w watchAll) applyWatch(s *watchState) {
	s.enabled = true
	if w.cfg.debounce > 0 {
		s.debounce = w.cfg.debounce
	}
	if len(w.cfg.keys) > 0 {
		s.keys = append(s.keys, w.cfg.keys...)
	}
}

type configState struct {
	sourceFile   string
	configType   string
	configName   string
	configPaths  []string
	envPrefix    string
	keyReplacer  *strings.Replacer
	automaticEnv bool
	optional     bool
	defaults     map[string]any
	hooks        []func(*viper.Viper) error
}

type watchState struct {
	enabled  bool
	debounce time.Duration
	keys     []string
}

type configNode[T any] struct {
	pathOrKey string
	opts      []any
}

type configBindNode[T any] struct {
	key  string
	opts []any
}

func Config[T any](pathOrKey string, opts ...any) di.Node {
	return configNode[T]{pathOrKey: pathOrKey, opts: withDefaults(opts)}
}

func ConfigBind[T any](key string, opts ...any) di.Node {
	return configBindNode[T]{key: key, opts: withDefaults(opts)}
}

func (n configNode[T]) Build() (fx.Option, error) {
	cfg, watch, err := parse(n.opts)
	if err != nil {
		return nil, err
	}
	path, key := resolveTarget(n.pathOrKey)
	if path == "" {
		path = cfg.sourceFile
	}
	if path == "" && !hasSource(cfg) {
		return nil, fmt.Errorf("config source not provided")
	}
	constructor := func() (T, error) {
		var out T
		v, err := load(cfg, path)
		if err != nil {
			return out, err
		}
		return out, decode(v, key, &out)
	}
	out := []fx.Option{fx.Provide(constructor)}
	if watch.enabled {
		out = append(out, buildWatchOption(cfg, watch, path, key))
	}
	return fx.Options(out...), nil
}

func (n configBindNode[T]) Build() (fx.Option, error) {
	cfg, watch, err := parse(n.opts)
	if err != nil {
		return nil, err
	}
	path := cfg.sourceFile
	if path == "" && !hasSource(cfg) {
		return nil, fmt.Errorf("config source not provided")
	}
	constructor := func() (T, error) {
		var out T
		v, err := load(cfg, path)
		if err != nil {
			return out, err
		}
		return out, decode(v, n.key, &out)
	}
	out := []fx.Option{fx.Provide(constructor)}
	if watch.enabled {
		out = append(out, buildWatchOption(cfg, watch, path, n.key))
	}
	return fx.Options(out...), nil
}

func WithSourceFile(path string) any {
	return optionFunc(func(s *configState) { s.sourceFile = path })
}

func WithType(kind string) any {
	return optionFunc(func(s *configState) { s.configType = kind })
}

func WithName(name string) any {
	return optionFunc(func(s *configState) { s.configName = name })
}

func WithPath(path string) any {
	return optionFunc(func(s *configState) { s.configPaths = append(s.configPaths, path) })
}

func WithOptional() any {
	return optionFunc(func(s *configState) { s.optional = true })
}

func WithEnvPrefix(prefix string) any {
	return optionFunc(func(s *configState) { s.envPrefix = prefix })
}

func WithEnvKeyReplacer(replacer *strings.Replacer) any {
	return optionFunc(func(s *configState) { s.keyReplacer = replacer })
}

func WithEnvOverride(prefix ...string) any {
	return optionFunc(func(s *configState) {
		if len(prefix) > 0 {
			s.envPrefix = prefix[0]
		}
		s.automaticEnv = true
		if s.keyReplacer == nil {
			s.keyReplacer = strings.NewReplacer(".", "_", "-", "_")
		}
	})
}

func WithAutomaticEnv() any {
	return optionFunc(func(s *configState) { s.automaticEnv = true })
}

func WithNoEnv() any {
	return optionFunc(func(s *configState) {
		s.automaticEnv = false
		s.envPrefix = ""
	})
}

func WithDefault(key string, value any) any {
	return optionFunc(func(s *configState) {
		if s.defaults == nil {
			s.defaults = map[string]any{}
		}
		s.defaults[key] = value
	})
}

func WithViper(fn func(*viper.Viper) error) any {
	return optionFunc(func(s *configState) {
		if fn != nil {
			s.hooks = append(s.hooks, fn)
		}
	})
}

func WithWatch(opts ...watchOption) watchOption {
	cfg := watchState{enabled: true}
	for _, opt := range opts {
		if opt != nil {
			opt.applyWatch(&cfg)
		}
	}
	return watchAll{cfg: cfg}
}

func WithWatchDebounce(d time.Duration) watchOption {
	return watchOptionFunc(func(s *watchState) { s.debounce = d })
}

func WithWatchKeys(keys ...string) watchOption {
	return watchOptionFunc(func(s *watchState) { s.keys = append(s.keys, keys...) })
}

func WithDisableWatch() watchOption {
	return watchOptionFunc(func(s *watchState) { s.enabled = false })
}

func WithDisableEnvOverride() any {
	return disableEnvOverride{}
}

func WithDisableOptional() any {
	return disableOptional{}
}

func withDefaults(opts []any) []any {
	envOverride := true
	optional := true
	filtered := make([]any, 0, len(opts))
	for _, opt := range opts {
		switch opt.(type) {
		case disableEnvOverride:
			envOverride = false
		case disableOptional:
			optional = false
		default:
			filtered = append(filtered, opt)
		}
	}
	out := make([]any, 0, len(filtered)+2)
	if envOverride {
		out = append(out, WithEnvOverride())
	}
	if optional {
		out = append(out, WithOptional())
	}
	out = append(out, filtered...)
	return out
}

func parse(opts []any) (configState, watchState, error) {
	cfg := configState{
		automaticEnv: true,
		keyReplacer:  strings.NewReplacer(".", "_", "-", "_"),
	}
	watch := watchState{}
	for _, opt := range opts {
		switch v := opt.(type) {
		case nil:
		case option:
			v.apply(&cfg)
		case watchOption:
			v.applyWatch(&watch)
		default:
			return cfg, watch, fmt.Errorf("unsupported config option type %T", opt)
		}
	}
	return cfg, watch, nil
}

func resolveTarget(pathOrKey string) (string, string) {
	pathOrKey = strings.TrimSpace(pathOrKey)
	if pathOrKey == "" {
		return "", ""
	}
	lower := strings.ToLower(pathOrKey)
	ext := filepath.Ext(lower)
	if strings.Contains(pathOrKey, "/") || strings.Contains(pathOrKey, "\\") {
		return pathOrKey, ""
	}
	switch ext {
	case ".toml", ".yaml", ".yml", ".json":
		return pathOrKey, ""
	default:
		return "", pathOrKey
	}
}

func hasSource(cfg configState) bool {
	return cfg.sourceFile != "" || cfg.configName != "" || cfg.configType != "" || len(cfg.configPaths) > 0
}

func load(cfg configState, path string) (*viper.Viper, error) {
	v := viper.New()
	if cfg.envPrefix != "" {
		v.SetEnvPrefix(cfg.envPrefix)
	}
	if cfg.keyReplacer != nil {
		v.SetEnvKeyReplacer(cfg.keyReplacer)
	}
	if cfg.automaticEnv {
		v.AutomaticEnv()
	}
	if path != "" {
		v.SetConfigFile(path)
	}
	if cfg.configType != "" {
		v.SetConfigType(cfg.configType)
	}
	if cfg.configName != "" {
		v.SetConfigName(cfg.configName)
	}
	for _, p := range cfg.configPaths {
		v.AddConfigPath(p)
	}
	for k, val := range cfg.defaults {
		v.SetDefault(k, val)
	}
	for _, hook := range cfg.hooks {
		if err := hook(v); err != nil {
			return nil, err
		}
	}
	if path != "" || cfg.configName != "" {
		if err := v.ReadInConfig(); err != nil {
			var nf viper.ConfigFileNotFoundError
			if cfg.optional && (errors.As(err, &nf) || errors.Is(err, os.ErrNotExist)) {
				return v, nil
			}
			if path != "" {
				if cleaned, ok := sanitize(path); ok {
					if cfg.configType == "" {
						if ext := strings.TrimPrefix(filepath.Ext(path), "."); ext != "" {
							v.SetConfigType(ext)
						}
					}
					if rerr := v.ReadConfig(bytes.NewReader(cleaned)); rerr == nil {
						return v, nil
					}
				}
			}
			return nil, err
		}
	}
	return v, nil
}

func sanitize(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	changed := false
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if i+2 < len(data) && data[i] == 0xEF && data[i+1] == 0xBB && data[i+2] == 0xBF {
			i += 2
			changed = true
			continue
		}
		if i+2 < len(data) && data[i] == 0xE2 && data[i+1] == 0x80 && data[i+2] == 0x8B {
			i += 2
			changed = true
			continue
		}
		out = append(out, data[i])
	}
	if changed {
		return out, true
	}
	return nil, false
}

func decode(v *viper.Viper, key string, out any) error {
	t := reflect.TypeOf(out)
	if t == nil || t.Kind() != reflect.Pointer {
		return fmt.Errorf("config target must be a pointer")
	}
	elem := t.Elem()
	if elem.Kind() != reflect.Struct {
		if key == "" {
			return v.Unmarshal(out)
		}
		return v.UnmarshalKey(key, out)
	}
	if key == "" {
		return v.Unmarshal(out, viper.DecodeHook(
			mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.StringToSliceHookFunc(","),
			),
		))
	}
	return v.UnmarshalKey(key, out, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	))
}

func buildWatchOption(cfg configState, watch watchState, path string, key string) fx.Option {
	if watch.debounce <= 0 {
		watch.debounce = 200 * time.Millisecond
	}
	keys := watch.keys
	if len(keys) == 0 && key != "" {
		keys = []string{key}
	}
	return fx.Invoke(fx.Annotate(func(logger *zap.Logger) {
		if path == "" {
			path = cfg.sourceFile
		}
		if path == "" && !hasSource(cfg) {
			return
		}
		v, err := load(cfg, path)
		if err != nil {
			return
		}
		installWatch(v, keys, watch.debounce, logger)
	}, fx.ParamTags(`optional:"true"`)))
}

func installWatch(v *viper.Viper, keys []string, debounce time.Duration, logger *zap.Logger) {
	last := snapshot(v, keys)
	timer := (*time.Timer)(nil)
	send := func() {
		if logger != nil {
			logger.Info("config changed")
		}
	}
	v.OnConfigChange(func(_ fsnotify.Event) {
		next := snapshot(v, keys)
		if next == last {
			return
		}
		last = next
		if debounce <= 0 {
			send()
			return
		}
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(debounce, send)
	})
	v.WatchConfig()
}

func snapshot(v *viper.Viper, keys []string) string {
	if len(keys) == 0 {
		return fmt.Sprintf("%v", v.AllSettings())
	}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%s=%v", key, v.Get(key)))
	}
	return strings.Join(out, "|")
}
