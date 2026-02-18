package web

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bronystylecrazy/ultrastructure/meta"
	"github.com/dustin/go-humanize"
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

type FiberAppOption interface {
	apply(*fiberAppOptions)
}

type fiberAppOptions struct {
	name string
}

type fiberAppOptionFunc func(*fiberAppOptions)

func (f fiberAppOptionFunc) apply(o *fiberAppOptions) {
	f(o)
}

func WithName(name string) FiberAppOption {
	return fiberAppOptionFunc(func(o *fiberAppOptions) {
		name = strings.TrimSpace(name)
		if name != "" {
			o.name = name
		}
	})
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

func NewFiberApp(config FiberConfig) *fiber.App {
	return NewFiberAppWithOptions(config)
}

func NewFiberAppWithOptions(config FiberConfig, opts ...FiberAppOption) *fiber.App {
	options := fiberAppOptions{
		name: meta.Name,
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&options)
		}
	}

	bodyLimit, err := ParseBodyLimit(config.App.BodyLimit)
	if err != nil {
		bodyLimit = fiber.DefaultBodyLimit
	}

	return fiber.New(fiber.Config{
		ServerHeader:       config.App.ServerHeader,
		BodyLimit:          bodyLimit,
		AppName:            buildAppName(options.name),
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
	})
}

func buildAppName(name string) string {
	return fmt.Sprintf("%s (%s %s %s)", name, meta.Version, meta.Commit, meta.BuildDate)
}

func RegisterFiberApp(lc fx.Lifecycle, app *fiber.App, logger *zap.Logger, config Config) {
	if logger == nil {
		logger = zap.NewNop()
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
				listenCfg := fiber.ListenConfig{
					ListenerNetwork:       config.Listen.ListenerNetwork,
					ShutdownTimeout:       config.Listen.ShutdownTimeout,
					DisableStartupMessage: config.Listen.DisableStartupMessage,
					EnablePrefork:         config.Listen.EnablePrefork,
					EnablePrintRoutes:     config.Listen.EnablePrintRoutes,
				}
				if config.TLS.CertFile != "" || config.TLS.CertKeyFile != "" || config.TLS.CertClientFile != "" {
					tlsVersion, err := ParseTLSMinVersion(config.TLS.TLSMinVersion)
					if err != nil {
						logger.Error("invalid tls_min_version", zap.String("tls_min_version", config.TLS.TLSMinVersion), zap.Error(err))
						return
					}
					listenCfg.CertFile = config.TLS.CertFile
					listenCfg.CertKeyFile = config.TLS.CertKeyFile
					listenCfg.CertClientFile = config.TLS.CertClientFile
					listenCfg.TLSMinVersion = tlsVersion
				}

				logger.Info("fiber listener starting",
					zap.String("address", addr),
					zap.String("network", listenCfg.ListenerNetwork),
					zap.Bool("prefork", listenCfg.EnablePrefork),
				)

				err := app.Listen(addr, listenCfg)
				if err != nil {
					logger.Error("failed to start fiber app", zap.Error(err))
					return
				}
				logger.Info("fiber listener stopped")
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return app.ShutdownWithContext(ctx)
		},
	})
}

func ParseTLSMinVersion(v string) (uint16, error) {
	switch v {
	case "", "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unsupported tls version %q (expected 1.2 or 1.3)", v)
	}
}

func ParseBodyLimit(v string) (int, error) {
	s := strings.TrimSpace(v)
	if s == "" {
		return fiber.DefaultBodyLimit, nil
	}

	n, err := humanize.ParseBytes(s)
	if err != nil {
		return 0, err
	}
	if n > uint64(math.MaxInt) {
		return 0, fmt.Errorf("body limit overflows int")
	}

	return int(n), nil
}
