package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/meta"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type FiberConfig struct {
	App           FiberAppConfig   `mapstructure:"app"`
	Proxy         FiberProxyConfig `mapstructure:"proxy"`
	CaseSensitive bool             `mapstructure:"case_sensitive" default:"false"`
	StrictRouting bool             `mapstructure:"strict_routing" default:"false"`
}

type FiberAppConfig struct {
	ServerHeader       string        `mapstructure:"server_header"`
	BodyLimit          string        `mapstructure:"body_limit" default:"4MiB"`
	ReadTimeout        time.Duration `mapstructure:"read_timeout" default:"10s"`
	WriteTimeout       time.Duration `mapstructure:"write_timeout" default:"10s"`
	IdleTimeout        time.Duration `mapstructure:"idle_timeout" default:"60s"`
	ReadBufferSize     int           `mapstructure:"read_buffer_size" default:"4096"`
	WriteBufferSize    int           `mapstructure:"write_buffer_size" default:"4096"`
	DisableKeepalive   bool          `mapstructure:"disable_keepalive" default:"false"`
	EnableIPValidation bool          `mapstructure:"enable_ip_validation" default:"true"`
}

type FiberProxyConfig struct {
	TrustProxy       bool                   `mapstructure:"trust_proxy" default:"false"`
	ProxyHeader      string                 `mapstructure:"proxy_header"`
	TrustProxyConfig fiber.TrustProxyConfig `mapstructure:"trust_proxy_config"`
}

type FiberConfigOption interface {
	FiberConfigConfigurer
	di.Node
}

const FiberConfigConfigurersGroupName = "us.web.fiber_config_configurers"

type FiberConfigConfigurer interface {
	MutateFiberConfig(*fiber.Config)
}

type fiberConfigNodeOption struct {
	mutate func(*fiber.Config)
	build  func() (fx.Option, error)
	err    error
}

func (o fiberConfigNodeOption) MutateFiberConfig(cfg *fiber.Config) {
	if o.mutate != nil {
		o.mutate(cfg)
	}
}

func (o fiberConfigNodeOption) Build() (fx.Option, error) {
	if o.err != nil {
		return nil, o.err
	}
	if o.build == nil {
		return nil, fmt.Errorf("web: missing WithFiberConfig builder")
	}
	return o.build()
}

type fiberConfigConfigurerFunc func(*fiber.Config)

func (f fiberConfigConfigurerFunc) MutateFiberConfig(cfg *fiber.Config) {
	f(cfg)
}

func WithFiberConfig(configure any) FiberConfigOption {
	switch fn := configure.(type) {
	case func(*fiber.Config):
		return fiberConfigNodeOption{
			mutate: fn,
			build: func() (fx.Option, error) {
				return di.Provide(
					func() FiberConfigConfigurer { return fiberConfigConfigurerFunc(fn) },
					di.Group(FiberConfigConfigurersGroupName),
				).Build()
			},
		}
	default:
		return fiberConfigNodeOption{
			err: fmt.Errorf("web: unsupported WithFiberConfig signature %T", configure),
		}
	}
}

func WithFiberAppName(name string) FiberConfigConfigurer {
	return fiberConfigConfigurerFunc(func(cfg *fiber.Config) {
		name = strings.TrimSpace(name)
		if name != "" {
			cfg.AppName = BuildAppName(name)
		}
	})
}

type FiberServer struct {
	App    *fiber.App
	Logger *zap.Logger
	Config Config
}

func NewFiberApp(config FiberConfig, configurers ...FiberConfigConfigurer) *fiber.App {
	bodyLimit, err := ParseBodyLimit(config.App.BodyLimit)
	if err != nil {
		bodyLimit = fiber.DefaultBodyLimit
	}

	appCfg := fiber.Config{
		ServerHeader:       config.App.ServerHeader,
		BodyLimit:          bodyLimit,
		AppName:            BuildAppName(meta.Name),
		ReadTimeout:        config.App.ReadTimeout,
		WriteTimeout:       config.App.WriteTimeout,
		IdleTimeout:        config.App.IdleTimeout,
		ReadBufferSize:     config.App.ReadBufferSize,
		WriteBufferSize:    config.App.WriteBufferSize,
		DisableKeepalive:   config.App.DisableKeepalive,
		EnableIPValidation: config.App.EnableIPValidation,
		TrustProxy:         config.Proxy.TrustProxy,
		ProxyHeader:        config.Proxy.ProxyHeader,
		TrustProxyConfig:   config.Proxy.TrustProxyConfig,
		CaseSensitive:      config.CaseSensitive,
		StrictRouting:      config.StrictRouting,
	}
	for _, configurer := range configurers {
		if configurer != nil {
			configurer.MutateFiberConfig(&appCfg)
		}
	}

	return fiber.New(appCfg)
}

func NewFiberServer(app *fiber.App, logger *zap.Logger, config Config) *FiberServer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FiberServer{
		App:    app,
		Logger: logger,
		Config: config,
	}
}

func (s *FiberServer) Listen() error {
	listenCfg, err := BuildFiberListenConfig(s.Config)
	if err != nil {
		return err
	}

	addr := ListenAddress(s.Config)
	s.Logger.Info("fiber listener starting",
		zap.String("address", addr),
		zap.String("network", listenCfg.ListenerNetwork),
		zap.Bool("prefork", listenCfg.EnablePrefork),
	)

	if err := s.App.Listen(addr, listenCfg); err != nil {
		return err
	}
	s.Logger.Info("fiber listener stopped")
	return nil
}

func (s *FiberServer) Shutdown(ctx context.Context) error {
	return s.App.ShutdownWithContext(ctx)
}

func RegisterFiberApp(lc fx.Lifecycle, app *fiber.App, logger *zap.Logger, config Config) {
	server := NewFiberServer(app, logger, config)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := server.Listen(); err != nil {
					server.Logger.Error("failed to start fiber app", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: server.Shutdown,
	})
}
