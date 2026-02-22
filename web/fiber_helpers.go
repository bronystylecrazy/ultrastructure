package web

import (
	"crypto/tls"
	"fmt"
	"math"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/meta"
	"github.com/dustin/go-humanize"
	"github.com/gofiber/fiber/v3"
)

func BuildAppName(name string) string {
	return fmt.Sprintf("%s (%s %s %s)", name, meta.Version, meta.Commit, meta.BuildDate)
}

func BuildFiberListenConfig(config Config) (fiber.ListenConfig, error) {
	out := fiber.ListenConfig{
		ListenerNetwork:       config.Listen.ListenerNetwork,
		ShutdownTimeout:       config.Listen.ShutdownTimeout,
		DisableStartupMessage: config.Listen.DisableStartupMessage,
		EnablePrefork:         config.Listen.EnablePrefork,
		EnablePrintRoutes:     config.Listen.EnablePrintRoutes,
	}
	if config.TLS.CertFile == "" && config.TLS.CertKeyFile == "" && config.TLS.CertClientFile == "" {
		return out, nil
	}

	tlsVersion, err := ParseTLSMinVersion(config.TLS.TLSMinVersion)
	if err != nil {
		return fiber.ListenConfig{}, err
	}

	out.CertFile = config.TLS.CertFile
	out.CertKeyFile = config.TLS.CertKeyFile
	out.CertClientFile = config.TLS.CertClientFile
	out.TLSMinVersion = tlsVersion
	return out, nil
}

func ParseAddr(config Config) string {
	return fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
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
