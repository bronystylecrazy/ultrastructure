package web

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/meta"
	"github.com/bronystylecrazy/ultrastructure/otel"
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
	FiberConfigurer
	di.Node
}

const FiberConfigurersGroupName = "us.web.fiber_config_configurers"

type FiberConfigurer interface {
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
					func() FiberConfigurer { return fiberConfigConfigurerFunc(fn) },
					di.Group(FiberConfigurersGroupName),
				).Build()
			},
		}
	default:
		return fiberConfigNodeOption{
			err: fmt.Errorf("web: unsupported WithFiberConfig signature %T", configure),
		}
	}
}

func WithFiberAppName(name string) FiberConfigurer {
	return fiberConfigConfigurerFunc(func(cfg *fiber.Config) {
		name = strings.TrimSpace(name)
		if name != "" {
			cfg.AppName = BuildAppName(name)
		}
	})
}

type Server interface {
	Listen() error
	Wait() <-chan struct{}
}

type FiberServer struct {
	otel.Telemetry

	App       *fiber.App
	Config    FiberConfig
	WebConfig Config

	startedCh   chan struct{}
	startedOnce sync.Once
	listenDone  chan struct{}
	listenOnce  sync.Once
	listenErr   error
}

func NewFiberServer(webConfig Config, config FiberConfig, configurers ...FiberConfigurer) *FiberServer {
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
		StructValidator:    NewFiberValidator(),
	}
	for _, configurer := range configurers {
		if configurer != nil {
			configurer.MutateFiberConfig(&appCfg)
		}
	}

	return &FiberServer{
		Telemetry:  otel.Nop(),
		App:        fiber.New(appCfg),
		Config:     config,
		WebConfig:  webConfig,
		startedCh:  make(chan struct{}),
		listenDone: make(chan struct{}),
	}
}

func (s *FiberServer) Listen() error {
	s.listenOnce.Do(func() {
		s.listenErr = s.listen()
		close(s.listenDone)
	})
	<-s.listenDone
	return s.listenErr
}

func (s *FiberServer) listen() error {
	listenAddr := ParseAddr(s.WebConfig)
	listenConfig, err := BuildFiberListenConfig(s.WebConfig)
	if err != nil {
		return err
	}

	s.App.Hooks().OnListen(func(data fiber.ListenData) error {
		s.markStarted()

		scheme := "http"
		if data.TLS {
			scheme = "https"
		}

		host := strings.TrimSpace(data.Host)
		port := strings.TrimSpace(data.Port)
		address := listenAddr
		if host != "" && port != "" {
			address = net.JoinHostPort(host, port)
		}

		s.Obs.Info("fiber server listening",
			zap.String("address", address),
			zap.String("endpoint", fmt.Sprintf("%s://%s", scheme, address)),
			zap.String("network", listenConfig.ListenerNetwork),
			zap.Bool("tls", data.TLS),
			zap.Bool("prefork", data.Prefork),
			zap.Int("pid", data.PID),
			zap.Int("process_count", data.ProcessCount),
			zap.Int("handler_count", data.HandlerCount),
		)
		return nil
	})

	s.App.Hooks().OnPostShutdown(func(err error) error {
		if err != nil {
			s.Obs.Error("fiber server stopped with error", zap.Error(err))
		} else {
			s.Obs.Info("fiber server stopped")
		}
		return nil
	})

	err = s.App.Listen(listenAddr, listenConfig)
	if err != nil {
		s.markStarted()
	}
	return err
}

func (s *FiberServer) Wait() <-chan struct{} {
	return s.startedCh
}

func (s *FiberServer) markStarted() {
	s.startedOnce.Do(func() {
		close(s.startedCh)
	})
}

func (s *FiberServer) Stop(ctx context.Context) error {
	return s.App.ShutdownWithContext(ctx)
}
