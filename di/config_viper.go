package di

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go.uber.org/fx"
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
	scope     string
}

type configScoper interface {
	withConfigScope(string) Node
}

func (n configNode[T]) withConfigScope(scope string) Node {
	n.scope = scope
	return n
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
	_, tagSets, err := buildProvideSpec(bindCfg, dummy, nil)
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
	constructor := func(store *configStore) (T, error) {
		var out T
		var v *viper.Viper
		if store != nil && store.v != nil {
			v = store.v
			for _, hook := range cfg.hooks {
				if err := hook(v); err != nil {
					return out, err
				}
			}
		} else {
			if path == "" && !hasConfigSource(cfg) {
				return out, fmt.Errorf(errConfigSourceMissing)
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
	wrapped := wrapConfigStoreParam(constructor, n.scope)
	spec, _, err := buildProvideSpec(bindCfg, wrapped, nil)
	if err != nil {
		return nil, err
	}
	provideOpt, err := buildProvideConstructorOption(spec, wrapped, bindCfg.metadata)
	if err != nil {
		return nil, err
	}
	var out []fx.Option
	out = append(out, provideOpt)
	if watchCfg.enabled {
		out = append(out, buildConfigWatchOption(watchCfg, key, cfg, path, n.scope))
	}
	out = append(out, extra...)
	return packOptions(out), nil
}

type configStore struct {
	v *viper.Viper

	mu          sync.Mutex
	watchers    []*configWatcher
	watchActive bool
}

type configFileNode struct {
	path  string
	opts  []any
	scope string
}

func (n configFileNode) withConfigScope(scope string) Node {
	n.scope = scope
	return n
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
	provideOpt := fx.Provide(constructor)
	if n.scope != "" {
		provideOpt = fx.Provide(fx.Annotate(constructor, fx.ResultTags(`name:"`+n.scope+`"`)))
	}
	_ = bindCfg
	var out []fx.Option
	out = append(out, provideOpt)
	out = append(out, extra...)
	return packOptions(out), nil
}

type configBindNode[T any] struct {
	key   string
	opts  []any
	scope string
}

func (n configBindNode[T]) withConfigScope(scope string) Node {
	n.scope = scope
	return n
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
		} else if hasConfigOptions(cfg) {
			loaded, err := loadViper(cfg, "")
			if err != nil {
				return out, err
			}
			v = loaded
		} else {
			return out, fmt.Errorf(errConfigSourceMissing)
		}
		if v == nil {
			return out, fmt.Errorf(errConfigSourceUnavailable)
		}
		return out, decodeConfig(v, n.key, &out)
	}
	wrapped := wrapConfigStoreParam(constructor, n.scope)
	spec, _, err := buildProvideSpec(bindCfg, wrapped, nil)
	if err != nil {
		return nil, err
	}
	provideOpt, err := buildProvideConstructorOption(spec, wrapped, bindCfg.metadata)
	if err != nil {
		return nil, err
	}
	var out []fx.Option
	out = append(out, provideOpt)
	out = append(out, extra...)
	return packOptions(out), nil
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
	_, tagSets, err := buildProvideSpec(bindCfg, dummy, nil)
	if err != nil {
		return nil, err
	}
	return tagSets, nil
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
			cfg.err = fmt.Errorf(errConfigTypeEmpty)
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
			cfg.err = fmt.Errorf(errConfigNameEmpty)
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
			cfg.err = fmt.Errorf(errConfigPathEmpty)
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
		case Node:
			configOpts, _, err := extractConfigOptionsFromNode(o)
			if err != nil {
				return cfg, bindConfig{}, nil, err
			}
			for _, co := range configOpts {
				co.applyConfig(&cfg)
				if cfg.err != nil {
					return cfg, bindConfig{}, nil, cfg.err
				}
			}
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
		case Node:
			configOpts, watchOpts, err := extractConfigOptionsFromNode(o)
			if err != nil {
				return cfg, watchCfg, bindConfig{}, nil, err
			}
			for _, co := range configOpts {
				co.applyConfig(&cfg)
				if cfg.err != nil {
					return cfg, watchCfg, bindConfig{}, nil, cfg.err
				}
			}
			for _, wo := range watchOpts {
				wo.applyWatch(&watchCfg)
			}
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

type configOptionNode struct {
	opt   configOption
	watch ConfigWatchOption
}

func (n configOptionNode) Build() (fx.Option, error) {
	return nil, fmt.Errorf(errConfigOptionOutsideConfig)
}

func wrapConfigStoreParam(fn any, scope string) any {
	fnVal := reflect.ValueOf(fn)
	fnType := fnVal.Type()
	if fnType.Kind() != reflect.Func || fnType.NumIn() != 1 {
		return fn
	}

	tag := `optional:"true"`
	if scope != "" {
		tag = tag + ` name:"` + scope + `"`
	}

	inType := reflect.StructOf([]reflect.StructField{
		{
			Name:      "In",
			Type:      reflect.TypeOf(fx.In{}),
			Anonymous: true,
		},
		{
			Name: "Store",
			Type: reflect.TypeOf((*configStore)(nil)),
			Tag:  reflect.StructTag(tag),
		},
	})

	outTypes := []reflect.Type{fnType.Out(0)}
	if fnType.NumOut() == 2 {
		outTypes = append(outTypes, fnType.Out(1))
	}

	wrapperType := reflect.FuncOf([]reflect.Type{inType}, outTypes, false)
	wrapper := reflect.MakeFunc(wrapperType, func(args []reflect.Value) []reflect.Value {
		store := args[0].Field(1)
		return fnVal.Call([]reflect.Value{store})
	})

	return wrapper.Interface()
}

func extractConfigOptionsFromNode(n Node) ([]configOption, []ConfigWatchOption, error) {
	if n == nil {
		return nil, nil, nil
	}
	switch v := n.(type) {
	case moduleNode:
		return extractConfigOptionsFromNodes(v.nodes)
	case optionsNode:
		return extractConfigOptionsFromNodes(v.nodes)
	case conditionalNode:
		ok, err := v.eval()
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, nil
		}
		return extractConfigOptionsFromNodes(v.nodes)
	case switchNode:
		selected, err := v.selectNodes()
		if err != nil {
			return nil, nil, err
		}
		return extractConfigOptionsFromNodes(selected)
	case provideNode:
		if opt, ok := v.constructor.(configOption); ok {
			return []configOption{opt}, nil, nil
		}
		if opt, ok := v.constructor.(ConfigWatchOption); ok {
			return nil, []ConfigWatchOption{opt}, nil
		}
		return nil, nil, fmt.Errorf(errUnsupportedConfigNode, v)
	case supplyNode:
		if opt, ok := v.value.(configOption); ok {
			return []configOption{opt}, nil, nil
		}
		if opt, ok := v.value.(ConfigWatchOption); ok {
			return nil, []ConfigWatchOption{opt}, nil
		}
		return nil, nil, fmt.Errorf(errUnsupportedConfigNode, v)
	case configOptionNode:
		if v.opt != nil {
			return []configOption{v.opt}, nil, nil
		}
		if v.watch != nil {
			return nil, []ConfigWatchOption{v.watch}, nil
		}
		return nil, nil, nil
	case errorNode:
		return nil, nil, v.err
	default:
		// Allow raw configOption / ConfigWatchOption nodes in Switch/If branches.
		if opt, ok := n.(configOption); ok {
			return []configOption{opt}, nil, nil
		}
		if opt, ok := n.(ConfigWatchOption); ok {
			return nil, []ConfigWatchOption{opt}, nil
		}
	}
	return nil, nil, fmt.Errorf(errUnsupportedConfigNode, n)
}

func extractConfigOptionsFromNodes(nodes []Node) ([]configOption, []ConfigWatchOption, error) {
	var configOpts []configOption
	var watchOpts []ConfigWatchOption
	for _, node := range nodes {
		opts, watches, err := extractConfigOptionsFromNode(node)
		if err != nil {
			return nil, nil, err
		}
		if len(opts) > 0 {
			configOpts = append(configOpts, opts...)
		}
		if len(watches) > 0 {
			watchOpts = append(watchOpts, watches...)
		}
	}
	return configOpts, watchOpts, nil
}

func loadViper(cfg configConfig, path string) (*viper.Viper, error) {
	v := viper.New()
	// Apply env/config options before reading from disk.
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
		// Explicit file path overrides name/path search.
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
		// Read config from file if a file path or name is configured.
		if err := v.ReadInConfig(); err != nil {
			var nf viper.ConfigFileNotFoundError
			if cfg.optional && (errors.As(err, &nf) || errors.Is(err, os.ErrNotExist)) {
				return v, nil
			}
			if path != "" {
				if cleaned, ok := sanitizeConfigBytes(path); ok {
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

func sanitizeConfigBytes(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	changed := false
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		// Strip UTF-8 BOM (EF BB BF)
		if i+2 < len(data) && data[i] == 0xEF && data[i+1] == 0xBB && data[i+2] == 0xBF {
			i += 2
			changed = true
			continue
		}
		// Strip zero-width space (E2 80 8B)
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

func hasConfigOptions(cfg configConfig) bool {
	return cfg.configName != "" ||
		cfg.configType != "" ||
		len(cfg.configPaths) > 0 ||
		cfg.envPrefix != "" ||
		cfg.keyReplacer != nil ||
		cfg.automaticEnv ||
		cfg.optional ||
		len(cfg.defaults) > 0 ||
		len(cfg.hooks) > 0
}

func decodeConfig(v *viper.Viper, key string, out any) error {
	t := reflect.TypeOf(out)
	if t == nil || t.Kind() != reflect.Pointer {
		return fmt.Errorf(errConfigTargetMustBePointer)
	}
	elem := t.Elem()
	if elem.Kind() != reflect.Struct {
		if key == "" {
			return v.Unmarshal(out)
		}
		return v.UnmarshalKey(key, out)
	}
	applyDefaultTags(v, key, elem)
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

func applyDefaultTags(v *viper.Viper, prefix string, t reflect.Type) {
	if v == nil || t == nil {
		return
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
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
		defaultVal := field.Tag.Get("default")
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct && !isTimeType(fieldType) && !hasTagOption(opts, "squash") {
			nextPrefix := name
			if prefix != "" {
				nextPrefix = prefix + "." + name
			}
			applyDefaultTags(v, nextPrefix, fieldType)
			continue
		}
		fullKey := name
		if prefix != "" && !hasTagOption(opts, "squash") {
			fullKey = prefix + "." + name
		} else if prefix != "" && hasTagOption(opts, "squash") {
			fullKey = prefix
		}
		if defaultVal != "" && !v.IsSet(fullKey) {
			v.SetDefault(fullKey, defaultVal)
		}
	}
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
