package otel

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

type TLSConfig struct {
	CAFile             string `mapstructure:"ca_file"`
	CertFile           string `mapstructure:"cert_file"`
	KeyFile            string `mapstructure:"key_file"`
	ServerName         string `mapstructure:"server_name"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

func (c TLSConfig) IsZero() bool {
	return c.CAFile == "" &&
		c.CertFile == "" &&
		c.KeyFile == "" &&
		c.ServerName == "" &&
		!c.InsecureSkipVerify
}

func (c TLSConfig) Load() (*tls.Config, error) {
	if c.IsZero() {
		return nil, nil
	}
	tlsCfg := &tls.Config{
		ServerName:         c.ServerName,
		InsecureSkipVerify: c.InsecureSkipVerify,
	}
	if c.CAFile != "" {
		b, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read ca file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("invalid ca file")
		}
		tlsCfg.RootCAs = pool
	}
	if c.CertFile != "" || c.KeyFile != "" {
		if c.CertFile == "" || c.KeyFile == "" {
			return nil, fmt.Errorf("cert_file and key_file must be set together")
		}
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return tlsCfg, nil
}

func mergeTLS(base, override TLSConfig) TLSConfig {
	out := base
	if override.CAFile != "" {
		out.CAFile = override.CAFile
	}
	if override.CertFile != "" {
		out.CertFile = override.CertFile
	}
	if override.KeyFile != "" {
		out.KeyFile = override.KeyFile
	}
	if override.ServerName != "" {
		out.ServerName = override.ServerName
	}
	if override.InsecureSkipVerify {
		out.InsecureSkipVerify = true
	}
	return out
}
