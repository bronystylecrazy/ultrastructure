package di

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/bronystylecrazy/ultrastructure/us"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Config loads a configuration file or binds a key from a shared ConfigFile.
// If the first argument looks like a file path, it is treated as a config file path.
// Otherwise it is treated as a key and requires ConfigFile or Config* options.
func Config[T any](pathOrKey string, opts ...any) Node {
	return configNode[T]{pathOrKey: pathOrKey, opts: opts}
}

// ConfigFile loads a config file once and exposes it for ConfigBind.
func ConfigFile(path string, opts ...any) Node {
	return configFileNode{path: path, opts: opts}
}

// ConfigBind provides a sub-config by key from a shared ConfigFile.
func ConfigBind[T any](key string, opts ...any) Node {
	return configBindNode[T]{key: key, opts: opts}
}

type configNode[T any] struct {
	pathOrKey string
	opts      []any
}

func (n configNode[T]) withConfigWatch(cfg configWatchConfig) Node {
	if configHasWatchOption(n.opts) {
		return n
	}
	n.opts = append(n.opts, configWatchAll{cfg: cfg})
	return n
}

func (n configNode[T]) provideTagSets() ([]tagSet, error) {
	_, _, bindCfg, _, err := parseConfigOptionsWithWatch(n.opts)
	if err != nil {
		return nil, err
	}
	dummy := func() T {
		var out T
		return out
	}
	_, tagSets, err := buildProvideOptions(bindCfg, dummy, nil)
	if err != nil {
		return nil, err
	}
	return tagSets, nil
}

func (n configNode[T]) Build() (fx.Option, error) {
	cfg, watchCfg, bindCfg, extra, err := parseConfigOptionsWithWatch(n.opts)
	if err != nil {
		return nil, err
	}
	path, key := resolveConfigTarget(n.pathOrKey)
	constructor := func(in struct {
		fx.In
		Store *configStore `optional:"true"`
	}) (T, error) {
		var out T
		var v *viper.Viper
		if in.Store != nil && in.Store.v != nil {
			v = in.Store.v
		} else {
			if path == "" && !hasConfigSource(cfg) {
				return out, fmt.Errorf("config source not provided; add di.ConfigFile or Config* options")
			}
			loaded, err := loadViper(cfg, path)
			if err != nil {
				return out, err
			}
			v = loaded
		}
		if err := decodeConfig(v, key, &out); err != nil {
			return out, err
		}
		return out, nil
	}
	provideOpts, _, err := buildProvideOptions(bindCfg, constructor, nil)
	if err != nil {
		return nil, err
	}
	provideOpt := us.Provide(constructor, provideOpts...)
	var out []fx.Option
	out = append(out, provideOpt)
	if watchCfg.enabled {
		out = append(out, buildConfigWatchOption(watchCfg, key, cfg, path))
	}
	out = append(out, extra...)
	if len(out) == 1 {
		return out[0], nil
	}
	return fx.Options(out...), nil
}

type configStore struct {
	v *viper.Viper

	mu          sync.Mutex
	watchers    []*configWatcher
	watchActive bool
}

type configFileNode struct {
	path string
	opts []any
}

func (n configFileNode) Build() (fx.Option, error) {
	cfg, bindCfg, extra, err := parseConfigOptions(n.opts)
	if err != nil {
		return nil, err
	}
	constructor := func() (*configStore, error) {
		v, err := loadViper(cfg, n.path)
		if err != nil {
			return nil, err
		}
		return &configStore{v: v}, nil
	}
	provideOpts, _, err := buildProvideOptions(bindCfg, constructor, nil)
	if err != nil {
		return nil, err
	}
	provideOpt := us.Provide(constructor, provideOpts...)
	var out []fx.Option
	out = append(out, provideOpt)
	out = append(out, extra...)
	if len(out) == 1 {
		return out[0], nil
	}
	return fx.Options(out...), nil
}

type configBindNode[T any] struct {
	key  string
	opts []any
}

func (n configBindNode[T]) Build() (fx.Option, error) {
	cfg, bindCfg, extra, err := parseConfigOptions(n.opts)
	if err != nil {
		return nil, err
	}
	constructor := func(store *configStore) (T, error) {
		var out T
		var v *viper.Viper
		if store != nil && store.v != nil {
			v = store.v
		} else if cfg.configName != "" || cfg.configType != "" || len(cfg.configPaths) > 0 || cfg.envPrefix != "" || cfg.keyReplacer != nil || cfg.automaticEnv || cfg.optional || len(cfg.defaults) > 0 || len(cfg.hooks) > 0 {
			loaded, err := loadViper(cfg, "")
			if err != nil {
				return out, err
			}
			v = loaded
		} else {
			return out, fmt.Errorf("config source not provided; add di.ConfigFile or Config* options")
		}
		if v == nil {
			return out, fmt.Errorf("config source not available")
		}
		return out, decodeConfig(v, n.key, &out)
	}
	provideOpts, _, err := buildProvideOptions(bindCfg, constructor, nil)
	if err != nil {
		return nil, err
	}
	provideOpt := us.Provide(constructor, provideOpts...)
	var out []fx.Option
	out = append(out, provideOpt)
	out = append(out, extra...)
	if len(out) == 1 {
		return out[0], nil
	}
	return fx.Options(out...), nil
}

func (n configBindNode[T]) provideTagSets() ([]tagSet, error) {
	bindCfg, _, _, err := parseBindOptions(n.opts)
	if err != nil {
		return nil, err
	}
	dummy := func() T {
		var out T
		return out
	}
	_, tagSets, err := buildProvideOptions(bindCfg, dummy, nil)
	if err != nil {
		return nil, err
	}
	return tagSets, nil
}

// ConfigWatch enables config watching for this config or for the whole app if used at top-level.
func ConfigWatch(opts ...ConfigWatchOption) ConfigWatchOption {
	cfg := configWatchConfig{enabled: true}
	for _, opt := range opts {
		if opt != nil {
			opt.applyWatch(&cfg)
		}
	}
	return configWatchAll{cfg: cfg}
}

type ConfigWatchOption interface {
	applyWatch(*configWatchConfig)
}

type configWatchOptionFunc func(*configWatchConfig)

func (f configWatchOptionFunc) applyWatch(cfg *configWatchConfig) { f(cfg) }

type configWatchAll struct {
	cfg configWatchConfig
}

func (c configWatchAll) applyWatch(cfg *configWatchConfig) {
	if cfg.disabled {
		return
	}
	cfg.enabled = true
	if c.cfg.debounce != 0 {
		cfg.debounce = c.cfg.debounce
	}
	cfg.keys = append(cfg.keys, c.cfg.keys...)
}

func (c configWatchAll) Build() (fx.Option, error) {
	return fx.Options(), nil
}

// ConfigWatchDebounce sets a debounce duration for config change events.
func ConfigWatchDebounce(d time.Duration) ConfigWatchOption {
	return configWatchOptionFunc(func(cfg *configWatchConfig) {
		if cfg.disabled {
			return
		}
		cfg.enabled = true
		if d < 0 {
			d = 0
		}
		cfg.debounce = d
	})
}

// ConfigWatchKeys watches only specific keys (e.g. "app", "db.host").
func ConfigWatchKeys(keys ...string) ConfigWatchOption {
	return configWatchOptionFunc(func(cfg *configWatchConfig) {
		if cfg.disabled {
			return
		}
		cfg.enabled = true
		cfg.keys = append(cfg.keys, keys...)
	})
}

type configWatchConfig struct {
	enabled  bool
	disabled bool
	debounce time.Duration
	keys     []string
}

type configDisableWatch struct{}

// ConfigDisableWatch disables watching for this config.
func ConfigDisableWatch() ConfigWatchOption {
	return configDisableWatch{}
}

func (configDisableWatch) applyWatch(cfg *configWatchConfig) {
	cfg.enabled = false
	cfg.disabled = true
	cfg.keys = nil
}

func snapshotKeys(v *viper.Viper, keys []string) (string, error) {
	if v == nil {
		return "", fmt.Errorf("viper is nil")
	}
	var payload any
	if len(keys) == 0 {
		payload = v.AllSettings()
	} else {
		out := make(map[string]any, len(keys))
		for _, key := range keys {
			if key == "" {
				continue
			}
			out[key] = v.Get(key)
		}
		payload = out
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type configConfig struct {
	configType   string
	configName   string
	configPaths  []string
	envPrefix    string
	keyReplacer  *strings.Replacer
	automaticEnv bool
	optional     bool
	defaults     map[string]any
	hooks        []func(*viper.Viper) error
	err          error
}

type configOption interface {
	applyConfig(*configConfig)
}

type configOptionFunc func(*configConfig)

func (f configOptionFunc) applyConfig(cfg *configConfig) { f(cfg) }

// ConfigType sets the config type (e.g. "toml").
func ConfigType(kind string) configOption {
	return configOptionFunc(func(cfg *configConfig) {
		if cfg.err != nil {
			return
		}
		if kind == "" {
			cfg.err = fmt.Errorf("config type must not be empty")
			return
		}
		cfg.configType = kind
	})
}

// ConfigName sets the config name for Viper to search for.
func ConfigName(name string) configOption {
	return configOptionFunc(func(cfg *configConfig) {
		if cfg.err != nil {
			return
		}
		if name == "" {
			cfg.err = fmt.Errorf("config name must not be empty")
			return
		}
		cfg.configName = name
	})
}

// ConfigPath adds a search path for ConfigName.
func ConfigPath(path string) configOption {
	return configOptionFunc(func(cfg *configConfig) {
		if cfg.err != nil {
			return
		}
		if path == "" {
			cfg.err = fmt.Errorf("config path must not be empty")
			return
		}
		cfg.configPaths = append(cfg.configPaths, path)
	})
}

// ConfigOptional allows missing config files without failing.
func ConfigOptional() configOption {
	return configOptionFunc(func(cfg *configConfig) {
		cfg.optional = true
	})
}

// ConfigEnvPrefix sets the environment prefix.
func ConfigEnvPrefix(prefix string) configOption {
	return configOptionFunc(func(cfg *configConfig) {
		if cfg.err != nil {
			return
		}
		cfg.envPrefix = prefix
	})
}

// ConfigEnvKeyReplacer sets the environment key replacer.
func ConfigEnvKeyReplacer(replacer *strings.Replacer) configOption {
	return configOptionFunc(func(cfg *configConfig) {
		cfg.keyReplacer = replacer
	})
}

// ConfigEnvOverride enables env overrides using an optional prefix and a dot-to-underscore replacer.
func ConfigEnvOverride(prefix ...string) configOption {
	return configOptionFunc(func(cfg *configConfig) {
		if len(prefix) > 0 {
			cfg.envPrefix = prefix[0]
		}
		cfg.automaticEnv = true
		if cfg.keyReplacer == nil {
			cfg.keyReplacer = strings.NewReplacer(".", "_", "-", "_")
		}
	})
}

// ConfigAutomaticEnv enables viper.AutomaticEnv.
func ConfigAutomaticEnv() configOption {
	return configOptionFunc(func(cfg *configConfig) {
		cfg.automaticEnv = true
	})
}

// ConfigNoEnv disables env overrides for this config.
func ConfigNoEnv() configOption {
	return configOptionFunc(func(cfg *configConfig) {
		cfg.automaticEnv = false
		cfg.envPrefix = ""
	})
}

// ConfigDefault sets a default value.
func ConfigDefault(key string, value any) configOption {
	return configOptionFunc(func(cfg *configConfig) {
		if cfg.defaults == nil {
			cfg.defaults = make(map[string]any)
		}
		cfg.defaults[key] = value
	})
}

// ConfigWithViper allows custom viper configuration.
func ConfigWithViper(fn func(*viper.Viper) error) configOption {
	return configOptionFunc(func(cfg *configConfig) {
		if fn == nil {
			return
		}
		cfg.hooks = append(cfg.hooks, fn)
	})
}

func parseConfigOptions(opts []any) (configConfig, bindConfig, []fx.Option, error) {
	cfg := configConfig{}
	cfg.automaticEnv = true
	cfg.keyReplacer = strings.NewReplacer(".", "_", "-", "_")
	var bindOpts []any
	for _, opt := range opts {
		switch o := opt.(type) {
		case nil:
			continue
		case configOption:
			o.applyConfig(&cfg)
		default:
			bindOpts = append(bindOpts, opt)
		}
		if cfg.err != nil {
			return cfg, bindConfig{}, nil, cfg.err
		}
	}
	bindCfg, _, extra, err := parseBindOptions(bindOpts)
	return cfg, bindCfg, extra, err
}

func parseConfigOptionsWithWatch(opts []any) (configConfig, configWatchConfig, bindConfig, []fx.Option, error) {
	cfg := configConfig{}
	cfg.automaticEnv = true
	cfg.keyReplacer = strings.NewReplacer(".", "_", "-", "_")
	watchCfg := configWatchConfig{}
	var bindOpts []any
	for _, opt := range opts {
		switch o := opt.(type) {
		case nil:
			continue
		case configOption:
			o.applyConfig(&cfg)
		case ConfigWatchOption:
			o.applyWatch(&watchCfg)
		default:
			bindOpts = append(bindOpts, opt)
		}
		if cfg.err != nil {
			return cfg, watchCfg, bindConfig{}, nil, cfg.err
		}
	}
	bindCfg, _, extra, err := parseBindOptions(bindOpts)
	return cfg, watchCfg, bindCfg, extra, err
}

func configHasWatchOption(opts []any) bool {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if _, ok := opt.(ConfigWatchOption); ok {
			return true
		}
	}
	return false
}

func loadViper(cfg configConfig, path string) (*viper.Viper, error) {
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
	for _, path := range cfg.configPaths {
		v.AddConfigPath(path)
	}
	if len(cfg.defaults) > 0 {
		for k, val := range cfg.defaults {
			v.SetDefault(k, val)
		}
	}
	for _, hook := range cfg.hooks {
		if err := hook(v); err != nil {
			return nil, err
		}
	}
	if path != "" || cfg.configName != "" {
		if err := v.ReadInConfig(); err != nil {
			var nf viper.ConfigFileNotFoundError
			if cfg.optional && errors.As(err, &nf) {
				return v, nil
			}
			return nil, err
		}
	}
	return v, nil
}

func resolveConfigTarget(pathOrKey string) (string, string) {
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

func hasConfigSource(cfg configConfig) bool {
	return cfg.configName != "" || cfg.configType != "" || len(cfg.configPaths) > 0
}

func buildConfigWatchOption(cfg configWatchConfig, key string, config configConfig, path string) fx.Option {
	if !cfg.enabled {
		return fx.Options()
	}
	if cfg.debounce == 0 {
		cfg.debounce = 200 * time.Millisecond
	}
	keys := cfg.keys
	if len(keys) == 0 && key != "" {
		keys = []string{key}
	}
	return fx.Invoke(func(in struct {
		fx.In
		Store   *configStore  `optional:"true"`
		Restart restartSignal `optional:"true"`
		Logger  *zap.Logger   `optional:"true"`
	}) {
		if in.Restart == nil {
			return
		}
		if in.Store != nil && in.Store.v != nil {
			in.Store.addWatch(keys, cfg.debounce, in.Restart, in.Logger)
			return
		}
		var v *viper.Viper
		if path != "" || hasConfigSource(config) {
			loaded, err := loadViper(config, path)
			if err != nil {
				return
			}
			v = loaded
		}
		if v == nil {
			return
		}
		installSingleWatch(v, keys, cfg.debounce, in.Restart, in.Logger)
	})
}

type configWatcher struct {
	keys     []string
	debounce time.Duration
	restart  restartSignal
	logger   *zap.Logger
	last     string
	timer    *time.Timer
	pending  bool
}

func (s *configStore) addWatch(keys []string, debounce time.Duration, restart restartSignal, logger *zap.Logger) {
	if s.v == nil || restart == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	w := &configWatcher{
		keys:     keys,
		debounce: debounce,
		restart:  restart,
		logger:   logger,
	}
	w.last, _ = snapshotKeys(s.v, keys)
	s.watchers = append(s.watchers, w)
	if s.watchActive {
		return
	}
	s.watchActive = true
	s.v.OnConfigChange(func(_ fsnotify.Event) {
		s.handleChange()
	})
	s.v.WatchConfig()
}

func (s *configStore) handleChange() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.watchers {
		next, err := snapshotKeys(s.v, w.keys)
		if err == nil && w.last == next {
			continue
		}
		w.last = next
		if w.logger != nil {
			w.logger.Info("config changed, restarting...")
		}
		scheduleRestart(w)
	}
}

func scheduleRestart(w *configWatcher) {
	if w.debounce == 0 {
		select {
		case w.restart <- struct{}{}:
		default:
		}
		return
	}
	if w.timer == nil {
		w.timer = time.NewTimer(w.debounce)
		w.pending = true
		go func() {
			for range w.timer.C {
				if !w.pending {
					return
				}
				w.pending = false
				select {
				case w.restart <- struct{}{}:
				default:
				}
				return
			}
		}()
		return
	}
	if !w.timer.Stop() {
		select {
		case <-w.timer.C:
		default:
		}
	}
	w.timer.Reset(w.debounce)
	w.pending = true
}

func installSingleWatch(v *viper.Viper, keys []string, debounce time.Duration, restart restartSignal, logger *zap.Logger) {
	w := &configWatcher{
		keys:     keys,
		debounce: debounce,
		restart:  restart,
		logger:   logger,
	}
	w.last, _ = snapshotKeys(v, keys)
	v.OnConfigChange(func(_ fsnotify.Event) {
		next, err := snapshotKeys(v, keys)
		if err == nil && w.last == next {
			return
		}
		w.last = next
		if w.logger != nil {
			w.logger.Info("config changed, restarting...")
		}
		scheduleRestart(w)
	})
	v.WatchConfig()
}

func decodeConfig(v *viper.Viper, key string, out any) error {
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
	data := buildConfigMap(v, key, elem)
	if len(data) == 0 {
		if key == "" {
			return v.Unmarshal(out)
		}
		return v.UnmarshalKey(key, out)
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
		Result: out,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(data)
}

func buildConfigMap(v *viper.Viper, prefix string, t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	out := make(map[string]any)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag := field.Tag.Get("mapstructure")
		name, opts := parseMapstructureTag(tag)
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		fullKey := name
		if prefix != "" {
			fullKey = prefix + "." + name
		}
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct && !isTimeType(fieldType) && !hasTagOption(opts, "squash") {
			nested := buildConfigMap(v, fullKey, fieldType)
			if len(nested) > 0 {
				out[name] = nested
			}
			continue
		}
		val := v.Get(fullKey)
		if val != nil {
			out[name] = val
		}
	}
	return out
}

func parseMapstructureTag(tag string) (string, []string) {
	if tag == "" {
		return "", nil
	}
	parts := strings.Split(tag, ",")
	if len(parts) == 1 {
		return parts[0], nil
	}
	return parts[0], parts[1:]
}

func hasTagOption(opts []string, target string) bool {
	for _, opt := range opts {
		if opt == target {
			return true
		}
	}
	return false
}

func isTimeType(t reflect.Type) bool {
	return t.PkgPath() == "time" && t.Name() == "Time"
}
