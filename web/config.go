package web

import "time"

type Config struct {
	Name   string       `mapstructure:"name" default:"Ultrastructure API"`
	Server ServerConfig `mapstructure:"server"`
	Listen ListenConfig `mapstructure:"listen"`
	TLS    TLSConfig    `mapstructure:"tls"`
}

type ServerConfig struct {
	Host string `mapstructure:"host" default:"0.0.0.0"`
	Port int    `mapstructure:"port" default:"8080"`
}

type ListenConfig struct {
	ListenerNetwork       string        `mapstructure:"listener_network" default:"tcp"`
	ShutdownTimeout       time.Duration `mapstructure:"shutdown_timeout" default:"10s"`
	DisableStartupMessage bool          `mapstructure:"disable_startup_message" default:"false"`
	EnablePrefork         bool          `mapstructure:"enable_prefork" default:"false"`
	EnablePrintRoutes     bool          `mapstructure:"enable_print_routes" default:"false"`
}

type TLSConfig struct {
	CertFile       string `mapstructure:"cert_file"`
	CertKeyFile    string `mapstructure:"cert_key_file"`
	CertClientFile string `mapstructure:"cert_client_file"`
	TLSMinVersion  string `mapstructure:"tls_min_version" default:"1.2"`
}
