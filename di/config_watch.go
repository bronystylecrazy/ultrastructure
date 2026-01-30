package di

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

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
		return "", fmt.Errorf(errViperNil)
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

func buildConfigWatchOption(cfg configWatchConfig, key string, config configConfig, path string, scope string) fx.Option {
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
	storeTag := `optional:"true"`
	if scope != "" {
		storeTag = storeTag + ` name:"` + scope + `"`
	}
	return fx.Invoke(fx.Annotate(func(store *configStore, restart restartSignal, logger *zap.Logger) {
		// ConfigWatch only works when a restart signal is provided (di.App().Run).
		if restart == nil {
			return
		}
		if store != nil && store.v != nil {
			store.addWatch(keys, cfg.debounce, restart, logger)
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
		installSingleWatch(v, keys, cfg.debounce, restart, logger)
	}, fx.ParamTags(storeTag, `optional:"true"`, `optional:"true"`)))
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
