package realtime

import "time"

type BrokerTLSConfig struct {
	CAFile             string `mapstructure:"ca_file"`
	CertFile           string `mapstructure:"cert_file"`
	KeyFile            string `mapstructure:"key_file"`
	ServerName         string `mapstructure:"server_name"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

type BrokerConfig struct {
	Mode           string          `mapstructure:"mode" default:"embedded"`
	Endpoint       string          `mapstructure:"endpoint"`
	ClientID       string          `mapstructure:"client_id"`
	Username       string          `mapstructure:"username"`
	Password       string          `mapstructure:"password"`
	CleanSession   bool            `mapstructure:"clean_session" default:"false"`
	ConnectTimeout time.Duration   `mapstructure:"connect_timeout" default:"10s"`
	TLS            BrokerTLSConfig `mapstructure:"tls"`
}

type SessionControlEMQXConfig struct {
	Endpoint    string          `mapstructure:"endpoint"`
	Username    string          `mapstructure:"username"`
	Password    string          `mapstructure:"password"`
	BearerToken string          `mapstructure:"bearer_token"`
	Timeout     time.Duration   `mapstructure:"timeout" default:"5s"`
	TLS         BrokerTLSConfig `mapstructure:"tls"`
}

type SessionControlConfig struct {
	Provider string                   `mapstructure:"provider"`
	EMQX     SessionControlEMQXConfig `mapstructure:"emqx"`
}

type TCPListenerConfig struct {
	ID      string `mapstructure:"id" default:"t1"`
	Address string `mapstructure:"address" default:":1883"`
}

type WebsocketListenerConfig struct {
	ID   string `mapstructure:"id" default:"ws1"`
	Path string `mapstructure:"path" default:"/realtime"`
}

type TopicACLConfig struct {
	Enabled         bool     `mapstructure:"enabled" default:"false"`
	AllowedPrefixes []string `mapstructure:"allowed_prefixes"`
}

type Config struct {
	Broker            BrokerConfig            `mapstructure:"broker"`
	SessionControl    SessionControlConfig    `mapstructure:"session_control"`
	TCPListener       TCPListenerConfig       `mapstructure:"tcp_listener"`
	WebsocketListener WebsocketListenerConfig `mapstructure:"websocket_listener"`
	TopicACL          TopicACLConfig          `mapstructure:"topic_acl"`
}
